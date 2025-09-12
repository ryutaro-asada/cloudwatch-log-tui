// Package view manages the user interface components for the CloudWatch Log TUI.
package view

import (
	"github.com/rivo/tview"
)

// Page represents different application pages
type Page int

const (
	// LogGroupAndStreamPage displays log groups and streams selection interface
	LogGroupAndStreamPage Page = iota
	// LogEventPage displays the log events viewer interface
	LogEventPage
)

// PageNames provides string identifiers for each page type.
// These are used internally by tview for page management.
var PageNames = map[Page]string{
	LogGroupAndStreamPage: "logGroups",
	LogEventPage:          "logEvents",
}

// Pages manages the different screens in the application.
// It embeds tview.Pages to handle page switching and display.
type Pages struct {
	*tview.Pages
}

// setUp initializes the Pages with layouts for each screen.
// The LogGroupAndStreamPage is set as the initially visible page.
func (p *Pages) setUp(l *Layouts) {
	p.Pages = tview.NewPages().
		AddPage(PageNames[LogGroupAndStreamPage], l.LogGroupAndStream, true, true).
		AddPage(PageNames[LogEventPage], l.LogEvent, true, false)
}
