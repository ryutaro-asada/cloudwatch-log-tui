// Package state manages the application state for the CloudWatch Log TUI.
package state

import (
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	awsr "github.com/ryutaro-asada/cloudwatch-log-tui/internal/aws"
)

// Bound selects the start or the end of the log event time range.
type Bound int

const (
	// Start is the beginning of the time range.
	Start Bound = iota
	// End is the end of the time range. The whole end minute is included.
	End
)

// TimeField is one component of a DateTime.
type TimeField int

const (
	// Year is the DateTime year.
	Year TimeField = iota
	// Month is the DateTime month (1-12).
	Month
	// Day is the DateTime day of month (1-31).
	Day
	// Hour is the DateTime hour (0-23).
	Hour
	// Minute is the DateTime minute (0-59).
	Minute
)

// DateTime is a local date and time with minute precision.
type DateTime struct {
	Year, Month, Day, Hour, Minute int
}

// Get returns the value of field f.
func (d DateTime) Get(f TimeField) int {
	switch f {
	case Year:
		return d.Year
	case Month:
		return d.Month
	case Day:
		return d.Day
	case Hour:
		return d.Hour
	default:
		return d.Minute
	}
}

func (d *DateTime) set(f TimeField, v int) {
	switch f {
	case Year:
		d.Year = v
	case Month:
		d.Month = v
	case Day:
		d.Day = v
	case Hour:
		d.Hour = v
	case Minute:
		d.Minute = v
	}
}

func (d DateTime) time() time.Time {
	return time.Date(d.Year, time.Month(d.Month), d.Day, d.Hour, d.Minute, 0, 0, time.Local)
}

func (d DateTime) validate() error {
	switch {
	case d.Year < 1:
		return fmt.Errorf("invalid year %d", d.Year)
	case d.Month < 1 || d.Month > 12:
		return fmt.Errorf("invalid month %d", d.Month)
	case d.Day < 1 || d.Day > DaysIn(d.Year, d.Month):
		return fmt.Errorf("invalid date %04d-%02d-%02d", d.Year, d.Month, d.Day)
	case d.Hour < 0 || d.Hour > 23:
		return fmt.Errorf("invalid hour %d", d.Hour)
	case d.Minute < 0 || d.Minute > 59:
		return fmt.Errorf("invalid minute %d", d.Minute)
	}
	return nil
}

func dateTimeOf(t time.Time) DateTime {
	return DateTime{t.Year(), int(t.Month()), t.Day(), t.Hour(), t.Minute()}
}

// DaysIn returns the number of days in the given month (1-12) of year.
func DaysIn(year, month int) int {
	return time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// Order is the order in which log events are shown.
type Order int

const (
	// NewestFirst shows the newest events first. It is the default.
	NewestFirst Order = iota
	// OldestFirst shows the oldest events first.
	OldestFirst
)

// LogEvent manages the state for CloudWatch log events,
// including time range, selected streams, filtering options, and output settings.
type LogEvent struct {
	mu             sync.RWMutex
	times          [2]DateTime
	logGroupName   string
	logStreamNames []string
	filterPattern  string
	outputFile     string
	order          Order
}

// SetLogGroupSelected updates the log group from which log events will be
// fetched. Selecting a different group clears the selected log streams, which
// belong to the previous group.
func (l *LogEvent) SetLogGroupSelected(logGroupName string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.logGroupName != logGroupName {
		l.logGroupName = logGroupName
		l.logStreamNames = nil
	}
}

// GetLogStreamsSelected returns a copy of the selected log stream names.
func (l *LogEvent) GetLogStreamsSelected() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return slices.Clone(l.logStreamNames)
}

// ClearLogStreams deselects all log streams, so events of every stream in the
// log group are fetched.
func (l *LogEvent) ClearLogStreams() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.logStreamNames = nil
}

// AddLogStream adds a log stream to the selection. Adding a stream that is
// already selected does nothing. It returns an error when the selection is
// full (FilterLogEvents accepts at most awsr.MaxLogStreamNames streams).
func (l *LogEvent) AddLogStream(name string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if slices.Contains(l.logStreamNames, name) {
		return nil
	}
	if len(l.logStreamNames) >= awsr.MaxLogStreamNames {
		return fmt.Errorf("at most %d log streams can be selected", awsr.MaxLogStreamNames)
	}
	l.logStreamNames = append(l.logStreamNames, name)
	return nil
}

// SetFilterPattern updates the filter pattern used to search log events.
func (l *LogEvent) SetFilterPattern(filterPattern string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.filterPattern = filterPattern
}

// SetOutputFile sets the output file path where log events will be saved.
func (l *LogEvent) SetOutputFile(outputFile string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.outputFile = outputFile
}

// Order returns the requested event order.
func (l *LogEvent) Order() Order {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.order
}

// ToggleOrder switches between NewestFirst and OldestFirst and returns the
// new order.
func (l *LogEvent) ToggleOrder() Order {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.order == NewestFirst {
		l.order = OldestFirst
	} else {
		l.order = NewestFirst
	}
	return l.order
}

// SetDefaultTime sets the time range to the last hour, up to the current minute.
func (l *LogEvent) SetDefaultTime() {
	l.SetLast(time.Hour, time.Now())
}

// SetLast sets the time range to the duration d before now, at minute
// precision: from the minute of now-d through the whole minute of now.
func (l *LogEvent) SetLast(d time.Duration, now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now = now.Local()
	l.times[Start] = dateTimeOf(now.Add(-d))
	l.times[End] = dateTimeOf(now)
}

// Time returns the start or end of the time range.
func (l *LogEvent) Time(b Bound) DateTime {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.times[b]
}

// SetTime updates one component of the start or end time. When the year or
// month changes, the day is clamped to the length of the new month so it never
// rolls over into the next month.
func (l *LogEvent) SetTime(b Bound, f TimeField, v int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	t := &l.times[b]
	t.set(f, v)
	if f == Year || f == Month {
		if t.Month >= 1 && t.Month <= 12 {
			t.Day = min(t.Day, DaysIn(t.Year, t.Month))
		}
	}
}

// BeforeGet fills input with the current query parameters. It returns an
// error, and leaves input unusable, when the state cannot form a valid query.
// input.NewestFirst is false when NewestFirst is requested but the start time
// is before awsr.NewestFirstMinStart.
func (l *LogEvent) BeforeGet(input *awsr.LogEventInput) error {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if err := l.validate(); err != nil {
		return err
	}

	input.LogGroupName = l.logGroupName
	input.LogStreamNames = slices.Clone(l.logStreamNames)
	input.FilterPattern = l.filterPattern
	input.OutputFile = l.outputFile
	input.StartTime = l.times[Start].time()
	input.EndTime = l.times[End].time().Add(time.Minute - time.Millisecond)
	input.NewestFirst = l.order == NewestFirst && awsr.NewestFirstSupported(input.StartTime)
	return nil
}

// validate must be called with l.mu held.
func (l *LogEvent) validate() error {
	if l.logGroupName == "" {
		return errors.New("no log group selected")
	}
	if err := l.times[Start].validate(); err != nil {
		return fmt.Errorf("start time: %w", err)
	}
	if err := l.times[End].validate(); err != nil {
		return fmt.Errorf("end time: %w", err)
	}
	if start, end := l.times[Start].time(), l.times[End].time(); start.After(end) {
		return fmt.Errorf("start time %s is after end time %s",
			start.Format("2006-01-02 15:04"), end.Format("2006-01-02 15:04"))
	}
	return nil
}
