package client

import (
	"fmt"
	"testing"

	_ "github.com/jackc/pgx/stdlib"
	"github.com/lucagez/tinyq/graph/model"
	"github.com/lucagez/tinyq/testutil"
	"github.com/stretchr/testify/assert"
)

func TestClient(t *testing.T) {
	pool, cleanup := testutil.PG.CreateDb("client_0")
	defer cleanup()

	port := pool.Config().ConnConfig.Port
	client, err := NewClient(Config{
		Dsn: fmt.Sprintf("postgres://postgres:postgres@localhost:%d/client_0", port),
	})
	assert.Nil(t, err)
	defer client.Close()

	t.Run("Should fetch", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			client.CreateJob("backup", model.CreateJobArgs{})
		}

	})
}
