package kubectl

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestBuildArgs_MinimalFlags(t *testing.T) {
	args := buildArgs("deployment", Flags{})
	expected := []string{"explain", "deployment"}
	assertArgsEqual(t, expected, args)
}

func TestBuildArgs_AllFlags(t *testing.T) {
	flags := Flags{
		Kubeconfig: "/path/to/kubeconfig",
		Context:    "my-context",
		APIVersion: "apps/v1",
	}
	args := buildArgs("deployment", flags)
	expected := []string{
		"explain", "deployment",
		"--kubeconfig", "/path/to/kubeconfig",
		"--context", "my-context",
		"--api-version", "apps/v1",
	}
	assertArgsEqual(t, expected, args)
}

func TestBuildArgs_FieldPath(t *testing.T) {
	args := buildArgs("deployment.spec.containers", Flags{})
	expected := []string{"explain", "deployment.spec.containers"}
	assertArgsEqual(t, expected, args)
}

func TestBuildArgs_PartialFlags(t *testing.T) {
	flags := Flags{
		Context: "prod-context",
	}
	args := buildArgs("pod", flags)
	expected := []string{"explain", "pod", "--context", "prod-context"}
	assertArgsEqual(t, expected, args)
}

func TestExplainRecursive_AddsRecursiveFlag(t *testing.T) {
	// We test that buildArgs + --recursive produces the correct args.
	// The actual execution requires kubectl, so we just verify arg construction.
	flags := Flags{Kubeconfig: "/kc"}
	args := buildArgs("deployment", flags)
	args = append(args, "--recursive")
	expected := []string{"explain", "deployment", "--kubeconfig", "/kc", "--recursive"}
	assertArgsEqual(t, expected, args)
}

func TestNotFoundError_Error(t *testing.T) {
	inner := fmt.Errorf("exec: not found")
	err := &NotFoundError{Err: inner}
	msg := err.Error()
	if !strings.Contains(msg, "kubectl not found") {
		t.Errorf("expected 'kubectl not found' in error, got %q", msg)
	}
	if !strings.Contains(msg, "exec: not found") {
		t.Errorf("expected inner error in message, got %q", msg)
	}
}

func TestNotFoundError_Unwrap(t *testing.T) {
	inner := fmt.Errorf("exec: not found")
	err := &NotFoundError{Err: inner}
	if !errors.Is(err, inner) {
		t.Error("Unwrap should return the inner error")
	}
}

func TestExecError_Error(t *testing.T) {
	err := &ExecError{Stderr: "something failed", ExitCode: 1}
	msg := err.Error()
	if !strings.Contains(msg, "code 1") {
		t.Errorf("expected exit code in error, got %q", msg)
	}
	if !strings.Contains(msg, "something failed") {
		t.Errorf("expected stderr in message, got %q", msg)
	}
}

func TestExecError_Error_RedactsURLs(t *testing.T) {
	err := &ExecError{Stderr: "dial tcp https://192.168.1.100:6443/connect: connection refused", ExitCode: 1}
	msg := err.Error()
	if strings.Contains(msg, "192.168.1.100") {
		t.Errorf("expected IP to be redacted, got %q", msg)
	}
	if !strings.Contains(msg, "[REDACTED]") {
		t.Errorf("expected redaction marker, got %q", msg)
	}
}

func TestExecError_Error_RedactsInternalDNS(t *testing.T) {
	err := &ExecError{Stderr: "dial tcp: lookup my-service.default.svc.cluster.local on 10.96.0.10:53", ExitCode: 1}
	msg := err.Error()
	if strings.Contains(msg, "my-service") {
		t.Errorf("expected internal DNS to be redacted, got %q", msg)
	}
	if !strings.Contains(msg, "[REDACTED]") {
		t.Errorf("expected redaction marker, got %q", msg)
	}
}

func TestExecError_Error_RedactsBearerTokens(t *testing.T) {
	//nolint:gosec // example JWT from RFC 7519, not a real credential
	token := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0"
	err := &ExecError{Stderr: "Unauthorized: Bearer " + token, ExitCode: 1}
	msg := err.Error()
	if strings.Contains(msg, token) {
		t.Errorf("expected bearer token to be redacted, got %q", msg)
	}
}

func TestExecError_Error_PreservesNonSensitive(t *testing.T) {
	err := &ExecError{Stderr: "the server doesn't have a resource type \"foo\"", ExitCode: 1}
	msg := err.Error()
	if !strings.Contains(msg, "the server doesn't have a resource type") {
		t.Errorf("expected non-sensitive text to be preserved, got %q", msg)
	}
}

func TestExecError_FriendlyMessage_RedactsURLs(t *testing.T) {
	err := &ExecError{Stderr: "dial tcp 10.0.0.5:6443: connection refused", ExitCode: 1}
	msg := err.FriendlyMessage()
	if strings.Contains(msg, "10.0.0.5") {
		t.Errorf("expected IP to be redacted in friendly message, got %q", msg)
	}
	if !strings.Contains(msg, "[REDACTED]") {
		t.Errorf("expected redaction marker, got %q", msg)
	}
}

func TestExecError_FriendlyMessage_UnknownResource(t *testing.T) {
	err := &ExecError{Stderr: "the server doesn't have a resource type \"foo\"", ExitCode: 1}
	msg := err.FriendlyMessage()
	if !strings.Contains(msg, "Unknown resource type") {
		t.Errorf("expected friendly unknown resource message, got %q", msg)
	}
}

func TestExecError_FriendlyMessage_ConnectionRefused(t *testing.T) {
	err := &ExecError{Stderr: "dial tcp 127.0.0.1:6443: connection refused", ExitCode: 1}
	msg := err.FriendlyMessage()
	if !strings.Contains(msg, "Cannot connect") {
		t.Errorf("expected friendly connection refused message, got %q", msg)
	}
}

func TestExecError_FriendlyMessage_UnableToConnect(t *testing.T) {
	err := &ExecError{Stderr: "unable to connect to the server", ExitCode: 1}
	msg := err.FriendlyMessage()
	if !strings.Contains(msg, "Cannot connect") {
		t.Errorf("expected friendly unable to connect message, got %q", msg)
	}
}

func TestExecError_FriendlyMessage_Default(t *testing.T) {
	err := &ExecError{Stderr: "some other error", ExitCode: 1}
	msg := err.FriendlyMessage()
	if msg != "some other error" {
		t.Errorf("expected raw stderr as default, got %q", msg)
	}
}

func TestExecError_FriendlyMessage_Default_Redacts(t *testing.T) {
	err := &ExecError{Stderr: "failed to get response from https://api.internal.example.com:443", ExitCode: 1}
	msg := err.FriendlyMessage()
	if strings.Contains(msg, "api.internal.example.com") {
		t.Errorf("expected URL to be redacted in default friendly message, got %q", msg)
	}
	if !strings.Contains(msg, "[REDACTED]") {
		t.Errorf("expected redaction marker, got %q", msg)
	}
}

func assertArgsEqual(t *testing.T, expected, actual []string) {
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
