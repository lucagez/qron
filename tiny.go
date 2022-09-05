package tinyq

//go:generate mage gen

import (
	"context"
	"github.com/cenkalti/backoff/v4"
	"github.com/deepmap/oapi-codegen/pkg/middleware"
	"github.com/georgysavva/scany/pgxscan"
	_ "github.com/jackc/pgx/stdlib"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/lucagez/tinyq/api"
	"github.com/lucagez/tinyq/executor"
	"log"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"time"
)

type TinyQ struct {
	Db            *pgxpool.Pool
	MaxInFlight   uint64
	mu            sync.Mutex
	FlushInterval time.Duration
	PollInterval  time.Duration
	Executors     map[string]Executor
	finishedJobs  chan executor.Job
}

type Executor interface {
	// Run Invoke job as defined by executor.
	// Updating job state / status is up to the sdk
	Run(executor.Job) (executor.Job, error)
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
		mu:            sync.Mutex{},
		Executors:     config.Executors,
		finishedJobs:  make(chan executor.Job),
	}
}

func (t *TinyQ) IncreaseInFlight() {
	atomic.AddUint64(&t.MaxInFlight, 1)
}

func (t *TinyQ) DecreaseInFlight() {
	atomic.AddUint64(&t.MaxInFlight, ^uint64(0))
}

func (t *TinyQ) Process(ctx context.Context, job executor.Job) {
	t.DecreaseInFlight()
	defer t.IncreaseInFlight()

	// TODO: Use timeout
	// TODO: In case of failure use max_attempts
	exe, ok := t.Executors[job.ExecutorType]
	if !ok {
		log.Println("no executor found for current job:", job)
		return
	}

	var updatedJob executor.Job
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

func (t *TinyQ) Fetch() ([]executor.Job, error) {
	time.Sleep(t.PollInterval)

	//tx, err := t.Db.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.Serializable})
	tx, err := t.Db.BeginTx(context.Background(), pgx.TxOptions{})
	if err != nil {
		return nil, err
	}

	// TODO: Add check for not running ever a job if
	// `last_run_at` happened less than x seconds ago
	var jobs []executor.Job
	result, err := tx.Query(context.Background(), `
		with due_jobs as (
			select *
			from tiny.job
			where tiny.is_due(run_at, coalesce(last_run_at, created_at), now())
			and status = 'READY'
			-- worker limit
			limit $1 for update
			skip locked
		)
		update tiny.job
		set status      = 'PENDING',
			last_run_at = now()
		from due_jobs
		where due_jobs.id = tiny.job.id
		returning due_jobs.*;
	`, t.MaxInFlight)
	if err != nil {
		tx.Rollback(context.Background())
		return nil, err
	}

	err = pgxscan.ScanAll(&jobs, result)
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
	// TODO: Make sure only error rows are ignored not the whole batch
	// keep track of errors!
	// TODO: Check if behavior is correct but should be fine.
	// Until a job is updated (with this batch) it should not be acquired from
	// any other worker
	batch := &pgx.Batch{}
	ticker := time.NewTicker(t.FlushInterval)
	for {
		shouldFlush := false

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			shouldFlush = true
		case job := <-t.finishedJobs:
			batch.Queue(`
				update tiny.job
				set last_run_at = $1,
					-- TODO: update
					state = $2,
					status = $3
				where id = $4
			`, time.Now(), job.State, job.Status, job.Id)
			if batch.Len() > 100 {
				shouldFlush = true
			}
		}

		if shouldFlush {
			tx, err := t.Db.Begin(context.Background())
			if err != nil {
				log.Println("error while acquiring tx", err)
				continue
			}

			// TODO: Check for errors inside batch
			br := tx.SendBatch(context.Background(), batch)
			err = br.Close()
			if err != nil {
				log.Println("error while closing batch", err)
			}

			err = tx.Commit(context.Background())
			if err != nil {
				log.Println("error while committing tx", err, tx.Rollback(context.Background()))
			}

			// reset
			batch = &pgx.Batch{}
		}
	}
}

func (t *TinyQ) start(ctx context.Context) {
	for {
		select {
		case <-time.After(t.PollInterval):
			result, err := t.Fetch()
			if err != nil {
				log.Println("error:", err)
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

	spec, err := api.GetSwagger()
	if err != nil {
		log.Fatalln("error while loading openapi spec")
	}

	spec.Servers = nil

	e.Use(echomiddleware.Logger())
	e.Use(middleware.OapiRequestValidator(spec))
	api.RegisterHandlers(e, api.Api{})

	ctx, cancel := context.WithCancel(context.Background())

	// TODO: they should be different contexts as behavior should
	// be: stop polling, keep flushing. Then exit
	go t.start(ctx)
	go t.flush(ctx)
	go e.Start(":1234")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	cancel()

	err = e.Shutdown(ctx)
	if err != nil {
		log.Fatalln(err)
	}
}
