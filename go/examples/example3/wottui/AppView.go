package wottui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/hiveot/hivekit/go/examples/wotmodel"
	"github.com/rivo/tview"
)

// The main application view with panels for header, menu main view and footer
// - header shows the current status
// - menu shows quick actions for discovery and viewing TDs
// - main shows details
// - footer shows last action
type AppView struct {
	model *wotmodel.WotModel

	App    *tview.Application
	grid   *tview.Grid
	header *AppHeader
	menu   *AppMenu
	main   *AppMain
	footer *AppFooter
}

func (appView *AppView) handleMenuEvent(ev string) {
	switch ev {
	case MenuEvDiscover:
		appView.StartDiscovery()

	case MenuEvListTDs:
		appView.main.ShowThings()

	case MenuEvReadTD:
	case MenuEvQuit:
		appView.App.Stop()
	}
}

// Start a discovery and refresh the header and main view.
func (appView *AppView) StartDiscovery() {

	mainView := appView.main.View
	mainView.SetTitle(" Discovery of Directories and Things ")
	mainView.SetText("Starting discovery...\n")
	go func() {
		// TODO use a callback to update UI as results come in
		appView.model.Discover()
		appView.main.ShowDiscoRecords()

		// refresh
		appView.App.QueueUpdateDraw(func() {
			appView.header.Refresh()
			appView.main.ShowDiscoRecords()
		})

	}()
}

func (appView *AppView) Run() {

	if err := appView.App.SetRoot(appView.grid, true).
		SetFocus(appView.menu.View).
		// EnableMouse(true).
		Run(); err != nil {
		panic(err)
	}

}

// Create a new instance of the application view
func NewAppView(model *wotmodel.WotModel) *AppView {

	app := tview.NewApplication()
	header := NewAppHeader(model)
	menu := NewAppMenu(model)
	main := NewAppMain(model)
	footer := NewAppFooter(model)

	grid := tview.NewGrid().
		SetRows(3, 0, 3).
		SetColumns(20, 0).
		SetBorders(false).
		AddItem(header.View, 0, 0, 1, 2, 0, 0, false).
		AddItem(footer.View, 2, 0, 1, 2, 0, 0, false).
		AddItem(menu.View, 1, 0, 1, 1, 0, 0, true).
		AddItem(main.View, 1, 1, 1, 1, 0, 0, false)

	appView := &AppView{
		model:  model,
		App:    app,
		grid:   grid,
		header: header,
		main:   main,
		menu:   menu,
		footer: footer,
	}

	// capture global key events
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case 'd':
			app.SetFocus(menu.View)
		case 'q':
			app.Stop()
		case tcell.KeyTab:
			if menu.View.HasFocus() {
				app.SetFocus(main.View)
			} else {
				app.SetFocus(menu.View)
			}
		}
		return event
	})

	menu.SetHandler(appView.handleMenuEvent)

	return appView
}
