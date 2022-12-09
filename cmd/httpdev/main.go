package main

import (
	"fmt"
	"log"
	"net"
	"net/http"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
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

	port := freePort()
	databaseUrl := fmt.Sprintf("postgres://tiny:tiny@localhost:%d/tiny", port)
	postgres := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().
			Username("tiny").
			Password("tiny").
			Database("tiny").
			Port(uint32(port)),
	)
	err := postgres.Start()
	if err != nil {
		log.Fatal(err)
	}
	defer postgres.Stop()

	client, err := tinyq.NewClient(tinyq.Config{
		Dsn: databaseUrl,
	})
	if err != nil {
		log.Fatal(err)
	}

	err = client.Migrate()
	if err != nil {
		log.Fatal(err)
	}

	httpJobs, cancel := client.Fetch("http")
	defer cancel()

	httpExecutor := executor.NewHttpExecutor(100)

	go func() {
		for {
			job := <-httpJobs
			httpExecutor.Run(job)
			log.Println("executoing http job", job)
		}
	}()

	log.Println("listening on: 9876")
	log.Fatal(http.ListenAndServe(":9876", client.Handler()))
}
