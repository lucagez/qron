package qron

import (
	"context"
	"encoding/json"
	"time"

	"github.com/lucagez/qron/graph/model"
	"github.com/lucagez/qron/sqlc"
)

type Scheduled[T any] struct {
	ExecutorName string
	State        T
	args         model.CreateJobArgs
	client       Client
}

func NewScheduled[T any](executorName string) Scheduled[T] {
	client, _ := NewClient(sugarDb, sugarCfg)
	return Scheduled[T]{
		ExecutorName: executorName,
		args:         model.CreateJobArgs{},
		client:       client,
	}
}

// Fork returns a new Scheduled[T] with the copy of the internal state.
// It is useful to create multiple jobs by sharing common configuration.
func (j Scheduled[T]) fork() Scheduled[T] {
	return Scheduled[T]{
		ExecutorName: j.ExecutorName,
		State:        j.State,
		client:       j.client,
		args: model.CreateJobArgs{
			Name:             j.args.Name,
			Expr:             j.args.Expr,
			Timeout:          j.args.Timeout,
			StartAt:          j.args.StartAt,
			Retries:          j.args.Retries,
			DeduplicationKey: j.args.DeduplicationKey,
		},
	}
}

func (j Scheduled[T]) DeduplicationKey(key string) Scheduled[T] {
	j.args.DeduplicationKey = &key
	return j.fork()
}

func (j Scheduled[T]) Name(name string) Scheduled[T] {
	j.args.Name = name
	return j.fork()
}

func (j Scheduled[T]) Expr(expr string) Scheduled[T] {
	j.args.Expr = expr
	return j.fork()
}

func (j Scheduled[T]) Timeout(timeout int) Scheduled[T] {
	j.args.Timeout = &timeout
	return j.fork()
}

func (j Scheduled[T]) StartAt(at time.Time) Scheduled[T] {
	j.args.StartAt = &at
	return j.fork()
}

func (j Scheduled[T]) Retries(retries int) Scheduled[T] {
	j.args.Retries = &retries
	return j.fork()
}

func (j Scheduled[T]) Schedule(ctx context.Context, state T) (sqlc.TinyJob, error) {
	// TODO: use bytea and encode/decode using gob
	buf, err := json.Marshal(state)
	if err != nil {
		return sqlc.TinyJob{}, err
	}

	return j.client.CreateJob(ctx, j.ExecutorName, model.CreateJobArgs{
		Name:             j.args.Name,
		Expr:             j.args.Expr,
		Timeout:          j.args.Timeout,
		StartAt:          j.args.StartAt,
		Retries:          j.args.Retries,
		State:            string(buf),
		DeduplicationKey: j.args.DeduplicationKey,
	})
}

type ScheduledJob[T any] struct {
	Job
	State T
}

func (j Scheduled[T]) Fetch(ctx context.Context) chan ScheduledJob[T] {
	ch := make(chan ScheduledJob[T])
	jobs := j.client.Fetch(ctx, j.ExecutorName)

	go func() {
		for job := range jobs {
			state := new(T)

			// TODO: use bytea and encode/decode using gob
			json.Unmarshal([]byte(job.State), state)
			ch <- ScheduledJob[T]{Job: job, State: *state}
		}

		close(ch)
	}()

	return ch
}
