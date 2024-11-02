package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	cwl "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	cwlTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type (
	Page   int
	Layout int
	Widget int
)

const (
	NoShortcut rune = 0

	// Page names
	LogGroupPage Page = iota
	LogEventPage

	// Layout names
	LogGroupLayout Layout = iota
	LogEventLayout

	// Widget names
	LogGroupList Widget = iota
	LogGroupSearch
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
)

// Create a map to hold the string representations of the enums
var pageNames = map[Page]string{
	LogGroupPage: "logGroups",
	LogEventPage: "logEvents",
}

var layoutNames = map[Layout]string{
	LogGroupLayout: "LogGroup",
	LogEventLayout: "LogEvent",
}

var widgetNames = map[Widget]string{
	LogGroupList:        "LogGroupeList",
	LogGroupSearch:      "LogGroupSearch",
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
}

type gui struct {
	tvApp   *tview.Application
	pages   *tview.Pages
	layouts map[Layout]*tview.Flex
	widgets map[Widget]tview.Primitive
	lEFrom  *logEventForm

	// logGroupFuncs map[string]func()
	// logEventFuncs map[string]func()
}
type logEventForm struct {
	startTimeSelected  bool
	startYear          int
	startMonth         time.Month
	startDay           int
	startHour          int
	startMinute        int
	endTimeSelected    bool
	endYear            int
	endMonth           time.Month
	endDay             int
	endHour            int
	endMinute          int
	logGroupName       string
	filterPatern       string
	enableFilterPatern bool
	outputFile         string
	enableOutputFile   bool
}

type logEventInut struct {
	awsInput   *cwl.FilterLogEventsInput
	outputFile string
}

func (g *gui) setGui(aw *awsResource) {
	g.setLogGroupLayout()
	g.setLogEventLayout()
	g.pages = tview.NewPages().
		AddPage(pageNames[LogGroupPage], g.layouts[LogGroupLayout], true, true).
		AddPage(pageNames[LogEventPage], g.layouts[LogEventLayout], true, false)

	g.setLogGroupToGui(aw.logGroups, "*")
	g.setKeybinding(aw)
}

func (g *gui) setLogEventLayout() {
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
	g.widgets[FilterPaternInput] = tview.NewInputField().SetLabel("Filter Pattern")

	g.widgets[OutputFileInput] = tview.NewInputField().SetLabel("Output File")

	g.widgets[SaveEventLogButton] = tview.NewButton("Save")

	g.layouts[LogEventLayout] = tview.NewFlex().
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(g.widgets[StartYearDropDown], 0, 1, true).
			AddItem(g.widgets[StartMonthDropDown], 0, 1, false).
			AddItem(g.widgets[StartDayDropDown], 0, 1, false).
			AddItem(g.widgets[StartHourDropDown], 0, 1, false).
			AddItem(g.widgets[StartMinuteDropDown], 0, 1, false), 0, 10, true).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(g.widgets[EndYearDropDown], 0, 1, false).
			AddItem(g.widgets[EndMonthDropDown], 0, 1, false).
			AddItem(g.widgets[EndDayDropDown], 0, 1, false).
			AddItem(g.widgets[EndHourDropDown], 0, 1, false).
			AddItem(g.widgets[EndMinuteDropDown], 0, 1, false), 0, 10, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(g.widgets[FilterPaternInput], 0, 1, false).
			AddItem(g.widgets[OutputFileInput], 0, 1, false).
			AddItem(g.widgets[SaveEventLogButton], 0, 1, false), 0, 1, false)
}

func (g *gui) setLogGroupLayout() {
	list := tview.NewList().ShowSecondaryText(false)
	list.SetBorder(true)
	list.SetTitle("Log Groups")
	g.widgets[LogGroupList] = list

	search := tview.NewInputField().SetLabel("Word")
	search.SetLabelWidth(6)
	search.SetTitle("search")
	search.SetTitleAlign(tview.AlignLeft)
	search.SetBorder(true)
	search.SetFieldBackgroundColor(tcell.ColorGray)
	g.widgets[LogGroupSearch] = search

	g.layouts[LogGroupLayout] = tview.NewFlex().
		AddItem(g.widgets[LogGroupList], 0, 30, false).SetDirection(tview.FlexRow).
		AddItem(g.widgets[LogGroupSearch], 0, 1, false)
}

