-- Experiments

-- TODO: Improve cron type. @after / @at .. are only input types? -> cron_expr, exact_expr, interval_expr
--                                                                   👉 As they are the 3 different kind
--                                                                      of supported recurrent executions

insert into tiny.job (run_at)
values ('@annually');

INSERT INTO tiny.job(run_at, status, state, kind)
SELECT '@every 1 minute', 'READY', '{"hello": "world"}', tiny.find_kind('@every 1 minute')
FROM generate_series(1, 1000000)

-- demo transaction
begin;

with due_jobs as (select *
              from tiny.job
              where tiny.is_due(run_at, coalesce(last_run_at, created_at))
              and status = 'READY'
              -- worker limit
              limit 100 for update
                  skip locked)
update tiny.job
set status      = 'PENDING',
    last_run_at = now()
from due_jobs
where due_jobs.id = tiny.job.id
returning due_jobs.*;

commit;

rollback;

vacuum;

select count(*)
from tiny.job
where status = 'READY';

select * from tiny.job limit 10;


/*
 * PSEUDOCODE:
 *
 * pool = connect_to_pg
 *
 * for {
 * 		// either wait for a notification NOTIFY/LISTEN
 * 		// (e.g. queue jobs that should be executed right away)
 * 		// or for the maximum waiting time
 * 		select {
 * 		case <-notificationChannel:
 * 			break?
 * 		case <-timeoutChannel:
 * 			break?
 * 		}
 *
 * 		start tx
 *
 * 		jobs :=

`
with jobs as (
	select *
	from tiny.job
	where tiny.is_due(run_at, coalesce(last_run_at, created_at))
	-- useful for @at jobs
	and status = 'pending'
	-- worker limit
	limit 100
	for update
	skip locked
)
update tiny.job as a
set status = 'pending',
	last_run_at = now()
from jobs b
where a.id = b.id
returning *;

`

 *
 * 		for job := range jobs {
 * 			go executeJob(job, terminationContext)
 * 		}
 * }
 *
 * on SIGTERM {
 * 		disconnect from pg
 * 		terminationContext.Done()
 * }
 *
 * func executeJob(job, terminationContext) {
 *
 * 		result := make(chan execResult, 1)
 *      go func() {
 *          result <- Executor(job)
 *      }()
 *
 * 		select {
 * 		case <-result:
 * 			- store execution result
 * 			- if task is `@at` -> set task as `TERMINATED`
 * 			- in case of error implement retry policy
 * 			- based on result decide if further jobs should be scheduled. Only valid for `@every` and `@at`
 * 		case <-time.After(timeout * time.Second):
 * 			interruptTask // How to interrupt correctly? e.g. not to leave open sockets around on http requests
 * 			set task as `READY`
 * 		case <- terminationContext.Done?:
 * 			cleanupTask
 * 			wait for up to x seconds and then terminate anyway
 *
 * 		}
 * }
 *
 * TODO: SDK leave `retry` to the user. e.g. inside a cron context the default retry policy is anyway at next execution
 * TODO: Should have a logs table for logs/results/errors
 * TODO: Should be possible to create graph of jobs. Job should have `parent_id` (which is again a job that triggered this very job)
 * TODO: Should have data column (e2e encrypted ONLY by SDK)
 * TODO: Should have executor column
 *
 * e.g. create job @asap
 * - job is created but then `retry @after 50 hours`
 * - successful, `TERMINATED`
 *
 * id  name     parent_id
 * 1   asap     null
 * 2   retry    1
 * 3   finally  2
 *
 * -> A chain of jobs is created automatically
 *
 * TODO: SDK should have ctx.log(anything serializable) -> So, it is possible to debug a full chain of jobs
 */