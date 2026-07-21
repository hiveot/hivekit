package tuiapp

import (
	"github.com/araddon/dateparse"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/rivo/tview"
)

// Show the loaded directories in the main view
type DirectoriesPage struct {
	*TuiTable
	evHandler func(ev ...string)
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
func (v *DirectoriesPage) Refresh(dirList []*td.TD) {
	v.TitleColor = tview.Styles.TertiaryTextColor

	v.SetTitle(" Discovered Directories ")
	v.SetBorders(false)
	v.SetSelectable(true, false)

	// start with an empty table
	v.Clear()
	v.SetTitleRow(0, "DirectoryID", "Title", "Security", "Base URL", "Modified")
	row := 0
	for _, tdoc := range dirList {
		row++
		sec := utils.DecodeAsString(tdoc.Security, 20)
		modified := dateparse.MustParse(tdoc.Modified).Local()
		modString := modified.Format("2006-01-02 15:04")

		v.SetTextRow(row, tdoc.ID, tdoc.Title, sec, tdoc.Base, modString)
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

// Return a new page with a table of known directory TDs
func NewDirectoriesPage() *DirectoriesPage {

	directoriesPage := &DirectoriesPage{
		TuiTable: NewTuiTable(),
	}
	directoriesPage.SetBorder(true)
	directoriesPage.Table.SetSelectedFunc(func(row int, column int) {
		thingID := directoriesPage.GetDirectoryID(row)
		directoriesPage.submitEvent(MenuEvShowDirectory, thingID)
	})

	return directoriesPage
}
