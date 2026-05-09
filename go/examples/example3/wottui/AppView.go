package wottui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/hiveot/hivekit/go/examples/wotmodel"
	"github.com/rivo/tview"
)

const (
	PageLanding   = "landing"
	PageThings    = "things"
	PageDiscovery = "discovery"
)

// The main application view with panels for header, menu main view and footer
// - header shows the current status
// - menu shows quick actions for discovery and viewing TDs
// - main shows details
// - footer shows last action
type TuiApp struct {
	model *wotmodel.WotModel

	View *tview.Application

	pages *tview.Pages

	landingPage *LandingPage
	discoPage   *DiscoPage
	thingsPage  *ThingsPage

	grid   *tview.Grid
	header *AppHeader
}

func (tui *TuiApp) handleEvent(ev string) {
	switch ev {
	case MenuEvDiscover:
		tui.StartDiscovery()

	case MenuEvListTDs:
		tui.thingsPage.Refresh()
		tui.pages.SwitchToPage(PageThings)

	case MenuEvReadTD:
	case MenuEvQuit:
		tui.View.Stop()
	}
}

// Show the loaded things in the main view
func (tui *TuiApp) ShowThings() {
	tui.thingsPage.Refresh()
	tui.pages.SwitchToPage(PageThings)
}

// Show the discovery records
func (tui *TuiApp) ShowDiscovery() {
	tui.discoPage.Refresh()
	tui.pages.SwitchToPage(PageDiscovery)
}

// Start a discovery and refresh the header and main view.
func (tui *TuiApp) StartDiscovery() {

	tui.landingPage.SetTitle(" Discovery of Directories and Things ")
	tui.landingPage.SetText("\nStarting discovery...")
	tui.pages.SwitchToPage(PageLanding)

	go func() {
		// TODO use a callback to update UI as results come in
		tui.model.Discover()
		tui.ShowDiscovery()

		// refresh
		tui.View.QueueUpdateDraw(func() {
			tui.header.Refresh()
			tui.discoPage.Refresh()
		})

	}()
}

func (tui *TuiApp) NextPage() {
	var name string
	var pageNr int
	var pageNames = []string{PageLanding, PageDiscovery, PageThings}

	// determine the next page to show
	currentPageName, _ := tui.pages.GetFrontPage()
	// pageNames := tui.pages.GetPageNames(false)
	for pageNr, name = range pageNames {
		if name == currentPageName {
			break
		}
	}
	pageNr++
	if pageNr >= len(pageNames) {
		pageNr = 1 // do not show the landing page when rotating through pages
	}
	pageName := pageNames[pageNr]
	tui.pages.SwitchToPage(pageName)
}

func (tui *TuiApp) Run() {

	// tui.menu.SetHandler(tui.handleMenuEvent)
	tui.header.SetHandler(tui.handleEvent)

	// capture global key events
	tui.View.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'd':
			tui.StartDiscovery()
		case 't':
			tui.ShowThings()
		case 'q':
			tui.View.Stop()
		}
		switch event.Key() {
		// tab-key rotates through pages
		case tcell.KeyTab:
			tui.NextPage()
			// if tui.menu.View.HasFocus() {
			// 	tui.View.SetFocus(tui.main.View)
			// } else {
			// 	tui.View.SetFocus(tui.menu.View)
			// }
		}
		return event
	})
	tui.pages.SwitchToPage(PageLanding)
	tui.landingPage.Refresh()
	// tui.View.SetFocus(tui.menu.View)

	if err := tui.View.Run(); err != nil {
		panic(err)
	}
}

// Create a new instance of the application view
func NewAppView(model *wotmodel.WotModel) *TuiApp {

	appView := tview.NewApplication()
	header := NewAppHeader(model)
	header.View.SetBorderColor(tcell.ColorDarkGray)

	pages := tview.NewPages()
	LandingPage := NewLandingPage(model)
	pages.AddPage(PageLanding, LandingPage, true, true)
	discoPage := NewDiscoPage(model)
	pages.AddPage(PageDiscovery, discoPage, true, false)
	thingsPage := NewThingsPage(model)
	pages.AddPage(PageThings, thingsPage, true, false)

	// footer := NewAppFooter(model)
	// footer.View.SetBorderColor(tcell.ColorDarkGray)

	grid := tview.NewGrid().
		SetRows(3, 0).
		SetColumns(0).
		// SetBorders(true).
		AddItem(header.View, 0, 0, 1, 1, 0, 0, false).
		AddItem(pages, 1, 0, 1, 1, 0, 0, true)

	// grid := tview.NewFlex().SetDirection(tview.FlexRow).
	// 	AddItem(header.View, 3, 0, false).
	// 	AddItem(pages, 0, 1, true)

	appView.SetRoot(grid, true).EnableMouse(true)

	tuiApp := &TuiApp{
		model:       model,
		View:        appView,
		grid:        grid,
		header:      header,
		pages:       pages,
		landingPage: LandingPage,
		discoPage:   discoPage,
		thingsPage:  thingsPage,
		// main:   main,
		// menu:   menu,
		// footer: footer,
	}

	return tuiApp
}
