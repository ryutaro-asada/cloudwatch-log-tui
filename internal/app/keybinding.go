// Package app provides the main application logic for the CloudWatch Log TUI.
package app

import (
	"log"
	"strconv"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/ryutaro-asada/cloudwatch-log-tui/internal/state"
	"github.com/ryutaro-asada/cloudwatch-log-tui/internal/view"
)

// yearsShown is the number of years, up to the current one, offered in the
// year drop-downs.
const yearsShown = 10

// timePreset sets the time range to the last d when its key is pressed.
type timePreset struct {
	key   rune
	label string
	d     time.Duration
}

var timePresets = []timePreset{
	{'1', "5m", 5 * time.Minute},
	{'2', "15m", 15 * time.Minute},
	{'3', "1h", time.Hour},
	{'4', "3h", 3 * time.Hour},
	{'5', "12h", 12 * time.Hour},
	{'6', "1d", 24 * time.Hour},
	{'7', "7d", 7 * 24 * time.Hour},
}

// timeDropDown binds a drop-down to the time component it edits.
type timeDropDown struct {
	dd    *tview.DropDown
	bound state.Bound
	field state.TimeField
}

// timeDropDowns returns the time drop-downs in focus order.
func (a *App) timeDropDowns() []timeDropDown {
	w := a.view.Widgets.LogEvent
	return []timeDropDown{
		{w.StartYear, state.Start, state.Year},
		{w.StartMonth, state.Start, state.Month},
		{w.StartDay, state.Start, state.Day},
		{w.StartHour, state.Start, state.Hour},
		{w.StartMinute, state.Start, state.Minute},
		{w.EndYear, state.End, state.Year},
		{w.EndMonth, state.End, state.Month},
		{w.EndDay, state.End, state.Day},
		{w.EndHour, state.End, state.Hour},
		{w.EndMinute, state.End, state.Minute},
	}
}

func (a *App) dayDropDown(b state.Bound) *tview.DropDown {
	if b == state.Start {
		return a.view.Widgets.LogEvent.StartDay
	}
	return a.view.Widgets.LogEvent.EndDay
}

// setUpKeyBindings initializes keyboard shortcuts for all UI components.
// It configures navigation, selection, and action key bindings.
func (a *App) setUpKeyBindings() {
	a.setUpKeybindingLogGroup()
	a.setUpKeybindingLogStream()
	a.setUpKeybindingLogEvent()
}

// setUpKeybindingLogGroup configures keyboard shortcuts for the log group interface.
// It handles table navigation, search functionality, and group selection.
func (a *App) setUpKeybindingLogGroup() {
	lgTable := a.view.Widgets.LogGroup.Table
	lgSearch := a.view.Widgets.LogGroup.Search
	lsTable := a.view.Widgets.LogStream.Table

	lgTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch {
		case event.Key() == tcell.KeyRune && event.Rune() == '/':
			a.tvApp.SetFocus(lgSearch)
			return nil
		case event.Key() == tcell.KeyTab:
			a.tvApp.SetFocus(lsTable)
			return nil
		}
		return event
	})
	lgTable.SetSelectedFunc(func(row, _ int) {
		ref, ok := rowRefAt(lgTable, row)
		if !ok || ref.kind != rowItem {
			return
		}
		a.state.LogEvent.SetLogGroupSelected(ref.name)
		a.state.LogStream.SetLogGroupSelected(ref.name)
		// Drop the previous group's streams right away so they cannot be
		// selected for the new group while its streams are loading.
		a.logStreams = nil
		a.setLogStreamToGui()
		a.tvApp.SetFocus(lsTable)
		a.LoadLogStreams(state.Home)
	})
	lgTable.SetSelectionChangedFunc(func(row, _ int) {
		ref, _ := rowRefAt(lgTable, row)
		switch ref.kind {
		case rowNext:
			a.LoadLogGroups(state.Next)
		case rowPrev:
			a.LoadLogGroups(state.Prev)
		}
	})

	lgSearch.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter || key == tcell.KeyEscape {
			a.tvApp.SetFocus(lgTable)
		}
	})
	lgSearch.SetChangedFunc(func(pattern string) {
		a.state.LogGroup.SetFilterPattern(pattern)
		a.LoadLogGroups(state.Home)
	})
}

