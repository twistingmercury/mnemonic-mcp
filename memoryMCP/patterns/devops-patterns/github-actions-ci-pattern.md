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
          echo "version=$(git describe --tags --abbrev=0 2>/dev/null || echo 'untagged')" >> "$GITHUB_OUTPUT"
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
    - export BUILD_VER=$(git describe --tags --abbrev=0 2>/dev/null || echo 'untagged')
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
      export BUILD_VER=$(git describe --tags --abbrev=0 2>/dev/null || echo 'untagged')
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

## Key Practices

- **Full git history**: Always use `fetch-depth: 0` for version detection
- **Env vars for metadata**: Pass version info to script, don't duplicate extraction logic
- **Named artifacts**: Include commit or version in artifact names
- **Retention policies**: Set appropriate artifact retention (30 days for CI, 90+ for releases)
- **Script does the work**: CI orchestrates, script implements
