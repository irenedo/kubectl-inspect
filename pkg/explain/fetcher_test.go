package explain

import (
	"context"
	"fmt"
	"testing"

	"github.com/irenedo/kubectl-inspect/pkg/kubectl"
)

type mockExecutor struct {
	recursiveOutput string
	recursiveErr    error
	fieldOutput     string
	fieldErr        error
	lastFieldPath   string
	lastFlags       kubectl.Flags
}

func (m *mockExecutor) ExplainRecursive(_ context.Context, _ string, _ kubectl.Flags) (string, error) {
	return m.recursiveOutput, m.recursiveErr
}

func (m *mockExecutor) ExplainField(_ context.Context, fieldPath string, flags kubectl.Flags) (string, error) {
	m.lastFieldPath = fieldPath
	m.lastFlags = flags
	return m.fieldOutput, m.fieldErr
}

func TestFetchDetail_TopLevel(t *testing.T) {
	mock := &mockExecutor{fieldOutput: "KIND: Deployment\n"}
	fetcher := NewFetcher(mock, "deployment", kubectl.Flags{})

	result := fetcher.FetchDetail(context.Background(), "")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if mock.lastFieldPath != "deployment" {
		t.Errorf("expected field path 'deployment', got %q", mock.lastFieldPath)
	}
	if result.RawOutput != "KIND: Deployment\n" {
		t.Errorf("unexpected output: %q", result.RawOutput)
	}
}

func TestFetchDetail_NestedPath(t *testing.T) {
	mock := &mockExecutor{fieldOutput: "FIELD: containers\n"}
	fetcher := NewFetcher(mock, "deployment", kubectl.Flags{})

	result := fetcher.FetchDetail(context.Background(), "spec.containers")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if mock.lastFieldPath != "deployment.spec.containers" {
		t.Errorf("expected 'deployment.spec.containers', got %q", mock.lastFieldPath)
	}
	if result.RawOutput != "FIELD: containers\n" {
		t.Errorf("unexpected output: %q", result.RawOutput)
	}
}

func TestFetchDetail_Error(t *testing.T) {
	mock := &mockExecutor{fieldErr: fmt.Errorf("connection refused")}
	fetcher := NewFetcher(mock, "deployment", kubectl.Flags{})

	result := fetcher.FetchDetail(context.Background(), "spec")
	if result.Err == nil {
		t.Error("expected error")
	}
}

func TestFetchDetail_FlagsPassthrough(t *testing.T) {
	mock := &mockExecutor{fieldOutput: "ok"}
	flags := kubectl.Flags{
		Kubeconfig: "/my/kubeconfig",
		Context:    "prod",
		APIVersion: "apps/v1",
	}
	fetcher := NewFetcher(mock, "deployment", flags)

	fetcher.FetchDetail(context.Background(), "spec")
	if mock.lastFlags.Kubeconfig != "/my/kubeconfig" {
		t.Errorf("kubeconfig not passed through: %q", mock.lastFlags.Kubeconfig)
	}
	if mock.lastFlags.Context != "prod" {
		t.Errorf("context not passed through: %q", mock.lastFlags.Context)
	}
	if mock.lastFlags.APIVersion != "apps/v1" {
		t.Errorf("api-version not passed through: %q", mock.lastFlags.APIVersion)
	}
}
