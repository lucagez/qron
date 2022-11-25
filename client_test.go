package tinyq

import (
	"fmt"
	"sync"
	"testing"
	"time"

	_ "github.com/jackc/pgx/stdlib"
	"github.com/lucagez/tinyq/graph/model"
	"github.com/lucagez/tinyq/testutil"
	"github.com/stretchr/testify/assert"
)

func TestClient(t *testing.T) {
	pool, cleanup := testutil.PG.CreateDb("client_0")
	defer cleanup()

	port := pool.Config().ConnConfig.Port
	client, err := NewClient(Config{
		Dsn:           fmt.Sprintf("postgres://postgres:postgres@localhost:%d/client_0", port),
		PollInterval:  10 * time.Millisecond,
		FlushInterval: 10 * time.Millisecond,
		MaxInFlight:   5,
	})
	assert.Nil(t, err)
	defer client.Close()

	t.Run("Should fetch", func(t *testing.T) {
		for i := 0; i < 50; i++ {
			client.CreateJob("backup", model.CreateJobArgs{
				RunAt: "@after 1s",
				Name:  "test",
			})
		}

		time.Sleep(1 * time.Second)

		// TODO: Probably not the best api for closing?
		// TODO: Should close anyway when main client is closed
		jobs, close := client.Fetch("backup")

		go func() {
			<-time.After(1 * time.Second)
			close()
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

		all, err := client.SearchJobs("backup", model.QueryJobsArgs{
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

	t.Run("Should fetch jobs in parallel without overlaps", func(t *testing.T) {
		_, cleanup := testutil.PG.CreateDb("no_overlaps")
		defer cleanup()

		port := pool.Config().ConnConfig.Port
		dsn := fmt.Sprintf("postgres://postgres:postgres@localhost:%d/no_overlaps", port)

		q0, err0 := NewClient(Config{
			Dsn:           dsn,
			FlushInterval: 10 * time.Millisecond,
			PollInterval:  10 * time.Millisecond,
			MaxInFlight:   5, // so to maximize change of getting concurrent reads
		})
		q1, err1 := NewClient(Config{
			Dsn:           dsn,
			FlushInterval: 10 * time.Millisecond,
			PollInterval:  10 * time.Millisecond,
			MaxInFlight:   5,
		})
		assert.Nil(t, err0)
		assert.Nil(t, err1)
		defer q0.Close()
		defer q1.Close()

		for i := 0; i < 100; i++ {
			_, err := q0.CreateJob("other-executor", model.CreateJobArgs{
				RunAt: "@after 100ms",
				Name:  "test",
			})
			assert.Nil(t, err)
		}

		time.Sleep(100 * time.Millisecond)

		// check that jobs are not fetched twice
		var q0jobs []Job
		var q1jobs []Job
		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			ch, close := q0.Fetch("other-executor")
			go func() {
				<-time.After(1 * time.Second)
				close()
			}()

			for job := range ch {
				q0jobs = append(q0jobs, job)
			}
		}()
		go func() {
			defer wg.Done()
			ch, close := q1.Fetch("other-executor")
			go func() {
				<-time.After(1 * time.Second)
				close()
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

		assert.Len(t, q0jobs, 50)
		assert.Len(t, q1jobs, 50)
	})
}