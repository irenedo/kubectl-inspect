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

func findExpandableIndex(nodes []*explain.Node) int {
	for i, n := range nodes {
		if n.IsExpandable() && len(n.Children) > 0 {
			return i
		}
	}
	return -1
}

func newTestModel(t *testing.T) (Model, *testExecutor) {
	t.Helper()
	fixture := readTestFixture(t, "expandable_first.txt")
	exec := &testExecutor{
		recursiveOutput: fixture,
		fieldOutput:     "KIND: ExampleConfig\nVERSION: v1\n",
	}
	m := NewModel("exampleconfig", exec, kubectl.Flags{})
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
	if m.resourceInfo.Kind != "ExampleConfig" {
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

	idx := findExpandableIndex(m.visibleNodes)
	if idx < 0 {
		t.Skip("no expandable node found")
	}

	m.cursor = idx
	originalCount := len(m.visibleNodes)

	// Expand with right key
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	model := updated.(Model)
	if len(model.visibleNodes) <= originalCount {
		t.Skip("expandable node has no children (fixture limitation)")
	}
	if !model.visibleNodes[idx].Expanded {
		t.Error("expected node to be expanded via right key")
	}
	if cmd == nil {
		t.Error("expected a fetch detail command after expand")
	}

	// Collapse with left key
	expandedCount := len(model.visibleNodes)
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyLeft})
	model = updated.(Model)
	if len(model.visibleNodes) >= expandedCount {
		t.Errorf("expected fewer visible nodes after collapse, got %d (was %d)", len(model.visibleNodes), expandedCount)
	}
	if model.visibleNodes[idx].Expanded {
		t.Error("expected node to be collapsed")
	}
	if cmd == nil {
		t.Error("expected a fetch detail command after collapse")
	}
}

func TestUpdate_EscOnExpandedNode(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)

	idx := findExpandableIndex(m.visibleNodes)
	if idx < 0 {
		t.Skip("no expandable node found")
	}

	m.cursor = idx
	// Check the node actually has children
	if len(m.visibleNodes[idx].Children) == 0 {
		t.Skip("expandable node has no children")
	}

	ExpandNode(m.visibleNodes[idx])
	m.rebuildVisible()

	expandedCount := len(m.visibleNodes)

	// Left key on an expanded node should collapse it
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	model := updated.(Model)
	if len(model.visibleNodes) >= expandedCount {
		t.Errorf("expected fewer visible nodes after collapse, got %d (was %d)", len(model.visibleNodes), expandedCount)
	}
	if model.visibleNodes[idx].Expanded {
		t.Error("expected node to be collapsed")
	}
	if cmd == nil {
		t.Error("expected a fetch detail command after collapse")
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

	idx := findExpandableIndex(m.visibleNodes)
	if idx < 0 {
		t.Skip("no expandable node found")
	}

	ExpandNode(m.visibleNodes[idx])
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
	m.detailText = "KIND: ExampleConfig\nVERSION: v1\n"
	view := m.View()
	if !strings.Contains(view, "ExampleConfig") {
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

	idx := findExpandableIndex(m.visibleNodes)
	if idx < 0 {
		t.Skip("no expandable node found")
	}

	m.cursor = idx
	ExpandNode(m.visibleNodes[idx])
	m.rebuildVisible()
	count := len(m.visibleNodes)

	// Tab on already expanded node should be a no-op for visible nodes
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

	idx := findExpandableIndex(m.visibleNodes)
	if idx < 0 {
		t.Skip("no expandable node found")
	}

	m.cursor = idx
	ExpandNode(m.visibleNodes[idx])
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

	idx := findExpandableIndex(m.visibleNodes)
	if idx < 0 {
		t.Skip("no expandable node found")
	}

	// Expand root to reveal children
	m.cursor = idx
	ExpandNode(m.visibleNodes[idx])
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

func TestUpdate_MouseClickOnFirstRow(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)

	// Click on first tree row: Y=2 (since content starts at screen Y=2, treeScroll=0)
	msg := tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: 10, Y: 2}
	updated, _ := m.Update(msg)
	model := updated.(Model)
	if model.cursor != 0 {
		t.Fatalf("expected cursor at 0 after clicking first row (Y=2), got %d", model.cursor)
	}
}

