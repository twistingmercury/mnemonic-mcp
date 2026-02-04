[![Mnemonic CD](https://github.com/twistingmercury/ace/actions/workflows/mnemonic-cd.yaml/badge.svg)](https://github.com/twistingmercury/ace/actions/workflows/mnemonic-cd.yaml)

# ACE (Agent Coordination Engine)

> **Maturity Level**: Emerging - CI/CD operational, Mnemonic repositories under development

***

ACE delivers deterministic agent routing and institutional knowledge retrieval for Claude Code, combining code-based routing logic with semantic knowledge graph search to provide consistent, context-aware agent orchestration.

## Usage

The ACE CLI will orchestrate agent selection and pattern retrieval:

```bash
# Route a user request to the appropriate agent
ace route "implement user authentication"

# Returns: go-software-agent with relevant security patterns

# Execute with context from Mnemonic's knowledge graph
ace exec --agent go-software-agent --task "implement JWT authentication"
```

Configuration via `~/.ace/config.yaml`:

```yaml
mnemonic:
  url: http://localhost:8080
  timeout: 30s

routing:
  default_agent: go-software-agent
  confidence_threshold: 0.8
```

## How it works

ACE is a monorepo containing two binaries built from a single Go module:

| Binary       | Purpose                                                             | Status         |
| ------------ | ------------------------------------------------------------------- | -------------- |
| **mnemonic** | Backend server providing routing and pattern retrieval via REST API | In development |
| **ace**      | CLI client that orchestrates routing decisions and execution        | Planned        |

The system provides three core capabilities:

**Deterministic routing**: Code-based logic maps requests to specialist agents using pattern matching and semantic analysis. Unlike LLM-based routing, the same input always routes to the same agent, ensuring predictable behavior.

**Semantic knowledge retrieval**: Mnemonic stores engineering patterns, guidelines, and institutional knowledge in a knowledge graph (Postgres + PGVector + Neo4j). Relevant patterns are retrieved using semantic search and graph traversal, providing agents with project-specific context.

**Local-first execution**: All LLM interactions and file operations happen on your workstation. Mnemonic only provides routing decisions and patterns; sensitive code never leaves your machine.

### Phased Approach

- **Phase 1** (Current): CLI invokes Claude Code for execution with enriched context
- **Phase 2** (Future): CLI calls Anthropic API directly, removing Claude Code dependency
- **Phase 3** (Future): Multi-user authentication, rate limiting, and remote deployment

## Key Considerations

- Mnemonic is under active development; CI/CD pipeline and repository layer are functional, API and routing engine are in progress
- ACE CLI implementation is planned after Mnemonic backend reaches MVP
- MVP targets local deployment with a single agent type (go-software-agent)
- Mnemonic serves only ACE for MVP (not a general-purpose memory service)
- Authentication and multi-region deployment are post-MVP features

## Development Considerations

### Quick Start

Clone and build:

```bash
git clone https://github.com/twistingmercury/ace.git
cd ace/src/mnemonic
./build/build.sh
```

Requires Go 1.25+, Docker 27+, and Docker Compose 2.32+ ([Go installation](https://go.dev/doc/install), [Docker installation](https://docs.docker.com/get-docker/))

### Building & running

Build and test Mnemonic:

```bash
cd src/mnemonic
./build/build.sh
```

The build script runs unit tests, integration tests (with PostgreSQL in Docker), and builds the Docker image.

### Testing

Run unit tests:

```bash
cd src/mnemonic
go test ./...
```

Run integration tests (requires Docker):

```bash
cd src/mnemonic/internal/repository/tests
./run-agent-integration-tests.sh
./run-pattern-integration-tests.sh
```

The build script runs both unit and integration tests automatically.

### Versioning

This project follows [Semantic Versioning 2.0.0](https://semver.org/).

Version is determined from git tags:

```bash
git describe --tags --always
```

No releases published yet. See [CHANGELOG.md](CHANGELOG.md) for development progress.

## Documentation

### Background

- [Project Blog](https://twistingmercury.github.io) - Development journey, design rationale, and updates

### Architecture

- [Architecture Overview](docs/architecture/00-overview.md) - System model, phased approach, key principles
- [Requirements](docs/architecture/01-requirements.md) - Problem statement and success criteria
- [Architectural Decisions](docs/architecture/02-architectural-decisions.md) - Major decisions with rationale
- [System Architecture](docs/architecture/03-system-architecture.md) - Component breakdown and data flow

### Design

- [API Specification](docs/design/mnemonic_service/api-specification.md) - OpenAPI spec for Mnemonic REST API
- [Pattern Processing](docs/design/mnemonic_service/pattern-processing.md) - Pattern enrichment and search pipeline
- [ACE CLI Configuration](docs/design/ace_cli/configuration.md) - CLI configuration reference
- [Mnemonic Configuration](docs/design/mnemonic_service/configuration.md) - Server configuration reference
