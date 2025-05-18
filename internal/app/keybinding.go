package app

import (
	"fmt"
	"log"
	"reflect"
	"slices"
	"strconv"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/ryutaro-asada/cloudwatch-log-tui/internal/state"
	"github.com/ryutaro-asada/cloudwatch-log-tui/internal/view"
)

// setupKeyBindings initializes the keyboard shortcuts
func (a *App) setUpKeyBindings() {
	a.setUpKeybindingLogGroup()
	a.setUpKeybindingLogStream()
	a.setUpKeybindingLogEvent()
}

func (a *App) setUpKeybindingLogGroup() {
	lgTable := a.view.Widgets.LogGroup.Table
	lgSearch := a.view.Widgets.LogGroup.Search
	lsTable := a.view.Widgets.LogStream.Table

	// log group table
	lgTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		row, _ := lgTable.GetSelection()
		max := lgTable.GetRowCount()

		switch event.Rune() {
		case 'k', 'j':
			// up/down
			lgTable.Select(row%max, 0)
		case '/':
			a.tvApp.SetFocus(lgSearch)
		}

		if event.Key() == tcell.KeyTab {
			a.tvApp.SetFocus(lsTable)
		}

		return event
	})
	lgTable.SetSelectedFunc(func(row, _ int) {
		cell := lgTable.GetCell(row, 0)
		groupName := cell.Text

		a.state.LogEvent.SetLogGroupSelected(groupName)
		a.tvApp.SetFocus(lsTable)
		a.state.LogStream.SetLogGroupSelected(groupName)
		a.LoadLogStreams(state.Home)
	})

	// Search form
	lgTable.SetSelectionChangedFunc(func(row, _ int) {
		cell := lgTable.GetCell(row, 0)
		switch cell.Text {
		case NextPage:
			a.LoadLogGroups(state.Next)
		case PrevPage:
			a.LoadLogGroups(state.Prev)
		}
	})
	lgSearch.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			a.tvApp.SetFocus(lgTable)
		}
	})
	lgSearch.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			a.tvApp.SetFocus(lgTable)
		}
		return event
	})
	lgSearch.SetChangedFunc(func(pattern string) {
		a.state.LogGroup.SetFilterPattern(pattern)
		a.LoadLogGroups(state.Home)
	})
}

func (a *App) setUpKeybindingLogStream() {
	lsTable := a.view.Widgets.LogStream.Table
	lsSearch := a.view.Widgets.LogStream.Search
	lgTable := a.view.Widgets.LogGroup.Table

	lsTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		max := lsTable.GetRowCount()
		row, _ := lsTable.GetSelection()
		switch event.Rune() {
		case 'k', 'j':
			// up/down
			lsTable.Select(row%max, 0)

		case '/':
			a.tvApp.SetFocus(lsSearch)
		// Space key
		case ' ':

			ls := lsTable.GetCell(row, 1).Text
			if ls == "All Log Streams" {
				return nil
			}
			lsTable.Clear()

			lsSelected := a.state.LogEvent.GetLogStreamsSelected()
			// check if log stream is selected
			switch slices.Contains(lsSelected, ls) {
			case true:
				// set log stream as unselected
				i := slices.Index(lsSelected, ls)
				a.state.LogEvent.SetLogStreamsSelected(slices.Delete(lsSelected, i, i+1))
			case false:
				// set log stream as selected
				a.state.LogEvent.SetLogStreamsSelected(append(lsSelected, ls))
			}

			a.refreshLogStreamTable()

			return nil
		}

		if event.Key() == tcell.KeyTab {
			a.tvApp.SetFocus(lgTable)
		}

		return event
	})

	// when table is selected (enter key pressed)
	lsTable.SetSelectedFunc(func(row, col int) {
		cell := lsTable.GetCell(row, 1)
		logStreamName := cell.Text
		if logStreamName == "All Log Streams" {
			a.state.LogEvent.SetLogStreamsSelected([]string{})
		} else {
			a.state.LogEvent.SetLogStreamsSelected(
				slices.Compact(
					append(a.state.LogEvent.GetLogStreamsSelected(),
						logStreamName)))
		}
		a.state.LogEvent.SetDefaultTime()
		a.setDefaultDropDownLogEvents()
		a.LoadLogEvents()
		a.view.Pages.SwitchToPage(view.PageNames[view.LogEventPage])
		a.tvApp.SetFocus(a.view.Widgets.LogEvent.StartYear)
	})

	// when table is focused
	lsTable.SetSelectionChangedFunc(func(row, column int) {
		cell := lsTable.GetCell(row, 1)
		switch cell.Text {
		case NextPage:
			a.LoadLogStreams(state.Next)
		case PrevPage:
			a.LoadLogStreams(state.Prev)
		}
	})

	// Search form
	lsSearch.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			a.tvApp.SetFocus(lsTable)
		}
	})
	lsSearch.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			a.tvApp.SetFocus(lsTable)
		}
		return event
	})
	lsSearch.SetChangedFunc(func(prefixPatern string) {
		a.state.LogStream.SetPrefixPattern(prefixPatern)
		a.LoadLogStreams(state.Home)
	})
}

