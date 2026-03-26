package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/irenedo/kubectl-inspect/pkg/explain"
)

var (
	// Selected line: white on dark cyan background
	selectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("24")).
			Foreground(lipgloss.Color("15")).
			Bold(true)

	// Tree node colors
	iconExpandedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("214")) // orange ▼
	iconCollapsedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))  // blue ►
	iconLeafStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("240")) // gray ●
	nameObjectStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("117")) // light blue for expandable
	nameScalarStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("252")) // white-ish for scalars
	typeStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("245")) // dim gray for type annotation
)

// VisibleNodes returns a flat slice of nodes that are currently visible,
// respecting the Expanded state of branch nodes.
func VisibleNodes(roots []*explain.Node) []*explain.Node {
	var result []*explain.Node
	for _, root := range roots {
		result = appendVisible(result, root)
	}
	return result
}

func appendVisible(result []*explain.Node, node *explain.Node) []*explain.Node {
	result = append(result, node)
	if node.IsExpandable() && node.Expanded {
		for _, child := range node.Children {
			result = appendVisible(result, child)
		}
	}
	return result
}

// ExpandNode expands a branch node. No-op on leaf nodes.
func ExpandNode(node *explain.Node) {
	if node.IsExpandable() {
		node.Expanded = true
	}
}

// CollapseNode collapses a branch node. No-op on leaf nodes.
func CollapseNode(node *explain.Node) {
	if node.IsExpandable() {
		node.Expanded = false
	}
}

// ToggleExpand toggles expand/collapse on a branch node. No-op on leaf nodes.
func ToggleExpand(node *explain.Node) {
	if node.IsExpandable() {
		node.Expanded = !node.Expanded
	}
}

// NodeIcon returns the display icon for a node.
func NodeIcon(node *explain.Node) string {
	if !node.IsExpandable() {
		return "●"
	}
	if node.Expanded {
		return "▼"
	}
	return "►"
}

// NodeLabel returns the formatted display label for a node (plain text, no styling).
func NodeLabel(node *explain.Node) string {
	indent := strings.Repeat("  ", node.Depth)
	icon := NodeIcon(node)
	typeAnnotation := ""
	if node.TypeStr != "" {
		typeAnnotation = fmt.Sprintf(" <%s>", node.TypeStr)
	}
	return fmt.Sprintf("%s%s %s%s", indent, icon, node.Name, typeAnnotation)
}

// coloredNodeLabel returns the label with lipgloss color styling applied.
func coloredNodeLabel(node *explain.Node) string {
	indent := strings.Repeat("  ", node.Depth)
	icon := NodeIcon(node)

	// Style icon
	var styledIcon string
	switch {
	case !node.IsExpandable():
		styledIcon = iconLeafStyle.Render(icon)
	case node.Expanded:
		styledIcon = iconExpandedStyle.Render(icon)
	default:
		styledIcon = iconCollapsedStyle.Render(icon)
	}

	// Style name
	var styledName string
	if node.IsExpandable() {
		styledName = nameObjectStyle.Render(node.Name)
	} else {
		styledName = nameScalarStyle.Render(node.Name)
	}

	// Style type
	styledType := ""
	if node.TypeStr != "" {
		styledType = " " + typeStyle.Render(fmt.Sprintf("<%s>", node.TypeStr))
	}

	return fmt.Sprintf("%s%s %s%s", indent, styledIcon, styledName, styledType)
}

// FormatTreeLine returns a styled line for the tree view.
func FormatTreeLine(node *explain.Node, isSelected bool, width int) string {
	if isSelected {
		// For selected: use plain text (no per-element colors) so the highlight is clean
		label := NodeLabel(node)
		if len(label) > width {
			label = label[:width]
		} else {
			label += strings.Repeat(" ", width-len(label))
		}
		return selectedStyle.Render(label)
	}

	// For non-selected: use colored label
	colored := coloredNodeLabel(node)
	plain := NodeLabel(node)
	// Pad based on plain-text length (colored has escape codes)
	if len(plain) < width {
		colored += strings.Repeat(" ", width-len(plain))
	}
	return colored
}
