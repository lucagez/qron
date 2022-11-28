-- +goose Up
-- +goose StatementBegin
create schema tiny;

create type tiny.status as enum ('READY', 'PENDING', 'FAILURE', 'SUCCESS');

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

create or replace function tiny.cron_next_run_wrap(
	from_ts timestamptz,
	page int,
	h_page int,
  expr text
) returns timestamptz as $$
declare
	day_ts timestamptz;
	hour_ts timestamptz;
	min_ts timestamptz;
	ts_parts int[];
	groups text[];
	field_min int[] := '{ 0,  0,  1,  1, 0}';
  field_max int[] := '{59, 23, 31, 12, 7}';
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
  from pg_catalog.generate_series(date_trunc('day', from_ts), date_trunc('day', from_ts) + '1 year'::interval, '1 day'::interval) as ts
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

  -- Find hour
  select ts into hour_ts
  from pg_catalog.generate_series(day_ts, day_ts + '1 day'::interval, '1 hour'::interval) as ts
  where ts >= date_trunc('hour', from_ts)
  and array [date_part('day', ts)::int] <@ day_fields
  and array [date_part('month', ts)::int] <@ month_fields
  and array [date_part('dow', ts)::int] <@ dow_fields
  and array [date_part('hour', ts)::int] <@ hour_fields
  limit 1
  offset h_page;
  
  if hour_ts is null then
    return tiny.cron_next_run_wrap(day_ts, page+1, h_page, expr);
  end if;

  -- Find minute
  select ts into min_ts
  from pg_catalog.generate_series(hour_ts, hour_ts + '1 hour'::interval, '1 minute'::interval) as ts
  where ts > date_trunc('minute', from_ts)
  and array [date_part('day', ts)::int] <@ day_fields
  and array [date_part('month', ts)::int] <@ month_fields
  and array [date_part('dow', ts)::int] <@ dow_fields
  and array [date_part('hour', ts)::int] <@ hour_fields
  and array [date_part('minute', ts)::int] <@ minute_fields;
  
  if min_ts is null then
    return tiny.cron_next_run_wrap(hour_ts, page, h_page+1, expr);
  end if;
  
	return min_ts;
end;
$$ language plpgsql strict;

create or replace function tiny.cron_next_run(
	from_ts timestamptz,
  	expr text
) returns timestamptz as $$
begin
	return tiny.cron_next_run_wrap(from_ts, 0, 0, expr);
end;
$$ language plpgsql strict;

-- last run default should be creation date
create or replace function tiny.is_due(cron text, last_run_at timestamptz, by timestamptz)
    returns boolean as
$code$
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
                  -- can't be more granular than minute for cron jobs
                  and date_trunc('minute', now()) - tiny.cron_next_run(last_run_at, cron) >= '1 minute'::interval
                  then true
               else false
        end;
end;
$code$
    strict
    language plpgsql;

create table tiny.job
(
    id               bigserial primary key,
    run_at           text not null,
    -- RIPARTIRE QUI!<---
    -- [] Turn tiny.is_due into tiny.next. Should give timestamptz of any cron experssion (@every, * * * * *, ...)
    -- [] `next_run_at` should be 👉 default tiny.next(coalesce(last_run_at, created_at), run_at)
    --    👆 Should be updated automatically on each insertion
    -- [] FetchDueJobs should just compare on `next_run_at`
    -- [] refactor signatures (make consistent)
    -- [] rename run_at to `expr`
    -- [] move tiny functions to separate migration
    -- [] should truncate by time unit to avoid future drifting?
    -- next_run_at      timestamptz not null default (custom function),

    -- TODO: Should `name` ever be null??
    name             text,
    last_run_at      timestamptz,
    created_at       timestamptz not null default now(),
    execution_amount integer              default 0,
    timeout          integer              default 0,
    status           tiny.status not null default 'READY',
    -- state is e2e encrypted so this is never
    -- visible from tinyq. this can hold sensitive data
    state            text,
    executor         text not null
);

alter table tiny.job add constraint run_format check (
  substr(run_at, 1, 6) in ('@every', '@after') and (substr(run_at, 7)::interval) is not null
  or run_at ~ '^@(annually|yearly|monthly|weekly|daily|hourly|minutely)$'
  or substr(run_at, 1, 3) = '@at' and (substr(run_at, 4)::timestamptz) is not null
  or tiny.crontab(run_at)
);

create index idx_job_name
    on tiny.job (name);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop schema if exists tiny cascade;
-- +goose StatementEnd
