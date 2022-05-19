package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"
)

type Job struct {
	URL string
}

func worker(id int, jobs <-chan Job, results chan<-int, client *http.Client){
	for j := range jobs{
		req, err := http.NewRequest("GET", j.URL, nil)
		log.Println("id, url:", id, j.URL)
		if err != nil {
			log.Fatal("Unable to create request", err)
		}
		
		res, err := client.Do(req)
		if err != nil {
			log.Println("request unsuccessful so site probably down", id, err)
			results <- 0
			return
		}
		
		if res.StatusCode != 200 {
			log.Println("site is down", id, res.StatusCode)
			results <- 0
		} else {
			log.Println("site is up", id, res.Status)
			results <- 1
		}
	}
}

func main() {
	url := flag.String("url", "https://httpbin.org/status/200", "url for site check")
	attempts := flag.Int("attempts", 3, "number of attempts per website")
	timeout := flag.Int("timeout", 5, "timeout for site check")
	flag.Parse()

	client := &http.Client{
		Timeout: time.Duration(*timeout) * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	jobs := make(chan Job, *attempts)
	results := make(chan int, *attempts)

	for w := 1; w <= 3; w++ {
		go worker(w, jobs, results, client)
	}

	for j := 1; j <= *attempts; j++ {
		jobs <- Job{
			URL: *url,
		}
	}
	close(jobs)

	healthScore := 0

	for a := 1; a <= *attempts; a++ {
		curr := <- results
		healthScore += curr
		fmt.Println("here is the healthScore:", healthScore)
	}

	fmt.Printf("Site probe was successful %d out of %d attempts", healthScore, *attempts)
}