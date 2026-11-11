// Package view manages the user interface components for the CloudWatch Log TUI.
package view

import (
	"strconv"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Widgets contains all UI widget components organized by feature area.
type Widgets struct {
	LogGroup  logGroupWidget
	LogStream logStreamWidget
	LogEvent  logEventWidget
	// Header is a one-line area shown above every page.
	Header *tview.TextView
	// Status is a one-line message area shown below every page.
	Status *tview.TextView
}

type logGroupWidget struct {
	Table  *tview.Table
	Search *tview.InputField
}
type logStreamWidget struct {
	Table *tview.Table
}
type logEventWidget struct {
	StartYear     *tview.DropDown
	StartMonth    *tview.DropDown
	StartDay      *tview.DropDown
	StartHour     *tview.DropDown
	StartMinute   *tview.DropDown
	EndYear       *tview.DropDown
	EndMonth      *tview.DropDown
	EndDay        *tview.DropDown
	EndHour       *tview.DropDown
	EndMinute     *tview.DropDown
	FilterPattern *tview.InputField
	OutputFile    *tview.InputField
	SaveEventLog  *tview.Button
	Back          *tview.Button
	// Hints shows the event order and the keys that change the query.
	Hints   *tview.TextView
	ViewLog *tview.TextView
}

// NumberOptions returns the decimal strings from..to (inclusive), for use as
// drop-down options.
func NumberOptions(from, to int) []string {
	var opts []string
	for i := from; i <= to; i++ {
		opts = append(opts, strconv.Itoa(i))
	}
	return opts
}

// setUp initializes all widget groups with their default configurations.
func (w *Widgets) setUp() {
	w.LogGroup.setUp()
	w.LogStream.setUp()
	w.LogEvent.setUp()
	w.Header = tview.NewTextView().SetDynamicColors(true)
	w.Status = tview.NewTextView()
}

// setUp initializes the log group widget with a table and search field.
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

// setUp initializes the log stream widget with a selectable table.
func (l *logStreamWidget) setUp() {
	table := tview.NewTable().
		SetSelectable(true, false).
		Select(0, 0).
		SetFixed(1, 1)

	table.SetTitle("Log Streams")
	table.SetTitleAlign(tview.AlignLeft)
	table.SetBorder(true)
	l.Table = table
}

func newDropDown(label string, options []string) *tview.DropDown {
	return tview.NewDropDown().
		SetLabel(label).
		SetOptions(options, nil).
		SetFieldBackgroundColor(tcell.ColorGray)
}

// setUp initializes the log event widget with date/time dropdowns,
// filter inputs, action buttons, and a text viewer for log display.
// The year and day options depend on the selected date and are set by the app.
func (l *logEventWidget) setUp() {
	months := NumberOptions(1, 12)
	hours := NumberOptions(0, 23)
	minutes := NumberOptions(0, 59)

	l.StartYear = newDropDown("StartYear", nil)
	l.StartMonth = newDropDown("StartMonth", months)
	l.StartDay = newDropDown("StartDay", nil)
	l.StartHour = newDropDown("StartHour", hours)
	l.StartMinute = newDropDown("StartMinute", minutes)
	l.EndYear = newDropDown("EndYear", nil)
	l.EndMonth = newDropDown("EndMonth", months)
	l.EndDay = newDropDown("EndDay", nil)
	l.EndHour = newDropDown("EndHour", hours)
	l.EndMinute = newDropDown("EndMinute", minutes)

	l.FilterPattern = tview.NewInputField().SetLabel("Filter Pattern (Enter)")
	l.OutputFile = tview.NewInputField().SetLabel("Write Output File")

	l.SaveEventLog = tview.NewButton("Save Button")
	l.Back = tview.NewButton("Back Button")

	l.Hints = tview.NewTextView().SetDynamicColors(true)
	l.ViewLog = tview.NewTextView().SetScrollable(true)
}
