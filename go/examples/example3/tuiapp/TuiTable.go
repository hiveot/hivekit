package tuiapp

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// A simple table with boilerplate code for adding titles and data rows.
type TuiTable struct {
	tview.Table
	titleColor tcell.Color
}

// Add the column titles to the table
// These use the titleColor and are not selectable
func (tbl *TuiTable) SetTitleRow(row int, titles ...string) {
	for i, title := range titles {
		tbl.SetCell(row, i,
			tview.NewTableCell(title).
				SetTextColor(tbl.titleColor).SetSelectable(false))
	}
}

// Add row data to the table, and return the increased column number
// Row should start at 1, as row 0 is for the titles
func (tbl *TuiTable) SetDataCell(row int, col int, content string) int {
	tbl.SetCell(row, col,
		tview.NewTableCell(content).SetSelectable(true))
	return col + 1
}

// Add a data row to the table
// Row should start at 1, as row 0 is for the titles
func (tbl *TuiTable) SetDataRow(row int, colData ...string) {
	for col, content := range colData {
		tbl.SetCell(row, col,
			tview.NewTableCell(content).SetSelectable(true))
	}

}

func NewTuiTable(titleColor tcell.Color) *TuiTable {
	tbl := &TuiTable{
		titleColor: titleColor,
		Table:      *tview.NewTable(),
	}
	// tbl.SetBorders(true)
	return tbl
}
