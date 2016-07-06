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
var trafficType string
var timeout int

func init() {
	flag.IntVar(&nbOfClients, "clients", 10, "number of clients making requests")
	flag.IntVar(&nbOfRequests, "requests", 10, "number of requests to be made by each clients")
	flag.IntVar(&avgMillisecondsToWait, "wait", 1000, "milliseconds to wait between each requests")
	flag.IntVar(&timeout, "timeout", 3, "HTTP timeout in seconds")
	flag.StringVar(&trafficType, "type", "http", "type of requests http/dns")
	flag.StringVar(&fileName, "urlSource", "./top-1m.txt", "filepath where to find the URLs")
	flag.Parse()
}

// getURLs will open the given file and read it to get a list of URLs
func getURLs() error {
	file, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		u := scanner.Text()
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
	var trafficFunc func(string) *Request
	switch trafficType {
	case "http":
		trafficFunc = getURL
	case "dns":
		trafficFunc = lookupURL
	default:
		log.Printf("%s trafficType is not handled, only http and dns are", trafficType)
		os.Exit(0)
	}

	if err := getURLs(); err != nil {
		log.Printf("Error while getting the URLs: %q", err)
		return
	}

	stats := newStats()
	var wg sync.WaitGroup
	for i := 1; i <= nbOfClients; i++ {
		wg.Add(1)
		go work(i, trafficFunc, stats, &wg)
	}
	wg.Wait()

	stats.Render()
}

// work represents a worker that will run nbOfRequests requests
func work(nb int, trafficFunc func(string) *Request, stats *Stats, wg *sync.WaitGroup) {
	defer wg.Done()
	start := time.Now()

	workerFmt := fmt.Sprintf("worker#%%0%dd", getPadding(nbOfClients))
	counterFmt := fmt.Sprintf(" - %%0%dd/%%d ", getPadding(nbOfRequests))

	prefix := fmt.Sprintf(workerFmt, nb)
	logger := log.New(os.Stdout, prefix, 0)
	for i := 1; i <= nbOfRequests; i++ {
		logger.SetPrefix(prefix + fmt.Sprintf(counterFmt, i, nbOfRequests))
		url := findRandomURL()
		r := trafficFunc(url)
		if r.err != nil {
			logger.Printf("| %s | %12s | %s", red("ERR"), r.duration, r.err)
			stats.addError(r)
			continue
		}

		logger.Print(r.Log())

		stats.addRequest(r)
		time.Sleep(time.Duration(avgMillisecondsToWait) * time.Millisecond)
	}
	stats.durations.execDuration = time.Since(start)
}

// getPadding returns the padding size of the int given
func getPadding(nb int) int {
	// Get the padding size : floor(log10(nb)) + 1
	return int(math.Log10(float64(nb))) + 1
}
