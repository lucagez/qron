package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/lucagez/tinyq"
	"github.com/lucagez/tinyq/graph/model"
	"github.com/lucagez/tinyq/sqlc"
	"github.com/lucagez/tinyq/testutil"
	"github.com/stretchr/testify/assert"
)

type TestServer struct {
	handler http.HandlerFunc
}

func (t TestServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.handler(w, r)
}

func createTestServer(handler http.HandlerFunc) (string, func()) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatalln("error while starting listener:", err)
	}

	port := fmt.Sprintf(":%d", l.Addr().(*net.TCPAddr).Port)
	srv := &http.Server{Addr: port, Handler: TestServer{handler: handler}}

	go srv.Serve(l)

	time.Sleep(10 * time.Millisecond)

	return fmt.Sprintf("http://localhost%s", port), func() {
		srv.Shutdown(context.Background())
	}
}

// TODO: Refactor http executor to read from state

func TestHttpExecutor(t *testing.T) {
	pool, cleanup := testutil.PG.CreateDb("http_test")
	defer cleanup()

	client, err := tinyq.NewClient(pool, tinyq.Config{
		PollInterval:  10 * time.Millisecond,
		FlushInterval: 10 * time.Millisecond,
		ResetInterval: 10 * time.Millisecond,
		MaxInFlight:   5,
	})
	assert.Nil(t, err)

	t.Run("Should mutate job properties", func(t *testing.T) {
		baseUrl, stop := createTestServer(func(w http.ResponseWriter, r *http.Request) {
			var request sqlc.TinyJob
			defer r.Body.Close()
			err := json.NewDecoder(r.Body).Decode(&request)
			if err != nil {
				log.Fatal(err)
			}

			log.Println("request:", request)

			request.State = `{"count": 2}`
			request.Status = sqlc.TinyStatusSUCCESS
			request.Expr = "@after 300h"

			w.WriteHeader(200)
			json.NewEncoder(w).Encode(request)
		})
		defer stop()

		exe := NewHttpExecutor(5)

		meta, _ := json.Marshal(HttpConfig{
			Url:    baseUrl,
			Method: "POST",
		})
		m := string(meta)
		j, err := client.CreateJob(context.Background(), "http_test_1", model.CreateJobArgs{
			Expr:  "@after 10ms",
			State: `{"count": 1}`,
			Meta:  &m,
		})
		assert.Nil(t, err)

		ctx, stop := context.WithCancel(context.Background())

		go func() {
			<-time.After(100 * time.Millisecond)
			stop()
		}()

		for job := range client.Fetch(ctx, "http_test_1") {
			exe.Run(job)
		}

		updated, err := client.QueryJobByID(context.Background(), "http_test_1", j.ID)
		assert.Nil(t, err)

		assert.Equal(t, sqlc.TinyStatusSUCCESS, updated.Status)
		assert.Equal(t, `{"count": 2}`, updated.State)
		assert.Equal(t, "@after 300h", updated.Expr)
	})
}
