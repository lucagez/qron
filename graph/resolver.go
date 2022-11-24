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
// - gqlgen go models are not introspected. Enhance graph/job.graphql:19
type Resolver struct {
	Queries *sqlc.Queries
	DB      *pgxpool.Pool
}
