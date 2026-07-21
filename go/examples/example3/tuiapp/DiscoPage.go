package tuiapp

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/hiveot/hivekit/go/modules/transport/discovery"
	"github.com/rivo/tview"
)

// Page with discovery records
type DiscoPage struct {
	*TuiTable
	titleColor tcell.Color
}

// Show a table of discovered directory and device records.
// records can be nil for initial display.
func (page *DiscoPage) Refresh(dirRecs, deviceRecs []*discovery.DiscoveryResult) {

	page.SetBorders(false)
	page.SetSelectable(true, false)
	page.Clear()

	if dirRecs == nil && deviceRecs == nil {
		page.SetTitle(" Discovery In Progress... ")
	} else {
		page.SetTitle(" Discovered Records ")
	}

	page.titleColor = tview.Styles.TertiaryTextColor
	row := 0
	page.SetTitleRow(row, "Type", "Address", "Port", "Instance", "Service", "TD URL")
	row++
	for _, rec := range dirRecs {
		tdURL := rec.AsURL()
		portStr := fmt.Sprintf("%d", rec.Port)
		page.SetTextRow(row, rec.Type, rec.Addr, portStr, rec.Instance, rec.Service, tdURL)
		row++
	}
	for _, rec := range deviceRecs {
		tdURL := rec.AsURL()
		portStr := fmt.Sprintf("%d", rec.Port)
		page.SetTextRow(row, rec.Type, rec.Addr, portStr, rec.Instance, rec.Service, tdURL)
		row++
	}
}

// Create a new discovery table page
func NewDiscoPage() *DiscoPage {

	discoPage := &DiscoPage{
		TuiTable: NewTuiTable(),
	}
	discoPage.SetBorder(true)
	discoPage.SetFixed(1, 1)
	discoPage.SetTitle(" Discovery ")

	return discoPage
}
