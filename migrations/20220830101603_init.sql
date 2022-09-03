-- +goose Up
-- +goose StatementBegin
create schema tiny;

CREATE TYPE tiny.status AS ENUM ('READY', 'PENDING', 'FAILURE', 'SUCCESS');

-- INTERVAL: @every x time ::interval
-- EXACT: @at x time ::timestamptz
-- CRON: {cron expr} ::text
CREATE TYPE tiny.job_kind AS ENUM ('INTERVAL', 'EXACT', 'CRON');

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

CREATE DOMAIN tiny.cron AS TEXT CHECK (
                substr(VALUE, 1, 6) IN ('@every', '@after') AND (substr(VALUE, 7)::INTERVAL) IS NOT null
        or VALUE ~ '^@(annually|yearly|monthly|weekly|daily|hourly|minutely)$'
        or substr(VALUE, 1, 3) = '@at' AND (substr(VALUE, 4)::timestamptz) IS NOT null
        OR tiny.crontab(VALUE)
    );

-- last run default should be creation date
CREATE OR REPLACE FUNCTION tiny.is_due(cron text, last_run_at timestamptz, by timestamptz)
    RETURNS boolean AS
$CODE$
begin
    return case
               when substr(cron, 1, 6) IN ('@every', '@after')
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
END;
$CODE$
    STRICT
    LANGUAGE plpgsql;

create type tiny.kind as enum ('INTERVAL', 'TASK', 'CRON');

-- format while inserting job
CREATE OR REPLACE FUNCTION tiny.find_kind(cron tiny.cron)
    RETURNS tiny.kind AS
$CODE$
begin
    return case
               when substr(cron, 1, 6) = '@every'
                   or cron ~ '^@(annually|yearly|monthly|weekly|daily|hourly|minutely)$'
                   then 'INTERVAL'::tiny.kind
               when substr(cron, 1, 3) = '@at'
                   or substr(cron, 1, 6) = '@after'
                   then 'TASK'::tiny.kind
               when tiny.crontab(cron)
                   then 'CRON'::tiny.kind
        end;
END;
$CODE$
    STRICT
    LANGUAGE plpgsql;


CREATE table tiny.job
(
    id               BIGSERIAL PRIMARY KEY,
    run_at           tiny.cron,
    name             text,
    last_run_at      timestamptz not null default now(),
    created_at       timestamptz not null default now(),
    execution_amount integer               default 0,
    timeout          INTEGER               DEFAULT 0,
    status           tiny.status not null default 'READY',
    -- state is e2e encrypted so this is never
    -- visible from tinyq. this can hold sensitive data
    state            text,
    -- config is not encrypted as it holds info for the
    -- worker on how to perform the job
    config           text,
    kind             tiny.kind,
    executor         text
);

CREATE INDEX idx_job_name
    ON tiny.job (name);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop schema if exists tiny cascade;
-- +goose StatementEnd
