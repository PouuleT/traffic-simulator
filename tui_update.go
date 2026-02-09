package main

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

// Update handles messages and updates the model
func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		// If we're done, allow quitting with more keys
		if m.done {
			switch msg.String() {
			case "q", "enter", "ctrl+c", "esc":
				return m, tea.Quit
			}
		} else {
			// During execution, only Ctrl+C quits
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.progress.Width = msg.Width - 20
		return m, nil

	case tickMsg:
		if m.done {
			return m, nil
		}
		// Continue ticking
		return m, tickCmd()

	case requestCompletedMsg:
		m.completed++

		// Add to ring buffer
		m.requestLog = append(m.requestLog, msg.entry)
		if len(m.requestLog) > m.maxLogSize {
			m.requestLog = m.requestLog[1:]
		}

		return m, nil

	case allDoneMsg:
		m.done = true
		m.completionTime = time.Since(m.startTime)
		return m, nil // Don't quit, let user inspect results

	case error:
		m.err = msg
		return m, tea.Quit
	}

	// Update progress bar only if not done
	if !m.done {
		prog, cmd := m.progress.Update(msg)
		if p, ok := prog.(progress.Model); ok {
			m.progress = p
		}
		return m, cmd
	}

	return m, nil
}

// View renders the TUI
func (m tuiModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	if m.done {
		return m.renderFinalSummary()
	}

	return m.renderLiveView()
}
