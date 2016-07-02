package main

import (
	"bufio"
	"errors"
	"flag"
	"log"
	"net/url"
	"os"
	"sync"
	"time"
)

// URLs represents the list of URL to test
var URLs = []string{}

var nbOfClients int
var nbOfRequests int
var avgMillisecondsToWait int
var fileName string

func init() {
	flag.IntVar(&nbOfClients, "clients", 10, "number of clients making requests")
	flag.IntVar(&nbOfRequests, "requests", 10, "number of requests to be made by each clients")
	flag.IntVar(&avgMillisecondsToWait, "wait", 1000, "milliseconds to wait between each requests")
	flag.StringVar(&fileName, "urlSource", "./top-1m.txt", "filepath where to find the URLs")
	flag.Parse()
}

func getURLs() error {
	file, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		u := "http://" + scanner.Text()
		if _, err := url.Parse(u); err != nil {
			log.Printf("Invalid URL: %q", u)
			continue
		}

		URLs = append(URLs, u)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	if len(URLs) == 0 {
		return errors.New("no URL found")
	}

	return nil
}

func main() {
	if err := getURLs(); err != nil {
		log.Printf("Error while getting the URLs: %q", err)
		return
	}

	stats := newStats()
	var wg sync.WaitGroup
	for i := 0; i < nbOfClients; i++ {
		wg.Add(1)
		go work(i, stats, &wg)
	}
	wg.Wait()

	stats.Render()
}

func work(nb int, stats *Stats, wg *sync.WaitGroup) {
	defer wg.Done()
	for i := 0; i < nbOfRequests; i++ {
		url := findRandomURL()
		r, err := getURL(url)
		if err != nil {
			log.Printf("Worker#%d\t %d/%d - ERROR:  %s", nb, i, nbOfRequests, err)
			stats.addError(err)
			continue
		}
		log.Printf("Worker#%d\t %d/%d - %d ( %s - %s )", nb, i, nbOfRequests, r.status, url, r.duration)
		stats.addRequest(r)
		time.Sleep(time.Duration(avgMillisecondsToWait) * time.Millisecond)
	}
}
