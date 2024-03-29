package executor

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/lucagez/qron"
	"github.com/lucagez/qron/sqlc"
)

type HttpExecutor struct {
	client  *http.Client
	limiter chan int
	Signer  func(job qron.Job, r *http.Request) error
}

type Signer func(job qron.Job, r *http.Request) error

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
		Signer: func(job qron.Job, r *http.Request) error {
			return nil
		},
	}
}

type HttpConfig struct {
	Url    string `json:"url,omitempty"`
	Method string `json:"method,omitempty"`
}

func (h HttpExecutor) Run(job qron.Job) {
	var config HttpConfig
	err := json.Unmarshal(job.Meta, &config)
	if err != nil {
		job.Fail()
		return
	}

	payload, _ := json.Marshal(job)
	req, err := http.NewRequest(config.Method, config.Url, bytes.NewReader(payload))
	if err != nil {
		log.Println("request creation error:", err)
		job.Fail()
		return
	}

	req.Header.Add("content-type", "application/json")

	err = h.Signer(job, req)
	if err != nil {
		log.Println("signer error:", err)
		job.Fail()
		return
	}

	h.limiter <- 0

	res, err := h.client.Do(req)
	if err != nil {
		log.Println("http error:", err)
		job.Fail()
		return
	}
	defer res.Body.Close()

	<-h.limiter

	var execRes qron.Job
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
