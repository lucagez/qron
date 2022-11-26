// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.16.0
// source: queries.sql

package sqlc

import (
	"context"
	"database/sql"
	"time"
)

const createJob = `-- name: CreateJob :one
insert into tiny.job(run_at, name, state, status, executor)
values (
  $1,
  $2,
  $3,
  'READY',
  $4
)
returning id, run_at, name, last_run_at, created_at, execution_amount, timeout, status, state, executor
`

type CreateJobParams struct {
	RunAt    string
	Name     sql.NullString
	State    sql.NullString
	Executor string
}

func (q *Queries) CreateJob(ctx context.Context, arg CreateJobParams) (TinyJob, error) {
	row := q.db.QueryRow(ctx, createJob,
		arg.RunAt,
		arg.Name,
		arg.State,
		arg.Executor,
	)
	var i TinyJob
	err := row.Scan(
		&i.ID,
		&i.RunAt,
		&i.Name,
		&i.LastRunAt,
		&i.CreatedAt,
		&i.ExecutionAmount,
		&i.Timeout,
		&i.Status,
		&i.State,
		&i.Executor,
	)
	return i, err
}

const deleteJobByID = `-- name: DeleteJobByID :one
delete from tiny.job
where id = $1
and executor = $2 
returning id, run_at, name, last_run_at, created_at, execution_amount, timeout, status, state, executor
`

type DeleteJobByIDParams struct {
	ID       int64
	Executor string
}

