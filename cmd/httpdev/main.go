package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/briandowns/spinner"
	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/lucagez/qron"
	"github.com/lucagez/qron/executor"
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

func init() {
	// Load .env file. Do not fail if it does not exist.
	godotenv.Load()
}

func main() {
	t0 := time.Now()

	s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
	s.Suffix = " Initializing postgres 🐘\n"
	s.Start()

	port := freePort()
	databaseUrl := os.Getenv("DATABASE_URL")
	var postgres *embeddedpostgres.EmbeddedPostgres

	if databaseUrl == "" {
		binariesPath := path.Join(os.TempDir(), ".pg-binaries")
		dataPath := path.Join(os.TempDir(), ".pg/data")
		runtimePath := path.Join(os.TempDir(), ".pg")
		databaseUrl = fmt.Sprintf("postgres://tiny:tiny@localhost:%d/tiny", port)
		postgres = embeddedpostgres.NewDatabase(
			embeddedpostgres.DefaultConfig().
				Username("tiny").
				Password("tiny").
				Database("tiny").
				Port(uint32(port)).
				Logger(os.Stdout).
				Locale("en_US").
				BinariesPath(binariesPath).
				DataPath(dataPath).
				RuntimePath(runtimePath),
		)
		err := postgres.Start()
		if err != nil {
			log.Fatal(err)
		}
	}

	config, err := pgxpool.ParseConfig(databaseUrl)
	if err != nil {
		log.Fatal("unable to parse config:", err)
	}

	db, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		log.Fatal("failed to connect to postgres:", err)
	}

	s.Stop()

	client, err := qron.NewClient(db, qron.Config{
		PollInterval:  1 * time.Second,
		FlushInterval: 1 * time.Second,
		ResetInterval: 10 * time.Second,
	})
	if err != nil {
		log.Fatal("failed to create qron client:", err)
	}

	err = client.Migrate()
	if err != nil {
		log.Fatal("failed to migrate db:", err)
	}

	ctx, stop := context.WithCancel(context.Background())
	httpJobs := client.Fetch(ctx, "http")

	httpExecutor := executor.NewHttpExecutor(100)

	go func() {
		for job := range httpJobs {
			go httpExecutor.Run(job)
		}
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGHUP)

	log.Println("listening on: 9876", "started in:", time.Since(t0))

	go func() {
		err := http.ListenAndServe(":9876", client.Handler())
		if err != nil {
			log.Fatal("failed to start qron deamon:", err)
		}
	}()

	<-sigs
	stop()
	client.Close()
	if postgres != nil {
		postgres.Stop()
	}

	log.Println("exiting 👋")
}
