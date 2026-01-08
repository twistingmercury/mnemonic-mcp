---
entity_name: Azure DevOps Pipeline Pattern
entity_type: devops-pattern
language: yaml
domain: devops
description: Azure DevOps CI/CD pipeline pattern with multi-stage builds, testing, security scanning, and deployment stages for Go services
tags:
  - Azure DevOps
  - CI/CD
  - pipeline
  - YAML
  - deployment
---

# Azure DevOps Pipeline Pattern

## Philosophy

Create comprehensive Azure DevOps pipelines that execute the complete build workflow: tool installation, code analysis, testing, Docker build, E2E tests, coverage reporting, and ACR push. Use environment variables for configuration.

## Complete Pipeline YAML

**azure-pipelines.yml**:

```yaml
trigger:
  branches:
    include:
      - main
  tags:
    include:
      - v*

pool:
  vmImage: "ubuntu-latest"

variables:
  SERVICE_NAME: "your-service"
  CONTAINER_REGISTRY: "yourregistry.azurecr.io"
  GO_VERSION: "1.25"

stages:
  - stage: Build
    jobs:
      - job: BuildAndTest
        steps:
          - task: GoTool@0
            inputs:
              version: "$(GO_VERSION)"

          - task: AzureCLI@2
            displayName: "Azure CLI Login"
            inputs:
              azureSubscription: "your-service-connection"
              scriptType: "bash"
              scriptLocation: "inlineScript"
              inlineScript: |
                az acr login --name $(echo $(CONTAINER_REGISTRY) | cut -d'.' -f1)

          - script: |
              export VERSION=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
              export COMMIT=$(git rev-parse --short HEAD)
              export BUILD_DATE=$(date +%Y-%m-%dT%H:%M:%S)
              export CONTAINER_REGISTRY=$(CONTAINER_REGISTRY)
              ./build/build.sh
            displayName: "Run Build Pipeline"

          - task: PublishTestResults@2
            condition: succeededOrFailed()
            inputs:
              testResultsFormat: "JUnit"
              testResultsFiles: "**/junit.xml"
              failTaskOnFailedTests: true

          - task: PublishCodeCoverageResults@2
            inputs:
              codeCoverageTool: "Cobertura"
              summaryFileLocation: "**/coverage.xml"

          - script: |
              docker push $(CONTAINER_REGISTRY)/$(SERVICE_NAME):$(VERSION)
              docker push $(CONTAINER_REGISTRY)/$(SERVICE_NAME):latest
            displayName: "Push Images to ACR"
```

## Pipeline Components

### Trigger Configuration

```yaml
trigger:
  branches:
    include:
      - main          # Trigger on commits to main
  tags:
    include:
      - v*            # Trigger on version tags (v1.0.0, v2.0.0, etc.)
```

### Pool Selection

```yaml
pool:
  vmImage: "ubuntu-latest"    # Use latest Ubuntu agent
```

For self-hosted agents:
```yaml
pool:
  name: "Your-Agent-Pool"
```

### Variable Configuration

```yaml
variables:
  SERVICE_NAME: "your-service"
  CONTAINER_REGISTRY: "yourregistry.azurecr.io"
  GO_VERSION: "1.25"
  # Add more variables as needed
```

### Go Setup

```yaml
- task: GoTool@0
  inputs:
    version: "$(GO_VERSION)"
```

### Azure Container Registry Authentication

```yaml
- task: AzureCLI@2
  displayName: "Azure CLI Login"
  inputs:
    azureSubscription: "your-service-connection"
    scriptType: "bash"
    scriptLocation: "inlineScript"
    inlineScript: |
      az acr login --name $(echo $(CONTAINER_REGISTRY) | cut -d'.' -f1)
```

**Service Connection Setup**:
1. Go to Project Settings → Service connections
2. Create new Azure Resource Manager connection
3. Use service connection name in `azureSubscription` field

### Build Execution

```yaml
- script: |
    export VERSION=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
    export COMMIT=$(git rev-parse --short HEAD)
    export BUILD_DATE=$(date +%Y-%m-%dT%H:%M:%S)
    export CONTAINER_REGISTRY=$(CONTAINER_REGISTRY)
    ./build/build.sh
  displayName: "Run Build Pipeline"
```

The build script (`build/build.sh`) should orchestrate:
1. Tool installation (golangci-lint, govulncheck, gosec)
2. Code analysis (lint, vulnerability scan, security scan)
3. Unit tests with coverage
4. Docker image build
5. E2E tests via Docker Compose
6. Coverage report generation

