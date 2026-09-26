package tuiapp

import (
	"fmt"
	"strconv"

	"github.com/araddon/dateparse"
	"github.com/hiveot/hivekit/go/api/td"
)

// Show the loaded things in the main view
type ThingsPage struct {
	*TuiTable
	// model       *wotco.WotConsumer
	evHandler func(ev ...string)
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

	page.SetTitle(fmt.Sprintf(" %d Discovered Things ", len(tdList)))
	page.SetBorders(false)
	page.SetSelectable(true, false)

	lines := []string{}
	// start with an empty table and a title row
	page.Clear()
	titles := []string{"ThingID", "Title", "#Props", "#Events", "#Actions", "Modified", "Base"}
	page.SetTitleRow(0, titles...)

	// Add a list of known things
	row := 0
	for _, tdoc := range tdList {
		row++
		names := []string{}
		for name := range tdoc.Actions {
			names = append(names, name)
		}
		// sec := utils.DecodeAsString(tdoc.Security, 20)
		modifiedTime, err := dateparse.ParseAny(tdoc.Modified)
		modifiedStr := ""
		if err == nil {
			modifiedTime = modifiedTime.Local()
			modifiedStr = modifiedTime.Format("2006-01-02 15:04")
		}

		colData := []string{tdoc.ID, tdoc.Title}
		colData = append(colData,
			strconv.Itoa(len(tdoc.Properties)),
			strconv.Itoa(len(tdoc.Events)),
			strconv.Itoa(len(tdoc.Actions)),
			modifiedStr,
			tdoc.Base)
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
		TuiTable: NewTuiTable(),
	}
	thingsPage.Refresh(nil)
	thingsPage.SetBorder(true)
	thingsPage.Table.SetSelectedFunc(thingsPage.onRowSelect)

	return thingsPage
}
