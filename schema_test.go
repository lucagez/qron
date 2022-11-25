package tinyq

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/lucagez/tinyq/sqlc"
	"github.com/lucagez/tinyq/testutil"
	"github.com/robfig/cron/v3"
	"github.com/stretchr/testify/assert"
)

type SchemaTest struct {
	TableCatalog string `db:"table_catalog"`
	TableSchema  string `db:"table_schema"`
	TableName    string `db:"table_name"`
}

func TestSchema(t *testing.T) {
	db, cleanup := testutil.PG.CreateDb("schema_assertions")
	defer cleanup()

	queries := sqlc.New(db)

	t.Run("Should create job tables", func(t *testing.T) {
		rows, err := db.Query(context.Background(), `
			select table_catalog, 
				table_schema, 
				table_name 
			from information_schema.tables 
			where table_schema = 'tiny'
		`)

		assert.Nil(t, err)

		var result []SchemaTest
		err = pgxscan.ScanAll(&result, rows)

		assert.Nil(t, err)
		assert.Equal(t, "job", result[0].TableName)
	})

	// Mainly to track what's supported and what's not. Ideally
	// these expressions should move to the list above over time
	// TODO: Improve cron parser checks at api layer
	// until proper sql checks are implemented
	t.Run("Should create supported cron expressions", func(t *testing.T) {
		expressions := map[string]bool{
			"* * * * *":            true,
			"0 12 * * *":           true,
			"15 10 */5 * *":        true,
			"15 10 * * 1":          true,
			"5 0 * 8 *":            true,
			"15 14 1 * *":          true,
			"0 22 * * 1-5":         true,
			"0 0,12 1 */2 *":       true,
			"0 4 8-14 * *":         true,
			"0 0 1,15 * 3":         true,
			"0 0 1,15 * MON":       true,
			"0 0 1,15 * MON-FRI":   true,
			"0 0 1,15 AUG MON-FRI": true,
			"0 0 1,15 JAN-FEB SUN": true,
			"0 0 1,15 JAN-FEB *":   true,

			"23 0-20/2 * * *":        false,
			"15 10 * * ? *":          false,
			"15 10 * * ? 2005":       false,
			"* 14 * * ?":             false,
			"0/5 14 * * ?":           false,
			"0/5 14,18 * * ?":        false,
			"0-5 14 * * ?":           false,
			"15 10 15 * ?":           false,
			"15 10 L * ?":            false,
			"15 10 ? * 6L 2002-2005": false,
			"15 10 ? * 6#3":          false,
			"0 12 1/5 * ?":           false,
			"11 11 11 11 ?":          false,
		}

		for expr, valid := range expressions {
			_, err := db.Exec(context.Background(), `
				insert into tiny.job (run_at, executor) values ($1, 'BANANA')
			`, expr)

			if valid {
				assert.Nil(t, err, expr)
			} else {
				assert.Error(t, err, expr)
			}
		}
	})

	// TODO: Tell people to set timezone and communicate default timezone
	t.Run("Should create @at <timestamp> expressions", func(t *testing.T) {
		expressions := map[string]bool{
			"@at 2022-08-30T11:14:22.607Z":     true,
			"@at 2004-10-19 10:23:54+02":       true,
			"@at 00:00:00.00 UTC":              false,
			"@at 12/17/1997 07:37:16.00 PST":   true,
			"@at Wed Dec 17 07:37:16 1997 PST": true,
			"@at P0001-02-03T04:05:06":         false,
			"@at not-a-timestamp":              false,
		}

		for expr, valid := range expressions {
			_, err := db.Exec(context.Background(), `
				insert into tiny.job (run_at, executor) values ($1, 'BANANA')
			`, expr)

			if valid {
				assert.Nil(t, err)
			} else {
				assert.Error(t, err)
			}
		}
	})

	t.Run("Should create @every <interval> expressions", func(t *testing.T) {
		expressions := map[string]bool{
			"@every 1 hours":     true,
			"@every 234 minutes": true,
			"@every 234 bananas": false,
			"@every 4 year":      true,
			"@every 3 week":      true,
			"@every ok week":     false,
		}

		for expr, valid := range expressions {
			_, err := db.Exec(context.Background(), `
				insert into tiny.job (run_at, executor) values ($1, 'BANANA')
			`, expr)

			if valid {
				assert.Nil(t, err)
			} else {
				assert.Error(t, err)
			}
		}
	})

	t.Run("Should create @after <interval> expressions", func(t *testing.T) {
		expressions := map[string]bool{
			"@after 1 hours":     true,
			"@after 234 minutes": true,
			"@after 234 bananas": false,
			"@after 4 year":      true,
			"@after 3 week":      true,
			"@after ok week":     false,
		}

		for expr, valid := range expressions {
			_, err := db.Exec(context.Background(), `
				insert into tiny.job (run_at, executor) values ($1, 'BANANA')
			`, expr)

			if valid {
				assert.Nil(t, err)
			} else {
				assert.Error(t, err)
			}
		}
	})

	t.Run("Should create defined shortcuts", func(t *testing.T) {
		expressions := map[string]bool{
			"@definitely":  false,
			"@immediately": false,
			"@annually":    true,
			"@yearly":      true,
			"@monthly":     true,
			"@weekly":      true,
			"@daily":       true,
			"@hourly":      true,
			"@minutely":    true,
		}

		for expr, valid := range expressions {
			_, err := db.Exec(context.Background(), `
				insert into tiny.job (run_at, executor) values ($1, 'BANANA')
			`, expr)

			if valid {
				assert.Nil(t, err)
			} else {
				assert.Error(t, err)
			}
		}
	})

	// Only for modified cronexp
	t.Run("Should match cron execution time", func(t *testing.T) {
		type IsMatch struct {
			Expr  string
			Ts    time.Time
			Match bool
		}
		parseTime := func(t string) time.Time {
			parsed, err := time.Parse(time.RFC3339, t)
			if err != nil {
				log.Fatalln("invalid date format", t)
			}
			return parsed
		}
		jobs := []IsMatch{
			{Expr: "* * * * *", Ts: time.Now(), Match: true},
			{Expr: "* * * * *", Ts: time.Now().Add(-1 * time.Minute), Match: true},
			{Expr: "5 4 * * *", Ts: parseTime("2021-08-31T02:05:00.000Z"), Match: false},
			{Expr: "5 0 * * MON", Ts: parseTime("2022-09-05T00:05:00.000Z"), Match: true},
			{Expr: "5 0 * AUG MON", Ts: parseTime("2023-08-07T00:05:00.000Z"), Match: true},
			{Expr: "5 0 * AUG SUN", Ts: parseTime("2023-08-06T00:05:00.000Z"), Match: true},
			{Expr: "* * * APR-AUG SUN", Ts: parseTime("2023-04-02T00:00:00.000Z"), Match: true},
			{Expr: "* * * APR-AUG SUN", Ts: parseTime("2023-04-02T01:02:23.000Z"), Match: true},
			{Expr: "* * * APR-AUG SUN", Ts: parseTime("2023-04-03T01:02:23.000Z"), Match: false},
			{Expr: "* * * APR-AUG SUN", Ts: parseTime("2023-04-09T01:02:23.000Z"), Match: true},
			{Expr: "* * * APR-AUG SUN", Ts: parseTime("2023-07-16T01:02:23.000Z"), Match: true},
			{Expr: "* * * APR-AUG SUN", Ts: parseTime("2023-07-17T01:02:23.000Z"), Match: false},
			{Expr: "* * * * MON-TUE", Ts: parseTime("2022-09-05T01:02:23.000Z"), Match: true},
			{Expr: "* * * * MON-TUE", Ts: parseTime("2022-09-06T01:02:23.000Z"), Match: true},
			{Expr: "* * * * MON-TUE", Ts: parseTime("2022-09-07T01:02:23.000Z"), Match: false},
			{Expr: "5 0 * 8 *", Ts: parseTime("2023-08-01T00:05:00.000Z"), Match: true},
			{Expr: "5 0 * AUG *", Ts: parseTime("2023-08-01T00:05:00.000Z"), Match: true},
			{Expr: "5 0 * FEB *", Ts: parseTime("2023-08-01T00:05:00.000Z"), Match: false},
		}

		for _, job := range jobs {
			rows, err := db.Query(context.Background(), `
				select cronexp.match($1, $2)
			`, job.Ts, job.Expr)
			assert.Nil(t, err)

			var result bool
			err = pgxscan.ScanOne(&result, rows)

			assert.Nil(t, err)
			assert.Equal(t, job.Match, result, job.Expr, job.Ts.String())
		}
	})

	t.Run("Should calculate next execution time", func(t *testing.T) {
		type IsMatch struct {
			Expr string
		}
		jobs := []IsMatch{
			{Expr: "* * * * *"},
			{Expr: "*/5 * * * *"},
			{Expr: "*/4 * 3 * *"},
			{Expr: "*/4 * */3 * *"},
			{Expr: "0 0,12 1 */2 *"},
			{Expr: "0 4 8-14 * *"},
			{Expr: "23 0-20/2 * * *"},
			{Expr: "5 4 * * *"},
			{Expr: "5 0 * * MON"},
			{Expr: "5 0 * 8 MON"},
			{Expr: "5 0 * AUG MON"},
			{Expr: "5 0 * AUG SUN"},
			{Expr: "* * * APR-AUG SUN"},
			{Expr: "* * * APR-AUG SUN"},
			{Expr: "* * * APR-AUG SUN"},
			{Expr: "* * * APR-AUG SUN"},
			{Expr: "* * * APR-AUG SUN"},
			{Expr: "* * * APR-AUG SUN"},
			{Expr: "* * * * MON-TUE"},
			{Expr: "* * * * MON-TUE"},
			{Expr: "* * * * MON-TUE"},
			{Expr: "5 0 * 8 *"},
			{Expr: "5 0 * AUG *"},
			{Expr: "5 0 * FEB *"},
		}

		p := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		for _, job := range jobs {
			schedule, err := p.Parse(job.Expr)
			assert.Nil(t, err)

			nextRun := schedule.Next(time.Now())

			calculatedRun, err := queries.NextRuns(context.Background(), job.Expr)

			assert.Nil(t, err)
			assert.Equal(t, nextRun.Year(), int(calculatedRun.Year), job.Expr)
			assert.Equal(t, nextRun.Month(), time.Month(calculatedRun.Month), job.Expr)
			assert.Equal(t, nextRun.Day(), int(calculatedRun.Day), job.Expr)
			assert.Equal(t, nextRun.Minute(), int(calculatedRun.Min), job.Expr)
			assert.Equal(t, nextRun.Weekday(), time.Weekday(calculatedRun.Dow), job.Expr)
		}
	})

	t.Run("Should find due jobs", func(t *testing.T) {
		type IsDue struct {
			Expr      string
			LastRunAt time.Time
			By        time.Time
			Due       bool
		}
		parseTime := func(t string) time.Time {
			parsed, err := time.Parse(time.RFC3339, t)
			if err != nil {
				log.Fatalln("invalid date format", t)
			}
			return parsed
		}
		cronNextRuns := func(from time.Time, expr string) time.Time {
			p := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
			scheduler, err := p.Parse(expr)
			if err != nil {
				log.Fatal(err)
			}
			return scheduler.Next(from)
		}
		jobs := []IsDue{
			{Expr: "* * * * *", LastRunAt: time.Now(), By: time.Now(), Due: false},
			{Expr: "* * * * *", LastRunAt: time.Now().Add(-1 * time.Minute), By: time.Now(), Due: true},
			{Expr: "15 14 1 * *", LastRunAt: time.Now().Add(-10000 * time.Hour), By: cronNextRuns(time.Now(), "15 14 1 * *"), Due: true},
			// going back for 2 weeks approx
			{Expr: "*/5 * * * MON", LastRunAt: time.Now().Add(-300 * time.Hour), By: cronNextRuns(time.Now(), "*/5 * * * MON"), Due: true},
			{Expr: "@every 10 minutes", LastRunAt: time.Now().Add(-5 * time.Minute), By: time.Now(), Due: false},
			{Expr: "@every 10 minutes", LastRunAt: time.Now().Add(-10 * time.Minute), By: time.Now(), Due: true},
			{Expr: "@after 18 hours", LastRunAt: time.Now().Add(-5 * time.Hour), By: time.Now(), Due: false},
			{Expr: "@after 18 hours", LastRunAt: time.Now().Add(-24 * time.Hour), By: time.Now(), Due: true},
			{Expr: "@at 2023-01-01T00:00:00.000Z", LastRunAt: time.Now(), By: time.Date(2023, 1, 1, 1, 0, 0, 0, time.Local), Due: true},
			{Expr: "@at 2023-01-01T00:00:00.000Z", LastRunAt: time.Now(), By: parseTime("2023-01-01T00:01:00.000Z"), Due: true},
			{Expr: "@annually", LastRunAt: time.Now(), By: time.Now().Add(10000 * time.Hour), Due: true},
			{Expr: "@annually", LastRunAt: time.Now(), By: time.Now().Add(10 * time.Hour), Due: false},
			{Expr: "@yearly", LastRunAt: time.Now(), By: time.Now().Add(10000 * time.Hour), Due: true},
			{Expr: "@yearly", LastRunAt: time.Now(), By: time.Now().Add(10 * time.Hour), Due: false},
			{Expr: "@monthly", LastRunAt: time.Now(), By: time.Now().Add(31 * 24 * time.Hour), Due: true},
			{Expr: "@monthly", LastRunAt: parseTime("2022-01-01T00:00:00.000Z"), By: parseTime("2022-01-31T00:00:00.000Z"), Due: false},
			{Expr: "@monthly", LastRunAt: parseTime("2022-01-01T00:00:00.000Z"), By: parseTime("2022-02-01T00:00:00.000Z"), Due: true},
			{Expr: "@weekly", LastRunAt: parseTime("2022-01-01T00:00:00.000Z"), By: parseTime("2022-01-07T00:00:00.000Z"), Due: false},
			{Expr: "@weekly", LastRunAt: parseTime("2022-01-01T00:00:00.000Z"), By: parseTime("2022-01-08T00:00:00.000Z"), Due: true},
			{Expr: "@daily", LastRunAt: parseTime("2022-01-01T00:00:00.000Z"), By: parseTime("2022-01-01T23:23:00.000Z"), Due: false},
			{Expr: "@daily", LastRunAt: parseTime("2022-01-01T00:00:00.000Z"), By: parseTime("2022-01-02T00:00:00.000Z"), Due: true},
			{Expr: "@hourly", LastRunAt: parseTime("2022-01-01T00:00:00.000Z"), By: parseTime("2022-01-01T00:59:00.000Z"), Due: false},
			{Expr: "@hourly", LastRunAt: parseTime("2022-01-01T00:00:00.000Z"), By: parseTime("2022-01-01T01:00:00.000Z"), Due: true},
			{Expr: "@minutely", LastRunAt: parseTime("2022-01-01T00:00:00.000Z"), By: parseTime("2022-01-01T00:00:59.000Z"), Due: false},
			{Expr: "@minutely", LastRunAt: parseTime("2022-01-01T00:00:00.000Z"), By: parseTime("2022-01-01T00:01:00.000Z"), Due: true},
		}

		for _, job := range jobs {
			rows, err := db.Query(context.Background(), `
				select tiny.is_due($1, $2, $3)
			`, job.Expr, job.LastRunAt, job.By)
			assert.Nil(t, err)

			var result bool
			err = pgxscan.ScanOne(&result, rows)

			assert.Nil(t, err)
			assert.Equal(t, job.Due, result, job.Expr, job.LastRunAt, job.By)
		}
	})
}
