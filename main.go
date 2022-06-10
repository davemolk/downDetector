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
	URL     string
	Success int
	Try     int
	Err     []error
}

// makeRequest makes a GET request to a specified URL, tracking the success, current
// number of attempts, and any errors.
func makeRequest(client *http.Client, job Job, jobs chan Job) error {
	req, err := http.NewRequest("GET", job.URL, nil)
	if err != nil {
		return fmt.Errorf("unable to create request for %s: %v", job.URL, err)
	}

	uAgent := randomUA()

	req.Header.Set("User-Agent", uAgent)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request unsuccessful for %s: %v", job.URL, err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		job.Try++
		jobs <- job
	} else {
		job.Success++
		job.Try++
		jobs <- job
	}
	return nil
}

// worker handles the makeRequest goroutines, their results, and any errors that occur
func worker(attempts *int, jobs chan Job, results chan<- Job, client *http.Client) {
	for job := range jobs {
		if job.Try >= *attempts {
			results <- job
			continue
		}
		internalError := make(chan error, 1)
		go func() {
			internalError <- makeRequest(client, job, jobs)
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

// randomUA returns a user agent randomly drawn from six possibilities.
func randomUA() string {
	userAgents := getUA()
	r := rand.New(rand.NewSource(time.Now().Unix()))
	rando := r.Intn(len(userAgents))

	return userAgents[rando]
}

// getUA returns a string slice of six user agents.
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

// readLines converts the contents of an input text file to a string slice.
func readLines(r io.Reader) ([]string, error) {
	var lines []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// makeClient takes in a flag-specified timeout and returns an *http.Client.
func makeClient(timeout int) *http.Client {
	return &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// getUrls takes in the name of an input file and returns a string slice of its contents (and any errors)
func getUrls(inputFile string) ([]string, error) {
	f, err := os.Open(inputFile)
	if err != nil {
		return []string{}, fmt.Errorf("unable to open input file: %v", err)
	}
	defer f.Close()

	lines, err := readLines(f)
	if err != nil {
		return []string{}, fmt.Errorf("unable to read input file: %v", err)
	}

	return lines, nil
}

func main() {
	url := flag.String("url", "", "url for site check")
	inputFile := flag.String("input", "", "use a file with urls")
	attempts := flag.Int("attempts", 3, "number of attempts per website")
	timeout := flag.Int("timeout", 5, "timeout for site check")
	flag.Parse()

	start := time.Now()

	client := makeClient(*timeout)

	var urls []string

	if *url != "" {
		urls = append(urls, *url)
	}

	if *inputFile != "" {
		fileUrls, err := getUrls(*inputFile)
		if err != nil {
			log.Fatalf("unable to read file: %v", err)
		}
		urls = fileUrls
	}

	fmt.Println("Probing:", urls)

	numJobs := len(urls)

	jobs := make(chan Job, numJobs)
	results := make(chan Job, numJobs)

	for w := 1; w <= 3; w++ {
		go worker(attempts, jobs, results, client)
	}

	for _, url := range urls {
		jobs <- Job{
			URL: url,
		}
	}

	for a := 1; a <= numJobs; a++ {
		job := <-results
		fmt.Printf("\nRESULTS: %s", job.URL)
		fmt.Printf("\nsite probe was successful %d out of %d attempts\n", job.Success, *attempts)
		if job.Err != nil {
			fmt.Printf("\nbut had the following error(s): %v\n", job.Err)
		}
	}

	close(jobs)
	close(results)

	fmt.Printf("\ntook: %f seconds\n", time.Since(start).Seconds())
}
