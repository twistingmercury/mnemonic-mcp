---
entity_name: GitHub Actions CI Pattern
entity_type: devops-pattern
language: agnostic
domain: devops
description: Minimal GitHub Actions workflow pattern that delegates build logic to containerized scripts for portability
tags:
  - GitHub-Actions
  - CI
  - Docker
  - portable
---

# GitHub Actions CI Pattern

## Philosophy

When build logic lives in Docker and shell scripts, CI configuration becomes minimal. The CI platform just orchestrates - it doesn't own your build process. This makes switching between GitHub, GitLab, Azure DevOps trivial.

The build script contains all the complexity: multi-stage Docker builds, test execution, binary exports. The CI workflow simply calls that script and handles artifacts. This separation means:

- Local builds work identically to CI builds
- CI vendor lock-in disappears
- Build logic is testable and version-controlled
- Platform-specific quirks stay out of your build process

## The Pattern

A minimal workflow that:

1. Checks out code with full git history (for tags/version)
2. Extracts build metadata from git
3. Runs the build script (which handles Docker builds internally)
4. Uploads artifacts

## Example Workflow

**.github/workflows/ci.yml**:

```yaml
name: CI

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main, develop]

jobs:
  build:
    name: Build and Test
    runs-on: ubuntu-latest

    steps:
      - name: Checkout code
        uses: actions/checkout@v4
        with:
          fetch-depth: 0  # Full history for git tags

      - name: Set build metadata
        id: meta
        run: |
          echo "version=$(git describe --tags --abbrev=0 2>/dev/null || echo 'v0.0.0')" >> "$GITHUB_OUTPUT"
          echo "date=$(date +%Y-%m-%d)" >> "$GITHUB_OUTPUT"
          echo "commit=$(git rev-parse --short=8 HEAD)" >> "$GITHUB_OUTPUT"

      - name: Run build script
        env:
          BUILD_VER: ${{ steps.meta.outputs.version }}
          BUILD_DATE: ${{ steps.meta.outputs.date }}
          BUILD_COMMIT: ${{ steps.meta.outputs.commit }}
        run: ./build/build.sh

      - name: Upload binaries
        uses: actions/upload-artifact@v4
        with:
          name: binaries-${{ steps.meta.outputs.commit }}
          path: .bin/
          retention-days: 30
```

## Key Points

- **fetch-depth: 0**: Required for git tags to be available for version detection
- **Metadata extraction**: Same logic as build script defaults, passed as env vars
- **Build script does the work**: All Docker complexity lives in the script
- **Artifacts named with commit**: Easy to identify which build produced which artifacts

## Why This Works

- CI config is ~40 lines of simple YAML
- Build logic is tested locally (same script, same Docker)
- Switching to GitLab CI or Azure Pipelines is near copy-paste
- No CI-specific build steps that could drift from local builds

## Comparison: Same Build on Different Platforms

Because the build script handles all complexity, CI configurations across platforms look nearly identical.

### GitLab CI

**.gitlab-ci.yml**:

```yaml
stages:
  - build

build:
  stage: build
  image: docker:24
  services:
    - docker:24-dind
  variables:
    DOCKER_TLS_CERTDIR: "/certs"
  before_script:
    - apk add --no-cache git bash
  script:
    - export BUILD_VER=$(git describe --tags --abbrev=0 2>/dev/null || echo 'v0.0.0')
    - export BUILD_DATE=$(date +%Y-%m-%d)
    - export BUILD_COMMIT=$(git rev-parse --short=8 HEAD)
    - ./build/build.sh
  artifacts:
    paths:
      - .bin/
    expire_in: 30 days
```

### Azure Pipelines

**azure-pipelines.yml**:

```yaml
trigger:
  branches:
    include:
      - main
      - develop

pr:
  branches:
    include:
      - main
      - develop

pool:
  vmImage: ubuntu-latest

steps:
  - checkout: self
    fetchDepth: 0

  - script: |
      export BUILD_VER=$(git describe --tags --abbrev=0 2>/dev/null || echo 'v0.0.0')
      export BUILD_DATE=$(date +%Y-%m-%d)
      export BUILD_COMMIT=$(git rev-parse --short=8 HEAD)
      ./build/build.sh
    displayName: Build and Test

  - publish: .bin/
    artifact: binaries-$(Build.SourceVersion)
```

## Workflow Variations

### Release Workflow

Trigger on tags for releases with additional artifact naming:

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  release:
    name: Build Release
    runs-on: ubuntu-latest

    steps:
      - name: Checkout code
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Get version from tag
        id: meta
        run: |
          echo "version=${GITHUB_REF_NAME}" >> "$GITHUB_OUTPUT"
          echo "date=$(date +%Y-%m-%d)" >> "$GITHUB_OUTPUT"
          echo "commit=$(git rev-parse --short=8 HEAD)" >> "$GITHUB_OUTPUT"

      - name: Run build script
        env:
          BUILD_VER: ${{ steps.meta.outputs.version }}
          BUILD_DATE: ${{ steps.meta.outputs.date }}
          BUILD_COMMIT: ${{ steps.meta.outputs.commit }}
        run: ./build/build.sh

      - name: Upload release artifacts
        uses: actions/upload-artifact@v4
        with:
          name: release-${{ steps.meta.outputs.version }}
          path: .bin/
          retention-days: 90
