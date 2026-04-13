package tui

import (
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

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "Q", "ctrl+c":
		return m, tea.Quit

	case "up", "k":
		return m.handleCursorUp()

	case "down", "j":
		return m.handleCursorDown()

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

func (m Model) handleCursorUp() (tea.Model, tea.Cmd) {
	if m.cursor > 0 {
		m.cursor--
		m.copiedPath = ""
		m.ensureCursorVisible()
		cmd := m.prepareFetchDetail()
		return m, cmd
	}
	return m, nil
}

func (m Model) handleCursorDown() (tea.Model, tea.Cmd) {
	if m.cursor < len(m.visibleNodes)-1 {
		m.cursor++
		m.copiedPath = ""
		m.ensureCursorVisible()
		cmd := m.prepareFetchDetail()
		return m, cmd
	}
	return m, nil
}

func (m Model) handleTab() (tea.Model, tea.Cmd) {
	if len(m.visibleNodes) == 0 || m.cursor >= len(m.visibleNodes) {
		return m, nil
	}
	node := m.visibleNodes[m.cursor]
	if !node.IsExpandable() {
		if node.Parent != nil {
			return m.collapseParent(node)
		}
		return m, nil
	}
	if node.Expanded {
		CollapseNode(node)
		m.rebuildVisible()
		m.clampCursor()
		m.ensureCursorVisible()
	} else {
		ExpandNode(node)
		m.rebuildVisible()
		if len(node.Children) > 0 && m.cursor+1 < len(m.visibleNodes) {
			m.cursor++
			m.ensureCursorVisible()
		}
	}
	cmd := m.prepareFetchDetail()
	return m, cmd
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
	h := m.height - 4 // borders(2) + title bar(1) + help bar(1)
	if h < 1 {
		h = 1
	}
	return h
}
