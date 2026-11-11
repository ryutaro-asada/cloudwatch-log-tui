// Package view manages the user interface components for the CloudWatch Log TUI.
// It provides the structure and organization of pages, layouts, and widgets.
package view

import "github.com/rivo/tview"

// View represents the main view structure containing all UI components.
type View struct {
	// Root holds the header, the pages and the status line, and is the
	// application root.
	Root    *tview.Flex
	Pages   *Pages
	Layouts *Layouts
	Widgets *Widgets
}

// New creates and initializes a new View instance with all UI components.
// It sets up widgets, layouts, and pages in the correct initialization order.
func New() *View {
	v := &View{
		Pages:   &Pages{},
		Layouts: &Layouts{},
		Widgets: &Widgets{},
	}
	v.Widgets.setUp()
	v.Layouts.setUp(v.Widgets)
	v.Pages.setUp(v.Layouts)
	v.Root = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(v.Widgets.Header, 1, 0, false).
		AddItem(v.Pages, 0, 1, true).
		AddItem(v.Widgets.Status, 1, 0, false)
	return v
}
