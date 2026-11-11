package app

import (
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	awsr "github.com/ryutaro-asada/cloudwatch-log-tui/internal/aws"
	"github.com/ryutaro-asada/cloudwatch-log-tui/internal/state"
)

const (
	nextPageText   = "NextPage ..."
	prevPageText   = "... PrevPage"
	allStreamsText = "All Log Streams"
	noValue        = "-"
)

type rowKind int

const (
	rowItem rowKind = iota
	rowPrev
	rowNext
	rowAllStreams
)

// rowRef identifies a table row. It is stored as the reference of the row's
// first cell, so a log group or stream whose name equals one of the paging
// labels is still handled as an item.
type rowRef struct {
	kind rowKind
	name string
}

func rowRefAt(table *tview.Table, row int) (rowRef, bool) {
	ref, ok := table.GetCell(row, 0).GetReference().(rowRef)
	return ref, ok
}

type column struct {
	header    string
	expansion int
}

var (
	logGroupColumns = []column{
		{"Name", 7},
		{"RetentionDays", 1},
		{"StoredBytes", 1},
	}
	logStreamColumns = []column{
		{"Selected", 1},
		{"Name", 10},
		{"LastEventTime", 2},
		{"FirstEventTime", 2},
	}
)

func setHeader(table *tview.Table, columns []column) {
	for i, c := range columns {
		table.SetCell(0, i, &tview.TableCell{
			Text:            c.header,
			NotSelectable:   true,
			Align:           tview.AlignLeft,
			Color:           tcell.ColorWhite,
			BackgroundColor: tcell.ColorDefault,
			Attributes:      tcell.AttrBold,
		})
	}
}

func setRow(table *tview.Table, row int, columns []column, ref rowRef, color tcell.Color, texts ...string) {
	for i, c := range columns {
		var text string
		if i < len(texts) {
			text = texts[i]
		}
		cell := tview.NewTableCell(text).
			SetTextColor(color).
			SetMaxWidth(1).
			SetExpansion(c.expansion)
		if i == 0 {
			cell.SetReference(ref)
		}
		table.SetCell(row, i, cell)
	}
}

func (a *App) setLogGroupToGui() {
	table := a.view.Widgets.LogGroup.Table
	table.Clear()
	setHeader(table, logGroupColumns)
	row := 1

	if a.state.LogGroup.HasPrev() {
		setRow(table, row, logGroupColumns, rowRef{kind: rowPrev}, tcell.ColorLightSalmon, prevPageText)
		row++
	}

	for _, lg := range a.logGroups {
		name := aws.ToString(lg.LogGroupName)
		setRow(table, row, logGroupColumns, rowRef{kind: rowItem, name: name}, tcell.ColorLightGreen,
			name, a.retentionText(lg.RetentionInDays), int64Text(lg.StoredBytes))
		row++
	}

	if a.state.LogGroup.HasNext() {
		setRow(table, row, logGroupColumns, rowRef{kind: rowNext}, tcell.ColorLightSteelBlue, nextPageText)
	}
}

// retentionText formats RetentionInDays. DescribeLogGroups leaves it unset
// both for groups that never expire and, when LogGroupNamePattern is used,
// for every group.
func (a *App) retentionText(days *int32) string {
	switch {
	case days != nil:
		return strconv.Itoa(int(*days))
	case a.logGroupsFiltered:
		return noValue
	default:
		return "never"
	}
}

func int64Text(v *int64) string {
	if v == nil {
		return noValue
	}
	return strconv.FormatInt(*v, 10)
}

func timestampText(ms *int64) string {
	if ms == nil {
		return noValue
	}
	return time.UnixMilli(*ms).Local().Format("2006-01-02 15:04:05")
}

func (a *App) setLogStreamToGui() {
	table := a.view.Widgets.LogStream.Table
	table.Clear()
	setHeader(table, logStreamColumns)
	row := 1

	if a.state.LogStream.HasPrev() {
		setRow(table, row, logStreamColumns, rowRef{kind: rowPrev}, tcell.ColorLightSalmon, "", prevPageText)
		row++
	}

	setRow(table, row, logStreamColumns, rowRef{kind: rowAllStreams}, tcell.ColorLightGreen, "", allStreamsText)
	row++

	selected := a.state.LogEvent.GetLogStreamsSelected()
	for _, ls := range a.logStreams {
		name := aws.ToString(ls.LogStreamName)
		mark := ""
		if slices.Contains(selected, name) {
			mark = "x"
		}
		setRow(table, row, logStreamColumns, rowRef{kind: rowItem, name: name}, tcell.ColorLightGreen,
			mark, name, timestampText(ls.LastEventTimestamp), timestampText(ls.FirstEventTimestamp))
		row++
	}

	if a.state.LogStream.HasNext() {
		setRow(table, row, logStreamColumns, rowRef{kind: rowNext}, tcell.ColorLightSteelBlue, "", nextPageText)
	}
}

