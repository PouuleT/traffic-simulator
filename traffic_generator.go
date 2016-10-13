package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"sync"
	"syscall"
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

var exitChan = make(chan struct{})

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
	// Create a channel that will listen to SIGINT / SIGTERM
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT)
	signal.Notify(c, syscall.SIGTERM)

	for i := 1; i <= nbOfClients; i++ {
		trafficGen.wg.Add(1)
		// Launch the workers in a go routine
		go trafficGen.NewWorker(i).work()
	}

	// Done channel to stop the loop when all the workers are done
	var done = make(chan struct{})
	go func() {
		// Wait for the workerss
		trafficGen.wg.Wait()
		// All the workers are done
		done <- struct{}{}
	}()

	// Wait for the workers to end
	// or a signal in the loop
	var forceShutdown bool
	for {
		select {
		case <-done:
			// All the workers are done, we quit
			return
		case sig := <-c:
			// We listen for signals
			switch sig {
			case syscall.SIGINT, syscall.SIGKILL:
				// If it's the second time we get a signal, quit
				if forceShutdown {
					os.Exit(1)
				}

				// Notify all the workers that they need to stop
				for i := 1; i <= nbOfClients; i++ {
					exitChan <- struct{}{}
				}

				// Next time we get a signal, need to quit directly
				forceShutdown = true
			}
		}
	}
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
	var exit bool
	defer w.trafficGen.wg.Done()
	start := time.Now()

	var done = make(chan struct{})
	// When the work is done, notify the watching go routine
	defer func() { done <- struct{}{} }()

	// Create the watching go routine that will watch if we need to quit early
	go func() {
		for {
			select {
			// If we need to exit
			case <-exitChan:
				exit = true
			// If everything is done
			case <-done:
				return
			}
		}
	}()

	workerFmt := fmt.Sprintf("worker#%%0%dd", getPadding(nbOfClients))
	counterFmt := fmt.Sprintf(" - %%0%dd/%%d ", getPadding(nbOfRequests))

	prefix := fmt.Sprintf(workerFmt, w.id)
	logger := log.New(os.Stdout, prefix, 0)

	// Repeat nbOfRequests requests
	for i := 1; i <= nbOfRequests; i++ {
		// If we got an exit signal, quit
		if exit {
			return
		}
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
