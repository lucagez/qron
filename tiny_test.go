package tinyq

import (
	"context"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/lucagez/tinyq/executor"
	"github.com/lucagez/tinyq/testutil"
	"github.com/stretchr/testify/assert"
	"log"
	"os"
	"sync"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	// db pool factory is initialized by importing it

	code := m.Run()

	if err := testutil.PG.Teardown(); err != nil {
		log.Fatalln("could not purge db pool:", err)
	}

	os.Exit(code)
}

func insertJob(db *pgxpool.Pool, n int, expr, state, executor string) {
	_, err := db.Exec(context.Background(), `
		insert into tiny.job(run_at, status, state, kind, executor, config)
		select $1, 'READY', $2, tiny.find_kind($1), $3, '{}'
		from generate_series(1, $4)
	`, expr, state, executor, n)
	if err != nil {
		log.Fatalln("failed to create job", n, expr, state, executor, err)
	}
}

func clearJob(db *pgxpool.Pool) {
	_, err := db.Exec(context.Background(), `
		delete from tiny.job
	`)
	if err != nil {
		log.Fatalln("failed to delete jobs", err)
	}
}

func countJobs(db *pgxpool.Pool, status string) int {
	rows, err := db.Query(context.Background(), `
		select count(*) from tiny.job where status = $1
	`, status)
	if err != nil {
		log.Fatalln("failed to count jobs", err)
	}
	var count int
	pgxscan.ScanOne(&count, rows)
	return count
}

type CounterExecutor struct {
	count *int
	mu    *sync.Mutex
}

func (c CounterExecutor) Run(job *executor.Job) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	*c.count++
	job.Status = executor.SUCCESS
	return nil
}

func TestTiny(t *testing.T) {
	t.Run("Should fetch jobs", func(t *testing.T) {
		db, cleanup := testutil.PG.CreateDb("fetch_jobs")
		defer cleanup()

		q := NewTinyQ(Config{
			Db:            db,
			FlushInterval: 1 * time.Second,
			PollInterval:  1 * time.Millisecond,
			MaxInFlight:   10,
		})
		defer clearJob(db)

		insertJob(db, 10, "@every 10 ms", "{}", "INCREMENT")

		assert.Equal(t, 10, countJobs(db, "READY"))

		jobs, err := q.Fetch()
		assert.Nil(t, err)
		assert.Len(t, jobs, 0)

		assert.Equal(t, 0, countJobs(db, "PENDING"))

		// wait for job schedule to expire
		time.Sleep(10 * time.Millisecond)

		jobs, err = q.Fetch()
		assert.Nil(t, err)
		assert.Len(t, jobs, 10)

		assert.Equal(t, 10, countJobs(db, "PENDING"))
	})

	t.Run("Should fetch jobs in parallel without overlaps", func(t *testing.T) {
		db, cleanup := testutil.PG.CreateDb("no_overlaps")
		defer cleanup()

		q0 := NewTinyQ(Config{
			Db:            db,
			FlushInterval: 1 * time.Second,
			PollInterval:  1 * time.Millisecond,
			MaxInFlight:   50,
		})
		q1 := NewTinyQ(Config{
			Db:            db,
			FlushInterval: 1 * time.Second,
			PollInterval:  1 * time.Millisecond,
			MaxInFlight:   50,
		})
		defer clearJob(db)

		insertJob(db, 110, "@every 10 ms", "{}", "INCREMENT")

		assert.Equal(t, 110, countJobs(db, "READY"))

		time.Sleep(11 * time.Millisecond)

		// check that jobs are not fetched twice
		var q0jobs []executor.Job
		var q1jobs []executor.Job
		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			var err error
			q0jobs, err = q0.Fetch()
			assert.Nil(t, err)
		}()
		go func() {
			defer wg.Done()
			var err error
			q1jobs, err = q1.Fetch()
			assert.Nil(t, err)
		}()

		wg.Wait()

		for _, j0 := range q0jobs {
			for _, j1 := range q1jobs {
				if j0.Id == j1.Id {
					assert.Fail(t, "found overlapping jobs", j0.Id, j1.Id)
				}
			}
		}

		assert.Len(t, q0jobs, 50)
		assert.Len(t, q1jobs, 50)
		assert.Equal(t, 10, countJobs(db, "READY"))
		assert.Equal(t, 100, countJobs(db, "PENDING"))
	})

	t.Run("Should process jobs concurrently without overlap", func(t *testing.T) {
		db, cleanup := testutil.PG.CreateDb("concurrent_process")
		defer cleanup()

		c := 0
		exe := CounterExecutor{count: &c, mu: &sync.Mutex{}}
		q0 := NewTinyQ(Config{
			Db:            db,
			FlushInterval: 1 * time.Second,
			PollInterval:  1 * time.Millisecond,
			MaxInFlight:   10,
			Executors: map[string]Executor{
				"INCREMENT": exe,
			},
		})
		q1 := NewTinyQ(Config{
			Db:            db,
			FlushInterval: 1 * time.Second,
			PollInterval:  1 * time.Millisecond,
			MaxInFlight:   15,
			Executors: map[string]Executor{
				"INCREMENT": exe,
			},
		})
		defer clearJob(db)

		insertJob(db, 110, "@every 10 ms", "{}", "INCREMENT")

		assert.Equal(t, 110, countJobs(db, "READY"))

		time.Sleep(10 * time.Millisecond)

		// concurrent processing
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go q0.start(ctx)
		go q1.start(ctx)

		time.Sleep(15 * time.Millisecond)

		assert.Equal(t, 85, countJobs(db, "READY"))
		assert.Equal(t, 25, countJobs(db, "PENDING"))
		assert.Equal(t, 25, *exe.count)
	})

	t.Run("Should flush jobs while they get completed", func(t *testing.T) {
		db, cleanup := testutil.PG.CreateDb("flush_test")
		defer cleanup()

		c := 0
		exe := CounterExecutor{count: &c, mu: &sync.Mutex{}}
		q := NewTinyQ(Config{
			Db:            db,
			FlushInterval: 5 * time.Millisecond,
			PollInterval:  1 * time.Millisecond,
			MaxInFlight:   10,
			Executors: map[string]Executor{
				"INCREMENT": exe,
			},
		})
		defer clearJob(db)

		insertJob(db, 30, "@every 10 ms", "{}", "INCREMENT")

		assert.Equal(t, 30, countJobs(db, "READY"))

		// concurrent processing
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go q.start(ctx)
		go q.flush(ctx)

		assert.Equal(t, 30, countJobs(db, "READY"))
		assert.Equal(t, 0, countJobs(db, "PENDING"))
		assert.Equal(t, 0, countJobs(db, "SUCCESS"))
		assert.Equal(t, 0, *exe.count)

		time.Sleep(40 * time.Millisecond)

		assert.Equal(t, 0, countJobs(db, "READY"))
		assert.Equal(t, 0, countJobs(db, "PENDING"))
		assert.Equal(t, 30, countJobs(db, "SUCCESS"))
		assert.Equal(t, 30, *exe.count)
	})
}
