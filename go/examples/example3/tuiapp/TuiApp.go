package tuiapp

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules/consumer"
	"github.com/hiveot/hivekit/go/modules/directory"
	"github.com/hiveot/hivekit/go/modules/transport/discovery"
	"github.com/hiveot/hivekit/go/modules/vcache"
	"github.com/hiveot/hivekit/go/utils"
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
	vcache  vcache.IValueCacheService

	// discovered directories
	dirTDs []*td.TD
	// dirRecs   []*discovery.DiscoveryResult
	// thingRecs []*discovery.DiscoveryResult

	// View pages
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

// handle ui event
func (tuiApp *TuiApp) handleEvent(args ...string) {

	ev := args[0]
	// don't run in background as it might cause concurrency issues
	// eg, table is cleared while it is filled.
	// go func() {
	switch ev {

	case MenuEvDiscover:
		tuiApp.StartDiscovery()

	case MenuEvListTDs:
		allThings := tuiApp.dirCl.Cache().GetAllThings(0, 100)
		// if len(tui.allThings) > 0 {
		tuiApp.thingsPage.Refresh(allThings)
		tuiApp.QueueSwitchToPage(PageThings)
		// }

	case MenuEvNextPage:
		allThings := tuiApp.dirCl.Cache().GetAllThings(0, 100)
		if len(allThings) > 0 {
			tuiApp.NextPage()
			tuiApp.SetFocus(tuiApp.pages)
		}

	case MenuEvShowDiscovered:
		tuiApp.ShowDiscovery()

	case MenuEvShowDirectories:
		tuiApp.ShowDirectories()

	case MenuEvSelectTD:
		tuiApp.SelectTD(args[1])

	case MenuEvShowThings:
		tuiApp.ShowThings()

	case MenuEvShowTD:
		if len(args) > 1 {
			thingID := args[1]
			tuiApp.ShowTD(thingID)
		} else {
			tuiApp.ShowThings()
		}

	case MenuEvQuit:
		tuiApp.Stop()
	default:
		slog.Warn("Unknown tui event", "ev", ev)
	}
	// }()
}

// invoke the requested action
func (tuiApp *TuiApp) invokeActionHandler(thingID, name string, input any) {
	err := tuiApp.InvokeAction(thingID, name, input, nil)
	if err != nil {
		tuiApp.ShowError(err)
	} else {
		tuiApp.header.ShowStatus(fmt.Sprintf("Action '%s' successful", name))
	}
}

func (tuiApp *TuiApp) NextPage() {
	var name string
	var pageNr int
	var pageNames = []string{PageDiscovery, PageThings}

	// determine the next page to show
	currentPageName, _ := tuiApp.pages.GetFrontPage()
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
	tuiApp.QueueSwitchToPage(pageName)
}

// Handle notifications
func (tuiApp *TuiApp) HandleNotification(notif *msg.NotificationMessage) {
	val := utils.DecodeAsString(notif.Data, 100)
	tuiApp.header.ShowStatus(fmt.Sprintf(
		"Notification %s; new value: %v", notif.Name, val))

	// todo: update view
	// option 1: full redraw of current view
	//   todo: refresh view command
	if notif.AffordanceType == msg.AffordanceTypeProperty {
		// tdoc := tui.dirCl.Cache().GetThing(notif.ThingID)
		// props, err := tui.co.ReadAllProperties(notif.ThingID)
		// tui.tdPage.Refresh(notif.ThingID, tdoc, props, events)
	} else if notif.AffordanceType == msg.AffordanceTypeEvent {
		// tdoc := tui.dirCl.Cache().GetThing(notif.ThingID)
		// events, err := tui.co.ReadAllEvents(notif.ThingID)
		// tui.tdPage.Refresh(notif.ThingID, tdoc, props, events)
	}

	// option 2: redraw of properties or event fields
	//   todo: how to identify and update these fields?

	// option 3: include a vcache in the module chain

	// option 4: include a vcache as part of a consumer
}

// Show to page.
// This is queued to avoid deadlock when invoking from the event handler.
// This can be called from the background or the main thread.
func (tuiApp *TuiApp) QueueSwitchToPage(pageName string) {
	// Calling QueueUpdate from the main applicationthread causes a deadlock,
	// so run in the background.
	go tuiApp.QueueUpdateDraw(func() {
		tuiApp.pages.SwitchToPage(pageName)
	})
}

// Select a TD in the menu. This is called by the ThingList to select a thing in the menu
// which in turn updates the TD view.
func (tuiApp *TuiApp) SelectTD(thingID string) {
	tuiApp.menu.SelectThing(thingID)
}

func (tuiApp *TuiApp) ShowDirectories() {
	tuiApp.mux.RLock()
	dirTDs := tuiApp.dirTDs
	tuiApp.mux.RUnlock()

	tuiApp.directoriesPage.Refresh(dirTDs)
	tuiApp.QueueSwitchToPage(PageDirectories)
}

