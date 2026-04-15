# GitHub Actions workflows

## Overview

This project uses two workflows:

### CI Workflow (`.github/workflows/ci.yaml`)

**Triggered on:** Pull requests (create/update to `main`)

**Purpose:** Code validation and testing

**What it does:**
- Installs Task (build tool)
- Runs `task check` (lint, test, trivy security scan, workflow lint)
- No builds or releases

### Release Workflow (`.github/workflows/release.yaml`)

**Triggered on:** Git tags matching `v*.*.*` pattern (e.g., `v1.0.0`)

**Purpose:** Build and publish releases

**What it does:**
- Uses GoReleaser to build cross-platform binaries
- Creates GitHub release with binary assets (linux/darwin/windows, amd64/arm64)
- Uploads artifacts for all supported platforms

## Related documentation

- Root `README.md`: Main project documentation
