package tui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

	"github.com/irenedo/kubectl-inspect/pkg/kubectl"
)

var (
	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62"))
	titleBarStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("62")).
			Align(lipgloss.Center)
	helpBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Align(lipgloss.Center)
	helpKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true)
	helpSepStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))
	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)
	separatorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("62"))
)

// View renders the TUI.
func (m Model) View() string {
	if m.err != nil {
		return m.renderError()
	}
	if m.resourceInfo == nil {
		return m.renderLoading()
	}
	return m.renderNormal()
}

func (m Model) renderError() string {
	errText := m.err.Error()

	var notFound *kubectl.NotFoundError
	var execErr *kubectl.ExecError
	switch {
	case errors.As(m.err, &notFound):
		errText = "kubectl not found in PATH. Please install kubectl."
	case errors.As(m.err, &execErr):
		errText = execErr.FriendlyMessage()
	}

	msg := errorStyle.Render(fmt.Sprintf("Error: %s", errText))
	return borderStyle.Width(m.width - 2).Height(m.height - 2).Render(
		centerText(msg, m.width-4, m.height-4),
	)
}

func (m Model) renderLoading() string {
	msg := "Loading..."
	if m.width > 35 {
		msg += " (this may take a moment for large CRDs)"
	}
	return borderStyle.Width(m.width - 2).Height(m.height - 2).Render(
		centerText(msg, m.width-4, m.height-4),
	)
}

func (m Model) renderNormal() string {
	innerWidth := m.width - 2 // subtract border left+right

	// Top title bar: Kind + API group/version (+ copied feedback)
	titleText := fmt.Sprintf(" %s (%s) ", m.resourceInfo.Kind, m.resourceInfo.APIVersion())
	if m.copiedPath != "" {
		titleText = fmt.Sprintf(" %s (%s)  Copied: %s ", m.resourceInfo.Kind, m.resourceInfo.APIVersion(), m.copiedPath)
	}
	titleBar := titleBarStyle.Width(innerWidth).Render(titleText)

	// Bottom help bar
	helpBar := m.renderHelpBar(innerWidth)

	// Content area height: total - topBorder(1) - titleBar(1) - helpBar - bottomBorder(1)
	contentHeight := max(m.height-m.helpHeight()-3, 1)

	leftWidth := int(float64(innerWidth-1) * m.leftRatio) // -1 for separator
	rightWidth := innerWidth - 1 - leftWidth

	leftPane := m.renderTreePane(leftWidth, contentHeight)
	rightPane := m.renderDetailPane(rightWidth, contentHeight)
	sep := m.renderSeparator(contentHeight)

	content := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, sep, rightPane)

	// Stack: title + content + help
	full := lipgloss.JoinVertical(lipgloss.Left, titleBar, content, helpBar)
	return borderStyle.Render(full)
}

func (m Model) renderHelpBar(width int) string {
	return helpBarStyle.Width(width).Render(formatHelp())
}

// helpKey is a single keybinding entry shown in the help bar.
type helpKey struct {
	key  string
	desc string
}

// helpKeys is the single source of truth for the help bar entries,
// shared by the styled bar and the width calculation.
var helpKeys = []helpKey{
	{"↑/↓", "navigate"},
	{"←/→", "expand/collapse"},
	{"Tab", "focus pane"},
	{"Enter", "copy path"},
	{"PgUp/PgDn", "scroll detail"},
	{"q/Q", "quit"},
}

func formatHelp() string {
	sep := helpSepStyle.Render(" • ")
	parts := make([]string, len(helpKeys))
	for i, k := range helpKeys {
		parts[i] = helpKeyStyle.Render(k.key) + " " + k.desc
	}
	return strings.Join(parts, sep)
}

// plainHelpText returns the help text without any lipgloss styling,
// for use in layout calculations where ANSI codes would skew width.
func plainHelpText() string {
	parts := make([]string, len(helpKeys))
	for i, k := range helpKeys {
		parts[i] = k.key + " " + k.desc
	}
	return strings.Join(parts, " • ")
}

func (m Model) renderTreePane(width, height int) string {
	var lines []string

	// Visible nodes in viewport
	endIdx := min(m.treeScroll+height, len(m.visibleNodes))

	for i := m.treeScroll; i < endIdx; i++ {
		node := m.visibleNodes[i]
		isSelected := i == m.cursor
		line := FormatTreeLine(node, isSelected, width)
		lines = append(lines, line)
	}

	// Pad remaining lines
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderDetailPane(width, height int) string {
	return RenderDetail(m.detailText, m.detailLoading, m.detailScroll, m.detailLineCount, width, height)
}

func (m Model) renderSeparator(height int) string {
	lines := make([]string, height)
	for i := range lines {
		lines[i] = "│"
	}
	return separatorStyle.Render(strings.Join(lines, "\n"))
}

func truncateOrPad(s string, width int) string {
	visible := runewidth.StringWidth(s)
	if visible > width {
		return runewidth.Truncate(s, width, "")
	}
	return s + strings.Repeat(" ", width-visible)
}
