package explain

import (
	"context"
	"fmt"
	"testing"

	"github.com/irenedo/kubectl-inspect/pkg/kubectl"
	"github.com/irenedo/kubectl-inspect/pkg/testutil"
)

func TestFetchDetail_TopLevel(t *testing.T) {
	mock := &testutil.MockExecutor{FieldOutput: "KIND: Deployment\n"}
	fetcher := NewFetcher(mock, "deployment", kubectl.Flags{})

	result := fetcher.FetchDetail(context.Background(), "")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if mock.LastFieldPath != "deployment" {
		t.Errorf("expected field path 'deployment', got %q", mock.LastFieldPath)
	}
	if result.RawOutput != "KIND: Deployment\n" {
		t.Errorf("unexpected output: %q", result.RawOutput)
	}
}

func TestFetchDetail_NestedPath(t *testing.T) {
	mock := &testutil.MockExecutor{FieldOutput: "FIELD: containers\n"}
	fetcher := NewFetcher(mock, "deployment", kubectl.Flags{})

	result := fetcher.FetchDetail(context.Background(), "spec.containers")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if mock.LastFieldPath != "deployment.spec.containers" {
		t.Errorf("expected 'deployment.spec.containers', got %q", mock.LastFieldPath)
	}
	if result.RawOutput != "FIELD: containers\n" {
		t.Errorf("unexpected output: %q", result.RawOutput)
	}
}

func TestFetchDetail_Error(t *testing.T) {
	mock := &testutil.MockExecutor{FieldErr: fmt.Errorf("connection refused")}
	fetcher := NewFetcher(mock, "deployment", kubectl.Flags{})

	result := fetcher.FetchDetail(context.Background(), "spec")
	if result.Err == nil {
		t.Error("expected error")
	}
}

func TestFetchDetail_FlagsPassthrough(t *testing.T) {
	mock := &testutil.MockExecutor{FieldOutput: "ok"}
	flags := kubectl.Flags{
		Kubeconfig: "/my/kubeconfig",
		Context:    "prod",
		APIVersion: "apps/v1",
	}
	fetcher := NewFetcher(mock, "deployment", flags)

	fetcher.FetchDetail(context.Background(), "spec")
	if mock.LastFlags.Kubeconfig != "/my/kubeconfig" {
		t.Errorf("kubeconfig not passed through: %q", mock.LastFlags.Kubeconfig)
	}
	if mock.LastFlags.Context != "prod" {
		t.Errorf("context not passed through: %q", mock.LastFlags.Context)
	}
	if mock.LastFlags.APIVersion != "apps/v1" {
		t.Errorf("api-version not passed through: %q", mock.LastFlags.APIVersion)
	}
}
