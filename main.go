package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/irenedo/kubectl-inspect/pkg/kubectl"
	"github.com/irenedo/kubectl-inspect/pkg/tui"
)

func main() {
	var flags kubectl.Flags

	binaryName := filepath.Base(os.Args[0])
	var displayName string
	if strings.HasPrefix(binaryName, "kubectl-") {
		displayName = "kubectl " + strings.TrimPrefix(binaryName, "kubectl-")
	} else {
		displayName = binaryName
	}
	example := displayName + " <resource>"

	cmd := &cobra.Command{
		Use:     displayName,
		Short:   "Interactively browse Kubernetes resource fields",
		Long:    "An interactive terminal UI for browsing Kubernetes resource and CRD field structures using kubectl explain.",
		Example: example,
		Args:    cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			resource := args[0]
			executor := kubectl.NewRealExecutor()
			model := tui.NewModel(resource, executor, flags)

			p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseAllMotion())
			if _, err := p.Run(); err != nil {
				return fmt.Errorf("running TUI: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&flags.Kubeconfig, "kubeconfig", "", "Path to kubeconfig file")
	cmd.Flags().StringVar(&flags.Context, "context", "", "Kubernetes context to use")
	cmd.Flags().StringVar(&flags.APIVersion, "api-version", "", "API version (e.g., apps/v1)")

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
