package explain

import (
	"fmt"
	"slices"
	"strings"
)

// headerResult holds the parsed header information from kubectl explain output.
type headerResult struct {
	kind        string
	group       string
	version     string
	description string
	fieldsIdx   int
}

// ParseRecursive parses the output of `kubectl explain <resource> --recursive`
// into a ResourceInfo tree.
func ParseRecursive(output string) (*ResourceInfo, error) {
	lines := strings.Split(output, "\n")

	hdr, err := parseHeader(lines)
	if err != nil {
		return nil, err
	}

	fieldLines := lines[hdr.fieldsIdx:]
	fields := buildTree(fieldLines)

	// Post-process: classify nodes that got children as expandable,
	// regardless of their type name.
	postClassify(fields)

	return &ResourceInfo{
		Kind:        hdr.kind,
		Group:       hdr.group,
		Version:     hdr.version,
		Description: hdr.description,
		Fields:      fields,
	}, nil
}

func parseHeader(lines []string) (headerResult, error) {
	var hdr headerResult
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "KIND:"):
			hdr.kind = strings.TrimSpace(strings.TrimPrefix(trimmed, "KIND:"))
		case strings.HasPrefix(trimmed, "VERSION:"):
			hdr.version = strings.TrimSpace(strings.TrimPrefix(trimmed, "VERSION:"))
		case strings.HasPrefix(trimmed, "GROUP:"):
			hdr.group = strings.TrimSpace(strings.TrimPrefix(trimmed, "GROUP:"))
		case trimmed == "DESCRIPTION:":
			var descLines []string
			for j := i + 1; j < len(lines); j++ {
				t := strings.TrimSpace(lines[j])
				if t == "FIELDS:" || strings.HasPrefix(t, "FIELD:") {
					break
				}
				if t != "" {
					descLines = append(descLines, t)
				}
			}
			hdr.description = strings.Join(descLines, " ")
		case trimmed == "FIELDS:":
			hdr.fieldsIdx = i + 1
			return hdr, nil
		}
	}

	if hdr.fieldsIdx == 0 {
		return headerResult{}, fmt.Errorf("no FIELDS: section found in output")
	}
	return hdr, nil
}

// parseFieldLine parses a single field line like "  image\t<string>".
// Returns the field name, raw type string (without angle brackets), and the
// number of leading spaces (raw indent). Callers determine depth from this.
func parseFieldLine(line string) (name, typeStr string, leadingSpaces int, err error) {
	if strings.TrimSpace(line) == "" {
		return "", "", 0, fmt.Errorf("empty line")
	}

	stripped := strings.TrimRight(line, " \t\r\n")
	leadingSpaces = len(stripped) - len(strings.TrimLeft(stripped, " "))

	trimmed := strings.TrimSpace(stripped)

	// Field lines always contain a tab separating name from type.
	// Lines without a tab (like "enum: Recreate, RollingUpdate") are metadata, not fields.
	if !strings.Contains(trimmed, "\t") {
		return "", "", 0, fmt.Errorf("not a field line (no tab separator): %q", line)
	}

	// Split by tab: name\t<type> or name\t<type> -required-
	parts := strings.SplitN(trimmed, "\t", 2)
	name = strings.TrimSpace(parts[0])
	if name == "" {
		return "", "", 0, fmt.Errorf("empty field name in line: %q", line)
	}

	if len(parts) > 1 {
		raw := strings.TrimSpace(parts[1])
		// Remove -required- suffix (may be outside or inside angle brackets)
		raw = strings.TrimSuffix(raw, " -required-")
		raw = strings.TrimSuffix(raw, "-required-")
		raw = strings.TrimSpace(raw)
		// Remove angle brackets
		raw = strings.TrimPrefix(raw, "<")
		raw = strings.TrimSuffix(raw, ">")
		// Also handle -required inside brackets: <string> -required-  already handled above,
		// but also <string -required> which shouldn't happen but be safe
		raw = strings.TrimSuffix(raw, " -required")
		typeStr = strings.TrimSpace(raw)
	}

	return name, typeStr, leadingSpaces, nil
}

