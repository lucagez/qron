package qron

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
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	tinyctx "github.com/lucagez/qron/ctx"
	"github.com/lucagez/qron/graph"
	"github.com/lucagez/qron/graph/generated"
	"github.com/lucagez/qron/graph/model"
	"github.com/lucagez/qron/migrations"
	"github.com/lucagez/qron/sqlc"
	"github.com/pressly/goose/v3"
)

type Client struct {
	Resolver      graph.Resolver
	MaxInFlight   uint64
	MaxFlushSize  int
	FlushInterval time.Duration
	PollInterval  time.Duration
	ResetInterval time.Duration
	OwnerSetter   func(http.Handler) http.Handler
	processedCh   chan Job
}

type Config struct {
	MaxInFlight   uint64
	MaxFlushSize  int
	FlushInterval time.Duration
	PollInterval  time.Duration
	ResetInterval time.Duration
	OwnerSetter   func(http.Handler) http.Handler
}

func NewClient(db *pgxpool.Pool, cfg Config) (Client, error) {
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
	if cfg.OwnerSetter == nil {
		cfg.OwnerSetter = tinyctx.ExecutorSetterMiddleware
	}
	if cfg.ResetInterval == 0 {
		cfg.ResetInterval = 60 * time.Second
	}
	if cfg.MaxFlushSize == 0 {
		cfg.MaxFlushSize = 100
	}

	return Client{
		Resolver:      resolver,
		MaxInFlight:   cfg.MaxInFlight,
		FlushInterval: cfg.FlushInterval,
		PollInterval:  cfg.PollInterval,
		OwnerSetter:   cfg.OwnerSetter,
		ResetInterval: cfg.ResetInterval,
		MaxFlushSize:  cfg.MaxFlushSize,
		processedCh:   make(chan Job),
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
			ids, err := t.Resolver.Queries.ResetTimeoutJobs(context.Background(), executorName)
			if len(ids) > 0 {
				log.Println("[RESETTING]", ids)
			}
			if err != nil {
				log.Println("error while resetting timed out jobs:", err)
			}
		}
	}
}

func (t *Client) flush(ctx context.Context, executorName string) {
	var commitBatch []model.CommitArgs
	var failBatch []model.CommitArgs
	var retryBatch []model.CommitArgs

	for {
		shouldFlush := false

		select {
		case <-ctx.Done():
			// Force flush after stop fetching
			shouldFlush = true
		case <-time.After(t.FlushInterval):
			shouldFlush = true
		case job := <-t.processedCh:
			commit := model.CommitArgs{
				ID: job.ID,
			}
			if job.State != "" {
				commit.State = &job.State
			}
			if job.Expr != "" {
				commit.Expr = &job.Expr
			}

			switch job.Status {
			case sqlc.TinyStatusSUCCESS:
				commitBatch = append(commitBatch, commit)
			case sqlc.TinyStatusFAILURE:
				failBatch = append(failBatch, commit)
			case sqlc.TinyStatusREADY:
				retryBatch = append(retryBatch, commit)
			}
			if len(commitBatch)+len(failBatch)+len(retryBatch) >= t.MaxFlushSize {
				shouldFlush = true
			}
		}

		if !shouldFlush {
			continue
		}

		if len(commitBatch)+len(failBatch)+len(retryBatch) > 0 {
			log.Println("[FLUSHING]", len(commitBatch), "commit.", len(failBatch), "fail.", len(retryBatch), "retry.")
		}

		// TODO: Handle failed commits + flush errors
		if len(commitBatch) > 0 {
			_, err := t.Resolver.Mutation().CommitJobs(ctx, executorName, commitBatch)
			if err != nil {
				log.Println(err)
			}
			commitBatch = []model.CommitArgs{}
		}
		if len(failBatch) > 0 {
			_, err := t.Resolver.Mutation().FailJobs(ctx, executorName, failBatch)
			if err != nil {
				log.Println(err)
			}
			failBatch = []model.CommitArgs{}
		}
		if len(retryBatch) > 0 {
			_, err := t.Resolver.Mutation().RetryJobs(ctx, executorName, retryBatch)
			if err != nil {
				log.Println(err)
			}
			retryBatch = []model.CommitArgs{}
		}
	}
}

func (c *Client) Close() {
	c.Resolver.DB.Close()
}

