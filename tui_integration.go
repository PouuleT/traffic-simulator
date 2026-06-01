package main

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/sync/errgroup"
)

// StartWithTUI starts the traffic generator with TUI
func (a *app) StartWithTUI() error {
	totalReqs := a.cfg.NbOfClients * a.cfg.NbOfRequests

	// Create requests channel for TUI updates
	a.requestsCh = make(chan requestLogEntry, 100)

	// Create a cancelable context for background workers
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create the TUI model
	model := newTUIModel(a.cfg, a.stats, totalReqs, cancel)

	// Create the Bubble Tea program
	p := tea.NewProgram(&model, tea.WithAltScreen())

	// Start the traffic generation in a goroutine
	go func() {
		_ = a.startTrafficGeneration(ctx)
		// Signal completion
		p.Send(allDoneMsg{})
	}()

	// Start forwarding request updates to the TUI
	go func() {
		for entry := range a.requestsCh {
			p.Send(requestCompletedMsg{entry: entry})
		}
	}()

	// Run the TUI (blocks until quit)
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}

// startTrafficGeneration starts the workers (used by both TUI and non-TUI modes)
func (a *app) startTrafficGeneration(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	for i := 1; i <= a.cfg.NbOfClients; i++ {
		w, err := a.NewWorker(i)
		if err != nil {
			return err
		}

		a.workers = append(a.workers, w)
		g.Go(func() error {
			w.workWithChannel(ctx, a.requestsCh)
			return nil
		})
	}

	// Wait for all workers
	_ = g.Wait()

	// Close the channel when done
	if a.requestsCh != nil {
		close(a.requestsCh)
	}

	return nil
}
