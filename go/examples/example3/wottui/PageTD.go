package wottui

import (
	"fmt"
	"strings"

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
		tdList := page.model.GetDirectories()
		tdoc, found = tdList[thingID]
	}
	if !found {
		page.SetText("Thing not found: " + thingID)
		return
	}
	secScheme, _ := tdoc.GetSecurityScheme()
	lines := []string{}
	lines = append(lines, fmt.Sprintf("Thing ID: %s", thingID))
	lines = append(lines, fmt.Sprintf("Title: %s", tdoc.Title))
	lines = append(lines, fmt.Sprintf("Base URL: %s", tdoc.Base))
	lines = append(lines, fmt.Sprintf("Modified: %s", tdoc.Modified))
	lines = append(lines, fmt.Sprintf("Security: %s (%s)", secScheme.Scheme, secScheme.Description))

	lines = append(lines, "Properties:")
	for name, aff := range tdoc.Properties {
		lines = append(lines, fmt.Sprintf("  %s: %s (%s)", name, aff.Title, aff.Type))
	}
	lines = append(lines, "Events:")
	for name, aff := range tdoc.Events {
		lines = append(lines, fmt.Sprintf("  %s: %s (%s)", name, aff.Title, aff.Data.Type))
	}
	lines = append(lines, "Actions:")
	for name, aff := range tdoc.Actions {
		lines = append(lines, fmt.Sprintf("  %s: %s ", name, aff.Title))
		if aff.Input != nil {
			lines = append(lines, fmt.Sprintf("     Input: %s (%s)", aff.Input.Title, aff.Input.Type))
		}
		if aff.Output != nil {
			lines = append(lines, fmt.Sprintf("     Output: %s (%s)", aff.Output.Title, aff.Output.Type))
		}
	}
	if len(tdoc.Forms) > 0 {
		lines = append(lines, "Forms:")
		for _, form := range tdoc.Forms {
			subProto, hasSubProto := form.GetSubprotocol()
			_ = hasSubProto
			href := form.GetHRef()
			lines = append(lines,
				fmt.Sprintf("  %v: %s href=%s", form.GetOperations(), subProto, href))
		}

	}
	page.SetText(strings.Join(lines, "\n"))
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
	page.SetTitle(" TD ")
	page.SetBorder(true)

	return page
}
