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
set expr = coalesce(nullif(sqlc.arg('expr'), ''), expr),
  state = coalesce(nullif(sqlc.arg('state'), ''), state),
  timeout = coalesce(nullif(sqlc.arg('timeout'), 0), timeout),
  -- `run_at` should always be consistent
  run_at = tiny.next(
    coalesce(last_run_at, created_at), 
    coalesce(nullif(sqlc.arg('expr'), ''), expr)
  )
where name = $1
and executor = $2 
returning *;

-- TODO: Should refactor usage of `name`

-- name: UpdateJobByID :one
update tiny.job
set expr = coalesce(nullif(sqlc.arg('expr'), ''), expr),
  state = coalesce(nullif(sqlc.arg('state'), ''), state),
  timeout = coalesce(nullif(sqlc.arg('timeout'), 0), timeout),
  -- `run_at` should always be consistent
  run_at = tiny.next(
    coalesce(last_run_at, created_at), 
    coalesce(nullif(sqlc.arg('expr'), ''), expr)
  )
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
insert into tiny.job(expr, name, state, status, executor, run_at, timeout, start_at)
values (
  $1,
  $2,
  $3,
  'READY',
  $4,
  tiny.next($6, $1),
  $5,
  $6
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
  status = $3,
  execution_amount = execution_amount + 1,
  run_at = tiny.next($1, expr)
where id = $4
and executor = $5; 

-- name: FetchDueJobs :many
update tiny.job as updated_jobs
set status = 'PENDING',
  last_run_at = now()
from (
  select id
  from tiny.job j
  where j.run_at < now()
  and j.status = 'READY'
  and j.executor = sqlc.arg('executor') 
  -- worker limit
  limit $1 for update
  skip locked
) as due_jobs
where due_jobs.id = updated_jobs.id
returning updated_jobs.*;

-- name: ResetTimeoutJobs :many
update tiny.job
set status = 'READY'
where timeout is not null
and timeout > 0
and now() - last_run_at > make_interval(secs => timeout)
and executor = $1
returning id;

-- name: NextRuns :one
select date_part('year', runs) as year,
  date_part('month', runs) as month,
  date_part('day', runs) as day,
  date_part('minute', runs) as min,
  date_part('dow', runs) as dow 
from tiny.cron_next_run(
  sqlc.arg('from')::timestamptz,
  sqlc.arg('expr')::text
) as runs;

-- name: Next :one
select run_at::timestamptz
from tiny.next(
  sqlc.arg('from')::timestamptz,
  sqlc.arg('expr')::text
) as run_at;

-- name: CountJobsInStatus :one
select count(*) from tiny.job
where executor = $1
and status = $2;