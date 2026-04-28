package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/irenedo/kubectl-inspect/pkg/explain"
	"github.com/irenedo/kubectl-inspect/pkg/kubectl"
)

// Message types for async operations.
type treeLoadedMsg struct {
	info *explain.ResourceInfo
}

type detailLoadedMsg struct {
	path   string
	result explain.DetailResult
}

type errMsg struct {
	err error
}

// modelMutator captures all mutating methods. *Model MUST satisfy this.
// If a mutating method is accidentally given a value receiver, *Model no longer
// satisfies this interface and the code won't compile.
type modelMutator interface {
	rebuildVisible()
	ensureCursorVisible()
	prepareFetchDetail() tea.Cmd
}

var _ modelMutator = (*Model)(nil)

// Model is the bubbletea model for the TUI.
type Model struct {
	resource     string
	resourceInfo *explain.ResourceInfo
	fetcher      *explain.Fetcher
	executor     kubectl.Executor
	flags        kubectl.Flags

	visibleNodes []*explain.Node
	cursor       int
	treeScroll   int

	detailText     string
	detailScroll   int
	detailLoading  bool
	lastDetailPath string

	copiedPath string

	width     int
	height    int
	leftRatio float64

	focusedPane string // "tree" or "detail"

	err error
}

// NewModel creates a new Model.
func NewModel(resource string, executor kubectl.Executor, flags kubectl.Flags) Model {
	return Model{
		resource:    resource,
		executor:    executor,
		flags:       flags,
		fetcher:     explain.NewFetcher(executor, resource, flags),
		leftRatio:   0.4,
		width:       80,
		height:      24,
		focusedPane: "tree",
	}
}

// Init returns the initial command to load the resource tree.
func (m Model) Init() tea.Cmd {
	return m.loadTreeCmd()
}

func (m Model) loadTreeCmd() tea.Cmd {
	return func() tea.Msg {
		output, err := m.executor.ExplainRecursive(context.Background(), m.resource, m.flags)
		if err != nil {
			return errMsg{err: err}
		}
		info, err := explain.ParseRecursive(output)
		if err != nil {
			return errMsg{err: err}
		}
		return treeLoadedMsg{info: info}
	}
}

func (m Model) fetchDetailCmd() tea.Cmd {
	if len(m.visibleNodes) == 0 || m.cursor >= len(m.visibleNodes) {
		return nil
	}
	node := m.visibleNodes[m.cursor]
	path := node.Path
	return func() tea.Msg {
		result := m.fetcher.FetchDetail(context.Background(), path)
		return detailLoadedMsg{path: path, result: result}
	}
}
