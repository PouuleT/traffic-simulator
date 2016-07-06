package main

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/olekukonko/tablewriter"
)

// DurationStats represents statistics of durations
type DurationStats struct {
	maxDuration   time.Duration
	minDuration   time.Duration
	totalDuration time.Duration
	execDuration  time.Duration
}

// Stats represents the stats of the requests
type Stats struct {
	sync.Mutex
	nbOfRequests int
	durations    DurationStats
	statusStats  map[string]int
}

// newStats will return an empty Stats object
func newStats() *Stats {
	return &Stats{
		statusStats: map[string]int{},
	}
}

// addError will add a Request error to the stats
func (s *Stats) addError(req *Request) {
	s.Lock()
	defer s.Unlock()

	s.nbOfRequests++
	s.addDuration(req)

	var errName string
	switch e := req.err.(type) {
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
			errName = "URL Error"
		}
	case net.Error:
		errName = "Net Error"
	default:
		errName = e.Error()
	}
	s.statusStats[errName]++
}

// addRequest will add a successful request to the stats
func (s *Stats) addRequest(req *Request) {
	s.Lock()
	defer s.Unlock()
	s.nbOfRequests++
	s.addDuration(req)

	// If some http statuses has no StatusText, return a simple string with the
	// http status
	statusText := http.StatusText(req.status)
	if statusText == "" {
		statusText = fmt.Sprintf("%d", req.status)
	}
	s.statusStats[statusText]++
}

// addDuration will add the duration of a requests to the stats
func (s *Stats) addDuration(req *Request) {
	s.durations.totalDuration += req.duration
	if s.durations.maxDuration < req.duration {
		s.durations.maxDuration = req.duration
	}
	if s.durations.minDuration == 0 || s.durations.minDuration > req.duration {
		s.durations.minDuration = req.duration
	}
}

// Render renders the results
func (s *Stats) Render() {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetAlignment(tablewriter.ALIGN_CENTRE)
	table.SetHeader([]string{
		"Number of requests ",
		"Min duration",
		"Max duration",
		"Average duration",
		"Total duration",
	})
	table.Append([]string{
		strconv.Itoa(s.nbOfRequests),
		s.durations.minDuration.String(),
		s.durations.maxDuration.String(),
		(s.durations.totalDuration / time.Duration(s.nbOfRequests)).String(),
		s.durations.execDuration.String(),
	})

	fmt.Printf("\nStats :\n")
	table.Render()

	statusTable := tablewriter.NewWriter(os.Stdout)
	statusTable.SetAlignment(tablewriter.ALIGN_CENTRE)
	statusTable.SetHeader([]string{"Result", "Count"})
	for key, value := range s.statusStats {
		statusTable.Append([]string{key, strconv.Itoa(value)})
	}

	fmt.Printf("\nStatuses :\n")
	statusTable.Render()
}
