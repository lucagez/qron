package main

import (
	"context"
	"log"
	"net"
	"net/http"

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
	db, err := pgxpool.New(context.Background(), "postgres://postgres:password@localhost:5435/postgres")
	if err != nil {
		log.Fatal(err)
	}

	client, err := tinyq.NewClient(db, tinyq.Config{})
	if err != nil {
		log.Fatal(err)
	}

	err = client.Migrate()
	if err != nil {
		log.Fatal(err)
	}

	httpJobs, cancelHttp := client.Fetch(context.Background(), "http")
	defer cancelHttp()

	dockerJobs, cancelDocker := client.Fetch(context.Background(), "docker")
	defer cancelDocker()

	httpExecutor := executor.NewHttpExecutor(100)
	dockerExecutor := executor.NewDockerExecutor()

	go func() {
		for {
			select {
			case job := <-httpJobs:
				httpExecutor.Run(job)
			case job := <-dockerJobs:
				go dockerExecutor.Run(job)
			}
		}
	}()

	log.Println("listening on: 9876")
	log.Fatal(http.ListenAndServe(":9876", client.Handler()))
}
