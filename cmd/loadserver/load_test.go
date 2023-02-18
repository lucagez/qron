package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lucagez/qron"
	"github.com/lucagez/qron/graph/model"
)

func BenchmarkFetch(b *testing.B) {
	// pool, cleanup := testutil.PG.CreateDb("bench_client")
	// defer cleanup()

	pool, err := pgxpool.New(context.Background(), "postgres://postgres:password@localhost:5435/postgres?sslmode=disable")
	if err != nil {
		b.Fatal(err)
	}

	client, err := qron.NewClient(pool, qron.Config{
		PollInterval:  1 * time.Millisecond,
		FlushInterval: 10 * time.Millisecond,
		ResetInterval: 10 * time.Millisecond,
		MaxInFlight:   1000,
	})
	if err != nil {
		b.Fatal(err)
	}

	fmt.Println("Creating jobs...")
	for i := 0; i < 10000; i++ {
		_, err := client.CreateJob(context.Background(), "benchmark", model.CreateJobArgs{
			Expr: "@every 1ms",
		})
		if err != nil {
			b.Fatal(err)
		}
	}

	fmt.Println("CREATED ALL JOBS!")

	ctx, stop := context.WithCancel(context.Background())
	defer stop()

	fmt.Println("Start benchmark...")
	jobs := client.Fetch(ctx, "benchmark")
	for i := 0; i < b.N; i++ {
		job := <-jobs
		job.Commit()
	}
}
