package testutil

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/irenedo/kubectl-inspect/pkg/kubectl"
)

// ReadFixture reads a test fixture from the given path (relative to the calling test's directory).
// Use "../explain/testdata/foo.txt" to read from another package's testdata.
func ReadFixture(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", path))
	if err != nil {
		t.Fatalf("failed to read fixture %s: %v", path, err)
	}
	return string(data)
}

// ReadFixtureDirect reads a test fixture from an arbitrary path (relative to the calling test's directory).
func ReadFixtureDirect(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read fixture %s: %v", path, err)
	}
	return string(data)
}

// SplitLines splits a string into lines.
func SplitLines(s string) []string {
	return strings.Split(s, "\n")
}

// FakeDetail returns a string of n repeated "line\n" lines, useful for testing scroll behavior.
func FakeDetail(n int) string {
	return strings.Repeat("line\n", n)
}

// ExpectFetchCmd asserts that a tea.Cmd is non-nil (expected to trigger a detail fetch).
func ExpectFetchCmd(t *testing.T, cmd tea.Cmd) {
	t.Helper()
	if cmd == nil {
		t.Error("expected a fetch detail command")
	}
}

// ExpectNoCmd asserts that a tea.Cmd is nil (no command expected).
func ExpectNoCmd(t *testing.T, cmd tea.Cmd) {
	t.Helper()
	if cmd != nil {
		t.Error("expected no command")
	}
}

// AssertArgsEqual asserts that two string slices are identical.
func AssertArgsEqual(t *testing.T, expected, actual []string) {
	t.Helper()
	if len(expected) != len(actual) {
		t.Fatalf("expected %d args, got %d: %v", len(expected), len(actual), actual)
	}
	for i := range expected {
		if expected[i] != actual[i] {
			t.Errorf("arg[%d]: expected %q, got %q", i, expected[i], actual[i])
		}
	}
}

// MockExecutor implements kubectl.Executor for tests.
type MockExecutor struct {
	RecursiveOutput string
	RecursiveErr    error
	FieldOutput     string
	FieldErr        error
	LastFieldPath   string
	LastFlags       kubectl.Flags
}

// ExplainRecursive returns the configured recursive output and error.
func (m *MockExecutor) ExplainRecursive(_ context.Context, _ string, _ kubectl.Flags) (string, error) {
	return m.RecursiveOutput, m.RecursiveErr
}

// ExplainField returns the configured field output and error, recording the path and flags.
func (m *MockExecutor) ExplainField(_ context.Context, fieldPath string, flags kubectl.Flags) (string, error) {
	m.LastFieldPath = fieldPath
	m.LastFlags = flags
	return m.FieldOutput, m.FieldErr
}

// FindNode returns the first node matching the predicate, or nil.
func FindNode[T any](nodes []*T, pred func(*T) bool) *T {
	for _, n := range nodes {
		if pred(n) {
			return n
		}
	}
	return nil
}

// FindIndex returns the index of the first element matching the predicate, or -1.
func FindIndex[T any](nodes []*T, pred func(*T) bool) int {
	for i, n := range nodes {
		if pred(n) {
			return i
		}
	}
	return -1
}
