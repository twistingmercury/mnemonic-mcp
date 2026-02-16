# MVP Implementation Plan

## **IMPORTANT! NON-NEGOTIABLES**

Before the next phase can begin:

| Rule                                                                                                                                                                                 | Responsible Agent(s)                    |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | --------------------------------------- |
| Data migration tests (if applicable) have been written and pass and existing unit tests run and pass. Any issues fixed.                                                              | Data Architect                          |
| Unit tests (if applicable) have been written and pass and existing unit tests run and pass. Any issues fixed.                                                                        | Go sofware agent                        |
| Code analysis ran and passes: goimports, golangci-lint, govulncheck, gosec                                                                                                           | **The User**                            |
| ''                                                                                                                                                                                   | Code review agent                       |
| ''                                                                                                                                                                                   | Go software agent                       |
| Code review workflow has been ran; any finding have been `DISMISSED`, `DEFERRED`, or `FIXED: <short description>`. Deliverables have been against requirements and any issues fixed. | **The user**                            |
| ''                                                                                                                                                                                   | Software architect                      |
| ''                                                                                                                                                                                   | Go architect                            |
| ''                                                                                                                                                                                   | Code review agent.                      |
| The CI build has been ran locally and runs successfully.                                                                                                                             | **The user**                            |
| Update any docs as needed, i.e., README, CHANGELOG, architecture and design docs.                                                                                                    | Go software agent & documentation agent |
| Commited and pushed, PR created, re and merged back to develop                                                                                                                       | **The user**                            |

## High level plan: Pre-Pivot

