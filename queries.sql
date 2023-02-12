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

-- name: StopJob :one
update tiny.job
set status = 'PAUSED'
where id = $1
and executor = $2
-- Cannot stop a currently running task as it is outside of control for now
-- Possible to add a notification system to listen on those kind of events
and status not in ('FAILURE', 'SUCCESS', 'PENDING')
returning *;

-- name: RestartJob :one
update tiny.job
set status = 'READY'
where id = $1
and executor = $2
and status = 'PAUSED'
returning *;

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

-- name: UpdateStateByID :one
update tiny.job
set state = coalesce(nullif(sqlc.arg('state'), ''), state)
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
insert into tiny.job(expr, name, state, status, executor, run_at, timeout, start_at, meta, owner, retries)
values (
  $1,
  coalesce(nullif(sqlc.arg('name'), ''), substr(md5(random()::text), 0, 25)),
  $2,
  'READY',
  $3,
  tiny.next(greatest($4, now()), $1),
  coalesce(nullif(sqlc.arg('timeout'), 0), 120),
  $4,
  $5,
  coalesce(nullif(sqlc.arg('owner'), ''), 'default'),
  coalesce(nullif(sqlc.arg('retries'), 0), 5)
)
-- on conflict on constraint job_name_owner_key
-- do ...
returning *;

-- TODO: This query is not working with dynamic params 🤔
-- name: SearchJobs :many
select * from tiny.job
where (name like concat(sqlc.arg('query')::text, '%')
  or name like concat('%', sqlc.arg('query')::text))
and executor = $3 
offset $1
limit $2;

-- name: SearchJobsByMeta :many
with jobs as (
  select * from tiny.job
  where meta::jsonb @> (sqlc.arg('query')::text)::jsonb
  and status::text = any(string_to_array(sqlc.arg('statuses')::text, ','))
  and created_at > sqlc.arg('from')::timestamptz
  and created_at < sqlc.arg('to')::timestamptz
  and (name like concat(sqlc.arg('name')::text, '%')
    or name like concat('%', sqlc.arg('name')::text))
  -- Filter recurring tasks
  and tiny.is_one_shot(expr) = sqlc.arg('is_one_shot')::boolean
  and executor = sqlc.arg('executor')::text
),
total as (
  select count(*) as total_count from jobs
)
select jobs.*, total_count from jobs, total
order by last_run_at desc
limit sqlc.arg('limit')::int
offset sqlc.arg('offset')::int;

-- name: BatchUpdateJobs :batchexec
update tiny.job
set last_run_at = now(),
  -- TODO: update
  state = coalesce(nullif(sqlc.arg('state')::text, ''), state),
  expr = coalesce(nullif(sqlc.arg('expr')::text, ''), expr),
  status = sqlc.arg('status'),
  execution_amount = execution_amount + 1,
  retries = sqlc.arg('retries'),
  run_at = tiny.next(
    now(),
    coalesce(nullif(sqlc.arg('expr')::text, ''), expr)
  )
where id = sqlc.arg('id')
and executor = sqlc.arg('executor'); 

-- name: BatchUpdateFailedJobs :batchexec
update tiny.job
set last_run_at = now(),
  state = coalesce(nullif(sqlc.arg('state')::text, ''), state),
  expr = coalesce(nullif(sqlc.arg('expr')::text, ''), expr),
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
      coalesce(nullif(sqlc.arg('expr')::text, ''), expr)
    )
  end
where id = sqlc.arg('id')
and executor = sqlc.arg('executor'); 

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
and status = 'PENDING'
returning id;

-- name: CronNextRun :one
select run_at::timestamptz 
from tiny.cron_next_run(
  sqlc.arg('from')::timestamptz,
  sqlc.arg('expr')::text
) as run_at;

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
