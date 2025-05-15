package ui

import (
	"github.com/rivo/tview"
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

func (a *App) setUpPages() {
	a.pages = tview.NewPages().
		AddPage(pageNames[LogGroupPage], a.layouts[LogGroupLayout], true, true).
		AddPage(pageNames[LogEventPage], a.layouts[LogEventLayout], true, false)
}
