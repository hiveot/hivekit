package tuiapp

import (
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/rivo/tview"
)

type TreeMenu struct {
	tview.TreeView
	root       *tview.TreeNode // discovered directories and things
	dirNodes   *tview.TreeNode
	thingNodes *tview.TreeNode

	evHandler func(ev ...string)
}

func (m *TreeMenu) HandleSelection(node *tview.TreeNode) {
	ref := node.GetReference()
	if node == m.dirNodes {
		m.submitEvent(MenuEvShowDirectories, "")
	} else if node == m.thingNodes {
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
func (m *TreeMenu) Refresh(allDirs []*td.TD, allThings []*td.TD) {

	m.dirNodes.ClearChildren()
	for dirID, td := range allDirs {
		treeNode := tview.NewTreeNode(td.Title)
		treeNode.SetReference(dirID)
		m.dirNodes.AddChild(treeNode)
	}

	m.thingNodes.ClearChildren()
	for thingID, td := range allThings {
		treeNode := tview.NewTreeNode(td.Title)
		treeNode.SetReference(thingID)
		m.thingNodes.AddChild(treeNode)
	}
}

// Select the Thing in the tree view
func (m *TreeMenu) SelectThing(thingID string) {
	for _, node := range m.thingNodes.GetChildren() {
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

func NewTreeMenu() *TreeMenu {
	menu := &TreeMenu{
		TreeView: *tview.NewTreeView(),
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
	menu.dirNodes = tview.NewTreeNode("Directories")
	menu.thingNodes = tview.NewTreeNode("Things")
	menu.root.AddChild(menu.dirNodes)
	menu.root.AddChild(menu.thingNodes)
	return menu
}
