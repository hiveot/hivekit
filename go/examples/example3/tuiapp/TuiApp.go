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
	"github.com/hiveot/hivekit/go/cells/consumer"
	"github.com/hiveot/hivekit/go/cells/directory"
	directory_client "github.com/hiveot/hivekit/go/cells/directory/client"
	"github.com/hiveot/hivekit/go/cells/transport/discovery"
	"github.com/hiveot/hivekit/go/cells/vcache"
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

// DirInfo is the discovered directory information
type DirectoryInfo struct {
	thingID string
	title   string
	tdd     *td.TD
	// the client/cache with directory content, if any
	dirClient directory.IDirectoryClient
}

// The main application view with panels for header, menu main view and footer
// - header shows the current status
// - menu shows quick actions for discovery and viewing TDs
// - main shows details
// - footer shows last action
type TuiApp struct {
	tview.Application
	*consumer.Consumer // for linking

	co *consumer.Consumer

	// client for discovery
	discoClient discovery.IDiscoveryClient
	// discovered directories by directory ThingID
	discoDirs []DirectoryInfo
	// cache for discovered things
	discoThings map[string]*td.TD
	// cache with TDs of all things, including discovered things and directories
	allThings directory.IDirectoryClient
	// cache for read values
	vcache vcache.IValueCacheService

	// discovered directories
	// dirTDs []*td.TD
	// dirRecs   []*discovery.DiscoveryResult
	// thingRecs []*discovery.DiscoveryResult

	//--- View pages
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
		tuiApp.ShowDiscovery()
		tuiApp.StartDiscovery()

	// case MenuEvListTDs:
	// 	allThings := tuiApp.dirCl.Cache().GetAllThings(0, 100)
	// 	// if len(tui.allThings) > 0 {
	// 	tuiApp.thingsPage.Refresh(allThings)
	// 	tuiApp.QueueSwitchToPage(PageThings)
	// 	// }

	// case MenuEvNextPage:
	// 	allThings := tuiApp.dirCl.Cache().GetAllThings(0, 100)
	// 	if len(allThings) > 0 {
	// 		tuiApp.NextPage()
	// 		tuiApp.SetFocus(tuiApp.pages)
	// 	}

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

	// option 3: include a vcache in the cell chain

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
	dirTDs := make([]*td.TD, 0, len(tuiApp.discoDirs))

	tuiApp.mux.RLock()
	for _, dirInfo := range tuiApp.discoDirs {
		dirTDs = append(dirTDs, dirInfo.tdd)
	}
	tuiApp.mux.RUnlock()

	tuiApp.directoriesPage.Refresh(dirTDs)
	tuiApp.QueueSwitchToPage(PageDirectories)
}

// Switch to the the discovery page
func (tuiApp *TuiApp) ShowDiscovery() {
	// tuiApp.discoPage.Refresh(dirRecs, deviceRecs)
	tuiApp.menu.SelectDiscovery()
	tuiApp.QueueSwitchToPage(PageDiscovery)
}

// Show an error message in the header
func (tuiApp *TuiApp) ShowError(err error) {
	// todo: show nested errors
	tuiApp.header.ShowStatus(err.Error())
	tuiApp.header.text.SetTextColor(tcell.ColorRed)
}

// Show the TD page with the thingID details
// This reads and subscribes to properties and events
func (tuiApp *TuiApp) ShowTD(thingID string) {
	tuiApp.QueueSwitchToPage(PageTD)

	// locate the TD from discovered things and directories
	tdoc := tuiApp.allThings.Cache().GetThing(thingID)
	if tdoc == nil {
		for _, dirInfo := range tuiApp.discoDirs {
			if dirInfo.thingID == thingID {
				tdoc = dirInfo.tdd
				break
			}
		}
	}
	if tdoc == nil {
		err := fmt.Errorf("TD document not found for Thing '%s'", thingID)
		tuiApp.ShowError(err)
		return
	}
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
	// convert map to list
	allTDs := tuiApp.allThings.Cache().GetAllThings(0, 1000)

	tuiApp.thingsPage.Refresh(allTDs)
	tuiApp.QueueSwitchToPage(PageThings)
}

