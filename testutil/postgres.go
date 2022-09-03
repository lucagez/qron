package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/lucagez/tinyq/migrations"
	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
	"github.com/pressly/goose/v3"
	"log"
	"time"
)

type PgFactory struct {
	Docker             *dockertest.Pool
	MaintainanceDb     *dockertest.Resource
	MaintainanceClient *pgxpool.Pool
}

var PG PgFactory

func init() {
	goose.SetDialect("postgres")
	goose.SetBaseFS(migrations.MigrationsFS)

	PG = NewPgFactory()
}

func NewPgFactory() PgFactory {
	log.Println("initializing PgFactory 🐘")

	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalln(err)
	}

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		// TODO: Use consistent version
		//Tag:        "11",
		Env: []string{
			"POSTGRES_PASSWORD=postgres",
			"POSTGRES_USER=postgres",
			"POSTGRES_DB=postgres",
			"listen_addresses='*'",
		},
	}, func(config *docker.HostConfig) {
		// set AutoRemove to true so that stopped container goes away by itself
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		log.Fatalf("could not start resource: %s", err)
	}

	// TODO: find a best way if waiting for pg to be ready

	var client *pgxpool.Pool
	counter := 0

	for {
		time.Sleep(500 * time.Millisecond)

		client, err = pgxpool.Connect(
			context.Background(),
			fmt.Sprintf(
				"postgres://postgres:postgres@%s/%s?sslmode=disable",
				resource.GetHostPort("5432/tcp"),
				"postgres",
			),
		)
		if client != nil {
			log.Println("pool is up and running")
			break
		}
		if err != nil && counter > 5 {
			log.Fatalln("could not connect to maintenance db:", err)
		}

		counter++
	}

	return PgFactory{
		Docker:             pool,
		MaintainanceDb:     resource,
		MaintainanceClient: client,
	}
}

func (p PgFactory) CreateDb(name string) (*pgxpool.Pool, func()) {
	_, err := p.MaintainanceClient.Exec(
		context.Background(), fmt.Sprintf("create database %s", name))
	if err != nil {
		log.Fatalln("failed to created db", name, ":", err)
	}

	dbUrl := fmt.Sprintf(
		"postgres://postgres:postgres@%s/%s?sslmode=disable",
		p.MaintainanceDb.GetHostPort("5432/tcp"),
		"postgres",
	)

	migrationClient, err := sql.Open("pgx", dbUrl)
	if err != nil {
		log.Fatalln("unable to connect to database:", err)
	}
	defer migrationClient.Close()

	err = goose.Up(migrationClient, ".")
	if err != nil {
		log.Fatalln("unable to run migrations:", err)
	}

	client, err := pgxpool.Connect(
		context.Background(),
		dbUrl,
	)
	if err != nil {
		log.Fatalln("failed to connect to db", name, ":", err)
	}

	return client, func() {
		client.Close()
		_, err = p.MaintainanceClient.Exec(
			context.Background(), fmt.Sprintf("drop database %s", name))
		if err != nil {
			log.Fatalln("failed to drop db", name, ":", err)
		}
	}
}

func (p PgFactory) Teardown() error {
	return p.MaintainanceDb.Close()
}
