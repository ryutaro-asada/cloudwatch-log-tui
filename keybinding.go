package main

import (
	"fmt"
	"log"
	"reflect"
	"slices"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	cwl "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (g *gui) setKeybinding(aw *awsResource) {
	g.setLogGroupKeybinding(aw)
	g.setLogStreamKeybinding(aw)
	g.setLogEventKeybinding(aw)
}

func (g *gui) setLogGroupKeybinding(aw *awsResource) {
	lgTable := g.widgets[LogGroupTable].(*tview.Table)
	lgSearch := g.widgets[LogGroupSearch].(*tview.InputField)
	lsTable := g.widgets[LogStreamTable].(*tview.Table)

	lgTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		max := lgTable.GetRowCount()
		row, _ := lgTable.GetSelection()
		switch event.Rune() {
		case 'k':
			// up
			lgTable.Select((row)%max, 0)
		case 'j':
			// down
			lgTable.Select((row)%max, 0)

		case '/':
			g.tvApp.SetFocus(lgSearch)
		}

		if event.Key() == tcell.KeyTab {
			g.tvApp.SetFocus(lsTable)
		}
		return event
	})

	lgTable.SetSelectedFunc(func(row, column int) {
		cell := lgTable.GetCell(row, 0)
		g.lEForm.logGroupName = cell.Text
		g.logStream.logGroupName = cell.Text
		g.tvApp.SetFocus(g.widgets[LogStreamTable])
		go func() {
			aw.getLogStreams(g.logStream)
			g.tvApp.QueueUpdateDraw(func() {
				g.setLogStreamToGui(aw)
			})
		}()
	})

	lgTable.SetSelectionChangedFunc(func(row, column int) {
		cell := lgTable.GetCell(row, 0)
		switch cell.Text {
		case PrevPage:
			lgTable.Clear()
			g.logGroup.direction = Prev
			aw.getLogGroups(g.logGroup)
			g.setLogGroupToGui(aw)
			lgTable.Select(lgTable.GetRowCount()-2, 0)

		case NextPage:
			lgTable.Clear()
			g.logGroup.direction = Next
			aw.getLogGroups(g.logGroup)
			g.setLogGroupToGui(aw)
			lgTable.Select(2, 0)
		}
	})

	lgSearch.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			g.tvApp.SetFocus(g.widgets[LogGroupTable])
		}
	})
	lgSearch.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			g.tvApp.SetFocus(lgTable)
		}
		return event
	})
	lgSearch.SetChangedFunc(func(filterPatern string) {
		g.logGroup.filterPatern = filterPatern
		g.logGroup.direction = Home
		go func() {
			aw.getLogGroups(g.logGroup)
			g.tvApp.QueueUpdateDraw(func() {
				lgTable.Clear()
				g.setLogGroupToGui(aw)
			})
		}()
	})
}

