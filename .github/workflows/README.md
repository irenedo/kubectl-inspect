# GitHub Actions workflow for krew release

## Overview

Triggered on git tags matching `v*` pattern (e.g., `v1.0.0`).

## What it does

- Builds cross-platform binaries (linux/darwin/windows, amd64/arm64)
- Runs tests
- Creates GitHub release with binary assets
- Publishes to krew

## Related documentation

- `.github/workflows-README.md`: Overview of all workflows in `.github/`
- Root `README.md`: Main project documentation

## Flags

| Flag | Description |
|------|-------------|
| `--api-version` | API version (e.g., `apps/v1`) |
| `--context` | Kubernetes context to use |
| `--kubeconfig` | Path to kubeconfig file |
| `-h, --help` | Show help |

## Examples

```bash
kubectl inspect deployment
kubectl inspect pod.spec.containers
kubectl inspect certificates.cert-manager.io
```

## Layout

```bash
# Root README.md — main project docs (displayed on GitHub)
# .github/README.md — not present (would override root if exists)
# .github/workflows/README.md — workflow-specific docs (this file)
# .github/workflows-README.md — workflow directory overview
```

## Development

Requires [Task](https://taskfile.dev/) for task running.

```bash
# Run tests
task test

# Build
task build

# Build with coverage
task test-cover
```
