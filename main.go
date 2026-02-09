package main

import (
	"errors"
	"flag"
	"io"
	"log"
	"math/rand/v2"
	"sync"
	"time"
)

var (
	ErrInvalidTrafficType = errors.New("invalid traffic type")
)

type config struct {
	NbOfClients           int
	NbOfRequests          int
	AvgMillisecondsToWait int
	FileName              string
	TrafficType           string
	Timeout               int
	Seed                  int64
	FollowHTTPRedirect    bool
	UseTUI                bool // Enable TUI mode
}

type app struct {
	URLMutex sync.Mutex
	URLs     []string
	rng      *rand.Rand
	cfg      config

	stats      Stats
	workers    []*Worker
	requestsCh chan requestLogEntry // Channel for TUI updates
}

func (a *app) ParseFlags() error {
	flag.IntVar(&a.cfg.NbOfClients, "clients", 10, "number of clients making requests")
	flag.IntVar(&a.cfg.NbOfRequests, "requests", 10, "number of requests to be made by each clients")
	flag.IntVar(&a.cfg.AvgMillisecondsToWait, "wait", 1000, "milliseconds to wait between each requests")
	flag.IntVar(&a.cfg.Timeout, "timeout", 3, "HTTP timeout in seconds")
	flag.Int64Var(&a.cfg.Seed, "seed", time.Now().UTC().UnixNano(), "seed for the random")
	flag.StringVar(&a.cfg.TrafficType, "type", "http", "type of requests http/dns")
	flag.StringVar(&a.cfg.FileName, "urlSource", "", "optional filepath where to find the URLs")
	flag.BoolVar(&a.cfg.FollowHTTPRedirect, "followRedirect", true, "follow http redirects or not")

	var plainMode bool
	flag.BoolVar(&plainMode, "plain", false, "disable TUI and use plain text output")
	flag.Parse()

	// Set TUI mode (enabled by default unless -plain is used)
	a.cfg.UseTUI = !plainMode

	a.rng = rand.New(rand.NewPCG(uint64(a.cfg.Seed), 0))
	return nil
}

func main() {
	a := &app{}
	err := a.ParseFlags()
	if err != nil {
		log.Fatal(err)
	}

	log.SetFlags(0)

	// Disable log output in TUI mode to prevent console interference
	if a.cfg.UseTUI {
		log.SetOutput(io.Discard)
	}

	stats, err := newStats(a.cfg.TrafficType)
	if err != nil {
		log.Fatalf("Error: %q", err)
	}
	a.stats = stats

	// Get the URLs
	if err := a.fillURLs(); err != nil {
		log.Fatalf("Error while getting the URLs: %q", err)
	}

	// Start the traffic with TUI or plain output
	if a.cfg.UseTUI {
		err = a.StartWithTUI()
	} else {
		err = a.Start()
	}

	if err != nil {
		log.Fatalf("Error while generating traffic: %q", err)
	}

	// Display the statistics (only in plain mode; TUI handles its own display)
	if !a.cfg.UseTUI {
		a.DisplayStats()
	}
}
