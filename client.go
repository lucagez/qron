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
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	tinyctx "github.com/lucagez/tinyq/ctx"
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
	MaxInFlight    uint64
	FlushInterval  time.Duration
	PollInterval   time.Duration
	ResetInterval  time.Duration
	ExecutorSetter func(http.Handler) http.Handler
}

// TODO: There should be alway a global job that make sure
// that tasks that exceed timeouts get cleared and set back to READY.
// -> this behavior should be configurable?
func NewClient(db *pgxpool.Pool, cfg Config) (Client, error) {
	// if cfg.Dsn != "" {
	// 	var err error
	// 	db, err = pgxpool.New(context.Background(), cfg.Dsn)
	// 	if err != nil {
	// 		return Client{}, err
	// 	}
	// }

	// if cfg.Conn != nil {
	// 	db = cfg.Conn
	// }

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
		cfg.ExecutorSetter = tinyctx.ExecutorSetterMiddleware
	}
	if cfg.ResetInterval == 0 {
		cfg.ResetInterval = 60 * time.Second
	}

	return Client{
		resolver:       resolver,
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
			ids, err := t.resolver.Queries.ResetTimeoutJobs(context.Background(), executorName)
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
	ticker := time.NewTicker(t.FlushInterval)

	for {
		shouldFlush := false

		select {
		case <-ctx.Done():
			// Force flush after stop fetching
			shouldFlush = true
		case <-ticker.C:
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
			if len(commitBatch)+len(failBatch)+len(retryBatch) > 100 {
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
			_, err := t.resolver.Mutation().CommitJobs(ctx, executorName, commitBatch)
			if err != nil {
				log.Println(err)
			}
			commitBatch = []model.CommitArgs{}
		}
		if len(failBatch) > 0 {
			_, err := t.resolver.Mutation().FailJobs(ctx, executorName, failBatch)
			if err != nil {
				log.Println(err)
			}
			failBatch = []model.CommitArgs{}
		}
		if len(retryBatch) > 0 {
			_, err := t.resolver.Mutation().RetryJobs(ctx, executorName, retryBatch)
			if err != nil {
				log.Println(err)
			}
			retryBatch = []model.CommitArgs{}
		}
	}
}

func (c *Client) Close() {
	c.resolver.DB.Close()
}

func (c *Client) Fetch(ctx context.Context, executorName string) (chan Job, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)
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
				jobs, err := c.resolver.
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
					ch <- Job{
						job,
						c.processedCh,
					}
				}
			}
		}
	}()

	return ch, cancel
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

func (c *Client) CreateJob(ctx context.Context, executorName string, args model.CreateJobArgs) (sqlc.TinyJob, error) {
	return c.resolver.Mutation().CreateJob(
		ctx,
		executorName,
		args,
	)
}

func (c *Client) UpdateJobByName(ctx context.Context, executorName, name string, args model.UpdateJobArgs) (sqlc.TinyJob, error) {
	return c.resolver.Mutation().UpdateJobByName(
		ctx,
		executorName,
		name,
		args,
	)
}

func (c *Client) UpdateJobByID(ctx context.Context, executorName string, id int64, args model.UpdateJobArgs) (sqlc.TinyJob, error) {
	return c.resolver.Mutation().UpdateJobByID(
		ctx,
		executorName,
		id,
		args,
	)
}

func (c *Client) DeleteJobByName(ctx context.Context, executorName, name string) (sqlc.TinyJob, error) {
	return c.resolver.Mutation().DeleteJobByName(
		ctx,
		executorName,
		name,
	)
}

func (c *Client) DeleteJobByID(ctx context.Context, executorName string, id int64) (sqlc.TinyJob, error) {
	return c.resolver.Mutation().DeleteJobByID(
		ctx,
		executorName,
		id,
	)
}

func (c *Client) SearchJobs(ctx context.Context, executorName string, args model.QueryJobsArgs) ([]sqlc.TinyJob, error) {
	return c.resolver.Query().SearchJobs(
		ctx,
		executorName,
		args,
	)
}

func (c *Client) QueryJobByName(ctx context.Context, executorName, name string) (sqlc.TinyJob, error) {
	return c.resolver.Query().QueryJobByName(
		ctx,
		executorName,
		name,
	)
}

func (c *Client) QueryJobByID(ctx context.Context, executorName string, id int64) (sqlc.TinyJob, error) {
	return c.resolver.Query().QueryJobByID(
		ctx,
		executorName,
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
