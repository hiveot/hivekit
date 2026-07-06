package tuiapp

import (
	"fmt"

	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules/transport/discovery"
	"github.com/rivo/tview"
)

// The application header are that shows the connect connection and loaded status
type AppHeader struct {
	View *tview.Flex
	text *tview.TextView
}

// Reload the text being shown using updated values
func (header *AppHeader) Refresh(discoRecs []*discovery.DiscoveryResult, tdList []*td.TD) {

	newText := fmt.Sprintf("Discovery records: %d,  Loaded %d TDs",
		len(discoRecs), len(tdList))
	header.text.SetText(newText)
}

// Create a new instance of the application view
func NewAppHeader() *AppHeader {
	view := tview.NewFlex().SetDirection(tview.FlexColumn)
	text := tview.NewTextView().SetTextAlign(tview.AlignLeft).
		SetText("Start 'Discover' to find devices on the network")
	// discoBtn := tview.NewButton("(d) Discover")
	// view := tview.NewTextView().
	// 	SetTextAlign(tview.AlignLeft).SetText("Start 'Discover' to find devices on the network")
	view.SetBorder(true)
	view.AddItem(text, 0, 1, false)
	// view.AddItem(discoBtn, 15, 0, false)
	// view.SetFocusable(false)  // not supported by TextView
	// view.SetBackgroundColor(tcell.ColorDarkGray)

	header := &AppHeader{
		View: view,
		text: text,
		// btn:   discoBtn,
	}
	// discoBtn.SetSelectedFunc(func() {
	// header.submit(MenuEvDiscover)
	// })

	return header
}
