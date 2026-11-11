// Package state manages the application state for the CloudWatch Log TUI.
// It maintains the current state of log groups, streams, and events during navigation.
package state

// Direction specifies navigation Direction in logs
type Direction int

const (
	// Next navigates to the next page of results
	Next Direction = iota
	// Home returns to the initial page
	Home
	// Prev navigates to the previous page of results
	Prev
)

// UIState maintains the current state of the user interface,
// including selected log groups, streams, and events.
type UIState struct {
	LogGroup  *LogGroup
	LogStream *LogStream
	LogEvent  *LogEvent
}

// New creates a new UIState instance with initialized sub-components.
func New() *UIState {
	return &UIState{
		LogEvent:  &LogEvent{},
		LogGroup:  &LogGroup{},
		LogStream: &LogStream{},
	}
}
