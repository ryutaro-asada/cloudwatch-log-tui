package ui

import (
	"context"
	"log"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/ryutaro-asada/cloudwatch-log-tui/internal/aws"
)

// App represents the main UI application
type App struct {
	*tview.Application
	pages      *tview.Pages
	logGroups  *tview.List
	logStreams *tview.List
	logEvents  *tview.TextView
	awsClient  *aws.Client
	ctx        context.Context
}

// New creates a new UI application
func New(ctx context.Context, awsClient *aws.Client) *App {
	app := &App{
		Application: tview.NewApplication(),
		pages:       tview.NewPages(),
		logGroups:   tview.NewList().ShowSecondaryText(false),
		logStreams:  tview.NewList().ShowSecondaryText(false),
		logEvents:   tview.NewTextView().SetDynamicColors(true),
		awsClient:   awsClient,
		ctx:         ctx,
	}

	app.setupUI()
	app.setupKeyBindings()
	return app
}

// setupUI initializes the UI layout
func (a *App) setupUI() {
	// Setup log groups list
	a.logGroups.SetBorder(true).SetTitle("Log Groups")
	a.logGroups.SetSelectedFunc(a.onLogGroupSelected)

	// Setup log streams list
	a.logStreams.SetBorder(true).SetTitle("Log Streams")
	a.logStreams.SetSelectedFunc(a.onLogStreamSelected)

	// Setup log events view
	a.logEvents.SetBorder(true).SetTitle("Log Events")
	a.logEvents.SetScrollable(true)

	// Create layout
	flex := tview.NewFlex().
		AddItem(a.logGroups, 0, 1, true).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(a.logStreams, 0, 1, false).
			AddItem(a.logEvents, 0, 3, false), 0, 3, false)

	a.pages.AddPage("main", flex, true, true)
	a.SetRoot(a.pages, true)
}

// setupKeyBindings initializes the keyboard shortcuts
func (a *App) setupKeyBindings() {
	a.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Global shortcuts
		switch event.Key() {
		case tcell.KeyCtrlC:
			a.Stop()
			return nil
		case tcell.KeyTab:
			a.focusNext()
			return nil
		case tcell.KeyBacktab:
			a.focusPrev()
			return nil
		case tcell.KeyCtrlR:
			// Refresh current view
			if a.logGroups.HasFocus() {
				go a.LoadLogGroups()
			} else if a.logStreams.HasFocus() {
				index := a.logGroups.GetCurrentItem()
				groupName, _ := a.logGroups.GetItemText(index)
				go a.loadLogStreams(groupName)
			} else if a.logEvents.HasFocus() {
				groupName, _ := a.logGroups.GetItemText(a.logGroups.GetCurrentItem())
				streamName, _ := a.logStreams.GetItemText(a.logStreams.GetCurrentItem())
				go a.loadLogEvents(groupName, streamName)
			}
			return nil
		}

		// Handle vim-style navigation
		switch event.Rune() {
		case 'j':
			if a.logGroups.HasFocus() {
				current := a.logGroups.GetCurrentItem()
				if current < a.logGroups.GetItemCount()-1 {
					a.logGroups.SetCurrentItem(current + 1)
					a.onLogGroupSelected(current+1, "", "", 0)
				}
				return nil
			} else if a.logStreams.HasFocus() {
				current := a.logStreams.GetCurrentItem()
				if current < a.logStreams.GetItemCount()-1 {
					a.logStreams.SetCurrentItem(current + 1)
					a.onLogStreamSelected(current+1, "", "", 0)
				}
				return nil
			}
		case 'k':
			if a.logGroups.HasFocus() {
				current := a.logGroups.GetCurrentItem()
				if current > 0 {
					a.logGroups.SetCurrentItem(current - 1)
					a.onLogGroupSelected(current-1, "", "", 0)
				}
				return nil
			} else if a.logStreams.HasFocus() {
				current := a.logStreams.GetCurrentItem()
				if current > 0 {
					a.logStreams.SetCurrentItem(current - 1)
					a.onLogStreamSelected(current-1, "", "", 0)
				}
				return nil
			}
		case 'g':
			if a.logGroups.HasFocus() {
				a.logGroups.SetCurrentItem(0)
				a.onLogGroupSelected(0, "", "", 0)
				return nil
			} else if a.logStreams.HasFocus() {
				a.logStreams.SetCurrentItem(0)
				a.onLogStreamSelected(0, "", "", 0)
				return nil
			} else if a.logEvents.HasFocus() {
				a.logEvents.ScrollToBeginning()
				return nil
			}
		case 'G':
			if a.logGroups.HasFocus() {
				lastIndex := a.logGroups.GetItemCount() - 1
				if lastIndex >= 0 {
					a.logGroups.SetCurrentItem(lastIndex)
					a.onLogGroupSelected(lastIndex, "", "", 0)
				}
				return nil
			} else if a.logStreams.HasFocus() {
				lastIndex := a.logStreams.GetItemCount() - 1
				if lastIndex >= 0 {
					a.logStreams.SetCurrentItem(lastIndex)
					a.onLogStreamSelected(lastIndex, "", "", 0)
				}
				return nil
			} else if a.logEvents.HasFocus() {
				a.logEvents.ScrollToEnd()
				return nil
			}
		}

		// View-specific shortcuts
		if a.logEvents.HasFocus() {
			switch event.Key() {
			case tcell.KeyPgUp:
				row, _ := a.logEvents.GetScrollOffset()
				a.logEvents.ScrollTo(row-1, 0)
				return nil
			case tcell.KeyPgDn:
				row, _ := a.logEvents.GetScrollOffset()
				a.logEvents.ScrollTo(row+1, 0)
				return nil
			case tcell.KeyHome:
				a.logEvents.ScrollToBeginning()
				return nil
			case tcell.KeyEnd:
				a.logEvents.ScrollToEnd()
				return nil
			}
		}

		return event
	})
}

