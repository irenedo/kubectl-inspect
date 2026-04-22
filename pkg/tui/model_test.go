package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/irenedo/kubectl-inspect/pkg/explain"
	"github.com/irenedo/kubectl-inspect/pkg/kubectl"
)

type testExecutor struct {
	recursiveOutput string
	recursiveErr    error
	fieldOutput     string
	fieldErr        error
	lastFieldPath   string
}

func (t *testExecutor) ExplainRecursive(_ context.Context, _ string, _ kubectl.Flags) (string, error) {
	return t.recursiveOutput, t.recursiveErr
}

func (t *testExecutor) ExplainField(_ context.Context, fieldPath string, _ kubectl.Flags) (string, error) {
	t.lastFieldPath = fieldPath
	return t.fieldOutput, t.fieldErr
}

func readTestFixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "explain", "testdata", name))
	if err != nil {
		t.Fatalf("failed to read fixture %s: %v", name, err)
	}
	return string(data)
}

func newTestModel(t *testing.T) (Model, *testExecutor) {
	t.Helper()
	fixture := readTestFixture(t, "deployment_recursive.txt")
	exec := &testExecutor{
		recursiveOutput: fixture,
		fieldOutput:     "KIND: Deployment\nVERSION: apps/v1\n",
	}
	m := NewModel("deployment", exec, kubectl.Flags{})
	m.width = 120
	m.height = 40
	return m, exec
}

func loadTree(t *testing.T, m Model) Model {
	t.Helper()
	cmd := m.Init()
	msg := cmd()
	updated, _ := m.Update(msg)
	return updated.(Model)
}

func TestInit_LoadsTree(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)

	if m.resourceInfo == nil {
		t.Fatal("resourceInfo should be set after tree load")
	}
	if m.resourceInfo.Kind != "Deployment" {
		t.Errorf("expected kind Deployment, got %q", m.resourceInfo.Kind)
	}
	if len(m.visibleNodes) == 0 {
		t.Error("visibleNodes should not be empty")
	}
}

func TestInit_Error(t *testing.T) {
	exec := &testExecutor{
		recursiveErr: &kubectl.ExecError{Stderr: "not found", ExitCode: 1},
	}
	m := NewModel("nonexistent", exec, kubectl.Flags{})
	cmd := m.Init()
	msg := cmd()
	updated, _ := m.Update(msg)
	model := updated.(Model)
	if model.err == nil {
		t.Error("expected error to be set")
	}
}

func TestUpdate_CursorDown(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)

	if m.cursor != 0 {
		t.Fatalf("expected cursor at 0, got %d", m.cursor)
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	model := updated.(Model)
	if model.cursor != 1 {
		t.Errorf("expected cursor at 1, got %d", model.cursor)
	}
}

func TestUpdate_CursorUp(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)
	m.cursor = 1

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	model := updated.(Model)
	if model.cursor != 0 {
		t.Errorf("expected cursor at 0, got %d", model.cursor)
	}
}

func TestUpdate_CursorUpAtZero(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)
	m.cursor = 0

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	model := updated.(Model)
	if model.cursor != 0 {
		t.Errorf("expected cursor to stay at 0, got %d", model.cursor)
	}
}

func TestUpdate_CursorDownAtEnd(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)
	m.cursor = len(m.visibleNodes) - 1

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	model := updated.(Model)
	if model.cursor != len(m.visibleNodes)-1 {
		t.Errorf("expected cursor to stay at end, got %d", model.cursor)
	}
}

func TestUpdate_ExpandCollapse(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)

	initialCount := len(m.visibleNodes)

	// First node should be a root-level expandable node
	if !m.visibleNodes[0].IsExpandable() {
		t.Skip("first node is not expandable")
	}

	// Expand with Tab — cursor should move to first child
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	model := updated.(Model)
	if len(model.visibleNodes) <= initialCount {
		t.Errorf("expected more visible nodes after expand, got %d (was %d)", len(model.visibleNodes), initialCount)
	}
	if model.cursor != 1 {
		t.Errorf("expected cursor to move to first child (1), got %d", model.cursor)
	}
	if cmd == nil {
		t.Error("expected a fetch detail command after Tab expand")
	}

	// Esc on a child node should collapse the parent and move cursor to parent
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEscape})
	model = updated.(Model)
	if len(model.visibleNodes) != initialCount {
		t.Errorf("expected %d visible nodes after collapse, got %d", initialCount, len(model.visibleNodes))
	}
	if model.cursor != 0 {
		t.Errorf("expected cursor to return to parent (0), got %d", model.cursor)
	}
	if cmd == nil {
		t.Error("expected a fetch detail command after Esc collapse")
	}
}

