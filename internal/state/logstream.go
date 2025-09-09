// Package state manages the application state for the CloudWatch Log TUI.
package state

import (
	"sync"

	awsr "github.com/ryutaro-asada/cloudwatch-log-tui/internal/aws"
)

// LogStream manages the state for CloudWatch log streams within a log group,
// including pagination, filtering, and navigation state.
type LogStream struct {
	prefixPatern string
	logGroupName string
	currentPage  int
	hasNext      bool
	hasPrev      bool
	pageTokens   map[int]*string
	mu           sync.RWMutex
}

// BeforeGet prepares the input parameters before fetching log streams.
// It sets the log group name and pagination token based on the navigation direction.
func (l *LogStream) BeforeGet(input *awsr.LogStreamInput, direct Direction) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	input.LogGroupName = l.logGroupName

	switch direct {
	case Next:
		input.NextToken = l.pageTokens[l.currentPage+1]

	case Prev:
		// if currentPageLogGroup == 1, a.pageTokenLogGroup will return nil and params.NextToken will be nil
		// so, first page will be fetched
		input.NextToken = l.pageTokens[l.currentPage-1]
	}
}

// AfterGet updates the state after fetching log streams.
// It manages pagination tokens and updates navigation flags based on the results.
func (l *LogStream) AfterGet(output *awsr.LogStreamOutput, direct Direction) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	switch direct {
	case Next:
		l.currentPage++
	case Prev:
		l.currentPage--
	case Home:
		l.currentPage = 1
	}

	if output.NextToken != nil && len(output.LogStreams) == int(awsr.MaxItemsInLayout) {
		l.pageTokens[l.currentPage+1] = output.NextToken
		l.hasNext = true
	} else {
		l.hasNext = false
	}

	if l.currentPage > 1 {
		l.hasPrev = true
	} else {
		l.hasPrev = false
	}
}

// HasPrev returns true if there is a previous page of log streams available.
func (l *LogStream) HasPrev() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.hasPrev
}

// HasNext returns true if there is a next page of log streams available.
func (l *LogStream) HasNext() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.hasNext
}

// SetLogGroupSelected updates the currently selected log group name
// for which log streams will be fetched.
func (l *LogStream) SetLogGroupSelected(logGroupName string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.logGroupName = logGroupName
}
