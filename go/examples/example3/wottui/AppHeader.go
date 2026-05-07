package wottui

import (
	"fmt"

	"github.com/hiveot/hivekit/go/examples/wotmodel"
	"github.com/rivo/tview"
)

// The application header are that shows the connect connection and loaded status
type AppHeader struct {
	View  *tview.TextView
	model *wotmodel.WotModel
}

// Reload the text being shown using updated values
func (header *AppHeader) Refresh() {

	newText := fmt.Sprintf("Discovery records: %d,  Loaded %d TDs",
		len(header.model.GetRecords()), len(header.model.GetThings()))
	header.View.SetText(newText)
}

// Create a new instance of the application view
func NewAppHeader(model *wotmodel.WotModel) *AppHeader {

	view := tview.NewTextView().
		SetTextAlign(tview.AlignLeft).SetText("Start 'Discover' to find devices on the network")
	view.SetBorder(true)
	// view.SetFocusable(false)  // not supported by TextView
	// view.SetBackgroundColor(tcell.ColorDarkGray)

	panel := &AppHeader{
		model: model,
		View:  view,
	}
	return panel
}