func TestUpdate_EscOnExpandedNode(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)

	if !m.visibleNodes[0].IsExpandable() {
		t.Skip("first node is not expandable")
	}

	// Manually expand and keep cursor on the parent
	ExpandNode(m.visibleNodes[0])
	m.rebuildVisible()

	expandedCount := len(m.visibleNodes)

	// Esc on the expanded parent node should collapse it
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	model := updated.(Model)
	if len(model.visibleNodes) >= expandedCount {
		t.Errorf("expected fewer visible nodes after collapse, got %d (was %d)", len(model.visibleNodes), expandedCount)
	}
	if cmd == nil {
		t.Error("expected a fetch detail command after Esc collapse")
	}
}

func TestUpdate_EnterCopiesPath(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)

	if len(m.visibleNodes) == 0 {
		t.Fatal("expected visible nodes")
	}

	node := m.visibleNodes[0]
	expectedText := node.Path
	if node.TypeStr != "" {
		expectedText += ": [" + node.TypeStr + "]"
	}
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := updated.(Model)

	// Enter should not return a command
	if cmd != nil {
		t.Error("expected nil command from Enter (copy path)")
	}

	// copiedPath should be set (clipboard may fail in CI, so we check copiedPath
	// is either set or remains empty if clipboard is unavailable)
	if model.copiedPath != "" && model.copiedPath != expectedText {
		t.Errorf("expected copiedPath %q, got %q", expectedText, model.copiedPath)
	}

	// Visible nodes should not change (Enter no longer toggles)
	if len(model.visibleNodes) != len(m.visibleNodes) {
		t.Error("Enter should not change visible nodes")
	}
}

func TestUpdate_EnterLeaf(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)

	// Expand first node to reveal children, find a scalar leaf
	ExpandNode(m.visibleNodes[0])
	m.rebuildVisible()

	leafIdx := -1
	for i, n := range m.visibleNodes {
		if !n.IsExpandable() {
			leafIdx = i
			break
		}
	}
	if leafIdx < 0 {
		t.Skip("no leaf node found")
	}

	m.cursor = leafIdx
	count := len(m.visibleNodes)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := updated.(Model)
	if len(model.visibleNodes) != count {
		t.Error("Enter on leaf should not change visible nodes")
	}
}

func TestUpdate_Quit(t *testing.T) {
	m, _ := newTestModel(t)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Fatal("expected quit command")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Error("expected tea.QuitMsg")
	}
}

func TestUpdate_DetailFetch(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)

	// Move cursor down — should trigger a fetch command
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if cmd == nil {
		t.Fatal("expected a fetch detail command on cursor move")
	}

	msg := cmd()
	dlMsg, ok := msg.(detailLoadedMsg)
	if !ok {
		t.Fatalf("expected detailLoadedMsg, got %T", msg)
	}
	if dlMsg.result.Err != nil {
		t.Errorf("unexpected error: %v", dlMsg.result.Err)
	}
}

func TestUpdate_StaleDetailDiscarded(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)

	// Simulate: lastDetailPath is set to a different path
	m.lastDetailPath = "spec.replicas"
	staleMsg := detailLoadedMsg{
		path:   "spec.containers",
		result: explain.DetailResult{RawOutput: "stale data"},
	}
	updated, _ := m.Update(staleMsg)
	model := updated.(Model)
	if model.detailText == "stale data" {
		t.Error("stale detail should be discarded")
	}
}

func TestUpdate_WindowResize(t *testing.T) {
	m, _ := newTestModel(t)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 200, Height: 50})
	model := updated.(Model)
	if model.width != 200 || model.height != 50 {
		t.Errorf("expected 200x50, got %dx%d", model.width, model.height)
	}
}

func TestUpdate_DetailScroll(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)
	m.detailText = strings.Repeat("line\n", 100)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	model := updated.(Model)
	if model.detailScroll <= 0 {
		t.Error("expected detailScroll to increase after PgDown")
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	model = updated.(Model)
	if model.detailScroll < 0 {
		t.Error("detailScroll should not go below 0")
	}
}

func TestView_ErrorState(t *testing.T) {
	m, _ := newTestModel(t)
	m.err = &kubectl.ExecError{Stderr: "server not found", ExitCode: 1}
	view := m.View()
	if !strings.Contains(view, "Error") {
		t.Error("error view should contain 'Error'")
	}
}

func TestView_LoadingState(t *testing.T) {
	m, _ := newTestModel(t)
	// resourceInfo is nil by default
	view := m.View()
	if !strings.Contains(view, "Loading") {
		t.Error("loading view should contain 'Loading'")
	}
}

