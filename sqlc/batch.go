// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.16.0
// source: batch.go

package sqlc

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const batchCreateJobs = `-- name: BatchCreateJobs :batchexec
insert into tiny.job(expr, name, state, status, executor, run_at, timeout, start_at, meta, owner, retries)
values (
  $1,
  coalesce(nullif($2, ''), substr(md5(random()::text), 0, 25)),
  $3,
  'READY',
  $4,
  tiny.next(greatest($5, now()), $1),
  coalesce(nullif($6, 0), 120),
  $5,
  $7,
  coalesce(nullif($8, ''), 'default'),
  coalesce(nullif($9, 0), 5)
)
`

type BatchCreateJobsBatchResults struct {
	br     pgx.BatchResults
	tot    int
	closed bool
}

type BatchCreateJobsParams struct {
	Expr     string             `json:"expr"`
	Name     interface{}        `json:"name"`
	State    string             `json:"state"`
	Executor string             `json:"executor"`
	StartAt  pgtype.Timestamptz `json:"start_at"`
	Timeout  interface{}        `json:"timeout"`
	Meta     []byte             `json:"meta"`
	Owner    interface{}        `json:"owner"`
	Retries  interface{}        `json:"retries"`
}

func (q *Queries) BatchCreateJobs(ctx context.Context, arg []BatchCreateJobsParams) *BatchCreateJobsBatchResults {
	batch := &pgx.Batch{}
	for _, a := range arg {
		vals := []interface{}{
			a.Expr,
			a.Name,
			a.State,
			a.Executor,
			a.StartAt,
			a.Timeout,
			a.Meta,
			a.Owner,
			a.Retries,
		}
		batch.Queue(batchCreateJobs, vals...)
	}
	br := q.db.SendBatch(ctx, batch)
	return &BatchCreateJobsBatchResults{br, len(arg), false}
}

func (b *BatchCreateJobsBatchResults) Exec(f func(int, error)) {
	defer b.br.Close()
	for t := 0; t < b.tot; t++ {
		if b.closed {
			if f != nil {
				f(t, errors.New("batch already closed"))
			}
			continue
		}
		_, err := b.br.Exec()
		if f != nil {
			f(t, err)
		}
	}
}

func (b *BatchCreateJobsBatchResults) Close() error {
	b.closed = true
	return b.br.Close()
}

const batchUpdateFailedJobs = `-- name: BatchUpdateFailedJobs :batchexec
update tiny.job
set last_run_at = now(),
  state = coalesce(nullif($1::text, ''), state),
  expr = coalesce(nullif($2::text, ''), expr),
  status = case 
    when tiny.is_one_shot(expr) and retries - 1 <= 0 then 'FAILURE'::tiny.status
    else 'READY'::tiny.status
  end,
  retries = retries - 1,
  execution_amount = execution_amount + 1,
  run_at = case
    when tiny.is_one_shot(expr) then now() + concat(power(2, execution_amount)::text, 's')::interval
    else tiny.next(
      now(),
      coalesce(nullif($2::text, ''), expr)
    )
  end
where id = $3
and executor = $4
`

type BatchUpdateFailedJobsBatchResults struct {
	br     pgx.BatchResults
	tot    int
	closed bool
}

type BatchUpdateFailedJobsParams struct {
	State    string `json:"state"`
	Expr     string `json:"expr"`
	ID       int64  `json:"id"`
	Executor string `json:"executor"`
}

func (q *Queries) BatchUpdateFailedJobs(ctx context.Context, arg []BatchUpdateFailedJobsParams) *BatchUpdateFailedJobsBatchResults {
	batch := &pgx.Batch{}
	for _, a := range arg {
		vals := []interface{}{
			a.State,
			a.Expr,
			a.ID,
			a.Executor,
		}
		batch.Queue(batchUpdateFailedJobs, vals...)
	}
	br := q.db.SendBatch(ctx, batch)
	return &BatchUpdateFailedJobsBatchResults{br, len(arg), false}
}

func (b *BatchUpdateFailedJobsBatchResults) Exec(f func(int, error)) {
	defer b.br.Close()
	for t := 0; t < b.tot; t++ {
		if b.closed {
			if f != nil {
				f(t, errors.New("batch already closed"))
			}
			continue
		}
		_, err := b.br.Exec()
		if f != nil {
			f(t, err)
		}
	}
}

func (b *BatchUpdateFailedJobsBatchResults) Close() error {
	b.closed = true
	return b.br.Close()
}

const batchUpdateJobs = `-- name: BatchUpdateJobs :batchexec
update tiny.job
set last_run_at = now(),
  state = coalesce(nullif($1::text, ''), state),
  expr = coalesce(nullif($2::text, ''), expr),
  status = $3,
  execution_amount = execution_amount + 1,
  retries = $4,
  run_at = tiny.next(
    now(),
    coalesce(nullif($2::text, ''), expr)
  )
where id = $5
and executor = $6
`

type BatchUpdateJobsBatchResults struct {
	br     pgx.BatchResults
	tot    int
	closed bool
}

type BatchUpdateJobsParams struct {
	State    string     `json:"state"`
	Expr     string     `json:"expr"`
	Status   TinyStatus `json:"status"`
	Retries  int32      `json:"retries"`
	ID       int64      `json:"id"`
	Executor string     `json:"executor"`
}

func (q *Queries) BatchUpdateJobs(ctx context.Context, arg []BatchUpdateJobsParams) *BatchUpdateJobsBatchResults {
	batch := &pgx.Batch{}
	for _, a := range arg {
		vals := []interface{}{
			a.State,
			a.Expr,
			a.Status,
			a.Retries,
			a.ID,
			a.Executor,
		}
		batch.Queue(batchUpdateJobs, vals...)
	}
	br := q.db.SendBatch(ctx, batch)
	return &BatchUpdateJobsBatchResults{br, len(arg), false}
}

func (b *BatchUpdateJobsBatchResults) Exec(f func(int, error)) {
	defer b.br.Close()
	for t := 0; t < b.tot; t++ {
		if b.closed {
			if f != nil {
				f(t, errors.New("batch already closed"))
			}
			continue
		}
		_, err := b.br.Exec()
		if f != nil {
			f(t, err)
		}
	}
}

func (b *BatchUpdateJobsBatchResults) Close() error {
	b.closed = true
	return b.br.Close()
}
