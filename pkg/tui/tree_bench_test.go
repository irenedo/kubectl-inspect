package tui

import (
	"os"
	"testing"

	"github.com/irenedo/kubectl-inspect/pkg/explain"
)

func readFixture(b *testing.B, path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		b.Skipf("fixture not available: %v", err)
	}
	return string(data)
}

func BenchmarkVisibleNodes_Simple(b *testing.B) {
	input := readFixture(b, "../explain/testdata/expandable_first.txt")
	info, err := explain.ParseRecursive(input)
	if err != nil {
		b.Fatalf("unexpected error: %v", err)
	}
	b.ResetTimer()
	for b.Loop() {
		_ = VisibleNodes(info.Fields)
	}
}

func BenchmarkVisibleNodes_Deployment(b *testing.B) {
	input := readFixture(b, "../explain/testdata/deployment_recursive.txt")
	info, err := explain.ParseRecursive(input)
	if err != nil {
		b.Fatalf("unexpected error: %v", err)
	}
	b.ResetTimer()
	for b.Loop() {
		_ = VisibleNodes(info.Fields)
	}
}

func BenchmarkVisibleNodes_RealOutput(b *testing.B) {
	input := readFixture(b, "../explain/testdata/real_deployment_recursive.txt")
	info, err := explain.ParseRecursive(input)
	if err != nil {
		b.Fatalf("unexpected error: %v", err)
	}
	b.ResetTimer()
	for b.Loop() {
		_ = VisibleNodes(info.Fields)
	}
}
