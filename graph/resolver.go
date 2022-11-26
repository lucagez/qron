package graph

//go:generate go run github.com/99designs/gqlgen@latest generate

import (
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/lucagez/tinyq/sqlc"
)

// RIPARTIRE QUI!<--
// - Start working on SDK 👈 🎉
// - graphql-generator with graphql-request
// - create class wrapper so to get fluent config
// - implement timeout for automatic clearing of job
// - strategy for fast cron due comparison
// - implement `start_at` e.g. every week starting from monday
// - implement `aquired_at` e.g. useful for keeping track of jobs that failed silenlty
type Resolver struct {
	Queries *sqlc.Queries
	DB      *pgxpool.Pool
}
