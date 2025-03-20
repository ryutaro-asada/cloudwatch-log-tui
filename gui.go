package main

import (
	"fmt"
	"log"
	"slices"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	cwl "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type (
	Page   int
	Layout int
	Widget int
	// page direction of Log
	Direction int
)

const (
	NoShortcut rune = 0

	// Page names
	LogGroupPage Page = iota
	LogEventPage

	// Layout names
	LogGroupLayout Layout = iota
	LogStreamLayout
	LogEventLayout

	// Widget names
	LogGroupTable Widget = iota
	LogGroupSearch
	LogStreamTable
	LogStreamSearch
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
	ViewLog

	Next Direction = iota
	Home
	Prev

	NextPage = "NextPage ..."
	PrevPage = "... PrevPage"
)

// Create a map to hold the string representations of the enums
var pageNames = map[Page]string{
	LogGroupPage: "logGroups",
	LogEventPage: "logEvents",
}

var layoutNames = map[Layout]string{
	LogGroupLayout:  "LogGroup",
	LogStreamLayout: "LogStream",
	LogEventLayout:  "LogEvent",
}

var widgetNames = map[Widget]string{
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
	ViewLog:             "ViewLog",
}

type gui struct {
	tvApp *tview.Application
	pages *tview.Pages
	// layouts   map[Layout]*tview.Flex
	layouts   map[Layout]tview.Primitive
	widgets   map[Widget]tview.Primitive
	lEFrom    *logEventForm
	logGroup  logGroup
	logStream logStream

	// logGroupFuncs map[string]func()
	// logEventFuncs map[string]func()
}

type logGroup struct {
	filterPatern string
	direction    Direction
}

type logStream struct {
	prefixPatern string
	direction    Direction
	logGroupName string
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
	logStreamNames     []string
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

	g.setLogGroupToGui(aw)
	g.setKeybinding(aw)
}

func (g *gui) setLogEventLayout() {
	g.setLogEventWidget()
	// newPrimitive := func(text string) tview.Primitive {
	// 	return tview.NewTextView().
	// 		SetTextAlign(tview.AlignCenter).
	// 		SetText(text)
	// }

	// g.layouts[LogEventLayout] = tview.NewGrid().
	// 	SetRows(3, 0).
	// 	SetColumns(0).
	// 	SetBorders(true).
	// 	AddItem(tview.NewGrid().
	// 		SetRows(0, 0, 0).
	// 		SetColumns(0, 0, 0, 0, 0).
	// 		SetBorders(true).
	// 		AddItem(newPrimitive("Menu"), 0, 0, 1, 1, 0, 100, false).
	// 		AddItem(newPrimitive("Menu"), 0, 1, 1, 1, 0, 100, false).
	// 		AddItem(newPrimitive("Menu"), 0, 2, 1, 1, 0, 100, false).
	// 		AddItem(newPrimitive("Menu"), 0, 3, 1, 1, 0, 100, false).
	// 		AddItem(newPrimitive("Menu"), 0, 4, 1, 1, 0, 100, false),
	// 		0, 0, 1, 3, 0, 0, false)
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
			0, 100, // minGridSize, maxGridSize
			false). // rowFixed
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
		// Log View
		AddItem(g.widgets[ViewLog],
			3, 0,
			1, 5,
			0, 100,
			false)

	// g.layouts[LogEventLayout] = tview.NewFlex().
	// 	AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
	// 		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
	// 			// column direction
	// 			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
	// 				// row direction
	// 				AddItem(g.widgets[StartYearDropDown], 0, 1, true).
	// 				AddItem(g.widgets[StartMonthDropDown], 0, 1, false).
	// 				AddItem(g.widgets[StartDayDropDown], 0, 1, false).
	// 				AddItem(g.widgets[StartHourDropDown], 0, 1, false).
	// 				AddItem(g.widgets[StartMinuteDropDown], 0, 1, false),
	// 				0, 10, true).
	// 			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
	// 				// row direction
	// 				AddItem(g.widgets[EndYearDropDown], 0, 1, false).
	// 				AddItem(g.widgets[EndMonthDropDown], 0, 1, false).
	// 				AddItem(g.widgets[EndDayDropDown], 0, 1, false).
	// 				AddItem(g.widgets[EndHourDropDown], 0, 1, false).
	// 				AddItem(g.widgets[EndMinuteDropDown], 0, 1, false),
	// 				0, 10, false).
	// 			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
	// 				// row direction
	// 				AddItem(g.widgets[FilterPaternInput], 0, 1, false).
	// 				AddItem(g.widgets[OutputFileInput], 0, 1, false).
	// 				AddItem(g.widgets[SaveEventLogButton], 0, 1, false),
	// 				0, 1, false),
	// 			0, 1, false).
	// 		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
	// 			AddItem(g.widgets[ViewLog], 0, 1, true),
	// 			0, 10, false),
	// 		0, 1, false)
}

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

	g.widgets[ViewLog] = tview.NewTextView()
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

