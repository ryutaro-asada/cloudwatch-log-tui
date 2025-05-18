package state

import (
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/rivo/tview"
	awsr "github.com/ryutaro-asada/cloudwatch-log-tui/internal/aws"
	"github.com/ryutaro-asada/cloudwatch-log-tui/internal/view"
)

type LogEvent struct {
	startYear          int
	startMonth         int
	startDay           int
	startHour          int
	startMinute        int
	endYear            int
	endMonth           int
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

func (l *LogEvent) GetStartTime() (int, int, int, int, int) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.startYear, l.startMonth, l.startDay, l.startHour, l.startMinute
}

func (l *LogEvent) GetEndTime() (int, int, int, int, int) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.endYear, l.endMonth, l.endDay, l.endHour, l.endMinute
}

func (l *LogEvent) GetCurrntState() LogEvent {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return LogEvent{
		startYear:          l.startYear,
		startMonth:         l.startMonth,
		startDay:           l.startDay,
		startHour:          l.startHour,
		startMinute:        l.startMinute,
		endYear:            l.endYear,
		endMonth:           l.endMonth,
		endDay:             l.endDay,
		endHour:            l.endHour,
		endMinute:          l.endMinute,
		logGroupName:       l.logGroupName,
		logStreamNames:     l.logStreamNames,
		filterPatern:       l.filterPatern,
		enableFilterPatern: l.enableFilterPatern,
		outputFile:         l.outputFile,
		enableOutputFile:   l.enableOutputFile,
	}
}

func (l *LogEvent) SetLogStreamsSelected(logStreamNames []string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.logStreamNames = logStreamNames
}

func (l *LogEvent) SetFilterPatern(filterPatern string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.filterPatern = filterPatern
}

func (l *LogEvent) SetOutputFile(outputFile string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.outputFile = outputFile
}

func (l *LogEvent) SetDefaultTime() {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	oneHourBefore := now.Add(-1 * time.Hour)

	l.startYear = oneHourBefore.Year()
	l.startMonth = int(oneHourBefore.Month())
	l.startDay = oneHourBefore.Day()
	l.startHour = oneHourBefore.Hour()
	l.startMinute = oneHourBefore.Minute()

	l.endYear = now.Year()
	l.endMonth = int(now.Month())
	l.endDay = now.Day()
	l.endHour = now.Hour()
	l.endMinute = now.Minute()
}

func (l *LogEvent) BeforeGet(input *awsr.LogEventInput) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.isInValid() {
		log.Fatalf("invalid log event state")
	}

	input.LogGroupName = l.logGroupName
	input.LogStreamNames = l.logStreamNames
	input.FilterPattern = l.filterPatern
	input.StartTime = time.Date(l.startYear, time.Month(l.startMonth), l.startDay, l.startHour, l.startMinute, 0, 0, time.UTC)
	input.EndTime = time.Date(l.endYear, time.Month(l.endMonth), l.endDay, l.endHour, l.endMinute, 0, 0, time.UTC)
}

func (l *LogEvent) isInValid() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if l.logGroupName == "" {
		return true
	}
	if len(l.logStreamNames) == 0 {
		return true
	}
	return l.startYear == 0 ||
		l.startMonth == 0 ||
		l.startDay == 0 ||
		l.startHour == 0 ||
		l.startMinute == 0 ||
		l.endYear == 0 ||
		l.endMonth == 0 ||
		l.endDay == 0 ||
		l.endHour == 0 ||
		l.endMinute == 0
}

func (l *LogEvent) SetTime(label string, text string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	switch label {
	case view.WidgetNames[view.StartYearDropDown]:
		l.startYear = string2int(text)
	case view.WidgetNames[view.StartMonthDropDown]:
		l.startMonth = string2int(text)
	case view.WidgetNames[view.StartDayDropDown]:
		l.startDay = string2int(text)
	case view.WidgetNames[view.StartHourDropDown]:
		l.startHour = string2int(text)
	case view.WidgetNames[view.StartMinuteDropDown]:
		l.startMinute = string2int(text)
	case view.WidgetNames[view.EndYearDropDown]:
		l.endYear = string2int(text)
	case view.WidgetNames[view.EndMonthDropDown]:
		l.endMonth = string2int(text)
	case view.WidgetNames[view.EndDayDropDown]:
		l.endDay = string2int(text)
	case view.WidgetNames[view.EndHourDropDown]:
		l.endHour = string2int(text)
	case view.WidgetNames[view.EndMinuteDropDown]:
		l.endMinute = string2int(text)
	}
}

func string2int(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		log.Fatalf("unable to list tables, %v", err)
	}
	return i
}

func (l *LogEvent) Print(textView *tview.TextView) {
	fmt.Fprintf(textView, "Your setting is\n")
	fmt.Fprintf(textView, "%s\n", l.logGroupName)
	fmt.Fprintf(textView, "%s\n", l.logStreamNames)
	fmt.Fprintf(textView, "%s\n", l.filterPatern)

	fmt.Fprintf(textView, "%s/%s/%s %s:%s\n",
		strconv.Itoa(l.startYear),
		strconv.Itoa(l.startMonth),
		strconv.Itoa(l.startDay),
		strconv.Itoa(l.startHour),
		strconv.Itoa(l.startMinute),
	)
	fmt.Fprintf(textView, "    ~     \n")
	fmt.Fprintf(textView, "%s/%s/%s %s:%s\n",
		strconv.Itoa(l.endYear),
		strconv.Itoa(l.endMonth),
		strconv.Itoa(l.endDay),
		strconv.Itoa(l.endHour),
		strconv.Itoa(l.endMinute),
	)
	fmt.Fprintf(textView, " \n")
}
