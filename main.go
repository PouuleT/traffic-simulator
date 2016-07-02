package main

import (
	"bufio"
	"flag"
	"log"
	"math/rand"
	"net"
	"net/http"
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

var stats = newStats()
var fileName string

func init() {
	flag.IntVar(&nbOfClients, "clients", 10, "number of clients making requests")
	flag.IntVar(&nbOfRequests, "requests", 10, "number of requests to be made by each clients")
	flag.IntVar(&avgMillisecondsToWait, "wait", 1000, "milliseconds to wait between each requests")
	flag.StringVar(&fileName, "urlSource", "./top-1m.txt", "filepath where to find the URLs")
	flag.Parse()
}

// Stats represents the stats of the requests
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

// DurationStats represents statistics of durations
type DurationStats struct {
	maxDuration   time.Duration
	minDuration   time.Duration
	totalDuration time.Duration
}

// Request represents a requsest reponse, with the return code and the duration
type Request struct {
	status   int
	duration time.Duration
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
	defer s.Unlock()

	s.nbOfRequests++
	var errName string
	switch e := err.(type) {
	case *net.DNSError:
		errName = "DNS lookup error"
	case *net.DNSConfigError:
		errName = "DNS config error"
	case *net.AddrError:
		errName = "Addr Error"
	case *net.OpError:
		errName = "Op Error"
	case *url.Error:
		if e.Timeout() {
			errName = "URL Timeout"
		} else {
			errName = "Url Error"
		}
	case net.Error:
		errName = "Net Error"
	default:
		errName = err.Error()
	}
	s.statusStats[errName]++
}
func (s *Stats) addRequest(req *Request) {
	s.Lock()
	defer s.Unlock()
	s.nbOfRequests++
	s.durations.totalDuration += req.duration
	if s.durations.maxDuration < req.duration {
		s.durations.maxDuration = req.duration
	}
	if s.durations.minDuration == 0 || s.durations.minDuration > req.duration {
		s.durations.minDuration = req.duration
	}
	s.statusStats[http.StatusText(req.status)]++
}

func main() {
	if err := getURLs(); err != nil {
		log.Printf("Error while getting the URLs: %q", err)
		return
	}

	var wg sync.WaitGroup
	for i := 0; i < nbOfClients; i++ {
		wg.Add(1)
		go work(i, &wg)
	}
	wg.Wait()

	log.Println("********************************************************")
	log.Printf("Number of requests : %d", stats.nbOfRequests)
	log.Printf("Max duration : %s", stats.durations.maxDuration)
	log.Printf("Min duration : %s", stats.durations.minDuration)
	log.Printf("Average duration : %s", stats.durations.totalDuration/time.Duration(stats.nbOfRequests))
	log.Println("********************************************************")
	log.Printf("Statuses :")
	for key, value := range stats.statusStats {
		log.Printf("\t%20s\t -> %d", key, value)
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

func work(nb int, wg *sync.WaitGroup) {
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
