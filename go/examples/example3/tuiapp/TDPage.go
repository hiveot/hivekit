package tuiapp

import (
	"fmt"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/rivo/tview"
)

// Page for showing details of a TD document:
// - header area with thingID, title, base URL, ...
// - list of properties and latest value (if available)
// - list of events and latest value (if available)
// - list of actions
// - bottom info on selected affordance: forms
type TDPage struct {
	tview.Flex
	header         *TuiTable
	affordances    *TuiTable
	infoPanel      *TDInfoPanel
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
	page.header.SetTitleCell(0, 0, "ThingID:", thingID)
	page.header.SetTitleCell(1, 0, "Title:", tdoc.Title)
	page.header.SetTitleCell(2, 0, "Base URL:", tdoc.Base)
	page.header.SetTitleCell(3, 0, "Modified:", tdoc.Modified)

	secScheme, _ := tdoc.GetSecurityScheme()
	page.header.SetTitleCell(4, 0, "Security:", fmt.Sprintf("%s (%s)", secScheme.Scheme, secScheme.Description))

	// Start with a clean Properties table
	tbl := page.affordances
	tbl.Select(-1, 9999) // hide selection
	tbl.Clear()
	tbl.SetSelectable(true, false)
	row := 0
	tbl.SetTitleRow(row, fmt.Sprintf("Properties (%d)", len(tdoc.Properties)))
	row++
	tbl.SetTitleRow(row, "Name", "Title", "DataType", "Latest Value")
	row++
	keys := utils.OrderedMapKeys(tdoc.Properties)
	for _, name := range keys {
		aff := tdoc.Properties[name]
		var propValue = "n/a"
		if props != nil {
			prop := props[name]
			if prop != nil {
				propValue = utils.DecodeAsString(props[name], 50)
			}
		}
		tbl.SetTextRow(row, name, aff.Title, aff.Type, propValue)
		tbl.GetCell(row, 3).SetSelectable(true).SetTextColor(tbl.DataColor)
		tbl.SetSelectableCell(row, 0, name).SetClickedFunc(
			func() bool {
				page.infoPanel.ShowPropAff(name, aff)
				return false
			})
		// tbl.GetCell(row, 3).SetSelectable(true).SetClickedFunc(
		// 	func() bool {
		// 		// todo: send event to refresh property value
		// 		return false
		// 	})

		row++
	}

	// Events table
	row++
	tbl.SetTitleRow(row, fmt.Sprintf("Events (%d)", len(tdoc.Events)))
	row++
	tbl.SetTitleRow(row, "Name", "Title", "DataType", "Latest Value", "Updated")
	row++
	keys = utils.OrderedMapKeys(tdoc.Events)
	for _, name := range keys {
		aff := tdoc.Events[name]
		var evValue string = "n/a"
		var evTimestamp string = "n/a"
		if events != nil {
			notif := events[name]
			if notif != nil {
				_ = notif.Decode(&evValue)
				evTimestamp = utils.FormatDateTime(notif.Timestamp, "S")
			}
		}
		tbl.SetTextRow(row, name, aff.Title, aff.Data.Type, evValue, evTimestamp)
		tbl.GetCell(row, 3).SetSelectable(true).SetTextColor(tbl.DataColor)
		tbl.GetCell(row, 3).SetTextColor(tbl.DataColor)

		tbl.SetSelectableCell(row, 0, name).SetClickedFunc(
			func() bool {
				page.infoPanel.ShowEventAff(name, aff)
				return false
			})

		// // the value is selectable to support forced refresh
		// tbl.GetCell(row, 3).SetSelectable(true).SetClickedFunc(
		// 	func() bool {
		// 		// todo: send event to refresh event value
		// 		return false
		// 	})

		row++
	}

	// Actions table
	row++
	tbl.SetTitleRow(row, fmt.Sprintf("Actions (%d)", len(tdoc.Actions)))
	row++
	keys = utils.OrderedMapKeys(tdoc.Actions)
	for _, name := range keys {
		aff := tdoc.Actions[name]
		tbl.SetSelectableCell(row, 0, name).SetClickedFunc(
			func() bool {
				page.infoPanel.ShowActionAff(name, aff)
				return false
			})

		tbl.SetTextCell(row, 1, aff.Title)
		if aff.Input != nil {
			tbl.SetTitleCell(row, 2, "Input: ", aff.Input.Type)
		} else {
			tbl.SetSelectableCell(row, 2, "[red]run").SetClickedFunc(func() bool {
				page.invokeActionCb(thingID, name, nil)
				return false
			})
		}
		if aff.Output != nil {
			tbl.SetTitleCell(row, 3, "Output: ", aff.Output.Type)
		}
		row++
	}

	// Info panel
	page.infoPanel.ShowForms(tdoc.Forms)

	tbl.ScrollToBeginning()
}

// set the handler for page events
func (page *TDPage) SetHandler(h func(ev ...string)) {
	page.evHandler = h
}

// send event when a thing is selected
func (page *TDPage) submitEvent(ev string, thingID string) {
	if page.evHandler != nil {
		page.evHandler(ev, thingID)
	}
}

// return a new TD Page that shows the content of a TD document.
func NewTDPage(invokeActionCb func(thingID, name string, input any)) *TDPage {
	header := NewTuiTable()
	affordances := NewTuiTable()
	infoPanel := NewTDInfoPanel()

	page := &TDPage{
		Flex:           *tview.NewFlex(),
		header:         header,
		affordances:    affordances,
		invokeActionCb: invokeActionCb,
		infoPanel:      infoPanel,
	}
	page.SetTitle(" TD ")
	page.SetBorder(true)
	page.SetDirection(tview.FlexColumnCSS)

	page.AddItem(header, 6, 1, false)
	page.AddItem(affordances, 0, 1, false)
	page.AddItem(infoPanel, 0, 1, false)

	return page
}
