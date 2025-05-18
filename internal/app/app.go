package app

import (
	"context"
	"fmt"
	"log"
	"slices"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	awsr "github.com/ryutaro-asada/cloudwatch-log-tui/internal/aws"
	"github.com/ryutaro-asada/cloudwatch-log-tui/internal/state"
	"github.com/ryutaro-asada/cloudwatch-log-tui/internal/view"
)

// Constants and enum types
const (
	NoShortcut rune = 0
	NextPage        = "NextPage ..."
	PrevPage        = "... PrevPage"
)

// App represents the main UI application
type App struct {
	tvApp     *tview.Application
	view      *view.View
	state     *state.UIState
	awsClient *awsr.Client
	ctx       context.Context
}

func (a *App) Run() error {
	return a.tvApp.SetRoot(a.view.Pages, true).
		EnableMouse(true).
		SetFocus(a.view.Widgets.LogGroup.Table).
		Run()
}

// New creates a new UI application
func New(ctx context.Context, awsClient *awsr.Client) *App {
	app := &App{
		tvApp:     tview.NewApplication(),
		awsClient: awsClient,
		ctx:       ctx,
	}
	app.state = state.New()
	app.view = view.New()
	app.setUpKeyBindings()
	return app
}

// LoadLogGroups loads and displays the log groups
func (a *App) LoadLogGroups(direct state.Direction) {
	go func() {
		input := &awsr.LogGroupInput{
			Ctx: a.ctx,
		}
		a.state.LogGroup.BeforeGet(input, direct)
		output, err := a.awsClient.GetLogGroups(input)
		if err != nil {
			log.Fatalf("unable to list tables, %v", err)
		}
		a.state.LogGroup.AfterGet(output, direct)

		a.tvApp.QueueUpdateDraw(func() {
			a.setLogGroupToGui(output)
			table := a.view.Widgets.LogGroup.Table
			a.initTableRowPosition(table, direct)
		})
	}()
}

// loadLogStreams loads and displays the log streams for a given group
func (a *App) LoadLogStreams(direct state.Direction) {
	go func() {
		input := &awsr.LogStreamInput{
			Ctx: a.ctx,
		}
		a.state.LogStream.BeforeGet(input, direct)
		output, err := a.awsClient.GetLogStreams(input)
		if err != nil {
			log.Fatalf("unable to list tables, %v", err)
		}
		a.state.LogStream.AfterGet(output, direct)

		a.tvApp.QueueUpdateDraw(func() {
			a.setLogStreamToGui(output)
			table := a.view.Widgets.LogStream.Table
			a.initTableRowPosition(table, direct)
		})
	}()
}

// loadLogEvents loads and displays the log events
func (a *App) LoadLogEvents() {
	textView := a.view.Widgets.LogEvent.ViewLog
	textView.Clear()
	fmt.Fprintln(textView, "Now Loading... ")
	go func() {
		input := &awsr.LogEventInput{
			Ctx: a.ctx,
		}
		a.state.LogEvent.BeforeGet(input)
		output, err := a.awsClient.GetLogEvents(input)
		if err != nil {
			log.Fatalf("unnable to write logs, %v", err)
		}

		a.tvApp.QueueUpdateDraw(func() {
			a.setLogEventToGui(output)
		})
	}()
}

func (a *App) setDefaultDropDownLogEvents() {
	_, startMonth, startDay, startHour, startMinute := a.state.LogEvent.GetStartTime()
	a.view.Widgets.LogEvent.StartYear.
		SetCurrentOption(1)
	a.view.Widgets.LogEvent.StartMonth.
		SetCurrentOption(startMonth - 1)
	a.view.Widgets.LogEvent.StartDay.
		SetCurrentOption(startDay - 1)
	a.view.Widgets.LogEvent.StartHour.
		SetCurrentOption(startHour)
	a.view.Widgets.LogEvent.StartMinute.
		SetCurrentOption(startMinute)

	_, endMonth, endDay, endHour, endMinute := a.state.LogEvent.GetEndTime()
	a.view.Widgets.LogEvent.EndYear.
		SetCurrentOption(1)
	a.view.Widgets.LogEvent.EndMonth.
		SetCurrentOption(endMonth - 1)
	a.view.Widgets.LogEvent.EndDay.
		SetCurrentOption(endDay - 1)
	a.view.Widgets.LogEvent.EndHour.
		SetCurrentOption(endHour)
	a.view.Widgets.LogEvent.EndMinute.
		SetCurrentOption(endMinute)
}

