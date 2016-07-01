package main

import (
	"bufio"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

var URLs = []string{}

var nbOfWorkers = 50
var nbOfRequests = 10
var stats = newStats()
var wg sync.WaitGroup
var fileName = "./top-1m.csv"
var averageTimeToWait = time.Duration(rand.Intn(1000))

type Stats struct {
	sync.Mutex
	nbOfRequests int
	durations    DurationStats
	statusStats  map[string]int
}

func newStats() *Stats {
	return &Stats{
		statusStats: map[string]int{},
	}
}

type DurationStats struct {
	maxDuration   time.Duration
	minDuration   time.Duration
	totalDuration time.Duration
}

type Request struct {
	status   int
	duration time.Duration
}

type Worker struct {
	// done is channel used to stop the app
	done chan struct{}

	// wait group sync the goroutines launched by the app
	wg sync.WaitGroup
}

func getURLs() error {
	file, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		url := scanner.Text()
		URLs = append(URLs, "http://"+url)
	}

	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func (s *Stats) addError(err error) {
	s.Lock()
	s.nbOfRequests++
	if _, ok := err.(net.Error); ok {
		s.statusStats["neterr"] += 1
	} else {
		s.statusStats[err.Error()] += 1
	}
	s.Unlock()
}
func (s *Stats) addRequest(req *Request) {
	s.Lock()
	s.nbOfRequests++
	s.durations.totalDuration += req.duration
	if s.durations.maxDuration < req.duration {
		s.durations.maxDuration = req.duration
	}
	if s.durations.minDuration == 0 || s.durations.minDuration > req.duration {
		s.durations.minDuration = req.duration
	}
	s.statusStats[http.StatusText(req.status)] += 1

	s.Unlock()
}

func main() {
	getURLs()
	for i := 0; i < nbOfWorkers; i++ {
		wg.Add(1)
		go work(i)
	}
	wg.Wait()

	log.Printf("Number of requests : %d", stats.nbOfRequests)
	log.Printf("Max duration : %s", stats.durations.maxDuration)
	log.Printf("Min duration : %s", stats.durations.minDuration)
	log.Printf("Average duration : %s", stats.durations.totalDuration/time.Duration(stats.nbOfRequests))
	log.Printf("Status")
	for key, value := range stats.statusStats {
		log.Printf("%s\t -> %d", key, value)
	}
}

func findRandomURL() string {
	return URLs[rand.Intn(len(URLs))]
}

func getURL(url string) (*Request, error) {

	client := http.DefaultClient
	client.Timeout = 3 * time.Second

	t := time.Now()
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	dur := time.Since(t)

	r := Request{
		duration: dur,
		status:   resp.StatusCode,
	}
	return &r, nil
}

func work(nb int) {
	for i := 0; i < nbOfRequests; i++ {
		url := findRandomURL()
		r, err := getURL(url)
		if err != nil {
			log.Println("error - ", err)
			stats.addError(err)
			continue
		}
		log.Printf("%d\t= %d / %d = %d - %s - %s", nb, i, nbOfRequests, r.status, url, r.duration)
		stats.addRequest(r)
		time.Sleep(averageTimeToWait * time.Millisecond)
	}
	wg.Done()
}
