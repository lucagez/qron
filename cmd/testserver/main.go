package main

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/lucagez/tinyq"
	"github.com/lucagez/tinyq/executor"
	"github.com/lucagez/tinyq/sqlc"
	"github.com/pyroscope-io/client/pyroscope"
	"log"
	_ "net/http/pprof"
	"runtime/pprof"
	"time"
)

func main() {
	db, err := pgxpool.Connect(context.Background(), "postgres://postgres:password@localhost:5435/postgres")
	if err != nil {
		log.Fatalln(err)
	}
	tiny := tinyq.NewTinyQ(tinyq.Config{
		Db:            db,
		FlushInterval: 1 * time.Second,
		PollInterval:  1 * time.Second,
		MaxInFlight:   1000,
		Executors: map[string]tinyq.Executor{
			"HTTP": executor.NewHttpExecutor(50),
		},
	})

	// Profiling
	_, err = pyroscope.Start(pyroscope.Config{
		ApplicationName: "simple.golang.app",
		ServerAddress:   "http://localhost:4040",
		Logger:          pyroscope.StandardLogger,
	})
	if err != nil {
		log.Fatalf("error starting pyroscope profiler: %v", err)
	}

	pyroscope.TagWrapper(context.Background(), pyroscope.Labels("fetching", "jobs"), func(c context.Context) {
		go func() {
			batchN := 0
			for {
				batchN++

				t0 := time.Now()
				fmt.Println("BATCH", batchN)
				result, err := tiny.Fetch()
				if result != nil {
					fmt.Println("Fetched:", len(result))
				}
				if err != nil {
					fmt.Println("Error:", err)
				}

				for _, job := range result {
					// profiler
					go func(j sqlc.TinyJob) {
						pprof.Do(c, pprof.Labels("process", "http"), func(ctx context.Context) {
							tiny.Process(ctx, j)
						})
					}(job)
				}
				fmt.Println("Elapsed:", time.Now().Sub(t0))
			}
		}()
	})

	tiny.Listen()
}
