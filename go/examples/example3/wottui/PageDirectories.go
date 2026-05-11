package wottui

import (
	"fmt"

	"github.com/araddon/dateparse"
	"github.com/gdamore/tcell/v2"
	"github.com/hiveot/hivekit/go/examples/wotmodel"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/rivo/tview"
)

// Show the loaded directories in the main view
type DirectoriesPage struct {
	tview.Table
	model      *wotmodel.WotModel
	titleColor tcell.Color
	evHandler  func(ev ...string)
}

// add a cell to the table and return the increased column number
func (v *DirectoriesPage) addTitle(title string, row int, col int) int {
	v.SetCell(row, col,
		tview.NewTableCell(title).
			SetTextColor(v.titleColor).SetSelectable(false))
	return col + 1
}
func (v *DirectoriesPage) addData(content string, row int, col int) int {
	v.SetCell(row, col,
		tview.NewTableCell(content).SetSelectable(true))
	return col + 1
}

// Return the directoryID of the selected row, or empty string if not found
func (v *DirectoriesPage) GetDirectoryID(row int) string {
	cell := v.GetCell(row, 0)
	if cell == nil {
		return ""
	}
	return cell.Text
}

// Show the loaded directories in the main view
// this rotates through different tables
func (v *DirectoriesPage) Refresh() {
	v.titleColor = tview.Styles.TertiaryTextColor

	v.SetTitle(" Discovered Directories ")
	v.SetBorders(false)
	v.SetSelectable(true, false)

	tdList := v.model.GetDirectories()
	lines := []string{}
	// start with an empty table
	v.Clear()
	v.titleColor = tview.Styles.TertiaryTextColor
	col := v.addTitle("DirectoryID", 0, 0)
	col = v.addTitle("Title", 0, col)
	col = v.addTitle("Security", 0, col)
	col = v.addTitle("Base URL", 0, col)
	col = v.addTitle("Modified", 0, col)

	row := 0
	for thingID, tdoc := range tdList {
		row++
		sec := utils.DecodeAsString(tdoc.Security, 20)
		modified := dateparse.MustParse(tdoc.Modified).Local()

		col := v.addData(thingID, row, 0)
		col = v.addData(tdoc.Title, row, col)
		col = v.addData(sec, row, col)
		col = v.addData(tdoc.Base, row, col)
		col = v.addData(modified.Format("2006-01-02 15:04"), row, col)
	}
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("%d directories found", len(tdList)))
}

func (footer *DirectoriesPage) SetHandler(h func(ev ...string)) {
	footer.evHandler = h
}

// send event when a thing is selected
func (v *DirectoriesPage) submitEvent(ev string, thingID string) {
	if v.evHandler != nil {
		v.evHandler(ev, thingID)
	}
}

// Return a new page with a table of known thing TDs
func NewDirectoriesPage(model *wotmodel.WotModel) *DirectoriesPage {

	directoriesPage := &DirectoriesPage{
		Table: *tview.NewTable(),
		model: model,
	}
	directoriesPage.SetBorder(true)
	directoriesPage.Refresh()
	directoriesPage.Table.SetSelectedFunc(func(row int, column int) {
		thingID := directoriesPage.GetDirectoryID(row)
		directoriesPage.submitEvent(MenuEvShowDirectory, thingID)
	})

	return directoriesPage
}