// setUpKeybindingLogStream configures keyboard shortcuts for the log stream interface.
// It handles stream selection and navigation to log events.
func (a *App) setUpKeybindingLogStream() {
	lsTable := a.view.Widgets.LogStream.Table
	lgTable := a.view.Widgets.LogGroup.Table

	lsTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			a.tvApp.SetFocus(lgTable)
			return nil
		}
		return event
	})

	lsTable.SetSelectedFunc(func(row, _ int) {
		ref, ok := rowRefAt(lsTable, row)
		if !ok {
			return
		}
		switch ref.kind {
		case rowAllStreams:
			a.state.LogEvent.ClearLogStreams()
		case rowItem:
			if err := a.state.LogEvent.AddLogStream(ref.name); err != nil {
				a.showError(err)
				return
			}
		default:
			return
		}
		a.setLogStreamToGui()
		a.openLogEvents()
	})

	lsTable.SetSelectionChangedFunc(func(row, _ int) {
		ref, _ := rowRefAt(lsTable, row)
		switch ref.kind {
		case rowNext:
			a.LoadLogStreams(state.Next)
		case rowPrev:
			a.LoadLogStreams(state.Prev)
		}
	})
}

func (a *App) openLogEvents() {
	a.state.LogEvent.SetDefaultTime()
	a.setDefaultDropDownLogEvents()
	a.view.Pages.SwitchToPage(view.PageNames[view.LogEventPage])
	a.tvApp.SetFocus(a.view.Widgets.LogEvent.StartYear)
	a.LoadLogEvents()
}

func (a *App) showLogGroupPage() {
	a.view.Pages.SwitchToPage(view.PageNames[view.LogGroupAndStreamPage])
	a.tvApp.SetFocus(a.view.Widgets.LogStream.Table)
}

// setUpKeybindingLogEvent configures keyboard shortcuts for the log event viewer.
// It manages date/time selection, filtering, and navigation between form elements.
func (a *App) setUpKeybindingLogEvent() {
	w := a.view.Widgets.LogEvent
	dds := a.timeDropDowns()

	for i, t := range dds {
		var nextWidget tview.Primitive = w.FilterPattern
		if i < len(dds)-1 {
			nextWidget = dds[i+1].dd
		}
		t.dd.SetInputCapture(a.dropDownInputCapture(t.dd, nextWidget))
		t.dd.SetSelectedFunc(a.timeSelected(t.bound, t.field))
	}

	w.FilterPattern.
		SetDoneFunc(func(key tcell.Key) {
			switch key {
			case tcell.KeyEnter:
				a.LoadLogEvents()
			case tcell.KeyTab:
				a.tvApp.SetFocus(w.OutputFile)
			}
		}).
		SetChangedFunc(func(text string) {
			a.state.LogEvent.SetFilterPattern(text)
		})

	w.OutputFile.
		SetDoneFunc(func(key tcell.Key) {
			if key == tcell.KeyTab {
				a.tvApp.SetFocus(w.SaveEventLog)
			}
		}).
		SetChangedFunc(func(text string) {
			a.state.LogEvent.SetOutputFile(text)
		})

	w.SaveEventLog.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			a.tvApp.SetFocus(w.Back)
			return nil
		}
		return event
	})
	w.SaveEventLog.SetSelectedFunc(a.SaveLogEvents)

	w.Back.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			a.tvApp.SetFocus(w.ViewLog)
			return nil
		}
		return event
	})
	w.Back.SetSelectedFunc(a.showLogGroupPage)

	w.ViewLog.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			a.tvApp.SetFocus(w.StartYear)
			return nil
		}
		return event
	})
	a.view.Layouts.LogEvent.SetInputCapture(a.logEventPageInputCapture)
}

// logEventPageInputCapture handles the keys that work anywhere on the log
// event page. It runs before the focused widget, so it leaves keys alone
// while the user types in an input field or picks from an open drop-down.
func (a *App) logEventPageInputCapture(event *tcell.EventKey) *tcell.EventKey {
	if event.Key() != tcell.KeyRune || !a.pageKeysEnabled() {
		return event
	}
	if event.Rune() == 'o' {
		a.state.LogEvent.ToggleOrder()
		a.LoadLogEvents()
		return nil
	}
	for _, p := range timePresets {
		if event.Rune() == p.key {
			a.applyTimePreset(p.d)
			return nil
		}
	}
	return event
}

func (a *App) applyTimePreset(d time.Duration) {
	a.state.LogEvent.SetLast(d, time.Now())
	a.setDefaultDropDownLogEvents()
	a.LoadLogEvents()
}