### Test Results Publishing

```yaml
- task: PublishTestResults@2
  condition: succeededOrFailed()    # Run even if build fails
  inputs:
    testResultsFormat: "JUnit"
    testResultsFiles: "**/junit.xml"
    failTaskOnFailedTests: true      # Fail pipeline if tests fail
```

### Code Coverage Publishing

```yaml
- task: PublishCodeCoverageResults@2
  inputs:
    codeCoverageTool: "Cobertura"
    summaryFileLocation: "**/coverage.xml"
```

### Docker Image Push

```yaml
- script: |
    docker push $(CONTAINER_REGISTRY)/$(SERVICE_NAME):$(VERSION)
    docker push $(CONTAINER_REGISTRY)/$(SERVICE_NAME):latest
  displayName: "Push Images to ACR"
```

## Multi-Stage Pipeline Pattern

For more complex workflows with deployment stages:

```yaml
stages:
  - stage: Build
    jobs:
      - job: BuildAndTest
        steps:
          # Build steps as shown above

  - stage: DeployDev
    dependsOn: Build
    condition: and(succeeded(), eq(variables['Build.SourceBranch'], 'refs/heads/main'))
    jobs:
      - deployment: DeployToDev
        environment: 'development'
        strategy:
          runOnce:
            deploy:
              steps:
                - script: |
                    # Deployment steps (kubectl, helm, etc.)
                  displayName: "Deploy to Development"

  - stage: DeployProd
    dependsOn: DeployDev
    condition: and(succeeded(), startsWith(variables['Build.SourceBranch'], 'refs/tags/v'))
    jobs:
      - deployment: DeployToProd
        environment: 'production'
        strategy:
          runOnce:
            deploy:
              steps:
                - script: |
                    # Production deployment steps
                  displayName: "Deploy to Production"
```

## CLI Tool Pipeline Pattern

For CLI tools that need multi-platform builds:

```yaml
trigger:
  branches:
    include:
      - main
  tags:
    include:
      - v*

pool:
  vmImage: "ubuntu-latest"

variables:
  BINARY_NAME: "your-cli"
  GO_VERSION: "1.25"

stages:
  - stage: Build
    jobs:
      - job: BuildCLI
        steps:
          - task: GoTool@0
            inputs:
              version: "$(GO_VERSION)"

          - script: |
              export VERSION=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
              export COMMIT=$(git rev-parse --short HEAD)
              export BUILD_DATE=$(date +%Y-%m-%dT%H:%M:%S)
              ./scripts/build/build.sh
            displayName: "Build All Platforms"

          - task: PublishBuildArtifacts@1
            inputs:
              pathToPublish: '.bin'
              artifactName: 'binaries'
            displayName: "Publish Binaries"

          - task: GitHubRelease@1
            condition: startsWith(variables['Build.SourceBranch'], 'refs/tags/v')
            inputs:
              gitHubConnection: 'GitHub-Connection'
              repositoryName: '$(Build.Repository.Name)'
              action: 'create'
              target: '$(Build.SourceVersion)'
              tagSource: 'gitTag'
              assets: '.bin/**/*'
              changeLogCompareToRelease: 'lastFullRelease'
            displayName: "Create GitHub Release"
```

## Key Practices

- **Trigger on main and tags**: Automated builds on merge and release
- **Use GoTool@0 task**: Ensures correct Go version
- **ACR authentication via AzureCLI@2**: Secure credential management
- **Export version from git tags**: Automatic semantic versioning
- **Publish test results always**: Even when build fails, see test failures
- **Publish coverage**: Track coverage trends over time
- **Tag images properly**: Both version tag and latest
- **Use service connections**: Never hardcode credentials
- **Condition on branch/tag**: Different behavior for main vs tags

## Required Azure DevOps Setup

1. **Service Connection**: Azure Resource Manager connection for ACR
2. **Agent Pool**: ubuntu-latest or self-hosted
3. **Pipeline Variables**: Set in Azure DevOps UI if needed
4. **Environments**: For deployment stages (development, production)
5. **GitHub Connection**: For GitHub Release task (CLI tools)

## Troubleshooting

- **ACR login fails**: Check service connection permissions
- **Go version mismatch**: Verify GO_VERSION variable
- **Test results not appearing**: Check junit.xml path
- **Coverage not showing**: Verify coverage.xml format (Cobertura)
- **Docker push fails**: Ensure image was tagged with full registry path
