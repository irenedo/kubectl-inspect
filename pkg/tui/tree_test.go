package tui

import (
	"testing"

	"github.com/irenedo/kubectl-inspect/pkg/explain"
)

func makeTestTree() []*explain.Node {
	grandchild1 := &explain.Node{Name: "key", TypeStr: "string", FieldType: explain.FieldTypeScalar, Depth: 2, Path: "spec.selector.key"}
	grandchild2 := &explain.Node{Name: "operator", TypeStr: "string", FieldType: explain.FieldTypeScalar, Depth: 2, Path: "spec.selector.operator"}

	child1 := &explain.Node{Name: "replicas", TypeStr: "integer", FieldType: explain.FieldTypeScalar, Depth: 1, Path: "spec.replicas"}
	child2 := &explain.Node{
		Name:      "selector",
		TypeStr:   "Object",
		FieldType: explain.FieldTypeObject,
		Depth:     1,
		Path:      "spec.selector",
		Children:  []*explain.Node{grandchild1, grandchild2},
	}
	grandchild1.Parent = child2
	grandchild2.Parent = child2

	root1 := &explain.Node{
		Name:      "spec",
		TypeStr:   "Object",
		FieldType: explain.FieldTypeObject,
		Depth:     0,
		Path:      "spec",
		Children:  []*explain.Node{child1, child2},
	}
	child1.Parent = root1
	child2.Parent = root1

	root2 := &explain.Node{Name: "status", TypeStr: "Object", FieldType: explain.FieldTypeObject, Depth: 0, Path: "status"}

	return []*explain.Node{root1, root2}
}

func TestVisibleNodes_AllCollapsed(t *testing.T) {
	roots := makeTestTree()
	visible := VisibleNodes(roots)
	if len(visible) != 2 {
		t.Errorf("expected 2 visible nodes (roots only), got %d", len(visible))
	}
	if visible[0].Name != "spec" || visible[1].Name != "status" {
		t.Errorf("unexpected nodes: %v, %v", visible[0].Name, visible[1].Name)
	}
}

func TestVisibleNodes_OneExpanded(t *testing.T) {
	roots := makeTestTree()
	ExpandNode(roots[0]) // expand spec
	visible := VisibleNodes(roots)
	// spec + replicas + selector + status = 4
	if len(visible) != 4 {
		t.Errorf("expected 4 visible nodes, got %d", len(visible))
	}
	if visible[1].Name != "replicas" {
		t.Errorf("expected replicas at index 1, got %q", visible[1].Name)
	}
	if visible[2].Name != "selector" {
		t.Errorf("expected selector at index 2, got %q", visible[2].Name)
	}
}

func TestVisibleNodes_DeepExpansion(t *testing.T) {
	roots := makeTestTree()
	ExpandNode(roots[0])             // expand spec
	ExpandNode(roots[0].Children[1]) // expand selector
	visible := VisibleNodes(roots)
	// spec + replicas + selector + key + operator + status = 6
	if len(visible) != 6 {
		t.Errorf("expected 6 visible nodes, got %d", len(visible))
	}
	if visible[3].Name != "key" {
		t.Errorf("expected key at index 3, got %q", visible[3].Name)
	}
	if visible[4].Name != "operator" {
		t.Errorf("expected operator at index 4, got %q", visible[4].Name)
	}
}

func TestVisibleNodes_CollapseHidesDescendants(t *testing.T) {
	roots := makeTestTree()
	ExpandNode(roots[0])             // expand spec
	ExpandNode(roots[0].Children[1]) // expand selector
	CollapseNode(roots[0])           // collapse spec
	visible := VisibleNodes(roots)
	// Only spec + status visible
	if len(visible) != 2 {
		t.Errorf("expected 2 visible nodes after collapse, got %d", len(visible))
	}
}

func TestToggleExpand_Scalar(t *testing.T) {
	node := &explain.Node{Name: "replicas", TypeStr: "integer", FieldType: explain.FieldTypeScalar}
	toggleExpand(node)
	if node.Expanded {
		t.Error("scalar node should not be expandable")
	}
}

func TestToggleExpand_Object(t *testing.T) {
	node := &explain.Node{Name: "spec", TypeStr: "Object", FieldType: explain.FieldTypeObject}
	toggleExpand(node)
	if !node.Expanded {
		t.Error("expected node to be expanded after toggle")
	}
	toggleExpand(node)
	if node.Expanded {
		t.Error("expected node to be collapsed after second toggle")
	}
}

func TestNodeIcon(t *testing.T) {
	tests := []struct {
		name     string
		node     *explain.Node
		expected string
	}{
		{"expanded object", &explain.Node{FieldType: explain.FieldTypeObject, Expanded: true}, "▼"},
		{"collapsed object", &explain.Node{FieldType: explain.FieldTypeObject, Expanded: false}, "►"},
		{"expanded list", &explain.Node{FieldType: explain.FieldTypeList, Expanded: true}, "▼"},
		{"collapsed list", &explain.Node{FieldType: explain.FieldTypeList, Expanded: false}, "►"},
		{"scalar", &explain.Node{FieldType: explain.FieldTypeScalar}, "●"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NodeIcon(tt.node)
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestNodeLabel(t *testing.T) {
	node := &explain.Node{
		Name:      "containers",
		TypeStr:   "[]Object",
		FieldType: explain.FieldTypeList,
		Depth:     1,
		Expanded:  false,
	}
	label := NodeLabel(node)
	expected := "  ► containers <[]Object>"
	if label != expected {
		t.Errorf("expected %q, got %q", expected, label)
	}
}

func TestNodeLabel_RootScalar(t *testing.T) {
	node := &explain.Node{
		Name:      "apiVersion",
		TypeStr:   "string",
		FieldType: explain.FieldTypeScalar,
		Depth:     0,
	}
	label := NodeLabel(node)
	expected := "● apiVersion <string>"
	if label != expected {
		t.Errorf("expected %q, got %q", expected, label)
	}
}