func (g *gui) setKeybinding(aw *awsResource) {
	g.setLogGroupKeybinding(aw.logGroups)
	g.setLogEventKeybinding(aw)
}

func (g *gui) setLogGroupToGui(loggs []cwlTypes.LogGroup, filterPatern string) {
	for _, lg := range loggs {

		if filterPatern != "*" {
			if !strings.Contains(aws.ToString(lg.LogGroupName), filterPatern) {
				continue
			}
		}

		lgList := g.widgets[LogGroupList].(*tview.List)
		lgList.AddItem(aws.ToString(lg.LogGroupName), "", NoShortcut, nil)
	}
}

func (g *gui) setLogGroupKeybinding(resLogGroup []cwlTypes.LogGroup) {
	lgList := g.widgets[LogGroupList].(*tview.List)
	lgSearch := g.widgets[LogGroupSearch].(*tview.InputField)

	lgList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		max := lgList.GetItemCount()
		now := lgList.GetCurrentItem()
		switch event.Rune() {
		case 'k':
			// up
			lgList.SetCurrentItem((now - 1) % max)
		case 'j':
			// down
			lgList.SetCurrentItem((now + 1) % max)

		case '/':
			g.tvApp.SetFocus(lgSearch)
		}
		return event
	})

	lgList.SetSelectedFunc(func(index int, mtxt string, stxt string, shortcut rune) {
		// app.getLogEvents(mtxt)
		//
		// log.Println(mtxt)
		g.lEFrom.logGroupName = mtxt
		g.pages.SwitchToPage(pageNames[LogEventPage])
		g.tvApp.SetFocus(g.widgets[StartYearDropDown])
		// time.Sleep(3 * time.Second)
		// filterLogEventForm.Clear(false)
	})

	lgSearch.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			g.tvApp.SetFocus(lgList)
		}
	})
	lgSearch.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			g.tvApp.SetFocus(lgList)
		}
		return event
	})
	lgSearch.SetChangedFunc(func(text string) {
		lgList.Clear()
		g.setLogGroupToGui(resLogGroup, text)
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
				if nowDropdown.IsOpen() {
					// switch event.Rune() {
					// case 'k':
					// 	// up
					// 	nowDropdown.
					// case 'j':
					// 	// down
					// }

					return event
				}
				if event.Key() == tcell.KeyEsc {
					g.tvApp.SetFocus(g.widgets[LogGroupList])
				} else if event.Key() == tcell.KeyTab {
					g.tvApp.SetFocus(nextWidget)
				}
				return event
			})

		nowDropdown.SetSelectedFunc(func(text string, index int) {
			g.inputForm(name, text)
		})

		g.widgets[FilterPaternInput].(*tview.InputField).
			SetDoneFunc(func(key tcell.Key) {
				if key == tcell.KeyTab {
					g.tvApp.SetFocus(g.widgets[OutputFileInput])
				}
			}).
			SetChangedFunc(func(text string) {
				g.lEFrom.filterPatern = text
			})

		g.widgets[OutputFileInput].(*tview.InputField).
			SetDoneFunc(func(key tcell.Key) {
				if key == tcell.KeyTab {
					g.tvApp.SetFocus(g.widgets[SaveEventLogButton])
				}
			}).
			SetChangedFunc(func(text string) {
				g.lEFrom.outputFile = text
			})

		// if event.Key() == tcell.KeyEsc {
		// 	g.tvApp.SetFocus(g.widgets[logGroupList])
		// }

		// SetSelectedFunc(func(text string, index int) {
		// 	g.tvApp.SetFocus(g.widgets[dropDowns[(i+1)%len(dropDowns)]])
		// })
	}
	button := g.widgets[SaveEventLogButton].(*tview.Button)
	button.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			g.tvApp.SetFocus(g.widgets[StartYearDropDown])
		}
		return event
	})
	button.SetSelectedFunc(func() {
		log.Println(g.lEFrom.startYear)
		log.Println(g.lEFrom.startMonth)
		log.Println(g.lEFrom.startDay)
		log.Println(g.lEFrom.startHour)
		log.Println(g.lEFrom.startMinute)
		log.Println(g.lEFrom.endYear)
		log.Println(g.lEFrom.endMonth)
		log.Println(g.lEFrom.endDay)
		log.Println(g.lEFrom.endHour)
		log.Println(g.lEFrom.endMinute)
		log.Println("save")
		result := g.makeFormResult()
		aw.getLogEvents(result)
	})
}

