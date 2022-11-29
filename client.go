package tinyq

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v4/pgxpool"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/lucagez/tinyq/executor"
	"github.com/lucagez/tinyq/graph"
	"github.com/lucagez/tinyq/graph/generated"
	"github.com/lucagez/tinyq/graph/model"
	"github.com/lucagez/tinyq/migrations"
	"github.com/lucagez/tinyq/sqlc"
	"github.com/pressly/goose/v3"
)

// var processedCh = make(chan Job)

type Client struct {
	resolver       graph.Resolver
	dsn            string
	MaxInFlight    uint64
	FlushInterval  time.Duration
	PollInterval   time.Duration
	ResetInterval  time.Duration
	ExecutorSetter func(http.Handler) http.Handler
	processedCh    chan Job
}

type Config struct {
	Dsn            string
	MaxInFlight    uint64
	FlushInterval  time.Duration
	PollInterval   time.Duration
	ResetInterval  time.Duration
	ExecutorSetter func(http.Handler) http.Handler
}

// TODO: There should be alway a global job that make sure
// that tasks that exceed timeouts get cleared and set back to READY.
// -> this behavior should be configurable?
func NewClient(cfg Config) (Client, error) {
	db, err := pgxpool.Connect(context.Background(), cfg.Dsn)
	if err != nil {
		return Client{}, err
	}
	queries := sqlc.New(db)
	resolver := graph.Resolver{Queries: queries, DB: db}

	if cfg.MaxInFlight == 0 {
		cfg.MaxInFlight = 100
	}
	if cfg.FlushInterval == 0 {
		cfg.FlushInterval = 1 * time.Second
	}
	if cfg.PollInterval == 0 {
		cfg.PollInterval = 1 * time.Second
	}
	if cfg.ExecutorSetter == nil {
		cfg.ExecutorSetter = executor.ExecutorSetterMiddleware
	}
	if cfg.ResetInterval == 0 {
		cfg.ResetInterval = 1 * time.Second
	}

	return Client{
		resolver:       resolver,
		dsn:            cfg.Dsn,
		MaxInFlight:    cfg.MaxInFlight,
		FlushInterval:  cfg.FlushInterval,
		PollInterval:   cfg.PollInterval,
		ExecutorSetter: cfg.ExecutorSetter,
		ResetInterval:  cfg.ResetInterval,
		processedCh:    make(chan Job),
	}, nil
}

func (t *Client) IncreaseInFlight() {
	atomic.AddUint64(&t.MaxInFlight, 1)
}

func (t *Client) DecreaseInFlight() {
	atomic.AddUint64(&t.MaxInFlight, ^uint64(0))
}

func (t *Client) reset(ctx context.Context, executorName string) {
	ticker := time.NewTicker(t.ResetInterval)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, err := t.resolver.Queries.ResetTimeoutJobs(context.Background(), executorName)
			if err != nil {
				log.Println("error while resetting timed out jobs:", err)
			}
		}
	}
}

// TODO: Should optimize `flush` behavior. It currently
// weights for 80% of total ram
func (t *Client) flush(ctx context.Context, executorName string) {
	var commitBatch []int64
	var failBatch []int64
	var retryBatch []int64
	ticker := time.NewTicker(t.FlushInterval)

	for {
		shouldFlush := false

		select {
		case <-ctx.Done():
			// time.Sleep(10 * time.Millisecond)
			return
		case <-ticker.C:
			shouldFlush = true
		case job := <-t.processedCh:
			switch job.Status {
			case sqlc.TinyStatusSUCCESS:
				commitBatch = append(commitBatch, job.ID)
			case sqlc.TinyStatusFAILURE:
				failBatch = append(failBatch, job.ID)
			case sqlc.TinyStatusREADY:
				retryBatch = append(retryBatch, job.ID)
			}
			if len(commitBatch)+len(failBatch)+len(retryBatch) > 100 {
				shouldFlush = true
			}
		}

		if !shouldFlush {
			continue
		}

		// TODO: Handle failed commits + flush errors
		if len(commitBatch) > 0 {
			_, err := t.resolver.Mutation().CommitJobs(executor.NewCtx(ctx, executorName), commitBatch)
			if err != nil {
				log.Println(err)
			}
			commitBatch = []int64{}
		}
		if len(failBatch) > 0 {
			_, err := t.resolver.Mutation().FailJobs(executor.NewCtx(ctx, executorName), failBatch)
			if err != nil {
				log.Println(err)
			}
			failBatch = []int64{}
		}
		if len(retryBatch) > 0 {
			_, err := t.resolver.Mutation().RetryJobs(executor.NewCtx(ctx, executorName), retryBatch)
			if err != nil {
				log.Println(err)
			}
			retryBatch = []int64{}
		}
	}
}

