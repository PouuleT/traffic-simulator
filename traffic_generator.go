package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"sync"
	"time"
)

// TrafficGenerator represents the traffic generation object
type TrafficGenerator struct {
	stats       Stats
	trafficFunc func(string) Request
	wg          sync.WaitGroup
}

// Worker represents a client making the requests
type Worker struct {
	id         int
	trafficGen *TrafficGenerator
}

var trafficMap = map[string]func(string) Request{
	"http": getURL,
	"dns":  lookupURL,
}

var statsMap = map[string]func() Stats{
	"http": newHTTPStats,
	"dns":  newDNSStats,
}

// NewTrafficGenerator will return a new TrafficGenerator object
func NewTrafficGenerator(trafficType string) (*TrafficGenerator, error) {
	tFunc, ok := trafficMap[trafficType]
	if !ok {
		return nil, ErrInvalidTrafficType
	}
	stats, err := newStats(trafficType)
	if err != nil {
		return nil, err
	}
	return &TrafficGenerator{
		trafficFunc: tFunc,
		stats:       stats,
	}, nil
}

// Generate generates traffic
func (trafficGen *TrafficGenerator) Generate() {
	for i := 1; i <= nbOfClients; i++ {
		trafficGen.wg.Add(1)
		go trafficGen.NewWorker(i).work()
	}
	trafficGen.wg.Wait()
}

// NewWorker creates a new worker for traffic generation
func (trafficGen *TrafficGenerator) NewWorker(i int) *Worker {
	return &Worker{
		id:         i,
		trafficGen: trafficGen,
	}
}

// DisplayStats renders the statistics of the traffic generation
func (trafficGen *TrafficGenerator) DisplayStats() {
	trafficGen.stats.Render()
}

func (w *Worker) work() {
	defer w.trafficGen.wg.Done()
	start := time.Now()

	workerFmt := fmt.Sprintf("worker#%%0%dd", getPadding(nbOfClients))
	counterFmt := fmt.Sprintf(" - %%0%dd/%%d ", getPadding(nbOfRequests))

	prefix := fmt.Sprintf(workerFmt, w.id)
	logger := log.New(os.Stdout, prefix, 0)
	// Repeat nbOfRequests requests
	for i := 1; i <= nbOfRequests; i++ {
		logger.SetPrefix(prefix + fmt.Sprintf(counterFmt, i, nbOfRequests))
		// Find an URL
		url := findRandomURL()
		// Make the request
		r := w.trafficGen.trafficFunc(url)
		// Add the request to the stats
		w.trafficGen.stats.AddRequest(r)
		// Print the request
		logger.Print(r.String())

		time.Sleep(time.Duration(avgMillisecondsToWait) * time.Millisecond)
	}
	w.trafficGen.stats.SetDuration(time.Since(start))
}

// getPadding returns the padding size of the int given
func getPadding(nb int) int {
	// Get the padding size : floor(log10(nb)) + 1
	return int(math.Log10(float64(nb))) + 1
}

func getAvgDuration(total time.Duration, number int) string {
	if number == 0 {
		return "NaN"
	}
	return (total / time.Duration(number)).String()
}
