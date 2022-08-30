package tinyq

//go:generate mage gen

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/cenkalti/backoff/v4"
	"github.com/georgysavva/scany/pgxscan"
	_ "github.com/jackc/pgx/stdlib"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/lucagez/tinyq/api"
	"net/http"
	"sync"
	"time"
)

type TinyQ struct {
	Db            *pgxpool.Pool
	MaxInFlight   uint64
	Limiter       chan int
	mu            sync.Mutex
	FlushInterval time.Duration
	PollInterval  time.Duration

	// Experiment
	httpClient   *http.Client
	limiter      chan int
	finishedJobs chan Job
}

func NewTinyQ(db *pgxpool.Pool, poll, flushInterval time.Duration, maxInFlight uint64) TinyQ {

	transport := &http.Transport{
		MaxIdleConns:        10, // global number of idle conns
		MaxIdleConnsPerHost: 5,  // subset of MaxIdleConns, per-host
		// declare a conn idle after 10 seconds. too low and conns are recycled too much, too high and conns aren't recycled enough
		IdleConnTimeout: 10 * time.Second,
		// DisableKeepAlives: true, // this means create a new connection per request. not recommended
	}
	httpClient := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	return TinyQ{
		Db:            db,
		MaxInFlight:   maxInFlight,
		FlushInterval: flushInterval,
		PollInterval:  poll,
		Limiter:       make(chan int, maxInFlight),
		mu:            sync.Mutex{},
		httpClient:    httpClient,
		limiter:       make(chan int, 50), // TODO: test limiter for httpclient
		finishedJobs:  make(chan Job),
	}
}

func (t *TinyQ) IncreaseInFlight() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.MaxInFlight++
}

func (t *TinyQ) DecreaseInFlight() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.MaxInFlight--
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
	Kind            Kind           `json:"kind" db:"kind"`
}

func (t *TinyQ) Process(ctx context.Context, job Job) {
	t.DecreaseInFlight()
	defer t.IncreaseInFlight()

	// TODO: Use timeout
	// TODO: In case of failure use max_attempts
	// TODO: Pick executor (HTTP)

	// Process and get next state + status
	//duration := time.Duration(rand.Intn(20)) * time.Second
	//time.Sleep(duration)
	//fmt.Println("Processed:", job.Id, "in", duration, "seconds")
	//t.httpClient.Get(nil, "http://localhost:8081/counter")

	// Limit amount of in-flight http requests
	//fmt.Println("Pushing")
	t.limiter <- 0

	err := backoff.Retry(func() error {
		res, err := t.httpClient.Get("http://localhost:8081/counter")
		if err != nil {
			fmt.Println("Http error!", err)
			return err
		}
		return res.Body.Close()
	}, backoff.NewExponentialBackOff())

	<-t.limiter
	//fmt.Println("Unpushing")

	job.Status = READY
	if job.Kind == TASK {
		// TODO: compute based on result of processing
		job.Status = SUCCESS
	}

	if err != nil {
		// Handle error.
		fmt.Println("Http UNRECOVERABLE!")
		//status = FAILURE
		return
	}

	t.finishedJobs <- job

	if err != nil {
		fmt.Println("There's an error! do something", err)
	}

}

func (t *TinyQ) Fetch() ([]Job, error) {
	time.Sleep(t.PollInterval)
	fmt.Println("Fetching", t.MaxInFlight, "jobs...")

	//tx, err := t.Db.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelSerializable})
	tx, err := t.Db.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.Serializable})
	fmt.Println("Something happened right after")
	if err != nil {
		fmt.Println("Something was an for TX", err)
		return nil, err
	}
	fmt.Println("Something was NOT an err for TX")

	var jobs []Job
	result, err := tx.Query(context.Background(), `
		with due_jobs as (
			select *
			from tiny.job
			where tiny.is_due(run_at, coalesce(last_run_at, created_at))
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

	//err = sqlx.StructScan(result, &jobs)
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

func (t *TinyQ) flush() {
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
			fmt.Printf("====== FLUSHING JOBS: %d ======\n", batch.Len())
			// TODO: Check for errors
			tx, err := t.Db.Begin(context.Background())
			if err != nil {
				fmt.Println("error while acquiring tx", err)
			}
			br := tx.SendBatch(context.Background(), batch)
			if br.Close() != nil {
				fmt.Println("error while closing batch", err)
			}
			if tx.Commit(context.Background()) != nil {
				tx.Rollback(context.Background())
				fmt.Println("error while committing tx")
			}

			// reset
			batch = &pgx.Batch{}
		}
	}
}

func (t *TinyQ) Listen() {

	e := echo.New()

	e.GET("/demo", func(c echo.Context) error {
		var result string
		return c.JSON(http.StatusOK, result)
	})

	api.RegisterHandlers(e, api.Api{})

	fmt.Println("Listening 🦕")

	// Batch save in the background
	go t.flush()

	e.Start(":1234")
}
