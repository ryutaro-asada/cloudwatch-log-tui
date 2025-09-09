// Package state manages the application state for the CloudWatch Log TUI.
package state

import (
	"sync"
	// "log"

	awsr "github.com/ryutaro-asada/cloudwatch-log-tui/internal/aws"
)

// LogGroup manages the state for CloudWatch log groups,
// including pagination, filtering, and navigation state.
type LogGroup struct {
	filterPatern string
	currentPage  int
	hasNext      bool
	hasPrev      bool
	pageTokens   map[int]*string
	mu           sync.RWMutex
}

// BeforeGet prepares the input parameters before fetching log groups.
// It sets the filter pattern and pagination token based on the navigation direction.
func (l *LogGroup) BeforeGet(input *awsr.LogGroupInput, direct Direction) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	input.FilterPattern = l.filterPatern

	switch direct {
	case Next:
		input.NextToken = l.pageTokens[l.currentPage+1]

	case Prev:
		// if currentPageLogGroup == 1, a.pageTokenLogGroup will return nil and params.NextToken will be nil
		// so, first page will be fetched
		input.NextToken = l.pageTokens[l.currentPage-1]
	}
}

// AfterGet updates the state after fetching log groups.
// It manages pagination tokens and updates navigation flags based on the results.
func (l *LogGroup) AfterGet(output *awsr.LogGroupOutput, direct Direction) {
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

	if output.NextToken != nil && len(output.LogGroups) == int(awsr.MaxItemsInLayout) {
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

// HasPrev returns true if there is a previous page of log groups available.
func (l *LogGroup) HasPrev() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.hasPrev
}

// HasNext returns true if there is a next page of log groups available.
func (l *LogGroup) HasNext() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.hasNext
}

// SetFilterPattern updates the filter pattern for log group queries.
// The filter pattern is used to search for specific log groups.
func (l *LogGroup) SetFilterPattern(filterPatern string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.filterPatern = filterPatern
}
