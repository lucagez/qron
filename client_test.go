package qron

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/lucagez/qron/graph/model"
	"github.com/lucagez/qron/sqlc"
	"github.com/lucagez/qron/testutil"
	"github.com/stretchr/testify/assert"
)

func TestClientClient(t *testing.T) {
	pool, cleanup := testutil.PG.CreateDb("client_0")
	defer cleanup()

	port := pool.Config().ConnConfig.Port
	dsn := fmt.Sprintf("postgres://postgres:postgres@localhost:%d/client_0", port)
	clientPool, _ := pgxpool.New(context.Background(), dsn)
	client, err := NewClient(clientPool, Config{
		PollInterval:  10 * time.Millisecond,
		FlushInterval: 10 * time.Millisecond,
		ResetInterval: 10 * time.Millisecond,
		MaxInFlight:   5,
	})
	assert.Nil(t, err)
	defer client.Close()

	t.Run("Should fetch", func(t *testing.T) {
		for i := 0; i < 50; i++ {
			retries := 1
			client.CreateJob(context.Background(), "backup", model.CreateJobArgs{
				Expr:    "@after 100ms",
				Name:    fmt.Sprintf("test-%d", i),
				Retries: &retries,
			})
		}

		// TODO: Probably not the best api for closing?
		// TODO: Should close anyway when main client is closed
		ctx, stop := context.WithCancel(context.Background())
		jobs := client.Fetch(ctx, "backup")

		go func() {
			<-time.After(300 * time.Millisecond)
			stop()
		}()

		counter := 0
		for job := range jobs {
			counter += 1
			if counter < 10 {
				job.Commit()
			}
			if counter > 10 && counter < 20 {
				job.Fail()
			}
		}
		assert.Equal(t, 50, counter)

		all, err := client.SearchJobs(context.Background(), "backup", model.QueryJobsArgs{
			Limit:  100,
			Skip:   0,
			Filter: "test",
		})
		assert.Nil(t, err)

		success := 0
		fail := 0
		for _, job := range all {
			if job.Status == "SUCCESS" {
				success += 1
			}
			if job.Status == "FAILURE" {
				fail += 1
			}
		}

		assert.Equal(t, 9, success)
		assert.Equal(t, 9, fail)
	})

	t.Run("Should flush", func(t *testing.T) {
		for i := 0; i < 2; i++ {
			client.CreateJob(context.Background(), "flush", model.CreateJobArgs{
				Expr: "@every 100ms",
				Name: fmt.Sprintf("test-%d", i),
			})
		}

		// TODO: Probably not the best api for closing?
		// TODO: Should close anyway when main client is closed
		ctx, stop := context.WithCancel(context.Background())

		go func() {
			<-time.After(350 * time.Millisecond)
			stop()
		}()

		counter := 0
		for job := range client.Fetch(ctx, "flush") {
			counter += 1
			job.Commit()
		}

		// Wait for next flush to happen
		time.Sleep(400 * time.Millisecond)

		all, err := client.SearchJobs(context.Background(), "flush", model.QueryJobsArgs{
			Limit:  100,
			Skip:   0,
			Filter: "test",
		})
		assert.Nil(t, err)

		executions := 0
		for _, job := range all {
			// Flushing increases executions
			executions += int(job.ExecutionAmount)
			assert.GreaterOrEqual(t, int(job.ExecutionAmount), 3)
		}

		// 2 jobs executed 3 times
		assert.Equal(t, executions, counter)
	})

	t.Run("Should reset jobs after timeout is reached", func(t *testing.T) {
		for i := 0; i < 50; i++ {
			timeout := 1
			client.CreateJob(context.Background(), "timeout", model.CreateJobArgs{
				Expr:    "@after 100ms",
				Timeout: &timeout,
			})
		}

		ctx, stop := context.WithCancel(context.Background())
		jobs := client.Fetch(ctx, "timeout")

		go func() {
			<-time.After(1500 * time.Millisecond)
			stop()
		}()

		counter := 0
		for range jobs {
			counter += 1
			// Jobs are never committed
		}

		all, err := client.SearchJobs(context.Background(), "timeout", model.QueryJobsArgs{
			Limit:  100,
			Skip:   0,
			Filter: "test",
		})
		assert.Nil(t, err)

		for _, job := range all {
			// Jobs are executed again as the timeout is exceeded
			assert.Greater(t, 1, int(job.ExecutionAmount))
		}
		assert.Equal(t, 100, counter)
	})

	t.Run("Should calculate next execution based on start time", func(t *testing.T) {
		startAt := time.Now().Add(1 * time.Second)
		j, _ := client.CreateJob(context.Background(), "delayed_start", model.CreateJobArgs{
			Expr:    "@after 100ms",
			StartAt: &startAt,
		})

		assert.Equal(t, j.RunAt.Time.Sub(startAt).Milliseconds(), int64(100))
	})

	t.Run("Should fetch jobs in parallel without overlaps", func(t *testing.T) {
		_, cleanup := testutil.PG.CreateDb("no_overlaps")
		defer cleanup()

		port := pool.Config().ConnConfig.Port
		dsn := fmt.Sprintf("postgres://postgres:postgres@localhost:%d/no_overlaps", port)

		q0Pool, _ := pgxpool.New(context.Background(), dsn)
		defer q0Pool.Close()

		q1Pool, _ := pgxpool.New(context.Background(), dsn)
		defer q1Pool.Close()

		q0, err0 := NewClient(q0Pool, Config{
			FlushInterval: 10 * time.Millisecond,
			PollInterval:  10 * time.Millisecond,
			MaxInFlight:   5, // so to maximize chance of getting concurrent reads
		})
		q1, err1 := NewClient(q1Pool, Config{
			FlushInterval: 10 * time.Millisecond,
			PollInterval:  10 * time.Millisecond,
			MaxInFlight:   5,
		})
		assert.Nil(t, err0)
		assert.Nil(t, err1)
		defer q0.Close()
		defer q1.Close()

		for i := 0; i < 100; i++ {
			_, err := q0.CreateJob(context.Background(), "other-executor", model.CreateJobArgs{
				Expr: "@after 100ms",
			})
			assert.Nil(t, err)
		}

		// check that jobs are not fetched twice
		var q0jobs []Job
		var q1jobs []Job
		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			ctx, stop := context.WithCancel(context.Background())
			ch := q0.Fetch(ctx, "other-executor")
			go func() {
				<-time.After(100 * time.Millisecond)
				stop()
			}()

			for job := range ch {
				q0jobs = append(q0jobs, job)
			}
		}()
		go func() {
			defer wg.Done()
			ctx, stop := context.WithCancel(context.Background())
			ch := q1.Fetch(ctx, "other-executor")
			go func() {
				<-time.After(1 * time.Second)
				stop()
			}()

			for job := range ch {
				q1jobs = append(q1jobs, job)
			}
		}()

		wg.Wait()

		for _, j0 := range q0jobs {
			for _, j1 := range q1jobs {
				if j0.ID == j1.ID {
					assert.Fail(t, "found overlapping jobs", j0.ID, j1.ID)
				}
			}
		}

		assert.Equal(t, len(q0jobs)+len(q1jobs), 100)
	})

	t.Run("Should reschedule job", func(t *testing.T) {
		created, err := client.CreateJob(context.Background(), "reschedule", model.CreateJobArgs{
			Expr: "@after 100ms",
		})
		assert.Nil(t, err)

		ctx, stop := context.WithCancel(context.Background())
		jobs := client.Fetch(ctx, "reschedule")

		go func() {
			<-time.After(600 * time.Millisecond)
			stop()
		}()

		counter := 0
		for job := range jobs {
			if job.ID == created.ID {
				counter++
				job.Expr = "@after 200ms"
				job.Retry()
			}
		}

		assert.Equal(t, 3, counter)
	})

	t.Run("Should serialize job generated from sqlc", func(t *testing.T) {
		timeout := 100
		startAt := time.Now().Add(1 * time.Hour)
		meta := `{"some":"meta"}`
		created, err := client.CreateJob(context.Background(), "serialize", model.CreateJobArgs{
			Expr:    "@after 100ms",
			Name:    "test",
			State:   "some rand state",
			Timeout: &timeout,
			StartAt: &startAt,
			Meta:    &meta,
		})
		assert.Nil(t, err)

		serialized, err := json.Marshal(created)
		assert.Nil(t, err)

		var job Job
		json.Unmarshal(serialized, &job)
		reserialized, err := json.Marshal(job)
		assert.Nil(t, err)

		assert.Equal(t, string(serialized), string(reserialized))
	})
}

