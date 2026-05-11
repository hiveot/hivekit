package wottui

import (
	"github.com/hiveot/hivekit/go/examples/wotmodel"
	"github.com/rivo/tview"
)

type LandingPage struct {
	tview.TextView
	model *wotmodel.WotModel
}

func (v *LandingPage) Refresh() {
	v.SetBorder(true)
	v.SetTextAlign(tview.AlignCenter)
	v.SetText("Welcome to the WoT TUI example application.\n" +
		"\nPress 'd' to start discovery of Things and Directories\n" +
		"\nPress 't' to show a list of discovered Things\n" +
		"\nPress 'q' to quit.")

}

// Create a new instance of the landing page view
func NewLandingPage(model *wotmodel.WotModel) *LandingPage {

	landingPage := &LandingPage{
		TextView: *tview.NewTextView(),
		model:    model,
	}
	landingPage.SetBorder(true)

	return landingPage
}
