package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

func main() {
	t0 := time.Now()
	totalRequests := 0
	count := 0
	mu := sync.Mutex{}

	go func() {
		for {
			time.Sleep(100 * time.Millisecond)
			fmt.Printf("\x1bc")
			fmt.Println("In flight requests:", count, "Total:", totalRequests, "Elapsed time:", time.Now().Sub(t0))
		}
	}()

	http.HandleFunc("/counter", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		count++
		totalRequests++
		mu.Unlock()

		duration := time.Duration(rand.Intn(1)) * time.Second
		time.Sleep(duration)
		//fmt.Println("Slept for", duration, "seconds")
		w.Write([]byte("OK"))

		mu.Lock()
		count--
		mu.Unlock()
	})
	http.ListenAndServe(":8081", nil)
}
