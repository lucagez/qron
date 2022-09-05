package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"log"
	"net"
	"net/http"
	"testing"
	"time"
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

func TestHttpExecutor(t *testing.T) {

	t.Run("Should perform http requests", func(t *testing.T) {
		baseUrl, stop := createTestServer(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			json.NewEncoder(w).Encode(Job{Id: 1})
		})
		defer stop()

		exe := NewHttpExecutor(5)

		conf, _ := json.Marshal(HttpConfig{
			Url:    baseUrl,
			Method: "GET",
		})
		job := Job{
			Status: PENDING,
			Config: string(conf),
			State:  "",
		}

		updated, err := exe.Run(job)

		assert.Nil(t, err)
		assert.Equal(t, 1, updated.Id)
	})

	t.Run("Should mutate job properties", func(t *testing.T) {
		baseUrl, stop := createTestServer(func(w http.ResponseWriter, r *http.Request) {
			var request Job
			defer r.Body.Close()
			json.NewDecoder(r.Body).Decode(&request)

			request.State = `{"hello":"world"}`
			request.Status = SUCCESS
			// TODO: Should add validation to make impossible
			// to pass invalid qron types
			request.RunAt = "bananas"

			w.WriteHeader(200)
			json.NewEncoder(w).Encode(request)
		})
		defer stop()

		exe := NewHttpExecutor(5)

		conf, _ := json.Marshal(HttpConfig{
			Url:    baseUrl,
			Method: "GET",
		})
		job := Job{
			Status: PENDING,
			Config: string(conf),
			State:  "",
			RunAt:  "@yearly",
		}

		updated, err := exe.Run(job)

		assert.Nil(t, err)
		assert.Equal(t, SUCCESS, updated.Status)
		assert.Equal(t, `{"hello":"world"}`, updated.State)
		assert.Equal(t, "bananas", updated.RunAt)
	})
}