func (c *Client) Fetch(ctx context.Context, executorName string) chan Job {
	// ctx, cancel := context.WithCancel(ctx)
	ch := make(chan Job)

	go c.flush(ctx, executorName)
	go c.reset(ctx, executorName)

	go func() {
		for {
			select {
			case <-ctx.Done():
				close(ch)
				return
			// TODO: replace with ticker!
			case <-time.After(c.PollInterval):
				jobs, err := c.Resolver.
					Mutation().
					FetchForProcessing(ctx, executorName, int(c.MaxInFlight))
				if len(jobs) > 0 {
					log.Println("[FETCHING]", len(jobs), "jobs")
				}
				if err != nil {
					// TODO: how to handle err?
					log.Println(err)
				}
				for _, job := range jobs {
					ch <- Job{TinyJob: job, ch: c.processedCh}
				}
			}
		}
	}()

	return ch
}

func (c *Client) Handler() http.Handler {
	router := chi.NewRouter()
	api := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{
		Resolvers: &c.Resolver,
	}))

	router.Use(c.OwnerSetter)
	router.Handle("/graphql", api)
	router.Handle("/", playground.Handler("GraphQL Playground", "/graphql"))

	return router
}

func (c *Client) CreateJob(ctx context.Context, executorName string, args model.CreateJobArgs) (sqlc.TinyJob, error) {
	return c.Resolver.Mutation().CreateJob(
		ctx,
		executorName,
		args,
	)
}

func (c *Client) BatchCreateJobs(ctx context.Context, executorName string, args []model.CreateJobArgs) ([]int64, error) {
	return c.Resolver.Mutation().BatchCreateJobs(
		ctx,
		executorName,
		args,
	)
}

func (c *Client) UpdateJobByName(ctx context.Context, executorName, name string, args model.UpdateJobArgs) (sqlc.TinyJob, error) {
	return c.Resolver.Mutation().UpdateJobByName(
		ctx,
		executorName,
		name,
		args,
	)
}

func (c *Client) UpdateJobByID(ctx context.Context, executorName string, id int64, args model.UpdateJobArgs) (sqlc.TinyJob, error) {
	return c.Resolver.Mutation().UpdateJobByID(
		ctx,
		executorName,
		id,
		args,
	)
}

func (c *Client) DeleteJobByName(ctx context.Context, executorName, name string) (sqlc.TinyJob, error) {
	return c.Resolver.Mutation().DeleteJobByName(
		ctx,
		executorName,
		name,
	)
}

func (c *Client) DeleteJobByID(ctx context.Context, executorName string, id int64) (sqlc.TinyJob, error) {
	return c.Resolver.Mutation().DeleteJobByID(
		ctx,
		executorName,
		id,
	)
}

func (c *Client) SearchJobs(ctx context.Context, executorName string, args model.QueryJobsArgs) ([]sqlc.TinyJob, error) {
	return c.Resolver.Query().SearchJobs(
		ctx,
		executorName,
		args,
	)
}

func (c *Client) QueryJobByName(ctx context.Context, executorName, name string) (sqlc.TinyJob, error) {
	return c.Resolver.Query().QueryJobByName(
		ctx,
		executorName,
		name,
	)
}

func (c *Client) QueryJobByID(ctx context.Context, executorName string, id int64) (sqlc.TinyJob, error) {
	return c.Resolver.Query().QueryJobByID(
		ctx,
		executorName,
		id,
	)
}

func (c *Client) StopJob(ctx context.Context, executorName string, id int64) (sqlc.TinyJob, error) {
	return c.Resolver.Mutation().StopJob(
		ctx,
		executorName,
		id,
	)
}

func (c *Client) RestartJob(ctx context.Context, executorName string, id int64) (sqlc.TinyJob, error) {
	return c.Resolver.Mutation().RestartJob(
		ctx,
		executorName,
		id,
	)
}

func (c *Client) Migrate() error {
	goose.SetDialect("postgres")
	goose.SetBaseFS(migrations.MigrationsFS)
	// migrations are scoped so not to interfere with other
	// schema diffing tools.
	goose.SetTableName("qron.qron_migrations")

	dsn := c.Resolver.DB.Config().ConnConfig.ConnString()
	migrationClient, err := sql.Open("pgx", dsn)
	if err != nil {
		return err
	}

	_, err = migrationClient.Exec("create schema if not exists qron")
	if err != nil {
		return err
	}

	return goose.Up(migrationClient, ".")
}

// TODO: This should be `InflightJob` as it has
// additional methods for committing/failing job.
// TODO: Client should not expose sqlc.TinyJob as type.
type Job struct {
	// TODO: RENAME TO QRON
	sqlc.TinyJob
	ch chan<- Job
}

func (j Job) isOneShot() bool {
	return strings.HasPrefix(j.Expr, "@at") || strings.HasPrefix(j.Expr, "@after")
}

func (j Job) Commit() {
	if j.isOneShot() {
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
