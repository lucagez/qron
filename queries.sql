-- name: GetJobByName :one
select * from tiny.job
where name = $1 limit 1;

-- name: GetJobByID :one
select * from tiny.job
where id = $1 limit 1;

-- TODO: Implement search

-- name: UpdateJobByName :one
update tiny.job
set run_at = coalesce(nullif($2, ''), run_at),
    state = coalesce(nullif($3, ''), state),
    config = coalesce(nullif($4, ''), config)
where name = $1
returning *;

-- TODO: Should refactor usage of `name`

-- name: UpdateJobByID :one
update tiny.job
set run_at = coalesce(nullif($2, ''), run_at),
    state = coalesce(nullif($3, ''), state),
    config = coalesce(nullif($4, ''), config)
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

-- name: SearchJobs :many
select * from tiny.job
where run_at like concat($1, '%')
or run_at like concat('%', $1)
offset $2
limit $3;
