package main

import (
	"fmt"
	"log"
	"slices"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	cwl "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Constants and enum types
const (
	NoShortcut rune = 0
	NextPage        = "NextPage ..."
	PrevPage        = "... PrevPage"
)

// Direction specifies navigation direction in logs
type Direction int

const (
	Next Direction = iota
	Home
	Prev
)

// Page represents different application pages
type Page int

const (
	LogGroupPage Page = iota
	LogEventPage
)

var pageNames = map[Page]string{
	LogGroupPage: "logGroups",
	LogEventPage: "logEvents",
}

type gui struct {
	tvApp     *tview.Application
	pages     *tview.Pages
	layouts   map[Layout]tview.Primitive
	widgets   map[Widget]tview.Primitive
	lEForm    *logEventForm
	logGroup  *logGroup
	logStream *logStream
}

type logGroup struct {
	filterPatern string
	direction    Direction
	currentPage  int
	hasNext      bool
	hasPrev      bool
	pageTokens   map[int]*string
}

type logStream struct {
	prefixPatern string
	direction    Direction
	logGroupName string
	currentPage  int
	hasNext      bool
	hasPrev      bool
	pageTokens   map[int]*string
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

type logEventInput struct {
	awsInput   *cwl.FilterLogEventsInput
	outputFile string
}

func (g *gui) setGui(aw *awsResource) {
	g.setLogGroupLayout()
	g.setLogEventLayout()
	g.pages = tview.NewPages().
		AddPage(pageNames[LogGroupPage], g.layouts[LogGroupLayout], true, true).
		AddPage(pageNames[LogEventPage], g.layouts[LogEventLayout], true, false)

	g.setKeybinding(aw)
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

	if g.logGroup.hasPrev {
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

	if g.logGroup.hasNext {
		log.Printf("hasNext: %v", g.logGroup.hasNext)
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

	if g.logStream.hasPrev {
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

	if g.logStream.hasNext {
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