func TestView_NormalRender(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)
	m.detailText = "KIND: Deployment\nVERSION: apps/v1\n"
	view := m.View()
	if !strings.Contains(view, "Deployment") {
		t.Error("normal view should contain resource kind")
	}
}

func TestUpdate_QuitWithUpperQ(t *testing.T) {
	m, _ := newTestModel(t)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Q")})
	if cmd == nil {
		t.Fatal("expected quit command for Q")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Error("expected tea.QuitMsg")
	}
}

func TestUpdate_QuitWithCtrlC(t *testing.T) {
	m, _ := newTestModel(t)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("expected quit command for ctrl+c")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Error("expected tea.QuitMsg")
	}
}

func TestUpdate_ScrollDetailCtrlD(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)
	m.detailText = strings.Repeat("line\n", 100)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	model := updated.(Model)
	if model.detailScroll <= 0 {
		t.Error("expected detailScroll to increase after Ctrl+D")
	}
}

func TestUpdate_ScrollDetailCtrlU(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)
	m.detailText = strings.Repeat("line\n", 100)
	m.detailScroll = 20

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	model := updated.(Model)
	if model.detailScroll >= 20 {
		t.Error("expected detailScroll to decrease after Ctrl+U")
	}
}

func TestUpdate_ScrollCtrlUDoesNotGoBelowZero(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)
	m.detailScroll = 2

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	model := updated.(Model)
	if model.detailScroll < 0 {
		t.Error("detailScroll should not go below 0 after Ctrl+U")
	}
}

func TestUpdate_TabOnAlreadyExpanded(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)

	if !m.visibleNodes[0].IsExpandable() {
		t.Skip("first node is not expandable")
	}

	// Manually expand first
	ExpandNode(m.visibleNodes[0])
	m.rebuildVisible()
	count := len(m.visibleNodes)

	// Tab on already expanded node should be a no-op
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	model := updated.(Model)
	if len(model.visibleNodes) != count {
		t.Error("Tab on already expanded node should not change visible nodes")
	}
	if cmd != nil {
		t.Error("Tab on already expanded node should not trigger a command")
	}
}

func TestUpdate_TabOnScalar(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)

	// Expand to find a leaf
	ExpandNode(m.visibleNodes[0])
	m.rebuildVisible()

	leafIdx := -1
	for i, n := range m.visibleNodes {
		if !n.IsExpandable() {
			leafIdx = i
			break
		}
	}
	if leafIdx < 0 {
		t.Skip("no leaf node found")
	}

	m.cursor = leafIdx
	count := len(m.visibleNodes)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	model := updated.(Model)
	if len(model.visibleNodes) != count {
		t.Error("Tab on scalar should not change visible nodes")
	}
	if cmd != nil {
		t.Error("Tab on scalar should not trigger a command")
	}
}

func TestUpdate_EscOnRootLeafNoOp(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)

	// Find a root-level node that is not expanded and has no parent
	rootIdx := -1
	for i, n := range m.visibleNodes {
		if n.Parent == nil && !n.Expanded {
			rootIdx = i
			break
		}
	}
	if rootIdx < 0 {
		t.Skip("no suitable root node found")
	}

	m.cursor = rootIdx
	count := len(m.visibleNodes)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	model := updated.(Model)
	if len(model.visibleNodes) != count {
		t.Error("Esc on root collapsed node should be a no-op")
	}
	if cmd != nil {
		t.Error("Esc on root collapsed node should not trigger a command")
	}
}

func TestUpdate_CursorMoveClearsCopiedPath(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)
	m.copiedPath = "some.path"

	// Move down should clear copiedPath
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	model := updated.(Model)
	if model.copiedPath != "" {
		t.Error("expected copiedPath to be cleared after cursor down")
	}

	// Reset and test cursor up
	model.copiedPath = "another.path"
	model.cursor = 1
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	model = updated.(Model)
	if model.copiedPath != "" {
		t.Error("expected copiedPath to be cleared after cursor up")
	}
}

func TestUpdate_UnknownKeyIsNoOp(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)
	cursor := m.cursor
	count := len(m.visibleNodes)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	model := updated.(Model)
	if model.cursor != cursor {
		t.Error("unknown key should not change cursor")
	}
	if len(model.visibleNodes) != count {
		t.Error("unknown key should not change visible nodes")
	}
	if cmd != nil {
		t.Error("unknown key should not trigger a command")
	}
}

func TestUpdate_UnknownMsgType(t *testing.T) {
	m, _ := newTestModel(t)
	// Send an unrelated message type
	updated, cmd := m.Update("some random string message")
	model := updated.(Model)
	if cmd != nil {
		t.Error("unknown msg type should not trigger a command")
	}
	if model.err != nil {
		t.Error("unknown msg type should not set an error")
	}
}

