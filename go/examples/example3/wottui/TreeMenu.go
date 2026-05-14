package wottui

import (
	"github.com/hiveot/hivekit/go/examples/wotco"
	"github.com/rivo/tview"
)

type TreeMenu struct {
	tview.TreeView
	root   *tview.TreeNode // discovered directories and things
	dirs   *tview.TreeNode
	things *tview.TreeNode

	model     *wotco.WotConsumer
	evHandler func(ev ...string)
}

func (m *TreeMenu) HandleSelection(node *tview.TreeNode) {
	ref := node.GetReference()
	if node == m.dirs {
		m.submitEvent(MenuEvShowDirectories, "")
	} else if node == m.things {
		m.submitEvent(MenuEvShowThings, "")
	} else if ref == nil {
		// root node has no reference, show discovered things
		m.submitEvent(MenuEvShowDiscovered, "")
	} else {
		thingID := node.GetReference().(string)

		m.submitEvent(MenuEvShowTD, thingID)

		// ref points to directory or thing, show thing details
	}
}

// Refresh the menu with the latest discovered things and directories
func (m *TreeMenu) Refresh() {

	m.dirs.ClearChildren()
	dirs := m.model.GetDirectories()
	for dirID, td := range dirs {
		treeNode := tview.NewTreeNode(td.Title)
		treeNode.SetReference(dirID)
		m.dirs.AddChild(treeNode)
	}

	things := m.model.GetThings()
	m.things.ClearChildren()
	for thingID, td := range things {
		treeNode := tview.NewTreeNode(td.Title)
		treeNode.SetReference(thingID)
		m.things.AddChild(treeNode)
	}
}

// Select the Thing in the tree view
func (m *TreeMenu) SelectThing(thingID string) {
	for _, node := range m.things.GetChildren() {
		if node.GetReference() == thingID {
			m.SetCurrentNode(node)
			return
		}
	}
}

func (m *TreeMenu) SetHandler(h func(ev ...string)) {
	m.evHandler = h
}

func (m *TreeMenu) submitEvent(ev string, thingID string) {
	if m.evHandler != nil {
		m.evHandler(ev, thingID)
	}
}

func NewTreeMenu(model *wotco.WotConsumer) *TreeMenu {
	menu := &TreeMenu{
		TreeView: *tview.NewTreeView(),
		model:    model,
		root:     tview.NewTreeNode("Discovery"),
	}
	menu.SetBorder(true)
	menu.SetRoot(menu.root)
	menu.root.SetSelectable(true)
	menu.SetCurrentNode(menu.root)
	menu.SetChangedFunc(func(node *tview.TreeNode) {
		menu.HandleSelection(node)
	})
	// menu.SetSelectedFunc(menu.HandleSelection)
	menu.dirs = tview.NewTreeNode("Directories")
	menu.things = tview.NewTreeNode("Things")
	menu.root.AddChild(menu.dirs)
	menu.root.AddChild(menu.things)
	return menu
}
