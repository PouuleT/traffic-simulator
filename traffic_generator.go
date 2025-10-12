package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Generator represents a generator interface
type Generator interface {
	MakeRequest(string) Request
}

// Worker represents a client making the requests
type Worker struct {
	id                    int
	trafficGen            *app
	exitChan              chan struct{}
	generator             Generator
	NbOfClients           int
	NbOfRequests          int
	AvgMillisecondsToWait int
}

var generatorMap = map[string]func(cfg config) Generator{
	"http": newHTTPGenerator,
	"dns":  newDNSGenerator,
}

var statsMap = map[string]func() Stats{
	"http": newHTTPStats,
	"dns":  newDNSStats,
}

func (a *app) newGenerator() (Generator, error) {
	newGenerator, ok := generatorMap[a.cfg.TrafficType]
	if !ok {
		return nil, ErrInvalidTrafficType
	}
	return newGenerator(a.cfg), nil
}

// Start the TrafficGenerator
func (a *app) Start() error {
	// Create a channel that will listen to SIGINT / SIGTERM
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

	start := time.Now()
	defer func() {
		a.stats.SetDuration(time.Since(start))
	}()

	for i := 1; i <= a.cfg.NbOfClients; i++ {
		a.wg.Add(1)
		// Launch the workers in a go routine
		w, err := a.NewWorker(i)
		if err != nil {
			return err
		}

		a.workers = append(a.workers, w)
		go func() {
			w.work()
			a.wg.Done()
		}()
	}

	// Done channel to stop the loop when all the workers are done
	var done = make(chan struct{})
	go func() {
		// Wait for the workerss
		a.wg.Wait()
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
			return nil
		case sig := <-c:
			// We listen for signals
			switch sig {
			case syscall.SIGINT, syscall.SIGTERM:
				// If it's the second time we get a signal, quit
				if forceShutdown {
					os.Exit(0)
				}

				// Notify all the workers that they need to stop
				for _, w := range a.workers {
					w.exitChan <- struct{}{}
				}

				// Next time we get a signal, need to quit directly
				forceShutdown = true
			}
		}
	}
}

// NewWorker creates a new worker for traffic generation
func (a *app) NewWorker(i int) (*Worker, error) {
	generator, err := a.newGenerator()
	if err != nil {
		return nil, ErrInvalidTrafficType
	}
	return &Worker{
		id:                    i,
		trafficGen:            a,
		generator:             generator,
		exitChan:              make(chan struct{}, 1),
		NbOfClients:           a.cfg.NbOfClients,
		NbOfRequests:          a.cfg.NbOfRequests,
		AvgMillisecondsToWait: a.cfg.AvgMillisecondsToWait,
	}, nil
}

// DisplayStats renders the statistics of the traffic generation
func (a *app) DisplayStats() {
	a.stats.Render()
}

func (w *Worker) work() {
	workerFmt := fmt.Sprintf("worker#%%0%dd", getPadding(w.NbOfClients))
	counterFmt := fmt.Sprintf(" - %%0%dd/%%d ", getPadding(w.NbOfRequests))

	prefix := fmt.Sprintf(workerFmt, w.id)
	logger := log.New(os.Stdout, prefix, 0)

	// Repeat nbOfRequests requests
	for i := 1; i <= w.NbOfRequests; i++ {
		select {
		// If we need to exit
		case <-w.exitChan:
			return
		default:
		}
		logger.SetPrefix(prefix + fmt.Sprintf(counterFmt, i, w.NbOfRequests))
		// Find an URL and make the request
		r := w.generator.MakeRequest(w.trafficGen.findRandomURL())
		// Add the request to the stats
		w.trafficGen.stats.AddRequest(r)
		// Print the request
		logger.Print(r.String())

		time.Sleep(time.Duration(w.AvgMillisecondsToWait) * time.Millisecond)
	}
}

// getPadding returns the padding size of the int given
func getPadding(nb int) int {
	// Get the padding size : floor(log10(nb)) + 1
	if nb <= 0 {
		return 1
	}
	return int(math.Log10(float64(nb))) + 1
}

func getAvgDuration(total time.Duration, number int) string {
	if number == 0 {
		return "NaN"
	}
	return (total / time.Duration(number)).String()
}
