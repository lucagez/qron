package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"runtime/pprof"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lucagez/tinyq"
	"github.com/lucagez/tinyq/graph/model"
	"github.com/pyroscope-io/client/pyroscope"
)

func main() {
	pool, err := pgxpool.New(context.Background(), "postgres://postgres:password@localhost:5435/postgres")
	if err != nil {
		log.Fatal(err)
	}

	tiny, err := tinyq.NewClient(pool, tinyq.Config{
		FlushInterval: 1 * time.Second,
		PollInterval:  1 * time.Second,
		MaxInFlight:   100,
	})
	if err != nil {
		log.Fatal(err)
	}

	transport := &http.Transport{
		MaxIdleConns:        10, // global number of idle conns
		MaxIdleConnsPerHost: 5,  // subset of MaxIdleConns, per-host
		// declare a conn idle after 10 seconds. too low and conns are recycled too much, too high and conns aren't recycled enough
		IdleConnTimeout: 10 * time.Second,
		// DisableKeepAlives: true, // this means create a new connection per request. not recommended
	}
	httpClient := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	// Profiling
	_, err = pyroscope.Start(pyroscope.Config{
		ApplicationName: "simple.golang.app",
		ServerAddress:   "http://localhost:4040",
		Logger:          pyroscope.StandardLogger,
	})
	if err != nil {
		log.Fatalf("error starting pyroscope profiler: %v", err)
	}

	go func() {
		for {
			time.Sleep(100 * time.Millisecond)
			fmt.Println("creating jobs")

			for i := 0; i < 1000; i++ {
				_, err := tiny.CreateJob("admin", model.CreateJobArgs{
					Expr:  "@after 1s",
					State: `http://localhost:8081/counter`,
				})
				if err != nil {
					log.Fatal(err)
				}
			}
		}
	}()

	go func() {
		pyroscope.TagWrapper(context.Background(), pyroscope.Labels("fetching", "jobs"), func(c context.Context) {
			jobs, _ := tiny.Fetch("admin")
			fmt.Println("============ FETCHING =============")

			for job := range jobs {
				fmt.Println("fetching job:", job.ID)
				// profiler
				go func(j tinyq.Job) {
					pprof.Do(c, pprof.Labels("process", "http"), func(ctx context.Context) {
						_, err := httpClient.Get(j.State)
						if err != nil {
							log.Println("failed to execute:", j.ID, err)
							j.Fail()
							return
						}
						j.Commit()
					})
				}(job)
			}
		})
	}()

	router := chi.NewRouter()

	router.Handle("/*", tiny.Handler())

	fmt.Println("======= LISTENING ON :1234 =======")
	http.ListenAndServe(":1234", router)
}
