package ui

import (
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	cwl "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/ryutaro-asada/cloudwatch-log-tui/internal/state"
)

// setupKeyBindings initializes the keyboard shortcuts
func (a *App) setUpKeyBindings() {
	a.setUpKeybindingLogGroup()
	a.setUpKeybindingLogStream()
	a.setUpKeybindingLogEvent()
}

func (a *App) setUpKeybindingLogGroup() {
	lgTable := a.widgets[LogGroupTable].(*tview.Table)
	lgSearch := a.widgets[LogGroupSearch].(*tview.InputField)
	lsTable := a.widgets[LogStreamTable].(*tview.Table)

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
		a.tvApp.SetFocus(a.widgets[LogStreamTable])
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
	lsTable := a.widgets[LogStreamTable].(*tview.Table)
	lsSearch := a.widgets[LogStreamSearch].(*tview.InputField)
	lgTable := a.widgets[LogGroupTable].(*tview.Table)

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
		setDefaultDropDownValue(a)
		a.pages.SwitchToPage(pageNames[LogEventPage])
		a.tvApp.SetFocus(a.widgets[StartYearDropDown])
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

func startDropDowns() []Widget {
	return []Widget{
		StartYearDropDown,
		StartMonthDropDown,
		StartDayDropDown,
		StartHourDropDown,
		StartMinuteDropDown,
	}
}

func endDropDowns() []Widget {
	return []Widget{
		EndYearDropDown,
		EndMonthDropDown,
		EndDayDropDown,
		EndHourDropDown,
		EndMinuteDropDown,
	}
}

func (a *App) setUpKeybindingLogEvent() {
	dds := append(startDropDowns(), endDropDowns()...)

	for i, dd := range dds {
		name := dd

		nowDropdown := a.widgets[name].(*tview.DropDown)

		var nextWidget tview.Primitive
		if i == len(dds)-1 {
			nextWidget = a.widgets[FilterPaternInput]
		} else {
			nextWidget = a.widgets[dds[(i+1)%len(dds)]]
		}
		nowDropdown.
			SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
				switch event.Rune() {
				case 'k':
					// up
					max := nowDropdown.GetOptionCount()
					idx, _ := nowDropdown.GetCurrentOption()
					if idx >= 1 {
						nowDropdown.SetCurrentOption((idx - 1) % max)
					}

				case 'j':
					// down
					max := nowDropdown.GetOptionCount()
					idx, _ := nowDropdown.GetCurrentOption()
					if idx < max-1 {
						nowDropdown.SetCurrentOption((idx + 1) % max)
					}
				}

				if event.Key() == tcell.KeyEsc {
					a.tvApp.SetFocus(a.widgets[LogGroupTable])
				} else if event.Key() == tcell.KeyTab {
					a.tvApp.SetFocus(nextWidget)
				} else if event.Key() == tcell.KeyEnter {
					if nowDropdown.IsOpen() {
						if i, _ := nowDropdown.GetCurrentOption(); i != -1 {
							a.applyLogEvent(aw)
						}
					}
				}
				return event
			})

		switch name {
		case StartMonthDropDown:
			nowDropdown.SetSelectedFunc(func(text string, index int) {
				a.inputForm(name, text)
				a.widgets[StartDayDropDown].(*tview.DropDown).SetOptions(getDaysByMonth(text), nil).SetSelectedFunc(func(text string, index int) {
					a.inputForm(StartDayDropDown, text)
				})
			})
		case EndMonthDropDown:
			nowDropdown.SetSelectedFunc(func(text string, index int) {
				a.inputForm(name, text)
				a.widgets[EndDayDropDown].(*tview.DropDown).SetOptions(getDaysByMonth(text), nil).SetSelectedFunc(func(text string, index int) {
					a.inputForm(EndDayDropDown, text)
				})
			})
		default:
			nowDropdown.SetSelectedFunc(func(text string, index int) {
				a.inputForm(name, text)
			})
		}
	}
	a.widgets[FilterPaternInput].(*tview.InputField).
		SetDoneFunc(func(key tcell.Key) {
			if key == tcell.KeyTab {
				a.tvApp.SetFocus(a.widgets[OutputFileInput])
			}
		}).
		SetChangedFunc(func(text string) {
			a.lEForm.filterPatern = text
			a.applyLogEvent(aw)
		})

	a.widgets[OutputFileInput].(*tview.InputField).
		SetDoneFunc(func(key tcell.Key) {
			if key == tcell.KeyTab {
				a.tvApp.SetFocus(a.widgets[SaveEventLogButton])
			}
		}).
		SetChangedFunc(func(text string) {
			a.lEForm.outputFile = text
			a.applyLogEvent(aw)
		})

	saveButton := a.widgets[SaveEventLogButton].(*tview.Button)
	saveButton.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			a.tvApp.SetFocus(a.widgets[BackButton])
		}
		return event
	})
	saveButton.SetSelectedFunc(func() {
		a.applyLogEvent(aw)
	})

	backButton := a.widgets[BackButton].(*tview.Button)
	backButton.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			a.tvApp.SetFocus(a.widgets[ViewLog])
		}
		return event
	})
	backButton.SetSelectedFunc(func() {
		a.pages.SwitchToPage(pageNames[LogGroupPage])
		a.tvApp.SetFocus(a.widgets[LogStreamTable])
	})

	viewLog := a.widgets[ViewLog].(*tview.TextView)
	viewLoa.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			a.tvApp.SetFocus(a.widgets[StartYearDropDown])
		}
		return event
	})
	viewLoa.SetScrollable(true)
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