func (g *gui) setLogStreamKeybinding(aw *awsResource) {
	lsTable := g.widgets[LogStreamTable].(*tview.Table)
	lsSearch := g.widgets[LogStreamSearch].(*tview.InputField)
	lgTable := g.widgets[LogGroupTable].(*tview.Table)

	lsTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		max := lsTable.GetRowCount()
		row, _ := lsTable.GetSelection()
		switch event.Rune() {
		case 'k':
			// up
			lsTable.Select((row)%max, 1)
		case 'j':
			// down
			lsTable.Select((row)%max, 1)

		case '/':
			g.tvApp.SetFocus(lsSearch)
		// Space key
		case ' ':

			cell := lsTable.GetCell(row, 1)
			logStreamName := cell.Text
			if logStreamName == "All Log Streams" {
				return nil
			}
			lsTable.Clear()

			switch slices.Contains(g.lEForm.logStreamNames, logStreamName) {
			case true:
				i := slices.Index(g.lEForm.logStreamNames, logStreamName)
				g.lEForm.logStreamNames = slices.Delete(g.lEForm.logStreamNames, i, i+1)
			case false:
				g.lEForm.logStreamNames = append(g.lEForm.logStreamNames, logStreamName)
			}

			g.setLogStreamToGui(aw)

			return nil
		}

		if event.Key() == tcell.KeyTab {
			g.tvApp.SetFocus(lgTable)
		}

		return event
	})

	lsTable.SetSelectedFunc(func(row, col int) {
		cell := lsTable.GetCell(row, 1)
		logStreamName := cell.Text
		if logStreamName == "All Log Streams" {
			g.lEForm.logStreamNames = nil
		} else {
			g.lEForm.logStreamNames = append(g.lEForm.logStreamNames, logStreamName)
			g.lEForm.logStreamNames = slices.Compact(g.lEForm.logStreamNames)
		}
		setDefaultDropDownValue(g)
		g.pages.SwitchToPage(pageNames[LogEventPage])
		g.tvApp.SetFocus(g.widgets[StartYearDropDown])
	})

	lsTable.SetSelectionChangedFunc(func(row, column int) {
		cell := lsTable.GetCell(row, 1)
		switch cell.Text {
		case PrevPage:
			g.logStream.direction = Prev
			aw.getLogStreams(g.logStream)
			lsTable.Clear()
			g.setLogStreamToGui(aw)
			lsTable.Select(lsTable.GetRowCount()-2, 1)

		case NextPage:
			g.logStream.direction = Next
			aw.getLogStreams(g.logStream)
			lsTable.Clear()
			g.setLogStreamToGui(aw)
			lsTable.Select(2, 1)
		}
	})

	lsSearch.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			g.tvApp.SetFocus(lsTable)
		}
	})
	lsSearch.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			g.tvApp.SetFocus(lsTable)
		}
		return event
	})
	lsSearch.SetChangedFunc(func(prefixPatern string) {
		lsTable.Clear()
		g.logStream.prefixPatern = prefixPatern
		g.logStream.direction = Home
		aw.getLogStreams(g.logStream)
		g.setLogStreamToGui(aw)
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

func (g *gui) setLogEventKeybinding(aw *awsResource) {
	dds := append(startDropDowns(), endDropDowns()...)

	for i, dd := range dds {
		name := dd

		nowDropdown := g.widgets[name].(*tview.DropDown)

		var nextWidget tview.Primitive
		if i == len(dds)-1 {
			nextWidget = g.widgets[FilterPaternInput]
		} else {
			nextWidget = g.widgets[dds[(i+1)%len(dds)]]
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
					g.tvApp.SetFocus(g.widgets[LogGroupTable])
				} else if event.Key() == tcell.KeyTab {
					g.tvApp.SetFocus(nextWidget)
				} else if event.Key() == tcell.KeyEnter {
					if nowDropdown.IsOpen() {
						if i, _ := nowDropdown.GetCurrentOption(); i != -1 {
							g.applyLogEvent(aw)
						}
					}
				}
				return event
			})

		switch name {
		case StartMonthDropDown:
			nowDropdown.SetSelectedFunc(func(text string, index int) {
				g.inputForm(name, text)
				g.widgets[StartDayDropDown].(*tview.DropDown).SetOptions(getDaysByMonth(text), nil).SetSelectedFunc(func(text string, index int) {
					// log.Println(".......in start day Selected:", text, index)
					// log.Println(".......in start day Selected:", name)
					g.inputForm(StartDayDropDown, text)
				})
			})
		case EndMonthDropDown:
			nowDropdown.SetSelectedFunc(func(text string, index int) {
				g.inputForm(name, text)
				g.widgets[EndDayDropDown].(*tview.DropDown).SetOptions(getDaysByMonth(text), nil).SetSelectedFunc(func(text string, index int) {
					// log.Println("....... in end day Selected:", text, index)
					g.inputForm(EndDayDropDown, text)
				})
			})
		default:
			// log.Println("..name:", name)
			nowDropdown.SetSelectedFunc(func(text string, index int) {
				// log.Println(".......Selected:", text, index)
				g.inputForm(name, text)
			})
		}
	}
	g.widgets[FilterPaternInput].(*tview.InputField).
		SetDoneFunc(func(key tcell.Key) {
			if key == tcell.KeyTab {
				g.tvApp.SetFocus(g.widgets[OutputFileInput])
			}
		}).
		SetChangedFunc(func(text string) {
			g.lEForm.filterPatern = text
			g.applyLogEvent(aw)
		})

	g.widgets[OutputFileInput].(*tview.InputField).
		SetDoneFunc(func(key tcell.Key) {
			if key == tcell.KeyTab {
				g.tvApp.SetFocus(g.widgets[SaveEventLogButton])
			}
		}).
		SetChangedFunc(func(text string) {
			g.lEForm.outputFile = text
			g.applyLogEvent(aw)
		})

	saveButton := g.widgets[SaveEventLogButton].(*tview.Button)
	saveButton.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			g.tvApp.SetFocus(g.widgets[BackButton])
		}
		return event
	})
	saveButton.SetSelectedFunc(func() {
		g.applyLogEvent(aw)
	})

	backButton := g.widgets[BackButton].(*tview.Button)
	backButton.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			g.tvApp.SetFocus(g.widgets[ViewLog])
		}
		return event
	})
	backButton.SetSelectedFunc(func() {
		g.pages.SwitchToPage(pageNames[LogGroupPage])
		g.tvApp.SetFocus(g.widgets[LogStreamTable])
	})

	viewLog := g.widgets[ViewLog].(*tview.TextView)
	viewLog.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			g.tvApp.SetFocus(g.widgets[StartYearDropDown])
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

