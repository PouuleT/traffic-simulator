package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
)

// Generator represents a generator interface
type Generator interface {
	MakeRequest(context.Context, string) Request
}

// Worker represents a client making the requests
type Worker struct {
	id                    int
	trafficGen            *app
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
	// Create a context that will be cancelled on SIGINT / SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	start := time.Now()
	defer func() {
		a.stats.SetDuration(time.Since(start))
	}()

	g, ctx := errgroup.WithContext(ctx)

	for i := 1; i <= a.cfg.NbOfClients; i++ {
		// Launch the workers in a go routine
		w, err := a.NewWorker(i)
		if err != nil {
			return err
		}

		a.workers = append(a.workers, w)
		g.Go(func() error {
			w.work(ctx)
			return nil
		})
	}

	// Done channel to stop the loop when all the workers are done
	var done = make(chan struct{})
	go func() {
		// Wait for the workers
		_ = g.Wait()
		// All the workers are done
		done <- struct{}{}
	}()

	// Wait for the workers to end or a signal
	var forceShutdown bool
	for {
		select {
		case <-done:
			// All the workers are done, we quit
			return nil
		case <-ctx.Done():
			// We received a signal
			if forceShutdown {
				os.Exit(0)
			}
			// First signal: let context cancellation stop workers gracefully
			// Next time we get a signal, need to quit directly
			forceShutdown = true
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
		NbOfClients:           a.cfg.NbOfClients,
		NbOfRequests:          a.cfg.NbOfRequests,
		AvgMillisecondsToWait: a.cfg.AvgMillisecondsToWait,
	}, nil
}

// DisplayStats renders the statistics of the traffic generation
func (a *app) DisplayStats() {
	a.stats.Render()
}

func (w *Worker) work(ctx context.Context) {
	w.workWithChannel(ctx, nil)
}

func (w *Worker) workWithChannel(ctx context.Context, requestsCh chan<- requestLogEntry) {
	workerFmt := fmt.Sprintf("worker#%%0%dd", getPadding(w.NbOfClients))
	counterFmt := fmt.Sprintf(" - %%0%dd/%%d ", getPadding(w.NbOfRequests))

	prefix := fmt.Sprintf(workerFmt, w.id)
	var logger *log.Logger
	if requestsCh == nil {
		// Plain mode: use logger
		logger = log.New(os.Stdout, prefix, 0)
	}

	// Repeat nbOfRequests requests
	for i := 1; i <= w.NbOfRequests; i++ {
		select {
		// If we need to exit
		case <-ctx.Done():
			return
		default:
		}

		// Find an URL and make the request
		r := w.generator.MakeRequest(ctx, w.trafficGen.findRandomURL())

		// Add the request to the stats
		w.trafficGen.stats.AddRequest(r)

		// Send to channel or log
		if requestsCh != nil {
			// TUI mode: send to channel
			entry := requestLogEntry{
				workerID:    w.id,
				reqNumber:   i,
				totalReqs:   w.NbOfRequests,
				status:      getShortStatus(r),
				isError:     r.IsError(),
				duration:    r.Duration(),
				url:         extractURL(r),
				sizeOrError: getSizeOrError(r),
			}
			select {
			case requestsCh <- entry:
			case <-ctx.Done():
				return
			}
		} else {
			// Plain mode: log to stdout
			logger.SetPrefix(prefix + fmt.Sprintf(counterFmt, i, w.NbOfRequests))
			logger.Print(r.String())
		}

		time.Sleep(time.Duration(w.AvgMillisecondsToWait) * time.Millisecond)
	}
}

// getShortStatus extracts a short status string from a request
func getShortStatus(r Request) string {
	if r.IsError() {
		return "ERR"
	}
	status := r.Status()
	if len(status) >= 3 {
		return status[:3]
	}
	return status
}

// extractURL extracts the URL from a request (strips http:// prefix)
func extractURL(r Request) string {
	s := r.String()
	// Parse the URL from the string representation
	// Format is typically "| STATUS | DURATION | Get URL ..."
	parts := strings.Split(s, "Get ")
	if len(parts) >= 2 {
		urlPart := strings.TrimSpace(parts[1])
		// Remove http://
		urlPart = strings.TrimPrefix(urlPart, "http://")
		urlPart = strings.TrimPrefix(urlPart, "https://")
		// Remove trailing parts after space
		if idx := strings.Index(urlPart, " "); idx > 0 {
			urlPart = urlPart[:idx]
		}
		return urlPart
	}
	return ""
}

// getSizeOrError gets the size or error message from a request
func getSizeOrError(r Request) string {
	if r.IsError() {
		return r.Error()
	}
	if r.Size() > 0 {
		return fmt.Sprintf("(%s)", humanizeBytes(uint64(r.Size())))
	}
	return "(0 B)"
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
