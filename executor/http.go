package executor

import (
	"encoding/json"
	"github.com/lucagez/tinyq"
	"log"
	"net/http"
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

// TODO: This is leading to a circular dependency
func (h HttpExecutor) Run(job *tinyq.Job) error {
	var config HttpConfig
	// TODO: Should receive job as input
	//err := json.Unmarshal([]byte(job.Config), &config)
	err := json.Unmarshal([]byte(`{"method":"GET", "url":"https://www.google.com"}`), &config)
	if err != nil {
		log.Panicln("error while decoding config payload:", err)
		return err
	}

	// TODO: should pass job state. How to pass encrypted? How to set headers?
	req, err := http.NewRequest(config.Method, config.Url, nil)
	if err != nil {
		log.Println("error while assembling http request", err)
		return err
	}

	h.limiter <- 0

	res, err := h.client.Do(req)
	if err != nil {
		log.Println("http error:", err)
		return err
	}
	defer res.Body.Close()

	<-h.limiter

	// TODO: Should encode response
	//var execRes tinyq.ExecResponse
	//err = json.NewDecoder(res.Body).Decode(&execRes)
	//if err != nil {
	//	log.Println("invalid response payload:", err)
	//	// TODO: What to do here? Job should be retried until max attempts?
	//	return err
	//}

	return nil
}
