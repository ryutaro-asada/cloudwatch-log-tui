package ui

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

var layoutNames = map[Layout]string{
	LogGroupLayout:  "LogGroup",
	LogStreamLayout: "LogStream",
	LogEventLayout:  "LogEvent",
}

func (a *App) setUpLayouts() {
	a.setUpLayoutLogGroupAndStream()
	a.setUpLayoutLogEvent()
}

func (a *App) setUpLayoutLogGroupAndStream() {
	a.layouts[LogGroupLayout] = tview.NewFlex().
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(a.widgets[LogGroupTable], 0, 30, false).
			AddItem(a.widgets[LogGroupSearch], 0, 1, false), 0, 10, true).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(a.widgets[LogStreamTable], 0, 30, true).
			AddItem(a.widgets[LogStreamSearch], 0, 1, false), 0, 10, false)
}

func (a *App) setUpLayoutLogEvent() {
	a.layouts[LogEventLayout] = tview.NewGrid().
		SetRows(
			// drop down options
			1, 1, 1,
			// text view
			0).
		SetColumns(0, 0, 0, 0, 0).
		SetBorders(true).
		// start date
		AddItem(a.widgets[StartYearDropDown],
			0, 0, // row, column position
			1, 1, // rowSpan, columnSpan
			0, 100, // minHeight, minWidth
			false). // focusable
		AddItem(a.widgets[StartMonthDropDown],
			0, 1,
			1, 1,
			0, 100,
			false).
		AddItem(a.widgets[StartDayDropDown],
			0, 2,
			1, 1,
			0, 100,
			false).
		AddItem(a.widgets[StartHourDropDown],
			0, 3,
			1, 1,
			0, 100,
			false).
		AddItem(a.widgets[StartMinuteDropDown],
			0, 4,
			1, 1,
			0, 100,
			false).
		// end date
		AddItem(a.widgets[EndYearDropDown],
			1, 0,
			1, 1,
			0, 100,
			false).
		AddItem(a.widgets[EndMonthDropDown],
			1, 1,
			1, 1,
			0, 100,
			false).
		AddItem(a.widgets[EndDayDropDown],
			1, 2,
			1, 1,
			0, 100,
			false).
		AddItem(a.widgets[EndHourDropDown],
			1, 3,
			1, 1,
			0, 100,
			false).
		AddItem(a.widgets[EndMinuteDropDown],
			1, 4,
			1, 1,
			0, 100,
			false).
		// aditional input
		AddItem(a.widgets[FilterPaternInput],
			2, 0,
			1, 1,
			0, 100,
			false).
		AddItem(a.widgets[OutputFileInput],
			2, 1,
			1, 1,
			0, 100,
			false).
		AddItem(a.widgets[SaveEventLogButton],
			2, 2,
			1, 1,
			0, 100,
			false).
		AddItem(a.widgets[BackButton],
			2, 3,
			1, 1,
			0, 100,
			false).
		// Log View
		AddItem(a.widgets[ViewLog],
			3, 0,
			1, 5,
			0, 100,
			false)
}
