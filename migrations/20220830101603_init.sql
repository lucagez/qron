-- +goose Up
-- +goose StatementBegin
create schema tiny;

create type tiny.status as enum ('READY', 'PENDING', 'FAILURE', 'SUCCESS');

create or replace function tiny.crontab(expr text)
    returns bool as
$$
declare
    c text := '^(((\d+,)+\d+|(\d+(\/|-)\d+)|(\*(\/|-)\d+)|\d+|\*) +){4}(((\d+,)+\d+|(\d+(\/|-)\d+)|(\*(\/|-)\d+)|\d+|\*) ?)$';
--     c text := '^((((\d+,)+\d+|(\d+(\/|-)\d+)|\d+|\*) ?){5})';
begin
    return case
               when expr ~ c then true
        -- TODO: terrible but keeps monster regex complexity low for now
               when expr ~ 'MON|TUE|WED|THU|FRI|SAT|SUN' then true
               when expr ~ 'JAN|FEB|MAR|APR|MAY|JUN|JUL|AUG|SEP|OCT|NOV|DEC' then true
               else false
        end;
end
$$ language 'plpgsql' immutable;

create domain tiny.cron as text check (
                substr(value, 1, 6) in ('@every', '@after') and (substr(value, 7)::interval) is not null
        or value ~ '^@(annually|yearly|monthly|weekly|daily|hourly|minutely)$'
        or substr(value, 1, 3) = '@at' and (substr(value, 4)::timestamptz) is not null
        or tiny.crontab(value)
    );

-- last run default should be creation date
create or replace function tiny.is_due(cron text, last_run_at timestamptz, by timestamptz)
    returns boolean as
$CODE$
begin
    return case
               when substr(cron, 1, 6) in ('@every', '@after')
                   and (last_run_at + substr(cron, 7)::interval) <= by
                   then true
               when substr(cron, 1, 3) = '@at'
                   and substr(cron, 4)::timestamp <= by
                   and last_run_at < by
                   then true
               when cron ~ '^@(annually|yearly|monthly|weekly|daily|hourly|minutely)$'
                   then case
                            when cron in ('@annually', '@yearly')
                                and (last_run_at + '1 year'::interval) <= by
                                then true
                            when cron = '@monthly'
                                and (last_run_at + '1 month'::interval) <= by
                                then true
                            when cron = '@weekly'
                                and (last_run_at + '1 week'::interval) <= by
                                then true
                            when cron = '@daily'
                                and (last_run_at + '1 day'::interval) <= by
                                then true
                            when cron = '@hourly'
                                and (last_run_at + '1 hour'::interval) <= by
                                then true
                            when cron = '@minutely'
                                and (last_run_at + '1 minute'::interval) <= by
                                then true
                            else false
                   end
               when tiny.crontab(cron)
                   and cronexp.match(by, cron)
                   -- can't be more granular than minute for cron jobs
                   and date_trunc('minute', last_run_at) < date_trunc('minute', by)
                   then true
               else false
        end;
end;
$CODE$
    strict
    language plpgsql;

create table tiny.job
(
    id               bigserial primary key,
    run_at           tiny.cron,
    name             text,
    last_run_at      timestamptz not null default now(),
    created_at       timestamptz not null default now(),
    execution_amount integer              default 0,
    timeout          integer              default 0,
    status           tiny.status not null default 'READY',
    -- state is e2e encrypted so this is never
    -- visible from tinyq. this can hold sensitive data
    state            text,
    -- config is not encrypted as it holds info for the
    -- worker on how to perform the job
    config           text,
    executor         text
);

create index idx_job_name
    on tiny.job (name);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop schema if exists tiny cascade;
-- +goose StatementEnd
