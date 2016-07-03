package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log"
	"math"
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
	for i := 1; i <= nbOfClients; i++ {
		wg.Add(1)
		go work(i, stats, &wg)
	}
	wg.Wait()

	stats.Render()
}

func work(nb int, stats *Stats, wg *sync.WaitGroup) {
	defer wg.Done()

	// Get the padding size : floor(log10(nbOfClients)) + 1
	workerFmt := fmt.Sprintf("worker#%%0%dd", int(math.Log10(float64(nbOfClients))+1))
	counterDecimals := int(math.Log10(float64(nbOfRequests))) + 1
	counterFmt := fmt.Sprintf(" - %%0%dd/%%d ", counterDecimals)

	prefix := fmt.Sprintf(workerFmt, nb)
	logger := log.New(os.Stdout, prefix, log.LstdFlags)
	for i := 1; i <= nbOfRequests; i++ {
		logger.SetPrefix(prefix + fmt.Sprintf(counterFmt, i, nbOfRequests))
		url := findRandomURL()
		r, err := getURL(url)
		if err != nil {
			logger.Printf("| ERR | %12s | %s", "", err)
			stats.addError(err)
			continue
		}

		logger.Printf("| %d | %12s | GET %s", r.status, r.duration, url)
		stats.addRequest(r)
		time.Sleep(time.Duration(avgMillisecondsToWait) * time.Millisecond)
	}
}
