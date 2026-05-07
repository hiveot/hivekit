package wottui

import (
	"github.com/hiveot/hivekit/go/examples/wotmodel"
	"github.com/rivo/tview"
)

// The application footer are that shows the current activity and ?
type AppFooter struct {
	View  *tview.TextView
	model *wotmodel.WotModel
}

// Create a new instance of the application view
func NewAppFooter(model *wotmodel.WotModel) *AppFooter {

	view := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).SetText("Footer")
	view.SetBorder(true)

	panel := &AppFooter{
		View:  view,
		model: model}
	return panel
}