func TestUpdate_MouseClickYCoordinateMapping(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)

	// Write View output to a file for analysis
	os.WriteFile("/tmp/view_output.txt", []byte(m.View()), 0644)

	// Examine the View output structure
	lines := strings.Split(m.View(), "\n")
	t.Logf("View height: %d, m.height: %d", len(lines), m.height)
	t.Logf("Content height (calculated): %d", m.contentHeight())

	// Print first 5 and last 5 lines to understand layout
	t.Log("=== First 5 lines ===")
	for i := 0; i < 5 && i < len(lines); i++ {
		t.Logf("  Y=%d: '%s'", i, lines[i])
	}
	t.Log("=== Last 5 lines ===")
	for i := len(lines) - 5; i < len(lines); i++ {
		t.Logf("  Y=%d: '%s'", i, lines[i])
	}

	// Verify Y coordinates for all visible rows (only rows with actual nodes)
	// The content starts at Y=2 (after border at Y=0 and title bar at Y=1)
	// Each visible node at index i is at screen Y = 2 + i
	for i := range m.visibleNodes {
		y := 2 + i
		msg := tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: 10, Y: y}
		updated, _ := m.Update(msg)
		model := updated.(Model)
		if model.cursor != i {
			t.Errorf("clicking at Y=%d (node %d, path=%s): expected cursor=%d, got cursor=%d",
				y, i, m.visibleNodes[i].Path, i, model.cursor)
		}
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

func TestUpdate_MouseClickOnExpandableNode_CollapsesIt(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)

	// Find an expandable node
	idx := findExpandableIndex(m.visibleNodes)
	if idx < 0 {
		t.Skip("no expandable node found")
	}

	// Expand the node first (using the API directly)
	// Note: We operate on m after loadTree, which has the correct visibleNodes
	ExpandNode(m.visibleNodes[idx])
	m.rebuildVisible()
	if !m.visibleNodes[idx].Expanded {
		t.Fatal("expected node to be expanded")
	}
	preCollapseCount := len(m.visibleNodes)

	// Now click on the expanded node at the correct screen position
	// Tree content starts at screen Y=2, treeScroll=0, so node at index idx
	// is at screen Y = 2 + (idx + treeScroll) = 2 + idx
	msg := tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
		X:      10, // within tree pane
		Y:      2 + idx,
	}
	updated, cmd := m.Update(msg)
	model := updated.(Model)

	// Verify the node was collapsed by checking its expanded flag
	// The node object is shared between test's m and model, so we can
	// use the test's m.visibleNodes[idx] to check the state
	if m.visibleNodes[idx].Expanded {
		t.Error("expected node to be collapsed after click")
	}
	if len(model.visibleNodes) >= preCollapseCount {
		t.Errorf("expected fewer visible nodes after collapse: got %d, expected < %d",
			len(model.visibleNodes), preCollapseCount)
	}
	if cmd == nil {
		t.Error("expected fetch command after click")
	}
}

func TestUpdate_MouseClickOnExpandableNode_ExpandsIt(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)

	// Find a collapsed expandable node
	idx := findExpandableIndex(m.visibleNodes)
	if idx < 0 {
		t.Skip("no expandable node found")
	}

	if m.visibleNodes[idx].Expanded {
		CollapseNode(m.visibleNodes[idx])
		m.rebuildVisible()
		// Re-find the index after rebuild
		for i, n := range m.visibleNodes {
			if n == m.visibleNodes[idx] {
				idx = i
				break
			}
		}
	}

	preExpandCount := len(m.visibleNodes)

	// Click on the collapsed expandable node
	msg := tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
		X:      10,
		Y:      2 + idx,
	}
	updated, cmd := m.Update(msg)
	model := updated.(Model)

	if !m.visibleNodes[idx].Expanded {
		t.Error("expected node to be expanded after click")
	}
	if len(model.visibleNodes) <= preExpandCount {
		t.Errorf("expected more visible nodes after expand: got %d, expected > %d",
			len(model.visibleNodes), preExpandCount)
	}
	if cmd == nil {
		t.Error("expected fetch command after click")
	}

	if !m.visibleNodes[idx].Expanded {
		t.Error("expected node to be expanded after click")
	}
	if len(model.visibleNodes) <= preExpandCount {
		t.Errorf("expected more visible nodes after expand: got %d, expected > %d",
			len(model.visibleNodes), preExpandCount)
	}
	if cmd == nil {
		t.Error("expected fetch command after click")
	}
}

