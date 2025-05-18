package view

import (
	"github.com/rivo/tview"
)

// Page represents different application pages
type Page int

const (
	LogGroupAndStreamPage Page = iota
	LogEventPage
)

var PageNames = map[Page]string{
	LogGroupAndStreamPage: "logGroups",
	LogEventPage:          "logEvents",
}

type Pages struct {
	*tview.Pages
}

func (p *Pages) setUp(l *Layouts) {
	p.Pages = tview.NewPages().
		AddPage(PageNames[LogGroupAndStreamPage], l.LogGroupAndStream, true, true).
		AddPage(PageNames[LogEventPage], l.LogEvent, true, false)
}
