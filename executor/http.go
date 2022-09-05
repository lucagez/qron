package executor

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
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
		Timeout:   30 * time.Second,
	}
	return HttpExecutor{
		client:  client,
		limiter: make(chan int, maxConcurrency),
	}
}

type HttpConfig struct {
	Url    string `json:"url"`
	Method string `json:"method"`
}

func (h HttpExecutor) Run(job Job) (Job, error) {
	var config HttpConfig
	err := json.Unmarshal([]byte(job.Config), &config)
	if err != nil {
		log.Panicln("error while decoding config payload:", err)
		return job, err
	}

	// TODO: Check null readers do not cause issues
	req, err := http.NewRequest(config.Method, config.Url, strings.NewReader(job.State))
	if err != nil {
		log.Println("error while assembling http request", err)
		return job, err
	}

	h.limiter <- 0

	res, err := h.client.Do(req)
	if err != nil {
		log.Println("http error:", err)
		return job, err
	}
	defer res.Body.Close()

	<-h.limiter

	var execRes Job
	err = json.NewDecoder(res.Body).Decode(&execRes)
	if err != nil {
		log.Println("invalid response payload:", err)
		return job, err
	}

	return execRes, nil
}
