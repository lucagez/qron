-- name: GetJobByName :one
select * from tiny.job
where name = $1 limit 1;

-- name: GetJobByID :one
select * from tiny.job
where id = $1 limit 1;

-- TODO: Implement search

-- name: UpdateJobByName :one
update tiny.job
set run_at = coalesce(nullif(sqlc.arg('run_at'), ''), run_at),
    state = coalesce(nullif(sqlc.arg('state'), ''), state),
    config = coalesce(
        nullif(sqlc.arg('config'), '{}') || nullif(sqlc.arg('config'), ''),
        config
    )
where name = $1
returning *;

-- TODO: Should refactor usage of `name`

-- name: UpdateJobByID :one
update tiny.job
set run_at = coalesce(nullif(sqlc.arg('run_at'), ''), run_at),
    state = coalesce(nullif(sqlc.arg('state'), ''), state),
    config = config::json || sqlc.arg('config')::json
--     config = coalesce(
--         nullif(sqlc.arg('config'), '{}'),
--         config
--     )
where id = $1
returning *;

-- name: DeleteJobByName :one
delete from tiny.job
where name = $1
returning *;

-- name: DeleteJobByID :one
delete from tiny.job
where id = $1
returning *;

-- name: CreateHttpJob :one
insert into tiny.job(run_at, name, state, config, status, executor)
values (
   $1,
   $2,
   $3,
   $4,
   'READY',
   'HTTP'
)
returning *;

-- TODO: This query is not working wit dynamic params 🤔
-- name: SearchJobs :many
select * from tiny.job
where name like concat(sqlc.arg('query')::text, '%')
or name like concat('%', sqlc.arg('query')::text)
offset $1
limit $2;

-- name: BatchUpdateJobs :batchexec
update tiny.job
set last_run_at = $1,
    -- TODO: update
    state = $2,
    status = $3
where id = $4;

-- TODO: Add check for not running ever a job if
-- `last_run_at` happened less than x seconds ago
-- name: FetchDueJobs :many
update tiny.job as updated_jobs
set status      = 'PENDING',
    last_run_at = now()
from (
    select *
    from tiny.job
    where tiny.is_due(run_at, coalesce(last_run_at, created_at), now())
    and status = 'READY'
    -- worker limit
    limit $1 for update
    skip locked
) as due_jobs
where due_jobs.id = updated_jobs.id
returning updated_jobs.*;

-- with due_jobs as (
--     select *
--     from tiny.job as updated_jobs
--     where tiny.is_due(run_at, coalesce(last_run_at, created_at), now())
--       and status = 'READY'
--       -- worker limit
--     limit $1 for update
--         skip locked
-- )
-- update tiny.job as updated_jobs
-- set status      = 'PENDING',
--     last_run_at = now()
-- from due_jobs
-- where due_jobs.id = tiny.job.id
-- returning updated_jobs.*;
