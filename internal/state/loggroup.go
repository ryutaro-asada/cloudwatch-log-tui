// Package state manages the application state for the CloudWatch Log TUI.
package state

import (
	"sync"

	awsr "github.com/ryutaro-asada/cloudwatch-log-tui/internal/aws"
)

// LogGroup manages the state for CloudWatch log groups,
// including pagination and the search pattern.
type LogGroup struct {
	Pagination
	mu            sync.Mutex
	filterPattern string
}

// BeforeGet fills input for fetching the page in direction direct.
// It returns the page number to pass to AfterGet, and false when there is no
// page in that direction.
func (l *LogGroup) BeforeGet(input *awsr.LogGroupInput, direct Direction) (int, bool) {
	page, token, ok := l.Target(direct)
	if !ok {
		return 0, false
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	input.FilterPattern = l.filterPattern
	input.NextToken = token
	return page, true
}

// AfterGet records the fetched page. It returns false when the page was an
// empty later page; the previously shown page stays current.
func (l *LogGroup) AfterGet(page int, output *awsr.LogGroupOutput) bool {
	return l.Apply(page, len(output.LogGroups), output.NextToken)
}

// SetFilterPattern updates the search pattern for log group queries.
// Changing the pattern invalidates the page tokens.
func (l *LogGroup) SetFilterPattern(filterPattern string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.filterPattern != filterPattern {
		l.filterPattern = filterPattern
		l.Reset()
	}
}
