package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
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
func RenderDetail(text string, loading bool, scroll, width, height int) string {
	if loading {
		return centerText(detailDimStyle.Render("Fetching..."), width, height)
	}
	if text == "" {
		return centerText(detailDimStyle.Render("Select a field to view details."), width, height)
	}

	lines := colorizeDetail(text, width)

	// Apply scroll offset
	if scroll > 0 {
		if scroll >= len(lines) {
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
			wrapped := wrapLines("   "+detailDescStyle.Render(trimmed), width)
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
			pad := (width - len(text)) / 2
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
	var result []string
	for _, line := range strings.Split(text, "\n") {
		if len(line) <= width {
			result = append(result, line)
		} else {
			for len(line) > width {
				breakAt := width
				for i := width; i > width/2; i-- {
					if line[i] == ' ' {
						breakAt = i
						break
					}
				}
				result = append(result, line[:breakAt])
				line = line[breakAt:]
				if line != "" && line[0] == ' ' {
					line = line[1:]
				}
			}
			if line != "" {
				result = append(result, line)
			}
		}
	}
	return result
}
