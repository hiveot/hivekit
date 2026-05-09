package wottui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/hiveot/hivekit/go/examples/wotmodel"
	"github.com/rivo/tview"
)

// Page with discovery records
type DiscoPage struct {
	tview.Table
	model      *wotmodel.WotModel
	titleColor tcell.Color
}

// add a cell to the table and return the increased column number
func (v *DiscoPage) addTitle(title string, row int, col int) int {
	v.SetCell(row, col,
		tview.NewTableCell(title).
			SetTextColor(v.titleColor).SetSelectable(false))
	return col+1
}
func (v *DiscoPage) addData(content string, row int, col int) int {
	v.SetCell(row, col,
		tview.NewTableCell(content).SetSelectable(true))
	return col+1
}

// Show the discovered records in the main view
func (v *DiscoPage) Refresh() {

	v.SetTitle(" Discovery Records ")
	v.SetBorders(false)
	v.SetSelectable(true, false)

	records := v.model.GetRecords()
	v.titleColor = tview.Styles.TertiaryTextColor

	col := v.addTitle("Type", 0, 0)
	col = v.addTitle("Address", 0, col)
	col = v.addTitle("Port", 0, col)
	col = v.addTitle("Instance", 0, col)
	col = v.addTitle("Schema", 0, col)
	col = v.addTitle("TD URL", 0, col)

	for row, rec := range records {
		tdURL := rec.AsURL()
		col := v.addData(rec.Type, row+1, 0)
		col = v.addData(rec.Addr, row+1, col)
		col = v.addData(fmt.Sprintf("%d", rec.Port), row+1, col)
		col = v.addData(rec.Instance, row+1, col)
		col = v.addData(rec.Schema, row+1, col)
		col = v.addData(tdURL, row+1, col)
	}
}

// Create a new discovery table page
func NewDiscoPage(model *wotmodel.WotModel) *DiscoPage {

	recordsView := &DiscoPage{
		Table: *tview.NewTable(),
		model: model,
	}
	recordsView.SetBorder(true)
	recordsView.SetFixed(1, 1)
	recordsView.Refresh()

	return recordsView
}
