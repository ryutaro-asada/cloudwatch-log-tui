// Package state manages the application state for the CloudWatch Log TUI.
package state

import (
	"sync"

	awsr "github.com/ryutaro-asada/cloudwatch-log-tui/internal/aws"
)

// LogStream manages the state for CloudWatch log streams within a log group,
// including pagination.
type LogStream struct {
	Pagination
	mu           sync.Mutex
	logGroupName string
}

// BeforeGet fills input for fetching the page in direction direct.
// It returns the page number to pass to AfterGet, and false when there is no
// page in that direction.
func (l *LogStream) BeforeGet(input *awsr.LogStreamInput, direct Direction) (int, bool) {
	page, token, ok := l.Target(direct)
	if !ok {
		return 0, false
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	input.LogGroupName = l.logGroupName
	input.NextToken = token
	return page, true
}

// AfterGet records the fetched page. It returns false when the page was an
// empty later page; the previously shown page stays current.
func (l *LogStream) AfterGet(page int, output *awsr.LogStreamOutput) bool {
	return l.Apply(page, len(output.LogStreams), output.NextToken)
}

// SetLogGroupSelected updates the log group whose streams are listed.
// Selecting a different group invalidates the page tokens.
func (l *LogStream) SetLogGroupSelected(logGroupName string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.logGroupName != logGroupName {
		l.logGroupName = logGroupName
		l.Reset()
	}
}
