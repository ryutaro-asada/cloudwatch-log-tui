package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	cwlTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	noShortcut rune = 0

	// Page names
	logGroupPage = "logGroups"
	logEventPage = "logEvents"
	// Layout names
	logGroupLayout = "LogGroup"
	logEventLayout = "LogEvent"
	// Widget names
	logGroupList        = "LogGroupeList"
	logGroupSearch      = "LogGroupSearch"
	startYearDropDown   = "StartYear"
	startMonthDropDown  = "StartMonth"
	startDayDropDown    = "StartDay"
	startHourDropDown   = "StartHour"
	startMinuteDropDown = "StartMinute"
	endYearDropDown     = "EndYear"
	endMonthDropDown    = "EndMonth"
	endDayDropDown      = "EndDay"
	endHourDropDown     = "EndHour"
	endMinuteDropDown   = "EndMinute"
	saveEventLogButton  = "SaveEventLog"
)

type gui struct {
	tvApp   *tview.Application
	pages   *tview.Pages
	layouts map[string]*tview.Flex
	widgets map[string]tview.Primitive
	lEFrom  *logEventForm

	// logGroupFuncs map[string]func()
	// logEventFuncs map[string]func()
}
type logEventForm struct {
	startYear   string
	startMonth  string
	startDay    string
	startHour   string
	startMinute string
	endYear     string
	endMonth    string
	endDay      string
	endHour     string
	endMinute   string
}

func (g *gui) setGui(aw *awsResource) {
	g.setLogGroupLayout()
	g.setLogEventLayout()
	g.pages = tview.NewPages().
		AddPage(logGroupPage, g.layouts[logGroupLayout], true, true).
		AddPage(logEventPage, g.layouts[logEventLayout], true, false)

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

	formMap := map[string][]string{
		startYearDropDown:   {"2024", "2025"},
		startMonthDropDown:  months,
		startDayDropDown:    days,
		startHourDropDown:   hours,
		startMinuteDropDown: minutes,
		endYearDropDown:     {"2024", "2025"},
		endMonthDropDown:    months,
		endDayDropDown:      days,
		endHourDropDown:     hours,
		endMinuteDropDown:   minutes,
	}

	for key, value := range formMap {
		DropDown := tview.NewDropDown().
			SetLabel(key).
			SetOptions(value, nil).
			SetFieldBackgroundColor(tcell.ColorGray)

		g.widgets[key] = DropDown
	}

	g.widgets[saveEventLogButton] = tview.NewButton("Save")

	g.layouts[logEventLayout] = tview.NewFlex().
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(g.widgets[startYearDropDown], 0, 1, true).
			AddItem(g.widgets[startMonthDropDown], 0, 1, false).
			AddItem(g.widgets[startDayDropDown], 0, 1, false).
			AddItem(g.widgets[startHourDropDown], 0, 1, false).
			AddItem(g.widgets[startMinuteDropDown], 0, 1, false), 0, 10, true).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(g.widgets[endYearDropDown], 0, 1, false).
			AddItem(g.widgets[endMonthDropDown], 0, 1, false).
			AddItem(g.widgets[endDayDropDown], 0, 1, false).
			AddItem(g.widgets[endHourDropDown], 0, 1, false).
			AddItem(g.widgets[endMinuteDropDown], 0, 1, false), 0, 10, false).
		AddItem(g.widgets[saveEventLogButton], 0, 1, false)
}

func (g *gui) setLogGroupLayout() {
	list := tview.NewList().ShowSecondaryText(false)
	list.SetBorder(true)
	list.SetTitle("Log Groups")
	g.widgets[logGroupList] = list

	search := tview.NewInputField().SetLabel("Word")
	search.SetLabelWidth(6)
	search.SetTitle("search")
	search.SetTitleAlign(tview.AlignLeft)
	search.SetBorder(true)
	search.SetFieldBackgroundColor(tcell.ColorGray)
	g.widgets[logGroupSearch] = search

	g.layouts[logGroupLayout] = tview.NewFlex().
		AddItem(g.widgets[logGroupList], 0, 30, false).SetDirection(tview.FlexRow).
		AddItem(g.widgets[logGroupSearch], 0, 1, false)
}

func (g *gui) setKeybinding(aw *awsResource) {
	g.setLogGroupKeybinding(aw.logGroups)
	g.setLogEventKeybinding()
}

func (g *gui) setLogGroupToGui(loggs []cwlTypes.LogGroup, filterPatern string) {
	for _, lg := range loggs {

		if filterPatern != "*" {
			if !strings.Contains(aws.ToString(lg.LogGroupName), filterPatern) {
				continue
			}
		}

		lgList := g.widgets[logGroupList].(*tview.List)
		lgList.AddItem(aws.ToString(lg.LogGroupName), "", noShortcut, nil)
	}
}

func (g *gui) setLogGroupKeybinding(resLogGroup []cwlTypes.LogGroup) {
	lgList := g.widgets[logGroupList].(*tview.List)
	lgSearch := g.widgets[logGroupSearch].(*tview.InputField)

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
		g.pages.SwitchToPage(logEventPage)
		g.tvApp.SetFocus(g.layouts[logEventLayout])
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

func (g *gui) setLogEventKeybinding() {
	dropDowns := []string{
		startYearDropDown,
		startMonthDropDown,
		startDayDropDown,
		startHourDropDown,
		startMinuteDropDown,
		endYearDropDown,
		endMonthDropDown,
		endDayDropDown,
		endHourDropDown,
		endMinuteDropDown,
	}

	for i, dd := range dropDowns {
		name := dd

		nowDropdown := g.widgets[name].(*tview.DropDown)
		
		var nextWidget tview.Primitive
		if i == len(dropDowns)-1 {
			nextWidget = g.widgets[saveEventLogButton]
		} else {
			nextWidget = g.widgets[dropDowns[(i+1)%len(dropDowns)]]
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
					g.tvApp.SetFocus(g.widgets[logGroupList])
				} else if event.Key() == tcell.KeyTab {
					g.tvApp.SetFocus(nextWidget)
				}
				return event
			})

		nowDropdown.SetSelectedFunc(func(text string, index int) {
			g.inputForm(name, text)
		})

		// if event.Key() == tcell.KeyEsc {
		// 	g.tvApp.SetFocus(g.widgets[logGroupList])
		// }

		// SetSelectedFunc(func(text string, index int) {
		// 	g.tvApp.SetFocus(g.widgets[dropDowns[(i+1)%len(dropDowns)]])
		// })
	}
	button := g.widgets[saveEventLogButton].(*tview.Button)
	button.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			g.tvApp.SetFocus(g.widgets[startYearDropDown])
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
	})
}

func (g *gui) inputForm(ddk string, text string) {
	switch ddk {
	case startYearDropDown:
		g.lEFrom.startYear = text
	case startMonthDropDown:
		g.lEFrom.startMonth = text
	case startDayDropDown:
		g.lEFrom.startDay = text
	case startHourDropDown:
		g.lEFrom.startHour = text
	case startMinuteDropDown:
		g.lEFrom.startMinute = text
	case endYearDropDown:
		g.lEFrom.endYear = text
	case endMonthDropDown:
		g.lEFrom.endMonth = text
	case endDayDropDown:
		g.lEFrom.endDay = text
	case endHourDropDown:
		g.lEFrom.endHour = text
	case endMinuteDropDown:
		g.lEFrom.endMinute = text
	}
}
