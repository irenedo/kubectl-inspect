# GitHub Actions for kubectl-inspect

This directory (`.github/`) contains CI/CD workflows and related configuration.

## Documentation structure

- **Root README** (root): Main project documentation
- **`.github/workflows-README.md`**: This file — overview of workflows in `.github/`
- **`.github/workflows/`**: Contains actual workflow YAML files
- **`.github/workflows/README.md`**: Specific workflow documentation (currently minimal)

## Release Workflow (`.github/workflows/release.yaml`)

## Release Workflow (`.github/workflows/release.yaml`)

Triggers on git tags matching `v*` pattern:

- Builds cross-platform binaries (linux/darwin/windows, amd64/arm64)
- Runs tests
- Creates GitHub release with binary assets
- Publishes to krew

## Plugin Manifest (`plugin.yaml`)

Krew manifest defining installation for all supported platforms.
Generated binaries must match the filenames in `plugin.yaml`.
