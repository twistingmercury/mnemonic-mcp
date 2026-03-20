# Mnemonic

[![Mnemonic CI](https://github.com/twistingmercury/mnemonic/actions/workflows/mnemonic-ci.yaml/badge.svg)](https://github.com/twistingmercury/mnemonic/actions/workflows/mnemonic-ci.yaml)

> **Maturity Level**: Emerging — MCP server functional, Admin REST API in a separate service (`mnemonic-api`)
> **Version**: v0.1.7

> - **Emerging**: Prototype, not production-ready, expect breaking changes
> - **Basic**: Production-ready but actively evolving, expect minor version changes
> - **Mature**: Stable, battle-tested, changes are rare

---

## Table of Contents

- [Usage](#usage)
- [How it works](#how-it-works)
- [Key Considerations](#key-considerations)
- [Development Considerations](#development-considerations)
- [Documentation](#documentation)

## Usage

Mnemonic exposes an MCP server for Claude Code (port 8081):

```json
{
  "tool": "search_patterns",
  "arguments": { "query": "Go error handling patterns" }
}
```

MCP tools available: `search_patterns`, `find_related_patterns`, `get_pattern`.

Pattern and agent data is managed via the companion [mnemonic-api](https://github.com/twistingmercury/mnemonic-api) service.

## How it works

Mnemonic stores curated engineering patterns in Postgres (with PGVector for embeddings) and Neo4j (for concept relationships). When a pattern is created via `mnemonic-api`, an enrichment job is queued. The enrichment worker in this service embeds the pattern content via OpenAI and syncs extracted concepts to Neo4j, enabling semantic search and graph traversal.

**Local dev stack (Docker Compose):**

| Service | Image | Role |
|---------|-------|------|
| `dev_mcp` | `ghcr.io/twistingmercury/mnemonic` | MCP server (port 8081) + enrichment worker |
| `dev_api` | `ghcr.io/twistingmercury/mnemonic-api` | Admin REST API (port 8080) |
| `dev_postgres` | `ghcr.io/twistingmercury/mnemonic-postgres` | Postgres + PGVector |
| `dev_neo4j` | `ghcr.io/twistingmercury/mnemonic-neo4j` | Neo4j + APOC |

Both database images are pre-configured with the required schema — no migration step needed.

## Key Considerations

- **MVP scope**: Local deployment via Docker Compose, single-user trusted environment, no authentication
- **This repo**: MCP server + enrichment worker only — Admin REST API lives in `mnemonic-api`
- **Enrichment**: Database-driven job queue; worker polls Postgres and processes pending jobs asynchronously
- **Post-MVP**: Event-driven enrichment (queue-based), multi-user auth, production deployment

## Development Considerations

### Quick Start

Requires Go 1.25+, Docker 27+, Docker Compose 2.32+.

```bash
git clone https://github.com/twistingmercury/mnemonic.git
cd mnemonic
make mnemonic       # build image + run full E2E test suite
make start          # start local dev stack
```

### Testing

```bash
# Unit tests
cd src/mnemonic
go test ./...

# Integration tests (requires Docker)
cd src/mnemonic/internal/repository/tests
./run-agent-integration-tests.sh
./run-pattern-integration-tests.sh

# Full build + E2E tests
make mnemonic
```

### Versioning

This project follows [Semantic Versioning 2.0.0](https://semver.org/).

Version is determined from git tags:

```bash
git describe --tags --always
```

See [CHANGELOG.md](CHANGELOG.md) for development progress.

## Documentation

### Architecture

- [Architectural Decisions](docs/architecture/00-architectural-decisions.md)
- [Security Architecture](docs/architecture/01-security-architecture.md)
- [System Architecture](docs/architecture/02-system-architecture.md)
- [Communication Patterns](docs/architecture/03-communication-patterns.md)
- [Data Architecture](docs/architecture/04-data-architecture.md)
- [Database Integration Flow](docs/architecture/05-database-integration-flow.md)
- [Deployment Architecture](docs/architecture/06-deployment-architecture.md)
- [Observability Architecture](docs/architecture/07-observability-architecture.md)
- [MCP Tools](docs/architecture/08-mcp-tools.md)

### Design

- [Pattern Processing](docs/design/pattern-processing.md) — enrichment and search pipeline
- [MCP Server](docs/design/mcp-server.md) — MCP protocol integration
- [Service Layer](docs/design/service-layer.md) — service package design
- [Observability](docs/design/observability-implementation.md) — metrics, tracing, logging
- [Configuration](docs/design/configuration.md) — server configuration reference
- [Data Storage](docs/design/data-storage.md) — storage design
