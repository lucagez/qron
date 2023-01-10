package sqlc

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ScopedPgx is a scoped connection pool that
// only leverages a lower privileges postgres role
// that is subject to RLS
type ScopedPgx struct {
	*pgxpool.Pool
}

type ownerCtx struct{}

var key = ownerCtx{}

func NewCtx(ctx context.Context, owner string) context.Context {
	return context.WithValue(ctx, key, owner)
}

func FromCtx(ctx context.Context) string {
	key, _ := ctx.Value(key).(string)
	return key
}

func NewScopedPgx(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	config.AfterConnect = func(_ctx context.Context, c *pgx.Conn) error {
		_, err := c.Exec(_ctx, "set role tinyrole")
		return err
	}
	config.BeforeAcquire = func(_ctx context.Context, c *pgx.Conn) bool {
		owner, ok := _ctx.Value(key).(string)
		// Prevent setting owner to empty string
		if !ok {
			log.Fatal("no owner in context")
			return false
		}

		// TODO: use positional arguments. Currently it throws a syntax error
		// TODO: After reset, tiny.owner is still set but empty
		_, err := c.Exec(_ctx, fmt.Sprintf(`set tiny.owner = '%s'`, owner))
		if err != nil {
			log.Fatal("failed to set owner in acquired connection")
			return false
		}

		return true
	}

	config.AfterRelease = func(c *pgx.Conn) bool {
		// TODO: After reset, tiny.owner is still set but empty
		// -> Add check for non empty string
		_, err := c.Exec(context.Background(), "reset tiny.owner")

		// Destroy conn and remove from pool as it was not reset properly
		return err == nil
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	return pool, nil
}
