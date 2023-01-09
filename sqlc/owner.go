package sqlc

import (
	"context"
	"errors"
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

func NewScopedPgx(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	config.AfterConnect = func(_ctx context.Context, c *pgx.Conn) error {
		log.Println("Setting role to tinyrole")
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

		// TODO: use positional arguments. Currenlty it throws a syntax error
		// TODO: After reset, tiny.owner is still set but empty
		_, err := c.Exec(_ctx, fmt.Sprintf(`set tiny.owner = '%s'`, owner))
		if err != nil {
			log.Fatal("failed to set owner in acquired connection")
			return false
		}

		log.Println("Acquiring conn from pool")
		return true
	}

	config.AfterRelease = func(c *pgx.Conn) bool {
		// TODO: After reset, tiny.owner is still set but empty
		// -> Add check for non empty string
		_, err := c.Exec(context.Background(), "reset tiny.owner")

		// Destroy conn and remove from pool as it was not reset properly
		log.Println("Returning conn to pool")
		return err == nil
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	return pool, nil
}

// Query wraps each query into a separate transaction to
// always enforce RLS on jobs by attaching local variables
func (p *ScopedPgx) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	owner, ok := ctx.Value(key).(string)
	if !ok {
		return nil, errors.New("no owner found in context")
	}

	conn, err := p.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	// TODO: Add additional timeout to prevent long hangs?
	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, err
	}

	log.Println("SETTING OWNER:", owner)
	_, err = tx.Exec(ctx, fmt.Sprintf(`set local tiny.owner = '%s'`, owner))
	if err != nil {
		tx.Rollback(ctx)
		return nil, err
	}

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		tx.Rollback(ctx)
		return nil, err
	}

	return rows, tx.Commit(ctx)
}
