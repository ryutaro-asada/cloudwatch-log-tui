// Package view manages the user interface components for the CloudWatch Log TUI.
// It provides the structure and organization of pages, layouts, and widgets.
package view

// View represents the main view structure containing all UI components.
// It serves as the root container for pages, layouts, and widgets.
type View struct {
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
	return v
}
