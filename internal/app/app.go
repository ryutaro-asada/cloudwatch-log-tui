// Package app provides the main application logic for the CloudWatch Log TUI.
// It manages the user interface, AWS client interactions, and state management.
package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	cwlTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	awsr "github.com/ryutaro-asada/cloudwatch-log-tui/internal/aws"
	"github.com/ryutaro-asada/cloudwatch-log-tui/internal/state"
	"github.com/ryutaro-asada/cloudwatch-log-tui/internal/view"
)

// App represents the main UI application
type App struct {
	tvApp     *tview.Application
	view      *view.View
	state     *state.UIState
	awsClient *awsr.Client
	ctx       context.Context

	// The fields below are only accessed on the tview event goroutine
	// (input handlers and QueueUpdateDraw callbacks), so they need no lock.
	logGroupReq  request
	logStreamReq request
	logEventReq  request
	// logGroups and logStreams hold the page currently shown in each table.
	logGroups         []cwlTypes.LogGroup
	logGroupsFiltered bool
	logStreams        []cwlTypes.LogStream
	saving            bool
	// firstYear is the first option of the year drop-downs.
	firstYear int
	// syncingWidgets is set while the app itself changes drop-down options,
	// so the selected handlers do not reload events once per drop-down.
	syncingWidgets bool
}

// request tracks the latest request of one kind. Starting a new request
// cancels the previous one, and finish reports whether a response still
// belongs to the latest request, so responses that arrive out of order are
// dropped.
type request struct {
	gen    uint64
	cancel context.CancelFunc
}

func (r *request) start(parent context.Context) (context.Context, uint64) {
	r.abort()
	ctx, cancel := context.WithCancel(parent)
	r.cancel = cancel
	return ctx, r.gen
}

// abort cancels the in-flight request, if any, and invalidates its response.
func (r *request) abort() {
	if r.cancel != nil {
		r.cancel()
		r.cancel = nil
	}
	r.gen++
}

func (r *request) finish(gen uint64) bool {
	if gen != r.gen {
		return false
	}
	if r.cancel != nil {
		r.cancel()
		r.cancel = nil
	}
	return true
}

// Run starts the TUI application and runs the main event loop.
// It returns an error if the application fails to start or encounters a fatal error.
func (a *App) Run() error {
	return a.tvApp.SetRoot(a.view.Root, true).
		EnableMouse(true).
		SetFocus(a.view.Widgets.LogGroup.Table).
		Run()
}

// Stop stops the TUI and restores the terminal. It is safe to call from any
// goroutine and after Run has returned.
func (a *App) Stop() {
	a.tvApp.Stop()
}

// New creates a new App instance with the provided context and AWS client.
// It initializes the application state, view components, and key bindings.
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

func (a *App) showError(err error) {
	log.Print(err)
	a.view.Widgets.Status.SetTextColor(tcell.ColorRed).SetText("Error: " + err.Error())
}

func (a *App) showInfo(msg string) {
	a.view.Widgets.Status.SetTextColor(tcell.ColorDefault).SetText(msg)
}

func (a *App) clearStatus() {
	a.view.Widgets.Status.SetText("")
}

// accountTimeout bounds GetCallerIdentity, including credential resolution,
// so a stuck credential provider shows an error instead of "loading" forever.
const accountTimeout = 30 * time.Second

func (a *App) setHeader(account string) {
	a.view.Widgets.Header.SetText(fmt.Sprintf("Profile: [yellow]%s[-]  Region: [yellow]%s[-]  Account: %s",
		tview.Escape(a.awsClient.Profile()), tview.Escape(a.awsClient.Region()), account))
}

// LoadAccount shows the profile and region in the header and fetches the
// account ID in the background. It must be called on the event goroutine, or
// before Run.
func (a *App) LoadAccount() {
	a.setHeader("loading...")
	go func() {
		ctx, cancel := context.WithTimeout(a.ctx, accountTimeout)
		defer cancel()
		account, err := a.awsClient.AccountID(ctx)
		a.tvApp.QueueUpdateDraw(func() {
			if err != nil {
				log.Print(err)
				a.setHeader("[red]" + tview.Escape(err.Error()) + "[-]")
				return
			}
			a.setHeader("[yellow]" + tview.Escape(account) + "[-]")
		})
	}()
}