// Restart a discovery and update the cached TDs.
// If a directory is found, set the TDD for the directory service.
func (tuiApp *TuiApp) StartDiscovery() {

	tuiApp.discoPage.SetTitle(" Running discovery... ")

	go func() {
		dirTDs := make([]*td.TD, 0)

		// TODO use a callback to update UI as results come in
		dirRecs, dirTDs, deviceRecs, deviceTDs :=
			tuiApp.discoClient.DiscoverThingTDs("", time.Second*2, nil)

		// add all local device TDs to the Thing cache
		for _, tdoc := range deviceTDs {
			if tdoc != nil {
				tuiApp.allThings.Cache().ImportTD(tdoc)
				tuiApp.discoThings[tdoc.ID] = tdoc
			}
		}
		// update discovered directories and load their TD
		for _, dirTD := range dirTDs {
			dirInfo := DirectoryInfo{
				thingID:   dirTD.ID,
				title:     dirTD.Title,
				tdd:       dirTD,
				dirClient: directory_client.NewDirectoryClient(dirTD, tuiApp),
			}
			tuiApp.allThings.Cache().ImportTD(dirTD)
			tuiApp.discoDirs = append(tuiApp.discoDirs, dirInfo)
			// try to load the directory content
			dirThings, _ := dirInfo.dirClient.RetrieveAllThings(0, 100)
			for _, tdoc := range dirThings {
				tuiApp.allThings.Cache().ImportTD(tdoc)
			}

		}

		// tuiApp.mux.Lock()
		// tuiApp.dirTDs = dirTDs
		// tuiApp.mux.Unlock()

		allTDs := tuiApp.allThings.Cache().GetAllThings(0, 1000)

		tuiApp.QueueUpdateDraw(func() {
			tuiApp.header.Refresh(dirRecs, allTDs)
			tuiApp.footer.Refresh(allTDs)
			tuiApp.discoPage.Refresh(dirRecs, deviceRecs)
			tuiApp.menu.Refresh(tuiApp.discoDirs, tuiApp.discoThings)
			tuiApp.thingsPage.Refresh(allTDs)
			tuiApp.directoriesPage.Refresh(dirTDs)

		})
	}()
}

// Run the application autonomous processes
func (tuiApp *TuiApp) Start() {

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
	tuiApp.ShowDiscovery()

	tuiApp.Application.Run()
}

// Return a new instance of the tui app.
// Call Start to run.
func NewTuiApp(f api.ICellFactory) *TuiApp {

	// adjust color scheme
	tview.Styles.TitleColor = tcell.ColorGreen
	tview.Styles.TertiaryTextColor = tcell.ColorWhite

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

	discoClient := api.GetFactoryCell[discovery.IDiscoveryClient](
		f, discovery.DiscoveryClientCellType)
	// allThings will hold *all* discovered and loaded TDs
	dirCl := api.GetFactoryCell[directory.IDirectoryClient](
		f, directory.DirectoryClientCellType)
	_ = dirCl // this might not be needed?

	tuiApp := &TuiApp{
		Application: *tview.NewApplication(),
		Consumer:    co,
		co:          co,
		discoClient: discoClient,
		// cache of discovered directories
		discoDirs: make([]DirectoryInfo, 0),
		// cache of locally discovered things
		discoThings: make(map[string]*td.TD),
		// cache of all discovered and directory things
		allThings: dirCl,

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
		// tdPage:     tdPage,  // added below
		thingsPage: thingsPage,
	}

	tdPage := NewTDPage(tuiApp.invokeActionHandler)
	tuiApp.tdPage = tdPage
	pages.AddPage(PageTD, tdPage, true, false)

	tuiApp.SetRoot(grid, true).EnableMouse(true)

	return tuiApp
}
