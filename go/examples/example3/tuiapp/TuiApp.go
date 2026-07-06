package tuiapp

import (
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules/consumer"
	"github.com/hiveot/hivekit/go/modules/directory"
	"github.com/hiveot/hivekit/go/modules/transport/discovery"
	"github.com/rivo/tview"
)

const (
	// PageLanding     = "landing"
	PageThings      = "things"
	PageDirectories = "directories"
	PageDiscovery   = "discovery"
	PageTD          = "td"
)

// menu events
const (
	MenuEvClose           = "close"
	MenuEvDiscover        = "discover"
	MenuEvListTDs         = "listTDs"
	MenuEvNextPage        = "nextPage"
	MenuEvSelectTD        = "selectTD"
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
	*consumer.Consumer // for linking

	co      *consumer.Consumer
	dirCl   directory.IDirectoryClient
	discoCl discovery.IDiscoveryClient

	// discovered directories
	dirTDs []*td.TD
	// dirRecs   []*discovery.DiscoveryResult
	// thingRecs []*discovery.DiscoveryResult

	menu *TreeMenu

	pages *tview.Pages

	directoriesPage *DirectoriesPage
	discoPage       *DiscoPage
	// landingPage     *LandingPage
	tdPage     *TDPage
	thingsPage *ThingsPage

	grid   *tview.Grid
	header *AppHeader
	footer *AppFooter

	mux sync.RWMutex
}

// handle event in the background
func (tui *TuiApp) handleEvent(args ...string) {

	ev := args[0]
	go func() {
		switch ev {

		case MenuEvDiscover:
			tui.StartDiscovery()

		case MenuEvListTDs:
			allThings := tui.dirCl.Cache().GetAllThings(0, 0)
			// if len(tui.allThings) > 0 {
			tui.thingsPage.Refresh(allThings)
			tui.pages.SwitchToPage(PageThings)
			// }

		case MenuEvNextPage:
			allThings := tui.dirCl.Cache().GetAllThings(0, 0)
			if len(allThings) > 0 {
				tui.NextPage()
				tui.SetFocus(tui.pages)
			}

		case MenuEvShowDiscovered:
			tui.ShowDiscovery(nil)

		case MenuEvShowDirectories:
			tui.ShowDirectories()

		case MenuEvSelectTD:
			tui.SelectTD(args[1])

		case MenuEvShowThings:
			tui.ShowThings()

		case MenuEvShowTD:
			if len(args) > 1 {
				thingID := args[1]
				tui.ShowTD(thingID)
			} else {
				tui.ShowThings()
			}

		case MenuEvQuit:
			tui.Stop()
		}
	}()
}

// invoke the requested action
func (tui *TuiApp) invokeActionHandler(thingID, name string, input any) {
	// todo
}