```

### Matrix Build for Multiple Runners

When you need builds on different OS runners (rare with containerized builds):

```yaml
name: CI Matrix

on:
  push:
    branches: [main]

jobs:
  build:
    name: Build on ${{ matrix.os }}
    runs-on: ${{ matrix.os }}
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest]

    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Run build script
        run: ./build/build.sh

      - uses: actions/upload-artifact@v4
        with:
          name: binaries-${{ matrix.os }}
          path: .bin/
```

## Build Script Contract

The CI workflow expects the build script to:

1. Accept version metadata via environment variables (BUILD_VER, BUILD_DATE, BUILD_COMMIT)
2. Default to extracting metadata from git if variables not set
3. Output artifacts to `.bin/` directory
4. Exit with non-zero code on failure

This contract keeps the CI configuration stable while allowing build logic to evolve independently.

## Permissions for Artifacts

When passing artifacts between workflows (CI → CD), specific permissions are required:

```yaml
permissions:
  contents: read
  actions: write  # Required for cross-workflow artifact access
```

**Permission Requirements**:

| Scenario | Permission | Why |
|----------|------------|-----|
| Upload artifacts for same workflow | None (default) | Same workflow access is implicit |
| Upload artifacts for other workflows | `actions: write` | Cross-workflow artifact access |
| Download from same workflow | None (default) | Same workflow access is implicit |
| Download from other workflow | `actions: read` | Cross-workflow artifact access |

**Example with Artifact Permissions**:

```yaml
name: CI

permissions:
  contents: read
  actions: write  # Enable CD workflow to download our artifacts

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Build
        run: ./build/build.sh

      - name: Save Docker image for CD
        run: docker save my-service:${{ steps.meta.outputs.version }} -o /tmp/image.tar

      - name: Upload for CD workflow
        uses: actions/upload-artifact@v4
        with:
          name: docker-image
          path: /tmp/image.tar
          retention-days: 1  # Short retention for intermediate artifacts
```

## Working Directory for Monorepos

For monorepos where the service lives in a subdirectory, use `defaults.run.working-directory`:

```yaml
name: CI

on:
  push:
    branches: [main, develop]
    paths:
      - 'services/my-service/**'
  pull_request:
    branches: [main, develop]
    paths:
      - 'services/my-service/**'

defaults:
  run:
    working-directory: services/my-service

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      # All run commands execute in services/my-service/
      - name: Build
        run: ./build/build.sh

      # Actions still need explicit paths
      - uses: actions/upload-artifact@v4
        with:
          name: binaries
          path: services/my-service/.bin/  # Full path required for actions
```

**Key Points**:

- `defaults.run.working-directory` affects all `run:` steps
- `paths:` filter limits workflow triggers to relevant files
- Actions (`uses:`) still require full paths from repo root
- Combine with `paths:` filter to avoid running CI for unrelated changes

## PR vs Push Behavior

Control build behavior differently for PRs vs direct pushes. Common pattern: skip registry push on PRs.

**Using Environment Variables**:

```yaml
jobs:
  build:
    runs-on: ubuntu-latest
    env:
      # LOCAL_BUILD=true on PRs to skip push operations
      LOCAL_BUILD: ${{ github.event_name == 'pull_request' }}

    steps:
      - uses: actions/checkout@v4

      - name: Build and test
        env:
          LOCAL_BUILD: ${{ env.LOCAL_BUILD }}
        run: ./build/build.sh
```

**In the build script**:

```bash
#!/usr/bin/env bash
set -e

# Skip push operations if LOCAL_BUILD is set
if [ "${LOCAL_BUILD}" = "true" ]; then
    echo "PR build - skipping registry push"
    # Build and test only, no push
    docker build -t "${IMAGE_NAME}:${VERSION}" .
else
    echo "Main branch build - will push to registry"
    # Full build with push
    docker build -t "${IMAGE_NAME}:${VERSION}" .
    docker push "${IMAGE_NAME}:${VERSION}"
fi
```

**Alternative: Conditional Job**:

```yaml
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: Build and test
        run: ./build/build.sh

  push:
    needs: build
    if: github.event_name == 'push'  # Skip on PRs
    runs-on: ubuntu-latest
    steps:
      - name: Push to registry
        run: ./scripts/push.sh
```

**Why Skip Push on PRs**:

- PRs should validate code, not deploy it
- Reduces CI time for PR feedback loop
- Prevents intermediate/broken images in registry
- CD workflow handles push after merge

## Key Practices

- **Full git history**: Always use `fetch-depth: 0` for version detection
- **Env vars for metadata**: Pass version info to script, don't duplicate extraction logic
- **Named artifacts**: Include commit or version in artifact names
- **Retention policies**: Set appropriate artifact retention (30 days for CI, 90+ for releases)
- **Script does the work**: CI orchestrates, script implements
- **Explicit permissions**: Declare `actions: write` when artifacts need cross-workflow access
- **Path filters for monorepos**: Use `paths:` to limit CI triggers to relevant directories
- **PR-aware builds**: Use `LOCAL_BUILD` or conditional jobs to skip push on PRs
