package explain

import (
	"os"
	"strings"
	"testing"
)

func readFixture(b *testing.B, path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		b.Skipf("fixture not available: %v", err)
	}
	return string(data)
}

func BenchmarkParseRecursive_Deployment(b *testing.B) {
	input := readFixture(b, "testdata/deployment_recursive.txt")
	for b.Loop() {
		_, _ = ParseRecursive(input)
	}
}

func BenchmarkParseRecursive_CRD(b *testing.B) {
	input := readFixture(b, "testdata/crd_recursive.txt")
	for b.Loop() {
		_, _ = ParseRecursive(input)
	}
}

func BenchmarkParseRecursive_RealOutput(b *testing.B) {
	input := readFixture(b, "testdata/real_deployment_recursive.txt")
	for b.Loop() {
		_, _ = ParseRecursive(input)
	}
}

func BenchmarkParseFieldLine(b *testing.B) {
	lines := []string{
		"  image\t<string>",
		"      matchLabels\t<map[string]string>",
		"    selector\t<LabelSelector> -required-",
		"      replicas\t<integer>",
		"    template\t<Object>",
		"      spec\t<Object>",
		"        containers\t[]Container",
	}
	b.ResetTimer()
	for b.Loop() {
		for _, line := range lines {
			_, _, _, _ = parseFieldLine(line)
		}
	}
}

func BenchmarkClassifyType(b *testing.B) {
	types := []string{
		"string", "integer", "boolean", "IntOrString",
		"Object", "LabelSelector", "[]Container", "map[string]string",
	}
	b.ResetTimer()
	for b.Loop() {
		for _, t := range types {
			_ = classifyType(t)
		}
	}
}

func BenchmarkBuildTree(b *testing.B) {
	input := readFixture(b, "testdata/deployment_recursive.txt")
	lines := strings.Split(input, "\n")
	hdr, err := parseHeader(lines)
	if err != nil {
		b.Fatalf("unexpected error: %v", err)
	}
	fieldLines := lines[hdr.fieldsIdx:]
	b.ResetTimer()
	for b.Loop() {
		_ = buildTree(fieldLines)
	}
}
