package wottui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/hiveot/hivekit/go/examples/wotmodel"
	"github.com/rivo/tview"
)

const (
	PageLanding     = "landing"
	PageThings      = "things"
	PageDirectories = "directories"
	PageDiscovery   = "discovery"
	PageModal       = "modal"
)

// menu events
const (
	MenuEvClose           = "close"
	MenuEvDiscover        = "discover"
	MenuEvListTDs         = "listTDs"
	MenuEvNextPage        = "nextPage"
	MenuEvShowDiscovered  = "showDiscovered"
	MenuEvShowDirectory   = "showDirectory"
	MenuEvShowDirectories = "showDirectories"
	MenuEvShowTD          = "showTD"
	MenuEvShowThings      = "showThings"
	MenuEvQuit            = "quit"
)

// The main application view with panels for header, menu main view and footer
// - header shows the current status
// - menu shows quick actions for discovery and viewing TDs
// - main shows details
// - footer shows last action
type TuiApp struct {
	tview.Application

	model *wotmodel.WotModel

	menu *TreeMenu

	pages *tview.Pages

	directoriesPage *DirectoriesPage
	discoPage       *DiscoPage
	landingPage     *LandingPage
	tdPage          *TDPage
	thingsPage      *ThingsPage

	grid   *tview.Grid
	header *AppHeader
	footer *AppFooter
}

func (tui *TuiApp) handleEvent(args ...string) {

	ev := args[0]

	switch ev {
	case MenuEvClose:
		tui.pages.HidePage(PageModal)

	case MenuEvDiscover:
		tui.StartDiscovery()

	case MenuEvListTDs:
		if len(tui.model.GetThings()) > 0 {
			tui.thingsPage.Refresh()
			tui.pages.SwitchToPage(PageThings)
		}

	case MenuEvNextPage:
		if len(tui.model.GetThings()) > 0 {
			tui.NextPage()
			tui.SetFocus(tui.pages)
		}

	case MenuEvShowDiscovered:
		tui.ShowDiscovery()

	case MenuEvShowDirectories:
		tui.ShowDirectories()

	case MenuEvShowThings:
		tui.ShowThings()

	case MenuEvShowTD:
		if len(args) > 1 {
			thingID := args[1]
			tui.ShowThingModal(thingID)
		} else {
			tui.ShowThings()
		}

	case MenuEvQuit:
		tui.Stop()
	}
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

func (tui *TuiApp) ShowDirectories() {
	tui.directoriesPage.Refresh()
	tui.pages.SwitchToPage(PageDirectories)
}

// Show the discovery records
func (tui *TuiApp) ShowDiscovery() {
	tui.discoPage.Refresh()
	tui.pages.SwitchToPage(PageDiscovery)
}

func (tui *TuiApp) ShowThingModal(thingID string) {
	tui.tdPage.Refresh(thingID)
	// tui.pages.SwitchToPage(PageModal)
	// do not switch to the modal page, but just show it on top of the current page
	tui.pages.ShowPage(PageModal)
}

// Show the loaded things in the main view
func (tui *TuiApp) ShowThings() {
	tui.thingsPage.Refresh()
	tui.pages.SwitchToPage(PageThings)
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
		tui.QueueUpdateDraw(func() {
			tui.header.Refresh()
			tui.footer.Refresh()
			tui.menu.Refresh()
			tui.discoPage.Refresh()
		})

	}()
}

// Start the application
func (tui *TuiApp) Run() {

	tui.footer.SetHandler(tui.handleEvent)
	tui.tdPage.SetHandler(tui.handleEvent)
	tui.thingsPage.SetHandler(tui.handleEvent)
	tui.menu.SetHandler(tui.handleEvent)

	// capture global key events
	tui.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'd':
			tui.handleEvent(MenuEvDiscover)
		case 'l':
			tui.handleEvent(MenuEvListTDs)
		case 'q':
			tui.handleEvent(MenuEvQuit)
		}
		switch event.Key() {
		// tab-key switches between menu and pages
		case tcell.KeyTab:
			if tui.menu.HasFocus() {
				tui.SetFocus(tui.pages)
			} else {
				tui.SetFocus(tui.menu)
			}
		}
		return event
	})
	tui.pages.SwitchToPage(PageLanding)
	tui.landingPage.Refresh()
	tui.menu.Refresh()

	// start discovery in the background, this will update the UI when results come in
	go tui.StartDiscovery()

	if err := tui.Application.Run(); err != nil {
		panic(err)
	}
}

// Create a new instance of the application view
func NewAppView(model *wotmodel.WotModel) *TuiApp {

	header := NewAppHeader(model)
	header.View.SetBorderColor(tcell.ColorDarkGray)

	pages := tview.NewPages()

	menu := NewTreeMenu(model)

	directoriesPage := NewDirectoriesPage(model)
	pages.AddPage(PageDirectories, directoriesPage, true, false)

	discoPage := NewDiscoPage(model)
	pages.AddPage(PageDiscovery, discoPage, true, false)

	landingPage := NewLandingPage(model)
	pages.AddPage(PageLanding, landingPage, true, false)

	thingsPage := NewThingsPage(model)
	pages.AddPage(PageThings, thingsPage, true, false)

	footer := NewAppFooter(model)
	footer.View.SetBorderColor(tcell.ColorDarkGray)

	tdPage := NewTDPage(model)
	pages.AddPage(PageModal, tdPage, true, false)

	grid := tview.NewGrid().
		SetRows(3, 0, 1).
		SetColumns(30, 0).
		AddItem(header.View, 0, 0, 1, 2, 0, 0, false).
		AddItem(menu, 1, 0, 1, 1, 0, 0, true).
		AddItem(pages, 1, 1, 1, 1, 0, 0, true).
		AddItem(footer.View, 2, 0, 1, 2, 0, 0, false)

	tuiApp := &TuiApp{
		Application: *tview.NewApplication(),
		model:       model,

		// grid layout
		grid:   grid,
		header: header,
		menu:   menu,
		pages:  pages,
		footer: footer,

		// pages
		directoriesPage: directoriesPage,
		discoPage:       discoPage,
		landingPage:     landingPage,
		tdPage:          tdPage,
		thingsPage:      thingsPage,
	}
	tuiApp.SetRoot(grid, true).EnableMouse(true)

	return tuiApp
}
