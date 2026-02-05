# ACE (Agent Coordination Engine)

[![Mnemonic CD](https://github.com/twistingmercury/ace/actions/workflows/mnemonic-cd.yaml/badge.svg)](https://github.com/twistingmercury/ace/actions/workflows/mnemonic-cd.yaml)

> **Maturity Level**: Emerging - CI/CD operational, Mnemonic repositories
> under development

***

ACE delivers deterministic agent routing and institutional knowledge
retrieval for Claude Code, combining code-based routing logic with
semantic knowledge graph search to provide consistent, context-aware
agent orchestration.

## Usage

Mnemonic provides REST API endpoints for agent routing and pattern
retrieval. The ACE CLI (planned for separate repository) will consume
these endpoints to orchestrate agent selection.

Example routing request:

```bash
# Request routing decision from Mnemonic
curl -X POST http://localhost:8080/v1/api/route \
  -H "Content-Type: application/json" \
  -d '{"prompt": "implement user authentication"}'

# Returns: {"agent": "go-software-agent", "patterns": [...]}
```

See the [API Specification](docs/design/mnemonic_service/api-specification.md)
for complete endpoint documentation.

## How it works

This repository contains Mnemonic, the backend service for ACE.
The ACE CLI will be developed in a separate repository.

Mnemonic provides three core capabilities:

**Deterministic routing**: Code-based logic maps requests to specialist
agents using pattern matching and semantic analysis. Unlike LLM-based
routing, the same input always routes to the same agent, ensuring
predictable behavior.

**Semantic knowledge retrieval**: Mnemonic stores engineering patterns,
guidelines, and institutional knowledge in a knowledge graph (Postgres +
PGVector + Neo4j). Relevant patterns are retrieved using semantic search
and graph traversal, providing agents with project-specific context.

**REST API interface**: Mnemonic exposes routing and pattern retrieval
via REST endpoints, allowing any client to consume its capabilities.

### Phased Approach

- **Phase 1** (Current): Mnemonic backend with routing and pattern API
- **Phase 2** (Future): ACE CLI in separate repository
- **Phase 3** (Future): Multi-user authentication, rate limiting, and
  remote deployment

## Key Considerations

- Mnemonic is under active development; CI/CD pipeline and repository
  layer are functional, API and routing engine are in progress
- ACE CLI will be developed in a separate repository after Mnemonic
  reaches MVP
- MVP targets local deployment with a single agent type
  (go-software-agent)
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

Requires Go 1.25+, Docker 27+, and Docker Compose 2.32+
([Go installation](https://go.dev/doc/install),
[Docker installation](https://docs.docker.com/get-docker/))

### Building & running

Build and test Mnemonic:

```bash
cd src/mnemonic
./build/build.sh
```

The build script runs unit tests, integration tests (with PostgreSQL in
Docker), and builds the Docker image.

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

No releases published yet. See [CHANGELOG.md](CHANGELOG.md) for
development progress.

## Documentation

### Background

- [Project Blog](https://twistingmercury.github.io) - Development
  journey, design rationale, and updates

### Architecture

- [Architecture Overview](docs/architecture/00-overview.md) - System
  model, phased approach, key principles
- [Requirements](docs/architecture/01-requirements.md) - Problem
  statement and success criteria
- [Architectural Decisions](docs/architecture/02-architectural-decisions.md) -
  Major decisions with rationale
- [System Architecture](docs/architecture/03-system-architecture.md) -
  Component breakdown and data flow

### Design

- [API Specification](docs/design/mnemonic_service/api-specification.md) -
  OpenAPI spec for Mnemonic REST API
- [Pattern Processing](docs/design/mnemonic_service/pattern-processing.md) -
  Pattern enrichment and search pipeline
- [Mnemonic Configuration](docs/design/mnemonic_service/configuration.md) -
  Server configuration reference
