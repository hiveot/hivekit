package wottui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/hiveot/hivekit/go/examples/wotmodel"
	"github.com/rivo/tview"
)

// The application footer are that shows the current activity and ?
type AppFooter struct {
	View          *tview.Flex
	model         *wotmodel.WotModel
	listThingsBtn *tview.Button
	nextPageBtn   *tview.Button
	handler       func(ev ...string)
}

// Reload the text being shown using updated values
func (footer *AppFooter) Refresh() {
	hasThings := len(footer.model.GetThings()) > 0

	// disable the list things button if there are no things loaded
	footer.listThingsBtn.SetDisabled(!hasThings)
	footer.nextPageBtn.SetDisabled(!hasThings)
}

func (footer *AppFooter) SetHandler(h func(ev ...string)) {
	footer.handler = h
}

func (footer *AppFooter) submit(ev string) {
	if footer.handler != nil {
		footer.handler(ev)
	}
}

// Create a new instance of the application view
func NewAppFooter(model *wotmodel.WotModel) *AppFooter {

	view := tview.NewFlex().SetDirection(tview.FlexColumn)

	discoThingsBtn := tview.NewButton("(d) Discover")
	view.AddItem(discoThingsBtn, 15, 1, false)

	listThingsBtn := tview.NewButton("(l) List Things")
	view.AddItem(listThingsBtn, 17, 1, false)
	listThingsBtn.SetDisabled(len(model.GetThings()) == 0)

	nextPageBtn := tview.NewButton("(tab) Toggle Views")
	view.AddItem(nextPageBtn, 20, 1, false)
	nextPageBtn.SetDisabled(len(model.GetThings()) == 0)

	filler := tview.NewBox()
	filler.SetBackgroundColor(tcell.Color(tview.Styles.ContrastBackgroundColor))
	view.AddItem(filler, 0, 2, false)

	b4 := tview.NewButton("(q) Quit")
	view.AddItem(b4, 10, 1, false)

	view.SetBorder(false)

	footer := &AppFooter{
		View:          view,
		model:         model,
		listThingsBtn: listThingsBtn,
		nextPageBtn:   nextPageBtn,
	}

	discoThingsBtn.SetSelectedFunc(func() {
		footer.submit(MenuEvDiscover)
	})
	listThingsBtn.SetSelectedFunc(func() {
		footer.submit(MenuEvListTDs)
	})
	nextPageBtn.SetSelectedFunc(func() {
		footer.submit(MenuEvNextPage)
	})
	b4.SetSelectedFunc(func() {
		footer.submit(MenuEvQuit)
	})
	return footer
}
