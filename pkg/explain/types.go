package explain

// FieldType classifies the type of a Kubernetes resource field.
type FieldType int

// FieldType constants define the possible classifications for resource fields.
const (
	FieldTypeScalar FieldType = iota
	FieldTypeObject
	FieldTypeList
	FieldTypeMap
)

// Node represents a single field in the resource tree.
type Node struct {
	Name      string
	TypeStr   string
	FieldType FieldType
	Depth     int
	Path      string
	Children  []*Node
	Expanded  bool
	Parent    *Node
}

// IsExpandable returns true if the node can have children.
func (n *Node) IsExpandable() bool {
	return n.FieldType == FieldTypeObject || n.FieldType == FieldTypeList || n.FieldType == FieldTypeMap
}

// ResourceInfo holds the parsed result of kubectl explain --recursive.
type ResourceInfo struct {
	Kind        string
	Group       string
	Version     string
	Description string
	Fields      []*Node
}

// APIVersion returns the full API group/version string (e.g. "apps/v1").
func (r *ResourceInfo) APIVersion() string {
	if r.Group == "" {
		return r.Version
	}
	return r.Group + "/" + r.Version
}

// DetailResult holds the result of a single kubectl explain call.
type DetailResult struct {
	RawOutput string
	Err       error
}
