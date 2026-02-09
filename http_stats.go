package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/olekukonko/tablewriter"
)

// HTTPStats represents the stats of the requests
type HTTPStats struct {
	DurationStats
	sync.Mutex
	nbOfRequests     int
	successRequests  int
	statusStats      map[string]int
	totalSize        int64
	responseTimeline *ResponseTimeline
}

// newHTTPStats will return an empty Stats object
func newHTTPStats() Stats {
	return &HTTPStats{
		DurationStats:    DurationStats{},
		statusStats:      map[string]int{},
		responseTimeline: &ResponseTimeline{},
	}
}

// AddRequest will add a request to the stats
func (s *HTTPStats) AddRequest(req Request) {
	s.Lock()
	defer s.Unlock()
	s.nbOfRequests++
	s.addDuration(req)
	s.totalSize += req.Size()

	if req.IsError() {
		s.statusStats[req.Error()]++
		return
	}
	s.successRequests++
	s.statusStats[req.Status()]++
}

// addDuration will add the duration of a requests to the stats
func (s *HTTPStats) addDuration(req Request) {
	r, ok := req.(*HTTPRequest)
	if !ok {
		log.Fatal("Handling an unexpected request")
	}
	s.updateDuration(req.Duration())
	if r.responseTimeline == nil {
		return
	}
	s.responseTimeline.DNSLookup += r.responseTimeline.DNSLookup
	s.responseTimeline.TCPConnection += r.responseTimeline.TCPConnection
	s.responseTimeline.EstablishingConnection += r.responseTimeline.EstablishingConnection
	s.responseTimeline.ServerProcessing += r.responseTimeline.ServerProcessing
	s.responseTimeline.ContentTransfer += r.responseTimeline.ContentTransfer
}

// Render renders the results
func (s *HTTPStats) Render() {
	table := tablewriter.NewTable(os.Stdout)
	table.Header([]string{
		"Number of requests ",
		"Min duration",
		"Max duration",
		"Average duration",
		"Exec duration",
		"Avg speed",
		"Total size",
	})

	var speed uint64
	if s.execDuration.Seconds() > 0 {
		speed = uint64(float64(s.totalSize) / s.execDuration.Seconds())
	}
	_ = table.Append([]string{
		strconv.Itoa(s.nbOfRequests),
		s.minDuration.String(),
		s.maxDuration.String(),
		getAvgDuration(s.totalDuration, s.nbOfRequests),
		s.execDuration.String(),
		fmt.Sprintf("%s/s", humanize.Bytes(speed)),
		humanize.Bytes(uint64(s.totalSize)),
	})

	fmt.Printf("\nStats :\n")
	_ = table.Render()

	statusTable := tablewriter.NewTable(os.Stdout)
	statusTable.Header([]string{"Result", "Count"})
	for key, value := range s.statusStats {
		_ = statusTable.Append([]string{key, strconv.Itoa(value)})
	}

	fmt.Printf("\nStatuses :\n")
	_ = statusTable.Render()

	timeTable := tablewriter.NewTable(os.Stdout)
	timeTable.Header([]string{"Step", "Average duration"})
	_ = timeTable.Append([]string{
		"DNSLookup", getAvgDuration(s.responseTimeline.DNSLookup, s.successRequests),
	})
	_ = timeTable.Append([]string{
		"TCPConnection", getAvgDuration(s.responseTimeline.TCPConnection, s.successRequests),
	})
	_ = timeTable.Append([]string{
		"EstablishingConnection", getAvgDuration(s.responseTimeline.EstablishingConnection, s.successRequests),
	})
	_ = timeTable.Append([]string{
		"ServerProcessing", getAvgDuration(s.responseTimeline.ServerProcessing, s.successRequests),
	})
	_ = timeTable.Append([]string{
		"ContentTransfer", getAvgDuration(s.responseTimeline.ContentTransfer, s.successRequests),
	})

	fmt.Printf("\nRequest details :\n")
	_ = timeTable.Render()
}

// SetDuration will set the total duration of the simulation
func (s *HTTPStats) SetDuration(t time.Duration) {
	s.execDuration = t
}

// Snapshot returns a thread-safe snapshot of current stats
func (s *HTTPStats) Snapshot() StatsData {
	s.Lock()
	defer s.Unlock()

	// Copy the map
	statusCopy := make(map[string]int, len(s.statusStats))
	for k, v := range s.statusStats {
		statusCopy[k] = v
	}

	// Copy timeline
	var timelineCopy *ResponseTimeline
	if s.responseTimeline != nil {
		tl := *s.responseTimeline
		timelineCopy = &tl
	}

	return StatsData{
		NbOfRequests:     s.nbOfRequests,
		SuccessRequests:  s.successRequests,
		StatusStats:      statusCopy,
		MinDuration:      s.minDuration,
		MaxDuration:      s.maxDuration,
		TotalDuration:    s.totalDuration,
		ExecDuration:     s.execDuration,
		TotalSize:        s.totalSize,
		ResponseTimeline: timelineCopy,
	}
}
