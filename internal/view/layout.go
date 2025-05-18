package view

import (
	"github.com/rivo/tview"
)

// Layout represents different UI layouts
type Layout int

const (
	LogGroupLayout Layout = iota
	LogStreamLayout
	LogEventLayout
)

var LayoutNames = map[Layout]string{
	LogGroupLayout:  "LogGroup",
	LogStreamLayout: "LogStream",
	LogEventLayout:  "LogEvent",
}

type Layouts struct {
	LogGroupAndStream *tview.Flex
	LogEvent          *tview.Grid
}

func (l *Layouts) setUp(w *Widgets) {
	l.setUpLayoutLogGroupAndStream(w)
	l.setUpLayoutLogEvent(w)
}

func (l *Layouts) setUpLayoutLogGroupAndStream(w *Widgets) {
	l.LogGroupAndStream = tview.NewFlex().
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(w.LogGroup.Table, 0, 30, false).
			AddItem(w.LogGroup.Search, 0, 1, false), 0, 10, true).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(w.LogStream.Table, 0, 30, true).
			AddItem(w.LogStream.Search, 0, 1, false), 0, 10, false)
}

func (l *Layouts) setUpLayoutLogEvent(w *Widgets) {
	l.LogEvent = tview.NewGrid().
		SetRows(
			// drop down options
			1, 1, 1,
			// text view
			0).
		SetColumns(0, 0, 0, 0, 0).
		SetBorders(true).
		// start date
		AddItem(w.LogEvent.StartYear,
			0, 0, // row, column position
			1, 1, // rowSpan, columnSpan
			0, 100, // minHeight, minWidth
			false). // focusable
		AddItem(w.LogEvent.StartMonth,
			0, 1,
			1, 1,
			0, 100,
			false).
		AddItem(w.LogEvent.StartDay,
			0, 2,
			1, 1,
			0, 100,
			false).
		AddItem(w.LogEvent.StartHour,
			0, 3,
			1, 1,
			0, 100,
			false).
		AddItem(w.LogEvent.StartMinute,
			0, 4,
			1, 1,
			0, 100,
			false).
		// end date
		AddItem(w.LogEvent.EndYear,
			1, 0,
			1, 1,
			0, 100,
			false).
		AddItem(w.LogEvent.EndMonth,
			1, 1,
			1, 1,
			0, 100,
			false).
		AddItem(w.LogEvent.EndDay,
			1, 2,
			1, 1,
			0, 100,
			false).
		AddItem(w.LogEvent.EndHour,
			1, 3,
			1, 1,
			0, 100,
			false).
		AddItem(w.LogEvent.EndMinute,
			1, 4,
			1, 1,
			0, 100,
			false).
		// aditional input
		AddItem(w.LogEvent.FilterPatern,
			2, 0,
			1, 1,
			0, 100,
			false).
		AddItem(w.LogEvent.OutputFile,
			2, 1,
			1, 1,
			0, 100,
			false).
		AddItem(w.LogEvent.SaveEventLog,
			2, 2,
			1, 1,
			0, 100,
			false).
		AddItem(w.LogEvent.Back,
			2, 3,
			1, 1,
			0, 100,
			false).
		// Log View
		AddItem(w.LogEvent.ViewLog,
			3, 0,
			1, 5,
			0, 100,
			false)
}
