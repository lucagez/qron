package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lucagez/tinyq"
	"github.com/lucagez/tinyq/executor"
)

func freePort() int {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		log.Fatal(err)
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

type Tracer struct{}

func (tracer *Tracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	// log.Println("[POSTGRES]", data.SQL, data.Args)
	return ctx
}

func (tracer *Tracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
}

func main() {
	log.Println("Initializing postgres 🐘")

	// port := freePort()
	// log.Println("free port:", port)

	// databaseUrl := fmt.Sprintf("postgres://tiny:tiny@localhost:%d/tiny", port)
	// postgres := embeddedpostgres.NewDatabase(
	// 	embeddedpostgres.DefaultConfig().
	// 		Username("tiny").
	// 		Password("tiny").
	// 		Database("tiny").
	// 		Port(uint32(port)).
	// 		Logger(os.Stdout).
	// 		Locale("en_US").
	// 		BinariesPath("/tmp/.pg").
	// 		DataPath("/tmp/.pg/data").
	// 		RuntimePath("/tmp/.pg"),
	// )
	// err := postgres.Start()
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer postgres.Stop()

	config, err := pgxpool.ParseConfig("postgres://postgres:password@localhost:5435/postgres")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to parse config: %v\n", err)
		os.Exit(1)
	}
	config.ConnConfig.Tracer = &Tracer{}

	db, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		log.Fatal("FAILED TO CONNECT:", err)
	}

	client, err := tinyq.NewClient(db, tinyq.Config{
		// PollInterval:  5 * time.Second,
		// FlushInterval: 5 * time.Second,
	})
	if err != nil {
		log.Fatal("FAILED TO CREATE CLIENT:", err)
	}

	err = client.Migrate()
	if err != nil {
		log.Fatal("FAILED MIGRATE:", err)
	}

	ctx, stop := context.WithCancel(context.Background())
	defer stop()

	httpJobs := client.Fetch(ctx, "http")

	httpExecutor := executor.NewHttpExecutor(100)

	go func() {
		for {
			select {
			case job := <-httpJobs:
				go httpExecutor.Run(job)
			}
		}
	}()

	log.Println("listening on: 9876")
	log.Fatal(http.ListenAndServe(":9876", client.Handler()))
}
