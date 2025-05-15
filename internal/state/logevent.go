package state

import (
	"sync"
	"time"

	awsr "github.com/ryutaro-asada/cloudwatch-log-tui/internal/aws"
)

type LogEvent struct {
	startYear          int
	startMonth         time.Month
	startDay           int
	startHour          int
	startMinute        int
	endYear            int
	endMonth           time.Month
	endDay             int
	endHour            int
	endMinute          int
	logGroupName       string
	logStreamNames     []string
	filterPatern       string
	enableFilterPatern bool
	outputFile         string
	enableOutputFile   bool
	mu                 sync.RWMutex
}

func (l *LogEvent) SetLogGroupSelected(logGroupName string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.logGroupName = logGroupName
}

func (l *LogEvent) GetLogStreamsSelected() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.logStreamNames
}

func (l *LogEvent) SetLogStreamsSelected(logStreamNames []string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.logStreamNames = logStreamNames
}