func (a *App) refreshLogStreamTable() {
	lsTable := a.view.Widgets.LogStream.Table
	lsNames := make([]string, 0)
	lsLastEvents := make([]string, 0)
	lsFirstEvents := make([]string, 0)

	for i := 0; i < lsTable.GetRowCount(); i++ {
		lsName := lsTable.GetCell(i, 1).Text
		lsLastEvent := lsTable.GetCell(i, 2).Text
		lsFirstEvent := lsTable.GetCell(i, 3).Text
		if lsName == "" || lsName == "All Log Streams" || lsName == NextPage || lsName == PrevPage {
			continue
		}
		lsNames = append(lsNames, lsName)
		lsLastEvents = append(lsLastEvents, lsLastEvent)
		lsFirstEvents = append(lsFirstEvents, lsFirstEvent)
	}
	lsOutput := a.awsClient.SetLogStreamOutput(lsNames, lsLastEvents, lsFirstEvents)
	a.setLogStreamToGui(lsOutput)
}

// setLogGroupToGui sets the log group data to the GUI
func (a *App) setLogGroupToGui(aw *awsr.LogGroupOutput) {
	lgTable := a.view.Widgets.LogGroup.Table
	lgTable.Clear()

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

	if a.state.LogGroup.HasPrev() {
		lgTable.SetCell(row, 0, tview.NewTableCell(PrevPage).
			SetTextColor(tcell.ColorLightSalmon).
			SetMaxWidth(1).
			SetExpansion(7))
		row++
	}

	for _, lg := range aw.LogGroups {
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

	if a.state.LogGroup.HasNext() {
		lgTable.SetCell(row, 0, tview.NewTableCell(NextPage).
			SetTextColor(tcell.ColorLightSteelBlue).
			SetMaxWidth(1).
			SetExpansion(7))
	}
}

func (a *App) setLogStreamToGui(aw *awsr.LogStreamOutput) {
	lsTable := a.view.Widgets.LogStream.Table
	lsTable.Clear()

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

	if a.state.LogStream.HasPrev() {
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

	for _, ls := range aw.LogStreams {
		lsName := aws.ToString(ls.LogStreamName)
		lastEventTime := time.Unix(aws.ToInt64(ls.LastEventTimestamp), 0).Local().Format("2006-01-02 15:04:05")
		firstEventTime := time.Unix(aws.ToInt64(ls.FirstEventTimestamp), 0).Local().Format("2006-01-02 15:04:05")
		selectedMark := ""

		if slices.Contains(a.state.LogEvent.GetLogStreamsSelected(), lsName) {
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

	if a.state.LogStream.HasNext() {
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

func (a *App) setLogEventToGui(output *awsr.LogEventOutput) {
	textView := a.view.Widgets.LogEvent.ViewLog
	textView.Clear()
	a.state.LogEvent.Print(textView)

	if len(output.LogEvents) == 0 {
		fmt.Fprintf(textView, "no events\n")
		return
	}

	for _, event := range output.LogEvents {
		fmt.Fprintf(textView, "%s\n", aws.ToString(event.Message))
	}
}

func (a *App) initTableRowPosition(table *tview.Table, direct state.Direction) {
	var selectRow int
	switch direct {
	case state.Next:
		selectRow = 2
	case state.Prev:
		selectRow = table.GetRowCount() - 2
	case state.Home:
		selectRow = 2
	}

	table.Select(selectRow, 0)
}

// func (a *App) SaveLogEvent() {
// 	go func() {
// 		input := &awsr.LogEventInput{
// 			Ctx: a.ctx,
// 		}
// 		a.state.LogEvent.BeforeGet(input)
// 		output, err := a.awsClient.WrireLogEvents(input)
// 		if err != nil {
// 			log.Fatalf("unnable to write logs, %v", err)
// 		}
// 	}()
// }
