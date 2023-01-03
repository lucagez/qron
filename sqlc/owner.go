package sqlc

import (
	"context"
	"errors"

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

func NewScopedPgx(ctx context.Context, dsn string) (*ScopedPgx, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	config.AfterConnect = func(_ctx context.Context, c *pgx.Conn) error {
		_, err := c.Exec(_ctx, "set role tinyrole")
		return err
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	return &ScopedPgx{
		pool,
	}, nil
}

// Query wraps each query into a separate transaction to
// always enforce RLS on jobs by attaching local variables
func (p *ScopedPgx) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	owner, ok := ctx.Value(key).(string)
	if !ok {
		return nil, errors.New("no owner found in context")
	}

	// TODO: Add additional timeout to prevent long hangs?
	tx, err := p.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}

	_, err = tx.Query(context.Background(), "set local tiny.owner = $1", owner)
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