func (q *Queries) DeleteJobByID(ctx context.Context, arg DeleteJobByIDParams) (TinyJob, error) {
	row := q.db.QueryRow(ctx, deleteJobByID, arg.ID, arg.Executor)
	var i TinyJob
	err := row.Scan(
		&i.ID,
		&i.RunAt,
		&i.Name,
		&i.LastRunAt,
		&i.CreatedAt,
		&i.ExecutionAmount,
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
returning id, run_at, name, last_run_at, created_at, execution_amount, timeout, status, state, executor
`

type DeleteJobByNameParams struct {
	Name     sql.NullString
	Executor string
}

func (q *Queries) DeleteJobByName(ctx context.Context, arg DeleteJobByNameParams) (TinyJob, error) {
	row := q.db.QueryRow(ctx, deleteJobByName, arg.Name, arg.Executor)
	var i TinyJob
	err := row.Scan(
		&i.ID,
		&i.RunAt,
		&i.Name,
		&i.LastRunAt,
		&i.CreatedAt,
		&i.ExecutionAmount,
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
  select id, run_at, name, last_run_at, created_at, execution_amount, timeout, status, state, executor
  from tiny.job j
  where tiny.is_due(j.run_at, coalesce(j.last_run_at, j.created_at), now())
  and j.status = 'READY'
  and j.executor = $2 
  -- worker limit
  limit $1 for update
  skip locked
) as due_jobs
where due_jobs.id = updated_jobs.id
returning updated_jobs.id, updated_jobs.run_at, updated_jobs.name, updated_jobs.last_run_at, updated_jobs.created_at, updated_jobs.execution_amount, updated_jobs.timeout, updated_jobs.status, updated_jobs.state, updated_jobs.executor
`

type FetchDueJobsParams struct {
	Limit    int32
	Executor string
}

// TODO: Add check for not running ever a job if
// `last_run_at` happened less than x seconds ago
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
			&i.RunAt,
			&i.Name,
			&i.LastRunAt,
			&i.CreatedAt,
			&i.ExecutionAmount,
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
select id, run_at, name, last_run_at, created_at, execution_amount, timeout, status, state, executor from tiny.job
where id = $1
and executor = $2 
limit 1
`

type GetJobByIDParams struct {
	ID       int64
	Executor string
}

func (q *Queries) GetJobByID(ctx context.Context, arg GetJobByIDParams) (TinyJob, error) {
	row := q.db.QueryRow(ctx, getJobByID, arg.ID, arg.Executor)
	var i TinyJob
	err := row.Scan(
		&i.ID,
		&i.RunAt,
		&i.Name,
		&i.LastRunAt,
		&i.CreatedAt,
		&i.ExecutionAmount,
		&i.Timeout,
		&i.Status,
		&i.State,
		&i.Executor,
	)
	return i, err
}

const getJobByName = `-- name: GetJobByName :one
select id, run_at, name, last_run_at, created_at, execution_amount, timeout, status, state, executor from tiny.job
where name = $1 
and executor = $2
limit 1
`

type GetJobByNameParams struct {
	Name     sql.NullString
	Executor string
}

func (q *Queries) GetJobByName(ctx context.Context, arg GetJobByNameParams) (TinyJob, error) {
	row := q.db.QueryRow(ctx, getJobByName, arg.Name, arg.Executor)
	var i TinyJob
	err := row.Scan(
		&i.ID,
		&i.RunAt,
		&i.Name,
		&i.LastRunAt,
		&i.CreatedAt,
		&i.ExecutionAmount,
		&i.Timeout,
		&i.Status,
		&i.State,
		&i.Executor,
	)
	return i, err
}

const nextRuns = `-- name: NextRuns :one
select date_part('year', runs) as year,
  date_part('month', runs) as month,
  date_part('day', runs) as day,
  date_part('minute', runs) as min,
  date_part('dow', runs) as dow 
from tiny.cron_next_run(
  $1::timestamptz,
  0,
  0, 
  $2::text
) as runs
`

type NextRunsParams struct {
	From time.Time
	Expr string
}

type NextRunsRow struct {
	Year  float64
	Month float64
	Day   float64
	Min   float64
	Dow   float64
}

func (q *Queries) NextRuns(ctx context.Context, arg NextRunsParams) (NextRunsRow, error) {
	row := q.db.QueryRow(ctx, nextRuns, arg.From, arg.Expr)
	var i NextRunsRow
	err := row.Scan(
		&i.Year,
		&i.Month,
		&i.Day,
		&i.Min,
		&i.Dow,
	)
	return i, err
}

const searchJobs = `-- name: SearchJobs :many
select id, run_at, name, last_run_at, created_at, execution_amount, timeout, status, state, executor from tiny.job
where (name like concat($4::text, '%')
  or name like concat('%', $4::text))
and executor = $3 
offset $1
limit $2
`

type SearchJobsParams struct {
	Offset   int32
	Limit    int32
	Executor string
	Query    string
}

// TODO: This query is not working wit dynamic params 🤔
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
			&i.RunAt,
			&i.Name,
			&i.LastRunAt,
			&i.CreatedAt,
			&i.ExecutionAmount,
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
set run_at = coalesce(nullif($3, ''), run_at),
  state = coalesce(nullif($4, ''), state)
where id = $1
and executor = $2 
returning id, run_at, name, last_run_at, created_at, execution_amount, timeout, status, state, executor
`

type UpdateJobByIDParams struct {
	ID       int64
	Executor string
	RunAt    interface{}
	State    interface{}
}

// TODO: Should refactor usage of `name`
func (q *Queries) UpdateJobByID(ctx context.Context, arg UpdateJobByIDParams) (TinyJob, error) {
	row := q.db.QueryRow(ctx, updateJobByID,
		arg.ID,
		arg.Executor,
		arg.RunAt,
		arg.State,
	)
	var i TinyJob
	err := row.Scan(
		&i.ID,
		&i.RunAt,
		&i.Name,
		&i.LastRunAt,
		&i.CreatedAt,
		&i.ExecutionAmount,
		&i.Timeout,
		&i.Status,
		&i.State,
		&i.Executor,
	)
	return i, err
}

const updateJobByName = `-- name: UpdateJobByName :one

update tiny.job
set run_at = coalesce(nullif($3, ''), run_at),
  state = coalesce(nullif($4, ''), state)
where name = $1
and executor = $2 
returning id, run_at, name, last_run_at, created_at, execution_amount, timeout, status, state, executor
`

type UpdateJobByNameParams struct {
	Name     sql.NullString
	Executor string
	RunAt    interface{}
	State    interface{}
}

// TODO: Implement search
func (q *Queries) UpdateJobByName(ctx context.Context, arg UpdateJobByNameParams) (TinyJob, error) {
	row := q.db.QueryRow(ctx, updateJobByName,
		arg.Name,
		arg.Executor,
		arg.RunAt,
		arg.State,
	)
	var i TinyJob
	err := row.Scan(
		&i.ID,
		&i.RunAt,
		&i.Name,
		&i.LastRunAt,
		&i.CreatedAt,
		&i.ExecutionAmount,
		&i.Timeout,
		&i.Status,
		&i.State,
		&i.Executor,
	)
	return i, err
}
