# ACE MVP Scope

[Back to Architecture Overview](../architecture/00-overview.md) | [Back to Project README](../../README.md)

## Executive Summary

ACE MVP delivers Phase 1 Claude Code integration with centralized routing and pattern retrieval through Mnemonic. LLM execution remains local on user workstations while routing logic and pattern storage are centralized server-side.

## MVP Scope - What's Included

### ACE CLI (Future - Separate Repository)

The ACE CLI will be developed in a separate repository once Mnemonic reaches MVP status. Planned high-level capabilities include:

- Routing requests and pattern retrieval via Mnemonic API
- Claude Code invocation with enriched context -> [Architecture Overview - Phase 1](../architecture/00-overview.md#phase-1-claude-code-integration)
- Configuration management
- Authentication to Mnemonic

**Note:** Detailed CLI specifications (configuration precedence, caching behavior, etc.) will be documented in the CLI repository. For MVP testing, Mnemonic's REST API can be accessed using curl, Postman, or other HTTP clients.

### Mnemonic Server

- REST API endpoints (`/v1/api/route`, `/v1/api/patterns`, `/v1/api/agents`) -> [API Specification](../design/mnemonic_service/api-specification.md)
- Deterministic routing engine with priority-ordered rules -> [Routing Engine](../design/mnemonic_service/routing-engine.md)
- Four match types: keyword, regex, pattern (semantic), default
- In-memory rule caching (restart required to reload)
- Latency targets documented in routing engine design

### Data Storage

- PostgreSQL 15+ as source of truth with 7 migrations
- PGVector for embeddings (IVFFlat index, 1536 dimensions)
- Neo4j 5.x Community Edition for knowledge graph relationships
- Schema details -> Architecture and design docs (data architecture in progress)

### Pattern Enrichment

- Postgres-backed job queue with background worker -> [Pattern Processing](../design/mnemonic_service/pattern-processing.md)
- OpenAI text-embedding-3-small for embeddings
- OpenAI gpt-4o-mini for concept extraction
- Neo4j sync is best-effort (failures logged, processing continues)

### Observability (Stage 1)

- OpenTelemetry instrumentation via `otelx` package -> [Observability Implementation](../design/mnemonic_service/observability-implementation.md)
- Zerolog with automatic trace correlation
- Prometheus metrics emission via OTLP
- W3C Trace Context propagation
- Collection infrastructure is Post-MVP

### Deployment

- Local: Docker Compose -> [Deployment Architecture](../architecture/05-deployment-architecture.md)
- Production: Kubernetes (single Mnemonic pod initially)
- Independent CI/CD pipelines for application and migrations

## Out of Scope (Post-MVP)

### Phase 2 - Direct Anthropic API

- Direct API calls without Claude Code dependency -> [Architecture Overview - Phase 2](../architecture/00-overview.md#phase-2-direct-api-integration)

### Phase 3 - Authentication and Authorization

- Envoy proxy, OPA sidecar, OAuth2 device flow -> [Security Architecture](../architecture/06-security-architecture.md)

### Observability Stages 2-3

- Collection infrastructure (Collector, Prometheus, Loki, Jaeger) -> [Observability Architecture](../architecture/07-observability-architecture.md)
- Grafana dashboards, alerting, runbooks

### Advanced Features

- Redis distributed caching (required for multi-pod)
- Background rule refresh without restart
- Rate limiting enforcement (config exists, not enforced)
- HNSW vector index (for >100K patterns)
- Alternative embedding providers
- Dedicated enrichment processor
- Backup and recovery procedures
- Multi-tenant support
- CLI distribution via package managers

## Known Limitations

| Limitation                      | Impact                                | Workaround                                |
| ------------------------------- | ------------------------------------- | ----------------------------------------- |
| Rules require restart to reload | Changes not immediately effective     | Restart Mnemonic after modifications      |
| Single point of failure         | CLI cannot function without server    | No offline fallback in MVP                |
| Full prompts sent for routing   | Privacy consideration (not persisted) | Evaluate prompt sensitivity               |
| Neo4j sync is best-effort       | Graph may be temporarily inconsistent | Failures logged, processing continues     |
| No rate limiting enforcement    | Potential resource exhaustion         | Operational monitoring                    |
| Single pod deployment           | Limited availability                  | Horizontal scaling available but untested |
| No backup procedures            | Data loss risk                        | Manual database backups                   |

## Success Criteria

Functional requirements and quality attributes are defined in [Requirements](../architecture/01-requirements.md).

**Key metrics:**

- Mnemonic correctly routes requests to appropriate agents via REST API
- Patterns are retrievable via REST API
- Routing decisions are deterministic and reproducible
- Enriched context (agent + patterns) can be consumed by external clients

## Resolved Inconsistencies

| Issue                  | Resolution                                                                                     |
| ---------------------- | ---------------------------------------------------------------------------------------------- |
| Rate limiting default  | Changed to `enabled: false` in [configuration.md](../design/mnemonic_service/configuration.md) |
| Cache refresh settings | Comments clarify settings are IGNORED in MVP                                                   |
| Neo4j constraints      | Startup logs warnings if missing, does not fail                                                |
| CLI telemetry          | Entire design is Post-MVP                                                                      |
