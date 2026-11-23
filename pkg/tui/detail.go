package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

var (
	detailHeaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("39")).
				Bold(true)
	detailValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))
	detailDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("250"))
	detailFieldNameStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("117"))
	detailFieldTypeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("245"))
	detailDimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("242"))
)

// RenderDetail renders the right-pane detail view with colors.
func RenderDetail(text string, loading bool, scroll, lineCount, width, height int) string {
	if loading {
		return centerText(detailDimStyle.Render("Fetching..."), width, height)
	}
	if text == "" {
		return centerText(detailDimStyle.Render("Select a field to view details."), width, height)
	}

	lines := colorizeDetail(text, width)

	// Apply scroll offset
	if scroll > 0 {
		if scroll >= lineCount {
			lines = nil
		} else {
			lines = lines[scroll:]
		}
	}

	// Take only what fits in the viewport
	if len(lines) > height {
		lines = lines[:height]
	}

	// Pad to fill height
	for len(lines) < height {
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

// colorizeDetail applies syntax highlighting to kubectl explain output.
func colorizeDetail(text string, width int) []string {
	var result []string
	inDescription := false
	inFields := false

	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)

		switch {
		case strings.HasPrefix(trimmed, "GROUP:"),
			strings.HasPrefix(trimmed, "KIND:"),
			strings.HasPrefix(trimmed, "VERSION:"),
			strings.HasPrefix(trimmed, "FIELD:"):
			inDescription = false
			inFields = false
			parts := strings.SplitN(trimmed, ":", 2)
			colored := detailHeaderStyle.Render(parts[0]+":") + " " + detailValueStyle.Render(strings.TrimSpace(parts[1]))
			result = append(result, " "+colored)

		case trimmed == "DESCRIPTION:":
			inDescription = true
			inFields = false
			result = append(result, "", " "+detailHeaderStyle.Render("DESCRIPTION:"))

		case trimmed == "FIELDS:":
			inDescription = false
			inFields = true
			result = append(result, "", " "+detailHeaderStyle.Render("FIELDS:"))

		case inFields && strings.Contains(line, "\t"):
			// Field line: "  name\t<type>"
			parts := strings.SplitN(strings.TrimSpace(line), "\t", 2)
			name := detailFieldNameStyle.Render(parts[0])
			typeStr := ""
			if len(parts) > 1 {
				typeStr = "\t" + detailFieldTypeStyle.Render(parts[1])
			}
			result = append(result, "   "+name+typeStr)

		case inFields && trimmed != "":
			// Description text under a field entry
			result = append(result, "   "+detailDescStyle.Render(trimmed))

		case inDescription && trimmed != "":
			wrapped := wrapLines("   "+trimmed, width)
			for i, w := range wrapped {
				wrapped[i] = detailDescStyle.Render(w)
			}
			result = append(result, wrapped...)

		case trimmed == "":
			result = append(result, "")

		default:
			result = append(result, " "+detailDescStyle.Render(line))
		}
	}

	return result
}

func centerText(text string, width, height int) string {
	lines := make([]string, height)
	midRow := height / 2
	for i := range lines {
		if i == midRow {
			pad := (width - runewidth.StringWidth(text)) / 2
			if pad < 0 {
				pad = 0
			}
			lines[i] = strings.Repeat(" ", pad) + text
		} else {
			lines[i] = ""
		}
	}
	return strings.Join(lines, "\n")
}

func wrapLines(text string, width int) []string {
	if width <= 0 {
		width = 80
	}
	lines := strings.Split(text, "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		result = append(result, wrapLine(line, width)...)
	}
	return result
}

// wrapLine wraps a single logical line at width, breaking on spaces when
// possible and hard-breaking when a word is longer than width.
func wrapLine(line string, width int) []string {
	if len(line) <= width {
		return []string{line}
	}
	var out []string
	for len(line) > width {
		breakAt := findBreak(line, width)
		out = append(out, line[:breakAt])
		line = line[breakAt:]
		if line != "" && line[0] == ' ' {
			line = line[1:]
		}
	}
	if line != "" {
		out = append(out, line)
	}
	return out
}

// findBreak returns the index to break at: the last space in the second half
// of the overflowing segment, or width when the segment has no usable space.
func findBreak(line string, width int) int {
	for i := width; i > width/2; i-- {
		if line[i] == ' ' {
			return i
		}
	}
	return width
}