func (g *gui) makeFormResult() (logEventInput, error) {
	lef := g.lEForm
	for _, dd := range startDropDowns() {
		nowDropdown := g.widgets[dd].(*tview.DropDown)
		if oi, _ := nowDropdown.GetCurrentOption(); oi == -1 {
			return logEventInput{}, fmt.Errorf("start time is not selected")
		}
	}

	for _, dd := range endDropDowns() {
		nowDropdown := g.widgets[dd].(*tview.DropDown)
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
		g.lEForm.startYear = string2int(text)
	case StartMonthDropDown:
		g.lEForm.startMonth = string2month(text)
	case StartDayDropDown:
		log.Println(".......in start day 1:", text)
		g.lEForm.startDay = string2int(text)
		log.Println(".......in start day 2:", g.lEForm.startDay)
	case StartHourDropDown:
		g.lEForm.startHour = string2int(text)
	case StartMinuteDropDown:
		g.lEForm.startMinute = string2int(text)
	case EndYearDropDown:
		g.lEForm.endYear = string2int(text)
	case EndMonthDropDown:
		g.lEForm.endMonth = string2month(text)
	case EndDayDropDown:
		g.lEForm.endDay = string2int(text)
	case EndHourDropDown:
		g.lEForm.endHour = string2int(text)
	case EndMinuteDropDown:
		g.lEForm.endMinute = string2int(text)
	}
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

func setDefaultDropDownValue(g *gui) {
	now := time.Now()

	oneHourBefore := now.Add(-1 * time.Hour)
	// oneHourBeforeYear := oneHourBefore.Year()
	oneHourBeforeMonth := oneHourBefore.Month()
	oneHourBeforeDay := oneHourBefore.Day()
	oneHourBeforeHour := oneHourBefore.Hour()
	oneHourBeforeMinute := oneHourBefore.Minute()

	g.widgets[StartYearDropDown].(*tview.DropDown).SetCurrentOption(1)
	g.widgets[StartMonthDropDown].(*tview.DropDown).SetCurrentOption(int(oneHourBeforeMonth) - 1)
	g.widgets[StartDayDropDown].(*tview.DropDown).SetCurrentOption(oneHourBeforeDay - 1)
	g.widgets[StartHourDropDown].(*tview.DropDown).SetCurrentOption(oneHourBeforeHour)
	g.widgets[StartMinuteDropDown].(*tview.DropDown).SetCurrentOption(oneHourBeforeMinute)

	// currentYear := now.Year()
	currentMonth := now.Month()
	currentDay := now.Day()
	currentHour := now.Hour()
	currentMinute := now.Minute()

	g.widgets[EndYearDropDown].(*tview.DropDown).SetCurrentOption(1)
	g.widgets[EndMonthDropDown].(*tview.DropDown).SetCurrentOption(int(currentMonth) - 1)
	g.widgets[EndDayDropDown].(*tview.DropDown).SetCurrentOption(currentDay - 1)
	g.widgets[EndHourDropDown].(*tview.DropDown).SetCurrentOption(currentHour)
	g.widgets[EndMinuteDropDown].(*tview.DropDown).SetCurrentOption(currentMinute)
}

func (g *gui) applyLogEvent(aw *awsResource) {
	textView := g.widgets[ViewLog].(*tview.TextView)

	form, err := g.makeFormResult()
	if err != nil {
		textView.Clear()
		fmt.Fprintf(textView, "error: %v\n", err)
		return
	}

	textView.Clear()
	fmt.Fprintln(textView, "Now Loading... ")

	go func() {
		res, err := aw.getLogEvents(form)
		g.tvApp.QueueUpdateDraw(func() {
			if err != nil {
				textView.Clear()
				fmt.Fprintf(textView, "error: %v\n", err)
				return
			}
			if len(res.Events) == 0 {
				textView.Clear()
				fmt.Fprintf(textView, "no events\n")
				printInputForm(*g.lEForm, textView)
				return
			}

			textView.Clear()
			printInputForm(*g.lEForm, textView)
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
