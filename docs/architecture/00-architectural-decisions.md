# Architectural Decisions

[Back to Overview](README.md) | [Back to Project README](../../README.md)

## Table of Contents

- [Decision Record Format](#decision-record-format)
- [ADR-001: Team Knowledge Graph and Tooling Synchronization](#adr-001-team-knowledge-graph-and-tooling-synchronization) (ACTIVE)
- [ADR-002: MCP Protocol for Claude Code Integration](#adr-002-mcp-protocol-for-claude-code-integration) (ACTIVE)
- [ADR-003: Entity Storage with JSONB Document Model](#adr-003-entity-storage-with-jsonb-document-model) (ACTIVE)
- [ADR-004: Pattern Storage and Enrichment Pipeline](#adr-004-pattern-storage-and-enrichment-pipeline) (ACTIVE)
- [Decision Summary](#decision-summary)

## Decision Record Format

Each architectural decision follows this structure:

- **Context**: The situation and forces at play
- **Decision**: What we decided to do
- **Consequences**: The results of the decision, both positive and negative

## <a id="adr-001"></a>ADR-001: Team Knowledge Graph and Tooling Synchronization

**Date:** 2026-02-14
**Status:** Accepted

### Context

Teams using Claude Code develop inconsistent tooling over time. Each developer accumulates their own set of agents, skills, and commands at different versions, with no shared baseline. Alongside this, knowledge silos form — patterns and approaches discovered by one team member don't reliably reach others. The result is duplicated effort, divergent practices, and no persistent team memory.

Claude Code already handles orchestration well. The real problems are tooling drift and knowledge isolation.

### Decision

**Mnemonic is a team knowledge graph and tooling synchronization service.**

Core capabilities:

1. **Team knowledge graph**: Curated patterns with semantic search (PGVector) and knowledge relationships (Neo4j)
2. **Tooling synchronization**: Agents, skills, and commands synchronized across team members via MCP protocol
3. **User is the orchestrator**: Mnemonic provides memory and consistent tools; the user decides workflow

### Consequences

**Positive:**

- Architecture is focused on the actual team pain points: tooling drift and knowledge silos
- MCP protocol provides natural Claude Code integration without a separate CLI
- Manual orchestration is intentional and supported by team knowledge
- Single server with two listeners simplifies deployment

**Negative:**

- Pattern_agent_associations table retained to support "which agents use this pattern" queries
- Users must understand that Mnemonic provides knowledge and tools, not routing decisions

## <a id="adr-002"></a>ADR-002: MCP Protocol for Claude Code Integration

**Date:** 2026-02-15
**Status:** Accepted

### Context

Mnemonic needs a way to expose its knowledge graph and tooling catalog to Claude Code sessions. The integration must support read-only access — Claude Code queries patterns and retrieves agent/skill/command definitions, but does not write to Mnemonic directly. Admin operations (data loading, updates) still need a programmatic interface.

Options considered:

1. **MCP protocol**: Leverage Claude Code's native Model Context Protocol support
2. **Claude Code skills + shell scripts**: Wrapper scripts calling a REST API

Key considerations:

- Seamless Claude Code integration without a separate CLI repository
- Native protocol support vs. custom integration
- Read-only access pattern for Claude Code
- Admin operations still need a programmatic interface

### Decision

**Use MCP (Model Context Protocol) over Streamable HTTP for Claude Code integration.**

Architecture:

- **MCP Server** (`:8081`): Read-only access to patterns, agents, skills, commands
- **Admin REST API** (`:8080`): Write operations for data loading
- **Single server**: Two HTTP listeners on one Go server

MCP tools (3 pattern search tools):

- `search_patterns`: Semantic search over team knowledge graph
- `find_related_patterns`: Find patterns related to a given pattern
- `get_pattern`: Retrieve specific pattern by ID

### Consequences

**Positive:**

- No separate CLI repository needed
- Native Claude Code integration via MCP
- Read-only MCP pattern keeps Claude Code safe
- Admin REST API for data loading via curl/scripts
- Single server simplifies deployment

**Negative:**

- Must implement MCP server in Go (new protocol support)
- Two protocol surfaces to maintain (REST admin + MCP read-only)
- MCP is less mature than REST for debugging and tooling

## <a id="adr-003"></a>ADR-003: Entity Storage with JSONB Document Model

**Date:** 2026-02-15
**Status:** Accepted

### Context

Agents, skills, commands, and skill files are first-class entities in Mnemonic (see [ADR-001](#adr-001)). Each entity type has a different internal schema: agents have system prompts and allowed tools; skills have content and child files; commands have content. Traditional column-per-field modeling creates tight coupling between Go structs and database schema, requiring migrations for every field change.

### Decision

**Use a JSONB document model for entity definitions.** Each entity table follows a common pattern:

| Column       | Type            | Purpose                                                    |
| ------------ | --------------- | ---------------------------------------------------------- |
| `id`         | UUID PK         | Stable reference (entities may be renamed)                 |
| `name`       | VARCHAR UNIQUE  | Human-readable identifier, URL-safe (`^[a-z][a-z0-9-]*$`) |
| `definition` | JSONB NOT NULL  | Complete entity document (all fields)                      |
| `crc64`      | BIGINT NOT NULL | CRC-64/ISO checksum for change detection                   |
| `created_at` | TIMESTAMPTZ     | Row creation time                                          |
| `updated_at` | TIMESTAMPTZ     | Last modification time                                     |

**Name conventions:**

- Agents: max 64 chars, `^[a-z][a-z0-9-]*$`
- Skills: max 64 chars, `^[a-z]([a-z0-9](-[a-z0-9])*)*$` (stricter hyphen rules)
- Commands: max 255 chars

**CRC-64 change detection:** Computed in Go via `hash/crc64` with ISO polynomial. Stored as BIGINT. Enables efficient diff without comparing full JSONB documents.

**Skill files** use a variation: `skill_id` FK with CASCADE delete, plus `file_type` (script/reference/asset) and `filename` columns for direct lookup. The file content and metadata live in a `document` JSONB column.

### Consequences

**Positive:**

- Schema-agnostic: new fields in definitions require no migration
- Single source of truth: Go struct marshals directly to JSONB
- Efficient sync: CRC-64 comparison avoids deep document diffs

**Negative:**

- No column-level database constraints on definition content (application validates)
- Queries into definition fields require JSONB operators (acceptable; list queries use `name` column)
- Future option: pg_jsonschema extension for database-level JSONB validation if needed

## <a id="adr-004"></a>ADR-004: Pattern Storage and Enrichment Pipeline

**Date:** 2026-02-15
**Status:** Accepted

### Context

Patterns are the core knowledge artifacts in Mnemonic. Unlike entities (agents, skills, commands) which use JSONB documents, patterns require relational columns for vector search, enrichment tracking, and graph relationships. The enrichment pipeline must generate embeddings via an external API (OpenAI text-embedding-3-small) and extract entities for the Neo4j knowledge graph.

### Decision

**Patterns use relational columns (not JSONB) with async enrichment via a Postgres-backed job queue.**

**Pattern table design:**

- UUID primary key (patterns may be renamed; need stable reference)
- Separate `embedding` column (vector(1536)) for PGVector similarity search
- Enrichment status tracking (`pending` / `enriched` / `failed`) with error capture
- JSONB `tags` column for flexible categorization

**Enrichment pipeline:**

- Postgres-backed queue using `FOR UPDATE SKIP LOCKED` for safe concurrent processing
- Retry with exponential backoff for transient API failures (max 3 attempts)
- CASCADE delete: enrichment jobs auto-cleaned when pattern deleted
- Two-phase enrichment: (1) generate embedding via OpenAI API, (2) extract entities and create Neo4j relationships

**Why relational columns for patterns (not JSONB):**

- Vector similarity search requires a dedicated `embedding` column for PGVector indexing
- Enrichment status must be queryable for job processing (`WHERE enrichment_status = 'pending'`)
- Content size constraint (10KB max) enforced at database level via CHECK

### Consequences

**Positive:**

- PGVector index operates directly on the embedding column (no JSONB extraction)
- Enrichment queue requires no external message broker
- Concurrent enrichment workers are safe via `SKIP LOCKED`
- Pattern deletion cascades to jobs (no orphans)

**Negative:**

- Mnemonic calls an external embedding API (OpenAI) — adds an external dependency
- Async enrichment means patterns are not immediately searchable after creation
- Two databases to keep in sync (Postgres source of truth, Neo4j projection)

## Decision Summary

| Decision | Choice                       | Rationale                                                      | Status |
| -------- | ---------------------------- | -------------------------------------------------------------- | ------ |
| ADR-001  | Team knowledge graph         | Solve real problems: tooling drift and knowledge silos         | ACTIVE |
| ADR-002  | MCP protocol integration     | Native Claude Code integration, dual protocol architecture     | ACTIVE |
| ADR-003  | JSONB document model         | Schema-agnostic entity storage, CRC-64 change detection        | ACTIVE |
| ADR-004  | Pattern storage + enrichment | Relational columns for vectors, Postgres-backed async queue    | ACTIVE |

## Related Design Docs

- [Pivot API Specification](../design/2026-02-15-pivot-api-specification.md) - Current API design
- [Pattern Processing](../design/pattern-processing.md)
- [Configuration](../design/configuration.md)
- [Observability Implementation](../design/observability-implementation.md)

**Next:** [System Architecture](02-system-architecture.md)
