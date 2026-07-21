package tuiapp

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// A simple table with boilerplate code for adding titles and data rows.
type TuiTable struct {
	tview.Table
	TitleColor      tcell.Color
	DataColor       tcell.Color
	TextColor       tcell.Color
	SelectableColor tcell.Color
}

// Add a text row to the table
// Row should start at 1, as row 0 is for the titles
func (tbl *TuiTable) SetTextRow(row int, texts ...string) {
	for col, content := range texts {
		cell := tview.NewTableCell(content).
			SetSelectable(false).
			SetTextColor(tbl.TextColor)
		tbl.SetCell(row, col, cell)
	}
}

// Add a selectable data cell to the table, and return the tabel cell
// Row should start at 1, as row 0 is for the titles
func (tbl *TuiTable) SetSelectableCell(row int, col int, content string) *tview.TableCell {
	cell := tview.NewTableCell(content).SetSelectable(true).SetTextColor(tbl.SelectableColor)
	tbl.SetCell(row, col, cell)
	return cell
}

// Set plain text in the table and return the cell
// multiple text strings can be passed
// this returns the next column number
func (tbl *TuiTable) SetTextCell(row int, col int, content ...string) int {
	for _, text := range content {
		cell := tview.NewTableCell(text).SetSelectable(false).SetTextColor(tbl.TextColor)
		tbl.SetCell(row, col, cell)
		col++
	}
	return col + 1
}

// Set a cell table in title color and text in text color
// This returns the text cell
func (tbl *TuiTable) SetTitleCell(row int, col int, title string, text string) *tview.TableCell {
	titleCell := tview.NewTableCell(title).
		SetTextColor(tbl.TitleColor).SetSelectable(false)
	tbl.SetCell(row, col, titleCell)
	textCell := tview.NewTableCell(text).
		SetTextColor(tbl.TextColor).SetSelectable(false)
	tbl.SetCell(row, col+1, textCell)
	return textCell
}

// Add the column titles to the table
// These use the titleColor and are not selectable
func (tbl *TuiTable) SetTitleRow(row int, titles ...string) {
	for i, title := range titles {
		cell := tview.NewTableCell(title).SetSelectable(false).SetTextColor(tbl.TitleColor)
		tbl.SetCell(row, i, cell)
	}
}

func NewTuiTable() *TuiTable {
	tbl := &TuiTable{
		Table:           *tview.NewTable(),
		TitleColor:      tview.Styles.TitleColor,
		DataColor:       tview.Styles.SecondaryTextColor,
		TextColor:       tview.Styles.TertiaryTextColor,
		SelectableColor: tview.Styles.ContrastSecondaryTextColor,
	}
	// tbl.SetBorders(true)
	return tbl
}