func TestClientDelivery(t *testing.T) {
	pool, cleanup := testutil.PG.CreateDb("delivery")
	defer cleanup()

	port := pool.Config().ConnConfig.Port
	deliveryPool, _ := pgxpool.New(context.Background(), fmt.Sprintf("postgres://postgres:postgres@localhost:%d/delivery", port))
	defer deliveryPool.Close()

	client, err := NewClient(deliveryPool, Config{})
	assert.Nil(t, err)
	defer client.Close()
	ctx := context.Background()

	t.Run("Should deliver job at least once", func(t *testing.T) {
		for i := 0; i < 50; i++ {
			client.CreateJob(ctx, "backup", model.CreateJobArgs{
				Expr: "@after 100ms",
				Name: fmt.Sprintf("test-%d", i),
			})
		}

		// Wait for job time to be elapsed
		time.Sleep(500 * time.Millisecond)

		delayedPool, _ := pgxpool.New(context.Background(), fmt.Sprintf("postgres://postgres:postgres@localhost:%d/delivery", port))
		defer delayedPool.Close()
		delayedClient, err := NewClient(delayedPool, Config{
			FlushInterval: 10 * time.Millisecond,
			PollInterval:  10 * time.Millisecond,
		})
		assert.Nil(t, err)
		defer delayedClient.Close()

		ctx, stop := context.WithCancel(context.Background())
		jobs := delayedClient.Fetch(ctx, "backup")

		go func() {
			<-time.After(500 * time.Millisecond)
			stop()
		}()

		counter := 0
		for job := range jobs {
			counter++
			job.Commit()
		}

		assert.Equal(t, 50, counter)
	})
}

