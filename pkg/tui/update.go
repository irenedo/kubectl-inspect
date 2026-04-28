package tui

import (
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/irenedo/kubectl-inspect/pkg/explain"
)

// Update handles messages and returns the updated model and any commands.
//
// Bubble Tea uses value receivers for Update/View (the Elm Architecture).
// Helper methods that mutate (rebuildVisible, clampCursor, ensureCursorVisible,
// prepareFetchDetail) use pointer receivers and operate on the local copy
// that will be returned. This is intentional.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.MouseMsg:
		return m.handleMouse(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case treeLoadedMsg:
		m.resourceInfo = msg.info
		m.visibleNodes = VisibleNodes(msg.info.Fields)
		m.cursor = 0
		m.treeScroll = 0
		m.detailLoading = true
		if len(m.visibleNodes) > 0 {
			m.lastDetailPath = m.visibleNodes[0].Path
		}
		return m, m.fetchDetailCmd()

	case detailLoadedMsg:
		if msg.path == m.lastDetailPath {
			m.detailLoading = false
			m.detailScroll = 0
			if msg.result.Err != nil {
				m.detailText = "Error: " + msg.result.Err.Error()
			} else {
				m.detailText = msg.result.RawOutput
			}
		}
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, nil
	}

	return m, nil
}

const (
	focusedPaneTree   = "tree"
	focusedPaneDetail = "detail"
)

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "Q", "ctrl+c":
		return m, tea.Quit

	case "left", "h":
		return m.handleLeft()

	case "right", "l":
		return m.handleRight()

	case "up", "k":
		return m.handleUp()

	case "down", "j":
		return m.handleDown()

	case "tab":
		return m.handleTab()

	case "enter":
		return m.handleEnter()

	case "pgdown", "ctrl+d":
		m.detailScroll += m.contentHeight() / 2
		return m, nil

	case "pgup", "ctrl+u":
		m.detailScroll -= m.contentHeight() / 2
		if m.detailScroll < 0 {
			m.detailScroll = 0
		}
		return m, nil
	}

	return m, nil
}

func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	event := tea.MouseEvent(msg)
	switch {
	case event.Action == tea.MouseActionPress && event.Button == tea.MouseButtonLeft:
		return m.handleMouseClick(&event)
	case event.IsWheel():
		m.focusedPane = m.paneForMouse(event.X)
		return m.handleMouseWheel(event.Button)
	}
	return m, nil
}

// paneForMouse returns the pane the mouse is hovering over.
func (m Model) paneForMouse(x int) string {
	innerWidth := m.width - 2
	leftWidth := int(float64(innerWidth-1) * m.leftRatio)
	if x < leftWidth {
		return focusedPaneTree
	}
	return focusedPaneDetail
}

// handleMouseWheel handles scroll wheel events for the focused pane.
func (m Model) handleMouseWheel(button tea.MouseButton) (tea.Model, tea.Cmd) {
	switch button {
	case tea.MouseButtonWheelUp:
		return m.handleWheelUp()
	case tea.MouseButtonWheelDown:
		return m.handleWheelDown()
	}
	return m, nil
}

func (m Model) handleWheelUp() (tea.Model, tea.Cmd) {
	switch m.focusedPane {
	case focusedPaneDetail:
		if m.detailScroll > 0 {
			m.detailScroll--
		}
		return m, nil
	case focusedPaneTree:
		if m.cursor > 0 {
			m.cursor--
			m.copiedPath = ""
			m.ensureCursorVisible()
			cmd := m.prepareFetchDetail()
			return m, cmd
		}
	}
	return m, nil
}

func (m Model) handleWheelDown() (tea.Model, tea.Cmd) {
	switch m.focusedPane {
	case focusedPaneDetail:
		lines := len(strings.Split(m.detailText, "\n"))
		if m.detailScroll < lines-1 {
			m.detailScroll++
		}
		return m, nil
	case focusedPaneTree:
		if m.cursor < len(m.visibleNodes)-1 {
			m.cursor++
			m.copiedPath = ""
			m.ensureCursorVisible()
			cmd := m.prepareFetchDetail()
			return m, cmd
		}
	}
	return m, nil
}

func (m Model) handleMouseClick(msg *tea.MouseEvent) (tea.Model, tea.Cmd) {
	contentStartRow := 2
	contentEndRow := 2 + m.contentHeight()
	if msg.Y < contentStartRow || msg.Y >= contentEndRow {
		return m, nil
	}

	innerWidth := m.width - 2
	leftWidth := int(float64(innerWidth-1) * m.leftRatio)

	if msg.X < leftWidth {
		return m.handleTreeClick(msg.Y, contentStartRow)
	}
	m.focusedPane = focusedPaneDetail
	return m, nil
}

func (m Model) handleTreeClick(y, contentStartRow int) (tea.Model, tea.Cmd) {
	m.focusedPane = focusedPaneTree
	clickRow := y - contentStartRow
	visibleIdx := clickRow + m.treeScroll
	if visibleIdx < 0 || visibleIdx >= len(m.visibleNodes) {
		return m, nil
	}

	m.cursor = visibleIdx
	node := m.visibleNodes[m.cursor]

	if node.IsExpandable() {
		m.toggleClickExpand(node)
		m.ensureCursorVisible()
	}

	m.copiedPath = ""
	cmd := m.prepareFetchDetail()
	return m, cmd
	
}

