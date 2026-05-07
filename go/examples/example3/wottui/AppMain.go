package wottui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/araddon/dateparse"
	"github.com/hiveot/hivekit/go/examples/wotmodel"
	"github.com/rivo/tview"
)

// The application main panel that shows the selected page
type AppMain struct {
	View *tview.TextView

	model *wotmodel.WotModel

	thingViewNr int
}

// Show the discovered records in the main view
func (appMain *AppMain) ShowDiscoRecords() {

	appMain.View.SetTitle(" Discovered Directories and Things ")

	records := appMain.model.GetRecords()
	lines := []string{
		"Type       Address    Port   Instance             Schema    TD URL  ",
		"---------- ---------- -----  -------------------  -------   -------  ",
	}
	recLines := []string{}
	for _, r := range records {
		tdURL := r.AsURL()
		line := fmt.Sprintf("%-10s %-10s %-5d  %-20s %-8s  %s ",
			r.Type, r.Addr, r.Port, r.Instance, r.Schema, tdURL)

		recLines = append(recLines, line)
	}
	// Directory first
	slices.Sort(recLines)

	lines = append(lines, recLines...)
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("Discovery complete. %d records found", len(records)))

	content := strings.Join(lines, "\n")
	appMain.View.SetText(content)
	// appMain.View.SetTitle("Discovery of Things and Directories")
}

// Show the loaded things in the main view
// this rotates through different tables
func (appMain *AppMain) ShowThings() {

	viewNr := appMain.thingViewNr
	appMain.thingViewNr++
	if appMain.thingViewNr > 2 {
		appMain.thingViewNr = 0
	}
	appMain.View.SetTitle(fmt.Sprintf(" Discovered Things (%d) ", viewNr))

	tdList := appMain.model.GetThings()
	lines := []string{}

	// todo: use selectable columns
	switch viewNr {
	case 0:
		lines = append(lines, "Thing ID                     Title                         Base URL")
		lines = append(lines, "---------------------------  ----------------------------  ----------")
		for thingID, tdoc := range tdList {
			propNames := []string{}
			for name := range tdoc.Properties {
				propNames = append(propNames, name)
			}
			lines = append(lines,
				fmt.Sprintf("%-28s %-28.28s  %s",
					thingID, tdoc.Title, tdoc.Base))

		}
	case 1:
		lines = append(lines, "Thing ID                     Title                         #Props #Events #Actions  Modified (local)")
		lines = append(lines, "---------------------------  ----------------------------  ------ ------- --------  ----------------")
		for thingID, tdoc := range tdList {
			modified := dateparse.MustParse(tdoc.Modified).Local()

			lines = append(lines,
				fmt.Sprintf("%-28s %-28.28s %6d %7d %8d   %-16s",
					thingID, tdoc.Title, len(tdoc.Properties), len(tdoc.Events), len(tdoc.Actions),
					modified.Format("2006-01-02 15:04")))

		}
	case 2:
		lines = append(lines, "Thing ID                     Title                         Actions")
		lines = append(lines, "---------------------------  ----------------------------  ----------")
		for thingID, tdoc := range tdList {
			names := []string{}
			for name := range tdoc.Actions {
				names = append(names, name)
			}
			lines = append(lines,
				fmt.Sprintf("%-28s %-28.28s  %s",
					thingID, tdoc.Title, strings.Join(names, ", ")))

		}
	}
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("%d things found", len(tdList)))
	content := strings.Join(lines, "\n")
	appMain.View.SetText(content)
}

// Create a new instance of the application view
func NewAppMain(model *wotmodel.WotModel) *AppMain {

	view := tview.NewTextView().
		SetTextAlign(tview.AlignLeft).SetText("")
	view.SetBorder(true)

	panel := &AppMain{
		model: model,
		View:  view,
	}
	return panel
}