func TestClientOwner(t *testing.T) {
	t.SkipNow()

	pool, cleanup := testutil.PG.CreateDb("owner")
	defer cleanup()

	port := pool.Config().ConnConfig.Port
	dsn := fmt.Sprintf("postgres://postgres:postgres@localhost:%d/owner", port)

	scopedConn, err := sqlc.NewScopedPgx(context.Background(), dsn)
	assert.Nil(t, err)

	scopedClient, err := NewClient(scopedConn, Config{})
	assert.Nil(t, err)
	defer scopedClient.Close()

	adminClient, err := NewClient(pool, Config{})
	assert.Nil(t, err)
	defer adminClient.Close()
	ctx := context.Background()

	t.Run("Should configure scoped conn pool for reading", func(t *testing.T) {
		for i := 0; i < 50; i++ {
			adminClient.CreateJob(ctx, "backup", model.CreateJobArgs{
				Expr: "@after 1 hour",
				Name: fmt.Sprintf("test-%d", i),
			})
		}

		var wg sync.WaitGroup
		wg.Add(2)

		// Perform fetches concurrently to test connections
		// are configured are reset to pool correctly
		go func() {
			for i := 0; i < 100; i++ {
				adminJobs, err := adminClient.SearchJobs(ctx, "backup", model.QueryJobsArgs{
					Limit:  100,
					Skip:   0,
					Filter: "test",
				})
				assert.Nil(t, err)
				assert.Len(t, adminJobs, 50)
			}

			wg.Done()
		}()

		go func() {
			// TODO: Bug when executing tests in parallel
			// -> Permission denied for schema tiny
			for i := 0; i < 100; i++ {
				scopedJobs, err := scopedClient.SearchJobs(sqlc.NewCtx(ctx, "bobby"), "backup", model.QueryJobsArgs{
					Limit:  100,
					Skip:   0,
					Filter: "test",
				})
				assert.Nil(t, err)
				assert.Len(t, scopedJobs, 0)
			}

			wg.Done()
		}()

		wg.Wait()
	})

	t.Run("Should configure scoped conn pool for writing", func(t *testing.T) {
		for i := 0; i < 50; i++ {
			adminClient.CreateJob(ctx, "write", model.CreateJobArgs{
				Expr: "@after 1 hour",
				Name: fmt.Sprintf("owned-test-%d", i),
			})
		}
		for i := 0; i < 20; i++ {
			scopedClient.CreateJob(sqlc.NewCtx(ctx, "bobby"), "write", model.CreateJobArgs{
				Expr: "@after 1 hour",
				Name: fmt.Sprintf("owned-test-scoped-%d", i),
			})
		}

		var wg sync.WaitGroup
		wg.Add(2)

		// Perform fetches concurrently to test connections
		// are configured are reset to pool correctly
		go func() {
			adminJobs, err := adminClient.SearchJobs(ctx, "write", model.QueryJobsArgs{
				Limit:  100,
				Skip:   0,
				Filter: "owned",
			})
			assert.Nil(t, err)
			assert.Len(t, adminJobs, 70)

			wg.Done()
		}()

		go func() {
			// TODO: Bug when executing tests in parallel
			// -> Permission denied for schema tiny
			scopedJobs, err := scopedClient.SearchJobs(sqlc.NewCtx(ctx, "bobby"), "write", model.QueryJobsArgs{
				Limit:  100,
				Skip:   0,
				Filter: "owned",
			})
			assert.Nil(t, err)
			assert.Len(t, scopedJobs, 20)

			wg.Done()
		}()

		wg.Wait()
	})
}