func (c *Client) Close() {
	c.resolver.DB.Close()
}

func (c *Client) Fetch(executorName string) (chan Job, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan Job)

	go c.flush(ctx, executorName)
	go c.reset(ctx, executorName)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(c.PollInterval):
				jobs, err := c.resolver.
					Mutation().
					FetchForProcessing(executor.NewCtx(ctx, executorName), int(c.MaxInFlight))
				if err != nil {
					// TODO: how to handle err?
					log.Println(err)
				}
				for _, job := range jobs {
					ch <- Job{
						job,
						c.processedCh,
					}
				}
			}
		}
	}()

	return ch, func() {
		close(ch)
		cancel()
	}
}

func (c *Client) Handler() http.Handler {
	router := chi.NewRouter()
	api := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{
		Resolvers: &c.resolver,
	}))

	router.Use(c.ExecutorSetter)
	router.Handle("/graphql", api)
	router.Handle("/", playground.Handler("GraphQL Playground", "/graphql"))

	return router
}

func (c *Client) CreateJob(executorName string, args model.CreateJobArgs) (sqlc.TinyJob, error) {
	return c.resolver.Mutation().CreateJob(
		executor.NewCtx(context.Background(), executorName),
		args,
	)
}

func (c *Client) UpdateJobByName(executorName, name string, args model.UpdateJobArgs) (sqlc.TinyJob, error) {
	return c.resolver.Mutation().UpdateJobByName(
		executor.NewCtx(context.Background(), executorName),
		name,
		args,
	)
}

func (c *Client) UpdateJobByID(executorName string, id int64, args model.UpdateJobArgs) (sqlc.TinyJob, error) {
	return c.resolver.Mutation().UpdateJobByID(
		executor.NewCtx(context.Background(), executorName),
		id,
		args,
	)
}

func (c *Client) DeleteJobByName(executorName, name string) (sqlc.TinyJob, error) {
	return c.resolver.Mutation().DeleteJobByName(
		executor.NewCtx(context.Background(), executorName),
		name,
	)
}

func (c *Client) DeleteJobByID(executorName string, id int64) (sqlc.TinyJob, error) {
	return c.resolver.Mutation().DeleteJobByID(
		executor.NewCtx(context.Background(), executorName),
		id,
	)
}

func (c *Client) SearchJobs(executorName string, args model.QueryJobsArgs) ([]sqlc.TinyJob, error) {
	return c.resolver.Query().SearchJobs(
		executor.NewCtx(context.Background(), executorName),
		args,
	)
}

func (c *Client) QueryJobByName(executorName, name string) (sqlc.TinyJob, error) {
	return c.resolver.Query().QueryJobByName(
		executor.NewCtx(context.Background(), executorName),
		name,
	)
}

func (c *Client) QueryJobByID(executorName string, id int64) (sqlc.TinyJob, error) {
	return c.resolver.Query().QueryJobByID(
		executor.NewCtx(context.Background(), executorName),
		id,
	)
}

func (c *Client) Migrate() error {
	goose.SetDialect("postgres")
	goose.SetBaseFS(migrations.MigrationsFS)

	migrationClient, err := sql.Open("pgx", c.dsn)
	if err != nil {
		return err
	}

	return goose.Up(migrationClient, ".")
}

type Job struct {
	sqlc.TinyJob
	ch chan<- Job
}

func (j Job) Commit() {
	if strings.HasPrefix(j.Expr, "@at") || strings.HasPrefix(j.Expr, "@after") {
		j.Status = sqlc.TinyStatusSUCCESS
	} else {
		// Else is cron. Should be ready to be picked up again
		j.Status = sqlc.TinyStatusREADY
	}
	j.ch <- j
}

func (j Job) Fail() {
	j.Status = sqlc.TinyStatusFAILURE
	j.ch <- j
}

func (j Job) Retry() {
	j.Status = sqlc.TinyStatusREADY
	j.ch <- j
}