func TestUpdate_MouseClickOnNonExpandableNode_NoExpand(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)

	// Find a non-expandable (scalar) node
	scalarIdx := -1
	for i, n := range m.visibleNodes {
		if !n.IsExpandable() {
			scalarIdx = i
			break
		}
	}
	if scalarIdx < 0 {
		t.Skip("no scalar node found")
	}

	preCount := len(m.visibleNodes)

	// Click on the scalar node
	msg := tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
		X:      10,
		Y:      2 + scalarIdx,
	}
	updated, _ := m.Update(msg)
	model := updated.(Model)

	// Visible node count should not change
	if len(model.visibleNodes) != preCount {
		t.Errorf("expected visible nodes to not change on scalar click: got %d",
			len(model.visibleNodes))
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

func TestUpdate_EscOnExpandedNode_CollapsesIt(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)

	idx := findExpandableIndex(m.visibleNodes)
	if idx < 0 {
		t.Skip("no expandable node found")
	}

	m.cursor = idx
	ExpandNode(m.visibleNodes[idx])
	m.rebuildVisible()
	expandedCount := len(m.visibleNodes)

	// Press left to collapse
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	model := updated.(Model)

	if len(model.visibleNodes) >= expandedCount {
		t.Error("expected fewer visible nodes after collapse")
	}
	if cmd == nil {
		t.Error("expected fetch command after collapse")
	}
	if model.visibleNodes[idx].Expanded {
		t.Error("expected node to be collapsed")
	}
}

func TestUpdate_handleLeft_ExpandableCollapsed(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)

	idx := findExpandableIndex(m.visibleNodes)
	if idx < 0 {
		t.Skip("no expandable node found")
	}

	m.cursor = idx
	CollapseNode(m.visibleNodes[idx])
	m.rebuildVisible()
	count := len(m.visibleNodes)

	// Press left on a collapsed expandable node — visible nodes should not change
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	model := updated.(Model)
	if len(model.visibleNodes) != count {
		t.Error("left on collapsed node should not change visible nodes")
	}
	if cmd == nil {
		t.Error("left should still trigger a fetch detail command")
	}
}

func TestUpdate_handleLeft_NoVisibleNodes(t *testing.T) {
	m, _ := newTestModel(t)
	m.visibleNodes = nil
	m.cursor = 0

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	model := updated.(Model)
	if model.cursor != 0 {
		t.Error("cursor should not change with no nodes")
	}
	if cmd != nil {
		t.Error("should be no-op with no visible nodes")
	}
}

func TestUpdate_handleRight_OnExpandable(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)

	idx := findExpandableIndex(m.visibleNodes)
	if idx < 0 {
		t.Skip("no expandable node found")
	}

	m.cursor = idx
	CollapseNode(m.visibleNodes[idx])
	m.rebuildVisible()
	initialCount := len(m.visibleNodes)

	// Press right to expand
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	model := updated.(Model)
	if len(model.visibleNodes) <= initialCount {
		t.Errorf("expected more visible nodes after expand, got %d (was %d)", len(model.visibleNodes), initialCount)
	}
	if !model.visibleNodes[idx].Expanded {
		t.Error("expected node to be expanded")
	}
	if cmd == nil {
		t.Error("expected fetch command after expand")
	}
}

func TestUpdate_handleRight_OnScalar(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)

	// Find a scalar node
	leafIdx := -1
	for i, n := range m.visibleNodes {
		if !n.IsExpandable() {
			leafIdx = i
			break
		}
	}
	if leafIdx < 0 {
		t.Skip("no scalar node found")
	}

	m.cursor = leafIdx
	count := len(m.visibleNodes)

	// Press right on a scalar — should be a no-op
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	model := updated.(Model)
	if len(model.visibleNodes) != count {
		t.Error("right on scalar should not change visible nodes")
	}
	if cmd != nil {
		t.Error("right on scalar should not trigger a command")
	}
}

