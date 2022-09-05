package graph

//go:generate go run github.com/99designs/gqlgen@latest generate

import (
	"github.com/lucagez/tinyq/sqlc"
)

// RIPARTIRE QUI!<--
// - Test gql api
// - If it works, remove openapi stuff
// - Start working on SDK
type Resolver struct {
	Queries sqlc.Queries
}