// focusNext cycles forward through the focusable elements
func (a *App) focusNext() {
	if a.logGroups.HasFocus() {
		a.SetFocus(a.logStreams)
	} else if a.logStreams.HasFocus() {
		a.SetFocus(a.logEvents)
	} else {
		a.SetFocus(a.logGroups)
	}
}

// focusPrev cycles backward through the focusable elements
func (a *App) focusPrev() {
	if a.logGroups.HasFocus() {
		a.SetFocus(a.logEvents)
	} else if a.logStreams.HasFocus() {
		a.SetFocus(a.logGroups)
	} else {
		a.SetFocus(a.logStreams)
	}
}

// LoadLogGroups loads and displays the log groups
func (a *App) LoadLogGroups() {
	a.logGroups.Clear()

	groups, err := a.awsClient.GetLogGroups(a.ctx)
	if err != nil {
		log.Printf("Error loading log groups: %v", err)
		return
	}

	for _, group := range groups {
		a.logGroups.AddItem(*group.LogGroupName, "", 0, nil)
	}
}

// loadLogStreams loads and displays the log streams for a given group
func (a *App) loadLogStreams(groupName string) {
	a.logStreams.Clear()
	a.logEvents.Clear()

	streams, err := a.awsClient.GetLogStreams(a.ctx, groupName)
	if err != nil {
		log.Printf("Error loading log streams: %v", err)
		return
	}

	for _, stream := range streams {
		a.logStreams.AddItem(*stream.LogStreamName, "", 0, nil)
	}
}

// loadLogEvents loads and displays the log events
func (a *App) loadLogEvents(groupName, streamName string) {
	a.logEvents.Clear()

	events, err := a.awsClient.GetLogEvents(a.ctx, groupName, streamName, nil, nil)
	if err != nil {
		log.Printf("Error loading log events: %v", err)
		return
	}

	for _, event := range events {
		a.logEvents.Write([]byte(*event.Message + "\n"))
	}
}

// onLogGroupSelected handles log group selection
func (a *App) onLogGroupSelected(index int, _ string, _ string, _ rune) {
	groupName, _ := a.logGroups.GetItemText(index)
	a.loadLogStreams(groupName)
}

// onLogStreamSelected handles log stream selection
func (a *App) onLogStreamSelected(index int, _ string, _ string, _ rune) {
	groupName, _ := a.logGroups.GetItemText(a.logGroups.GetCurrentItem())
	streamName, _ := a.logStreams.GetItemText(index)
	a.loadLogEvents(groupName, streamName)
}
