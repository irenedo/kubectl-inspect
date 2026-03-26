package explain

import "testing"

func TestAPIVersion_WithGroup(t *testing.T) {
	r := &ResourceInfo{Group: "apps", Version: "v1"}
	if got := r.APIVersion(); got != "apps/v1" {
		t.Errorf("expected 'apps/v1', got %q", got)
	}
}

func TestAPIVersion_WithoutGroup(t *testing.T) {
	r := &ResourceInfo{Group: "", Version: "v1"}
	if got := r.APIVersion(); got != "v1" {
		t.Errorf("expected 'v1', got %q", got)
	}
}

func TestIsExpandable(t *testing.T) {
	tests := []struct {
		name     string
		ft       FieldType
		expected bool
	}{
		{"scalar", FieldTypeScalar, false},
		{"object", FieldTypeObject, true},
		{"list", FieldTypeList, true},
		{"map", FieldTypeMap, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &Node{FieldType: tt.ft}
			if got := n.IsExpandable(); got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}