| Phase | Step | Goal                                                                                                                              | Agent(s)             | Design Reference                                                                                                                                                                                                                                                                                          | Status   |
| ----- | ---- | --------------------------------------------------------------------------------------------------------------------------------- | -------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- |
| 1     |      | Get the CI build working on Git. That included creating the docker image and pushing it to the ghcr repo.                         | go devops agent      | [Deployment Architecture - Independent Deployment Pipelines](../architecture/05-deployment-architecture.md#independent-deployment-pipelines)                                                                                                                                                              | COMPLETE |
| 2     |      | Implement and unit test the configuration functionality.                                                                          | go software agent    | [Mnemonic Configuration](../design/configuration.md)                                                                                                                                                                                                                                                      | COMPLETE |
| 3     |      | Implement and unit test the observability functionality.                                                                          | go software agent    | [Observability Implementation](../design/observability-implementation.md), [Observability Architecture](../architecture/07-observability-architecture.md)                                                                                                                                                 | COMPLETE |
| 4     |      | Create Docker Compose configuration for local development (Mnemonic, PostgreSQL, Neo4j).                                          | go devops agent      | [Deployment Architecture - Minimal Deployment](../architecture/05-deployment-architecture.md#minimal-deployment)                                                                                                                                                                                          | COMPLETE |
| 5     | A    | Design agent schema and implement migrations (002_agents).                                                                        | data architect agent | [Data Architecture - PostgreSQL Schema](../architecture/08-data-architecture.md#postgresql-schema), [Data Storage - PostgreSQL Migrations](../design/data-storage.md#postgresql-migrations)                                                                                                               | COMPLETE |
|       | B    | Implement AgentRepository with unit tests.                                                                                        | go software agent    | [Data Storage - Repository Interfaces](../design/data-storage.md#repository-interfaces), [Data Storage - Connection Configuration](../design/data-storage.md#connection-configuration)                                                                                                                    | COMPLETE |
| 6     | A    | Design pattern schema with embedding column, implement migrations (001_extensions, 003_patterns, 004_pattern_agent_associations). | data architect agent | [Data Architecture - PostgreSQL Schema](../architecture/08-data-architecture.md#postgresql-schema), [Data Architecture - PGVector Configuration](../architecture/08-data-architecture.md#pgvector-configuration), [Data Storage - PostgreSQL Migrations](../design/data-storage.md#postgresql-migrations) | COMPLETE |
|       | B    | Implement PatternRepository (including FindSimilar with cosine similarity) with unit tests.                                       | go software agent    | [Data Storage - PatternRepository](../design/data-storage.md#patternrepository), [Data Storage - Similarity Search Queries](../design/data-storage.md#similarity-search-queries)                                                                                                                          | COMPLETE |
| 7     | A    | Design routing_rules and enrichment_jobs schemas, implement migrations (005, 006, 007).                                           | data architect agent | [Data Architecture - PostgreSQL Schema](../architecture/08-data-architecture.md#postgresql-schema), [Data Storage - PostgreSQL Migrations](../design/data-storage.md#postgresql-migrations)                                                                                                               | COMPLETE |
|       | B    | Implement routingrule.Repository and enrichmentjob.Repository with unit tests.                                                    | go software agent    | [Data Storage - Repository Interfaces](../design/data-storage.md#repository-interfaces), [Data Storage - EnrichmentJobRepository](../design/data-storage.md#enrichmentjobrepository)                                                                                                                      | COMPLETE |
| 8     | A    | Design Neo4j schema, implement constraints and indexes.                                                                           | data architect agent | [Data Architecture - Neo4j Graph Model](../architecture/08-data-architecture.md#neo4j-graph-model), [Data Storage - Neo4j Setup](../design/data-storage.md#neo4j-setup), [Data Storage - Schema Constraints](../design/data-storage.md#schema-constraints)                                                | COMPLETE |
|       | B    | Implement GraphRepository with unit tests.                                                                                        | go software agent    | [Data Storage - GraphRepository](../design/data-storage.md#graphrepository), [Data Storage - Graph Synchronization Queries](../design/data-storage.md#graph-synchronization-queries)                                                                                                                      | COMPLETE |
| 9     |      | Implement deterministic routing engine with priority-ordered rule evaluation.                                                     | go software agent    | Routing engine implementation (archived)                                                                                                                                                                                                                                                                  | COMPLETE |
| 10    |      | Implement keyword matcher (exact and substring matching).                                                                         | go software agent    | Routing engine implementation (archived)                                                                                                                                                                                                                                                                  | COMPLETE |
| 11    |      | Implement regex matcher (compiled pattern caching).                                                                               | go software agent    | Routing engine implementation (archived)                                                                                                                                                                                                                                                                  | COMPLETE |
| 12    |      | Implement pattern/semantic matcher (vector similarity with confidence scoring).                                                   | go software agent    | Routing engine implementation (archived)                                                                                                                                                                                                                                                                  | COMPLETE |
| 13    |      | Remove default fallback, implement explicit no-match signaling.                                                                   | go software agent    | Routing engine implementation (archived)                                                                                                                                                                                                                                                                  | COMPLETE |
| 14    |      | Implement in-memory rule cache (requires restart to reload as per MVP scope).                                                     | go software agent    | [Mnemonic Configuration - routing.cache](../design/configuration.md#configuration-file)                                                                                                                                                                                                                   | COMPLETE |

---

## High level plan: Post-Pivot

After completing Phase 14, a fundamental architectural pivot was made. The routing engine built in Phases 9-14 was found to solve a problem users don't actually have — the user is the orchestrator, not the software. Mnemonic's focus shifts from agent routing to team knowledge graph and tooling synchronization. See [Mnemonic Architectural Pivot](2026-02-14-mnemonic-pivot-knowledge-sync.md) for the full rationale. Phases 15-17 remove the routing infrastructure; Phases 18 onward build the knowledge and sync capabilities that replace it.

| Phase | Step | Goal                                                                                                           | Agent(s)            | Design Reference                                                                                                                                                                                            | Detail                                                     | Status |
| ----- | ---- | -------------------------------------------------------------------------------------------------------------- | ------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------- | ------ |
| 15    |      | Delete migration 005, trim migration 007 (remove routing indexes).                                             | data engineer       | [Go Architecture Plan - Migration Ordering](2026-02-15-go-architecture-plan.md#7-migration-ordering)                                                                                                        | [Detail](phase-15-migration-cleanup.md)                    |        |
| 16    |      | Create new migrations: 008 (drop routing_rules), 009 (alter agents add version), 010 (skills), 011 (commands). | data engineer       | [Go Architecture Plan - Section 9](2026-02-15-go-architecture-plan.md#9-new-package-design)                                                                                                                 | [Detail](phase-16-new-migrations.md)                       |        |
| 17    |      | Remove all routing Go packages, routing metrics, routing handlers, routing E2E tests.                          | go software agent   | [Go Architecture Plan - Package Disposition](2026-02-15-go-architecture-plan.md#3-package-disposition)                                                                                                      | [Detail](phase-17-remove-routing-code.md)                  |        |
| 18    |      | Configuration overhaul: two listener configs, remove routing config.                                           | go software agent   | [Go Architecture Plan - Configuration Changes](2026-02-15-go-architecture-plan.md#10-configuration-changes)                                                                                                 | [Detail](phase-18-configuration-overhaul.md)               |        |
| 19    |      | Add `version` field to agent model and repository, update tests.                                               | go software agent   | [Go Architecture Plan - Schema Changes](2026-02-15-go-architecture-plan.md#8-schema-changes-for-existing-tables)                                                                                            | [Detail](phase-19-agent-repo-version.md)                   |        |
| 20    |      | Implement skill repository (model, errors, interface, pgx impl, tests).                                        | go software agent   | [Go Architecture Plan - Skill Repository](2026-02-15-go-architecture-plan.md#91-skill-repository)                                                                                                           | [Detail](phase-20-skill-repository.md)                     |        |
| 21    |      | Implement command repository (model, errors, interface, pgx impl, tests).                                      | go software agent   | [Go Architecture Plan - Command Repository](2026-02-15-go-architecture-plan.md#92-command-repository)                                                                                                       | [Detail](phase-21-command-repository.md)                   |        |
| 22    |      | Rewrite agent and pattern REST handlers with dependency injection, unit tests.                                 | go software agent   | [Go Architecture Plan - Handler Packages](2026-02-15-go-architecture-plan.md#93-handler-packages), [API Specification - Agents](../design/2026-02-15-pivot-api-specification.md#24-agents)                  | [Detail](phase-22-rest-handlers-agents-patterns.md)        |        |
| 23    |      | Implement skill, command, and search REST handlers, router setup, unit tests.                                  | go software agent   | [Go Architecture Plan - Handler Packages](2026-02-15-go-architecture-plan.md#93-handler-packages), [API Specification - Skills/Commands](../design/2026-02-15-pivot-api-specification.md#26-skills)         | [Detail](phase-23-rest-handlers-skills-commands-search.md) |        |
| 24    |      | Implement MCP server with all read-only tools, unit tests.                                                     | go software agent   | [Go Architecture Plan - MCP Server Design](2026-02-15-go-architecture-plan.md#5-mcp-server-design), [API Specification - MCP Tools](../design/2026-02-15-pivot-api-specification.md#3-mcp-tool-definitions) | [Detail](phase-24-mcp-server.md)                           |        |
| 25    |      | Rewrite server lifecycle for two listeners, new main entrypoint, health checks.                                | go software agent   | [Go Architecture Plan - Package Structure](2026-02-15-go-architecture-plan.md#6-package-structure)                                                                                                          | [Detail](phase-25-server-lifecycle.md)                     |        |
| 26    |      | E2E tests for Admin API and MCP server.                                                                        | go e2e test agent   | [Go Architecture Plan - E2E Tests](2026-02-15-go-architecture-plan.md#11-phased-implementation-plan)                                                                                                        | [Detail](phase-26-e2e-tests.md)                            |        |
| 27    |      | Update Dockerfile, docker-compose, CI/CD for new entrypoint and two ports.                                     | go devops agent     | [Go Architecture Plan - Deployment](2026-02-15-go-architecture-plan.md#11-phased-implementation-plan)                                                                                                       | [Detail](phase-27-deployment-update.md)                    |        |
| 28    |      | Documentation: ADR for pivot, update architecture docs, CHANGELOG.                  | documentation agent | N/A (documentation task)                                                                                                                                                                                    | [Detail](phase-28-documentation.md)                        |        |

---

## Phase Dependencies

| Phase                              | Depends On | Reason                                                                                      |
| ---------------------------------- | ---------- | ------------------------------------------------------------------------------------------- |
| 3 (Observability)                  | 1, 2       | Observability uses configuration for setup                                                  |
| 4 (Docker Compose)                 | 1          | Requires Docker image from CI/CD                                                            |
| 5 (Agent Slice)                    | 4          | Agent schema/repository need running PostgreSQL                                             |
| 6 (Pattern Slice)                  | 4          | Pattern schema/repository (with PGVector) need running PostgreSQL                           |
| 7 (Rules & Jobs Slice)             | 4          | Rules/jobs schemas and repositories need running PostgreSQL                                 |
| 8 (Neo4j Slice)                    | 4          | Neo4j schema/repository need running Neo4j                                                  |
| 9 (Routing Engine)                 | 2, 5, 6, 7 | Routing engine uses configuration and all PostgreSQL repositories (agents, patterns, rules) |
| 10 (Keyword Matcher)               | 9          | Keyword matcher is part of routing engine implementation                                    |
| 11 (Regex Matcher)                 | 9          | Regex matcher is part of routing engine implementation                                      |
| 12 (Pattern Matcher)               | 9          | Pattern matcher is part of routing engine implementation                                    |
| 13 (Default Matcher)               | 9          | Default matcher is part of routing engine implementation                                    |
| 14 (Rule Cache)                    | 9          | Rule cache is part of routing engine implementation                                         |
| 15 (Migration Cleanup)             | 14         | First pivot phase; depends on all pre-pivot phases being complete                           |
| 16 (New Migrations)                | 15         | New migrations require routing_rules migration to be removed first                          |
| 17 (Remove Routing Code)           | 15         | Safe to delete Go code after DB migration is removed                                        |
| 18 (Configuration Overhaul)        | 17         | Routing config references must be deleted before restructuring config                       |
| 19 (Agent Repo Modification)       | 16, 17     | Requires migration 008 (version column) and routing code removed                            |
| 20 (Skill Repository)              | 16         | Requires migration 009 (skills table)                                                       |
| 21 (Command Repository)            | 16         | Requires migration 010 (commands table)                                                     |
| 22 (Agent & Pattern Handlers)      | 18, 19     | Requires new config structure and updated agent repo                                        |
| 23 (Skill/Command/Search Handlers) | 20, 21, 22 | Requires skill/command repos and handler pattern from Phase 22                              |
| 24 (MCP Server)                    | 19, 20, 21 | Requires all repositories to be implemented                                                 |
| 25 (Server Lifecycle)              | 18, 23, 24 | Requires config, all handlers, and MCP server                                               |
| 26 (E2E Tests)                     | 25         | Requires the full server running with both listeners                                        |
| 27 (Deployment Update)             | 25         | Requires the new entrypoint and two-port architecture                                       |
| 28 (Documentation)                 | 26         | All implementation must be complete before final docs                                       |

## Success Criteria Reference

From the pivot proposal and architecture documents:

- Mnemonic serves as a team knowledge graph and tooling synchronization server
- REST Admin API (port 8080) provides CRUD for agents, patterns, skills, and commands
- MCP Endpoint (port 8081) provides read-only tools for Claude Code integration
- Claude Code can connect via MCP and discover all tools (`tools/list`)
- Claude Code can search patterns, retrieve agent definitions, and sync skill/command definitions
- All handler stubs replaced with working implementations backed by repositories
- Skills and commands tables created with full CRUD support
- Agent `version` field supported across model, repository, and API
- Pattern-agent associations retained and functional
- Health checks cover both Postgres and Neo4j
- Graceful shutdown handles both listeners
- All E2E tests pass against Docker Compose environment

## Known Limitations to Track

| Limitation                                 | Phase Impacted | Mitigation                                                                    | Source Documentation                                                                                    |
| ------------------------------------------ | -------------- | ----------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------- |
| Semantic search requires OpenAI embeddings | 23, 24         | Fall back to full-text search on name/description until enrichment pipeline   | [Go Architecture Plan - Open Decisions](2026-02-15-go-architecture-plan.md#12-open-decisions)           |
| Enrichment pipeline not yet built          | N/A (post-MVP) | Patterns created with `enrichment_status: pending`; graph context unavailable | [Go Architecture Plan - Summary](2026-02-15-go-architecture-plan.md#summary)                            |
| Single point of failure                    | 25             | Document operational requirements                                             | [Architectural Decisions](../architecture/02-architectural-decisions.md)                                |
| Neo4j sync is best-effort                  | 8              | Failures logged, processing continues                                         | [Data Architecture](../architecture/08-data-architecture.md), [Data Storage](../design/data-storage.md) |
| No rate limiting enforcement               | 22-23          | Operational monitoring                                                        | [API Specification](../design/2026-02-15-pivot-api-specification.md)                                    |
| No authentication in local MVP             | N/A            | Local MVP runs without authentication; production auth is post-MVP            | [Security Architecture](../architecture/06-security-architecture.md)                                    |
| Multi-file skills not supported            | 20             | Store only `instructions.md` content; multi-file is post-MVP                  | [Go Architecture Plan - Open Decisions](2026-02-15-go-architecture-plan.md#126-skill-content-model)     |
| MCP session management basic               | 24             | 30-minute default timeout; no persistent sessions across restarts             | [API Specification - MCP](../design/2026-02-15-pivot-api-specification.md#31-mcp-over-streamable-http)  |

## Design Gaps

### Resolved

| Gap                | Resolution                                                                                                      |
| ------------------ | --------------------------------------------------------------------------------------------------------------- |
| E2E test scenarios | Test scenarios documented in `src/mnemonic/tests/e2e/*_test.go`; implementation depends on Phases 4-8 and 16-18 |
| Routing engine     | **SUPERSEDED** by pivot. Routing code removed in Phase 17. Replaced by knowledge graph + MCP tools.             |

### Deferred to Post-MVP

| Gap                            | Reason                                                                                           |
| ------------------------------ | ------------------------------------------------------------------------------------------------ |
| API key authentication         | Local MVP runs in trusted environment. Production authentication (Envoy proxy) is post-MVP.      |
| Kubernetes deployment          | Production deployment is post-MVP. Local MVP uses Docker Compose only.                           |
| Migrations CI/CD pipeline      | Cloud database migrations pipeline is post-MVP. Local MVP applies migrations via Docker Compose. |
| OpenAI embedding integration   | Enrichment pipeline (embedding + concept extraction + Neo4j sync) is post-MVP.                   |
| MCP resource subscriptions     | Push-based sync is a post-MVP optimization. MVP uses pull-based polling via `get_sync_manifest` tool.  |
| Multi-file skill content       | Requires JSONB file manifest or `skill_files` table. Deferred until skill format stabilizes.     |
| Optimistic concurrency (ETags) | Single-admin MVP does not need concurrent edit protection.                                       |

### Local MVP Notes

- No authentication required (trusted local environment)
- Migrations applied automatically by Docker Compose
- All infrastructure runs locally via `docker-compose up`
- Data loaded via `curl` against the REST Admin API
- Claude Code connects to MCP endpoint at `http://localhost:8081/mcp`
