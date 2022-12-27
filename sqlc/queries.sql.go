// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.16.0
// source: queries.sql

package sqlc

import (
	"context"
	"time"

	"github.com/jackc/pgtype"
)

const countJobsInStatus = `-- name: CountJobsInStatus :one
select count(*) from tiny.job
where executor = $1
and status = $2
`

type CountJobsInStatusParams struct {
	Executor string     `json:"executor"`
	Status   TinyStatus `json:"status"`
}

func (q *Queries) CountJobsInStatus(ctx context.Context, arg CountJobsInStatusParams) (int64, error) {
	row := q.db.QueryRow(ctx, countJobsInStatus, arg.Executor, arg.Status)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const createJob = `-- name: CreateJob :one
insert into tiny.job(expr, name, state, status, executor, run_at, timeout, start_at, meta)
values (
  $1,
  coalesce(nullif($6, ''), substr(md5(random()::text), 0, 25)),
  $2,
  'READY',
  $3,
  tiny.next(greatest($4, now()), $1),
  coalesce(nullif($7, 0), 120),
  $4,
  $5
)
returning id, expr, run_at, last_run_at, created_at, start_at, execution_amount, name, meta, timeout, status, state, executor
`

type CreateJobParams struct {
	Expr     string      `json:"expr"`
	State    string      `json:"state"`
	Executor string      `json:"executor"`
	StartAt  time.Time   `json:"start_at"`
	Meta     pgtype.JSON `json:"meta"`
	Name     interface{} `json:"name"`
	Timeout  interface{} `json:"timeout"`
}

func (q *Queries) CreateJob(ctx context.Context, arg CreateJobParams) (TinyJob, error) {
	row := q.db.QueryRow(ctx, createJob,
		arg.Expr,
		arg.State,
		arg.Executor,
		arg.StartAt,
		arg.Meta,
		arg.Name,
		arg.Timeout,
	)
	var i TinyJob
	err := row.Scan(
		&i.ID,
		&i.Expr,
		&i.RunAt,
		&i.LastRunAt,
		&i.CreatedAt,
		&i.StartAt,
		&i.ExecutionAmount,
		&i.Name,
		&i.Meta,
		&i.Timeout,
		&i.Status,
		&i.State,
		&i.Executor,
	)
	return i, err
}

const cronNextRun = `-- name: CronNextRun :one
select run_at::timestamptz 
from tiny.cron_next_run(
  $1::timestamptz,
  $2::text
) as run_at
`

type CronNextRunParams struct {
	From time.Time `json:"from"`
	Expr string    `json:"expr"`
}

func (q *Queries) CronNextRun(ctx context.Context, arg CronNextRunParams) (time.Time, error) {
	row := q.db.QueryRow(ctx, cronNextRun, arg.From, arg.Expr)
	var run_at time.Time
	err := row.Scan(&run_at)
	return run_at, err
}

const deleteJobByID = `-- name: DeleteJobByID :one
delete from tiny.job
where id = $1
and executor = $2 
returning id, expr, run_at, last_run_at, created_at, start_at, execution_amount, name, meta, timeout, status, state, executor
`

type DeleteJobByIDParams struct {
	ID       int64  `json:"id"`
	Executor string `json:"executor"`
}

func (q *Queries) DeleteJobByID(ctx context.Context, arg DeleteJobByIDParams) (TinyJob, error) {
	row := q.db.QueryRow(ctx, deleteJobByID, arg.ID, arg.Executor)
	var i TinyJob
	err := row.Scan(
		&i.ID,
		&i.Expr,
		&i.RunAt,
		&i.LastRunAt,
		&i.CreatedAt,
		&i.StartAt,
		&i.ExecutionAmount,
		&i.Name,
		&i.Meta,
		&i.Timeout,
		&i.Status,
		&i.State,
		&i.Executor,
	)
	return i, err
}

const deleteJobByName = `-- name: DeleteJobByName :one
delete from tiny.job
where name = $1
and executor = $2 
returning id, expr, run_at, last_run_at, created_at, start_at, execution_amount, name, meta, timeout, status, state, executor
`

type DeleteJobByNameParams struct {
	Name     string `json:"name"`
	Executor string `json:"executor"`
}

func (q *Queries) DeleteJobByName(ctx context.Context, arg DeleteJobByNameParams) (TinyJob, error) {
	row := q.db.QueryRow(ctx, deleteJobByName, arg.Name, arg.Executor)
	var i TinyJob
	err := row.Scan(
		&i.ID,
		&i.Expr,
		&i.RunAt,
		&i.LastRunAt,
		&i.CreatedAt,
		&i.StartAt,
		&i.ExecutionAmount,
		&i.Name,
		&i.Meta,
		&i.Timeout,
		&i.Status,
		&i.State,
		&i.Executor,
	)
	return i, err
}

const fetchDueJobs = `-- name: FetchDueJobs :many
update tiny.job as updated_jobs
set status = 'PENDING',
  last_run_at = now()
from (
  select id
  from tiny.job j
  where j.run_at < now()
  and j.status = 'READY'
  and j.executor = $2 
  -- worker limit
  limit $1 for update
  skip locked
) as due_jobs
where due_jobs.id = updated_jobs.id
returning updated_jobs.id, updated_jobs.expr, updated_jobs.run_at, updated_jobs.last_run_at, updated_jobs.created_at, updated_jobs.start_at, updated_jobs.execution_amount, updated_jobs.name, updated_jobs.meta, updated_jobs.timeout, updated_jobs.status, updated_jobs.state, updated_jobs.executor
`

type FetchDueJobsParams struct {
	Limit    int32  `json:"limit"`
	Executor string `json:"executor"`
}

func (q *Queries) FetchDueJobs(ctx context.Context, arg FetchDueJobsParams) ([]TinyJob, error) {
	rows, err := q.db.Query(ctx, fetchDueJobs, arg.Limit, arg.Executor)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []TinyJob
	for rows.Next() {
		var i TinyJob
		if err := rows.Scan(
			&i.ID,
			&i.Expr,
			&i.RunAt,
			&i.LastRunAt,
			&i.CreatedAt,
			&i.StartAt,
			&i.ExecutionAmount,
			&i.Name,
			&i.Meta,
			&i.Timeout,
			&i.Status,
			&i.State,
			&i.Executor,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getJobByID = `-- name: GetJobByID :one
select id, expr, run_at, last_run_at, created_at, start_at, execution_amount, name, meta, timeout, status, state, executor from tiny.job
where id = $1
and executor = $2 
limit 1
`

type GetJobByIDParams struct {
	ID       int64  `json:"id"`
	Executor string `json:"executor"`
}

func (q *Queries) GetJobByID(ctx context.Context, arg GetJobByIDParams) (TinyJob, error) {
	row := q.db.QueryRow(ctx, getJobByID, arg.ID, arg.Executor)
	var i TinyJob
	err := row.Scan(
		&i.ID,
		&i.Expr,
		&i.RunAt,
		&i.LastRunAt,
		&i.CreatedAt,
		&i.StartAt,
		&i.ExecutionAmount,
		&i.Name,
		&i.Meta,
		&i.Timeout,
		&i.Status,
		&i.State,
		&i.Executor,
	)
	return i, err
}

const getJobByName = `-- name: GetJobByName :one
select id, expr, run_at, last_run_at, created_at, start_at, execution_amount, name, meta, timeout, status, state, executor from tiny.job
where name = $1 
and executor = $2
limit 1
`

type GetJobByNameParams struct {
	Name     string `json:"name"`
	Executor string `json:"executor"`
}

func (q *Queries) GetJobByName(ctx context.Context, arg GetJobByNameParams) (TinyJob, error) {
	row := q.db.QueryRow(ctx, getJobByName, arg.Name, arg.Executor)
	var i TinyJob
	err := row.Scan(
		&i.ID,
		&i.Expr,
		&i.RunAt,
		&i.LastRunAt,
		&i.CreatedAt,
		&i.StartAt,
		&i.ExecutionAmount,
		&i.Name,
		&i.Meta,
		&i.Timeout,
		&i.Status,
		&i.State,
		&i.Executor,
	)
	return i, err
}

const next = `-- name: Next :one
select run_at::timestamptz
from tiny.next(
  $1::timestamptz,
  $2::text
) as run_at
`

type NextParams struct {
	From time.Time `json:"from"`
	Expr string    `json:"expr"`
}

func (q *Queries) Next(ctx context.Context, arg NextParams) (time.Time, error) {
	row := q.db.QueryRow(ctx, next, arg.From, arg.Expr)
	var run_at time.Time
	err := row.Scan(&run_at)
	return run_at, err
}

const resetTimeoutJobs = `-- name: ResetTimeoutJobs :many
update tiny.job
set status = 'READY'
where timeout is not null
and timeout > 0
and now() - last_run_at > make_interval(secs => timeout)
and executor = $1
and status = 'PENDING'
returning id
`

func (q *Queries) ResetTimeoutJobs(ctx context.Context, executor string) ([]int64, error) {
	rows, err := q.db.Query(ctx, resetTimeoutJobs, executor)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		items = append(items, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const searchJobs = `-- name: SearchJobs :many
select id, expr, run_at, last_run_at, created_at, start_at, execution_amount, name, meta, timeout, status, state, executor from tiny.job
where (name like concat($4::text, '%')
  or name like concat('%', $4::text))
and executor = $3 
offset $1
limit $2
`

type SearchJobsParams struct {
	Offset   int32  `json:"offset"`
	Limit    int32  `json:"limit"`
	Executor string `json:"executor"`
	Query    string `json:"query"`
}

// TODO: This query is not working with dynamic params 🤔
func (q *Queries) SearchJobs(ctx context.Context, arg SearchJobsParams) ([]TinyJob, error) {
	rows, err := q.db.Query(ctx, searchJobs,
		arg.Offset,
		arg.Limit,
		arg.Executor,
		arg.Query,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []TinyJob
	for rows.Next() {
		var i TinyJob
		if err := rows.Scan(
			&i.ID,
			&i.Expr,
			&i.RunAt,
			&i.LastRunAt,
			&i.CreatedAt,
			&i.StartAt,
			&i.ExecutionAmount,
			&i.Name,
			&i.Meta,
			&i.Timeout,
			&i.Status,
			&i.State,
			&i.Executor,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const updateJobByID = `-- name: UpdateJobByID :one

update tiny.job
set expr = coalesce(nullif($3, ''), expr),
  state = coalesce(nullif($4, ''), state),
  timeout = coalesce(nullif($5, 0), timeout),
  -- ` + "`" + `run_at` + "`" + ` should always be consistent
  run_at = tiny.next(
    coalesce(last_run_at, created_at), 
    coalesce(nullif($3, ''), expr)
  )
where id = $1
and executor = $2 
returning id, expr, run_at, last_run_at, created_at, start_at, execution_amount, name, meta, timeout, status, state, executor
`

type UpdateJobByIDParams struct {
	ID       int64       `json:"id"`
	Executor string      `json:"executor"`
	Expr     interface{} `json:"expr"`
	State    interface{} `json:"state"`
	Timeout  interface{} `json:"timeout"`
}

// TODO: Should refactor usage of `name`
func (q *Queries) UpdateJobByID(ctx context.Context, arg UpdateJobByIDParams) (TinyJob, error) {
	row := q.db.QueryRow(ctx, updateJobByID,
		arg.ID,
		arg.Executor,
		arg.Expr,
		arg.State,
		arg.Timeout,
	)
	var i TinyJob
	err := row.Scan(
		&i.ID,
		&i.Expr,
		&i.RunAt,
		&i.LastRunAt,
		&i.CreatedAt,
		&i.StartAt,
		&i.ExecutionAmount,
		&i.Name,
		&i.Meta,
		&i.Timeout,
		&i.Status,
		&i.State,
		&i.Executor,
	)
	return i, err
}

const updateJobByName = `-- name: UpdateJobByName :one

update tiny.job
set expr = coalesce(nullif($3, ''), expr),
  state = coalesce(nullif($4, ''), state),
  timeout = coalesce(nullif($5, 0), timeout),
  -- ` + "`" + `run_at` + "`" + ` should always be consistent
  run_at = tiny.next(
    coalesce(last_run_at, created_at), 
    coalesce(nullif($3, ''), expr)
  )
where name = $1
and executor = $2 
returning id, expr, run_at, last_run_at, created_at, start_at, execution_amount, name, meta, timeout, status, state, executor
`

type UpdateJobByNameParams struct {
	Name     string      `json:"name"`
	Executor string      `json:"executor"`
	Expr     interface{} `json:"expr"`
	State    interface{} `json:"state"`
	Timeout  interface{} `json:"timeout"`
}

// TODO: Implement search
func (q *Queries) UpdateJobByName(ctx context.Context, arg UpdateJobByNameParams) (TinyJob, error) {
	row := q.db.QueryRow(ctx, updateJobByName,
		arg.Name,
		arg.Executor,
		arg.Expr,
		arg.State,
		arg.Timeout,
	)
	var i TinyJob
	err := row.Scan(
		&i.ID,
		&i.Expr,
		&i.RunAt,
		&i.LastRunAt,
		&i.CreatedAt,
		&i.StartAt,
		&i.ExecutionAmount,
		&i.Name,
		&i.Meta,
		&i.Timeout,
		&i.Status,
		&i.State,
		&i.Executor,
	)
	return i, err
}