func TestClientNameScope(t *testing.T) {
	t.SkipNow()

	pool, cleanup := testutil.PG.CreateDb("owner")
	defer cleanup()

	port := pool.Config().ConnConfig.Port
	dsn := fmt.Sprintf("postgres://postgres:postgres@localhost:%d/owner", port)

	scopedConn, err := sqlc.NewScopedPgx(context.Background(), dsn)
	assert.Nil(t, err)

	scopedClient, err := NewClient(scopedConn, Config{})
	assert.Nil(t, err)
	defer scopedClient.Close()

	adminClient, err := NewClient(pool, Config{})
	assert.Nil(t, err)
	defer adminClient.Close()
	ctx := context.Background()

	t.Run("Should prevent creating duplicated names on the same owner scope", func(t *testing.T) {
		a, err := adminClient.CreateJob(ctx, "overlap", model.CreateJobArgs{
			Expr: "@after 1 hour",
			Name: "overlapped-job-name",
		})
		assert.Nil(t, err)
		assert.Equal(t, "overlapped-job-name", a.Name)

		b, err := scopedClient.CreateJob(sqlc.NewCtx(ctx, "bobby"), "overlap", model.CreateJobArgs{
			Expr: "@after 1 hour",
			Name: "overlapped-job-name",
		})
		assert.Nil(t, err)
		assert.Equal(t, "overlapped-job-name", b.Name)

		_, adminErr := adminClient.CreateJob(ctx, "overlap", model.CreateJobArgs{
			Expr: "@after 1 hour",
			Name: "overlapped-job-name",
		})
		assert.Contains(t, adminErr.Error(), "duplicate key value violates unique constraint")
		assert.Contains(t, adminErr.Error(), "job_name_owner_key")

		_, scopedErr := scopedClient.CreateJob(sqlc.NewCtx(ctx, "bobby"), "overlap", model.CreateJobArgs{
			Expr: "@after 1 hour",
			Name: "overlapped-job-name",
		})
		assert.Contains(t, scopedErr.Error(), "duplicate key value violates unique constraint")
		assert.Contains(t, scopedErr.Error(), "job_name_owner_key")
	})
}

func TestClientPauseJobs(t *testing.T) {
	pool, cleanup := testutil.PG.CreateDb("start_and_stop")
	defer cleanup()

	port := pool.Config().ConnConfig.Port
	dsn := fmt.Sprintf("postgres://postgres:postgres@localhost:%d/start_and_stop", port)
	clientPool, _ := pgxpool.New(context.Background(), dsn)
	client, err := NewClient(clientPool, Config{
		PollInterval:  10 * time.Millisecond,
		FlushInterval: 10 * time.Millisecond,
		ResetInterval: 10 * time.Millisecond,
		MaxInFlight:   5,
	})
	assert.Nil(t, err)
	defer client.Close()

	t.Run("Should stop & restart execution", func(t *testing.T) {
		// TODO: How to to test this behavior?
		// race condition in getting jobs to pause when in the correct state
		t.SkipNow()

		// Stopped/Started job
		job, err := client.CreateJob(context.Background(), "backup", model.CreateJobArgs{
			Expr: "@every 100ms",
		})
		assert.Nil(t, err)

		// Normal job
		normalJob, err := client.CreateJob(context.Background(), "backup", model.CreateJobArgs{
			Expr: "@every 100ms",
		})
		assert.Nil(t, err)

		ctx, stop := context.WithCancel(context.Background())
		jobs := client.Fetch(ctx, "backup")

		go func() {
			cleanupTimer := time.After(600 * time.Millisecond)
			// After first execution
			stopTimer := time.After(101 * time.Millisecond)
			// Skip an execution
			restartTimer := time.After(510 * time.Millisecond)

			for {
				select {
				case <-cleanupTimer:
					stop()
				case <-stopTimer:
					client.StopJob(context.Background(), "backup", job.ID)
				case <-restartTimer:
					client.RestartJob(context.Background(), "backup", job.ID)
				}
			}
		}()

		// Just commit everything
		for job := range jobs {
			job.Commit()
		}

		jobAfterExec, err := client.QueryJobByID(context.Background(), "backup", job.ID)
		assert.Nil(t, err)

		normalJobAfterExec, err := client.QueryJobByID(context.Background(), "backup", normalJob.ID)
		assert.Nil(t, err)

		assert.Equal(t, int32(2), jobAfterExec.ExecutionAmount)
		assert.GreaterOrEqual(t, normalJobAfterExec.ExecutionAmount, int32(4))
	})
}