func (g *gui) makeFormResult() logEventInut {
	lef := g.lEFrom
	for di, dd := range startDropDowns() {
		nowDropdown := g.widgets[dd].(*tview.DropDown)
		if oi, _ := nowDropdown.GetCurrentOption(); oi == -1 {
			break
		}
		if di == len(startDropDowns())-1 {
			lef.startTimeSelected = true
		}
	}

	for di, dd := range endDropDowns() {
		nowDropdown := g.widgets[dd].(*tview.DropDown)
		if oi, _ := nowDropdown.GetCurrentOption(); oi == -1 {
			break
		}
		if di == len(endDropDowns())-1 {
			lef.endTimeSelected = true
		}
	}

	if lef.startTimeSelected && lef.endTimeSelected {
		startTime := time.Date(lef.startYear, lef.startMonth, lef.startDay, lef.startHour, lef.startMinute, 0, 0, time.Local)
		endTime := time.Date(lef.endYear, lef.endMonth, lef.endDay, lef.endHour, lef.endMinute, 0, 0, time.Local)
		if startTime.After(endTime) {
			log.Fatalf("start time is after end time")
		}
	}

	if lef.logGroupName == "" {
		log.Fatalf("log group name is empty")
	}
	if lef.filterPatern != "" {
		lef.enableFilterPatern = true
	}
	if lef.outputFile != "" {
		lef.enableOutputFile = true
	}
	return logEventInut{
		awsInput: &cwl.FilterLogEventsInput{
			LogGroupName:  aws.String(lef.logGroupName),
			StartTime:     startTime(lef),
			EndTime:       endTime(lef),
			FilterPattern: filterPattern(lef),
		},
		outputFile: lef.outputFile,
	}
}

// func (g *gui) logEventInput(lef *logEventForm) *cwl.FilterLogEventsInput {
// 	if lef.startTimeSelected && lef.endTimeSelected && lef.enableFilterPatern {
// 		return &cwl.FilterLogEventsInput{
// 			LogGroupName:  aws.String(lef.logGroupName),
// 			StartTime:     aws.Int64(startTime(lef)),
// 			EndTime:       aws.Int64(endTime(lef)),
// 			FilterPattern: aws.String(lef.filterPatern),
// 		}
// 	} else if lef.startTimeSelected && lef.endTimeSelected && !lef.enableFilterPatern {
// 		return &cwl.FilterLogEventsInput{
// 			LogGroupName:  aws.String(lef.logGroupName),
// 			StartTime:     aws.Int64(startTime(lef)),
// 			EndTime:       aws.Int64(endTime(lef)),
// 		}
// 	} else if lef.startTimeSelected && lef.endTimeSelected && !lef.enableFilterPatern {
// 		return &cwl.FilterLogEventsInput{
// 			LogGroupName:  aws.String(lef.logGroupName),
// 			StartTime:     aws.Int64(startTime(lef)),
// 			EndTime:       aws.Int64(endTime(lef)),
// 		}
// 	}
// }

func (g *gui) inputForm(ddk Widget, text string) {
	switch ddk {
	case StartYearDropDown:
		g.lEFrom.startYear = string2int(text)
	case StartMonthDropDown:
		g.lEFrom.startMonth = string2month(text)
	case StartDayDropDown:
		g.lEFrom.startDay = string2int(text)
	case StartHourDropDown:
		g.lEFrom.startHour = string2int(text)
	case StartMinuteDropDown:
		g.lEFrom.startMinute = string2int(text)
	case EndYearDropDown:
		g.lEFrom.endYear = string2int(text)
	case EndMonthDropDown:
		g.lEFrom.endMonth = string2month(text)
	case EndDayDropDown:
		g.lEFrom.endDay = string2int(text)
	case EndHourDropDown:
		g.lEFrom.endHour = string2int(text)
	case EndMinuteDropDown:
		g.lEFrom.endMinute = string2int(text)
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
