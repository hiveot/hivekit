package wottui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/araddon/dateparse"
	"github.com/gdamore/tcell/v2"
	"github.com/hiveot/hivekit/go/examples/wotmodel"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/rivo/tview"
)

// Show the loaded things in the main view
// this rotates through different tabl
type ThingsPage struct {
	tview.Table
	model       *wotmodel.WotModel
	thingViewNr int
	titleColor  tcell.Color
	evHandler   func(ev ...string)
}

// add a cell to the table and return the increased column number
func (v *ThingsPage) addTitle(title string, row int, col int) int {
	v.SetCell(row, col,
		tview.NewTableCell(title).
			SetTextColor(v.titleColor).SetSelectable(false))
	return col + 1
}
func (v *ThingsPage) addData(content string, row int, col int) int {
	v.SetCell(row, col,
		tview.NewTableCell(content).SetSelectable(true))
	return col + 1
}

// Return the thingID of the selected row, or empty string if not found
func (v *ThingsPage) GetThingID(row int) string {
	cell := v.GetCell(row, 0)
	if cell == nil {
		return ""
	}
	return cell.Text
}

// Show the loaded things in the main view
// this rotates through different tables
func (v *ThingsPage) Refresh() {
	v.titleColor = tview.Styles.TertiaryTextColor

	viewNr := v.thingViewNr
	v.thingViewNr++
	if v.thingViewNr > 2 {
		v.thingViewNr = 0
	}
	v.SetTitle(fmt.Sprintf(" Discovered Things - page %d ", viewNr+1))
	v.SetBorders(false)
	v.SetSelectable(true, false)

	tdList := v.model.GetThings()
	lines := []string{}
	// start with an empty table
	v.Clear()
	v.titleColor = tview.Styles.TertiaryTextColor
	col := v.addTitle("ThingID", 0, 0)
	col = v.addTitle("Title", 0, col)
	if viewNr == 0 {
		col = v.addTitle("Security", 0, col)
		col = v.addTitle("Base URL", 0, col)
	}
	if viewNr == 1 {
		col = v.addTitle("#Props", 0, col)
		col = v.addTitle("#Events", 0, col)
		col = v.addTitle("#Actions", 0, col)
		col = v.addTitle("Modified", 0, col)
	}
	if viewNr == 2 {
		col = v.addTitle("Actions", 0, col)
	}

	row := 0
	for thingID, tdoc := range tdList {
		row++
		names := []string{}
		for name := range tdoc.Actions {
			names = append(names, name)
		}
		sec := utils.DecodeAsString(tdoc.Security, 20)
		modified := dateparse.MustParse(tdoc.Modified).Local()

		col := v.addData(thingID, row, 0)
		col = v.addData(tdoc.Title, row, col)
		if viewNr == 0 {
			col = v.addData(sec, row, col)
			col = v.addData(tdoc.Base, row, col)
		}
		if viewNr == 1 {
			col = v.addData(strconv.Itoa(len(tdoc.Properties)), row, col)
			col = v.addData(strconv.Itoa(len(tdoc.Events)), row, col)
			col = v.addData(strconv.Itoa(len(tdoc.Actions)), row, col)
			col = v.addData(modified.Format("2006-01-02 15:04"), row, col)
		}
		if viewNr == 2 {
			col = v.addData(strings.Join(names, ", "), row, col)
		}
	}
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("%d things found", len(tdList)))
}

func (footer *ThingsPage) SetHandler(h func(ev ...string)) {
	footer.evHandler = h
}

// send event when a thing is selected
func (v *ThingsPage) submitEvent(ev string, thingID string) {
	if v.evHandler != nil {
		v.evHandler(ev, thingID)
	}
}

// Return a new page with a table of known thing TDs
func NewThingsPage(model *wotmodel.WotModel) *ThingsPage {

	thingsPage := &ThingsPage{
		Table:       *tview.NewTable(),
		model:       model,
		thingViewNr: 0,
	}
	thingsPage.Refresh()
	thingsPage.SetBorder(true)
	thingsPage.Table.SetSelectedFunc(func(row int, column int) {
		thingID := thingsPage.GetThingID(row)
		thingsPage.submitEvent(MenuEvShowTD, thingID)
	})

	return thingsPage
}
