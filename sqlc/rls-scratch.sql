ALTER TABLE tiny.job ENABLE ROW LEVEL SECURITY;

begin;
set local tiny.owner = 'pastrami';
select count(*) from tiny.job j;
commit;


begin;
set local tiny.owner = 'bobler';
insert into tiny.job (run_at, expr, state, executor, owner) values(now() + '1 hour'::interval, '* * * * *', 'ok', 'default', 'bobler');
commit;

rollback;

select current_user;

grant all on tiny.job to bob;

insert into tiny.job (run_at, expr, state, executor, owner) values(now() + '1 hour'::interval, '* * * * *', 'ok', 'default', 'samaiel');

alter table tiny.job add column owner text;

select coalesce(current_setting('tiny.owner', true), 'default');

create role client;

set role postgres;
set role client;

GRANT ALL ON SCHEMA public TO client;
GRANT USAGE, SELECT ON SEQUENCE tiny.job_id_seq TO client;
GRANT ALL ON SCHEMA tiny TO client;
GRANT ALL ON tiny.job TO client;



-- thiz
CREATE POLICY job_policy ON tiny.job
    FOR ALL
    USING (current_setting('tiny.owner') = owner)
   	with check (current_setting('tiny.owner') = owner);
   


   
drop policy job_policy on tiny.job;