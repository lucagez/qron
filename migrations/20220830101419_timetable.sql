-- +goose Up
-- +goose StatementBegin

-- PostgreSQL License

-- Copyright (c) 2018-2020, Cybertec Schönig & Schönig GmbH

-- Permission to use, copy, modify, and distribute this software and its
-- documentation for any purpose, without fee, and without a written agreement is
-- hereby granted, provided that the above copyright notice and this paragraph
-- and the following two paragraphs appear in all copies.

-- IN NO EVENT SHALL Cybertec Schönig & Schönig GmbH BE LIABLE TO ANY PARTY FOR DIRECT, INDIRECT,
-- SPECIAL, INCIDENTAL, OR CONSEQUENTIAL DAMAGES, INCLUDING LOST PROFITS, ARISING
-- OUT OF THE USE OF THIS SOFTWARE AND ITS DOCUMENTATION, EVEN IF Cybertec Schönig & Schönig GmbH
-- HAS BEEN ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

-- Cybertec Schönig & Schönig GmbH SPECIFICALLY DISCLAIMS ANY WARRANTIES, INCLUDING, BUT NOT
-- LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A
-- PARTICULAR PURPOSE. THE SOFTWARE PROVIDED HEREUNDER IS ON AN "AS IS" BASIS,
-- AND Cybertec Schönig & Schönig GmbH HAS NO OBLIGATIONS TO PROVIDE MAINTENANCE, SUPPORT, UPDATES,
-- ENHANCEMENTS, OR MODIFICATIONS.

-- Initial code adapted from: https://github.com/cybertec-postgresql/pg_timetable/blob/master/internal/pgengine/sql/cron_functions.sql

create schema if not exists timetable;

create or replace function timetable.cron_split_to_arrays(
    cron text,
    out mins integer[],
    out hours integer[],
    out days integer[],
    out months integer[],
    out dow integer[]
) returns record as $$
declare
    a_element text[];
    i_index integer;
    a_tmp text[];
    tmp_item text;
    a_range int[];
    a_split text[];
    a_res integer[];
    allowed_range integer[];
    max_val integer;
    min_val integer;
begin
    a_element := regexp_split_to_array(cron, '\s+');
    for i_index in 1..5 loop
        a_res := null;
        a_tmp := string_to_array(a_element[i_index],',');
        case i_index -- 1 - mins, 2 - hours, 3 - days, 4 - weeks, 5 - DOWs
            when 1 then allowed_range := '{0,59}';
            when 2 then allowed_range := '{0,23}';
            when 3 then allowed_range := '{1,31}';
            when 4 then allowed_range := '{1,12}';
        else
            allowed_range := '{0,7}';
        end case;
        foreach tmp_item in array a_tmp loop
	        
	        if tmp_item ~ 'JAN|FEB|MAR|APR|MAY|JUN|JUL|AUG|SEP|OCT|NOV|DEC' then
		        tmp_item = replace(tmp_item, 'JAN', '1');
		        tmp_item = replace(tmp_item, 'FEB', '2');
		        tmp_item = replace(tmp_item, 'MAR', '3');
		        tmp_item = replace(tmp_item, 'APR', '4');
		        tmp_item = replace(tmp_item, 'MAY', '5');
		        tmp_item = replace(tmp_item, 'JUN', '6');
		        tmp_item = replace(tmp_item, 'JUL', '7');
		        tmp_item = replace(tmp_item, 'AUG', '8');
		        tmp_item = replace(tmp_item, 'SEP', '9');
		        tmp_item = replace(tmp_item, 'OCT', '10');
		        tmp_item = replace(tmp_item, 'NOV', '11');
		        tmp_item = replace(tmp_item, 'DEC', '12');
		    end if;
		
		    if tmp_item ~ 'SUN|MON|TUE|WED|THU|FRI|SAT' then
		        tmp_item = replace(tmp_item, 'SUN', '0');
		        tmp_item = replace(tmp_item, 'MON', '1');
		        tmp_item = replace(tmp_item, 'TUE', '2');
		        tmp_item = replace(tmp_item, 'WED', '3');
		        tmp_item = replace(tmp_item, 'THU', '4');
		        tmp_item = replace(tmp_item, 'FRI', '5');
		        tmp_item = replace(tmp_item, 'SAT', '6');
		    end if;
	        
            if tmp_item ~ '^[0-9]+$' then -- normal integer
                a_res := array_append(a_res, tmp_item::int);
            elsif tmp_item ~ '^[*]+$' then -- '*' any value
                a_range := array(select generate_series(allowed_range[1], allowed_range[2]));
                a_res := array_cat(a_res, a_range);
            elsif tmp_item ~ '^[0-9]+[-][0-9]+$' then -- '-' range of values
                a_range := regexp_split_to_array(tmp_item, '-');
                a_range := array(select generate_series(a_range[1], a_range[2]));
                a_res := array_cat(a_res, a_range);
            elsif tmp_item ~ '^[0-9]+[\/][0-9]+$' then -- '/' step values
                a_range := regexp_split_to_array(tmp_item, '/');
                a_range := array(select generate_series(a_range[1], allowed_range[2], a_range[2]));
                a_res := array_cat(a_res, a_range);
            elsif tmp_item ~ '^[0-9-]+[\/][0-9]+$' then -- '-' range of values and '/' step values
                a_split := regexp_split_to_array(tmp_item, '/');
                a_range := regexp_split_to_array(a_split[1], '-');
                a_range := array(select generate_series(a_range[1], a_range[2], a_split[2]::int));
                a_res := array_cat(a_res, a_range);
            elsif tmp_item ~ '^[*]+[\/][0-9]+$' then -- '*' any value and '/' step values
                a_split := regexp_split_to_array(tmp_item, '/');
                a_range := array(select generate_series(allowed_range[1], allowed_range[2], a_split[2]::int));
                a_res := array_cat(a_res, a_range);
            else
                raise exception 'Value ("%") not recognized', a_element[i_index]
                    using hint = 'fields separated by space or tab.'+
                       'Values allowed: numbers (value list with ","), '+
                    'any value with "*", range of value with "-" and step values with "/"!';
            end if;
        end loop;
        select
           array_agg(x.val), min(x.val), max(x.val) into a_res, min_val, max_val
        from (
            select distinct unnest(a_res) as val order by val) as x;
        if max_val > allowed_range[2] or min_val < allowed_range[1] then
            raise exception '% is out of range: %', a_res, allowed_range;
        end if;
        case i_index
            when 1 then mins := a_res;
            when 2 then hours := a_res;
            when 3 then days := a_res;
            when 4 then months := a_res;
        else
            dow := a_res;
        end case;
    end loop;
    return;
