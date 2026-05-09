package wottui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/araddon/dateparse"
	"github.com/gdamore/tcell/v2"
	"github.com/hiveot/hivekit/go/examples/wotmodel"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/rivo/tview"
)

// // Show the loaded things in the main view
// // this rotates through different tables
// func (appMain *AppMain) ShowThings() {

// 	viewNr := appMain.thingViewNr
// 	appMain.thingViewNr++
// 	if appMain.thingViewNr > 2 {
// 		appMain.thingViewNr = 0
// 	}
// 	appMain.View.SetTitle(fmt.Sprintf(" Discovered Things - page %d ", viewNr))

// 	tdList := appMain.model.GetThings()
// 	lines := []string{}

// 	// todo: use selectable columns
// 	switch viewNr {
// 	case 0:
// 		lines = append(lines, "Thing ID                     Title                         Security       Base URL")
// 		lines = append(lines, "---------------------------  ----------------------------  ----------")
// 		for thingID, tdoc := range tdList {
// 			propNames := []string{}
// 			for name := range tdoc.Properties {
// 				propNames = append(propNames, name)
// 			}
// 			lines = append(lines,
// 				fmt.Sprintf("%-28s %-28.28s  %s",
// 					thingID, tdoc.Title, tdoc.Base))

// 		}
// 	case 1:
// 		lines = append(lines, "Thing ID                     Title                         #Props #Events #Actions  Modified (local)")
// 		lines = append(lines, "---------------------------  ----------------------------  ------ ------- --------  ----------------")
// 		for thingID, tdoc := range tdList {
// 			modified := dateparse.MustParse(tdoc.Modified).Local()

// 			lines = append(lines,
// 				fmt.Sprintf("%-28s %-28.28s %6d %7d %8d   %-16s",
// 					thingID, tdoc.Title, len(tdoc.Properties), len(tdoc.Events), len(tdoc.Actions),
// 					modified.Format("2006-01-02 15:04")))

// 		}
// 	case 2:
// 		lines = append(lines, "Thing ID                     Title                         Actions")
// 		lines = append(lines, "---------------------------  ----------------------------  ----------")
// 		for thingID, tdoc := range tdList {
// 			names := []string{}
// 			for name := range tdoc.Actions {
// 				names = append(names, name)
// 			}
// 			lines = append(lines,
// 				fmt.Sprintf("%-28s %-28.28s  %s",
// 					thingID, tdoc.Title, strings.Join(names, ", ")))

// 		}
// 	}
// 	lines = append(lines, "")
// 	lines = append(lines, fmt.Sprintf("%d things found", len(tdList)))
// 	content := strings.Join(lines, "\n")
// 	appMain.View.SetText(content)
// }

type ThingsPage struct {
	tview.Table
	model       *wotmodel.WotModel
	thingViewNr int
	titleColor  tcell.Color
}

// add a cell to the table and return the increased column number
func (v *ThingsPage) addTitle(title string, row int, col int) int {
	v.SetCell(row, col,
		tview.NewTableCell(title).
			SetTextColor(v.titleColor).SetSelectable(false))
	return col + 1
}
func (v *ThingsPage) addData(content string, row int, col int) int {
	v.SetCell(row, col,
		tview.NewTableCell(content).SetSelectable(true))
	return col + 1
}

// Show the loaded things in the main view
// this rotates through different tables
func (v *ThingsPage) Refresh() {
	v.titleColor = tview.Styles.TertiaryTextColor

	viewNr := v.thingViewNr
	v.thingViewNr++
	if v.thingViewNr > 2 {
		v.thingViewNr = 0
	}
	v.SetTitle(fmt.Sprintf(" Discovered Things - page %d ", viewNr+1))
	v.SetBorders(false)
	v.SetSelectable(true, false)

	tdList := v.model.GetThings()
	lines := []string{}
	v.Clear()
	v.titleColor = tview.Styles.TertiaryTextColor
	col := v.addTitle("ThingID", 0, 0)
	col = v.addTitle("Title", 0, col)
	if viewNr == 0 {
		col = v.addTitle("Security", 0, col)
		col = v.addTitle("Base URL", 0, col)
	}
	if viewNr == 1 {
		col = v.addTitle("#Props", 0, col)
		col = v.addTitle("#Events", 0, col)
		col = v.addTitle("#Actions", 0, col)
		col = v.addTitle("Modified", 0, col)
	}
	if viewNr == 2 {
		col = v.addTitle("Actions", 0, col)
	}

	row := 0
	for thingID, tdoc := range tdList {
		row++
		names := []string{}
		for name := range tdoc.Actions {
			names = append(names, name)
		}
		sec := utils.DecodeAsString(tdoc.Security, 20)
		modified := dateparse.MustParse(tdoc.Modified).Local()

		col := v.addData(thingID, row, 0)
		col = v.addData(tdoc.Title, row, col)
		if viewNr == 0 {
			col = v.addData(sec, row, col)
			col = v.addData(tdoc.Base, row, col)
		}
		if viewNr == 1 {
			col = v.addData(strconv.Itoa(len(tdoc.Properties)), row, col)
			col = v.addData(strconv.Itoa(len(tdoc.Events)), row, col)
			col = v.addData(strconv.Itoa(len(tdoc.Actions)), row, col)
			col = v.addData(modified.Format("2006-01-02 15:04"), row, col)
		}
		if viewNr == 2 {
			col = v.addData(strings.Join(names, ", "), row, col)
		}
	}
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("%d things found", len(tdList)))
}

// Return a new page with a table of known thing TDs
func NewThingsPage(model *wotmodel.WotModel) *ThingsPage {

	thingsPage := &ThingsPage{
		Table:       *tview.NewTable(),
		model:       model,
		thingViewNr: 0,
	}
	thingsPage.SetBorder(true)
	thingsPage.Refresh()

	return thingsPage
}
