package wottui

import (
	"fmt"

	"github.com/hiveot/hivekit/go/examples/wotmodel"
	"github.com/rivo/tview"
)

// The application header are that shows the connect connection and loaded status
type AppHeader struct {
	View    *tview.Flex
	text    *tview.TextView
	btn     *tview.Button
	model   *wotmodel.WotModel
	handler func(ev string)
}

// Reload the text being shown using updated values
func (header *AppHeader) Refresh() {

	newText := fmt.Sprintf("Discovery records: %d,  Loaded %d TDs",
		len(header.model.GetRecords()), len(header.model.GetThings()))
	header.text.SetText(newText)
}

func (header *AppHeader) SetHandler(h func(ev string)) {
	header.handler = h
}

func (header *AppHeader) submit(ev string) {
	if header.handler != nil {
		header.handler(ev)
	}
}

// Create a new instance of the application view
func NewAppHeader(model *wotmodel.WotModel) *AppHeader {
	view := tview.NewFlex().SetDirection(tview.FlexColumn)
	text := tview.NewTextView().SetTextAlign(tview.AlignLeft).
		SetText("Start 'Discover' to find devices on the network")
	discoBtn := tview.NewButton("(d) Discover")
	// view := tview.NewTextView().
	// 	SetTextAlign(tview.AlignLeft).SetText("Start 'Discover' to find devices on the network")
	view.SetBorder(true)
	view.AddItem(text, 0, 1, false)
	view.AddItem(discoBtn, 15, 0, false)
	// view.SetFocusable(false)  // not supported by TextView
	// view.SetBackgroundColor(tcell.ColorDarkGray)

	header := &AppHeader{
		model: model,
		View:  view,
		text:  text,
		btn:   discoBtn,
	}
	discoBtn.SetSelectedFunc(func() {
		header.submit(MenuEvDiscover)
	})

	return header
}
