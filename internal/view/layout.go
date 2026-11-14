// Package view manages the user interface components for the CloudWatch Log TUI.
package view

import (
	"github.com/rivo/tview"
)

// Layouts manages the different layout configurations for the application.
// It contains the main layouts for log group/stream selection and log event viewing.
type Layouts struct {
	LogGroupAndStream *tview.Flex
	LogEvent          *tview.Grid
}

// setUp initializes all layouts with their respective widget configurations.
func (l *Layouts) setUp(w *Widgets) {
	l.setUpLayoutLogGroupAndStream(w)
	l.setUpLayoutLogEvent(w)
}

// setUpLayoutLogGroupAndStream creates the layout for log group and stream selection.
// It arranges tables and search fields in a split view configuration.
func (l *Layouts) setUpLayoutLogGroupAndStream(w *Widgets) {
	l.LogGroupAndStream = tview.NewFlex().
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(w.LogGroup.Table, 0, 20, false).
			AddItem(w.LogGroup.Search, 0, 1, false), 0, 10, true).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(w.LogStream.Table, 0, 1, true), 0, 10, false)
}

// setUpLayoutLogEvent creates the grid layout for the log event viewer.
// It arranges date/time selectors, filter options, and the log display area.
func (l *Layouts) setUpLayoutLogEvent(w *Widgets) {
	l.LogEvent = tview.NewGrid().
		SetRows(
			// drop down options, hints
			1, 1, 1, 1,
			// text view
			0).
		SetColumns(0, 0, 0, 0, 0).
		SetBorders(true).
		// start date
		AddItem(w.LogEvent.StartYear,
			0, 0, // row, column position
			1, 1, // rowSpan, columnSpan
			0, 0, // minHeight, minWidth
			false). // focusable
		AddItem(w.LogEvent.StartMonth,
			0, 1,
			1, 1,
			0, 0,
			false).
		AddItem(w.LogEvent.StartDay,
			0, 2,
			1, 1,
			0, 0,
			false).
		AddItem(w.LogEvent.StartHour,
			0, 3,
			1, 1,
			0, 0,
			false).
		AddItem(w.LogEvent.StartMinute,
			0, 4,
			1, 1,
			0, 0,
			false).
		// end date
		AddItem(w.LogEvent.EndYear,
			1, 0,
			1, 1,
			0, 0,
			false).
		AddItem(w.LogEvent.EndMonth,
			1, 1,
			1, 1,
			0, 0,
			false).
		AddItem(w.LogEvent.EndDay,
			1, 2,
			1, 1,
			0, 0,
			false).
		AddItem(w.LogEvent.EndHour,
			1, 3,
			1, 1,
			0, 0,
			false).
		AddItem(w.LogEvent.EndMinute,
			1, 4,
			1, 1,
			0, 0,
			false).
		// additional input
		AddItem(w.LogEvent.FilterPattern,
			2, 0,
			1, 1,
			0, 0,
			false).
		AddItem(w.LogEvent.OutputFile,
			2, 1,
			1, 1,
			0, 0,
			false).
		AddItem(w.LogEvent.SaveEventLog,
			2, 2,
			1, 1,
			0, 0,
			false).
		AddItem(w.LogEvent.Back,
			2, 3,
			1, 1,
			0, 0,
			false).
		AddItem(w.LogEvent.Hints,
			3, 0,
			1, 5,
			0, 0,
			false).
		// Log View
		AddItem(w.LogEvent.ViewLog,
			4, 0,
			1, 5,
			0, 0,
			false)
}