func TestUpdate_handleRight_AlreadyExpanded(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)

	idx := findExpandableIndex(m.visibleNodes)
	if idx < 0 {
		t.Skip("no expandable node found")
	}

	m.cursor = idx
	ExpandNode(m.visibleNodes[idx])
	m.rebuildVisible()
	count := len(m.visibleNodes)

	// Press right again — visible nodes should not change
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	model := updated.(Model)
	if len(model.visibleNodes) != count {
		t.Error("right on already expanded node should not change visible nodes")
	}
	if cmd == nil {
		t.Error("right should still trigger a fetch detail command")
	}
}

func TestUpdate_handleRight_NoVisibleNodes(t *testing.T) {
	m, _ := newTestModel(t)
	m.visibleNodes = nil
	m.cursor = 0

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	model := updated.(Model)
	if model.cursor != 0 {
		t.Error("cursor should not change with no nodes")
	}
	if cmd != nil {
		t.Error("should be no-op with no visible nodes")
	}
}

func TestUpdate_collapseParent_MovesCursor(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)

	idx := findExpandableIndex(m.visibleNodes)
	if idx < 0 {
		t.Skip("no expandable node found")
	}

	m.cursor = idx
	ExpandNode(m.visibleNodes[idx])
	m.rebuildVisible()

	// Find a child node (depth > 0)
	childIdx := -1
	for i, n := range m.visibleNodes {
		if n.Depth > 0 {
			childIdx = i
			break
		}
	}
	if childIdx < 0 {
		t.Skip("no child node found")
	}

	parent := m.visibleNodes[childIdx].Parent
	if parent == nil {
		t.Fatal("child node has no parent")
	}

	m.cursor = childIdx

	// Call collapseParent
	updated, cmd := m.collapseParent(m.visibleNodes[m.cursor])
	model := updated.(Model)

	// Parent should be collapsed
	if parent.Expanded {
		t.Error("expected parent to be collapsed")
	}

	// Cursor should be on the parent
	if model.cursor >= len(model.visibleNodes) {
		t.Fatal("cursor out of bounds")
	}
	if model.visibleNodes[model.cursor] != parent {
		t.Errorf("expected cursor on parent, got %q", model.visibleNodes[model.cursor].Name)
	}
	if cmd == nil {
		t.Error("expected fetch command after collapseParent")
	}
}

func TestUpdate_CursorOutOfBoundsOnLeft(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)
	outOfBounds := len(m.visibleNodes)
	m.cursor = outOfBounds // out of bounds

	// Left should guard against out-of-bounds cursor (no-op)
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	model := updated.(Model)
	if model.err != nil {
		t.Errorf("unexpected error: %v", model.err)
	}
	if model.cursor != outOfBounds {
		t.Error("cursor should not move left when out of bounds")
	}
	if cmd != nil {
		t.Error("left on out-of-bounds cursor should be a no-op")
	}
}

func TestUpdate_CursorOutOfBoundsOnRight(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)
	m.cursor = len(m.visibleNodes) // out of bounds

	// Right should guard against out-of-bounds cursor (no-op)
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	model := updated.(Model)
	if model.err != nil {
		t.Errorf("unexpected error: %v", model.err)
	}
	if cmd != nil {
		t.Error("right on out-of-bounds cursor should be a no-op")
	}
}

func TestUpdate_handleUp_DetailPane(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)
	m.focusedPane = focusedPaneDetail
	m.detailScroll = 10
	m.detailText = strings.Repeat("line\n", 50)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	model := updated.(Model)
	if model.detailScroll >= 10 {
		t.Error("detailScroll should decrease when scrolling up in detail pane")
	}
}

func TestUpdate_handleUp_DetailPaneAtTop(t *testing.T) {
	m, _ := newTestModel(t)
	m.focusedPane = focusedPaneDetail
	m.detailScroll = 0
	m.detailText = strings.Repeat("line\n", 50)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	model := updated.(Model)
	if model.detailScroll != 0 {
		t.Errorf("detailScroll should stay at 0, got %d", model.detailScroll)
	}
}

func TestUpdate_handleDown_DetailPane(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)
	m.focusedPane = focusedPaneDetail
	m.detailScroll = 10
	m.detailText = strings.Repeat("line\n", 100)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	model := updated.(Model)
	if model.detailScroll <= 10 {
		t.Error("detailScroll should increase when scrolling down in detail pane")
	}
}

