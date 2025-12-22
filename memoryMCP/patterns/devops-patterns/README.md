# DevOps Patterns for Cognee

This directory has comprehensive DevOps patterns for building, containerizing, and deploying Go applications. The `go-devops-engineer` agent uses these patterns when setting up build infrastructure and CI/CD pipelines.

## Pattern Files

Each pattern file has:

- **Frontmatter**: Entity metadata for Cognee (entity_name, entity_type)
- **Philosophy**: Core approach and principles
- **Complete Examples**: Working Dockerfiles, scripts, and pipeline YAML
- **Key Practices**: Best practices and security considerations
- **Troubleshooting**: Common issues and solutions

### Available Patterns

1. **service-dockerfile-pattern.md** - Multi-stage Dockerfile (Alpine → Scratch) for REST/gRPC services
2. **cli-dockerfile-pattern.md** - Multi-platform CLI tool cross-compilation
3. **azure-devops-pipeline-pattern.md** - Complete Azure DevOps pipeline YAML
4. **service-build-script-pattern.md** - Comprehensive build automation script
5. **library-build-pattern.md** - Containerized builds for Go libraries/packages (GitHub Actions)

## Pattern Coverage

### Service Deployment

- Multi-stage Dockerfiles with Alpine and Scratch
- Version embedding via ldflags
- OCI labels for metadata
- Security best practices

### CLI Tools

- Cross-compilation for Linux/macOS/Windows
- Multi-platform binary organization
- Distribution strategies

### CI/CD

- Azure DevOps pipeline configuration
- ACR authentication and push
- Test result publishing (JUnit XML)
- Code coverage publishing (Cobertura XML)
- Multi-stage pipelines with deployment

### Build Automation

- Comprehensive build scripts
- Quality gates (lint, security, tests)
- Coverage report generation
- Docker Compose E2E testing
- Utility library patterns

### Library/Package

- Containerized builds for local/CI parity
- SKIP_E2E pattern for hybrid execution
- GitHub Actions workflow
- No binary output (validation-focused)

## Usage with go-devops-engineer Agent

The `go-devops-engineer` agent grabs patterns from Cognee as needed:

1. **Agent identifies project type** (Service, CLI tool, etc.)
2. **Agent searches Cognee**: `search(search_query="Service Dockerfile pattern", search_type="GRAPH_COMPLETION")`
3. **Agent applies pattern**: Uses retrieved examples and practices

### Example Workflow

```text
User: "I've finished implementing the user management API. Set up build and deployment."

Agent (go-devops-engineer):
  1. Queries Cognee: search(search_query="Service Dockerfile pattern", search_type="GRAPH_COMPLETION")
  2. Retrieves patterns: Dockerfile, build script, Azure DevOps pipeline
  3. Creates complete build infrastructure following patterns

Result: Dockerfiles, build scripts, CI/CD pipeline YAML ready to use.
```

## Benefits

### Context Efficiency

- **Before**: 978-line agent definition eating up lots of context
- **After**: Condensed agent + on-demand pattern retrieval
- **Savings**: More context available for your project code

### Knowledge Centralization

- Single source of truth for DevOps patterns
- Consistent practices across all projects
- Easy updates (edit pattern, reload Cognee)

### Maintainability

- Patterns versioned in git
- Clear audit trail of how infrastructure evolves
- Team-wide shared knowledge

## Prerequisites

These patterns are already loaded in Cognee. If you need to reload them:

1. Patterns are stored in this directory
2. Use Cognee MCP tools to create entities
3. Patterns become automatically available to the go-devops-engineer agent

## Related Documentation

- [go-devops-engineer agent](../../definitions/go/go-devops-agent.md)
- [go-software-engineer agent](../../definitions/go/go-software-agent.md)
- [go-e2e-test-engineer agent](../../definitions/go/go-e2e-test-agent.md)
- [E2E Testing Patterns](../e2e-patterns/README.md)
