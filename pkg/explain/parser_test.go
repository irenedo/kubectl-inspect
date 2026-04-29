package explain

import (
	"testing"

	"github.com/irenedo/kubectl-inspect/pkg/testutil"
)

func TestParseHeader(t *testing.T) {
	input := testutil.ReadFixture(t, "crd_recursive.txt")
	lines := testutil.SplitLines(input)
	hdr, err := parseHeader(lines)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hdr.kind != "Certificate" {
		t.Errorf("kind: expected Certificate, got %q", hdr.kind)
	}
	if hdr.group != "cert-manager.io" {
		t.Errorf("group: expected cert-manager.io, got %q", hdr.group)
	}
	if hdr.version != "v1" {
		t.Errorf("version: expected v1, got %q", hdr.version)
	}
	if hdr.description == "" {
		t.Error("description should not be empty")
	}
	if hdr.fieldsIdx == 0 {
		t.Error("fieldsIdx should be > 0")
	}
}

func TestParseHeader_WithGroup(t *testing.T) {
	input := testutil.ReadFixture(t, "deployment_recursive.txt")
	lines := testutil.SplitLines(input)
	hdr, err := parseHeader(lines)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hdr.kind != "Deployment" {
		t.Errorf("kind: expected Deployment, got %q", hdr.kind)
	}
	if hdr.group != "apps" {
		t.Errorf("group: expected apps, got %q", hdr.group)
	}
	if hdr.version != "v1" {
		t.Errorf("version: expected v1, got %q", hdr.version)
	}
	if hdr.fieldsIdx == 0 {
		t.Error("fieldsIdx should be > 0")
	}
}

func TestParseFieldLine_Simple(t *testing.T) {
	name, typeStr, spaces, err := parseFieldLine("  image\t<string>")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "image" {
		t.Errorf("name: expected image, got %q", name)
	}
	if typeStr != "string" {
		t.Errorf("typeStr: expected string, got %q", typeStr)
	}
	if spaces != 2 {
		t.Errorf("spaces: expected 2, got %d", spaces)
	}
}

func TestParseFieldLine_NestedType(t *testing.T) {
	name, typeStr, spaces, err := parseFieldLine("      matchLabels\t<map[string]string>")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "matchLabels" {
		t.Errorf("name: expected matchLabels, got %q", name)
	}
	if typeStr != "map[string]string" {
		t.Errorf("typeStr: expected map[string]string, got %q", typeStr)
	}
	if spaces != 6 {
		t.Errorf("spaces: expected 6, got %d", spaces)
	}
}

func TestParseFieldLine_Required(t *testing.T) {
	name, typeStr, _, err := parseFieldLine("    selector\t<LabelSelector> -required-")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "selector" {
		t.Errorf("name: expected selector, got %q", name)
	}
	if typeStr != "LabelSelector" {
		t.Errorf("typeStr: expected LabelSelector, got %q", typeStr)
	}
}

func TestParseFieldLine_EnumLine(t *testing.T) {
	_, _, _, err := parseFieldLine("      enum: Recreate, RollingUpdate")
	if err == nil {
		t.Error("expected error for enum metadata line (no tab separator)")
	}
}

func TestParseFieldLine_Empty(t *testing.T) {
	_, _, _, err := parseFieldLine("    ")
	if err == nil {
		t.Error("expected error for empty line")
	}
}

func TestClassifyType(t *testing.T) {
	tests := []struct {
		typeStr  string
		expected FieldType
	}{
		{"string", FieldTypeScalar},
		{"integer", FieldTypeScalar},
		{"boolean", FieldTypeScalar},
		{"IntOrString", FieldTypeScalar},
		{"Object", FieldTypeObject},
		{"DeploymentSpec", FieldTypeObject},
		{"[]LabelSelectorRequirement", FieldTypeList},
		{"[]Container", FieldTypeList},
	}
	for _, tt := range tests {
		t.Run(tt.typeStr, func(t *testing.T) {
			got := classifyType(tt.typeStr)
			if got != tt.expected {
				t.Errorf("classifyType(%q): expected %d, got %d", tt.typeStr, tt.expected, got)
			}
		})
	}
}

