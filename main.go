package main

import (
	"errors"
	"flag"
	"log"
	"math/rand"
	"time"
)

var (
	// URLs represents the list of URL to test
	URLs = []string{}
	// ErrInvalidTrafficType is returned if the traffic type is invalid
	ErrInvalidTrafficType = errors.New("Invalid traffic type")

	nbOfClients           int
	nbOfRequests          int
	avgMillisecondsToWait int
	fileName              string
	trafficType           string
	timeout               int
	seed                  int64
	followHttpRedirect    bool
)

func init() {
	// Parse the arguments
	flag.IntVar(&nbOfClients, "clients", 10, "number of clients making requests")
	flag.IntVar(&nbOfRequests, "requests", 10, "number of requests to be made by each clients")
	flag.IntVar(&avgMillisecondsToWait, "wait", 1000, "milliseconds to wait between each requests")
	flag.IntVar(&timeout, "timeout", 3, "HTTP timeout in seconds")
	flag.Int64Var(&seed, "seed", time.Now().UTC().UnixNano(), "seed for the random")
	flag.StringVar(&trafficType, "type", "http", "type of requests http/dns")
	flag.StringVar(&fileName, "urlSource", "", "optional filepath where to find the URLs")
	flag.BoolVar(&followHttpRedirect, "followRedirect", true, "follow http redirects or not")
	flag.Parse()

	log.SetFlags(0)
	log.Println("Random URLs using seed", seed)
	rand.Seed(seed)
}

func main() {
	// Create the TrafficGenerator
	trafficGenerator, err := NewTrafficGenerator(trafficType)
	if err != nil {
		log.Printf("Error while creating TrafficGenerator: %q", err)
		return
	}

	// Get the URLs
	if err := getURLs(); err != nil {
		log.Printf("Error while getting the URLs: %q", err)
		return
	}

	// Generate the traffic
	trafficGenerator.Generate()

	// Display the statistics
	trafficGenerator.DisplayStats()
}
