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
		refObj := node.GetReference()
		thingID := refObj.(string)

		m.submitEvent(MenuEvShowTD, thingID)

		// ref points to directory or thing, show thing details
	}
}

// Refresh the menu with the latest discovered things and directories
func (m *TreeMenu) Refresh(dirList []DirectoryInfo, discoveredThings map[string]*td.TD) {
	m.thingNodes.ClearChildren()
	for _, tdoc := range discoveredThings {
		label := tdoc.Title
		if label == "" {
			label = "(" + tdoc.ID + ")"
		}
		treeNode := tview.NewTreeNode(label)
		treeNode.SetReference(tdoc.ID)
		m.thingNodes.AddChild(treeNode)
	}

	m.dirNodes.ClearChildren()
	for _, dirInfo := range dirList {
		label := dirInfo.title
		if label == "" {
			label = "(" + dirInfo.thingID + ")"
		}
		treeNode := tview.NewTreeNode(label)
		treeNode.SetReference(dirInfo.thingID)
		m.dirNodes.AddChild(treeNode)

		// nest Things contained in the directory, if accessible
		nodeThings := dirInfo.dirClient.Cache().GetAllThings(0, 100)
		for _, nodeThing := range nodeThings {
			thingNode := tview.NewTreeNode(nodeThing.Title)
			thingNode.SetReference(nodeThing.ID)
			treeNode.AddChild(thingNode)
		}
	}

}

// Select the Discovery entry in the tree view
func (m *TreeMenu) SelectDiscovery() {
	m.SetCurrentNode(m.root)
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

// Show a tree menu with discovered directories and things
func NewTreeMenu() *TreeMenu {
	menu := &TreeMenu{
		TreeView: *tview.NewTreeView(),
		root:     tview.NewTreeNode("Discovery"),
	}
	menu.SetBorder(true)
	menu.SetRoot(menu.root)
	menu.root.SetSelectable(true)
	menu.SetCurrentNode(menu.root)
	menu.SetSelectedFunc(func(node *tview.TreeNode) {
		menu.HandleSelection(node)
	})
	// menu.SetSelectedFunc(menu.HandleSelection)
	menu.thingNodes = tview.NewTreeNode("Things")
	menu.dirNodes = tview.NewTreeNode("Directories")
	menu.root.AddChild(menu.thingNodes)
	menu.root.AddChild(menu.dirNodes)
	return menu
}
