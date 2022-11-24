package graph

import (
	"context"
	"fmt"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/georgysavva/scany/pgxscan"
	_ "github.com/jackc/pgx/stdlib"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/lucagez/tinyq/executor"
	"github.com/lucagez/tinyq/graph/model"
	"github.com/lucagez/tinyq/sqlc"
	"github.com/lucagez/tinyq/testutil"
	"github.com/stretchr/testify/assert"
)

func countJobs(db *pgxpool.Pool, name string) int {
	rows, err := db.Query(context.Background(), `
		select count(*) from tiny.job where name = $1
	`, name)
	if err != nil {
		log.Fatalln("failed to count jobs", err)
	}
	var count int
	pgxscan.ScanOne(&count, rows)
	return count
}

func ptrstring(x string) *string {
	return &x
}

func TestJobResolvers(t *testing.T) {
	pool, cleanup := testutil.PG.CreateDb("job_resolvers")
	defer cleanup()

	queries := sqlc.New(pool)
	resolver := Resolver{Queries: queries, DB: pool}
	ctx := executor.NewCtx(context.Background(), "test-executor")

	t.Run("Should create job", func(t *testing.T) {
		job, err := resolver.Mutation().CreateJob(ctx, &model.CreateJobArgs{
			RunAt: "@weekly",
			Name:  "lmao",
			State: "{}",
		})

		assert.Nil(t, err)
		assert.Equal(t, 1, countJobs(pool, "lmao"))
		assert.Equal(t, "@weekly", job.RunAt)
		assert.Equal(t, "lmao", job.Name.String)
		assert.Equal(t, "{}", job.State.String)
	})

	t.Run("Should update job by ID", func(t *testing.T) {
		job, err := resolver.Mutation().CreateJob(ctx, &model.CreateJobArgs{
			RunAt: "@weekly",
			Name:  "update-lmao",
			State: "{}",
		})
		assert.Nil(t, err)

		updated, err := resolver.Mutation().UpdateJobByID(ctx, job.ID, &model.UpdateJobArgs{
			RunAt: ptrstring("@yearly"),
			State: ptrstring(`{"hello":"world"}`),
		})

		assert.Nil(t, err)
		assert.Equal(t, 1, countJobs(pool, "update-lmao"))
		assert.Equal(t, "@yearly", updated.RunAt)
		assert.Equal(t, "update-lmao", updated.Name.String)
		assert.Equal(t, `{"hello":"world"}`, updated.State.String)
	})

	t.Run("Should update job by name", func(t *testing.T) {
		job, err := resolver.Mutation().CreateJob(ctx, &model.CreateJobArgs{
			RunAt: "@weekly",
			Name:  "update-lmao-by-name",
			State: "{}",
		})
		assert.Nil(t, err)

		updated, err := resolver.Mutation().UpdateJobByName(ctx, job.Name.String, &model.UpdateJobArgs{
			RunAt: ptrstring("@yearly"),
			State: ptrstring(`{"hello":"world"}`),
		})

		assert.Nil(t, err)
		assert.Equal(t, 1, countJobs(pool, "update-lmao-by-name"))
		assert.Equal(t, "@yearly", updated.RunAt)
		assert.Equal(t, "update-lmao-by-name", updated.Name.String)
		assert.Equal(t, `{"hello":"world"}`, updated.State.String)
	})

	t.Run("Should conditionally update job config by name", func(t *testing.T) {
		job, err := resolver.Mutation().CreateJob(ctx, &model.CreateJobArgs{
			RunAt: "@weekly",
			Name:  "update-cond-lmao-by-name",
			State: "{}",
		})
		assert.Nil(t, err)

		updated0, err := resolver.Mutation().UpdateJobByName(ctx, job.Name.String, &model.UpdateJobArgs{
			State: ptrstring(`{"hello":"world"}`),
		})

		assert.Nil(t, err)
		assert.Equal(t, 1, countJobs(pool, "update-cond-lmao-by-name"))
		assert.Equal(t, "@weekly", updated0.RunAt)
		assert.Equal(t, "update-cond-lmao-by-name", updated0.Name.String)
		assert.Equal(t, `{"hello":"world"}`, updated0.State.String)

		updated1, err := resolver.Mutation().UpdateJobByName(ctx, job.Name.String, &model.UpdateJobArgs{})

		assert.Nil(t, err)
		assert.Equal(t, 1, countJobs(pool, "update-cond-lmao-by-name"))
		assert.Equal(t, "@weekly", updated1.RunAt)
		assert.Equal(t, "update-cond-lmao-by-name", updated1.Name.String)
		assert.Equal(t, `{"hello":"world"}`, updated1.State.String)

		updated2, err := resolver.Mutation().UpdateJobByName(ctx, job.Name.String, &model.UpdateJobArgs{
			State: ptrstring(`{"hello":"world2"}`),
		})

		assert.Nil(t, err)
		assert.Equal(t, 1, countJobs(pool, "update-cond-lmao-by-name"))
		assert.Equal(t, "@weekly", updated2.RunAt)
		assert.Equal(t, "update-cond-lmao-by-name", updated2.Name.String)
		assert.Equal(t, `{"hello":"world2"}`, updated2.State.String)
	})

	t.Run("Should delete job by name", func(t *testing.T) {
		_, err := resolver.Mutation().CreateJob(ctx, &model.CreateJobArgs{
			RunAt: "@weekly",
			Name:  "delete-lmao-by-name",
			State: "{}",
		})
		assert.Nil(t, err)
		assert.Equal(t, 1, countJobs(pool, "delete-lmao-by-name"))

		_, err = resolver.Mutation().DeleteJobByName(ctx, "delete-lmao-by-name")

		assert.Nil(t, err)
		assert.Equal(t, 0, countJobs(pool, "delete-lmao-by-name"))
	})

	t.Run("Should delete job by ID", func(t *testing.T) {
		_, err := resolver.Mutation().CreateJob(ctx, &model.CreateJobArgs{
			RunAt: "@weekly",
			Name:  "delete-lmao-by-id",
			State: "{}",
		})
		assert.Nil(t, err)
		assert.Equal(t, 1, countJobs(pool, "delete-lmao-by-id"))

		_, err = resolver.Mutation().DeleteJobByName(ctx, "delete-lmao-by-id")

		assert.Nil(t, err)
		assert.Equal(t, 0, countJobs(pool, "delete-lmao-by-id"))
	})

	t.Run("Should query job by ID", func(t *testing.T) {
		job, err := resolver.Mutation().CreateJob(ctx, &model.CreateJobArgs{
			RunAt: "@weekly",
			Name:  "query-lmao-by-id",
			State: "{}",
		})
		assert.Nil(t, err)
		assert.Equal(t, 1, countJobs(pool, "query-lmao-by-id"))

		queried, err := resolver.Query().QueryJobByID(ctx, job.ID)

		assert.Nil(t, err)
		assert.Equal(t, 1, countJobs(pool, "query-lmao-by-id"))
		assert.Equal(t, "@weekly", queried.RunAt)
		assert.Equal(t, "query-lmao-by-id", queried.Name.String)
		assert.Equal(t, `{}`, queried.State.String)
	})

	t.Run("Should query job by name", func(t *testing.T) {
		job, err := resolver.Mutation().CreateJob(ctx, &model.CreateJobArgs{
			RunAt: "@weekly",
			Name:  "query-lmao-by-name",
			State: "{}",
		})
		assert.Nil(t, err)
		assert.Equal(t, 1, countJobs(pool, "query-lmao-by-name"))

		queried, err := resolver.Query().QueryJobByID(ctx, job.ID)

		assert.Nil(t, err)
		assert.Equal(t, 1, countJobs(pool, "query-lmao-by-name"))
		assert.Equal(t, "@weekly", queried.RunAt)
		assert.Equal(t, "query-lmao-by-name", queried.Name.String)
		assert.Equal(t, `{}`, queried.State.String)
	})

	t.Run("Should search jobs", func(t *testing.T) {
		for i := 0; i < 50; i++ {
			_, err := resolver.Mutation().CreateJob(ctx, &model.CreateJobArgs{
				RunAt: "@weekly",
				Name:  fmt.Sprintf("search-%d", i),
				State: "{}",
			})
			assert.Nil(t, err)
		}

		search0, err := resolver.Query().SearchJobs(ctx, model.QueryJobsArgs{
			Limit:  10,
			Skip:   0,
			Filter: "sear",
		})
		assert.Nil(t, err)
		assert.Len(t, search0, 10)

		for index, s := range search0 {
			assert.Equal(t, fmt.Sprintf("search-%d", index+0), s.Name.String)
		}

		search1, err := resolver.Query().SearchJobs(ctx, model.QueryJobsArgs{
			Limit:  40,
			Skip:   10,
			Filter: "sear",
		})
		assert.Nil(t, err)
		assert.Len(t, search1, 40)

		for index, s := range search1 {
			assert.Equal(t, fmt.Sprintf("search-%d", index+10), s.Name.String)
		}
	})

}