func (g *gui) makeFormResult() (logEventInput, error) {
	lef := a.lEForm
	for _, dd := range startDropDowns() {
		nowDropdown := a.widgets[dd].(*tview.DropDown)
		if oi, _ := nowDropdown.GetCurrentOption(); oi == -1 {
			return logEventInput{}, fmt.Errorf("start time is not selected")
		}
	}

	for _, dd := range endDropDowns() {
		nowDropdown := a.widgets[dd].(*tview.DropDown)
		if oi, _ := nowDropdown.GetCurrentOption(); oi == -1 {
			return logEventInput{}, fmt.Errorf("end time is not selected")
		}
	}

	startTimeInput := time.Date(lef.startYear, lef.startMonth, lef.startDay, lef.startHour, lef.startMinute, 0, 0, time.Local)
	endTimeInput := time.Date(lef.endYear, lef.endMonth, lef.endDay, lef.endHour, lef.endMinute, 0, 0, time.Local)
	if startTimeInput.After(endTimeInput) {
		return logEventInput{}, fmt.Errorf("start time is after end time")
	}

	if lef.logGroupName == "" {
		return logEventInput{}, fmt.Errorf("log group name is not selected")
	}

	var filterPatern *string
	if lef.filterPatern != "" {
		filterPatern = aws.String(lef.filterPatern)
	}
	if lef.outputFile != "" {
		lef.enableOutputFile = true
	}
	return logEventInput{
		awsInput: &cwl.FilterLogEventsInput{
			LogGroupName:   aws.String(lef.logGroupName),
			LogStreamNames: lef.logStreamNames,
			StartTime:      aws.Int64(startTimeInput.UnixMilli()),
			EndTime:        aws.Int64(endTimeInput.UnixMilli()),
			FilterPattern:  filterPatern,
		},
		outputFile: lef.outputFile,
	}, nil
}

func (g *gui) inputForm(ddk Widget, text string) {
	switch ddk {
	case StartYearDropDown:
		a.lEForm.startYear = string2int(text)
	case StartMonthDropDown:
		a.lEForm.startMonth = string2month(text)
	case StartDayDropDown:
		loa.Println(".......in start day 1:", text)
		a.lEForm.startDay = string2int(text)
		loa.Println(".......in start day 2:", a.lEForm.startDay)
	case StartHourDropDown:
		a.lEForm.startHour = string2int(text)
	case StartMinuteDropDown:
		a.lEForm.startMinute = string2int(text)
	case EndYearDropDown:
		a.lEForm.endYear = string2int(text)
	case EndMonthDropDown:
		a.lEForm.endMonth = string2month(text)
	case EndDayDropDown:
		a.lEForm.endDay = string2int(text)
	case EndHourDropDown:
		a.lEForm.endHour = string2int(text)
	case EndMinuteDropDown:
		a.lEForm.endMinute = string2int(text)
	}
}