func (g *gui) setLogGroupWidget() {
	// list := tview.NewList().ShowSecondaryText(false)
	// list.SetBorder(true)
	// list.SetTitle("Log Groups")
	// g.widgets[LogGroupTable] = list

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
	// list := tview.NewList().ShowSecondaryText(false)
	// list.SetBorder(true)
	// list.SetTitle("Log Streams")

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

func (g *gui) setKeybinding(aw *awsResource) {
	g.setLogGroupKeybinding(aw)
	g.setLogStreamKeybinding(aw)
	g.setLogEventKeybinding(aw)
}

func (g *gui) setLogGroupToGui(aw *awsResource) {
	lgTable := g.widgets[LogGroupTable].(*tview.Table)

	headers := []string{
		"Name",
		"RetentionDays",
		"StoredBytes",
	}

	row := 0
	for i, header := range headers {
		lgTable.SetCell(row, i, &tview.TableCell{
			Text:            header,
			NotSelectable:   true,
			Align:           tview.AlignLeft,
			Color:           tcell.ColorWhite,
			BackgroundColor: tcell.ColorDefault,
			Attributes:      tcell.AttrBold,
		})
	}
	row++

	if aw.hasPrevLogGroup {
		lgTable.SetCell(row, 0, tview.NewTableCell(PrevPage).
			SetTextColor(tcell.ColorLightSalmon).
			SetMaxWidth(1).
			SetExpansion(7))
		row++
	}

	for _, lg := range aw.logGroups {
		lgName := aws.ToString(lg.LogGroupName)
		log.Println("lgName: ", lgName)
		// int32 to string
		retentionDays := fmt.Sprintf("%d", aws.ToInt32(lg.RetentionInDays))
		storedBytes := fmt.Sprintf("%d", aws.ToInt64(lg.StoredBytes))
		log.Println("storedBytes")
		log.Println(storedBytes)

		// if g.logGroup.filterPatern != "*" {
		// 	if !strings.Contains(lgName, g.logGroup.filterPatern) {
		// 		continue
		// 	}
		// }
		//
		lgTable.SetCell(row, 0, tview.NewTableCell(lgName).
			SetTextColor(tcell.ColorLightGreen).
			SetMaxWidth(1).
			SetExpansion(7))

		lgTable.SetCell(row, 1, tview.NewTableCell(retentionDays).
			SetTextColor(tcell.ColorLightGreen).
			SetMaxWidth(1).
			SetExpansion(1))

		lgTable.SetCell(row, 2, tview.NewTableCell(storedBytes).
			SetTextColor(tcell.ColorLightGreen).
			SetMaxWidth(1).
			SetExpansion(1))

		row++
	}

	if aw.hasNextLogGroup {
		lgTable.SetCell(row, 0, tview.NewTableCell(NextPage).
			SetTextColor(tcell.ColorLightSteelBlue).
			SetMaxWidth(1).
			SetExpansion(7))
	}
}

func (g *gui) setLogStreamToGui(aw *awsResource) {
	lsTable := g.widgets[LogStreamTable].(*tview.Table)

	headers := []string{
		"Selected",
		"Name",
		"LastEventTime",
		"FirstEventTime",
	}

	row := 0
	for i, header := range headers {
		lsTable.SetCell(row, i, &tview.TableCell{
			Text:            header,
			NotSelectable:   true,
			Align:           tview.AlignLeft,
			Color:           tcell.ColorWhite,
			BackgroundColor: tcell.ColorDefault,
			Attributes:      tcell.AttrBold,
		})
	}
	row++

	if aw.hasPrevLogStream {
		lsTable.SetCell(row, 0, tview.NewTableCell("").
			SetTextColor(tcell.ColorLightGreen).
			SetMaxWidth(1).
			SetExpansion(1))
		lsTable.SetCell(row, 1, tview.NewTableCell(PrevPage).
			SetTextColor(tcell.ColorLightSalmon).
			SetMaxWidth(1).
			SetExpansion(7))
		lsTable.SetCell(row, 2, tview.NewTableCell("").
			SetTextColor(tcell.ColorLightGreen).
			SetMaxWidth(1).
			SetExpansion(2))

		lsTable.SetCell(row, 3, tview.NewTableCell("").
			SetTextColor(tcell.ColorLightGreen).
			SetMaxWidth(1).
			SetExpansion(2))
		row++
	}

	lsTable.SetCell(row, 0, tview.NewTableCell("").
		SetTextColor(tcell.ColorLightGreen).
		SetMaxWidth(1).
		SetExpansion(1))

	lsTable.SetCell(row, 1, tview.NewTableCell("All Log Streams").
		SetTextColor(tcell.ColorLightGreen).
		SetMaxWidth(1).
		SetExpansion(10))

	lsTable.SetCell(row, 2, tview.NewTableCell("").
		SetTextColor(tcell.ColorLightGreen).
		SetMaxWidth(1).
		SetExpansion(2))

	lsTable.SetCell(row, 3, tview.NewTableCell("").
		SetTextColor(tcell.ColorLightGreen).
		SetMaxWidth(1).
		SetExpansion(2))

	row++

	for _, ls := range aw.logStreams {
		lsName := aws.ToString(ls.LogStreamName)
		lastEventTime := time.Unix(aws.ToInt64(ls.LastEventTimestamp), 0).Local().Format("2006-01-02 15:04:05")
		firstEventTime := time.Unix(aws.ToInt64(ls.FirstEventTimestamp), 0).Local().Format("2006-01-02 15:04:05")
		selectedMark := ""

		// if g.logStream.prefixPatern != "" {
		// 	if !strings.Contains(lsName, g.logStream.prefixPatern) {
		// 		continue
		// 	}
		// }

		if slices.Contains(g.lEFrom.logStreamNames, lsName) {
			selectedMark = "x"
		}

		lsTable.SetCell(row, 0, tview.NewTableCell(selectedMark).
			SetTextColor(tcell.ColorLightGreen).
			SetMaxWidth(1).
			SetExpansion(1))

		lsTable.SetCell(row, 1, tview.NewTableCell(lsName).
			SetTextColor(tcell.ColorLightGreen).
			SetMaxWidth(1).
			SetExpansion(10))

		lsTable.SetCell(row, 2, tview.NewTableCell(lastEventTime).
			SetTextColor(tcell.ColorLightGreen).
			SetMaxWidth(1).
			SetExpansion(2))

		lsTable.SetCell(row, 3, tview.NewTableCell(firstEventTime).
			SetTextColor(tcell.ColorLightGreen).
			SetMaxWidth(1).
			SetExpansion(2))

		row++

	}

	if aw.hasNextLogStream {
		lsTable.SetCell(row, 0, tview.NewTableCell("").
			SetTextColor(tcell.ColorLightGreen).
			SetMaxWidth(1).
			SetExpansion(1))

		lsTable.SetCell(row, 1, tview.NewTableCell(NextPage).
			SetTextColor(tcell.ColorLightSteelBlue).
			SetMaxWidth(1).
			SetExpansion(7))

		lsTable.SetCell(row, 2, tview.NewTableCell("").
			SetTextColor(tcell.ColorLightGreen).
			SetMaxWidth(1).
			SetExpansion(2))

		lsTable.SetCell(row, 3, tview.NewTableCell("").
			SetTextColor(tcell.ColorLightGreen).
			SetMaxWidth(1).
			SetExpansion(2))
	}
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
		log.Println("cell name")
		log.Println(cell.Text)
		log.Println("row ", row)
		log.Println("column ", column)
		g.lEFrom.logGroupName = cell.Text
		g.logStream.logGroupName = cell.Text
		aw.getLogStreams(g.logStream)
		g.setLogStreamToGui(aw)
		g.tvApp.SetFocus(g.widgets[LogStreamTable])
	})
	lgTable.SetSelectionChangedFunc(func(row, column int) {
		cell := lgTable.GetCell(row, 0)
		log.Println("cell name", cell.Text, row, column)
		if cell.Text == PrevPage {
			lgTable.Clear()
			g.logGroup.direction = Prev
			aw.getLogGroups(g.logGroup)
			g.setLogGroupToGui(aw)
			lgTable.Select(lgTable.GetRowCount()-2, 0)
			log.Println("get row cont.......................", lgTable.GetRowCount())

		} else if cell.Text == NextPage {
			lgTable.Clear()
			g.logGroup.direction = Next
			aw.getLogGroups(g.logGroup)
			log.Println("Next Page..............................")
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
		lgTable.Clear()
		g.logGroup.filterPatern = filterPatern
		g.logGroup.direction = Home
		aw.getLogGroups(g.logGroup)
		g.setLogGroupToGui(aw)
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
			log.Println("selected log stream name")
			log.Println(logStreamName)
			if logStreamName == "All Log Streams" {
				return nil
			}
			log.Println(logStreamName)
			// log.Println(selectedMark)
			lsTable.Clear()

			switch slices.Contains(g.lEFrom.logStreamNames, logStreamName) {
			case true:
				i := slices.Index(g.lEFrom.logStreamNames, logStreamName)
				g.lEFrom.logStreamNames = slices.Delete(g.lEFrom.logStreamNames, i, i+1)
			case false:
				g.lEFrom.logStreamNames = append(g.lEFrom.logStreamNames, logStreamName)
			}

			g.setLogStreamToGui(aw)

			log.Print("selected log stream names")
			log.Print(g.lEFrom.logStreamNames)

			log.Println("space")
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
			g.lEFrom.logStreamNames = nil
		} else {
			g.lEFrom.logStreamNames = append(g.lEFrom.logStreamNames, logStreamName)
			g.lEFrom.logStreamNames = slices.Compact(g.lEFrom.logStreamNames)
		}
		g.pages.SwitchToPage(pageNames[LogEventPage])
		g.tvApp.SetFocus(g.widgets[StartYearDropDown])
	})

	lsTable.SetSelectionChangedFunc(func(row, column int) {
		cell := lsTable.GetCell(row, 1)
		if cell.Text == PrevPage {
			g.logStream.direction = Prev
			aw.getLogStreams(g.logStream)
			lsTable.Clear()
			g.setLogStreamToGui(aw)
			lsTable.Select(lsTable.GetRowCount()-2, 1)
			log.Println("get row cont.......................", lsTable.GetRowCount())
		} else if cell.Text == NextPage {
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
				// if nowDropdown.IsOpen() {
				// 	return event
				// }

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
				}
				return event
			})

		switch name {
		case StartMonthDropDown:
			nowDropdown.SetSelectedFunc(func(text string, index int) {
				g.inputForm(name, text)
				g.widgets[StartDayDropDown].(*tview.DropDown).SetOptions(getDaysByMonth(text), nil)
			})
		case EndMonthDropDown:
			nowDropdown.SetSelectedFunc(func(text string, index int) {
				g.inputForm(name, text)
				g.widgets[EndDayDropDown].(*tview.DropDown).SetOptions(getDaysByMonth(text), nil)
			})
		default:
			nowDropdown.SetSelectedFunc(func(text string, index int) {
				g.inputForm(name, text)
			})
		}

		// if event.Key() == tcell.KeyEsc {
		// 	g.tvApp.SetFocus(g.widgets[logGroupList])
		// }

		// SetSelectedFunc(func(text string, index int) {
		// 	g.tvApp.SetFocus(g.widgets[dropDowns[(i+1)%len(dropDowns)]])
		// })
	}
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

	button := g.widgets[SaveEventLogButton].(*tview.Button)
	button.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			g.tvApp.SetFocus(g.widgets[StartYearDropDown])
		}
		return event
	})

	button.SetSelectedFunc(func() {
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
			LogGroupName:   aws.String(lef.logGroupName),
			LogStreamNames: lef.logStreamNames,
			StartTime:      startTime(lef),
			EndTime:        endTime(lef),
			FilterPattern:  filterPattern(lef),
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
