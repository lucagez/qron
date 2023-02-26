package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/lucagez/qron"
	"github.com/lucagez/qron/graph/model"
	"github.com/lucagez/qron/testutil"
)

func BenchmarkFetch(b *testing.B) {
	b.StopTimer()

	testutil.PG = testutil.NewPgFactory()
	defer testutil.PG.Teardown()

	pool, cleanup := testutil.PG.CreateDb("bench_client")
	defer cleanup()

	client, err := qron.NewClient(pool, qron.Config{
		PollInterval:  3 * time.Millisecond,
		FlushInterval: 6 * time.Millisecond,
		ResetInterval: 10 * time.Minute,
		MaxInFlight:   1000,
		MaxFlushSize:  1000,
	})
	if err != nil {
		b.Fatal(err)
	}
	defer client.Close()

	err = client.Migrate()
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < 10; i++ {
		fmt.Println("creating job batch:", i)
		var batch []model.CreateJobArgs
		for j := 0; j < 10000; j++ {
			batch = append(batch, model.CreateJobArgs{
				Expr: "@every 1ms",
			})
		}

		_, err = client.BatchCreateJobs(context.Background(), "benchmark-1", batch)
		if err != nil {
			b.Fatal(err)
		}

		_, err = client.BatchCreateJobs(context.Background(), "benchmark-2", batch)
		if err != nil {
			b.Fatal(err)
		}

		_, err = client.BatchCreateJobs(context.Background(), "benchmark-3", batch)
		if err != nil {
			b.Fatal(err)
		}
	}

	fmt.Println("created all batches!")

	ctx, stop := context.WithCancel(context.Background())
	fmt.Println("start benchmark...")

	jobs1 := client.Fetch(ctx, "benchmark-1")
	jobs2 := client.Fetch(ctx, "benchmark-2")
	jobs3 := client.Fetch(ctx, "benchmark-3")

	b.ResetTimer()
	t0 := time.Now()
	counter := 0

	go func() {
		<-time.After(10 * time.Second)
		stop()
	}()

loop:
	for {
		select {
		case job := <-jobs1:
			counter++
			job.Commit()
		case job := <-jobs2:
			counter++
			job.Commit()
		case job := <-jobs3:
			counter++
			job.Commit()
		case <-ctx.Done():
			break loop
		}
	}

	fmt.Println("processed", counter, "jobs in", time.Since(t0))
	fmt.Println("average throughput per second:", float64(counter)/time.Since(t0).Seconds())
}