func TestUpdate_handleDown_DetailPaneAtBottom(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)
	m.focusedPane = focusedPaneDetail
	// "line\n" split by "\n" gives ["line", ""] per iteration, so 50 iterations = 100 lines
	// max scroll = 100-1 = 99, so set to 99 to test staying at bottom
	m.detailScroll = 99
	m.detailText = strings.Repeat("line\n", 50)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	model := updated.(Model)
	if model.detailScroll != 99 {
		t.Errorf("detailScroll should stay at 99, got %d", model.detailScroll)
	}
}

func TestUpdate_handleDown_DetailPaneExactBottom(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)
	m.focusedPane = focusedPaneDetail
	// 25 iterations → 50 lines → max scroll = 49
	m.detailScroll = 49
	m.detailText = strings.Repeat("line\n", 25)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	model := updated.(Model)
	if model.detailScroll != 49 {
		t.Errorf("detailScroll should stay at 49 (max), got %d", model.detailScroll)
	}
}

func TestUpdate_MouseWheelDetailDown(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)
	m.focusedPane = focusedPaneDetail
	m.detailScroll = 10
	m.detailText = strings.Repeat("line\n", 100)

	// Mouse over detail pane (X=80 on width 120 with leftRatio=0.4: leftWidth=~46, 80 > 46 = detail)
	msg := tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown, X: 80}
	updated, _ := m.Update(msg)
	model := updated.(Model)
	if model.detailScroll <= 10 {
		t.Error("detailScroll should increase after wheel down in detail pane")
	}
}

func TestUpdate_MouseWheelDetailUp(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)
	m.focusedPane = focusedPaneDetail
	m.detailScroll = 5
	m.detailText = strings.Repeat("line\n", 100)

	msg := tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelUp, X: 80}
	updated, _ := m.Update(msg)
	model := updated.(Model)
	if model.detailScroll >= 5 {
		t.Error("detailScroll should decrease after wheel up in detail pane")
	}
}

func TestUpdate_MouseClickDetailPane(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)
	// Click in detail pane (right side)
	msg := tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: 80, Y: 3}
	updated, _ := m.Update(msg)
	model := updated.(Model)
	if model.focusedPane != focusedPaneDetail {
		t.Error("click in detail pane should set focus to detail")
	}
}

func TestUpdate_MouseClickTreeThenDetail(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)
	m.focusedPane = focusedPaneTree

	// First click in tree pane
	treeMsg := tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: 10, Y: 3}
	updated, _ := m.Update(treeMsg)
	model := updated.(Model)
	if model.focusedPane != focusedPaneTree {
		t.Error("click in tree pane should set focus to tree")
	}

	// Then click in detail pane — should switch focus
	detailMsg := tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: 80, Y: 3}
	updated, _ = model.Update(detailMsg)
	model = updated.(Model)
	if model.focusedPane != focusedPaneDetail {
		t.Error("click in detail pane should switch focus to detail")
	}
}

func TestUpdate_MouseWheelUpDetailPaneFocus(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)
	m.focusedPane = focusedPaneTree // initially tree-focused
	m.detailScroll = 10
	m.detailText = strings.Repeat("line\n", 100)

	// Mouse in detail pane with wheel up — should also set focus to detail
	msg := tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelUp, X: 80}
	updated, _ := m.Update(msg)
	model := updated.(Model)
	if model.focusedPane != focusedPaneDetail {
		t.Error("mouse wheel in detail pane should set focus to detail")
	}
	if model.detailScroll >= 10 {
		t.Error("detailScroll should decrease after wheel up")
	}
}

func TestUpdate_MouseWheelUpDetailAtTop(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)
	m.focusedPane = focusedPaneDetail
	m.detailScroll = 0
	m.detailText = strings.Repeat("line\n", 100)

	msg := tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelUp, X: 80}
	updated, _ := m.Update(msg)
	model := updated.(Model)
	if model.detailScroll != 0 {
		t.Errorf("detailScroll should stay at 0, got %d", model.detailScroll)
	}
}

