package view

type View struct {
	Pages   *Pages
	Layouts *Layouts
	Widgets *Widgets
}

// setupUI initializes the UI layout
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
