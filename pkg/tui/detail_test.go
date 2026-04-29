package tui

import (
	"strings"
	"testing"
)

func TestRenderDetail_Loading(t *testing.T) {
	result := RenderDetail("", true, 0, 0, 40, 10)
	if !strings.Contains(result, "Fetching") {
		t.Error("expected 'Fetching...' when loading")
	}
}

func TestRenderDetail_Empty(t *testing.T) {
	result := RenderDetail("", false, 0, 0, 40, 10)
	if !strings.Contains(result, "Select a field") {
		t.Error("expected prompt to select a field when text is empty")
	}
}

func TestRenderDetail_WithContent(t *testing.T) {
	text := "KIND: Deployment\nVERSION: apps/v1\nDESCRIPTION:\n  A deployment.\nFIELDS:\n  replicas\t<integer>\n"
	result := RenderDetail(text, false, 0, 7, 60, 20)
	if !strings.Contains(result, "KIND") {
		t.Error("expected KIND in rendered detail")
	}
}

func TestRenderDetail_ScrollBeyondContent(t *testing.T) {
	text := "line1\nline2\nline3\n"
	result := RenderDetail(text, false, 100, 3, 40, 10)
	// Should just be empty/padded lines
	for _, line := range strings.Split(result, "\n") {
		if strings.TrimSpace(line) != "" {
			t.Errorf("expected empty lines when scrolled past content, got %q", line)
		}
	}
}

func TestRenderDetail_ScrollPartial(t *testing.T) {
	lines := make([]string, 20)
	for i := range lines {
		lines[i] = "line"
	}
	text := strings.Join(lines, "\n")
	result := RenderDetail(text, false, 5, 20, 40, 10)
	// Should render without panic
	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestColorizeDetail_Headers(t *testing.T) {
	text := "GROUP: apps\nKIND: Deployment\nVERSION: v1\nFIELD: spec\n"
	lines := colorizeDetail(text, 80)
	if len(lines) == 0 {
		t.Fatal("expected colorized lines")
	}
	// Each header line should have been processed
	found := 0
	for _, line := range lines {
		if strings.Contains(line, "GROUP") || strings.Contains(line, "KIND") ||
			strings.Contains(line, "VERSION") || strings.Contains(line, "FIELD") {
			found++
		}
	}
	if found < 4 {
		t.Errorf("expected at least 4 header lines, found %d", found)
	}
}

func TestColorizeDetail_Description(t *testing.T) {
	text := "DESCRIPTION:\n  This is a description.\n"
	lines := colorizeDetail(text, 80)
	foundHeader := false
	foundDesc := false
	for _, line := range lines {
		if strings.Contains(line, "DESCRIPTION") {
			foundHeader = true
		}
		if strings.Contains(line, "description") {
			foundDesc = true
		}
	}
	if !foundHeader {
		t.Error("expected DESCRIPTION header")
	}
	if !foundDesc {
		t.Error("expected description text")
	}
}

func TestColorizeDetail_Fields(t *testing.T) {
	text := "FIELDS:\n  replicas\t<integer>\n  selector\t<Object>\n"
	lines := colorizeDetail(text, 80)
	foundFields := false
	foundReplicas := false
	for _, line := range lines {
		if strings.Contains(line, "FIELDS") {
			foundFields = true
		}
		if strings.Contains(line, "replicas") {
			foundReplicas = true
		}
	}
	if !foundFields {
		t.Error("expected FIELDS header")
	}
	if !foundReplicas {
		t.Error("expected replicas field")
	}
}

func TestColorizeDetail_FieldDescriptionText(t *testing.T) {
	text := "FIELDS:\n  replicas\t<integer>\n  Number of desired pods.\n"
	lines := colorizeDetail(text, 80)
	foundDesc := false
	for _, line := range lines {
		if strings.Contains(line, "desired pods") {
			foundDesc = true
		}
	}
	if !foundDesc {
		t.Error("expected field description text")
	}
}

func TestColorizeDetail_EmptyLines(t *testing.T) {
	text := "KIND: Deployment\n\nVERSION: v1\n"
	lines := colorizeDetail(text, 80)
	hasEmpty := false
	for _, line := range lines {
		if line == "" {
			hasEmpty = true
			break
		}
	}
	if !hasEmpty {
		t.Error("expected empty lines to be preserved")
	}
}

func TestColorizeDetail_DefaultLine(t *testing.T) {
	text := "some random text that is not a header\n"
	lines := colorizeDetail(text, 80)
	if len(lines) == 0 {
		t.Error("expected at least one line")
	}
}

func TestWrapLines_ShortLine(t *testing.T) {
	result := wrapLines("hello", 80)
	if len(result) != 1 || result[0] != "hello" {
		t.Errorf("expected single line 'hello', got %v", result)
	}
}

func TestWrapLines_LongLine(t *testing.T) {
	long := strings.Repeat("word ", 30) // ~150 chars
	result := wrapLines(long, 40)
	if len(result) < 2 {
		t.Errorf("expected wrapped lines, got %d lines", len(result))
	}
	for _, line := range result {
		// Allow some slack for word boundaries
		if len(line) > 45 {
			t.Errorf("wrapped line too long: %d chars", len(line))
		}
	}
}

func TestWrapLines_ZeroWidth(t *testing.T) {
	result := wrapLines("hello world", 0)
	// Should default to 80
	if len(result) != 1 {
		t.Errorf("expected 1 line with default width, got %d", len(result))
	}
}

func TestCenterText(t *testing.T) {
	result := centerText("hello", 20, 5)
	lines := strings.Split(result, "\n")
	if len(lines) != 5 {
		t.Fatalf("expected 5 lines, got %d", len(lines))
	}
	// Middle line should contain the text
	if !strings.Contains(lines[2], "hello") {
		t.Errorf("expected 'hello' in middle line, got %q", lines[2])
	}
}

func TestCenterText_WiderThanWidth(t *testing.T) {
	result := centerText("a very long text", 5, 3)
	lines := strings.Split(result, "\n")
	// Should not panic, pad should be 0
	if !strings.Contains(lines[1], "a very long text") {
		t.Errorf("expected text in middle line, got %q", lines[1])
	}
}

func TestTruncateOrPad_Truncate(t *testing.T) {
	result := truncateOrPad("hello world", 5)
	if result != "hello" {
		t.Errorf("expected 'hello', got %q", result)
	}
}

func TestTruncateOrPad_Pad(t *testing.T) {
	result := truncateOrPad("hi", 5)
	if result != "hi   " {
		t.Errorf("expected 'hi   ', got %q", result)
	}
}

func TestTruncateOrPad_Exact(t *testing.T) {
	result := truncateOrPad("hello", 5)
	if result != "hello" {
		t.Errorf("expected 'hello', got %q", result)
	}
}