end;
$$ language plpgsql strict;

create or replace function timetable.cron_months(
    from_ts timestamptz,
    allowed_months int[]
) returns setof timestamptz as $$
    with
    am(am) as (select unnest(allowed_months)),
    genm(ts) as ( --generated months
        select date_trunc('month', ts)
        from pg_catalog.generate_series(from_ts, from_ts + '1 year'::interval, '1 month'::interval) g(ts)
    )
    select ts from genm join am on date_part('month', genm.ts) = am.am
$$ language sql strict;

create or replace function timetable.cron_days(
    from_ts timestamptz,
    allowed_months int[],
    allowed_days int[],
    allowed_week_days int[]
) returns setof timestamptz as $$
    with
    ad(ad) as (select unnest(allowed_days)),
    am(am) as (select * from timetable.cron_months(from_ts, allowed_months)),
    gend(ts) as ( --generated days
        select date_trunc('day', ts)
        from am,
            pg_catalog.generate_series(am.am, am.am + '1 month'::interval
                - '1 day'::interval,  -- don't include the same day of the next month
                '1 day'::interval) g(ts)
    )
    select ts
    from gend join ad on date_part('day', gend.ts) = ad.ad
    where extract(dow from ts) = any(allowed_week_days)
$$ language sql strict;

create or replace function timetable.cron_times(
    allowed_hours int[],
    allowed_minutes int[]
) returns setof time AS $$
    with
    ah(ah) as (select unnest(allowed_hours)),
    am(am) as (select unnest(allowed_minutes))
    select make_time(ah.ah, am.am, 0) from ah cross join am
$$ language sql strict;

create or replace function timetable.cron_runs(
    from_ts timestamptz,
    to_ts timestamptz,
    cron text
) returns setof timestamptz as $$
    select cd + ct
    from
        timetable.cron_split_to_arrays(cron) a,
        timetable.cron_times(a.hours, a.mins) ct cross join
        timetable.cron_days(from_ts, a.months, a.days, a.dow) cd
    where cd + ct > from_ts
    and cd + ct < to_ts
    order by 1 asc;
$$ language sql strict;

-- is_cron_in_time returns TRUE if timestamp is listed in cron expression
create or replace function timetable.is_cron_in_time(
    run_at text,
    ts timestamptz
) returns boolean as $$
    select
    case when run_at is null then
        true
    else
        date_part('month', ts) = any(a.months)
        and (date_part('dow', ts) = any(a.dow) or date_part('isodow', ts) = any(a.dow))
        and date_part('day', ts) = any(a.days)
        and date_part('hour', ts) = any(a.hours)
        and date_part('minute', ts) = any(a.mins)
    end
    from
        timetable.cron_split_to_arrays(run_at) a
$$ language sql;


-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop schema timetable cascade;
-- +goose StatementEnd