// orderText describes the order of events fetched with newestFirst when
// requested was asked for.
func orderText(requested state.Order, newestFirst bool) string {
	switch {
	case newestFirst:
		return "newest first"
	case requested == state.NewestFirst:
		return "oldest first (newest first needs a start time on or after " +
			awsr.NewestFirstMinStart.Format("2006-01-02 15:04 MST") + ")"
	default:
		return "oldest first"
	}
}

// writeQuery prints the parameters of a log event request.
func writeQuery(w io.Writer, input *awsr.LogEventInput, order string) {
	const layout = "2006-01-02 15:04"
	streams := "ALL"
	if len(input.LogStreamNames) > 0 {
		streams = strings.Join(input.LogStreamNames, ", ")
	}

	fmt.Fprintf(w, "------------------------------------- \n")
	fmt.Fprintf(w, "[YOUR SETTING]\n")
	fmt.Fprintf(w, "LogGroup: %s\n", input.LogGroupName)
	fmt.Fprintf(w, "LogStreams: %s\n", streams)
	fmt.Fprintf(w, "FilterPattern: %s\n", input.FilterPattern)
	fmt.Fprintf(w, "%s\n", input.StartTime.Format(layout))
	fmt.Fprintf(w, "    ~     \n")
	fmt.Fprintf(w, "%s\n", input.EndTime.Format(layout))
	fmt.Fprintf(w, "Order: %s\n", order)
	fmt.Fprintf(w, "MaxEvent: %d\n", awsr.MaxLogEvents)
	fmt.Fprintf(w, "You can get all log events, oldest first, by pressing 'Save Button'.\n")
	fmt.Fprintf(w, "------------------------------------- \n")
}

func (a *App) setLogEventToGui(input *awsr.LogEventInput, output *awsr.LogEventOutput, order string) {
	var b strings.Builder
	writeQuery(&b, input, order)

	if len(output.LogEvents) == 0 {
		b.WriteString("no events\n")
	}
	for _, event := range output.LogEvents {
		b.WriteString(awsr.FormatMessage(event))
	}
	if output.NextToken != nil {
		fmt.Fprintf(&b, "------------------------------------- \n")
		which := "oldest"
		if input.NewestFirst {
			which = "newest"
		}
		fmt.Fprintf(&b, "Showing the %s %d events. Press 'Save Button' to get all of them.\n", which, len(output.LogEvents))
	}

	textView := a.view.Widgets.LogEvent.ViewLog
	textView.SetText(b.String())
	textView.ScrollToBeginning()
}

func (a *App) setHints(requested state.Order, newestFirst bool) {
	order := tview.Escape(orderText(requested, newestFirst))
	if requested == state.NewestFirst && !newestFirst {
		order = "[yellow]" + order + "[-]"
	}
	var presets strings.Builder
	for _, p := range timePresets {
		fmt.Fprintf(&presets, " %c:%s", p.key, p.label)
	}
	a.view.Widgets.LogEvent.Hints.SetText("Order (o): " + order + "    Last:" + presets.String())
}

// initTableRowPosition selects a row after a page was loaded: the first item
// after moving to the next page, the last item after moving back, and the
// first row otherwise.
func (a *App) initTableRowPosition(table *tview.Table, direct state.Direction) {
	selectRow := 1
	switch direct {
	case state.Next:
		for row := 1; row < table.GetRowCount(); row++ {
			if ref, ok := rowRefAt(table, row); ok && ref.kind == rowItem {
				selectRow = row
				break
			}
		}
	case state.Prev:
		for row := table.GetRowCount() - 1; row >= 1; row-- {
			if ref, ok := rowRefAt(table, row); ok && ref.kind == rowItem {
				selectRow = row
				break
			}
		}
	}
	table.Select(selectRow, 0)
}
