-- +goose Up
-- +goose StatementBegin
do $$
begin
create role tinyrole;
grant all on schema public to tinyrole;
grant usage, select on sequence tiny.job_id_seq to tinyrole;
grant all on schema tiny to tinyrole;
grant all on tiny.job to tinyrole;

exception when duplicate_object then raise notice '%, skipping', sqlerrm using errcode = sqlstate;
end
$$;

alter table tiny.job enable row level security;
create policy job_policy on tiny.job
    for all
    using (current_setting('tiny.owner') = owner)
   	with check (current_setting('tiny.owner') = owner);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop policy job_policy on tiny.job;
alter table tiny.job disable row level security;
-- +goose StatementEnd
