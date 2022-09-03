-- +goose Up
-- +goose StatementBegin

-- Copyright 2021 Chris Mair <chris@1006.org>
--
-- Redistribution and use in source and binary forms, with or without
-- modification, are permitted provided that the following conditions are met:
--
-- 1. Redistributions of source code must retain the above copyright notice, this
-- list of conditions and the following disclaimer.
--
-- 2. Redistributions in binary form must reproduce the above copyright notice,
-- this list of conditions and the following disclaimer in the documentation
-- and/or other materials provided with the distribution.
--
-- THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
-- ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
-- WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
-- DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
-- FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
-- DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
-- SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
-- CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
-- OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
-- OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

create schema if not exists cronexp;

create or replace function cronexp.expand_field(field text, min int, max int)
    returns int[] as
$$
declare
    part   text;
    groups text[];
    m      int;
    n      int;
    k      int;
    ret    int[];
    tmp    int[];
begin

    -- step 1: basic parameter check

    if coalesce(field, '') = '' then
        raise exception 'invalid parameter "field"';
    end if;

    if min is null or max is null or min < 0 or max < 0 or min > max then
        raise exception 'invalid parameter(s) "min" or "max"';
    end if;

    -- step 2: handle special cases * and */k

    if field = '*' then
        select array_agg(x::int) into ret from generate_series(min, max) as x;
        return ret;
    end if;

    if field ~ '^\*/\d+$' then
        groups = regexp_matches(field, '^\*/(\d+)$');
        k := groups[1];
        if k < 1 or k > max then
            raise exception 'invalid range step: expected a step between 1 and %, got %', max, k;
        end if;
        select array_agg(x::int) into ret from generate_series(min, max, k) as x;
        return ret;
    end if;

    -- step 3: handle generic expression with values, lists or ranges

    if field ~ 'JAN|FEB|MAR|APR|MAY|JUN|JUL|AUG|SEP|OCT|NOV|DEC' then
        field = replace(field, 'JAN', '1');
        field = replace(field, 'FEB', '2');
        field = replace(field, 'MAR', '3');
        field = replace(field, 'APR', '4');
        field = replace(field, 'MAY', '5');
        field = replace(field, 'JUN', '6');
        field = replace(field, 'JUL', '7');
        field = replace(field, 'AUG', '8');
        field = replace(field, 'SEP', '9');
        field = replace(field, 'OCT', '10');
        field = replace(field, 'NOV', '11');
        field = replace(field, 'DEC', '12');
    end if;

    if field ~ 'SUN|MON|TUE|WED|THU|FRI|SAT' then
        field = replace(field, 'SUN', '0');
        field = replace(field, 'MON', '1');
        field = replace(field, 'TUE', '2');
        field = replace(field, 'WED', '3');
        field = replace(field, 'THU', '4');
        field = replace(field, 'FRI', '5');
        field = replace(field, 'SAT', '6');
    end if;

    ret := '{}'::int[];
    for part in select * from regexp_split_to_table(field, ',')
        loop
            if part ~ '^\d+$' then
                n := part;
                if n < min or n > max then
                    raise exception 'value out of range: expected values between % and %, got %', min, max, n;
                end if;
                ret = ret || n;
            elseif part ~ '^\d+-\d+$' then
                groups = regexp_matches(part, '^(\d+)-(\d+)$');
                m := groups[1];
                n := groups[2];
                if m > n then
                    raise exception 'inverted range bounds';
                end if;
                if m < min or m > max or n < min or n > max then
                    raise exception 'invalid range bound(s): expected bounds between % and %, got % and %', min, max, m, n;
                end if;
                select array_agg(x) into tmp from generate_series(m, n) as x;
                ret := ret || tmp;
            elseif part ~ '^\d+-\d+/\d+$' then
                groups = regexp_matches(part, '^(\d+)-(\d+)/(\d+)$');
                m := groups[1];
                n := groups[2];
                k := groups[3];
                if m > n then
                    raise exception 'inverted range bounds';
                end if;
                if m < min or m > max or n < min or n > max then
                    raise exception 'invalid range bound(s): expected bounds between % and %, got % and %', min, max, m, n;
                end if;
                if k < 1 or k > max then
                    raise exception 'invalid range step: expected a step between 1 and %, got %', max, k;
                end if;
                select array_agg(x) into tmp from generate_series(m, n, k) as x;
                ret := ret || tmp;
            else
                raise exception 'invalid expression';
            end if;
        end loop;

    select array_agg(x)
    into ret
    from (select distinct unnest(ret) as x
          order by x) as sub;
    return ret;
end;
$$ language 'plpgsql' immutable;

create or replace function cronexp.match(ts timestamptz, exp text)
    returns boolean as
$$
declare
    field_min int[] := '{ 0,  0,  1,  1, 0}';
    field_max int[] := '{59, 23, 31, 12, 7}';
    groups    text[];
    fields    int[];
    ts_parts  int[];

begin

    if ts is null then
        raise exception 'invalid parameter "ts": must not be null';
    end if;

    if exp is null then
        raise exception 'invalid parameter "exp": must not be null';
    end if;

    groups = regexp_split_to_array(trim(exp), '\s+');
    if array_length(groups, 1) != 5 then
        raise exception 'invalid parameter "exp": five space-separated fields expected';
    end if;

    -- TODO: Build date parts to find next run.
    -- Only matching is not effective.
    -- e.g. 0 21 * * MON - runs at 21.00 on each Monday. But what if the scheduler
    -- for any reason picks that job up at 21.01? It's going to be ignored for a week 🤯
    -- EASY SOLUTION: Keep blind check + add a next_runs table that just every now and then
    -- takes jobs and calculate bunch of next runs. downside -> table is going to grow huge
    -- PROBLEM: How to ensure delayed execution if an outage fucks up db + app for
    -- a prolonged period of time?
    -- Just use everything in HA setup and hope for the best?

    ts_parts[1] := date_part('minute', ts);
    ts_parts[2] := date_part('hour', ts);
    ts_parts[3] := date_part('day', ts);
    ts_parts[4] := date_part('month', ts);
    ts_parts[5] := date_part('dow', ts); -- Sunday = 0

    for n in 1..5
        loop
            fields := cronexp.expand_field(groups[n], field_min[n], field_max[n]);
            -- hack for DOW: fields might contain 0 or 7 for Sunday; if there's a 7, make sure there's a 0 too
            if n = 5 and array [7] <@ fields then
                fields := array [0] || fields;
            end if;
            if not array [ts_parts[n]] <@ fields then
                return false;
            end if;
        end loop;

    return true;
end
$$ language 'plpgsql' immutable;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop schema cronexp cascade;
-- +goose StatementEnd
