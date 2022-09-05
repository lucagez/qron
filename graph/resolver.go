package graph

//go:generate go run github.com/99designs/gqlgen@latest generate

import (
	"github.com/lucagez/tinyq/sqlc"
)

// RIPARTIRE QUI!<--
// - Start working on SDK 👈 🎉
// - graphql-generator with graphql-request
// - create class wrapper so to get fluent config
type Resolver struct {
	Queries *sqlc.Queries
}
