package wottui

import (
	"fmt"
	"log/slog"

	"github.com/hiveot/hivekit/go/examples/wotmodel"
	clientspkg "github.com/hiveot/hivekit/go/modules/clients/pkg"
	"github.com/rivo/tview"
)

// Page for showing details of a TD document
// This consists of a header section with 3 tables for affordances and their value
// Including a button to subscribe and download the TD document as JSON.
type TDPage struct {
	tview.Flex
	header      *TuiTable
	affordances *TuiTable
	evHandler   func(ev ...string)

	model *wotmodel.WotModel
}

func (page *TDPage) Refresh(thingID string) {
	var consumer *clientspkg.Consumer

	tdList := page.model.GetThings()
	tdoc, found := tdList[thingID]
	if !found {
		tdList := page.model.GetDirectories()
		tdoc, found = tdList[thingID]
	}
	if !found {
		page.AddItem(tview.NewTextView().SetText("Thing not found: "+thingID), 0, 1, false)
		return
	}

	// connect to the thing to read its props
	cl, err := page.model.Connect(thingID)
	_ = cl
	if err == nil {
		consumer = clientspkg.NewConsumer("")
		consumer.SetRequestSink(cl.HandleRequest)
		cl.SetNotificationSink(consumer.HandleNotification)
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
	row := 0
	tbl.SetTitleRow(row, fmt.Sprintf("Properties (%d)", len(tdoc.Properties)))
	row++
	tbl.SetTitleRow(row, "Name", "Title", "DataType", "Latest Value")
	row++
	for name, aff := range tdoc.Properties {
		// latestValue := page.model.GetPropValue(thingID, name)
		var latestValue string
		if consumer != nil {
			err = consumer.ReadPropertyAs(thingID, name, &latestValue)
			if err != nil {
				slog.Error("Refresh error", "err", err.Error())
			}
		}
		tbl.SetDataRow(row, name, aff.Title, aff.Type, latestValue)
		row++
	}

	// Events table
	row++
	tbl.SetTitleRow(row, fmt.Sprintf("Events (%d)", len(tdoc.Events)))
	row++
	tbl.SetTitleRow(row, "Name", "Title", "DataType", "Latest Value", "Updated")
	row++
	for name, aff := range tdoc.Events {
		var latestValue string
		var updated string
		if consumer != nil {
			ev, err := consumer.ReadEvent(thingID, name)
			if err == nil {
				ev.Decode(&latestValue)
				updated = ev.Timestamp
			}
		}
		tbl.SetDataRow(row, name, aff.Title, aff.Data.Type, latestValue, updated)

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
func NewTDPage(model *wotmodel.WotModel) *TDPage {
	header := NewTuiTable(tview.Styles.TertiaryTextColor)
	affordances := NewTuiTable(tview.Styles.TertiaryTextColor)
	page := &TDPage{
		Flex:        *tview.NewFlex(),
		header:      header,
		affordances: affordances,
		model:       model,
	}
	page.SetTitle(" TD ")
	page.SetBorder(true)
	page.SetDirection(tview.FlexColumnCSS)

	page.AddItem(header, 6, 1, false)
	page.AddItem(affordances, 0, 1, false)

	return page
}