// classifyType returns the initial FieldType for a given type string.
// This is refined later by postClassify based on whether children exist.
func classifyType(typeStr string) FieldType {
	lower := strings.ToLower(typeStr)
	switch {
	case typeStr == "Object" || lower == "object":
		return FieldTypeObject
	case strings.HasPrefix(typeStr, "[]"):
		// []Object, []string, []NamedType — all start with []
		// If the inner type is not a scalar, it's a list of objects
		inner := typeStr[2:]
		if isScalarType(inner) {
			return FieldTypeScalar
		}
		return FieldTypeList
	case strings.HasPrefix(typeStr, "map["):
		return FieldTypeMap
	default:
		// Could be a named type like "ObjectMeta", "DeploymentSpec", "PodSpec"
		// We'll rely on postClassify to detect if it has children
		if isScalarType(typeStr) {
			return FieldTypeScalar
		}
		// Assume it might be an object; postClassify will fix if no children
		return FieldTypeObject
	}
}

// isScalarType returns true for types that are definitely scalar (no children).
func isScalarType(typeStr string) bool {
	switch strings.ToLower(typeStr) {
	case "string", "integer", "int32", "int64", "boolean", "bool",
		"number", "float", "float64", "double", "time", "intorstring",
		"quantity", "byte", "":
		return true
	}
	return false
}

// postClassify walks the tree and fixes classification:
// - Nodes with children are expandable (Object or List)
// - Nodes without children that were tentatively marked Object are demoted to Scalar
func postClassify(nodes []*Node) {
	for _, node := range nodes {
		if len(node.Children) > 0 {
			// Has children — ensure it's expandable
			promoteToExpandable(node)
			postClassify(node.Children)
		} else if node.IsExpandable() {
			// No children — demote any expandable type to Scalar
			// (e.g., map[string]Quantity with no sub-fields, or named types like ResourceList)
			node.FieldType = FieldTypeScalar
		}
	}
}

// promoteToExpandable upgrades a scalar node to its correct expandable type.
func promoteToExpandable(node *Node) {
	if node.FieldType != FieldTypeScalar {
		return
	}
	if strings.HasPrefix(node.TypeStr, "[]") {
		node.FieldType = FieldTypeList
	} else {
		node.FieldType = FieldTypeObject
	}
}

// buildTree converts parsed field lines into a tree of Nodes.
func buildTree(lines []string) []*Node {
	var roots []*Node
	var stack []*Node

	// Detect the indentation unit from the first two distinct indent levels.
	indentUnit := detectIndentUnit(lines)

	// Detect base indentation from the first non-empty field line.
	baseIndent := -1
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		_, _, spaces, err := parseFieldLine(line)
		if err != nil {
			continue
		}
		baseIndent = spaces
		break
	}
	if baseIndent < 0 {
		return roots
	}

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		name, typeStr, spaces, err := parseFieldLine(line)
		if err != nil {
			continue
		}

		depth := 0
		if indentUnit > 0 {
			depth = (spaces - baseIndent) / indentUnit
		}

		ft := classifyType(typeStr)
		node := &Node{
			Name:      name,
			TypeStr:   typeStr,
			FieldType: ft,
			Depth:     depth,
		}

		if depth == 0 {
			node.Path = name
			roots = append(roots, node)
			stack = []*Node{node}
		} else {
			roots, stack = placeChildNode(node, name, depth, roots, stack)
		}
	}

	return roots
}

// placeChildNode attaches a non-root node to its parent in the stack and updates the stack.
func placeChildNode(node *Node, name string, depth int, roots, stack []*Node) (updatedRoots, updatedStack []*Node) {
	for len(stack) > depth {
		stack = stack[:len(stack)-1]
	}

	if len(stack) > 0 {
		parent := stack[len(stack)-1]
		node.Parent = parent
		node.Path = parent.Path + "." + name
		parent.Children = append(parent.Children, node)
	} else {
		node.Path = name
		roots = append(roots, node)
	}

	if len(stack) <= depth {
		stack = append(stack, node)
	} else {
		stack[depth] = node
	}
	return roots, stack
}

// detectIndentUnit finds the number of spaces per indent level by looking at
// the first two distinct indentation levels in the field lines.
func detectIndentUnit(lines []string) int {
	indents := map[int]bool{}
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		_, _, spaces, err := parseFieldLine(line)
		if err != nil {
			continue
		}
		indents[spaces] = true
		if len(indents) >= 3 {
			break
		}
	}

	// Find the smallest non-zero indent difference
	vals := make([]int, 0, len(indents))
	for v := range indents {
		vals = append(vals, v)
	}
	if len(vals) < 2 {
		return 2 // default
	}

	slices.Sort(vals)

	minDiff := vals[1] - vals[0]
	for i := 2; i < len(vals); i++ {
		diff := vals[i] - vals[i-1]
		if diff < minDiff {
			minDiff = diff
		}
	}

	if minDiff <= 0 {
		return 2
	}
	return minDiff
}
