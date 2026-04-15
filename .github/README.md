# GitHub Actions for kubectl-inspect

This directory contains CI/CD workflows.

## Release Workflow (`.github/workflows/release.yaml`)

Triggers on push of version tags (e.g., `v1.0.0`):

- Builds cross-platform binaries (linux/darwin/windows, amd64/arm64)
- Runs tests
- Creates GitHub release with binary assets

## Plugin Manifest (`plugin.yaml`)

Krew manifest defining installation for all supported platforms.
Generated binaries must match the filenames in `plugin.yaml`.