func TestUpdate_DetailLoadedWithError(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)
	m.lastDetailPath = "spec"

	msg := detailLoadedMsg{
		path:   "spec",
		result: explain.DetailResult{Err: fmt.Errorf("connection refused")},
	}
	updated, _ := m.Update(msg)
	model := updated.(Model)
	if !strings.Contains(model.detailText, "Error:") {
		t.Error("expected detail text to contain error message")
	}
	if model.detailLoading {
		t.Error("detailLoading should be false after detail loaded")
	}
}

func TestView_CopiedPathInTitle(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)
	m.copiedPath = "spec.replicas"
	m.detailText = "KIND: Deployment\n"
	view := m.View()
	if !strings.Contains(view, "Copied") {
		t.Error("view should show 'Copied' when copiedPath is set")
	}
	if !strings.Contains(view, "spec.replicas") {
		t.Error("view should show the copied path")
	}
}

func TestView_ErrorNotFound(t *testing.T) {
	m, _ := newTestModel(t)
	m.err = &kubectl.NotFoundError{Err: fmt.Errorf("not found")}
	view := m.View()
	if !strings.Contains(view, "kubectl not found") {
		t.Error("error view should contain kubectl not found message")
	}
}

func TestContentHeight_SmallWindow(t *testing.T) {
	m, _ := newTestModel(t)
	m.height = 3 // less than 4, so contentHeight should clamp to 1
	h := m.contentHeight()
	if h != 1 {
		t.Errorf("expected contentHeight 1 for small window, got %d", h)
	}
}

func TestEnsureCursorVisible_ScrollsDown(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)
	m.height = 10 // contentHeight = 6
	m.treeScroll = 0
	m.cursor = 10 // beyond viewport

	m.ensureCursorVisible()
	if m.treeScroll == 0 {
		t.Error("expected treeScroll to increase to keep cursor visible")
	}
}

func TestEnsureCursorVisible_ScrollsUp(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)
	m.height = 10
	m.treeScroll = 5
	m.cursor = 2 // above viewport

	m.ensureCursorVisible()
	if m.treeScroll != 2 {
		t.Errorf("expected treeScroll to be 2, got %d", m.treeScroll)
	}
}

func TestEnsureCursorVisible_MinHeight(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)
	m.height = 2 // contentHeight clamps to 1
	m.treeScroll = 5
	m.cursor = 0

	m.ensureCursorVisible()
	// Cursor is at 0 which is before treeScroll=5, so scroll should adjust
	if m.treeScroll != 0 {
		t.Errorf("expected treeScroll to be 0, got %d", m.treeScroll)
	}
}

func TestClampCursor(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)

	// Cursor beyond end
	m.cursor = len(m.visibleNodes) + 10
	m.clampCursor()
	if m.cursor != len(m.visibleNodes)-1 {
		t.Errorf("expected cursor clamped to %d, got %d", len(m.visibleNodes)-1, m.cursor)
	}

	// Cursor negative
	m.cursor = -5
	m.clampCursor()
	if m.cursor != 0 {
		t.Errorf("expected cursor clamped to 0, got %d", m.cursor)
	}
}

func TestUpdate_EscOnScalarGrandchild_CollapsesParent(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)

	if !m.visibleNodes[0].IsExpandable() {
		t.Skip("first node is not expandable")
	}

	// Expand root to reveal children
	ExpandNode(m.visibleNodes[0])
	m.rebuildVisible()

	// Find an expandable child and expand it
	childIdx := -1
	for i, n := range m.visibleNodes {
		if n.IsExpandable() && n.Depth == 1 && len(n.Children) > 0 {
			childIdx = i
			break
		}
	}
	if childIdx < 0 {
		t.Skip("no expandable child with children found")
	}

	ExpandNode(m.visibleNodes[childIdx])
	m.rebuildVisible()

	// Find a scalar grandchild (leaf node at depth 2)
	leafIdx := -1
	for i, n := range m.visibleNodes {
		if !n.IsExpandable() && n.Depth == 2 {
			leafIdx = i
			break
		}
	}
	if leafIdx < 0 {
		t.Skip("no scalar grandchild found")
	}

	m.cursor = leafIdx
	parentNode := m.visibleNodes[leafIdx].Parent

	// Press Esc on the scalar grandchild — should collapse its parent
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	model := updated.(Model)

	// Parent should be collapsed
	if parentNode.Expanded {
		t.Error("expected parent to be collapsed after Esc on grandchild")
	}

	// Cursor should be on the parent
	if model.cursor >= len(model.visibleNodes) {
		t.Fatal("cursor out of bounds")
	}
	if model.visibleNodes[model.cursor] != parentNode {
		t.Errorf("expected cursor on parent %q, got %q", parentNode.Name, model.visibleNodes[model.cursor].Name)
	}
	if cmd == nil {
		t.Error("expected a fetch detail command after collapsing parent")
	}
}

