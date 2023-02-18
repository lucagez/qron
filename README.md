# <img alt="qron" src="https://github.com/lucagez/qron/blob/main/assets/qron.png?raw=true" width="220" />

<!-- TODO: add widgets -->

`qron` is a lightweight, idiomatic and composable job scheduler for building Go applications.
`qron` use Postgres as persistence backend. 
Can scale horizontally and process thousands of jobs per second on each service with minimal footprint.
It's built upon the high performance `pgx` driver to queeze every ounce of performance from Postgres.
It provides `at-least-once` delivery guarantees.

This project came to life from the need of having a reliable background processing library for
the day to day application that need a reliable background job infrastructure without having to pay for a big infrastructure and cognitive overhead.
Postgres is a popular database choice for Go application, paired with Go asynchronous semantics,
it makes possible the processing of quite a huge amount of jobs.

`qron` is **not** the final destination for all your scheduling needs. It will not bring you to unicorn level scale. But that's ok.
When you'll need to handle petabytes per day, it will gently step aside (:

`qron` is made for getting your application going quickly and reliably without having to modify your existing infrastructure. And staying with you for quite a long while (perhaps forever).

Bring your Go binary, Postgres and you are ready to go ⏰

## Features

* ⏰ **Polyglot** - speaking both `cron` spec and `one-off` execution semantics
* ⏳ **Workflow capable** - Providing a state infrastructure for delayed and resumable workflows
* 🪶 **Lightweight** - leveraging Postgres, which you probably are already running
* 🏎 **Fast** - thanks to batching can handle thousands of jobs per second
* 🧱 **Extensible** - providing only the building blocks you need to create reliable systems
* 🪨 **Reliable** - thanks to Postgres every job is delivered `at-least-once`
* 🗣 **Fluent** - using a friendly and intuitive language for scheduling jobs

## Install

`go get -u github.com/lucagez/qron`

## Examples

**Subscriber:**

```go
package main

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lucagez/qron"
)

func main() {
	db, _ := pgxpool.NewWithConfig(context.Background(), config)
	client, _ := qron.NewClient(db, qron.Config{
		PollInterval:  1 * time.Second,
		FlushInterval: 1 * time.Second,
		ResetInterval: 10 * time.Second,
	})

	ctx, stop := context.WithCancel(context.Background())
	defer stop()

	backupJob := client.Fetch(ctx, "backup")
	counterJob := client.Fetch(ctx, "counter")
	emailJob := client.Fetch(ctx, "email")

	for {
		select {
		case <-ctx.Done():
			return
		case job := <-backupJob:
			go executeDailyBackup(job)
		case job := <-counterJob:
			go increaseCounter(job)
		case job := <-emailJob:
			go sendEmail(job)
		}
	}
}

```

**Publisher:**
```go
package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lucagez/qron"
	"github.com/lucagez/qron/graph/model"
)

func main() {
	db, _ := pgxpool.NewWithConfig(context.Background(), config)
	client, _ := qron.NewClient(db, qron.Config{})

	// Every day at midnight
	client.CreateJob(context.Background(), "backup", model.CreateJobArgs{
		Expr: "0 0 * * *",
	})

	// Every 10 seconds
	client.CreateJob(context.Background(), "backup", model.CreateJobArgs{
		Expr: "@every 1 minute 10 seconds",
	})

	// One off
	client.CreateJob(context.Background(), "backup", model.CreateJobArgs{
		Expr: "@after 1 month 1 week 3 days",
	})
}
```

**job handler:**
```go
package handler

func executeDailyBackup(job qron.Job) {
	err := performBackup()
	if err != nil && job.ExecutionAmount > 3 {
		job.Fail()
		return
	}

	if err != nil {
		job.Retry()
		return
	}

	job.Commit()
}
```

## Expression language

The expression language supports both `cron` and `one-off` semantics.

### One off semantics

`@at <timestamp>`

**e.g.** `@at 2023-02-18T17:53:00.000Z`
The job will be picked up for execution exactly at `2023-02-18T17:53:00.000Z`.
The job will be executed only once unless the handler does reschedule it multiple times

`@after <interval>`

**e.g.** `@after 1 hour`, `@after 1s`, `@after 1 year`, `@after 1 week 6 days`, `@after 1 hour 20 minutes`
The job will be picked up for execution after the specified interval is elapsed.
The `<interval>` is any valid postgres `interval` data type. [What is an interval?](#what-is-an-interval)

### Cron semantics

`@every <interval>`

**e.g.** `@every 1 hour`, `@every 1s`, `@every 1 year`, `@every 1 week 6 days`, `@every 1 hour 20 minutes`
The job will be picked up for execution every time `<interval>` is elapsed.
The `<interval>` is any valid postgres `interval` data type. [What is an interval?](#what-is-an-interval)

`crontab`

**e.g.** `* * * * *`, `0 9 * * MON`
The job will be picked up for execution on an interval derived from the `crontab` expression.
`crontab` support scheduling at minute level. It does support ranges (**e.g.** `0 9 * * MON-FRI`).
And just about anything you might need. If in doubt on what is a valid `crontab` expression
you can visit [crontab.guru](https://crontab.guru/)

`@annually`, `@yearly`

Alias for `@every 1 year`

`@monthly`

Alias for `@every 1 month`

`@weekly`

Alias for `@every 1 week`

`@daily`

Alias for `@every 1 day`

`@hourly`

Alias for `@every 1 hour`

`@minutely`

Alias for `@every 1 minute`

## What is an interval?

A `<interval>` is any valid postgres `interval` data type. An interval can contain `years`, `months`, `weeks`, `days`, `hours`, `seconds`, and `microseconds`. Each part can be either positive or negative. However not all of these units play nicely together.
You can refer to [the relevant Postgres documentation](https://www.postgresql.org/docs/current/datatype-datetime.html#DATATYPE-INTERVAL-INPUT).
If in doubt of what is a valid interval, try various combinations in your Postgres repl
```sql
select now() + '1 day 22 hours'::interval;
```
Bear in mind that resolution at which jobs are picked up and processed is dependent on the polling configured polling interval

## License

Copyright (c) 2022-present [Luca Gesmundo](https://github.com/lucagez)

Licensed under [MIT License](./LICENSE)

<!-- TODO: Add links -->