func (m *Model) toggleClickExpand(node *explain.Node) {
	if node.Expanded {
		CollapseNode(node)
		m.rebuildVisible()
		for i, n := range m.visibleNodes {
			if n == node {
				m.cursor = i
				break
			}
		}
	} else {
		ExpandNode(node)
		m.rebuildVisible()
	}
}

func (m Model) handleUp() (tea.Model, tea.Cmd) {
	if m.focusedPane == focusedPaneDetail {
		// Scroll detail text up
		if m.detailScroll > 0 {
			m.detailScroll--
		}
		return m, nil
	}
	// Navigate tree up
	if m.cursor > 0 {
		m.cursor--
		m.copiedPath = ""
		m.ensureCursorVisible()
		cmd := m.prepareFetchDetail()
		return m, cmd
	}
	return m, nil
}

func (m Model) handleDown() (tea.Model, tea.Cmd) {
	if m.focusedPane == focusedPaneDetail {
		// Scroll detail text down
		lines := len(strings.Split(m.detailText, "\n"))
		if m.detailScroll < lines-1 {
			m.detailScroll++
		}
		return m, nil
	}
	// Navigate tree down
	if m.cursor < len(m.visibleNodes)-1 {
		m.cursor++
		m.copiedPath = ""
		m.ensureCursorVisible()
		cmd := m.prepareFetchDetail()
		return m, cmd
	}
	return m, nil
}

func (m Model) handleLeft() (tea.Model, tea.Cmd) {
	if len(m.visibleNodes) == 0 || m.cursor >= len(m.visibleNodes) {
		return m, nil
	}
	node := m.visibleNodes[m.cursor]
	if !node.IsExpandable() {
		// Collapse parent if currently on a collapsed child
		if node.Parent != nil {
			return m.collapseParent(node)
		}
		return m, nil
	}
	if node.Expanded {
		// Collapse this node
		CollapseNode(node)
		m.rebuildVisible()
		m.clampCursor()
		m.ensureCursorVisible()
	}
	cmd := m.prepareFetchDetail()
	return m, cmd
}

func (m Model) handleRight() (tea.Model, tea.Cmd) {
	if len(m.visibleNodes) == 0 || m.cursor >= len(m.visibleNodes) {
		return m, nil
	}
	node := m.visibleNodes[m.cursor]
	if !node.IsExpandable() {
		return m, nil
	}
	if !node.Expanded {
		// Expand this node
		ExpandNode(node)
		m.rebuildVisible()
		// Cursor still points to this node after expand
		m.ensureCursorVisible()
	}
	cmd := m.prepareFetchDetail()
	return m, cmd
}

func (m Model) handleTab() (tea.Model, tea.Cmd) {
	// Toggle focus between tree and detail panes
	if m.focusedPane == focusedPaneTree {
		m.focusedPane = focusedPaneDetail
	} else {
		m.focusedPane = focusedPaneTree
	}
	return m, nil
}

func (m Model) collapseParent(node *explain.Node) (tea.Model, tea.Cmd) {
	CollapseNode(node.Parent)
	m.rebuildVisible()
	for i, n := range m.visibleNodes {
		if n == node.Parent {
			m.cursor = i
			break
		}
	}
	m.ensureCursorVisible()
	cmd := m.prepareFetchDetail()
	return m, cmd
}

func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	if len(m.visibleNodes) > 0 && m.cursor < len(m.visibleNodes) {
		node := m.visibleNodes[m.cursor]
		text := node.Path
		if node.TypeStr != "" {
			text += ": [" + node.TypeStr + "]"
		}
		if err := clipboard.WriteAll(text); err == nil {
			m.copiedPath = text
		}
	}
	return m, nil
}

// prepareFetchDetail marks the model as loading and returns a fetch command.
func (m *Model) prepareFetchDetail() tea.Cmd {
	m.detailLoading = true
	m.lastDetailPath = m.visibleNodes[m.cursor].Path
	return m.fetchDetailCmd()
}

func (m *Model) rebuildVisible() {
	if m.resourceInfo != nil {
		m.visibleNodes = VisibleNodes(m.resourceInfo.Fields)
	}
}

func (m *Model) clampCursor() {
	if m.cursor >= len(m.visibleNodes) {
		m.cursor = len(m.visibleNodes) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *Model) ensureCursorVisible() {
	visibleHeight := m.contentHeight()
	if visibleHeight <= 0 {
		return
	}
	if m.cursor < m.treeScroll {
		m.treeScroll = m.cursor
	}
	if m.cursor >= m.treeScroll+visibleHeight {
		m.treeScroll = m.cursor - visibleHeight + 1
	}
}

func (m Model) contentHeight() int {
	h := m.height - m.helpHeight() - 3
	if h < 1 {
		h = 1
	}
	return h
}

// helpHeight returns the number of lines the help bar takes at the current content width.
// The help bar width equals the frame width minus 2 (left+right border chars).
func (m Model) helpHeight() int {
	w := m.width - 2
	if w <= 0 {
		return 1
	}
	lines := 134 / w
	if 134%w != 0 {
		lines++
	}
	if lines < 1 {
		lines = 1
	}
	return lines
}
