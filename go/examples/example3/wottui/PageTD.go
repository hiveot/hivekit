package wottui

import (
	"github.com/hiveot/hivekit/go/examples/wotmodel"
	"github.com/rivo/tview"
)

// Page for showing details of a TD document
type TDPage struct {
	tview.TextView
	evHandler func(ev ...string)

	model *wotmodel.WotModel
}

func (page *TDPage) Refresh(thingID string) {
	tdList := page.model.GetThings()
	tdoc, found := tdList[thingID]
	if !found {
		page.SetText("Thing not found: " + thingID)
		return
	}

	text := "Thing ID: " + thingID + "\n"
	text += "Title: " + tdoc.Title + "\n"
	text += "Base URL: " + tdoc.Base + "\n"
	// text += "Security: " + tdoc.Security[0].Scheme + "\n"
	text += "Properties:\n"
	for name, aff := range tdoc.Properties {
		text += "  - " + name + ": " + aff.Type + "\n"
	}
	text += "Events:\n"
	for name, aff := range tdoc.Events {
		text += "  - " + name + ": " + aff.Data.Type + "\n"
	}
	text += "Actions:\n"
	for name, aff := range tdoc.Actions {
		text += "  - " + name + ": " + aff.Title + "\n"
	}
	page.SetText(text)
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

	page := &TDPage{
		TextView: *tview.NewTextView(),
		model:    model,
	}
	page.SetBorder(true)

	return page
}
