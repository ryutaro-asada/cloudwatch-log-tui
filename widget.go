package main

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Widget represents UI components
type Widget int

const (
	// Log group widgets
	LogGroupTable Widget = iota
	LogGroupSearch

	// Log stream widgets
	LogStreamTable
	LogStreamSearch

	// Log event form widgets
	StartYearDropDown
	StartMonthDropDown
	StartDayDropDown
	StartHourDropDown
	StartMinuteDropDown
	EndYearDropDown
	EndMonthDropDown
	EndDayDropDown
	EndHourDropDown
	EndMinuteDropDown
	FilterPaternInput
	OutputFileInput
	SaveEventLogButton
	BackButton
	ViewLog
)

// Name mappings for enums
var (
	widgetNames = map[Widget]string{
		LogGroupTable:       "LogGroupTable",
		LogGroupSearch:      "LogGroupSearch",
		LogStreamTable:      "LogStreamTable",
		LogStreamSearch:     "LogStreamSearch",
		StartYearDropDown:   "StartYear",
		StartMonthDropDown:  "StartMonth",
		StartDayDropDown:    "StartDay",
		StartHourDropDown:   "StartHour",
		StartMinuteDropDown: "StartMinute",
		EndYearDropDown:     "EndYear",
		EndMonthDropDown:    "EndMonth",
		EndDayDropDown:      "EndDay",
		EndHourDropDown:     "EndHour",
		EndMinuteDropDown:   "EndMinute",
		FilterPaternInput:   "FilterPatern",
		OutputFileInput:     "OutputFile",
		SaveEventLogButton:  "SaveEventLog",
		BackButton:          "Back",
		ViewLog:             "ViewLog",
	}
)

func (g *gui) setLogEventWidget() {
	var months []string
	for i := 1; i <= 12; i++ {
		months = append(months, fmt.Sprintf("%d", i))
	}

	var days []string
	for i := 1; i <= 31; i++ {
		days = append(days, fmt.Sprintf("%d", i))
	}

	var hours []string
	for i := 0; i <= 23; i++ {
		hours = append(hours, fmt.Sprintf("%d", i))
	}
	var minutes []string
	for i := 0; i <= 59; i++ {
		minutes = append(minutes, fmt.Sprintf("%d", i))
	}

	formMap := map[Widget][]string{
		StartYearDropDown:   {"2024", "2025"},
		StartMonthDropDown:  months,
		StartDayDropDown:    days,
		StartHourDropDown:   hours,
		StartMinuteDropDown: minutes,
		EndYearDropDown:     {"2024", "2025"},
		EndMonthDropDown:    months,
		EndDayDropDown:      days,
		EndHourDropDown:     hours,
		EndMinuteDropDown:   minutes,
	}

	for key, value := range formMap {
		DropDown := tview.NewDropDown().
			SetLabel(widgetNames[key]).
			SetOptions(value, nil).
			SetFieldBackgroundColor(tcell.ColorGray)

		g.widgets[key] = DropDown
	}
	g.widgets[FilterPaternInput] = tview.NewInputField().SetLabel("Write Filter Pattern")

	g.widgets[OutputFileInput] = tview.NewInputField().SetLabel("Write Output File")

	g.widgets[SaveEventLogButton] = tview.NewButton("Save Button")
	g.widgets[BackButton] = tview.NewButton("Back Button")

	g.widgets[ViewLog] = tview.NewTextView()
}

func (g *gui) setLogGroupWidget() {
	table := tview.NewTable().
		SetSelectable(true, false).
		Select(0, 0).
		SetFixed(1, 1)

	table.SetTitle("Log Groups")
	table.SetTitleAlign(tview.AlignLeft)
	table.SetBorder(true)

	g.widgets[LogGroupTable] = table

	search := tview.NewInputField().SetLabel("Word")
	search.SetLabelWidth(6)
	search.SetTitle("Search for Log Groups")
	search.SetTitleAlign(tview.AlignLeft)
	search.SetBorder(true)
	search.SetFieldBackgroundColor(tcell.ColorGray)
	g.widgets[LogGroupSearch] = search
}

func (g *gui) setLogStreamWidget() {
	table := tview.NewTable().
		SetSelectable(true, false).
		Select(0, 0).
		SetFixed(1, 1)

	table.SetTitle("Log Streams")
	table.SetTitleAlign(tview.AlignLeft)
	table.SetBorder(true)
	g.widgets[LogStreamTable] = table

	search := tview.NewInputField().SetLabel("Word")
	search.SetLabelWidth(6)
	search.SetTitle("Search for Log Streams")
	search.SetTitleAlign(tview.AlignLeft)
	search.SetBorder(true)
	search.SetFieldBackgroundColor(tcell.ColorGray)
	g.widgets[LogStreamSearch] = search
}
