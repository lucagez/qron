package main

import (
	"log"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/lucagez/tinyq/graph"
	"github.com/lucagez/tinyq/graph/generated"
)

const defaultPort = "1234"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &graph.Resolver{}}))

	http.Handle("/", playground.Handler("GraphQL Playground", "/graphql"))
	http.Handle("/graphql", srv)

	log.Printf("starting qron on http://localhost:%s/ 🦕", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