func TestProcessing(t *testing.T) {
	pool, cleanup := testutil.PG.CreateDb("fetch_processing")
	defer cleanup()

	queries := sqlc.New(pool)
	resolver := Resolver{Queries: queries, DB: pool}
	ctx := executor.NewCtx(context.Background(), "test-executor")

	t.Run("Should fetch for processing", func(t *testing.T) {
		for i := 0; i < 50; i++ {
			_, err := resolver.Mutation().CreateJob(ctx, &model.CreateJobArgs{
				RunAt: "@after 1 second",
				Name:  fmt.Sprintf("search-%d", i),
				State: "{}",
			})
			assert.Nil(t, err)
		}

		time.Sleep(1 * time.Second)

		all, err := resolver.Query().SearchJobs(ctx, model.QueryJobsArgs{
			Limit:  100,
			Skip:   0,
			Filter: "search",
		})
		assert.Nil(t, err)

		pending := 0
		ready := 0
		for _, job := range all {
			if job.Status == "PENDING" {
				pending += 1
			}
			if job.Status == "READY" {
				ready += 1
			}
		}

		assert.Equal(t, 0, pending)
		assert.Equal(t, 50, ready)

		fetch, err := resolver.Mutation().FetchForProcessing(ctx, 20)
		assert.Nil(t, err)
		assert.Len(t, fetch, 20)

		for index, job := range fetch {
			assert.Equal(t, fmt.Sprintf("search-%d", index+0), job.Name.String)
			assert.Equal(t, sqlc.TinyStatusPENDING, job.Status)
		}

		all, err = resolver.Query().SearchJobs(ctx, model.QueryJobsArgs{
			Limit:  100,
			Skip:   0,
			Filter: "search",
		})
		assert.Nil(t, err)

		pending = 0
		ready = 0
		for _, job := range all {
			if job.Status == "PENDING" {
				pending += 1
			}
			if job.Status == "READY" {
				ready += 1
			}
		}

		assert.Equal(t, 20, pending)
		assert.Equal(t, 30, ready)
	})
}

