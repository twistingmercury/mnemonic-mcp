---
entity_name: GitHub Actions CD Pattern
entity_type: devops-pattern
language: agnostic
domain: devops
description: Continuous Delivery workflow pattern that deploys artifacts from a successful CI workflow to container registries
tags:
  - GitHub-Actions
  - CD
  - Docker
  - container-registry
  - workflow_run
---

# GitHub Actions CD Pattern

## Philosophy

Separate CI (build+test) from CD (deploy) workflows. CI produces artifacts and validates quality. CD only runs after CI succeeds and handles deployment tasks like registry pushes. This separation provides:

- Clear pipeline stages with explicit dependencies
- Failed builds never trigger deployment
- CI can run on PRs without pushing images
- CD workflow has minimal permissions (only what's needed for push)
- Easier auditing and debugging of pipeline failures

## The Pattern

A CD workflow that:

1. Triggers on successful completion of CI workflow
2. Downloads the Docker image artifact from CI
3. Loads the image into Docker
4. Authenticates to container registry
5. Pushes with version tag and conditional `latest` tag

## Permissions

CD workflows require specific permissions for artifact handling and registry push:

```yaml
permissions:
  contents: read     # Checkout code if needed
  packages: write    # Push to GitHub Container Registry
  actions: read      # Download artifacts from another workflow
```

**Key Permission Notes**:

- `actions: read` is required to download artifacts from a different workflow (CI → CD)
- `actions: write` is required in CI to upload artifacts for cross-workflow access
- `packages: write` enables pushing to GitHub Container Registry (ghcr.io)
- For external registries (ACR, ECR, Docker Hub), use secrets for authentication

## Example CD Workflow

**.github/workflows/cd.yml**:

```yaml
name: CD

on:
  workflow_run:
    workflows: ["CI"]
    types:
      - completed
    branches: [main, develop]

permissions:
  contents: read
  packages: write
  actions: read

jobs:
  deploy:
    name: Push to Registry
    runs-on: ubuntu-latest
    if: ${{ github.event.workflow_run.conclusion == 'success' }}

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Download Docker image artifact
        uses: actions/download-artifact@v4
        with:
          name: docker-image
          path: /tmp
          run-id: ${{ github.event.workflow_run.id }}
          github-token: ${{ secrets.GITHUB_TOKEN }}

      - name: Load Docker image
        run: docker load -i /tmp/image.tar

      - name: Set image metadata
        id: meta
        run: |
          echo "version=$(git describe --tags --abbrev=0 2>/dev/null || echo 'v0.0.0')" >> "$GITHUB_OUTPUT"
          echo "branch=${{ github.event.workflow_run.head_branch }}" >> "$GITHUB_OUTPUT"

      - name: Login to GitHub Container Registry
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Tag and push image
        env:
          REGISTRY: ghcr.io/${{ github.repository_owner }}
          IMAGE_NAME: my-service
          VERSION: ${{ steps.meta.outputs.version }}
          BRANCH: ${{ steps.meta.outputs.branch }}
        run: |
          # Tag with version
          docker tag "${IMAGE_NAME}:${VERSION}" "${REGISTRY}/${IMAGE_NAME}:${VERSION}"
          docker push "${REGISTRY}/${IMAGE_NAME}:${VERSION}"

          # Tag with 'latest' only on main branch
          if [ "$BRANCH" = "main" ]; then
            docker tag "${IMAGE_NAME}:${VERSION}" "${REGISTRY}/${IMAGE_NAME}:latest"
            docker push "${REGISTRY}/${IMAGE_NAME}:latest"
          fi
```

## CI Workflow Changes for Artifact Passing

The CI workflow must save the Docker image as an artifact. Add this to your CI workflow:

```yaml
permissions:
  contents: read
  actions: write  # Required for cross-workflow artifact access

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Set build metadata
        id: meta
        run: |
          echo "version=$(git describe --tags --abbrev=0 2>/dev/null || echo 'v0.0.0')" >> "$GITHUB_OUTPUT"

      # ... build and test steps ...

      - name: Save Docker image
        run: docker save my-service:${{ steps.meta.outputs.version }} -o /tmp/image.tar

      - name: Upload Docker image artifact
        uses: actions/upload-artifact@v4
        with:
          name: docker-image
          path: /tmp/image.tar
          retention-days: 1
```

## Conditional Latest Tag

The `latest` tag should only be applied on the main branch. This prevents development builds from overwriting production `latest`:

```bash
# Tag with 'latest' only on main branch
if [ "$BRANCH" = "main" ]; then
  docker tag "${IMAGE_NAME}:${VERSION}" "${REGISTRY}/${IMAGE_NAME}:latest"
  docker push "${REGISTRY}/${IMAGE_NAME}:latest"
fi
```

**Why This Matters**:

- `latest` typically means "latest stable/production release"
- Development branches should use version or commit tags
- Prevents accidental deployment of unstable code via `latest`

## Azure Container Registry Variant

For ACR instead of GHCR:

```yaml
      - name: Login to Azure Container Registry
        uses: docker/login-action@v3
        with:
          registry: ${{ secrets.ACR_LOGIN_SERVER }}
          username: ${{ secrets.ACR_USERNAME }}
          password: ${{ secrets.ACR_PASSWORD }}

      - name: Tag and push image
        env:
          REGISTRY: ${{ secrets.ACR_LOGIN_SERVER }}
          IMAGE_NAME: my-service
          VERSION: ${{ steps.meta.outputs.version }}
        run: |
          docker tag "${IMAGE_NAME}:${VERSION}" "${REGISTRY}/${IMAGE_NAME}:${VERSION}"
          docker push "${REGISTRY}/${IMAGE_NAME}:${VERSION}"
```

## Docker Hub Variant

For Docker Hub:

```yaml
      - name: Login to Docker Hub
        uses: docker/login-action@v3
        with:
          username: ${{ secrets.DOCKERHUB_USERNAME }}
          password: ${{ secrets.DOCKERHUB_TOKEN }}

      - name: Tag and push image
        env:
          REGISTRY: docker.io/${{ secrets.DOCKERHUB_USERNAME }}
          IMAGE_NAME: my-service
          VERSION: ${{ steps.meta.outputs.version }}
        run: |
          docker tag "${IMAGE_NAME}:${VERSION}" "${REGISTRY}/${IMAGE_NAME}:${VERSION}"
          docker push "${REGISTRY}/${IMAGE_NAME}:${VERSION}"
```

## workflow_run Considerations

The `workflow_run` trigger has specific behaviors:

1. **Branch Filtering**: The `branches` filter applies to the triggering workflow's branch, not the CD workflow
2. **Conclusion Check**: Always check `github.event.workflow_run.conclusion == 'success'`
3. **Context Differences**: `github.sha` and `github.ref` refer to the CD workflow trigger, use `github.event.workflow_run.*` for CI context

```yaml
# Access CI workflow context
${{ github.event.workflow_run.head_sha }}      # Commit that triggered CI
${{ github.event.workflow_run.head_branch }}   # Branch that triggered CI
${{ github.event.workflow_run.id }}            # Workflow run ID for artifact download
```

## Troubleshooting

### Artifact Not Found

**Symptom**: CD workflow fails with "Unable to find any artifacts" or similar error.

**Causes and Solutions**:

1. **Timing issue**: `workflow_run` can trigger before artifacts finish uploading
   ```yaml
   # Add a small delay or use artifact polling
   - name: Wait for artifact availability
     run: sleep 10
   ```

2. **Missing `run-id` parameter**: Cross-workflow downloads require the run ID
   ```yaml
   - uses: actions/download-artifact@v4
     with:
       name: docker-image
       run-id: ${{ github.event.workflow_run.id }}  # Required!
       github-token: ${{ secrets.GITHUB_TOKEN }}
   ```

3. **Permissions issue**: CI workflow needs `actions: write` to allow cross-workflow access
   ```yaml
   # In CI workflow
   permissions:
     actions: write
   ```

4. **Artifact expired**: Check retention settings (default 90 days, set to 1 day for intermediate artifacts)

### CD Workflow Not Triggering

**Symptom**: CI succeeds but CD never runs.

**Causes and Solutions**:

1. **Wrong workflow name**: The `workflows` array must match the CI workflow's `name` field exactly
   ```yaml
   # CD workflow
   on:
     workflow_run:
       workflows: ["CI"]  # Must match exactly
   ```

2. **Branch filter mismatch**: The `branches` filter in `workflow_run` applies to the CI workflow's branch
   ```yaml
   on:
     workflow_run:
       workflows: ["CI"]
       branches: [main, develop]  # CI must run on these branches
   ```

3. **CI workflow failed or was cancelled**: Check `conclusion` in the job condition
   ```yaml
   if: ${{ github.event.workflow_run.conclusion == 'success' }}
   ```

### Docker Image Load Fails

**Symptom**: `docker load` fails with "invalid tar header" or similar.

**Causes and Solutions**:

1. **Artifact compression**: GitHub may compress artifacts; ensure the tar is intact
   ```yaml
   - name: Load Docker image
     run: |
       ls -la /tmp/  # Verify artifact downloaded
       docker load -i /tmp/image.tar
   ```

2. **Wrong artifact path**: Verify the download path matches the load path

3. **Image name mismatch**: The loaded image has the name from `docker save`, not the artifact name

### Registry Push Fails

**Symptom**: `docker push` fails with authentication or permission errors.

**Causes and Solutions**:

1. **Missing permissions**: Ensure `packages: write` for GHCR
   ```yaml
   permissions:
     packages: write
   ```

2. **Incorrect registry URL**: GHCR uses `ghcr.io`, not `docker.pkg.github.com`

3. **Token scope**: `GITHUB_TOKEN` works for GHCR; external registries need secrets

## Key Practices

- **Explicit success check**: Always use `if: github.event.workflow_run.conclusion == 'success'`
- **Short artifact retention**: Use 1 day retention for intermediate artifacts
- **Minimal permissions**: Only request permissions actually needed
- **Conditional latest**: Only tag `latest` on main/master branch
- **Registry-agnostic pattern**: Same workflow structure works with any registry
- **Use docker/login-action**: Handles credential management securely
- **Version tagging**: Always include a version tag, not just `latest`

## Complete CI + CD Example

See the companion CI pattern for the full workflow:

1. CI workflow builds, tests, saves Docker image artifact
2. CD workflow downloads artifact, loads image, pushes to registry
3. Artifacts pass between workflows via `actions/upload-artifact` and `actions/download-artifact`

This separation keeps CI fast (no registry push on PRs) and CD focused (only deployment logic).
