package main

import (
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

// tuiModel is the Bubble Tea model for the traffic simulator TUI
type tuiModel struct {
	// Config
	cfg         config
	totalReqs   int // Total requests across all workers
	trafficType string

	// Stats reference
	stats Stats

	// Progress
	progress       progress.Model
	completed      int
	startTime      time.Time
	completionTime time.Duration // Duration when all requests completed

	// Request log ring buffer
	requestLog []requestLogEntry
	maxLogSize int

	// State
	done   bool
	width  int
	height int
	err    error
}

// requestLogEntry represents a single completed request for display
type requestLogEntry struct {
	workerID    int
	reqNumber   int
	totalReqs   int
	status      string
	isError     bool
	duration    time.Duration
	url         string
	sizeOrError string
}

// Bubble Tea messages
type tickMsg time.Time
type requestCompletedMsg struct {
	entry requestLogEntry
}
type allDoneMsg struct{}

// Init initializes the Bubble Tea model
func (m tuiModel) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
	)
}

// tickCmd sends periodic tick messages for stats refresh
func tickCmd() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// newTUIModel creates a new TUI model
func newTUIModel(cfg config, stats Stats, totalReqs int) tuiModel {
	prog := progress.New(progress.WithDefaultGradient())

	return tuiModel{
		cfg:         cfg,
		totalReqs:   totalReqs,
		trafficType: cfg.TrafficType,
		stats:       stats,
		progress:    prog,
		startTime:   time.Now(),
		requestLog:  make([]requestLogEntry, 0, 50),
		maxLogSize:  50,
		done:        false,
	}
}