// LoadLogGroups fetches the page of log groups in direction direct and shows
// it in the log group table. It must be called on the event goroutine, or
// before Run.
func (a *App) LoadLogGroups(direct state.Direction) {
	input := &awsr.LogGroupInput{}
	page, ok := a.state.LogGroup.BeforeGet(input, direct)
	if !ok {
		return
	}
	if err := awsr.ValidateLogGroupNamePattern(input.FilterPattern); err != nil {
		a.logGroupReq.abort()
		a.showError(err)
		return
	}
	ctx, gen := a.logGroupReq.start(a.ctx)
	input.Ctx = ctx

	go func() {
		output, err := a.awsClient.GetLogGroups(input)
		a.tvApp.QueueUpdateDraw(func() {
			if !a.logGroupReq.finish(gen) {
				return
			}
			if err != nil {
				a.showError(fmt.Errorf("unable to list log groups: %w", err))
				return
			}
			a.clearStatus()
			table := a.view.Widgets.LogGroup.Table
			if !a.state.LogGroup.AfterGet(page, output) {
				a.setLogGroupToGui()
				table.Select(table.GetRowCount()-1, 0)
				return
			}
			a.logGroups = output.LogGroups
			a.logGroupsFiltered = input.FilterPattern != ""
			a.setLogGroupToGui()
			a.initTableRowPosition(table, direct)
		})
	}()
}

// LoadLogStreams fetches the page of log streams in direction direct for the
// selected log group. It must be called on the event goroutine.
func (a *App) LoadLogStreams(direct state.Direction) {
	input := &awsr.LogStreamInput{}
	page, ok := a.state.LogStream.BeforeGet(input, direct)
	if !ok {
		return
	}
	ctx, gen := a.logStreamReq.start(a.ctx)
	input.Ctx = ctx

	go func() {
		output, err := a.awsClient.GetLogStreams(input)
		a.tvApp.QueueUpdateDraw(func() {
			if !a.logStreamReq.finish(gen) {
				return
			}
			if err != nil {
				a.showError(fmt.Errorf("unable to list log streams: %w", err))
				return
			}
			a.clearStatus()
			table := a.view.Widgets.LogStream.Table
			if !a.state.LogStream.AfterGet(page, output) {
				a.setLogStreamToGui()
				table.Select(table.GetRowCount()-1, 0)
				return
			}
			a.logStreams = output.LogStreams
			a.setLogStreamToGui()
			a.initTableRowPosition(table, direct)
		})
	}()
}

// LoadLogEvents fetches up to awsr.MaxLogEvents log events for the current
// query and shows them in the log viewer. It must be called on the event
// goroutine.
func (a *App) LoadLogEvents() {
	textView := a.view.Widgets.LogEvent.ViewLog
	textView.Clear()

	input := &awsr.LogEventInput{}
	requested := a.state.LogEvent.Order()
	if err := a.state.LogEvent.BeforeGet(input); err != nil {
		a.logEventReq.abort()
		a.setHints(requested, requested == state.NewestFirst)
		log.Print(err)
		fmt.Fprintf(textView, "Error: %v\n", err)
		return
	}
	a.setHints(requested, input.NewestFirst)
	order := orderText(requested, input.NewestFirst)
	ctx, gen := a.logEventReq.start(a.ctx)
	input.Ctx = ctx
	fmt.Fprintln(textView, "Now Loading... ")

	go func() {
		output, err := a.awsClient.GetLogEvents(input)
		a.tvApp.QueueUpdateDraw(func() {
			if !a.logEventReq.finish(gen) {
				return
			}
			if err != nil {
				log.Print(err)
				textView.Clear()
				writeQuery(textView, input, order)
				fmt.Fprintf(textView, "Error: %v\n", err)
				return
			}
			a.setLogEventToGui(input, output, order)
		})
	}()
}

// SaveLogEvents writes all log events of the current query to the output
// file in the background and reports the result in the status line.
// It must be called on the event goroutine.
func (a *App) SaveLogEvents() {
	if a.saving {
		a.showError(errors.New("a save is already running"))
		return
	}
	input := &awsr.LogEventInput{Ctx: a.ctx}
	if err := a.state.LogEvent.BeforeGet(input); err != nil {
		a.showError(err)
		return
	}
	a.saving = true
	a.showInfo("Saving log events...")

	go func() {
		path, count, err := a.awsClient.WriteLogEvents(input)
		a.tvApp.QueueUpdateDraw(func() {
			a.saving = false
			if err != nil {
				a.showError(fmt.Errorf("unable to save log events to %s: %w", path, err))
				return
			}
			log.Printf("saved %d log events to %s", count, path)
			a.showInfo(fmt.Sprintf("Saved %d log events to %s", count, path))
		})
	}()
}
