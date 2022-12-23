package graph

//go:generate go run github.com/99designs/gqlgen@latest generate

import (
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/lucagez/tinyq/sqlc"
)

// RIPARTIRE QUI!<--
// [✅] implement timeout for automatic clearing of job
// [✅] implement `start_at` e.g. every week starting from monday
// [✅] implement `aquired_at` e.g. useful for keeping track of jobs that failed silenlty
// [] how to add subscribe (fanout)?
// [] implement client interface (local / remote client)
// [] implement remote client (gqlgenc for autogenerated one)
// [] implement `tinyd` tiny daemon. that leverages remote client and replay messages through http (or docker container)
// [✅] flush remaining jobs after cancel (use separate context for flush and fetch)
// [] add test for flushing remaining in-flight job after canceling fetch
// [] better handling for `state` field. Should it be optional?
// [] implment idempotency for client
// [] implement `@asap` operator
type Resolver struct {
	Queries *sqlc.Queries
	DB      *pgxpool.Pool
}
