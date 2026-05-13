package wottui

import (
	"github.com/araddon/dateparse"
	"github.com/hiveot/hivekit/go/examples/wotmodel"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/rivo/tview"
)

// Show the loaded directories in the main view
type DirectoriesPage struct {
	TuiTable
	model     *wotmodel.WotModel
	evHandler func(ev ...string)
}

// // add a cell to the table and return the increased column number
// func (v *DirectoriesPage) addTitle(title string, row int, col int) int {
// 	v.SetCell(row, col,
// 		tview.NewTableCell(title).
// 			SetTextColor(v.titleColor).SetSelectable(false))
// 	return col + 1
// }
// func (v *DirectoriesPage) addData(content string, row int, col int) int {
// 	v.SetCell(row, col,
// 		tview.NewTableCell(content).SetSelectable(true))
// 	return col + 1
// }

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

	// start with an empty table
	v.Clear()
	v.SetTitleRow(0, "DirectoryID", "Title", "Security", "Base URL", "Modified")
	row := 0
	for thingID, tdoc := range tdList {
		row++
		sec := utils.DecodeAsString(tdoc.Security, 20)
		modified := dateparse.MustParse(tdoc.Modified).Local()
		modString := modified.Format("2006-01-02 15:04")

		v.SetDataRow(row, thingID, tdoc.Title, sec, tdoc.Base, modString)
	}
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
		TuiTable: *NewTuiTable(tview.Styles.TertiaryTextColor),
		model:    model,
	}
	directoriesPage.SetBorder(true)
	directoriesPage.Refresh()
	directoriesPage.Table.SetSelectedFunc(func(row int, column int) {
		thingID := directoriesPage.GetDirectoryID(row)
		directoriesPage.submitEvent(MenuEvShowDirectory, thingID)
	})

	return directoriesPage
}