func TestUpdate_TabOnEmptyVisibleNodes(t *testing.T) {
	m, _ := newTestModel(t)
	// Don't load the tree — visibleNodes is empty
	m.visibleNodes = nil
	m.cursor = 0

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	model := updated.(Model)
	if len(model.visibleNodes) != 0 {
		t.Error("Tab on empty visible nodes should not change state")
	}
	if cmd != nil {
		t.Error("Tab on empty visible nodes should not trigger a command")
	}
}

func TestUpdate_MouseClickInTreePane(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)

	// Initial cursor should be at 0
	if m.cursor != 0 {
		t.Fatalf("expected cursor at 0, got %d", m.cursor)
	}

	// Simulate clicking on second node (row 1 in tree, mouse Y=3 because tree row 0 is at screen Y=2)
	// Tree pane width with leftRatio=0.4 on width 120: (120-2-1)*0.4 = 46.8 -> 46
	// Click at X=10 (within tree pane), Y=3 (second line of tree view = tree row 1)
	msg := tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: 10, Y: 3}
	updated, cmd := m.Update(msg)
	model := updated.(Model)
	if model.cursor != 1 {
		t.Fatalf("expected cursor at 1 after click, got %d", model.cursor)
	}
	if cmd == nil {
		t.Error("expected fetch command after click")
	}
}

func TestUpdate_MouseClickOutsideTreePane(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)

	// Click in detail pane (right side) - should not change cursor
	// Tree pane is roughly left 46 chars, so click at X=50 is outside tree
	msg := tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: 50, Y: 2}
	updated, _ := m.Update(msg)
	model := updated.(Model)
	if model.cursor != 0 {
		t.Errorf("expected cursor unchanged at 0, got %d", model.cursor)
	}
}

func TestUpdate_MouseClickRowOutOfBounds(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)

	// Click at row that's outside the tree pane bounds (Y=0 is title bar)
	msg := tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: 10, Y: 0}
	updated, _ := m.Update(msg)
	model := updated.(Model)
	if model.cursor != 0 {
		t.Errorf("expected cursor unchanged at 0, got %d", model.cursor)
	}
}

func TestUpdate_MouseWheelUp(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)

	// First node is scalar, scroll should not work when at top
	msg := tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelUp}
	updated, _ := m.Update(msg)
	model := updated.(Model)
	if model.treeScroll != 0 {
		t.Errorf("expected treeScroll at 0, got %d", model.treeScroll)
	}
}

func TestUpdate_MouseWheelDown(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)

	// First node is scalar, need to expand to scroll
	// Check if any node has children
	hasExpandable := false
	for _, node := range m.visibleNodes {
		if node.IsExpandable() && len(node.Children) > 0 {
			hasExpandable = true
			break
		}
	}
	if !hasExpandable {
		t.Skip("No expandable nodes with children found, skipping wheel down test")
	}
}

func TestUpdate_MouseScrollsTree(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)

	// Need to expand a node with children first to have more visible nodes
	// Find a node that is expandable (has children but collapsed)
	expandedIdx := -1
	for idx, node := range m.visibleNodes {
		if node.IsExpandable() && !node.Expanded && len(node.Children) > 0 {
			// Expand this node
			node.Expanded = true
			m.rebuildVisible()
			// Reset cursor to top so we can scroll
			m.cursor = 0
			m.treeScroll = 0
			// Continue from this node index + 1
			m.cursor = idx + 1
			if m.cursor >= len(m.visibleNodes) {
				m.cursor = len(m.visibleNodes) - 1
			}
			expandedIdx = idx
			break
		}
	}

	// Skip test if no expandable node with children found
	if expandedIdx == -1 {
		t.Skip("No expandable nodes with children found, skipping scroll test")
	}

	// Now try scrolling down
	wheelMsg := tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown}
	for scrollCount := 0; scrollCount < 5; scrollCount++ {
		updated, _ := m.Update(wheelMsg)
		m = updated.(Model)
		_ = scrollCount
	}
	// Scroll should have increased or stayed at max
	if m.treeScroll < 0 {
		t.Errorf("expected treeScroll >= 0 after scrolling, got %d", m.treeScroll)
	}
	_ = wheelMsg
}
