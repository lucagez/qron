package graph

//go:generate go run github.com/99designs/gqlgen@latest generate

import (
	"github.com/lucagez/tinyq/sqlc"
)

// RIPARTIRE QUI!<--
// - briefly: check if it is possible to refactor
//    queries in tiny.go with sqlc compiled queries.
// - 1. Tx for grabbing jobs
// tx, _ := pool.Begin(context.Background())
// q := queries.WithTx(tx)
// q.FetchDueJobs(context.Background(), 50)

// - 2. Batch for updating jobs
// queries.BatchUpdateJobs(ctx, []batch...)

// - Start working on SDK 👈 🎉
// - graphql-generator with graphql-request
// - create class wrapper so to get fluent config
type Resolver struct {
	Queries *sqlc.Queries
}
