package view

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
	FilterPatternInput
	OutputFileInput
	SaveEventLogButton
	BackButton
	ViewLog
)

// Name mappings for enums
var (
	WidgetNames = map[Widget]string{
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

type Widgets struct {
	LogGroup  logGroupWidget
	LogStream logStreamWidget
	LogEvent  logEventWidget
}

type logGroupWidget struct {
	Table  *tview.Table
	Search *tview.InputField
}
type logStreamWidget struct {
	Table  *tview.Table
	Search *tview.InputField
}
type logEventWidget struct {
	StartYear    *tview.DropDown
	StartMonth   *tview.DropDown
	StartDay     *tview.DropDown
	StartHour    *tview.DropDown
	StartMinute  *tview.DropDown
	EndYear      *tview.DropDown
	EndMonth     *tview.DropDown
	EndDay       *tview.DropDown
	EndHour      *tview.DropDown
	EndMinute    *tview.DropDown
	FilterPatern *tview.InputField
	OutputFile   *tview.InputField
	SaveEventLog *tview.Button
	Back         *tview.Button
	ViewLog      *tview.TextView
}

func (w *Widgets) setUp() {
	w.LogGroup.setUp()
	w.LogStream.setUp()
	w.LogEvent.setUp()
}

func (l *logGroupWidget) setUp() {
	table := tview.NewTable().
		SetSelectable(true, false).
		Select(0, 0).
		SetFixed(1, 1)

	table.SetTitle("Log Groups")
	table.SetTitleAlign(tview.AlignLeft)
	table.SetBorder(true)

	l.Table = table

	search := tview.NewInputField().SetLabel("Word")
	search.SetLabelWidth(6)
	search.SetTitle("Search for Log Groups")
	search.SetTitleAlign(tview.AlignLeft)
	search.SetBorder(true)
	search.SetFieldBackgroundColor(tcell.ColorGray)
	l.Search = search
}

func (l *logStreamWidget) setUp() {
	table := tview.NewTable().
		SetSelectable(true, false).
		Select(0, 0).
		SetFixed(1, 1)

	table.SetTitle("Log Streams")
	table.SetTitleAlign(tview.AlignLeft)
	table.SetBorder(true)
	l.Table = table

	search := tview.NewInputField().SetLabel("Word")
	search.SetLabelWidth(6)
	search.SetTitle("Search for Log Streams")
	search.SetTitleAlign(tview.AlignLeft)
	search.SetBorder(true)
	search.SetFieldBackgroundColor(tcell.ColorGray)
	l.Search = search
}

func dropDownOptions() map[Widget][]string {
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

	return map[Widget][]string{
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
}

func (l *logEventWidget) setUp() {
	optons := dropDownOptions()
	l.StartYear = tview.NewDropDown().
		SetLabel(WidgetNames[StartYearDropDown]).
		SetOptions(optons[StartYearDropDown], nil).
		SetFieldBackgroundColor(tcell.ColorGray)
	l.StartMonth = tview.NewDropDown().
		SetLabel(WidgetNames[StartMonthDropDown]).
		SetOptions(optons[StartMonthDropDown], nil).
		SetFieldBackgroundColor(tcell.ColorGray)
	l.StartDay = tview.NewDropDown().
		SetLabel(WidgetNames[StartDayDropDown]).
		SetOptions(optons[StartDayDropDown], nil).
		SetFieldBackgroundColor(tcell.ColorGray)
	l.StartHour = tview.NewDropDown().
		SetLabel(WidgetNames[StartHourDropDown]).
		SetOptions(optons[StartHourDropDown], nil).
		SetFieldBackgroundColor(tcell.ColorGray)
	l.StartMinute = tview.NewDropDown().
		SetLabel(WidgetNames[StartMinuteDropDown]).
		SetOptions(optons[StartMinuteDropDown], nil).
		SetFieldBackgroundColor(tcell.ColorGray)
	l.EndYear = tview.NewDropDown().
		SetLabel(WidgetNames[EndYearDropDown]).
		SetOptions(optons[EndYearDropDown], nil).
		SetFieldBackgroundColor(tcell.ColorGray)
	l.EndMonth = tview.NewDropDown().
		SetLabel(WidgetNames[EndMonthDropDown]).
		SetOptions(optons[EndMonthDropDown], nil).
		SetFieldBackgroundColor(tcell.ColorGray)
	l.EndDay = tview.NewDropDown().
		SetLabel(WidgetNames[EndDayDropDown]).
		SetOptions(optons[EndDayDropDown], nil).
		SetFieldBackgroundColor(tcell.ColorGray)
	l.EndHour = tview.NewDropDown().
		SetLabel(WidgetNames[EndHourDropDown]).
		SetOptions(optons[EndHourDropDown], nil).
		SetFieldBackgroundColor(tcell.ColorGray)
	l.EndMinute = tview.NewDropDown().
		SetLabel(WidgetNames[EndMinuteDropDown]).
		SetOptions(optons[EndMinuteDropDown], nil).
		SetFieldBackgroundColor(tcell.ColorGray)

	l.FilterPatern = tview.NewInputField().SetLabel("Write Filter Pattern")
	l.OutputFile = tview.NewInputField().SetLabel("Write Output File")

	l.SaveEventLog = tview.NewButton("Save Button")
	l.Back = tview.NewButton("Back Button")

	l.ViewLog = tview.NewTextView()
}
