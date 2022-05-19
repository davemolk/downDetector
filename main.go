package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"
)

type Job struct {
	URL string
}

func worker(id int, jobs <-chan Job, results chan<-int, client *http.Client, ua string){
	for j := range jobs{
		req, err := http.NewRequest("GET", j.URL, nil)
		log.Println("id, url:", id, j.URL)
		if err != nil {
			log.Fatal("Unable to create request", err)
		}
		
		req.Header.Set("User-Agent", ua)

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

func randomUA() []string {
	userAgents := []string{
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.127 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/603.3.8 (KHTML, like Gecko)",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_6) AppleWebKit/601.7.7 (KHTML, like Gecko) Version/9.1.2 Safari/601.7.7",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.127 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/101.0.4951.54 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.127 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:99.0) Gecko/20100101 Firefox/99.0",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:100.0) Gecko/20100101 Firefox/100.0",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.88 Safari/537.36",
	}

	r := rand.New(rand.NewSource(time.Now().Unix()))
	for i := 0; i < len(userAgents); i++ {
		j := r.Intn(len(userAgents))

		ua1 := userAgents[i]
		ua2 := userAgents[j]
		userAgents[i] = ua2
		userAgents[j] = ua1
	}

	return userAgents
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

	userAgents := randomUA()

	for w := 1; w <= 3; w++ {
		go worker(w, jobs, results, client, userAgents[w-1])
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