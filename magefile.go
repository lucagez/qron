//go:build mage

package main

import (
	"database/sql"
	"github.com/pressly/goose/v3"
	"log"
	"os"
	"os/exec"
)

// Generate server from openapi spec
func Gen() error {
	f, _ := os.Create("api/gen.go")
	cmd := exec.Command("oapi-codegen", "-package", "api", "spec.yaml")
	cmd.Stderr = os.Stderr
	cmd.Stdout = f
	return cmd.Run()
}

// Create migration
func CreateMigration(name string) error {
	return goose.Create(nil, "migrations", name, "sql")
}

// Up
func UpMigrations() {
	log.Fatalln(goose.Up(connectDb(), "migrations"))
}

// Down
func DownMigrations() {
	log.Fatalln(goose.Down(connectDb(), "migrations"))
}

func connectDb() *sql.DB {
	migrationClient, err := sql.Open("pgx", "postgres://postgres:password@localhost:5435/postgres")
	if err != nil {
		log.Fatalln("unable to connect to database:", err)
	}
	defer migrationClient.Close()
	return migrationClient
}
