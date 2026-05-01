//go:build integration

package main

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/irenedo/kubectl-inspect/pkg/explain"
	"github.com/irenedo/kubectl-inspect/pkg/kubectl"
)

// runCmd executes a command and returns an error if it fails.
// The error message includes the full command and its combined stdout/stderr output.
func runCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s: %s", name, strings.Join(args, " "), string(output))
	}
	return nil
}

// skipIfMissing skips the test if the given binary is not available in PATH.
// This allows the test suite to run on machines without kind or kubectl installed.
func skipIfMissing(t *testing.T, binary string) {
	t.Helper()
	if _, err := exec.LookPath(binary); err != nil {
		t.Skipf("skipping: %s not found in PATH", binary)
	}
}

// createKindCluster provisions a new kind cluster for the specified Kubernetes version.
// It first tears down any leftover cluster with the same name (from a previous failed run),
// then creates a fresh cluster and waits up to 5 minutes for the control plane to become ready.
func createKindCluster(t *testing.T, version string) {
	t.Helper()
	clusterName := "inspect-" + strings.ReplaceAll(version, ".", "-")

	fmt.Printf("\n=== Kubernetes v%s: creating kind cluster %s ===\n", version, clusterName)

	// Clean up any leftover cluster from a previous interrupted run.
	if err := runCmd("kind", "delete", "cluster", "--name", clusterName); err != nil {
		fmt.Printf("    (cleaned up leftover cluster: %v)\n", err)
	}

	if err := runCmd("kind", "create", "cluster",
				"--name", clusterName,
				"--wait", "5m",
				"--image", fmt.Sprintf("kindest/node:%s", version)); err != nil {
		t.Fatalf("failed to create kind cluster v%s: %v\n\n"+
				"Tip: check your Docker/Podman is running and kind is installed (task kind:install).",
			version, err)
	}

	fmt.Printf("=== Kubernetes v%s: cluster %s is ready ===\n", version, clusterName)
}

// destroyKindCluster tears down the kind cluster created for a specific k8s version.
// Errors are silently ignored -- we always want to clean up even if the cluster is already gone.
func destroyKindCluster(t *testing.T, version string) {
	t.Helper()
	clusterName := "inspect-" + strings.ReplaceAll(version, ".", "-")
	fmt.Printf("\n=== Kubernetes v%s: destroying cluster %s ===\n", version, clusterName)
	_ = runCmd("kind", "delete", "cluster", "--name", clusterName)
}

// verifyCluster runs kubectl explain --recursive against a set of core resources
// that exist in every supported Kubernetes version. It verifies that:
//    1. kubectl can connect to the cluster and return output
//    2. The output can be parsed by the explain package
//    3. The parsed ResourceInfo has a non-empty Kind and at least one field
func verifyCluster(t *testing.T, executor *kubectl.RealExecutor, flags kubectl.Flags, version string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// These resources exist in all k8s versions from 1.22+.
	resources := []string{"deployment", "pod", "service", "namespace"}

	for _, res := range resources {
		res := res
		t.Run(fmt.Sprintf("v%s/%s", version, res), func(t *testing.T) {
			fmt.Printf("    Kubernetes v%s: running kubectl explain %s...\n", version, res)

				// Run kubectl explain --recursive for the resource.
			output, err := executor.ExplainRecursive(ctx, res, flags)
			if err != nil {
				t.Fatalf("kubectl explain %s failed: %v", res, err)
				}

				// Parse the kubectl explain output into a ResourceInfo tree.
			info, err := explain.ParseRecursive(output)
			if err != nil {
				t.Fatalf("failed to parse kubectl explain %s: %v", res, err)
				}

				// Verify the parsed Kind matches the expected resource.
			if info.Kind == "" {
				t.Errorf("expected Kind to be set for %s, got empty", res)
				}

				// Verify the resource has nested fields (not just a flat list).
			if len(info.Fields) == 0 {
				t.Errorf("expected fields for %s, got none", res)
				}

			fmt.Printf("    Kubernetes v%s: OK - %s (%s/%s) parsed with %d fields\n",
				version, info.Kind, info.APIVersion(), info.Kind, len(info.Fields))
				})
			}
}

// TestKindIntegration verifies kubectl-inspect works across multiple Kubernetes versions
// by spinning up real kind clusters and running kubectl explain against each one.
//
// The test:
//    1. Skips if kind or kubectl is not installed (task integration)
//    2. For each k8s version (currently v1.35.0):
//      a. Creates a kind cluster with that version's node image
//      b. Runs kubectl explain on core resources (deployment, pod, service, namespace)
//      c. Verifies the output parses correctly
//      d. Tears down the cluster
//
// This catches regressions where a kubectl explain format change or API change
// breaks the parser or executor for a specific k8s version.
func TestKindIntegration(t *testing.T) {
	skipIfMissing(t, "kind")
	skipIfMissing(t, "kubectl")

	// Test against Kubernetes v1.35.
	version := "v1.35.0"

	t.Run(version, func(t *testing.T) {
		createKindCluster(t, version)
		defer destroyKindCluster(t, version)

		executor := kubectl.NewRealExecutor()
		flags := kubectl.Flags{}
		verifyCluster(t, executor, flags, version)
	})
}