func TestBuildTree_Simple(t *testing.T) {
	input := testutil.ReadFixture(t, "crd_recursive.txt")
	info, err := ParseRecursive(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(info.Fields) == 0 {
		t.Fatal("expected fields, got none")
	}

	// Check apiVersion is a root scalar
	found := findNode(info.Fields, "apiVersion")
	if found == nil {
		t.Fatal("apiVersion not found")
	}
	if found.FieldType != FieldTypeScalar {
		t.Errorf("apiVersion should be scalar, got %d", found.FieldType)
	}
	if found.Path != "apiVersion" {
		t.Errorf("apiVersion path: expected apiVersion, got %q", found.Path)
	}

	// Check metadata is an Object with children
	metadata := findNode(info.Fields, "metadata")
	if metadata == nil {
		t.Fatal("metadata not found")
	}
	if !metadata.IsExpandable() {
		t.Errorf("metadata should be expandable, got FieldType %d", metadata.FieldType)
	}
	if len(metadata.Children) == 0 {
		t.Error("metadata should have children")
	}

	// Check nested field
	annotations := findNode(metadata.Children, "annotations")
	if annotations == nil {
		t.Fatal("metadata.annotations not found")
	}
	if annotations.Path != "metadata.annotations" {
		t.Errorf("annotations path: expected metadata.annotations, got %q", annotations.Path)
	}
}

func TestBuildTree_Deployment(t *testing.T) {
	input := testutil.ReadFixture(t, "deployment_recursive.txt")
	info, err := ParseRecursive(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.Kind != "Deployment" {
		t.Errorf("kind: expected Deployment, got %q", info.Kind)
	}
	if info.APIVersion() != "apps/v1" {
		t.Errorf("apiVersion: expected apps/v1, got %q", info.APIVersion())
	}

	// Check spec exists and has children
	spec := findNode(info.Fields, "spec")
	if spec == nil {
		t.Fatal("spec not found")
	}
	if !spec.IsExpandable() {
		t.Errorf("spec should be expandable, got FieldType %d", spec.FieldType)
	}
	if len(spec.Children) == 0 {
		t.Error("spec should have children")
	}

	// Check spec.replicas is scalar
	replicas := findNode(spec.Children, "replicas")
	if replicas == nil {
		t.Fatal("spec.replicas not found")
	}
	if replicas.FieldType != FieldTypeScalar {
		t.Errorf("replicas should be scalar, got %d", replicas.FieldType)
	}
	if replicas.Path != "spec.replicas" {
		t.Errorf("replicas path: expected spec.replicas, got %q", replicas.Path)
	}

	// Check deeply nested path: spec.selector.matchExpressions
	selector := findNode(spec.Children, "selector")
	if selector == nil {
		t.Fatal("spec.selector not found")
	}
	if !selector.IsExpandable() {
		t.Errorf("selector should be expandable, got FieldType %d", selector.FieldType)
	}
	matchExpr := findNode(selector.Children, "matchExpressions")
	if matchExpr == nil {
		t.Fatal("spec.selector.matchExpressions not found")
	}
	if matchExpr.FieldType != FieldTypeList {
		t.Errorf("matchExpressions should be List, got %d", matchExpr.FieldType)
	}
	if matchExpr.Path != "spec.selector.matchExpressions" {
		t.Errorf("path: expected spec.selector.matchExpressions, got %q", matchExpr.Path)
	}

	// Check containers is a list with children
	template := findNode(spec.Children, "template")
	if template == nil {
		t.Fatal("spec.template not found")
	}
	podSpec := findNode(template.Children, "spec")
	if podSpec == nil {
		t.Fatal("spec.template.spec not found")
	}
	containers := findNode(podSpec.Children, "containers")
	if containers == nil {
		t.Fatal("spec.template.spec.containers not found")
	}
	if !containers.IsExpandable() {
		t.Errorf("containers should be expandable, got FieldType %d", containers.FieldType)
	}
	if len(containers.Children) == 0 {
		t.Error("containers should have children")
	}

	// Check status exists
	status := findNode(info.Fields, "status")
	if status == nil {
		t.Fatal("status not found")
	}
	if len(status.Children) == 0 {
		t.Error("status should have children")
	}
}

func TestBuildTree_CRD(t *testing.T) {
	input := testutil.ReadFixture(t, "crd_recursive.txt")
	info, err := ParseRecursive(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.Kind != "Certificate" {
		t.Errorf("kind: expected Certificate, got %q", info.Kind)
	}

	// Check spec.issuerRef.name exists
	spec := findNode(info.Fields, "spec")
	if spec == nil {
		t.Fatal("spec not found")
	}
	issuerRef := findNode(spec.Children, "issuerRef")
	if issuerRef == nil {
		t.Fatal("spec.issuerRef not found")
	}
	nameField := findNode(issuerRef.Children, "name")
	if nameField == nil {
		t.Fatal("spec.issuerRef.name not found")
	}
	if nameField.Path != "spec.issuerRef.name" {
		t.Errorf("path: expected spec.issuerRef.name, got %q", nameField.Path)
	}
}

func TestBuildTree_RealOutput(t *testing.T) {
	input := testutil.ReadFixture(t, "deployment_recursive.txt")
	info, err := ParseRecursive(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify named types like ObjectMeta are expandable
	metadata := findNode(info.Fields, "metadata")
	if metadata == nil {
		t.Fatal("metadata not found")
	}
	if metadata.TypeStr != "ObjectMeta" {
		t.Errorf("metadata type: expected ObjectMeta, got %q", metadata.TypeStr)
	}
	if !metadata.IsExpandable() {
		t.Error("metadata (ObjectMeta) should be expandable because it has children")
	}

	// Verify -required- is stripped
	spec := findNode(info.Fields, "spec")
	if spec == nil {
		t.Fatal("spec not found")
	}
	selector := findNode(spec.Children, "selector")
	if selector == nil {
		t.Fatal("selector not found")
	}
	if selector.TypeStr != "LabelSelector" {
		t.Errorf("selector type: expected LabelSelector, got %q", selector.TypeStr)
	}
}

func TestBuildTree_RealKubectlOutput(t *testing.T) {
	input := testutil.ReadFixture(t, "real_deployment_recursive.txt")
	info, err := ParseRecursive(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.Kind != "Deployment" {
		t.Errorf("kind: expected Deployment, got %q", info.Kind)
	}

	// spec must be expandable
	spec := findNode(info.Fields, "spec")
	if spec == nil {
		t.Fatal("spec not found")
	}
	if !spec.IsExpandable() {
		t.Errorf("spec should be expandable (type: %q, fieldType: %d)", spec.TypeStr, spec.FieldType)
	}
	if len(spec.Children) == 0 {
		t.Fatal("spec should have children")
	}

	// metadata must be expandable (type is ObjectMeta, not Object)
	metadata := findNode(info.Fields, "metadata")
	if metadata == nil {
		t.Fatal("metadata not found")
	}
	if !metadata.IsExpandable() {
		t.Errorf("metadata should be expandable (type: %q, fieldType: %d)", metadata.TypeStr, metadata.FieldType)
	}

	// Deep path: spec.template.spec.containers must be expandable
	template := findNode(spec.Children, "template")
	if template == nil {
		t.Fatal("spec.template not found")
	}
	podSpec := findNode(template.Children, "spec")
	if podSpec == nil {
		t.Fatal("spec.template.spec not found")
	}
	containers := findNode(podSpec.Children, "containers")
	if containers == nil {
		t.Fatal("spec.template.spec.containers not found")
	}
	if !containers.IsExpandable() {
		t.Errorf("containers should be expandable (type: %q, fieldType: %d)", containers.TypeStr, containers.FieldType)
	}
	if len(containers.Children) == 0 {
		t.Error("containers should have children")
	}

	// Total root fields should be reasonable
	if len(info.Fields) < 3 {
		t.Errorf("expected at least 3 root fields, got %d", len(info.Fields))
	}
}

func TestParseRecursive_ErrorOutput(t *testing.T) {
	input := testutil.ReadFixture(t, "error_output.txt")
	_, err := ParseRecursive(input)
	if err == nil {
		t.Error("expected error for error output")
	}
}

func TestParseRecursive_EmptyInput(t *testing.T) {
	_, err := ParseRecursive("")
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestDetectIndentUnit(t *testing.T) {
	// 2-space indent
	lines2 := []string{"  apiVersion\t<string>", "  metadata\t<ObjectMeta>", "    name\t<string>"}
	if u := detectIndentUnit(lines2); u != 2 {
		t.Errorf("expected indent unit 2, got %d", u)
	}

	// 3-space indent
	lines3 := []string{"   apiVersion\t<string>", "   metadata\t<Object>", "      name\t<string>"}
	if u := detectIndentUnit(lines3); u != 3 {
		t.Errorf("expected indent unit 3, got %d", u)
	}

	// Single indent level (defaults to the smallest indent found)
	lines1 := []string{"  apiVersion\t<string>"}
	if u := detectIndentUnit(lines1); u != 2 {
		t.Errorf("expected indent unit 2 for single level, got %d", u)
	}

	// Mixed indent levels (detect smallest consistent unit)
	linesMixed := []string{"   api\t<string>", "   spec\t<Object>", "      name\t<string>"}
	if u := detectIndentUnit(linesMixed); u != 3 {
		t.Errorf("expected indent unit 3 for mixed levels, got %d", u)
	}
}

// TestPlaceChildNode_ListWithChildren verifies that list fields with children
// are properly placed in the tree (the FieldTypeList path in placeChildNode).
func TestPlaceChildNode_ListWithChildren(t *testing.T) {
	input := testutil.ReadFixture(t, "list_nested.txt")
	info, err := ParseRecursive(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// items is a root-level []Item field with children
	items := findNode(info.Fields, "items")
	if items == nil {
		t.Fatal("items not found")
	}
	if !items.IsExpandable() {
		t.Errorf("items should be expandable (list with children), got %d", items.FieldType)
	}
	if items.FieldType != FieldTypeList {
		t.Errorf("items should be FieldTypeList, got %d", items.FieldType)
	}
	if len(items.Children) == 0 {
		t.Error("items should have children")
	}

	// Check nested field inside items
	id := findNode(items.Children, "id")
	if id == nil {
		t.Fatal("items.id not found")
	}
	if id.Path != "items.id" {
		t.Errorf("items.id path: expected items.id, got %q", id.Path)
	}

	// Check deeply nested field inside items.config
	config := findNode(items.Children, "config")
	if config == nil {
		t.Fatal("items.config not found")
	}
	enabled := findNode(config.Children, "enabled")
	if enabled == nil {
		t.Fatal("items.config.enabled not found")
	}
	if enabled.Path != "items.config.enabled" {
		t.Errorf("path: expected items.config.enabled, got %q", enabled.Path)
	}
}

// TestPromoteToExpandable_EarlyReturn verifies the promoteToExpandable early return
// when FieldType is not scalar.
func TestPromoteToExpandable_EarlyReturn(t *testing.T) {
	node := &Node{
		Name:      "test",
		TypeStr:   "AlreadyObject",
		FieldType: FieldTypeObject,
		Depth:     0,
		Children:  []*Node{{Name: "child"}},
	}

	// promoteToExpandable should return early without modifying the node
	promoteToExpandable(node)
	if node.FieldType != FieldTypeObject {
		t.Errorf("expected FieldTypeObject, got %d (should not change)", node.FieldType)
	}
}

// TestPromoteToExpandable_List verifies promotion of []type to FieldTypeList.
func TestPromoteToExpandable_List(t *testing.T) {
	node := &Node{
		Name:      "items",
		TypeStr:   "[]Item",
		FieldType: FieldTypeScalar,
		Depth:     0,
	}

	promoteToExpandable(node)
	if node.FieldType != FieldTypeList {
		t.Errorf("expected FieldTypeList, got %d", node.FieldType)
	}
}

// TestPromoteToExpandable_Object verifies promotion of named type to FieldTypeObject.
func TestPromoteToExpandable_Object(t *testing.T) {
	node := &Node{
		Name:      "config",
		TypeStr:   "Config",
		FieldType: FieldTypeScalar,
		Depth:     0,
	}

	promoteToExpandable(node)
	if node.FieldType != FieldTypeObject {
		t.Errorf("expected FieldTypeObject, got %d", node.FieldType)
	}
}

func TestDetectIndentUnit_SingleIndent(t *testing.T) {
	// All lines at same indent — should return default of 2
	lines := []string{
		"    field1\t<string>",
		"    field2\t<string>",
	}
	unit := detectIndentUnit(lines)
	if unit != 2 {
		t.Errorf("expected default indent 2 for single indent level, got %d", unit)
	}
}

func TestDetectIndentUnit_ZeroOrInvalidLines(t *testing.T) {
	lines := []string{"", "   ", "no indent info here"}
	unit := detectIndentUnit(lines)
	if unit != 2 {
		t.Errorf("expected default indent 2 for invalid lines, got %d", unit)
	}
}

func TestDetectIndentUnit_ThreeLevels(t *testing.T) {
	lines := []string{
		"    field1\t<string>",
		"        nested\t<string>",
		"            deeper\t<string>",
	}
	unit := detectIndentUnit(lines)
	if unit != 4 {
		t.Errorf("expected indent 4 for 4-space levels, got %d", unit)
	}
}

func TestPlaceChildNode_RootLevel(t *testing.T) {
	node := &Node{Name: "foo", Depth: 0}
	roots, _ := placeChildNode(node, "foo", 0, nil, nil)
	if len(roots) != 1 || roots[0] != node {
		t.Error("root-level node should be appended to roots")
	}
	if node.Path != "foo" {
		t.Errorf("expected path 'foo', got %q", node.Path)
	}
}

func TestPlaceChildNode_DeepStackReplace(t *testing.T) {
	// Build a stack: [A, B, C] at depths 0, 1, 2
	// Now place a child of A (depth 0) — should replace stack[0]
	A := &Node{Name: "A", Depth: 0}
	B := &Node{Name: "B", Depth: 1}
	C := &Node{Name: "C", Depth: 2}

	var roots, stack []*Node
	roots, stack = placeChildNode(A, "A", 0, roots, stack)
	roots, stack = placeChildNode(B, "B", 1, roots, stack)
	roots, stack = placeChildNode(C, "C", 2, roots, stack)

	// D is a sibling of A (depth 0) — should trigger the stack[depth] = node branch
	D := &Node{Name: "D", Depth: 0}
	roots, stack = placeChildNode(D, "D", 0, roots, stack)

	if len(roots) != 2 {
		t.Errorf("expected 2 root nodes, got %d", len(roots))
	}
	if len(stack) != 1 && stack[0] != D {
		t.Errorf("expected stack top to be D, got %v", stack)
	}
}

// helpers

func findNode(nodes []*Node, name string) *Node {
	for _, n := range nodes {
		if n.Name == name {
			return n
		}
	}
	return nil
}
