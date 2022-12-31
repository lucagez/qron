package sqlc

import (
	"context"
	"errors"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ScopedPgx is a scoped connection pool that
// only leverages a lower privileges postgres role
// that is subject to RLS
type ScopedPgx struct {
	pool *pgxpool.Pool
}

type ownerCtx struct{}

var key = ownerCtx{}

func NewCtx(ctx context.Context, owner string) context.Context {
	return context.WithValue(ctx, key, owner)
}

func NewScopedPgx(dsn string) ScopedPgx {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatal("failed to parse config:", err)
	}
	config.AfterConnect = func(ctx context.Context, c *pgx.Conn) error {
		_, err := c.Exec(ctx, "set role tinyrole")
		return err
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		log.Fatal("error while establishing pool connection:", err)
	}

	return ScopedPgx{
		pool: pool,
	}
}

// Query wraps each query into a separate transaction to
// always enforce RLS on jobs by attaching local variables
func (p *ScopedPgx) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	owner, ok := ctx.Value(key).(string)
	if !ok {
		return nil, errors.New("no owner found in context")
	}

	// TODO: Add additional timeout to prevent long hangs?
	tx, err := p.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}

	_, err = tx.Query(context.Background(), "set local tiny.owner = $1", owner)
	if err != nil {
		tx.Rollback(ctx)
		return nil, err
	}
	rows, err := p.pool.Query(ctx, query, args...)
	if err != nil {
		tx.Rollback(ctx)
		return nil, err
	}

	return rows, tx.Commit(ctx)
}

func (p *ScopedPgx) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	return p.pool.Exec(ctx, query, args...)
}

func (p *ScopedPgx) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	return p.pool.QueryRow(ctx, query, args...)
}

func (p *ScopedPgx) SendBatch(ctx context.Context, batch *pgx.Batch) pgx.BatchResults {
	return p.pool.SendBatch(ctx, batch)
}