func TestConcurrentProcessing(t *testing.T) {
	pool, cleanup := testutil.PG.CreateDb("concurrent_fetch_processing")
	defer cleanup()

	queries := sqlc.New(pool)
	resolver := Resolver{Queries: queries, DB: pool}
	ctx := executor.NewCtx(context.Background(), "test-executor")

	t.Run("Should fetch for concurrent processing", func(t *testing.T) {
		for i := 0; i < 50; i++ {
			_, err := resolver.Mutation().CreateJob(ctx, &model.CreateJobArgs{
				RunAt: "@after 1 second",
				Name:  fmt.Sprintf("search-%d", i),
				State: "{}",
			})
			assert.Nil(t, err)
		}

		time.Sleep(1 * time.Second)

		var wg sync.WaitGroup

		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				fetch, err := resolver.Mutation().FetchForProcessing(ctx, 5)
				assert.Nil(t, err)
				assert.Len(t, fetch, 5)

				for _, job := range fetch {
					assert.Equal(t, sqlc.TinyStatusPENDING, job.Status)
				}
			}()
		}

		wg.Wait()

		all, err := resolver.Query().SearchJobs(ctx, model.QueryJobsArgs{
			Limit:  100,
			Skip:   0,
			Filter: "search",
		})
		assert.Nil(t, err)

		pending := 0
		ready := 0
		for _, job := range all {
			if job.Status == "PENDING" {
				pending += 1
			}
			if job.Status == "READY" {
				ready += 1
			}
		}

		assert.Equal(t, 40, pending)
		assert.Equal(t, 10, ready)
	})
}

func TestCommit(t *testing.T) {
	pool, cleanup := testutil.PG.CreateDb("commit_processing")
	defer cleanup()

	queries := sqlc.New(pool)
	resolver := Resolver{Queries: queries, DB: pool}
	ctx := executor.NewCtx(context.Background(), "test-executor")

	t.Run("Should commit after processing", func(t *testing.T) {
		for i := 0; i < 50; i++ {
			_, err := resolver.Mutation().CreateJob(ctx, &model.CreateJobArgs{
				RunAt: "@after 1 second",
				Name:  fmt.Sprintf("search-%d", i),
				State: "{}",
			})
			assert.Nil(t, err)
		}

		time.Sleep(1 * time.Second)

		fetch, err := resolver.Mutation().FetchForProcessing(ctx, 5)
		assert.Nil(t, err)
		assert.Len(t, fetch, 5)

		var commits []int64
		for _, job := range fetch {
			commits = append(commits, job.ID)
		}

		failedCommits, err := resolver.Mutation().CommitJobs(ctx, commits)
		assert.Nil(t, err)
		assert.Len(t, failedCommits, 0)

		all, err := resolver.Query().SearchJobs(ctx, model.QueryJobsArgs{
			Limit:  100,
			Skip:   0,
			Filter: "search",
		})
		assert.Nil(t, err)

		pending := 0
		success := 0
		ready := 0
		for _, job := range all {
			if job.Status == "PENDING" {
				pending += 1
			}
			if job.Status == "SUCCESS" {
				success += 1
			}
			if job.Status == "READY" {
				ready += 1
			}
		}

		assert.Equal(t, 0, pending)
		assert.Equal(t, 5, success)
		assert.Equal(t, 45, ready)
	})
}

