package state

import (
	"sync"

	awsr "github.com/ryutaro-asada/cloudwatch-log-tui/internal/aws"
)

type LogGroup struct {
	filterPatern string
	currentPage  int
	hasNext      bool
	hasPrev      bool
	pageTokens   map[int]*string
	mu           sync.RWMutex
}

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

func (l *LogGroup) HasPrev() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.hasPrev
}

func (l *LogGroup) HasNext() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.hasNext
}

func (l *LogGroup) SetFilterPattern(filterPatern string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.filterPatern = filterPatern
}

