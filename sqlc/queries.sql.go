// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.15.0
// source: queries.sql

package sqlc

import (
	"context"
	"database/sql"
)

const createHttpJob = `-- name: CreateHttpJob :one
insert into tiny.job(run_at, name, state, config, status, executor)
values (
   $1,
   $2,
   $3,
   $4,
   'READY',
   'HTTP'
)
returning id, run_at, name, last_run_at, created_at, execution_amount, timeout, status, state, config, executor
`

type CreateHttpJobParams struct {
	RunAt  interface{}
	Name   sql.NullString
	State  sql.NullString
	Config sql.NullString
}

func (q *Queries) CreateHttpJob(ctx context.Context, arg CreateHttpJobParams) (TinyJob, error) {
	row := q.db.QueryRowContext(ctx, createHttpJob,
		arg.RunAt,
		arg.Name,
		arg.State,
		arg.Config,
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
		&i.Config,
		&i.Executor,
	)
	return i, err
}

const deleteJobByID = `-- name: DeleteJobByID :one
delete from tiny.job
where id = $1
returning id, run_at, name, last_run_at, created_at, execution_amount, timeout, status, state, config, executor
`

func (q *Queries) DeleteJobByID(ctx context.Context, id int64) (TinyJob, error) {
	row := q.db.QueryRowContext(ctx, deleteJobByID, id)
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
		&i.Config,
		&i.Executor,
	)
	return i, err
}

const deleteJobByName = `-- name: DeleteJobByName :one
delete from tiny.job
where name = $1
returning id, run_at, name, last_run_at, created_at, execution_amount, timeout, status, state, config, executor
`

func (q *Queries) DeleteJobByName(ctx context.Context, name sql.NullString) (TinyJob, error) {
	row := q.db.QueryRowContext(ctx, deleteJobByName, name)
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
		&i.Config,
		&i.Executor,
	)
	return i, err
}

const getJobByID = `-- name: GetJobByID :one
select id, run_at, name, last_run_at, created_at, execution_amount, timeout, status, state, config, executor from tiny.job
where id = $1 limit 1
`

func (q *Queries) GetJobByID(ctx context.Context, id int64) (TinyJob, error) {
	row := q.db.QueryRowContext(ctx, getJobByID, id)
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
		&i.Config,
		&i.Executor,
	)
	return i, err
}

const getJobByName = `-- name: GetJobByName :one
select id, run_at, name, last_run_at, created_at, execution_amount, timeout, status, state, config, executor from tiny.job
where name = $1 limit 1
`

func (q *Queries) GetJobByName(ctx context.Context, name sql.NullString) (TinyJob, error) {
	row := q.db.QueryRowContext(ctx, getJobByName, name)
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
		&i.Config,
		&i.Executor,
	)
	return i, err
}

const searchJobs = `-- name: SearchJobs :many
select id, run_at, name, last_run_at, created_at, execution_amount, timeout, status, state, config, executor from tiny.job
where run_at like concat($1, '%')
or run_at like concat('%', $1)
offset $2
limit $3
`

type SearchJobsParams struct {
	Concat interface{}
	Offset int32
	Limit  int32
}

func (q *Queries) SearchJobs(ctx context.Context, arg SearchJobsParams) ([]TinyJob, error) {
	rows, err := q.db.QueryContext(ctx, searchJobs, arg.Concat, arg.Offset, arg.Limit)
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
			&i.Config,
			&i.Executor,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const updateJobByID = `-- name: UpdateJobByID :one

update tiny.job
set run_at = coalesce(nullif($2, ''), run_at),
    state = coalesce(nullif($3, ''), state),
    config = coalesce(nullif($4, ''), config)
where id = $1
returning id, run_at, name, last_run_at, created_at, execution_amount, timeout, status, state, config, executor
`

type UpdateJobByIDParams struct {
	ID      int64
	Column2 interface{}
	Column3 interface{}
	Column4 interface{}
}

// TODO: Should refactor usage of `name`
func (q *Queries) UpdateJobByID(ctx context.Context, arg UpdateJobByIDParams) (TinyJob, error) {
	row := q.db.QueryRowContext(ctx, updateJobByID,
		arg.ID,
		arg.Column2,
		arg.Column3,
		arg.Column4,
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
		&i.Config,
		&i.Executor,
	)
	return i, err
}

const updateJobByName = `-- name: UpdateJobByName :one

update tiny.job
set run_at = coalesce(nullif($2, ''), run_at),
    state = coalesce(nullif($3, ''), state),
    config = coalesce(nullif($4, ''), config)
where name = $1
returning id, run_at, name, last_run_at, created_at, execution_amount, timeout, status, state, config, executor
`

type UpdateJobByNameParams struct {
	Name    sql.NullString
	Column2 interface{}
	Column3 interface{}
	Column4 interface{}
}

// TODO: Implement search
func (q *Queries) UpdateJobByName(ctx context.Context, arg UpdateJobByNameParams) (TinyJob, error) {
	row := q.db.QueryRowContext(ctx, updateJobByName,
		arg.Name,
		arg.Column2,
		arg.Column3,
		arg.Column4,
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
		&i.Config,
		&i.Executor,
	)
	return i, err
}