func (a *App) pageKeysEnabled() bool {
	if _, typing := a.tvApp.GetFocus().(*tview.InputField); typing {
		return false
	}
	for _, t := range a.timeDropDowns() {
		if t.dd.IsOpen() {
			return false
		}
	}
	return true
}

// dropDownInputCapture handles keys on a time drop-down. While the list is
// open, tview handles the keys itself (Enter selects, Esc closes) and j/k are
// mapped to Down/Up. While it is closed, j/k change the option directly, Tab
// moves to nextWidget and Esc goes back to the log group page.
func (a *App) dropDownInputCapture(dd *tview.DropDown, nextWidget tview.Primitive) func(*tcell.EventKey) *tcell.EventKey {
	return func(event *tcell.EventKey) *tcell.EventKey {
		isRune := func(r rune) bool { return event.Key() == tcell.KeyRune && event.Rune() == r }

		if dd.IsOpen() {
			switch {
			case isRune('j'):
				return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
			case isRune('k'):
				return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
			}
			return event
		}

		switch {
		case event.Key() == tcell.KeyEscape:
			a.showLogGroupPage()
		case event.Key() == tcell.KeyTab:
			a.tvApp.SetFocus(nextWidget)
		case isRune('j'):
			moveOption(dd, 1)
		case isRune('k'):
			moveOption(dd, -1)
		default:
			return event
		}
		return nil
	}
}

func moveOption(dd *tview.DropDown, delta int) {
	idx, _ := dd.GetCurrentOption()
	if next := idx + delta; next >= 0 && next < dd.GetOptionCount() {
		dd.SetCurrentOption(next)
	}
}

// timeSelected returns the selected handler of the drop-down for field f of
// bound b. It runs after tview has applied the new option, so the reload
// uses the new value.
func (a *App) timeSelected(b state.Bound, f state.TimeField) func(string, int) {
	return func(text string, index int) {
		if index < 0 {
			return
		}
		v, err := strconv.Atoi(text)
		if err != nil {
			log.Printf("unexpected drop-down option %q: %v", text, err)
			return
		}
		a.state.LogEvent.SetTime(b, f, v)
		if f == state.Year || f == state.Month {
			a.syncDayDropDown(b)
		}
		if !a.syncingWidgets {
			a.LoadLogEvents()
		}
	}
}

// syncDayDropDown offers the days of the selected month and selects the
// state's day, which SetTime has already clamped to that month.
func (a *App) syncDayDropDown(b state.Bound) {
	dd := a.dayDropDown(b)
	t := a.state.LogEvent.Time(b)
	if days := state.DaysIn(t.Year, t.Month); dd.GetOptionCount() != days {
		dd.SetOptions(view.NumberOptions(1, days), a.timeSelected(b, state.Day))
	}
	a.selectOptionQuietly(dd, state.Day, t.Day)
}

// optionIndex returns the index of value v in the drop-down for field f, or
// -1 when v is not offered. Every time drop-down lists consecutive numbers in
// ascending order, so the index follows from the first option.
func (a *App) optionIndex(dd *tview.DropDown, f state.TimeField, v int) int {
	first := 0
	switch f {
	case state.Year:
		first = a.firstYear
	case state.Month, state.Day:
		first = 1
	}
	if i := v - first; i >= 0 && i < dd.GetOptionCount() {
		return i
	}
	return -1
}

// selectOptionQuietly selects value v without reloading log events.
func (a *App) selectOptionQuietly(dd *tview.DropDown, f state.TimeField, v int) {
	prev := a.syncingWidgets
	a.syncingWidgets = true
	defer func() { a.syncingWidgets = prev }()

	dd.SetCurrentOption(a.optionIndex(dd, f, v))
}

// setDefaultDropDownLogEvents shows the state's time range in the drop-downs.
func (a *App) setDefaultDropDownLogEvents() {
	w := a.view.Widgets.LogEvent
	currentYear := time.Now().Year()
	a.firstYear = currentYear - yearsShown + 1
	years := view.NumberOptions(a.firstYear, currentYear)
	w.StartYear.SetOptions(years, a.timeSelected(state.Start, state.Year))
	w.EndYear.SetOptions(years, a.timeSelected(state.End, state.Year))

	for _, t := range a.timeDropDowns() {
		a.selectOptionQuietly(t.dd, t.field, a.state.LogEvent.Time(t.bound).Get(t.field))
	}
}
