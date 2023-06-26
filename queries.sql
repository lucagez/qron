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

-- name: LastUpdate :one
select max(updated_at)::timestamptz as last_update 
from tiny.job
where executor = $1;

-- name: UpdateJobByName :one
update tiny.job
set expr = coalesce(nullif(sqlc.arg('expr'), ''), expr),
  state = coalesce(nullif(sqlc.arg('state'), ''), state),
  timeout = coalesce(nullif(sqlc.arg('timeout'), 0), timeout),
  updated_at = now(),
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
set status = 'PAUSED',
  updated_at = now()
where id = $1
and executor = $2
-- Cannot stop a currently running task as it is outside of control for now
-- Possible to add a notification system to listen on those kind of events
and status not in ('FAILURE', 'SUCCESS', 'PENDING')
returning *;

-- name: RestartJob :one
update tiny.job
set status = 'READY',
  updated_at = now() 
where id = $1
and executor = $2
and status = 'PAUSED'
returning *;

-- name: UpdateJobByID :one
update tiny.job
set expr = coalesce(nullif(sqlc.arg('expr'), ''), expr),
  updated_at = now(),
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
set state = coalesce(nullif(sqlc.arg('state'), ''), state),
  updated_at = now()
where id = $1
and executor = $2 
returning *;

-- name: UpdateExprByID :one
update tiny.job
set expr = coalesce(nullif(sqlc.arg('expr'), ''), expr),
  updated_at = now()
where id = $1
and executor = $2 
returning *;

-- name: ValidateExprFormat :one
select (substr($1::text, 1, 6) in ('@every', '@after') and (substr($1::text, 7)::interval) is not null
    or $1::text ~ '^@(annually|yearly|monthly|weekly|daily|hourly|minutely)$'
    or substr($1::text, 1, 3) = '@at' and (substr($1::text, 4)::timestamptz) is not null
    or tiny.crontab($1::text))::bool as valid;

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
insert into tiny.job(expr, name, state, status, executor, run_at, timeout, start_at, meta, owner, retries, deduplication_key)
values (
  sqlc.arg('expr'),
  coalesce(nullif(sqlc.arg('name'), ''), substr(md5(random()::text), 0, 25)),
  sqlc.arg('state'),
  'READY',
  sqlc.arg('executor'),
  tiny.next(greatest(sqlc.arg('start_at'), now()), sqlc.arg('expr')),
  coalesce(nullif(sqlc.arg('timeout'), 0), 120),
  sqlc.arg('start_at'),
  sqlc.arg('meta'),
  coalesce(nullif(sqlc.arg('owner'), ''), 'default'),
  coalesce(nullif(sqlc.arg('retries'), 0), 5),
  sqlc.arg('deduplication_key')
)
-- on conflict on constraint job_name_owner_key
-- do ...
returning *;

-- name: BatchCreateJobs :batchexec
insert into tiny.job(expr, name, state, status, executor, run_at, timeout, start_at, meta, owner, retries, deduplication_key)
values (
  sqlc.arg('expr'),
  coalesce(nullif(sqlc.arg('name'), ''), substr(md5(random()::text), 0, 25)),
  sqlc.arg('state'),
  'READY',
  sqlc.arg('executor'),
  tiny.next(greatest(sqlc.arg('start_at'), now()), sqlc.arg('expr')),
  coalesce(nullif(sqlc.arg('timeout'), 0), 120),
  sqlc.arg('start_at'),
  sqlc.arg('meta'),
  coalesce(nullif(sqlc.arg('owner'), ''), 'default'),
  coalesce(nullif(sqlc.arg('retries'), 0), 5),
  sqlc.arg('deduplication_key')
);

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
  state = coalesce(nullif(sqlc.arg('state')::text, ''), state),
  expr = coalesce(nullif(sqlc.arg('expr')::text, ''), expr),
  status = sqlc.arg('status'),
  updated_at = now(),
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
  updated_at = now(),
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
with due_jobs as (
  select id
  from tiny.job j
  where j.run_at < now()
    and j.status = 'READY'
    and j.executor = sqlc.arg('executor')
  order by j.created_at
  limit $1
  for update skip locked
)
update tiny.job as updated_jobs
set status = 'PENDING',
  updated_at = now(),
  last_run_at = now()
from due_jobs
where due_jobs.id = updated_jobs.id
returning updated_jobs.*;

-- name: ResetTimeoutJobs :many
update tiny.job
set status = 'READY',
  updated_at = now()
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
