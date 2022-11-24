package tinyq

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"sync/atomic"
	"time"

	"github.com/cenkalti/backoff/v4"
	_ "github.com/jackc/pgx/stdlib"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/lucagez/tinyq/sqlc"
)

type TinyQ struct {
	Db            *pgxpool.Pool
	MaxInFlight   uint64
	FlushInterval time.Duration
	PollInterval  time.Duration
	Executors     map[string]Executor
	finishedJobs  chan sqlc.TinyJob
	queries       *sqlc.Queries
}

type Executor interface {
	// Run Invoke job as defined by executor.
	// Updating job state / status is up to the sdk
	Run(sqlc.TinyJob) (sqlc.TinyJob, error)
}

type Config struct {
	Db            *pgxpool.Pool
	FlushInterval time.Duration
	PollInterval  time.Duration
	MaxInFlight   uint64
	Executors     map[string]Executor
}

func NewTinyQ(config Config) TinyQ {
	return TinyQ{
		Db:            config.Db,
		MaxInFlight:   config.MaxInFlight,
		FlushInterval: config.FlushInterval,
		PollInterval:  config.PollInterval,
		Executors:     config.Executors,
		finishedJobs:  make(chan sqlc.TinyJob),
		queries:       sqlc.New(config.Db),
	}
}

func (t *TinyQ) IncreaseInFlight() {
	atomic.AddUint64(&t.MaxInFlight, 1)
}

func (t *TinyQ) DecreaseInFlight() {
	atomic.AddUint64(&t.MaxInFlight, ^uint64(0))
}

func (t *TinyQ) Process(ctx context.Context, job sqlc.TinyJob) {
	t.DecreaseInFlight()
	defer t.IncreaseInFlight()

	// TODO: Use timeout
	// TODO: In case of failure use max_attempts
	exe, ok := t.Executors[job.Executor]
	if !ok {
		log.Println("no executor found for current job:", job)
		return
	}

	var updatedJob sqlc.TinyJob
	err := backoff.Retry(func() error {
		var execErr error
		updatedJob, execErr = exe.Run(job)
		return execErr
	}, backoff.NewExponentialBackOff())

	// update job to be saved with outcome from
	// sdk execution:

	// a user can choose if terminate their jobs
	job.Status = updatedJob.Status

	// tinyq can never read job sensitive info
	job.State = updatedJob.State

	// user can decide to update execution method
	// e.g. Delay next run by 50 minutes
	job.RunAt = updatedJob.RunAt

	t.finishedJobs <- job

	if err != nil {
		// Handle error.
		log.Println("unrecoverable. max attempts finished")
		//status = FAILURE
		return
	}
}

func (t *TinyQ) Fetch() ([]sqlc.TinyJob, error) {
	tx, err := t.Db.BeginTx(context.Background(), pgx.TxOptions{})
	if err != nil {
		return nil, err
	}

	q := t.queries.WithTx(tx)
	jobs, err := q.FetchDueJobs(context.Background(), sqlc.FetchDueJobsParams{
		Limit:    int32(t.MaxInFlight),
		Executor: "TODO",
	})
	if err != nil {
		tx.Rollback(context.Background())
		return nil, err
	}

	if tx.Commit(context.Background()) != nil {
		return nil, err
	}

	return jobs, nil
}

func (t *TinyQ) flush(ctx context.Context) {
	// TODO: Check if behavior is correct but should be fine.
	// Until a job is updated (with this batch) it should not be acquired from
	// any other worker

	var batch []sqlc.BatchUpdateJobsParams
	ticker := time.NewTicker(t.FlushInterval)
	for {
		shouldFlush := false

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			shouldFlush = true
		case job := <-t.finishedJobs:
			batch = append(batch, sqlc.BatchUpdateJobsParams{
				ID: job.ID,
				LastRunAt: sql.NullTime{
					Time:  time.Now(),
					Valid: true,
				},
				State:  job.State,
				Status: job.Status,
			})
			if len(batch) > 100 {
				shouldFlush = true
			}
		}

		if shouldFlush && len(batch) > 0 {
			t.queries.BatchUpdateJobs(context.Background(), batch).Exec(func(i int, err error) {
				// TODO: What to do in case of flush failures?
				if err != nil {
					log.Println("error while flushing job ", batch[i], ":", err)
				}
			})

			// reset
			batch = []sqlc.BatchUpdateJobsParams{}
		}
	}
}

func (t *TinyQ) start(ctx context.Context) {
	for {
		select {
		case <-time.After(t.PollInterval):
			result, err := t.Fetch()
			if err != nil {
				log.Println("error while fetching due jobs:", err)
				return
			}

			for _, job := range result {
				go t.Process(context.Background(), job)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (t *TinyQ) Listen() {
	e := echo.New()

	e.Use(echomiddleware.Logger())

	ctx, cancel := context.WithCancel(context.Background())

	// TODO: they should be different contexts as behavior should
	// be: stop polling, keep flushing. Then exit
	go t.start(ctx)
	go t.flush(ctx)
	go e.Start(":1234")

	// TODO: Load gql server here
	// TODO: Switch echo for chi at this point? Or just basic net/http

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	cancel()

	err := e.Shutdown(ctx)
	if err != nil {
		log.Fatalln(err)
	}
}
