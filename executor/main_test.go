package executor

import (
	"log"
	"testing"

	"github.com/lucagez/tinyq/migrations"
	"github.com/lucagez/tinyq/testutil"
	"github.com/pressly/goose/v3"
)

func TestMain(m *testing.M) {
	log.Println("initializing PgFactory 🐘")
	goose.SetDialect("postgres")
	goose.SetBaseFS(migrations.MigrationsFS)

	testutil.PG = testutil.NewPgFactory()
	defer testutil.PG.Teardown()

	m.Run()

	log.Println("cleaning up 🧹")
}
