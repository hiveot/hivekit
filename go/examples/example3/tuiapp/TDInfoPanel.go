package tuiapp

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/rivo/tview"
)

// TD Info panel for show selected affordance details
type TDInfoPanel struct {
	*TuiTable
}

func (infoPanel *TDInfoPanel) ShowForms(forms []td.Form) {
	infoPanel.Clear()
	infoPanel.SetTitle(" Thing Level Forms ")

	row := 0
	for i, form := range forms {
		row = infoPanel.ShowForm(row, form, i)
	}
	infoPanel.ScrollToBeginning()
}

// Show a basic description of the selected action affordance
func (infoPanel *TDInfoPanel) ShowActionAff(actionName string, aff *td.ActionAffordance) {
	infoPanel.Clear()
	infoPanel.SetTitle(" Action Affordance ")
	row := 0
	infoPanel.SetTitleCell(row, 0, "Action: ", actionName)
	if aff.Title != "" {
		row++
		infoPanel.SetTitleCell(row, 0, "Title: ", aff.Title)
	}
	if aff.Description != "" {
		row++
		infoPanel.SetTitleCell(row, 0, "Description: ", aff.Description)
	}
	if aff.AtType != nil {
		row++
		infoPanel.SetTitleCell(row, 0, "@type: ", aff.GetAtTypeString())
	}
	row++
	if aff.Input == nil {
		infoPanel.SetTitleCell(row, 0, "Input: ", "none")
	} else {
		inputInfo := aff.Input.Type + " (" + aff.Input.Title + ")"
		infoPanel.SetTitleCell(row, 0, "Input: ", inputInfo)
	}
	row++
	if aff.Output == nil {
		infoPanel.SetTitleCell(row, 0, "Output: ", "none")
	} else {
		outputInfo := aff.Output.Type + " (" + aff.Output.Title + ")"
		infoPanel.SetTitleCell(row, 0, "Output: ", outputInfo)
	}
	infoPanel.ScrollToBeginning()
}

// Show a basic description of the selected event affordance
func (infoPanel *TDInfoPanel) ShowEventAff(eventName string, aff *td.EventAffordance) {
	infoPanel.Clear()
	infoPanel.SetTitle(" Event Affordance ")
	row := 0
	infoPanel.SetTitleCell(row, 0, "Affordance: ", eventName)
	if aff.Title != "" {
		row++
		infoPanel.SetTitleCell(row, 0, "Title: ", aff.Title)
	}
	if aff.Description != "" {
		row++
		infoPanel.SetTitleCell(row, 0, "Description: ", aff.Description)
	}
	if aff.AtType != nil {
		row++
		infoPanel.SetTitleCell(row, 0, "@type: ", aff.GetAtTypeString())
	}
	if aff.Data != nil {
		row++
		infoPanel.SetTitleCell(row, 0, "Data type: ", aff.Data.Type)
		infoPanel.SetTextCell(row, 2, aff.Data.Unit)
	}
	if len(aff.Forms) > 0 {
		for i, form := range aff.Forms {
			row = infoPanel.ShowForm(row, form, i)
		}
	}
	infoPanel.ScrollToBeginning()
}

// Show a basic description of the selected property affordance
func (infoPanel *TDInfoPanel) ShowPropAff(propName string, aff *td.PropertyAffordance) {
	infoPanel.Clear()
	infoPanel.SetTitle(" Property Affordance ")
	row := 0
	infoPanel.SetTitleCell(row, 0, "Affordance: ", propName)
	if aff.Title != "" {
		row++
		infoPanel.SetTitleCell(row, 0, "Title: ", aff.Title)
	}
	if aff.Description != "" {
		row++
		infoPanel.SetTitleCell(row, 0, "Description: ", aff.Description)
	}
	if aff.AtType != nil {
		row++
		infoPanel.SetTitleCell(row, 0, "@type: ", aff.GetAtTypeString())
	}

	row++
	infoPanel.SetTitleCell(row, 0, "Data type: ", aff.Type)
	infoPanel.SetTextCell(row, 2, aff.Unit)
	// show the dataschema

	if len(aff.Forms) > 0 {
		for i, form := range aff.Forms {
			row = infoPanel.ShowForm(row, form, i)
		}
	}
	infoPanel.ScrollToBeginning()
}

// Add a form section to the table at the row
// This returns the last row+1.
func (infoPanel *TDInfoPanel) ShowForm(row int, form td.Form, i int) int {
	label := fmt.Sprintf("Form[%d]", i)
	infoPanel.SetTitleCell(row, 0, label, "")
	row++
	infoPanel.SetTitleCell(row, 0, "  op: ", form.GetOperation())
	row++
	infoPanel.SetTitleCell(row, 0, "  href: ", form.GetHRef())
	row++
	method, _ := form.GetMethodName()
	if method != "" {
		infoPanel.SetTitleCell(row, 0, "  method: ", method)
		row++
	}
	subp, found := form.GetSubprotocol()
	if found {
		infoPanel.SetTitleCell(row, 0, "  subprotocol: ", subp)
		row++
	}
	return row
}

func NewTDInfoPanel() *TDInfoPanel {
	infoPanel := &TDInfoPanel{
		TuiTable: NewTuiTable(),
	}
	// draw a line
	infoPanel.SetBorder(false).SetDrawFunc(
		func(screen tcell.Screen, x, y, width, height int) (int, int, int, int) {
			// Draw the top border
			leftBorder := tcell.RuneLTee
			rightBorder := tcell.RuneRTee
			if infoPanel.HasFocus() {
				rightBorder = tview.BoxDrawingsVerticalDoubleAndLeftSingle
				leftBorder = tview.BoxDrawingsVerticalDoubleAndRightSingle
			}

			screen.SetContent(x-1, y, leftBorder, nil, tcell.StyleDefault)
			for i := x; i < x+width; i++ {
				screen.SetContent(i, y, tcell.RuneHLine, nil, tcell.StyleDefault)
			}
			screen.SetContent(x+width, y, rightBorder, nil, tcell.StyleDefault)
			title := infoPanel.GetTitle()
			if title != "" {
				titleColor := tview.Styles.PrimaryTextColor
				tview.Print(screen, title, x+1, y, width-2, tview.AlignLeft, titleColor)
			}
			// Return the available inner dimensions (x, y, width, height)
			return x, y + 1, width, height - 1
		})

	return infoPanel
}
