-- +goose Up
-- +goose StatementBegin
create schema tiny;

create type tiny.status as enum ('READY', 'PENDING', 'FAILURE', 'SUCCESS', 'PAUSED');

create or replace function tiny.crontab(expr text)
  returns bool as
$$
declare
  c text := '^(((\d+,)+\d+|(\d+(\/|-)\d+)|(\*(\/|-)\d+)|\d+|\*) +){4}(((\d+,)+\d+|(\d+(\/|-)\d+)|(\*(\/|-)\d+)|\d+|\*) ?)$';
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

create or replace function tiny.is_one_shot(expr text)
  returns bool as
$$
begin
  return case
    when substr(expr, 1, 6) in ('@after') then true
    when substr(expr, 1, 3) in ('@at') then true
    else false
  end;
end
$$ language 'plpgsql' immutable;

create or replace function tiny.cron_next_run(
	from_ts timestamptz,
  expr text,
	page int default 0
) returns timestamptz as $$
declare
	day_ts timestamptz;
	result timestamptz;
	groups text[];
  day_fields int[];
  month_fields int[];
  dow_fields int[];
  hour_fields int[];
  minute_fields int[];
begin
	groups = regexp_split_to_array(trim(expr), '\s+');
  if array_length(groups, 1) != 5 then
    raise exception 'invalid parameter "exp": five space-separated fields expected';
  end if;
  
  minute_fields := cronexp.expand_field(groups[1], 0, 59);
  hour_fields := cronexp.expand_field(groups[2], 0, 23);
  day_fields := cronexp.expand_field(groups[3], 1, 31);
  month_fields := cronexp.expand_field(groups[4], 1, 12);
  dow_fields := cronexp.expand_field(groups[5], 0, 7);
  
  if array [7] <@ dow_fields then
    dow_fields := array [0] || dow_fields;
  end if;
  
  -- Find month, day and dow
  select ts into day_ts
  from pg_catalog.generate_series(date_trunc('day', from_ts), date_trunc('day', from_ts) + '5 year'::interval, '1 day'::interval) as ts
  where ts >= date_trunc('day', from_ts)
  and array [date_part('day', ts)::int] <@ day_fields
  and array [date_part('month', ts)::int] <@ month_fields
  and array [date_part('dow', ts)::int] <@ dow_fields
  limit 1
  offset page;
  
  if day_ts is null then
    -- result is out of bounds
    return day_ts;
  end if;

  -- Find hour and minute
  select ts into result
  from pg_catalog.generate_series(day_ts, day_ts + '1 day'::interval, '1 minute'::interval) as ts
  where ts > date_trunc('minute', from_ts)
  and array [date_part('day', ts)::int] <@ day_fields
  and array [date_part('month', ts)::int] <@ month_fields
  and array [date_part('dow', ts)::int] <@ dow_fields
  and array [date_part('hour', ts)::int] <@ hour_fields
  and array [date_part('minute', ts)::int] <@ minute_fields;
  
  if result is null then
    return tiny.cron_next_run(day_ts, expr, page+1);
  end if;
  
	return result;
end;
$$ language plpgsql strict;

-- last run default should be creation date
create or replace function tiny.is_due(last_run_at timestamptz, by timestamptz, expr text)
    returns boolean as
$code$
begin
    return case
               when substr(expr, 1, 6) in ('@every', '@after')
                   and (last_run_at + substr(expr, 7)::interval) <= by
                   then true
               when substr(expr, 1, 3) = '@at'
                   and substr(expr, 4)::timestamp <= by
                   and last_run_at < by
                   then true
               when expr ~ '^@(annually|yearly|monthly|weekly|daily|hourly|minutely)$'
                   then case
                            when expr in ('@annually', '@yearly')
                                and (last_run_at + '1 year'::interval) <= by
                                then true
                            when expr = '@monthly'
                                and (last_run_at + '1 month'::interval) <= by
                                then true
                            when expr = '@weekly'
                                and (last_run_at + '1 week'::interval) <= by
                                then true
                            when expr = '@daily'
                                and (last_run_at + '1 day'::interval) <= by
                                then true
                            when expr = '@hourly'
                                and (last_run_at + '1 hour'::interval) <= by
                                then true
                            when expr = '@minutely'
                                and (last_run_at + '1 minute'::interval) <= by
                                then true
                            else false
                   end
               when tiny.crontab(expr)
                  -- can't be more granular than minute for cron jobs
                  and date_trunc('minute', now()) - tiny.cron_next_run(last_run_at, expr) >= '1 minute'::interval
                  then true
               else false
        end;
end;
$code$
    strict
    language plpgsql;

create or replace function tiny.next(last_run_at timestamptz, expr text)
    returns timestamptz as
$code$
begin
    return case
               when substr(expr, 1, 6) in ('@every', '@after')
                   then (last_run_at + substr(expr, 7)::interval)
               when substr(expr, 1, 3) = '@at'
                   then substr(expr, 4)::timestamp
               when expr ~ '^@(annually|yearly|monthly|weekly|daily|hourly|minutely)$'
                   then case
                            when expr in ('@annually', '@yearly')
                                then (last_run_at + '1 year'::interval)
                            when expr = '@monthly'
                                then (last_run_at + '1 month'::interval)
                            when expr = '@weekly'
                                then (last_run_at + '1 week'::interval)
                            when expr = '@daily'
                                then (last_run_at + '1 day'::interval)
                            when expr = '@hourly'
                                then (last_run_at + '1 hour'::interval)
                            when expr = '@minutely'
                                then (last_run_at + '1 minute'::interval)
                   end
               else tiny.cron_next_run(last_run_at, expr)
        end;
end;
$code$
    strict
    language plpgsql;

create table tiny.job
(
    id               bigserial primary key,
    expr             text not null,

    -- TODO: should truncate by time unit to avoid future drifting?
    run_at           timestamptz not null,
    last_run_at      timestamptz not null default now(),
    created_at       timestamptz not null default now(),
    start_at         timestamptz not null default now(),

    execution_amount integer     not null default 0,
    retries          integer     not null default 5,

    -- TODO: Should `name` ever be null??
    name             text unique not null default substr(md5(random()::text), 0, 25),

    -- meta is used by the executor to
    -- understand how to invoke the job
    meta             json not null default '{}',
    
    -- timeout in seconds
    timeout          integer not null default 120,
    status           tiny.status not null default 'READY',
    -- state is e2e encrypted so this is never
    -- visible from tinyq. this can hold sensitive data
    state            text not null,
    executor         text not null,
    owner            text not null default 'default'
);

alter table tiny.job add constraint positive_timeout check (timeout > 0);

alter table tiny.job add constraint run_format check (
  substr(expr, 1, 6) in ('@every', '@after') and (substr(expr, 7)::interval) is not null
  or expr ~ '^@(annually|yearly|monthly|weekly|daily|hourly|minutely)$'
  or substr(expr, 1, 3) = '@at' and (substr(expr, 4)::timestamptz) is not null
  or tiny.crontab(expr)
);

-- TODO: update to support higher number of retries. 
-- Should increase backoff delay up to `max_delay` that should be configurable as well
alter table tiny.job add constraint max_retries check (retries <= 20);

create index idx_job_name
    on tiny.job (name);

create index idx_job_run_at 
  on tiny.job (run_at);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop schema if exists tiny cascade;
-- +goose StatementEnd
