package wottui

import (
	"github.com/hiveot/hivekit/go/examples/wotmodel"
	"github.com/rivo/tview"
)

const MenuEvDiscover = "discover"
const MenuEvReadTD = "readTD"
const MenuEvListTDs = "listTDs"
const MenuEvQuit = "quit"

// The application menu panel with the main actions and pages
type AppMenu struct {
	View *tview.List

	model *wotmodel.WotModel

	handler func(ev string)
}

func (menu *AppMenu) SetHandler(h func(ev string)) {
	menu.handler = h
}

func (menu *AppMenu) submit(ev string) {
	if menu.handler != nil {
		menu.handler(ev)
	}
}

// Create a new instance of the application view
// evHandler handles menu requests
func NewAppMenu(model *wotmodel.WotModel) *AppMenu {
	view := tview.NewList()

	menu := &AppMenu{
		View:  view,
		model: model,
	}

	view.AddItem("Discover", "(re)discover", 'd', func() {
		menu.submit(MenuEvDiscover)
	})
	view.AddItem("List TDs", "rotates views", 'l', func() {
		menu.submit(MenuEvListTDs)
		// StartDiscovery(app, cli, main)
	})
	view.AddItem("Read TD", "", 'r', func() {
		menu.submit(MenuEvReadTD)
		// StartDiscovery(app, cli, main)
	})
	view.AddItem("Quit", "", 'q', func() {
		menu.submit(MenuEvQuit)
	})
	view.SetBorder(true)

	return menu
}
