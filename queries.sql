-- name: GetJobByName :one
select * from tiny.job
where name = $1 
and executor = $2
limit 1;

-- name: GetJobByID :one
select * from tiny.job
where id = $1
and executor = $2 
limit 1;

-- TODO: Implement search

-- name: UpdateJobByName :one
update tiny.job
set run_at = coalesce(nullif(sqlc.arg('run_at'), ''), run_at),
  state = coalesce(nullif(sqlc.arg('state'), ''), state)
where name = $1
and executor = $2 
returning *;

-- TODO: Should refactor usage of `name`

-- name: UpdateJobByID :one
update tiny.job
set run_at = coalesce(nullif(sqlc.arg('run_at'), ''), run_at),
  state = coalesce(nullif(sqlc.arg('state'), ''), state)
where id = $1
and executor = $2 
returning *;

-- name: DeleteJobByName :one
delete from tiny.job
where name = $1
and executor = $2 
returning *;

-- name: DeleteJobByID :one
delete from tiny.job
where id = $1
and executor = $2 
returning *;

-- name: CreateJob :one
insert into tiny.job(run_at, name, state, status, executor)
values (
  $1,
  $2,
  $3,
  'READY',
  $4
)
returning *;

-- TODO: This query is not working wit dynamic params 🤔
-- name: SearchJobs :many
select * from tiny.job
where (name like concat(sqlc.arg('query')::text, '%')
  or name like concat('%', sqlc.arg('query')::text))
and executor = $3 
offset $1
limit $2;

-- name: BatchUpdateJobs :batchexec
update tiny.job
set last_run_at = $1,
  -- TODO: update
  state = $2,
  status = $3
where id = $4
and executor = $5; 

-- TODO: Add check for not running ever a job if
-- `last_run_at` happened less than x seconds ago
-- name: FetchDueJobs :many
update tiny.job as updated_jobs
set status = 'PENDING',
  last_run_at = now()
from (
  select *
  from tiny.job j
  where tiny.is_due(j.run_at, coalesce(j.last_run_at, j.created_at), now())
  and j.status = 'READY'
  and j.executor = sqlc.arg('executor') 
  -- worker limit
  limit $1 for update
  skip locked
) as due_jobs
where due_jobs.id = updated_jobs.id
returning updated_jobs.*;
