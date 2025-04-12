package main

import (
	"fmt"
	// "log"
	// "reflect"
	"slices"
	// "strconv"
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
	// ModalPage

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
	BackButton
	ViewLog
	// Modal

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
	// ModalPage:    "modal",
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
	BackButton:          "Back",
	ViewLog:             "ViewLog",
	// Modal:               "Modal",
}

type gui struct {
	tvApp     *tview.Application
	pages     *tview.Pages
	layouts   map[Layout]tview.Primitive
	widgets   map[Widget]tview.Primitive
	lEForm    *logEventForm
	logGroup  logGroup
	logStream logStream
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
	startYear          int
	startMonth         time.Month
	startDay           int
	startHour          int
	startMinute        int
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
	// g.widgets[Modal] = tview.NewModal().
	// 	SetText("Do you want to quit the application?").
	// 	AddButtons([]string{"Quit", "Cancel"})
	g.pages = tview.NewPages().
		AddPage(pageNames[LogGroupPage], g.layouts[LogGroupLayout], true, true).
		AddPage(pageNames[LogEventPage], g.layouts[LogEventLayout], true, false)
		// AddPage(pageNames[ModalPage], g.widgets[Modal], false, false)

	g.setKeybinding(aw)
}

func (g *gui) setLogEventLayout() {
	g.setLogEventWidget()
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
		AddItem(g.widgets[BackButton],
			2, 3,
			1, 1,
			0, 100,
			false).
		// Log View
		AddItem(g.widgets[ViewLog],
			3, 0,
			1, 5,
			0, 100,
			false)
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
	g.widgets[BackButton] = tview.NewButton("Back Button")

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
		// int32 to string
		retentionDays := fmt.Sprintf("%d", aws.ToInt32(lg.RetentionInDays))
		storedBytes := fmt.Sprintf("%d", aws.ToInt64(lg.StoredBytes))

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

		if slices.Contains(g.lEForm.logStreamNames, lsName) {
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

