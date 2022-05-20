package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"
)

type Job struct {
	URL string
	Success int
	Try int
	Err []error
}

func makeRequest(client *http.Client, job Job, ua string, jobs chan Job) error {
	req, err := http.NewRequest("GET", job.URL, nil)
	if err != nil {
		return fmt.Errorf("unable to create request: %w", err)
	}
	
	req.Header.Set("User-Agent", ua)

	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request unsuccessful: %w", err)
	}
	
	defer res.Body.Close()

	if res.StatusCode != 200 {
		log.Println("site is down", res.StatusCode)
		job.Try++
		jobs <- job
	} else {
		log.Println("site is up", res.Status)
		job.Success++
		job.Try++
		jobs <- job
	}
	return nil
}

func worker(attempts *int, jobs chan Job, results chan<-Job, client *http.Client, ua string){
	for job := range jobs{
		if job.Try >= *attempts {
			results <- job
			continue
		}
		internalError := make(chan error, 1)
		go func() {
			internalError <- makeRequest(client, job, ua, jobs)
		}()
		err := <-internalError
		if err != nil {
			job.Err = append(job.Err, err)
			job.Try++
			jobs <- job
		} else {
			job.Err = nil
			job.Try++
		}
	}
}

func randomUA() []string {
	userAgents := getUA()
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

func getUA() []string {
	return []string{
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
}


func readLines(r io.Reader) ([]string, error) {
	var lines []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func main() {
	url := flag.String("url", "", "url for site check")
	inputFile := flag.String("input", "", "use a file with urls")
	attempts := flag.Int("attempts", 3, "number of attempts per website")
	timeout := flag.Int("timeout", 5, "timeout for site check")
	flag.Parse()
	
	start := time.Now()

	client := &http.Client{
		Timeout: time.Duration(*timeout) * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	var urls []string
	
	if *url != "" {
		urls = append(urls, *url)
	} 
	
	if *inputFile != "" {
		f, err := os.Open(*inputFile)
		if err != nil {
			log.Fatal("Couldn't open input file: %w", err)
		}
		defer f.Close()

		lines, err := readLines(f)
		if err != nil {
			log.Fatal("Couldn't read input file: %w", err)
		}
		urls = append(urls, lines...)
	}
	
	fmt.Println("URLS", urls)

	numJobs := len(urls)

	jobs := make(chan Job, numJobs)
	results := make(chan Job, numJobs)

	userAgents := randomUA()

	for w := 1; w <= 3; w++ {
		go worker(attempts, jobs, results, client, userAgents[w-1])
	}
	
	for _, url := range urls{
		jobs <- Job{
			URL: url,
		}
	}

	for a := 1; a <= numJobs; a++ {
		job := <- results
		fmt.Printf("\nRESULTS: %s", job.URL)
		fmt.Printf("\nsite probe for was successful %d out of %d attempts\n", job.Success, *attempts)
		if job.Err != nil {
			fmt.Println("\nbut had the following error(s): %w\n", job.Err)
		}
	}

	close(jobs)
	close(results)

	fmt.Printf("\ntook: %f seconds\n", time.Since(start).Seconds())
}