// Switch to the the discovery page
func (tuiApp *TuiApp) ShowDiscovery() {
	// tuiApp.discoPage.Refresh(dirRecs, deviceRecs)
	tuiApp.QueueSwitchToPage(PageDiscovery)
}

// Show an error message in the header
func (tuiApp *TuiApp) ShowError(err error) {
	// todo: show nested errors
	tuiApp.header.ShowStatus(err.Error())
	tuiApp.header.text.SetTextColor(tcell.ColorRed)
}

// Show the TD page with the thingID details
// This subscribes to properties and events
func (tuiApp *TuiApp) ShowTD(thingID string) {
	tuiApp.QueueSwitchToPage(PageTD)

	tdoc := tuiApp.dirCl.Cache().GetThing(thingID)
	props, err := tuiApp.co.ReadAllProperties(thingID)
	if err != nil {
		tuiApp.tdPage.Refresh(thingID, tdoc, nil, nil)
		tuiApp.ShowError(err)
	} else {
		tuiApp.header.ShowStatus(fmt.Sprintf("Showing TD of '%s'", thingID))
		events, _ := tuiApp.co.ReadAllEvents(thingID)
		tuiApp.tdPage.Refresh(thingID, tdoc, props, events)
		// subscribe to TD properties and events
		tuiApp.Subscribe(thingID, "")
		tuiApp.ObserveProperty(thingID, "")
	}
}

// Show the loaded things in the main view
func (tuiApp *TuiApp) ShowThings() {
	allThings := tuiApp.dirCl.Cache().GetAllThings(0, 100)

	tuiApp.thingsPage.Refresh(allThings)
	tuiApp.QueueSwitchToPage(PageThings)
}

// Start a discovery and refresh the header and main view.
// If a directory is found, set the TDD for the directory service.
func (tuiApp *TuiApp) StartDiscovery() {

	tuiApp.discoPage.SetTitle(" Running discovery... ")
	tuiApp.QueueSwitchToPage(PageDiscovery)

	go func() {
		// TODO use a callback to update UI as results come in
		dirRecs, dirTDs, deviceRecs, deviceTDs :=
			tuiApp.discoCl.DiscoverThingTDs("", time.Second*2, nil)

		if len(dirTDs) > 0 {
			tuiApp.dirCl.SetTDD(dirTDs[0])
		}
		// add all TDs to the directory client cache
		// TBD: should directory client catch discovery notifications?
		for _, tdoc := range dirTDs {
			tuiApp.dirCl.Cache().ImportTD(tdoc)
		}
		for _, tdoc := range deviceTDs {
			tuiApp.dirCl.Cache().ImportTD(tdoc)
		}

		tuiApp.mux.Lock()
		tuiApp.dirTDs = dirTDs
		tuiApp.mux.Unlock()

		allThings := tuiApp.dirCl.Cache().GetAllThings(0, 100)
		tuiApp.QueueUpdateDraw(func() {
			tuiApp.header.Refresh(dirRecs, allThings)
			tuiApp.footer.Refresh(allThings)
			tuiApp.discoPage.Refresh(dirRecs, deviceRecs)
			tuiApp.menu.Refresh(dirTDs, deviceTDs)
			tuiApp.thingsPage.Refresh(allThings)
			tuiApp.directoriesPage.Refresh(dirTDs)

		})
	}()
}

// Start the application
func (tuiApp *TuiApp) Start() error {

	// vcache collects received notifications
	// tuiApp.vcache = tuiApp.GetVCache()

	tuiApp.footer.SetHandler(tuiApp.handleEvent)
	tuiApp.tdPage.SetHandler(tuiApp.handleEvent)
	tuiApp.thingsPage.SetHandler(tuiApp.handleEvent)
	tuiApp.menu.SetHandler(tuiApp.handleEvent)

	// capture global key events
	tuiApp.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'd':
			tuiApp.handleEvent(MenuEvDiscover)
		case 'l':
			tuiApp.handleEvent(MenuEvListTDs)
		case 'q':
			tuiApp.handleEvent(MenuEvQuit)
		}
		switch event.Key() {
		// tab-key switches between menu and pages
		case tcell.KeyTab:
			if tuiApp.menu.HasFocus() {
				tuiApp.SetFocus(tuiApp.pages)
			} else {
				tuiApp.SetFocus(tuiApp.menu)
			}
		}
		return event
	})
	// tui.menu.Refresh(tui.allDirs, tui.allThings)

	// start discovery in the background, this will update the UI when results come in
	go tuiApp.StartDiscovery()

	err := tuiApp.Application.Run()
	return err
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
	tuiApp.tdPage = tdPage
	pages.AddPage(PageTD, tdPage, true, false)

	tuiApp.SetRoot(grid, true).EnableMouse(true)

	return tuiApp
}
