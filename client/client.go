package client

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/lucagez/tinyq/executor"
	"github.com/lucagez/tinyq/graph"
	"github.com/lucagez/tinyq/graph/generated"
	"github.com/lucagez/tinyq/sqlc"
)

var processedCh chan Job

type Client struct {
	resolver       graph.Resolver
	MaxInFlight    uint64
	FlushInterval  time.Duration
	PollInterval   time.Duration
	ExecutorSetter func(http.Handler) http.Handler
}

type Config struct {
	Dsn            string
	MaxInFlight    uint64
	FlushInterval  time.Duration
	PollInterval   time.Duration
	ExecutorSetter func(http.Handler) http.Handler
}

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

	return Client{
		resolver:       resolver,
		MaxInFlight:    cfg.MaxInFlight,
		FlushInterval:  cfg.FlushInterval,
		PollInterval:   cfg.PollInterval,
		ExecutorSetter: cfg.ExecutorSetter,
	}, nil
}

func (t *Client) IncreaseInFlight() {
	atomic.AddUint64(&t.MaxInFlight, 1)
}

func (t *Client) DecreaseInFlight() {
	atomic.AddUint64(&t.MaxInFlight, ^uint64(0))
}

func (t *Client) flush(ctx context.Context, executorName string) {
	var commitBatch []string
	var failBatch []string
	var retryBatch []string
	ticker := time.NewTicker(t.FlushInterval)

	for {
		shouldFlush := false

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			shouldFlush = true
		case job := <-processedCh:
			switch job.Status {
			case sqlc.TinyStatusSUCCESS:
				commitBatch = append(commitBatch, strconv.FormatInt(job.ID, 10))
			case sqlc.TinyStatusFAILURE:
				failBatch = append(failBatch, strconv.FormatInt(job.ID, 10))
			case sqlc.TinyStatusREADY:
				retryBatch = append(retryBatch, strconv.FormatInt(job.ID, 10))
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
		}
		if len(failBatch) > 0 {
			_, err := t.resolver.Mutation().FailJobs(executor.NewCtx(ctx, executorName), failBatch)
			if err != nil {
				log.Println(err)
			}
		}
		if len(retryBatch) > 0 {
			_, err := t.resolver.Mutation().RetryJobs(executor.NewCtx(ctx, executorName), retryBatch)
			if err != nil {
				log.Println(err)
			}
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
					ch <- Job{job}
				}
			}
		}
	}()

	return ch, func() {
		cancel()
		close(ch)
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

type Job struct {
	sqlc.TinyJob
}

func (j Job) Commit() {
	j.Status = sqlc.TinyStatusSUCCESS
	processedCh <- j
}

func (j Job) Fail() {
	j.Status = sqlc.TinyStatusFAILURE
	processedCh <- j
}

func (j Job) Retry() {
	j.Status = sqlc.TinyStatusREADY
	processedCh <- j
}
