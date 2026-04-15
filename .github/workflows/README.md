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


