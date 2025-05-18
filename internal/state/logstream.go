package state

import (
	"sync"

	awsr "github.com/ryutaro-asada/cloudwatch-log-tui/internal/aws"
)

type LogStream struct {
	prefixPatern string
	logGroupName string
	currentPage  int
	hasNext      bool
	hasPrev      bool
	pageTokens   map[int]*string
	mu           sync.RWMutex
}

func (l *LogStream) BeforeGet(input *awsr.LogStreamInput, direct Direction) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	input.LogGroupName = l.logGroupName
	input.PrefixPattern = l.prefixPatern

	switch direct {
	case Next:
		input.NextToken = l.pageTokens[l.currentPage+1]

	case Prev:
		// if currentPageLogGroup == 1, a.pageTokenLogGroup will return nil and params.NextToken will be nil
		// so, first page will be fetched
		input.NextToken = l.pageTokens[l.currentPage-1]
	}
}

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

func (l *LogStream) HasPrev() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.hasPrev
}

func (l *LogStream) HasNext() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.hasNext
}

func (l *LogStream) SetLogGroupSelected(logGroupName string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.logGroupName = logGroupName
}

func (l *LogStream) SetPrefixPattern(prefixPatern string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.prefixPatern = prefixPatern
}

