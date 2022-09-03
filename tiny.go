package tinyq

//go:generate mage gen

import (
	"context"
	"database/sql"
	"github.com/cenkalti/backoff/v4"
	"github.com/deepmap/oapi-codegen/pkg/middleware"
	"github.com/georgysavva/scany/pgxscan"
	_ "github.com/jackc/pgx/stdlib"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/lucagez/tinyq/api"
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
	finishedJobs  chan Job
}

// ExecResponse
// TODO: Response should have:
// - state, as it should update existing state
type ExecResponse struct {
	// TODO: How should the response from a job look like?
	Status string `json:"status"`
}

type Executor interface {
	// Run should return any payload as result of the run
	// TODO: Improve signature when architecture is working
	Run(*Job) error
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
		finishedJobs:  make(chan Job),
	}
}

func (t *TinyQ) IncreaseInFlight() {
	// TODO: Use atomics
	//t.mu.Lock()
	//defer t.mu.Unlock()
	//t.MaxInFlight++
	atomic.AddUint64(&t.MaxInFlight, 1)
}

func (t *TinyQ) DecreaseInFlight() {
	//t.mu.Lock()
	//defer t.mu.Unlock()
	//t.MaxInFlight--
	atomic.AddUint64(&t.MaxInFlight, ^uint64(0))
}

type Job struct {
	Id              int            `json:"id" db:"id"`
	Status          Status         `json:"status" db:"status"`
	LastRunAt       sql.NullTime   `json:"last_run_at" db:"last_run_at"`
	CreatedAt       time.Time      `json:"created_at" db:"created_at"`
	RunAt           string         `json:"run_at" db:"run_at"`
	Name            sql.NullString `json:"name" db:"name"`
	ExecutionAmount int            `json:"execution_amount" db:"execution_amount"`
	Timeout         int            `json:"timeout" db:"timeout"`
	State           string         `json:"state" db:"state"`
	Config          string         `json:"config" db:"config"`
	Kind            Kind           `json:"kind" db:"kind"`
	ExecutorType    string         `json:"executor" db:"executor"`
}

func (t *TinyQ) Process(ctx context.Context, job Job) {
	t.DecreaseInFlight()
	defer t.IncreaseInFlight()

	// TODO: Use timeout
	// TODO: In case of failure use max_attempts
	// TODO: Pick executor (HTTP)
	executor, ok := t.Executors[job.ExecutorType]
	if !ok {
		log.Println("no executor found for current job:", job)
		return
	}

	err := backoff.Retry(func() error {
		// TODO: keep an eye on this. ideally better to pass by value
		// to minimize heap allocations
		return executor.Run(&job)
	}, backoff.NewExponentialBackOff())

	t.finishedJobs <- job

	if err != nil {
		// Handle error.
		log.Println("unrecoverable. max attempts finished")
		//status = FAILURE
		return
	}
}

func (t *TinyQ) Fetch() ([]Job, error) {
	time.Sleep(t.PollInterval)

	//tx, err := t.Db.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.Serializable})
	tx, err := t.Db.BeginTx(context.Background(), pgx.TxOptions{})
	if err != nil {
		return nil, err
	}

	// TODO: Add check for not running ever a job if
	// `last_run_at` happened less than x seconds ago
	var jobs []Job
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