func (tui *TuiApp) NextPage() {
	var name string
	var pageNr int
	var pageNames = []string{PageDiscovery, PageThings}

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

// Select a TD in the menu. This is called by the ThingList to select a thing in the menu
// which in turn updates the TD view.
func (tui *TuiApp) SelectTD(thingID string) {
	// tui.QueueUpdate(func() {
	tui.menu.SelectThing(thingID)
	// })
}

func (tui *TuiApp) ShowDirectories() {
	tui.mux.RLock()
	dirTDs := tui.dirTDs
	tui.mux.RUnlock()

	tui.directoriesPage.Refresh(dirTDs)
	tui.pages.SwitchToPage(PageDirectories)
}

// Show the discovery records
// if recs is empty then start a discovery
func (tui *TuiApp) ShowDiscovery(recs []*discovery.DiscoveryResult) {
	tui.pages.SwitchToPage(PageDiscovery)

	if recs == nil {
		recs, _ = tui.discoCl.DiscoverThings("", 0, nil)
	}
	tui.discoPage.Refresh(recs)
}

// Show the TD page with the thingID details
func (tui *TuiApp) ShowTD(thingID string) {
	tui.pages.SwitchToPage(PageTD)
	tdoc := tui.dirCl.Cache().GetThing(thingID)
	props, _ := tui.co.ReadAllProperties(thingID)
	events, _ := tui.co.ReadAllEvents(thingID)
	tui.tdPage.Refresh(thingID, tdoc, props, events)
	tui.Draw()
}

// Show the loaded things in the main view
func (tui *TuiApp) ShowThings() {
	allThings := tui.dirCl.Cache().GetAllThings(0, 0)

	tui.thingsPage.Refresh(allThings)
	tui.pages.SwitchToPage(PageThings)
}

// Start a discovery and refresh the header and main view.
// If a directory is found, set the TDD for the directory service.
func (tui *TuiApp) StartDiscovery() {

	tui.discoPage.SetTitle(" Starting discovery... ")
	tui.pages.SwitchToPage(PageDiscovery)

	go func() {
		// TODO use a callback to update UI as results come in
		dirRecs, dirTDs := tui.discoCl.DiscoverDirectoryTDs(time.Second)
		thingRecs, thingTDs := tui.discoCl.DiscoverThingTDs("", time.Second, nil)

		if len(dirTDs) > 0 {
			tui.dirCl.SetTDD(dirTDs[0])
		}
		for _, tdoc := range thingTDs {
			tui.dirCl.Cache().ImportTD(tdoc)
		}

		tui.mux.Lock()
		tui.dirTDs = dirTDs
		tui.mux.Unlock()
		tui.ShowDiscovery(thingRecs)

		// refresh
		tui.QueueUpdateDraw(func() {

			allThings := tui.dirCl.Cache().GetAllThings(0, 0)

			tui.header.Refresh(dirRecs, allThings)
			tui.footer.Refresh(allThings)
			tui.menu.Refresh(dirTDs, allThings)
			tui.thingsPage.Refresh(allThings)
			tui.directoriesPage.Refresh(dirTDs)
			tui.discoPage.Refresh(thingRecs)
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
	// tui.menu.Refresh(tui.allDirs, tui.allThings)

	// start discovery in the background, this will update the UI when results come in
	go tui.StartDiscovery()

	if err := tui.Application.Run(); err != nil {
		panic(err)
	}
}

// Create a new instance of the tui app
func NewTuiApp(f api.IModuleFactory) *TuiApp {

	co := consumer.NewConsumer(nil, nil)

	header := NewAppHeader()
	header.View.SetBorderColor(tcell.ColorDarkGray)
	pages := tview.NewPages()
	menu := NewTreeMenu()

	directoriesPage := NewDirectoriesPage()
	pages.AddPage(PageDirectories, directoriesPage, true, false)

	discoPage := NewDiscoPage()
	pages.AddPage(PageDiscovery, discoPage, true, false)

	// landingPage := NewLandingPage(model)
	// pages.AddPage(PageLanding, landingPage, true, false)

	thingsPage := NewThingsPage()
	pages.AddPage(PageThings, thingsPage, true, false)

	footer := NewAppFooter()
	footer.View.SetBorderColor(tcell.ColorDarkGray)

	grid := tview.NewGrid().
		SetRows(3, 0, 1).
		SetColumns(30, 0).
		AddItem(header.View, 0, 0, 1, 2, 0, 0, false).
		AddItem(menu, 1, 0, 1, 1, 0, 0, true).
		AddItem(pages, 1, 1, 1, 1, 0, 0, true).
		AddItem(footer.View, 2, 0, 1, 2, 0, 0, false)

	discoCl := api.GetFactoryModule[discovery.IDiscoveryClient](
		f, discovery.DiscoveryClientModuleType)
	dirCl := api.GetFactoryModule[directory.IDirectoryClient](
		f, directory.DirectoryClientModuleType)

	tuiApp := &TuiApp{
		Application: *tview.NewApplication(),
		Consumer:    co,
		co:          co,
		discoCl:     discoCl,
		dirCl:       dirCl,

		// grid layout
		grid:   grid,
		header: header,
		menu:   menu,
		pages:  pages,
		footer: footer,

		// pages
		directoriesPage: directoriesPage,
		discoPage:       discoPage,
		// landingPage:     landingPage,
		// tdPage:     tdPage,
		thingsPage: thingsPage,
	}

	tdPage := NewTDPage(tuiApp.invokeActionHandler)
	pages.AddPage(PageTD, tdPage, true, false)

	tuiApp.SetRoot(grid, true).EnableMouse(true)

	return tuiApp
}