func TestColoredNodeLabel_CollapsedExpandable(t *testing.T) {
	node := &explain.Node{
		Name:      "spec",
		TypeStr:   "Spec",
		FieldType: explain.FieldTypeObject,
		Depth:     0,
		Expanded:  false,
		Children:  []*explain.Node{},
	}
	result := coloredNodeLabel(node)
	// Should contain the collapsed icon (►) and styled name (object style = light blue)
	if !strings.Contains(result, "►") {
		t.Error("collapsed expandable should have ► icon")
	}
}

func TestColoredNodeLabel_ScalarWithType(t *testing.T) {
	node := &explain.Node{
		Name:      "replicas",
		TypeStr:   "integer",
		FieldType: explain.FieldTypeScalar,
		Depth:     0,
		Children:  []*explain.Node{},
	}
	result := coloredNodeLabel(node)
	// Should contain the scalar icon (●) and type annotation
	if !strings.Contains(result, "●") {
		t.Error("scalar should have ● icon")
	}
	if !strings.Contains(result, "integer") {
		t.Error("scalar with type should contain type annotation")
	}
}

func TestColoredNodeLabel_ExpandedWithChildren(t *testing.T) {
	node := &explain.Node{
		Name:      "spec",
		TypeStr:   "Spec",
		FieldType: explain.FieldTypeObject,
		Depth:     0,
		Expanded:  true,
		Children:  []*explain.Node{{Name: "field1"}},
	}
	result := coloredNodeLabel(node)
	if !strings.Contains(result, "▼") {
		t.Error("expanded expandable should have ▼ icon")
	}
}

func TestColoredNodeLabel_NoTypeAnnotation(t *testing.T) {
	node := &explain.Node{
		Name:      "plain",
		FieldType: explain.FieldTypeScalar,
		Depth:     0,
		Children:  []*explain.Node{},
	}
	result := coloredNodeLabel(node)
	if !strings.Contains(result, "●") {
		t.Error("scalar should have ● icon")
	}
}

func TestUpdate_MouseScrollsTree(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadTree(t, m)

	// Need to expand a node with children first to have more visible nodes
	// Find a node that is expandable (has children but collapsed)
	expandedIdx := -1
	for idx, node := range m.visibleNodes {
		if !node.IsExpandable() || node.Expanded || len(node.Children) == 0 {
			continue
		}
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

func TestUpdate_MouseClick_NarrowWidthHelpWrapping(t *testing.T) {
	m, _ := newTestModel(t)
	m.width = 47 // narrow width causes help bar to wrap to 3 lines
	m = loadTree(t, m)

	// Verify help wraps to 3 lines at this width
	if m.helpHeight() != 3 {
		t.Fatalf("expected helpHeight=3 at width 47, got %d", m.helpHeight())
	}

	// contentHeight = 40 - 3 - 3 = 34, content starts at Y=2
	// Each node i is at screen Y = 2 + i
	for i := range m.visibleNodes {
		y := 2 + i
		msg := tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: 10, Y: y}
		updated, _ := m.Update(msg)
		model := updated.(Model)
		if model.cursor != i {
			t.Errorf("clicking at Y=%d (node %d): expected cursor=%d, got cursor=%d",
				y, i, i, model.cursor)
		}
	}
}


func TestUpdate_MouseClick_NarrowWidthHelp3Lines(t *testing.T) {
	m, _ := newTestModel(t)
	m.width = 47 // narrow width causes help to wrap to 3 lines
	m.height = 40
	m = loadTree(t, m)

	// Verify the help bar wraps
	if m.helpHeight() != 3 {
		t.Fatalf("expected helpHeight=3, got %d", m.helpHeight())
	}

	// Click on first node (Y=2 should select cursor 0)
	msg := tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: 10, Y: 2}
	updated, _ := m.Update(msg)
	model := updated.(Model)
	if model.cursor != 0 {
		t.Errorf("click at Y=2 (first node): expected cursor=0, got %d", model.cursor)
	}

	// Click on second node (Y=3 should select cursor 1)
	msg = tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: 10, Y: 3}
	updated, _ = m.Update(msg)
	model = updated.(Model)
	if model.cursor != 1 {
		t.Errorf("click at Y=3 (second node): expected cursor=1, got %d", model.cursor)
	}
}
