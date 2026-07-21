package tuiapp

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/araddon/dateparse"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/utils"
)

// Show the loaded things in the main view
// this rotates through different tabl
type ThingsPage struct {
	*TuiTable
	// model       *wotco.WotConsumer
	thingViewNr int
	evHandler   func(ev ...string)
}

// Return the thingID of the selected row, or empty string if not found
func (page *ThingsPage) GetThingID(row int) string {
	cell := page.GetCell(row, 0)
	if cell == nil {
		return ""
	}
	return cell.Text
}

// Show the loaded things in the main view
// this rotates through different tables
func (page *ThingsPage) Refresh(tdList []*td.TD) {

	viewNr := page.thingViewNr
	page.thingViewNr++
	if page.thingViewNr > 2 {
		page.thingViewNr = 0
	}
	page.SetTitle(fmt.Sprintf(" Discovered Things - page %d ", viewNr+1))
	page.SetBorders(false)
	page.SetSelectable(true, false)

	lines := []string{}
	// start with an empty table and a title row
	page.Clear()
	titles := []string{"ThingID", "Title"}
	if viewNr == 0 {
		titles = append(titles, "Security", "Base URL")
	}
	if viewNr == 1 {
		titles = append(titles, "#Props", "#Events", "#Actions", "Modified")
	}
	if viewNr == 2 {
		titles = append(titles, "Actions")
	}
	page.SetTitleRow(0, titles...)

	// Add a list of known things
	row := 0
	for _, tdoc := range tdList {
		row++
		names := []string{}
		for name := range tdoc.Actions {
			names = append(names, name)
		}
		sec := utils.DecodeAsString(tdoc.Security, 20)
		modified := dateparse.MustParse(tdoc.Modified).Local()

		colData := []string{tdoc.ID, tdoc.Title}
		if viewNr == 0 {
			colData = append(colData, sec, tdoc.Base)
		}
		if viewNr == 1 {
			colData = append(colData,
				strconv.Itoa(len(tdoc.Properties)),
				strconv.Itoa(len(tdoc.Events)),
				strconv.Itoa(len(tdoc.Actions)),
				modified.Format("2006-01-02 15:04"))
		}
		if viewNr == 2 {
			colData = append(colData, strings.Join(names, ", "))
		}
		page.SetTextRow(row, colData...)
	}
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("%d things found", len(tdList)))
}

func (page *ThingsPage) SetHandler(h func(ev ...string)) {
	page.evHandler = h
}

// send event when a thing is selected
func (page *ThingsPage) submitEvent(ev string, thingID string) {
	if page.evHandler != nil {
		page.evHandler(ev, thingID)
	}
}

// Select a Thing from the list and show its details
func (page *ThingsPage) onRowSelect(row int, column int) {
	thingID := page.GetThingID(row)
	page.submitEvent(MenuEvSelectTD, thingID)
}

// Return a new page with a table of known thing TDs
func NewThingsPage() *ThingsPage {

	thingsPage := &ThingsPage{
		TuiTable:    NewTuiTable(),
		thingViewNr: 0,
	}
	thingsPage.Refresh(nil)
	thingsPage.SetBorder(true)
	thingsPage.Table.SetSelectedFunc(thingsPage.onRowSelect)

	return thingsPage
}
