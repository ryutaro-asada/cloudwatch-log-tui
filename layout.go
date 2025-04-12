package main

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

func (g *gui) setLogGroupLayout() {
	g.setLogGroupWidget()
	g.setLogStreamWidget()
	g.layouts[LogGroupLayout] = tview.NewFlex().
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(g.widgets[LogGroupTable], 0, 30, false).
			AddItem(g.widgets[LogGroupSearch], 0, 1, false), 0, 10, true).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(g.widgets[LogStreamTable], 0, 30, true).
			AddItem(g.widgets[LogStreamSearch], 0, 1, false), 0, 10, false)
}

func (g *gui) setLogEventLayout() {
	g.setLogEventWidget()
	g.layouts[LogEventLayout] = tview.NewGrid().
		SetRows(
			// drop down options
			1, 1, 1,
			// text view
			0).
		SetColumns(0, 0, 0, 0, 0).
		SetBorders(true).
		// start date
		AddItem(g.widgets[StartYearDropDown],
			0, 0, // row, column position
			1, 1, // rowSpan, columnSpan
			0, 100, // minHeight, minWidth
			false). // focusable
		AddItem(g.widgets[StartMonthDropDown],
			0, 1,
			1, 1,
			0, 100,
			false).
		AddItem(g.widgets[StartDayDropDown],
			0, 2,
			1, 1,
			0, 100,
			false).
		AddItem(g.widgets[StartHourDropDown],
			0, 3,
			1, 1,
			0, 100,
			false).
		AddItem(g.widgets[StartMinuteDropDown],
			0, 4,
			1, 1,
			0, 100,
			false).
		// end date
		AddItem(g.widgets[EndYearDropDown],
			1, 0,
			1, 1,
			0, 100,
			false).
		AddItem(g.widgets[EndMonthDropDown],
			1, 1,
			1, 1,
			0, 100,
			false).
		AddItem(g.widgets[EndDayDropDown],
			1, 2,
			1, 1,
			0, 100,
			false).
		AddItem(g.widgets[EndHourDropDown],
			1, 3,
			1, 1,
			0, 100,
			false).
		AddItem(g.widgets[EndMinuteDropDown],
			1, 4,
			1, 1,
			0, 100,
			false).
		// aditional input
		AddItem(g.widgets[FilterPaternInput],
			2, 0,
			1, 1,
			0, 100,
			false).
		AddItem(g.widgets[OutputFileInput],
			2, 1,
			1, 1,
			0, 100,
			false).
		AddItem(g.widgets[SaveEventLogButton],
			2, 2,
			1, 1,
			0, 100,
			false).
		AddItem(g.widgets[BackButton],
			2, 3,
			1, 1,
			0, 100,
			false).
		// Log View
		AddItem(g.widgets[ViewLog],
			3, 0,
			1, 5,
			0, 100,
			false)
}
