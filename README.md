# kubectl-inspect

A terminal UI kubectl plugin for interactively browsing Kubernetes resource and CRD field structures.

Instead of manually typing `kubectl explain <resource.path.to.field>` for each nested field, `kubectl-inspect` gives you a split-pane TUI with a collapsible tree on the left and the full `kubectl explain` output on the right.

## Motivation

After repeatedly typing `kubectl explain kind.field`, reading the subfields, then typing `kubectl explain kind.field.subfield`, and doing this over and over again, I wanted a kubectl plugin that would let me explore resource fields interactively and more efficiently.

## Features

- Browse any Kubernetes resource or CRD fields as a collapsible tree
- See full `kubectl explain` details for the selected field in real time
- Handles named types (`ObjectMeta`, `DeploymentSpec`, `[]Container`, etc.) and enum metadata lines
- Always fetches fresh data from the API server
- Copy the field path to the clipboard with `Enter` (e.g., `spec.template.spec.containers`)
- Works with built-in resources and custom CRDs

## Installation

```bash
go install github.com/irenedo/kubectl-inspect@latest
```

Or build from source:

```bash
git clone https://github.com/irenedo/kubectl-inspect.git
cd kubectl-inspect
go build -o kubectl-inspect .
mv kubectl-inspect /usr/local/bin/
```

kubectl discovers the plugin automatically via the `kubectl-` prefix:

```bash
kubectl inspect deployment
```

## Usage

```
kubectl inspect <resource> [flags]

Examples:
  kubectl inspect deployment
  kubectl inspect pod.spec.containers
  kubectl inspect certificates.cert-manager.io

Flags:
      --api-version string   API version (e.g., apps/v1)
      --context string       Kubernetes context to use
      --kubeconfig string    Path to kubeconfig file
  -h, --help                 Help for inspect
```

## Layout

```
╭───────────────────────────────────────────────────────────────────────────────────╮
│                              Deployment (apps/v1)                                 │
│ ► apiVersion <string>         │ GROUP:  apps                                      │
│ ► kind <string>               │ KIND:   Deployment                                │
│ ▼ spec <DeploymentSpec>       │ VERSION: v1                                       │
│   ● replicas <integer>        │                                                   │
│   ► selector <LabelSelector>  │ DESCRIPTION:                                      │
│   ► strategy <DeploymentStrat>│   The deployment strategy to use to replace...    │
│   ► template <PodTemplateSpec>│                                                   │
│ ► status <DeploymentStatus>   │ FIELDS:                                           │
│                               │   replicas  <integer>                             │
│                               │   selector  <LabelSelector>                       │
│                               │   ...                                             │
│    ↑/↓ navigate • Tab expand • Esc collapse • Enter copy path • q/Q quit          │
╰───────────────────────────────────────────────────────────────────────────────────╯
```

## Key Bindings

| Key | Action |
|-----|--------|
| `Up` / `k` | Move cursor up |
| `Down` / `j` | Move cursor down |
| `Tab` | Expand branch and move cursor to first child |
| `Esc` | Collapse current branch (or parent if on a child/leaf) |
| `Enter` | Copy field path to clipboard (e.g., `spec.replicas`) |
| `PgDown` / `Ctrl+D` | Scroll detail pane down |
| `PgUp` / `Ctrl+U` | Scroll detail pane up |
| `q` / `Ctrl+C` | Quit |

## Development

Requires [Task](https://taskfile.dev/) for task running. Install it following the [official instructions](https://taskfile.dev/docs/installation).

```bash
# Run tests
task test

# Build
task build

# Build with coverage
task test-cover
```
