package tinyq

import (
	"context"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/lucagez/tinyq/testutil"
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
	"time"
)

type SchemaTest struct {
	TableCatalog string `db:"table_catalog"`
	TableSchema  string `db:"table_schema"`
	TableName    string `db:"table_name"`
}

func TestSchema(t *testing.T) {
	db, cleanup := testutil.PG.CreateDb("schema_assertions")
	defer cleanup()

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
	t.Run("Should create supported cron expressions", func(t *testing.T) {
		expressions := map[string]bool{
			"* * * * *":      true,
			"0 12 * * *":     true,
			"15 10 */5 * *":  true,
			"15 10 * * 1":    true,
			"5 0 * 8 *":      true,
			"15 14 1 * *":    true,
			"0 22 * * 1-5":   true,
			"0 0,12 1 */2 *": true,
			"0 4 8-14 * *":   true,
			"0 0 1,15 * 3":   true,

			"23 0-20/2 * * *":        false,
			"15 10 * * ? *":          false,
			"15 10 * * ? 2005":       false,
			"* 14 * * ?":             false,
			"0/5 14 * * ?":           false,
			"0/5 14,18 * * ?":        false,
			"0-5 14 * * ?":           false,
			"10,44 14 ? 3 WED":       false,
			"15 10 ? * MON-FRI":      false,
			"15 10 15 * ?":           false,
			"15 10 L * ?":            false,
			"15 10 ? * 6L 2002-2005": false,
			"15 10 ? * 6#3":          false,
			"0 12 1/5 * ?":           false,
			"11 11 11 11 ?":          false,
		}

		for expr, valid := range expressions {
			_, err := db.Exec(context.Background(), `
				insert into tiny.job (run_at) values ($1::tiny.cron)
			`, expr)

			if valid {
				assert.Nil(t, err)
			} else {
				assert.Error(t, err)
			}
		}
	})

	// TODO: Tell people to set timezone and communicate default timezone
	t.Run("Should create @at expressions", func(t *testing.T) {
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
				insert into tiny.job (run_at) values ($1::tiny.cron)
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
				insert into tiny.job (run_at) values ($1::tiny.cron)
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
				insert into tiny.job (run_at) values ($1::tiny.cron)
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
				insert into tiny.job (run_at) values ($1::tiny.cron)
			`, expr)

			if valid {
				assert.Nil(t, err)
			} else {
				assert.Error(t, err)
			}
		}
	})

	t.Run("Should find correct kind", func(t *testing.T) {
		expressions := map[string]Kind{
			"@annually": INTERVAL,
			"@yearly":   INTERVAL,
			"@monthly":  INTERVAL,
			"@weekly":   INTERVAL,
			"@daily":    INTERVAL,
			"@hourly":   INTERVAL,
			"@minutely": INTERVAL,

			"@after 1 hours":     TASK,
			"@after 234 minutes": TASK,
			"@after 4 year":      TASK,
			"@after 3 week":      TASK,

			"@every 1 hours":     INTERVAL,
			"@every 234 minutes": INTERVAL,
			"@every 4 year":      INTERVAL,
			"@every 3 week":      INTERVAL,

			"@at 2022-08-30T11:14:22.607Z":     TASK,
			"@at 2004-10-19 10:23:54+02":       TASK,
			"@at 12/17/1997 07:37:16.00 PST":   TASK,
			"@at Wed Dec 17 07:37:16 1997 PST": TASK,

			"* * * * *":      CRON,
			"0 12 * * *":     CRON,
			"15 10 */5 * *":  CRON,
			"15 10 * * 1":    CRON,
			"5 0 * 8 *":      CRON,
			"15 14 1 * *":    CRON,
			"0 22 * * 1-5":   CRON,
			"0 0,12 1 */2 *": CRON,
			"0 4 8-14 * *":   CRON,
			"0 0 1,15 * 3":   CRON,
		}

		for expr, kind := range expressions {
			rows, err := db.Query(context.Background(), `
				select tiny.find_kind($1)
			`, expr)

			var result Kind
			err = pgxscan.ScanOne(&result, rows)

			assert.Nil(t, err)
			assert.Equal(t, kind, result, expr)
		}
	})

	t.Run("Should find due jobs", func(t *testing.T) {
		type IsDue struct {
			Expr      string
			LastRunAt time.Time
			Due       bool
		}
		parseTime := func(t string) time.Time {
			parsed, err := time.Parse(time.RFC3339, t)
			if err != nil {
				log.Fatalln("invalid date format", t)
			}
			return parsed
		}
		// RIPARTIRE QUI! <--
		// - Test is due stuff..https://codeberg.org/chris-mair/postgres-cronexp
		// - add `by` in jobs to check
		// - test cron due dates with https://github.com/adhocore/gronx
		jobs := []IsDue{
			{Expr: "* * * * *", LastRunAt: time.Now(), Due: false},
			{Expr: "* * * * *", LastRunAt: time.Now().Add(-1 * time.Minute), Due: true},
			{Expr: "5 4 * * *", LastRunAt: parseTime("2021-08-31T02:05:00.000Z"), Due: false},
			{Expr: "5 4 * * *", LastRunAt: parseTime("2021-08-30T00:05:00.000Z"), Due: true},
		}

		for _, job := range jobs {
			rows, err := db.Query(context.Background(), `
				select tiny.is_due($1, $2, now())
			`, job.Expr, job.LastRunAt)

			var result bool
			err = pgxscan.ScanOne(&result, rows)

			assert.Nil(t, err)
			assert.Equal(t, job.Due, result, job.Expr, job.LastRunAt.String())
		}
	})
}