func (a *App) setUpKeybindingLogEvent() {
	dds := []*tview.DropDown{
		a.view.Widgets.LogEvent.StartYear,
		a.view.Widgets.LogEvent.StartMonth,
		a.view.Widgets.LogEvent.StartDay,
		a.view.Widgets.LogEvent.StartHour,
		a.view.Widgets.LogEvent.StartMinute,
		a.view.Widgets.LogEvent.EndYear,
		a.view.Widgets.LogEvent.EndMonth,
		a.view.Widgets.LogEvent.EndDay,
		a.view.Widgets.LogEvent.EndHour,
		a.view.Widgets.LogEvent.EndMinute,
	}

	for i, dd := range dds {
		currentD := dd
		currentL := currentD.GetLabel()

		var nextWidget tview.Primitive
		if i == len(dds)-1 {
			nextWidget = a.view.Widgets.LogEvent.FilterPatern
		} else {
			nextWidget = dds[i+1]
		}
		currentD.
			SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
				switch event.Rune() {
				case 'k':
					// up
					max := currentD.GetOptionCount()
					idx, _ := currentD.GetCurrentOption()
					if idx >= 1 {
						currentD.SetCurrentOption((idx - 1) % max)
					}

				case 'j':
					// down
					max := currentD.GetOptionCount()
					idx, _ := currentD.GetCurrentOption()
					if idx < max-1 {
						currentD.SetCurrentOption((idx + 1) % max)
					}
				}

				if event.Key() == tcell.KeyEsc {
					a.view.Pages.SwitchToPage(view.PageNames[view.LogGroupAndStreamPage])
					a.tvApp.SetFocus(a.view.Widgets.LogStream.Table)
				} else if event.Key() == tcell.KeyTab {
					a.tvApp.SetFocus(nextWidget)
				} else if event.Key() == tcell.KeyEnter {
					if currentD.IsOpen() {
						if i, _ := currentD.GetCurrentOption(); i != -1 {
							a.LoadLogEvents()
						}
					}
				}
				return event
			})

		switch currentL {
		case view.WidgetNames[view.StartMonthDropDown]:
			currentD.SetSelectedFunc(func(text string, index int) {
				a.state.LogEvent.SetTime(currentL, text)

				a.view.Widgets.LogEvent.StartDay.
					SetOptions(getDaysByMonth(text), nil).
					SetSelectedFunc(func(text string, index int) {
						a.state.LogEvent.SetTime(view.WidgetNames[view.StartDayDropDown], text)
					})
			})
		case view.WidgetNames[view.EndMonthDropDown]:
			currentD.SetSelectedFunc(func(text string, index int) {
				a.state.LogEvent.SetTime(currentL, text)

				a.view.Widgets.LogEvent.EndDay.
					SetOptions(getDaysByMonth(text), nil).
					SetSelectedFunc(func(text string, index int) {
						a.state.LogEvent.SetTime(view.WidgetNames[view.EndDayDropDown], text)
					})
			})
		default:
			currentD.SetSelectedFunc(func(text string, index int) {
				a.state.LogEvent.SetTime(currentL, text)
			})
		}
	}

	a.view.Widgets.LogEvent.FilterPatern.
		SetDoneFunc(func(key tcell.Key) {
			if key == tcell.KeyTab {
				a.tvApp.SetFocus(a.view.Widgets.LogEvent.OutputFile)
			}
		}).
		SetChangedFunc(func(text string) {
			a.state.LogEvent.SetFilterPatern(text)
			a.LoadLogEvents()
		})

	a.view.Widgets.LogEvent.OutputFile.
		SetDoneFunc(func(key tcell.Key) {
			if key == tcell.KeyTab {
				a.tvApp.SetFocus(a.view.Widgets.LogEvent.SaveEventLog)
			}
		}).
		SetChangedFunc(func(text string) {
			a.state.LogEvent.SetOutputFile(text)
		})

	saveButton := a.view.Widgets.LogEvent.SaveEventLog
	saveButton.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			a.tvApp.SetFocus(a.view.Widgets.LogEvent.Back)
		}
		return event
	})
	saveButton.SetSelectedFunc(func() {
		// a.SaveLogEvents()
	})

	backButton := a.view.Widgets.LogEvent.Back
	backButton.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			a.tvApp.SetFocus(a.view.Widgets.LogEvent.ViewLog)
		}
		return event
	})
	backButton.SetSelectedFunc(func() {
		a.view.Pages.SwitchToPage(view.PageNames[view.LogGroupAndStreamPage])
		a.tvApp.SetFocus(a.view.Widgets.LogStream.Table)
	})

	viewLog := a.view.Widgets.LogEvent.ViewLog
	viewLog.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			a.tvApp.SetFocus(a.view.Widgets.LogEvent.StartYear)
		}
		return event
	})
	viewLog.SetScrollable(true)
}

func PrintStructFields(s interface{}) []string {
	typ := reflect.TypeOf(s)
	val := reflect.ValueOf(s)

	if typ.Kind() != reflect.Struct {
		return []string{"Provided data is not a struct."}
	}

	list := make([]string, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		list[i] = fmt.Sprintf("%s: %v", typ.Field(i).Name, val.Field(i))
	}
	return list
}

func string2int(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		log.Fatalf("unable to list tables, %v", err)
	}
	return i
}

func string2month(s string) time.Month {
	i, err := strconv.Atoi(s)
	if err != nil {
		log.Fatalf("unable to list tables, %v", err)
	}
	return time.Month(i)
}

func getDaysByMonth(month string) []string {
	var days []string
	y := 2024
	intMonth, err := strconv.Atoi(month)
	if err != nil {
		log.Fatalf("unable to convert month to integer, %v", err)
	}
	m := time.Month(intMonth)

	// Start from the first day of the month
	startDate := time.Date(y, m, 1, 0, 0, 0, 0, time.UTC)
	// Get the number of days in the month
	for d := startDate; d.Month() == m; d = d.AddDate(0, 0, 1) {
		// Convert day to a string without leading zero
		dayStr := strconv.Itoa(d.Day())
		days = append(days, dayStr)
	}
	return days
}