func TestFailure(t *testing.T) {
	pool, cleanup := testutil.PG.CreateDb("failure_processing")
	defer cleanup()

	queries := sqlc.New(pool)
	resolver := Resolver{Queries: queries, DB: pool}
	ctx := executor.NewCtx(context.Background(), "test-executor")

	t.Run("Should fail commit after processing", func(t *testing.T) {
		for i := 0; i < 50; i++ {
			_, err := resolver.Mutation().CreateJob(ctx, &model.CreateJobArgs{
				RunAt: "@after 1 second",
				Name:  fmt.Sprintf("search-%d", i),
				State: "{}",
			})
			assert.Nil(t, err)
		}

		time.Sleep(1 * time.Second)

		fetch, err := resolver.Mutation().FetchForProcessing(ctx, 5)
		assert.Nil(t, err)
		assert.Len(t, fetch, 5)

		var commits []int64
		for _, job := range fetch {
			commits = append(commits, job.ID)
		}

		failedCommits, err := resolver.Mutation().FailJobs(ctx, commits)
		assert.Nil(t, err)
		assert.Len(t, failedCommits, 0)

		all, err := resolver.Query().SearchJobs(ctx, model.QueryJobsArgs{
			Limit:  100,
			Skip:   0,
			Filter: "search",
		})
		assert.Nil(t, err)

		pending := 0
		success := 0
		failure := 0
		ready := 0
		for _, job := range all {
			if job.Status == "PENDING" {
				pending += 1
			}
			if job.Status == "SUCCESS" {
				success += 1
			}
			if job.Status == "FAILURE" {
				failure += 1
			}
			if job.Status == "READY" {
				ready += 1
			}
		}

		assert.Equal(t, 0, pending)
		assert.Equal(t, 0, success)
		assert.Equal(t, 5, failure)
		assert.Equal(t, 45, ready)
	})
}

func TestRetry(t *testing.T) {
	pool, cleanup := testutil.PG.CreateDb("retry_processing")
	defer cleanup()

	queries := sqlc.New(pool)
	resolver := Resolver{Queries: queries, DB: pool}
	ctx := executor.NewCtx(context.Background(), "test-executor")

	t.Run("Should retry commit after processing", func(t *testing.T) {
		for i := 0; i < 50; i++ {
			_, err := resolver.Mutation().CreateJob(ctx, &model.CreateJobArgs{
				RunAt: "@after 1 second",
				Name:  fmt.Sprintf("search-%d", i),
				State: "{}",
			})
			assert.Nil(t, err)
		}

		time.Sleep(1 * time.Second)

		fetch, err := resolver.Mutation().FetchForProcessing(ctx, 5)
		assert.Nil(t, err)
		assert.Len(t, fetch, 5)

		var commits []int64
		for _, job := range fetch {
			commits = append(commits, job.ID)
		}

		failedCommits, err := resolver.Mutation().RetryJobs(ctx, commits)
		assert.Nil(t, err)
		assert.Len(t, failedCommits, 0)

		all, err := resolver.Query().SearchJobs(ctx, model.QueryJobsArgs{
			Limit:  100,
			Skip:   0,
			Filter: "search",
		})
		assert.Nil(t, err)

		pending := 0
		success := 0
		failure := 0
		ready := 0
		for _, job := range all {
			if job.Status == "PENDING" {
				pending += 1
			}
			if job.Status == "SUCCESS" {
				success += 1
			}
			if job.Status == "FAILURE" {
				failure += 1
			}
			if job.Status == "READY" {
				ready += 1
			}
		}

		assert.Equal(t, 0, pending)
		assert.Equal(t, 0, success)
		assert.Equal(t, 0, failure)
		assert.Equal(t, 50, ready)
	})
}