func string2int(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		loa.Fatalf("unable to list tables, %v", err)
	}
	return i
}

func string2month(s string) time.Month {
	i, err := strconv.Atoi(s)
	if err != nil {
		loa.Fatalf("unable to list tables, %v", err)
	}
	return time.Month(i)
}

func setDefaultDropDownValue(a *App) {
	now := time.Now()

	oneHourBefore := now.Add(-1 * time.Hour)
	// oneHourBeforeYear := oneHourBefore.Year()
	oneHourBeforeMonth := oneHourBefore.Month()
	oneHourBeforeDay := oneHourBefore.Day()
	oneHourBeforeHour := oneHourBefore.Hour()
	oneHourBeforeMinute := oneHourBefore.Minute()

	a.widgets[StartYearDropDown].(*tview.DropDown).SetCurrentOption(1)
	a.widgets[StartMonthDropDown].(*tview.DropDown).SetCurrentOption(int(oneHourBeforeMonth) - 1)
	a.widgets[StartDayDropDown].(*tview.DropDown).SetCurrentOption(oneHourBeforeDay - 1)
	a.widgets[StartHourDropDown].(*tview.DropDown).SetCurrentOption(oneHourBeforeHour)
	a.widgets[StartMinuteDropDown].(*tview.DropDown).SetCurrentOption(oneHourBeforeMinute)

	// currentYear := now.Year()
	currentMonth := now.Month()
	currentDay := now.Day()
	currentHour := now.Hour()
	currentMinute := now.Minute()

	a.widgets[EndYearDropDown].(*tview.DropDown).SetCurrentOption(1)
	a.widgets[EndMonthDropDown].(*tview.DropDown).SetCurrentOption(int(currentMonth) - 1)
	a.widgets[EndDayDropDown].(*tview.DropDown).SetCurrentOption(currentDay - 1)
	a.widgets[EndHourDropDown].(*tview.DropDown).SetCurrentOption(currentHour)
	a.widgets[EndMinuteDropDown].(*tview.DropDown).SetCurrentOption(currentMinute)
}

func (g *gui) applyLogEvent(aw *awsResource) {
	textView := a.widgets[ViewLog].(*tview.TextView)

	form, err := a.makeFormResult()
	if err != nil {
		textView.Clear()
		fmt.Fprintf(textView, "error: %v\n", err)
		return
	}

	textView.Clear()
	fmt.Fprintln(textView, "Now Loadina... ")

	go func() {
		res, err := aw.getLogEvents(form)
		a.tvApp.QueueUpdateDraw(func() {
			if err != nil {
				textView.Clear()
				fmt.Fprintf(textView, "error: %v\n", err)
				return
			}
			if len(res.Events) == 0 {
				textView.Clear()
				fmt.Fprintf(textView, "no events\n")
				printInputForm(*a.lEForm, textView)
				return
			}

			textView.Clear()
			printInputForm(*a.lEForm, textView)
			for _, event := range res.Events {
				fmt.Fprintf(textView, "%s\n", aws.ToString(event.Message))
			}
		})
	}()
}

func printInputForm(form logEventForm, textView *tview.TextView) {
	fmt.Fprintf(textView, "Your setting is\n")
	fmt.Fprintf(textView, "%s\n", form.logGroupName)
	fmt.Fprintf(textView, "%s\n", form.logStreamNames)
	fmt.Fprintf(textView, "%s\n", form.filterPatern)

	fmt.Fprintf(textView, "%s/%s/%s %s:%s\n",
		strconv.Itoa(form.startYear),
		strconv.Itoa(int(form.startMonth)),
		strconv.Itoa(form.startDay),
		strconv.Itoa(form.startHour),
		strconv.Itoa(form.startMinute),
	)
	fmt.Fprintf(textView, "    ~     \n")
	fmt.Fprintf(textView, "%s/%s/%s %s:%s\n",
		strconv.Itoa(form.endYear),
		strconv.Itoa(int(form.endMonth)),
		strconv.Itoa(form.endDay),
		strconv.Itoa(form.endHour),
		strconv.Itoa(form.endMinute),
	)
}
