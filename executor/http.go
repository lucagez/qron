package executor

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/lucagez/tinyq"
	"github.com/lucagez/tinyq/sqlc"
)

type HttpExecutor struct {
	client  *http.Client
	limiter chan int
}

func NewHttpExecutor(maxConcurrency int) HttpExecutor {
	transport := &http.Transport{
		MaxIdleConns:        10, // global number of idle conns
		MaxIdleConnsPerHost: 5,  // subset of MaxIdleConns, per-host
		// declare a conn idle after 10 seconds. too low and conns are recycled too much, too high and conns aren't recycled enough
		IdleConnTimeout: 10 * time.Second,
		// DisableKeepAlives: true, // this means create a new connection per request. not recommended
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   60 * time.Second,
	}
	return HttpExecutor{
		client:  client,
		limiter: make(chan int, maxConcurrency),
	}
}

type HttpConfig struct {
	Url    string `json:"url,omitempty"`
	Method string `json:"method,omitempty"`
}

func (h HttpExecutor) Run(job tinyq.Job) {
	var config HttpConfig
	err := json.Unmarshal(job.Meta.Bytes, &config)
	if err != nil {
		job.Fail()
		return
	}

	// TODO: Check null readers do not cause issues
	// TODO: auth happens via e2e encrypted state
	// TODO: provide signature?
	payload, _ := json.Marshal(TinyDto{
		ID:              job.ID,
		Expr:            job.Expr,
		RunAt:           job.RunAt.Time,
		LastRunAt:       job.LastRunAt.Time,
		StartAt:         job.StartAt,
		ExecutionAmount: job.ExecutionAmount,
		Name:            job.Name.String,
		Meta:            string(job.Meta.Bytes),
		Timeout:         job.Timeout,
		Status:          job.Status,
		State:           job.State,
		Executor:        job.Executor,
	})
	req, err := http.NewRequest(config.Method, config.Url, bytes.NewReader(payload))
	if err != nil {
		job.Fail()
		return
	}

	req.Header.Add("content-type", "application/json")

	h.limiter <- 0

	res, err := h.client.Do(req)
	if err != nil {
		log.Println("http error:", err)
		job.Fail()
		return
	}
	defer res.Body.Close()

	<-h.limiter

	var execRes TinyDto
	err = json.NewDecoder(res.Body).Decode(&execRes)
	if err != nil {
		// TODO: In case body arrives but it's null
		// it should just not update job and NOT retrun an error
		log.Println("invalid response payload:", err)

		// TODO: Handle errors and automatic retries
		job.Fail()
		return
	}

	if execRes.Expr != "" {
		job.Expr = execRes.Expr
	}

	if execRes.State != "" {
		job.State = execRes.State
	}

	switch execRes.Status {
	case sqlc.TinyStatusSUCCESS:
		job.Commit()
	case sqlc.TinyStatusREADY:
		job.Retry()
	case sqlc.TinyStatusFAILURE:
		job.Fail()
	default:
		job.Commit()
	}
}
