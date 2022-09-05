package graph

//go:generate go run github.com/99designs/gqlgen generate

import "github.com/jackc/pgx/v4/pgxpool"

type Resolver struct {
	Db *pgxpool.Pool
}
