package graph

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"testing"

	"github.com/georgysavva/scany/pgxscan"
	_ "github.com/jackc/pgx/stdlib"
	"github.com/jackc/pgx/v4/pgxpool"
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
	resolver := Resolver{Queries: queries}

	t.Run("Should create job", func(t *testing.T) {
		job, err := resolver.Mutation().CreateJob(context.Background(), &model.CreateJobArgs{
			RunAt:  "@weekly",
			Name:   "lmao",
			State:  "{}",
			URL:    "http://localhost:1234",
			Method: "GET",
		})

		assert.Nil(t, err)
		assert.Equal(t, 1, countJobs(pool, "lmao"))
		assert.Equal(t, "@weekly", job.RunAt)
		assert.Equal(t, "lmao", job.Name.String)
		assert.Equal(t, "{}", job.State.String)
	})

	t.Run("Should update job by ID", func(t *testing.T) {
		job, err := resolver.Mutation().CreateJob(context.Background(), &model.CreateJobArgs{
			RunAt:  "@weekly",
			Name:   "update-lmao",
			State:  "{}",
			URL:    "http://localhost:1234",
			Method: "GET",
		})
		assert.Nil(t, err)

		updated, err := resolver.Mutation().UpdateJobByID(context.Background(), strconv.FormatInt(job.ID, 10), &model.UpdateJobArgs{
			RunAt:  ptrstring("@yearly"),
			State:  ptrstring(`{"hello":"world"}`),
			URL:    ptrstring("http://localhost:1234"),
			Method: ptrstring("POST"),
		})

		assert.Nil(t, err)
		assert.Equal(t, 1, countJobs(pool, "update-lmao"))
		assert.Equal(t, "@yearly", updated.RunAt)
		assert.Equal(t, "update-lmao", updated.Name.String)
		assert.Equal(t, `{"hello":"world"}`, updated.State.String)
	})

	t.Run("Should update job by name", func(t *testing.T) {
		job, err := resolver.Mutation().CreateJob(context.Background(), &model.CreateJobArgs{
			RunAt:  "@weekly",
			Name:   "update-lmao-by-name",
			State:  "{}",
			URL:    "http://localhost:1234",
			Method: "GET",
		})
		assert.Nil(t, err)

		updated, err := resolver.Mutation().UpdateJobByName(context.Background(), job.Name.String, &model.UpdateJobArgs{
			RunAt:  ptrstring("@yearly"),
			State:  ptrstring(`{"hello":"world"}`),
			URL:    ptrstring("http://localhost:1234"),
			Method: ptrstring("POST"),
		})

		assert.Nil(t, err)
		assert.Equal(t, 1, countJobs(pool, "update-lmao-by-name"))
		assert.Equal(t, "@yearly", updated.RunAt)
		assert.Equal(t, "update-lmao-by-name", updated.Name.String)
		assert.Equal(t, `{"hello":"world"}`, updated.State.String)
	})

	t.Run("Should conditionally update job config by name", func(t *testing.T) {
		job, err := resolver.Mutation().CreateJob(context.Background(), &model.CreateJobArgs{
			RunAt:  "@weekly",
			Name:   "update-cond-lmao-by-name",
			State:  "{}",
			URL:    "http://localhost:1234",
			Method: "GET",
		})
		assert.Nil(t, err)

		updated0, err := resolver.Mutation().UpdateJobByName(context.Background(), job.Name.String, &model.UpdateJobArgs{
			State: ptrstring(`{"hello":"world"}`),
		})

		assert.Nil(t, err)
		assert.Equal(t, 1, countJobs(pool, "update-cond-lmao-by-name"))
		assert.Equal(t, "@weekly", updated0.RunAt)
		assert.Equal(t, "update-cond-lmao-by-name", updated0.Name.String)
		assert.Equal(t, `{"hello":"world"}`, updated0.State.String)

		updated1, err := resolver.Mutation().UpdateJobByName(context.Background(), job.Name.String, &model.UpdateJobArgs{
			URL: ptrstring("http://localhost:9876"),
		})

		assert.Nil(t, err)
		assert.Equal(t, 1, countJobs(pool, "update-cond-lmao-by-name"))
		assert.Equal(t, "@weekly", updated1.RunAt)
		assert.Equal(t, "update-cond-lmao-by-name", updated1.Name.String)
		assert.Equal(t, `{"hello":"world"}`, updated1.State.String)

		updated2, err := resolver.Mutation().UpdateJobByName(context.Background(), job.Name.String, &model.UpdateJobArgs{
			Method: ptrstring("DELETE"),
		})

		assert.Nil(t, err)
		assert.Equal(t, 1, countJobs(pool, "update-cond-lmao-by-name"))
		assert.Equal(t, "@weekly", updated2.RunAt)
		assert.Equal(t, "update-cond-lmao-by-name", updated2.Name.String)
		assert.Equal(t, `{"hello":"world"}`, updated2.State.String)
	})

	t.Run("Should conditionally update job config by ID", func(t *testing.T) {
		job, err := resolver.Mutation().CreateJob(context.Background(), &model.CreateJobArgs{
			RunAt:  "@weekly",
			Name:   "update-cond-lmao-by-id",
			State:  "{}",
			URL:    "http://localhost:1234",
			Method: "GET",
		})
		assert.Nil(t, err)

		updated0, err := resolver.Mutation().UpdateJobByID(context.Background(), strconv.FormatInt(job.ID, 10), &model.UpdateJobArgs{
			State: ptrstring(`{"hello":"world"}`),
		})

		assert.Nil(t, err)
		assert.Equal(t, 1, countJobs(pool, "update-cond-lmao-by-id"))
		assert.Equal(t, "@weekly", updated0.RunAt)
		assert.Equal(t, "update-cond-lmao-by-id", updated0.Name.String)
		assert.Equal(t, `{"hello":"world"}`, updated0.State.String)

		updated1, err := resolver.Mutation().UpdateJobByID(context.Background(), strconv.FormatInt(job.ID, 10), &model.UpdateJobArgs{
			URL: ptrstring("http://localhost:9876"),
		})

		assert.Nil(t, err)
		assert.Equal(t, 1, countJobs(pool, "update-cond-lmao-by-id"))
		assert.Equal(t, "@weekly", updated1.RunAt)
		assert.Equal(t, "update-cond-lmao-by-id", updated1.Name.String)
		assert.Equal(t, `{"hello":"world"}`, updated1.State.String)

		updated2, err := resolver.Mutation().UpdateJobByID(context.Background(), strconv.FormatInt(job.ID, 10), &model.UpdateJobArgs{
			Method: ptrstring("DELETE"),
		})

		assert.Nil(t, err)
		assert.Equal(t, 1, countJobs(pool, "update-cond-lmao-by-id"))
		assert.Equal(t, "@weekly", updated2.RunAt)
		assert.Equal(t, "update-cond-lmao-by-id", updated2.Name.String)
		assert.Equal(t, `{"hello":"world"}`, updated2.State.String)
	})

	t.Run("Should delete job by name", func(t *testing.T) {
		_, err := resolver.Mutation().CreateJob(context.Background(), &model.CreateJobArgs{
			RunAt:  "@weekly",
			Name:   "delete-lmao-by-name",
			State:  "{}",
			URL:    "http://localhost:1234",
			Method: "GET",
		})
		assert.Nil(t, err)
		assert.Equal(t, 1, countJobs(pool, "delete-lmao-by-name"))

		_, err = resolver.Mutation().DeleteJobByName(context.Background(), "delete-lmao-by-name")

		assert.Nil(t, err)
		assert.Equal(t, 0, countJobs(pool, "delete-lmao-by-name"))
	})

	t.Run("Should delete job by ID", func(t *testing.T) {
		_, err := resolver.Mutation().CreateJob(context.Background(), &model.CreateJobArgs{
			RunAt:  "@weekly",
			Name:   "delete-lmao-by-id",
			State:  "{}",
			URL:    "http://localhost:1234",
			Method: "GET",
		})
		assert.Nil(t, err)
		assert.Equal(t, 1, countJobs(pool, "delete-lmao-by-id"))

		_, err = resolver.Mutation().DeleteJobByName(context.Background(), "delete-lmao-by-id")

		assert.Nil(t, err)
		assert.Equal(t, 0, countJobs(pool, "delete-lmao-by-id"))
	})

	t.Run("Should query job by ID", func(t *testing.T) {
		job, err := resolver.Mutation().CreateJob(context.Background(), &model.CreateJobArgs{
			RunAt:  "@weekly",
			Name:   "query-lmao-by-id",
			State:  "{}",
			URL:    "http://localhost:1234",
			Method: "GET",
		})
		assert.Nil(t, err)
		assert.Equal(t, 1, countJobs(pool, "query-lmao-by-id"))

		queried, err := resolver.Query().QueryJobByID(context.Background(), strconv.FormatInt(job.ID, 10))

		assert.Nil(t, err)
		assert.Equal(t, 1, countJobs(pool, "query-lmao-by-id"))
		assert.Equal(t, "@weekly", queried.RunAt)
		assert.Equal(t, "query-lmao-by-id", queried.Name.String)
		assert.Equal(t, `{}`, queried.State.String)
	})

	t.Run("Should query job by name", func(t *testing.T) {
		job, err := resolver.Mutation().CreateJob(context.Background(), &model.CreateJobArgs{
			RunAt:  "@weekly",
			Name:   "query-lmao-by-name",
			State:  "{}",
			URL:    "http://localhost:1234",
			Method: "GET",
		})
		assert.Nil(t, err)
		assert.Equal(t, 1, countJobs(pool, "query-lmao-by-name"))

		queried, err := resolver.Query().QueryJobByID(context.Background(), strconv.FormatInt(job.ID, 10))

		assert.Nil(t, err)
		assert.Equal(t, 1, countJobs(pool, "query-lmao-by-name"))
		assert.Equal(t, "@weekly", queried.RunAt)
		assert.Equal(t, "query-lmao-by-name", queried.Name.String)
		assert.Equal(t, `{}`, queried.State.String)
	})

	t.Run("Should search jobs", func(t *testing.T) {
		for i := 0; i < 50; i++ {
			_, err := resolver.Mutation().CreateJob(context.Background(), &model.CreateJobArgs{
				RunAt:  "@weekly",
				Name:   fmt.Sprintf("search-%d", i),
				State:  "{}",
				URL:    "http://localhost:1234",
				Method: "GET",
			})
			assert.Nil(t, err)
		}

		search0, err := resolver.Query().SearchJobs(context.Background(), model.QueryJobsArgs{
			Limit:  10,
			Skip:   0,
			Filter: "sear",
		})
		assert.Nil(t, err)
		assert.Len(t, search0, 10)

		for index, s := range search0 {
			assert.Equal(t, fmt.Sprintf("search-%d", index+0), s.Name.String)
		}

		search1, err := resolver.Query().SearchJobs(context.Background(), model.QueryJobsArgs{
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
