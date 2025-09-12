// Package state manages the application state for the CloudWatch Log TUI.
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

// LogEvent manages the state for CloudWatch log events,
// including time range, selected streams, filtering options, and output settings.
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

// SetLogGroupSelected updates the currently selected log group name
// from which log events will be fetched.
func (l *LogEvent) SetLogGroupSelected(logGroupName string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.logGroupName = logGroupName
}

// GetLogStreamsSelected returns the list of currently selected log stream names.
func (l *LogEvent) GetLogStreamsSelected() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.logStreamNames
}

// GetStartTime returns the start time components (year, month, day, hour, minute)
// for the log event time range.
func (l *LogEvent) GetStartTime() (int, int, int, int, int) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.startYear, l.startMonth, l.startDay, l.startHour, l.startMinute
}

// GetEndTime returns the end time components (year, month, day, hour, minute)
// for the log event time range.
func (l *LogEvent) GetEndTime() (int, int, int, int, int) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.endYear, l.endMonth, l.endDay, l.endHour, l.endMinute
}

// GetCurrntState returns a copy of the current LogEvent state.
// Note: This function name contains a typo (should be GetCurrentState).
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

// SetLogStreamsSelected updates the list of selected log streams
// from which log events will be fetched.
func (l *LogEvent) SetLogStreamsSelected(logStreamNames []string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.logStreamNames = logStreamNames
}

// SetFilterPatern updates the filter pattern used to search log events.
// Note: This function name contains a typo (should be SetFilterPattern).
func (l *LogEvent) SetFilterPatern(filterPatern string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.filterPatern = filterPatern
}

// SetOutputFile sets the output file path where log events will be saved.
func (l *LogEvent) SetOutputFile(outputFile string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.outputFile = outputFile
}

// SetDefaultTime sets the time range to default values:
// start time is one hour before current time, end time is current time.
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

// BeforeGet prepares the input parameters before fetching log events.
// It validates the state and sets all necessary query parameters.
func (l *LogEvent) BeforeGet(input *awsr.LogEventInput) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.isInValid() {
		log.Fatalf("invalid log event state")
	}

	input.LogGroupName = l.logGroupName
	input.LogStreamNames = l.logStreamNames
	input.FilterPattern = l.filterPatern
	input.StartTime = time.Date(l.startYear, time.Month(l.startMonth), l.startDay, l.startHour, l.startMinute, 0, 0, time.Local)
	input.EndTime = time.Date(l.endYear, time.Month(l.endMonth), l.endDay, l.endHour, l.endMinute, 0, 0, time.Local)
}

// isInValid checks if the LogEvent state has invalid or missing required fields.
// Returns true if any time component is zero or log group name is empty.
func (l *LogEvent) isInValid() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if l.logGroupName == "" {
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

// SetTime updates a specific time component based on the widget label.
// It converts the text value to integer and updates the corresponding field.
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

// string2int converts a string to integer.
// It terminates the program if conversion fails.
func string2int(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		log.Fatalf("unable to list tables, %v", err)
	}
	return i
}

// Print displays the current log event settings in the provided text view.
// It shows log group, streams, filter pattern, time range, and other query parameters.
func (l *LogEvent) Print(textView *tview.TextView) {
	fmt.Fprintf(textView, "------------------------------------- \n")
	fmt.Fprintf(textView, "[YOUR SETTING]\n")
	fmt.Fprintf(textView, "LogGroup: %s\n", l.logGroupName)
	if len(l.logStreamNames) == 0 {
		fmt.Fprintf(textView, "LogStreams: %s\n", "ALL")
	} else {
		fmt.Fprintf(textView, "LogStreams: %s\n", l.logStreamNames)
	}
	fmt.Fprintf(textView, "LogStreams: %s\n", l.logStreamNames)
	fmt.Fprintf(textView, "FilterPaterm: %s\n", l.filterPatern)

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

	fmt.Fprintf(textView, "MaxEvent: %s\n", "1000")
	fmt.Fprintf(textView, "You can get all log events by pressing 'Save Button'.\n")
	fmt.Fprintf(textView, "------------------------------------- \n")
}
