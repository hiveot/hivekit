package tuiapp

import (
	"fmt"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/rivo/tview"
)

// Page for showing details of a TD document
// This consists of a header section with 3 tables for affordances and their value
// Including a button to subscribe and download the TD document as JSON.
type TDPage struct {
	tview.Flex
	header         *TuiTable
	affordances    *TuiTable
	evHandler      func(ev ...string)
	invokeActionCb func(thingID, name string, input any)
}

func (page *TDPage) Refresh(thingID string,
	tdoc *td.TD, props map[string]any, events map[string]*msg.NotificationMessage) {

	if tdoc == nil {
		page.AddItem(tview.NewTextView().SetText("Thing not found: "+thingID), 0, 1, false)
		return
	}

	// Header
	titleColor := tview.Styles.TertiaryTextColor
	page.header.SetCell(0, 0,
		tview.NewTableCell("ThingID:").SetTextColor(titleColor))
	page.header.SetCellSimple(0, 1, thingID)
	page.header.SetCell(1, 0,
		tview.NewTableCell("Title:").SetTextColor(titleColor))
	page.header.SetCellSimple(1, 1, tdoc.Title)
	page.header.SetCell(2, 0,
		tview.NewTableCell("Base URL:").SetTextColor(titleColor))
	page.header.SetCellSimple(2, 1, tdoc.Base)
	page.header.SetCell(3, 0,
		tview.NewTableCell("Modified:").SetTextColor(titleColor))
	page.header.SetCellSimple(3, 1, tdoc.Modified)

	page.header.SetCell(4, 0,
		tview.NewTableCell("Security:").SetTextColor(titleColor))
	secScheme, _ := tdoc.GetSecurityScheme()
	page.header.SetCellSimple(4, 1, fmt.Sprintf("%s (%s)", secScheme.Scheme, secScheme.Description))

	// Properties table
	tbl := page.affordances
	tbl.Clear()
	tbl.SetSelectable(true, true)
	row := 0
	tbl.SetTitleRow(row, fmt.Sprintf("Properties (%d)", len(tdoc.Properties)))
	row++
	tbl.SetTitleRow(row, "Name", "Title", "DataType", "Latest Value")
	row++
	for name, aff := range tdoc.Properties {
		propValue := utils.DecodeAsString(props[name], 50)
		tbl.SetDataRow(row, name, aff.Title, aff.Type, propValue)
		row++
	}

	// Events table
	row++
	tbl.SetTitleRow(row, fmt.Sprintf("Events (%d)", len(tdoc.Events)))
	row++
	tbl.SetTitleRow(row, "Name", "Title", "DataType", "Latest Value", "Updated")
	row++
	for name, aff := range tdoc.Events {
		var evValue string
		notif := events[name]
		_ = notif.Decode(&evValue)
		updated := utils.FormatDateTime(notif.Timestamp, "S")
		tbl.SetDataRow(row, name, aff.Title, aff.Data.Type, evValue, updated)

		row++
	}

	// Actions table
	row++
	tbl.SetTitleRow(row, fmt.Sprintf("Actions (%d)", len(tdoc.Actions)))
	row++
	for name, aff := range tdoc.Actions {
		tbl.SetCell(row, 0,
			tview.NewTableCell(name).SetSelectable(false))
		tbl.SetCell(row, 1,
			tview.NewTableCell(aff.Title).SetSelectable(false))
		if aff.Input != nil {
			tbl.SetCell(row, 2,
				tview.NewTableCell("Input: "+aff.Input.Type).SetSelectable(false))
		} else {
			tbl.SetCell(row, 2,
				tview.NewTableCell("[red]run").SetSelectable(true).SetClickedFunc(func() bool {
					page.invokeActionCb(thingID, name, nil)
					return true
				}))

		}
		if aff.Output != nil {
			tbl.SetCell(row, 3,
				tview.NewTableCell("Output: "+aff.Output.Type).SetSelectable(false))
		}
		row++
	}
	tbl.ScrollToBeginning()
}

func (page *TDPage) SetHandler(h func(ev ...string)) {
	page.evHandler = h
}

// send event when a thing is selected
func (page *TDPage) submitEvent(ev string, thingID string) {
	if page.evHandler != nil {
		page.evHandler(ev, thingID)
	}
}
func NewTDPage(invokeActionCb func(thingID, name string, input any)) *TDPage {
	header := NewTuiTable(tview.Styles.TertiaryTextColor)
	affordances := NewTuiTable(tview.Styles.TertiaryTextColor)
	page := &TDPage{
		Flex:           *tview.NewFlex(),
		header:         header,
		affordances:    affordances,
		invokeActionCb: invokeActionCb,
	}
	page.SetTitle(" TD ")
	page.SetBorder(true)
	page.SetDirection(tview.FlexColumnCSS)

	page.AddItem(header, 6, 1, false)
	page.AddItem(affordances, 0, 1, false)

	return page
}
