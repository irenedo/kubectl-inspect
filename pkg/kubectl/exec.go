package kubectl

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Flags holds kubectl passthrough flags.
type Flags struct {
	Kubeconfig string
	Context    string
	APIVersion string
}

// Executor defines the interface for running kubectl explain commands.
type Executor interface {
	ExplainRecursive(ctx context.Context, resource string, flags Flags) (string, error)
	ExplainField(ctx context.Context, fieldPath string, flags Flags) (string, error)
}

// NotFoundError indicates kubectl is not in PATH.
type NotFoundError struct {
	Err error
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("kubectl not found in PATH: %v", e.Err)
}

// Unwrap returns the underlying error.
func (e *NotFoundError) Unwrap() error {
	return e.Err
}

// ExecError wraps a failed kubectl execution.
type ExecError struct {
	Stderr   string
	ExitCode int
}

func (e *ExecError) Error() string {
	return fmt.Sprintf("kubectl exited with code %d: %s", e.ExitCode, e.Stderr)
}

// FriendlyMessage returns a user-facing message for known error patterns.
func (e *ExecError) FriendlyMessage() string {
	stderr := strings.ToLower(e.Stderr)
	switch {
	case strings.Contains(stderr, "the server doesn't have a resource type"):
		return fmt.Sprintf("Unknown resource type. Run `kubectl api-resources` to see available types.\n\n%s", e.Stderr)
	case strings.Contains(stderr, "connection refused"):
		return fmt.Sprintf("Cannot connect to Kubernetes API server. Check your kubeconfig.\n\n%s", e.Stderr)
	case strings.Contains(stderr, "unable to connect"):
		return fmt.Sprintf("Cannot connect to Kubernetes API server. Check your kubeconfig.\n\n%s", e.Stderr)
	default:
		return e.Stderr
	}
}

// RealExecutor shells out to kubectl.
type RealExecutor struct{}

// NewRealExecutor creates a new RealExecutor.
func NewRealExecutor() *RealExecutor {
	return &RealExecutor{}
}

// ExplainRecursive runs kubectl explain with --recursive for the given resource.
func (r *RealExecutor) ExplainRecursive(ctx context.Context, resource string, flags Flags) (string, error) {
	args := buildArgs(resource, flags)
	args = append(args, "--recursive")
	return run(ctx, args)
}

// ExplainField runs kubectl explain for a specific field path.
func (r *RealExecutor) ExplainField(ctx context.Context, fieldPath string, flags Flags) (string, error) {
	args := buildArgs(fieldPath, flags)
	return run(ctx, args)
}

func buildArgs(resource string, flags Flags) []string {
	if strings.HasPrefix(resource, "-") {
		resource = "-" + resource
	}
	args := []string{"explain", resource}
	if flags.Kubeconfig != "" {
		args = append(args, "--kubeconfig", flags.Kubeconfig)
	}
	if flags.Context != "" {
		args = append(args, "--context", flags.Context)
	}
	if flags.APIVersion != "" {
		args = append(args, "--api-version", flags.APIVersion)
	}
	return args
}

func run(ctx context.Context, args []string) (string, error) {
	kubectlPath, err := exec.LookPath("kubectl")
	if err != nil {
		return "", &NotFoundError{Err: err}
	}

	cmd := exec.CommandContext(ctx, kubectlPath, args...) //nolint:gosec // args are constructed internally
	output, err := cmd.CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", &ExecError{
				Stderr:   strings.TrimSpace(string(output)),
				ExitCode: exitErr.ExitCode(),
			}
		}
		return "", fmt.Errorf("failed to execute kubectl: %w", err)
	}

	return string(output), nil
}
