# Data Storage Implementation

[Back to Architecture Overview](../../architecture/README.md) | [Back to Project README](../../../README.md)

## Table of Contents

- [Overview](#overview)
- [JSONB Document Model](#jsonb-document-model)
  - [Design Rationale](#design-rationale)
  - [Common Table Shape](#common-table-shape)
  - [CRC64 Change Detection](#crc64-change-detection)
  - [JSONB Contents by Entity](#jsonb-contents-by-entity)
  - [JSONB Indexing Strategy](#jsonb-indexing-strategy)
  - [Application-Layer Validation](#application-layer-validation)
- [PostgreSQL Migrations](#postgresql-migrations)
  - [Migration File Structure](#migration-file-structure)
  - [Migration 001: Extensions](#migration-001-extensions)
  - [Migration 002: Agents Table (Pre-Pivot)](#migration-002-agents-table-pre-pivot)
  - [Migration 003: Patterns Table](#migration-003-patterns-table)
  - [Migration 004: Pattern-Agent Associations](#migration-004-pattern-agent-associations)
  - [Migration 005: Routing Rules Table (Dropped)](#migration-005-routing-rules-table-dropped)
  - [Migration 006: Enrichment Jobs Table](#migration-006-enrichment-jobs-table)
  - [Migration 007: Performance Indexes](#migration-007-performance-indexes)
  - [Migration 008: Drop Routing Rules](#migration-008-drop-routing-rules)
  - [Migration 009: Migrate Agents to JSONB Document Model](#migration-009-migrate-agents-to-jsonb-document-model)
  - [Migration 010: Create Skills Table](#migration-010-create-skills-table)
  - [Migration 011: Create Commands Table](#migration-011-create-commands-table)
  - [Migration 012: Create Skill Files Table](#migration-012-create-skill-files-table)
- [PGVector Configuration](#pgvector-configuration)
  - [Index Selection Guidelines](#index-selection-guidelines)
  - [Similarity Search Queries](#similarity-search-queries)
  - [Index Maintenance](#index-maintenance)
- [Neo4j Setup](#neo4j-setup)
  - [Schema Constraints](#schema-constraints)
  - [Index Configuration](#index-configuration)
  - [Node Creation Patterns](#node-creation-patterns)
  - [Relationship Definitions](#relationship-definitions)
  - [Graph Synchronization Queries](#graph-synchronization-queries)
- [Repository Interfaces](#repository-interfaces)
  - [AgentRepository](#agentrepository)
  - [PatternRepository](#patternrepository)
  - [SkillRepository](#skillrepository)
  - [CommandRepository](#commandrepository)
  - [SkillFileRepository](#skillfilerepository)
  - [EnrichmentJob Repository](#enrichmentjob-repository)
  - [GraphRepository](#graphrepository)
- [Connection Configuration](#connection-configuration)
  - [PostgreSQL Connection](#postgresql-connection)
  - [Neo4j Connection](#neo4j-connection)
  - [Connection String Formats](#connection-string-formats)
- [References](#references)

## Overview

[Table of Contents](#table-of-contents)

> **Architecture Reference:** [Data Architecture](../../architecture/04-data-architecture.md) | [System Architecture - Mnemonic](../../architecture/02-system-architecture.md#mnemonic)

This document provides implementation details for the Mnemonic data storage layer. It translates the architectural specifications from [Data Architecture](../../architecture/04-data-architecture.md) into concrete SQL migrations, Cypher queries, and Go interface definitions.

**Implementation Scope:**

| Component | Technology | Purpose |
|-----------|------------|---------|
| Relational Storage | PostgreSQL 15+ | Document tables (JSONB), patterns, jobs |
| Vector Storage | PGVector extension | Pattern embeddings for semantic search |
| Graph Storage | Neo4j 5.x | Knowledge graph relationships |
| Migrations | golang-migrate | Schema versioning and deployment |

**Post-Pivot Changes (JSONB Document Model):**

- Migrated `agents` table from decomposed columns to JSONB document model
- Added `crc64` column to all document tables for change detection
- Created `skills` table using JSONB document model
- Created `commands` table using JSONB document model
- Created `skill_files` table for skill child resources
- Removed `routing_rules` table (dropped in migration 008)
- Neo4j is now required (no longer optional)

**Tables Unchanged by the Pivot:**

- `patterns` -- has enrichment workflow, pgvector embeddings, graph context; these require relational columns
- `pattern_agent_associations` -- join table, kept for agent-scoped pattern filtering
- `enrichment_jobs` -- background processing queue

**Deployment Independence:**

Database migrations and application code are versioned and deployed independently. Migrations have their own CI/CD pipeline that triggers only on changes to the `migrations/` directory. This enables:

- Logic bug fixes in Go without database deployment
- Schema changes without rebuilding application containers
- Forward-compatible migrations for zero-downtime deployments

## JSONB Document Model

[Table of Contents](#table-of-contents)

### Design Rationale

Agents, skills, commands, and skill files are markdown documents synced to disk. Their field sets evolve as the Claude Code Agent Skills spec evolves. Storing each field as an individual relational column creates migration churn every time a field is added, renamed, or removed.

The JSONB document model stores the full document as a single JSONB column. Only the fields required for database-level operations (lookup key, change detection, audit timestamps) are promoted to top-level columns. Everything else lives inside the JSONB document.

**Benefits:**

- **No migration churn.** Adding a field to the agent spec requires only an application change, not a database migration.
- **Spec alignment.** The JSONB column mirrors what the API returns and what the sync protocol transmits. No impedance mismatch between storage and wire format.
- **Simpler repository code.** One `definition JSONB` column replaces five or more individual columns. Reads and writes are straightforward marshal/unmarshal operations.

**Trade-offs:**

- **No column-level constraints on JSONB contents.** Field validation (max lengths, required fields, allowed values) is enforced by the application, not by CHECK constraints on individual columns.
- **GIN indexes required for JSONB queries.** Querying inside the document (tag filtering, field searches) requires GIN indexes instead of simple btree indexes.

### Common Table Shape

All document tables (agents, skills, commands) share the same column structure:

| Column | Type | Constraints | Purpose |
|--------|------|-------------|---------|
| `id` | UUID | PK, DEFAULT gen_random_uuid() | Internal identifier |
| `name` | VARCHAR | UNIQUE NOT NULL | Lookup key; the one field always queried by |
| `definition` | JSONB | NOT NULL | Complete document (all entity fields) |
| `crc64` | BIGINT | NOT NULL | CRC-64 checksum of serialized JSONB, for change detection |
| `created_at` | TIMESTAMPTZ | NOT NULL, DEFAULT now() | DB sets on INSERT |
| `updated_at` | TIMESTAMPTZ | NOT NULL, DEFAULT now() | App updates on UPDATE |

The `skill_files` table follows the same pattern with `document` instead of `definition`, and adds `skill_id`, `file_type`, and `filename` columns for the composite lookup key.

### CRC64 Change Detection

Each document table includes a `crc64` column storing a CRC-64 checksum of the JSONB content.

**Computation:**

- Computed server-side (in the Go application) on every INSERT and UPDATE.
- Input: the serialized JSONB content in canonical form (deterministic key ordering, no extra whitespace). Go's `encoding/json` produces deterministic output for the same struct, but the application should use a canonical serialization function to guarantee consistency.
- Algorithm: CRC-64 with ISO polynomial (matching Go's `hash/crc64` package with `crc64.MakeTable(crc64.ISO)`).
- Output: a 64-bit unsigned integer, stored as PostgreSQL BIGINT (64-bit signed). Go handles the uint64-to-int64 conversion.

**Usage in the sync protocol:**

- The `get_sync_manifest` MCP tool returns per-entity CRC64 values.
- The sync client compares local CRC64 values against the manifest to determine which entities have changed.
- A collection-level version hash can be derived from individual CRC64 values (e.g., XOR of all entity CRC64s for a given collection).

**Why CRC-64 and not SHA-256:**

CRC-64 is fast, fits in a single BIGINT column, and provides sufficient collision resistance for change detection (not security). The sync protocol uses it to answer "has this document changed?" not "is this document authentic?"

### JSONB Contents by Entity

#### Agents definition JSONB

The `definition` column stores the complete agent specification:

```json
{
  "description": "Implements Go code: functions, packages, tests",
  "system_prompt": "You are an expert Go engineer...",
  "model": "sonnet",
  "allowed_tools": ["Read", "Write", "Edit", "Bash"],
  "version": "1.2.0"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `description` | string | yes | Short summary of the agent's purpose |
| `system_prompt` | string | yes | Full system prompt content |
| `model` | string | yes | Claude model: sonnet, opus, haiku |
| `allowed_tools` | string[] | yes | MCP tool names this agent can use |
| `version` | string | yes | Semantic version of the definition |

#### Skills definition JSONB

Aligned with the [Claude Code Agent Skills spec](https://docs.anthropic.com/en/docs/agents-and-tools/claude-code/skills) frontmatter fields:

```json
{
  "description": "Synchronize agents, skills, and commands from Mnemonic",
  "content": "---\nname: mnemonic-sync\n---\n\nYou are synchronizing...",
  "tags": ["sync", "infrastructure"],
  "license": "MIT",
  "compatibility": "Claude Code 1.0+",
  "metadata": {"author": "team-platform"},
  "allowed_tools": ["Read", "Write", "Bash"],
  "version": "1.0.0"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `description` | string | yes | Short summary of the skill |
| `content` | string | yes | Full skill markdown content |
| `tags` | string[] | no | Categorization tags |
| `license` | string | no | License identifier (e.g., MIT) |
| `compatibility` | string | no | Compatible Claude Code versions |
| `metadata` | object | no | Arbitrary key-value metadata |
| `allowed_tools` | string[] | no | MCP tool names this skill can use |
| `version` | string | yes | Semantic version of the definition |

#### Commands definition JSONB

```json
{
  "description": "Initialize a Claude Code session",
  "content": "Load the project context...",
  "tags": ["initialization"],
  "version": "1.0.0"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `description` | string | yes | Short summary of the command |
| `content` | string | yes | Command content/instructions |
| `tags` | string[] | no | Categorization tags |
| `version` | string | yes | Semantic version of the definition |

#### Skill files document JSONB

```json
{
  "content_type": "text/x-python",
  "content": "#!/usr/bin/env python3\nimport sys...",
  "encoding": "utf-8",
  "size": 2048
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `content_type` | string | yes | MIME type of the file content |
| `content` | string | yes | File content (text or base64-encoded binary) |
| `encoding` | string | yes | Content encoding: utf-8 or base64 |
| `size` | integer | yes | Decoded size in bytes |

### JSONB Indexing Strategy

Document tables use GIN indexes on the `definition` (or `document`) column to support queries that filter by JSONB contents.

**Primary use case: tag filtering.**

Skills and commands support tag-based filtering. The GIN index enables efficient `@>` (contains) queries:

```sql
-- Find skills tagged with "sync"
SELECT id, name, definition
FROM skills
WHERE definition @> '{"tags": ["sync"]}'::jsonb;
```

**Index definitions** (created in migration 007 or respective table migrations):

```sql
-- GIN indexes on definition columns for tag filtering
CREATE INDEX idx_skills_definition ON skills USING GIN (definition);
CREATE INDEX idx_commands_definition ON commands USING GIN (definition);
```

**Path-specific GIN indexes** (alternative, more selective):

```sql
-- Index only the tags path within definition
CREATE INDEX idx_skills_definition_tags ON skills USING GIN ((definition -> 'tags'));
CREATE INDEX idx_commands_definition_tags ON commands USING GIN ((definition -> 'tags'));
```

The path-specific indexes are smaller and faster for tag-only queries. Use full-column GIN indexes if queries need to filter on other JSONB fields.

### Application-Layer Validation

Because JSONB contents are not constrained at the database level, the application enforces all field validation:

| Entity | Field | Constraint | Enforced By |
|--------|-------|-----------|-------------|
| Agent | name | max 64 chars, `^[a-z][a-z0-9-]*$` | DB (VARCHAR + CHECK) |
| Agent | definition.description | max 500 chars | Application |
| Agent | definition.system_prompt | max 50KB | Application |
| Agent | definition.model | one of: sonnet, opus, haiku | Application |
| Agent | definition.allowed_tools | must be string array | Application |
| Agent | definition.version | required, semver format | Application |
| Skill | name | max 64 chars, `^[a-z][a-z0-9-]*$` | DB (VARCHAR + CHECK) |
| Skill | definition.description | max 1024 chars | Application |
| Skill | definition.content | max 512KB | Application |
| Skill | definition.version | required, semver format | Application |
| Command | name | max 255 chars | DB (VARCHAR) |
| Command | definition.description | max 500 chars | Application |
| Command | definition.content | max 50KB | Application |
| Command | definition.version | required, semver format | Application |
| Skill File | filename | max 255 chars | DB (VARCHAR) |
| Skill File | file_type | one of: script, reference, asset | DB (CHECK) |
| Skill File | document.content | max 1MB | Application |

The database enforces only structural constraints (primary keys, uniqueness, foreign keys, the `name` column format). Content-level validation belongs to the application.

**Future Consideration:**

For database-level JSONB schema validation, consider the [pg_jsonschema](https://github.com/supabase/pg_jsonschema) PostgreSQL extension. This extension enables CHECK constraints that validate JSONB documents against JSON Schema specifications, providing an alternative to application-only validation.

## PostgreSQL Migrations

[Table of Contents](#table-of-contents)

> **Architecture Reference:** [Data Architecture - Migration Strategy](../../architecture/04-data-architecture.md#migration-strategy)

### Migration File Structure

All PostgreSQL migrations follow the golang-migrate convention:

```text
src/mnemonic/
└── migrations/
    └── postgres/
        ├── 001_extensions.up.sql
        ├── 001_extensions.down.sql
        ├── 002_create_agents.up.sql
        ├── 002_create_agents.down.sql
        ├── 003_create_patterns.up.sql
        ├── 003_create_patterns.down.sql
        ├── 004_create_pattern_agent_associations.up.sql
        ├── 004_create_pattern_agent_associations.down.sql
        ├── 005_create_routing_rules.up.sql
        ├── 005_create_routing_rules.down.sql
        ├── 006_create_enrichment_jobs.up.sql
        ├── 006_create_enrichment_jobs.down.sql
        ├── 007_create_performance_indexes.up.sql
        ├── 007_create_performance_indexes.down.sql
        ├── 008_drop_routing_rules.up.sql
        ├── 008_drop_routing_rules.down.sql
        ├── 009_migrate_agents_to_jsonb.up.sql
        ├── 009_migrate_agents_to_jsonb.down.sql
        ├── 010_create_skills.up.sql
        ├── 010_create_skills.down.sql
        ├── 011_create_commands.up.sql
        ├── 011_create_commands.down.sql
        ├── 012_create_skill_files.up.sql
        └── 012_create_skill_files.down.sql
```

**Post-Pivot Migrations:**

Migrations 008-012 implement the pivot from routing to knowledge sync with the JSONB document model:

- **008**: Drops `routing_rules` table
- **009**: Migrates `agents` to JSONB document model (moves column data into `definition` JSONB, adds `crc64`, drops individual columns)
- **010**: Creates `skills` table (JSONB document model)
- **011**: Creates `commands` table (JSONB document model)
- **012**: Creates `skill_files` table (JSONB document model)

**Running Migrations:**

```bash
# Apply all pending migrations
migrate -path src/mnemonic/migrations/postgres -database "$DATABASE_URL" up

# Rollback last migration
migrate -path src/mnemonic/migrations/postgres -database "$DATABASE_URL" down 1

# Check current version
migrate -path src/mnemonic/migrations/postgres -database "$DATABASE_URL" version
```

### Migration 001: Extensions

**Purpose:** Enable required PostgreSQL extensions.

```sql
-- src/mnemonic/migrations/postgres/001_extensions.up.sql
-- Enables required extensions
-- Part of Mnemonic MVP

-- Enable UUID generation (if not using gen_random_uuid)
create extension if not exists "uuid-ossp";

-- Enable vector operations for embeddings
create extension if not exists vector;
```

```sql
-- src/mnemonic/migrations/postgres/001_extensions.down.sql
-- Extensions are not dropped to avoid breaking other schemas
-- drop extension if exists vector;
-- drop extension if exists "uuid-ossp";
```

**Note:** Migration 001 no longer creates the `update_updated_at()` trigger function. Per the storage-only database philosophy, `updated_at` management is the application's responsibility. The application sets `updated_at = now()` on every UPDATE.

### Migration 002: Agents Table (Pre-Pivot)

**Purpose:** Create the agents table with decomposed columns (pre-pivot schema).

**Note:** This migration was created pre-pivot. Migration 009 replaces the decomposed columns with the JSONB document model.

```sql
-- src/mnemonic/migrations/postgres/002_create_agents.up.sql
-- Creates the agents table for storing agent definitions
-- Part of Mnemonic MVP (pre-pivot schema)

create table if not exists agents (
    -- Primary key: lowercase-with-hyphens format, URL-safe
    name varchar(64) primary key,

    -- Agent metadata
    description varchar(500) not null,

    -- System prompt content (up to 50KB)
    system_prompt text not null,

    -- Model preference: sonnet, opus, haiku
    model varchar(20) not null default 'sonnet',

    -- Allowed MCP tools (JSON array of tool names)
    allowed_tools jsonb not null default '[]'::jsonb,

    -- Keywords for fast routing (denormalized from routing_rules)
    -- NOTE: DEPRECATED post-pivot. Dropped in migration 009.
    routing_keywords jsonb not null default '[]'::jsonb,

    -- Audit timestamps
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),

    -- Constraints
    constraint agents_name_format
        check (name ~ '^[a-z][a-z0-9-]*$'),
    constraint agents_model_valid
        check (model in ('sonnet', 'opus', 'haiku')),
    constraint agents_system_prompt_length
        check (length(system_prompt) <= 51200),
    constraint agents_allowed_tools_array
        check (jsonb_typeof(allowed_tools) = 'array'),
    constraint agents_routing_keywords_array
        check (jsonb_typeof(routing_keywords) = 'array')
);

comment on table agents is 'Agent definitions for team tooling synchronization';
comment on column agents.name is 'Unique identifier, lowercase-with-hyphens format';
```

```sql
-- src/mnemonic/migrations/postgres/002_create_agents.down.sql
drop table if exists agents;
```

### Migration 003: Patterns Table

**Purpose:** Create the patterns table with PGVector embedding column for semantic search.

**Note:** This table is NOT affected by the JSONB document model pivot. Patterns have enrichment status, graph context, and pgvector embeddings that require relational columns.

```sql
-- src/mnemonic/migrations/postgres/003_create_patterns.up.sql
-- Creates the patterns table with vector embeddings
-- Part of Mnemonic MVP

create table if not exists patterns (
    -- UUID primary key for stable references (patterns may be renamed)
    id uuid primary key default gen_random_uuid(),

    -- Pattern metadata
    name varchar(128) not null,
    description varchar(500),

    -- Pattern content (up to 10KB)
    content text not null,

    -- Categorization tags (JSON array)
    tags jsonb not null default '[]'::jsonb,

    -- Vector embedding for semantic search (1536 dimensions for text-embedding-3-small)
    embedding vector(1536),

    -- Enrichment processing state
    enrichment_status varchar(20) not null default 'pending',
    enrichment_error text,
    enriched_at timestamptz,

    -- Audit timestamps
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),

    -- Constraints
    constraint patterns_name_unique unique (name),
    constraint patterns_content_length
        check (length(content) <= 10240),
    constraint patterns_tags_array
        check (jsonb_typeof(tags) = 'array'),
    constraint patterns_enrichment_status_valid
        check (enrichment_status in ('pending', 'enriched', 'failed'))
);

comment on table patterns is 'Reusable context patterns for prompt enrichment';
comment on column patterns.embedding is 'Vector embedding (1536d) for semantic similarity search';
comment on column patterns.enrichment_status is 'Processing state: pending, enriched, or failed';
```

```sql
-- src/mnemonic/migrations/postgres/003_create_patterns.down.sql
drop table if exists patterns;
```

### Migration 004: Pattern-Agent Associations

**Purpose:** Create the many-to-many association table between patterns and agents with relevance scores.

```sql
-- src/mnemonic/migrations/postgres/004_create_pattern_agent_associations.up.sql
-- Creates the pattern-agent association table
-- Part of Mnemonic MVP

create table if not exists pattern_agent_associations (
    -- Composite primary key
    pattern_id uuid not null,
    agent_name varchar(64) not null,

    -- Relevance score (0.0 to 1.0)
    relevance double precision not null,

    -- Foreign keys
    constraint fk_pattern_agent_assoc_pattern
        foreign key (pattern_id) references patterns(id) on delete cascade,
    constraint fk_pattern_agent_assoc_agent
        foreign key (agent_name) references agents(name) on delete cascade,

    -- Primary key
    primary key (pattern_id, agent_name),

    -- Constraints
    constraint pattern_agent_assoc_relevance_range
        check (relevance >= 0 and relevance <= 1)
);

-- Indexes for foreign key lookups
create index idx_pattern_agent_assoc_pattern
    on pattern_agent_associations(pattern_id);
create index idx_pattern_agent_assoc_agent
    on pattern_agent_associations(agent_name);

comment on table pattern_agent_associations is
    'Many-to-many relationship between patterns and agents with relevance scores';
```

```sql
-- src/mnemonic/migrations/postgres/004_create_pattern_agent_associations.down.sql
drop index if exists idx_pattern_agent_assoc_agent;
drop index if exists idx_pattern_agent_assoc_pattern;
drop table if exists pattern_agent_associations;
```

### Migration 005: Routing Rules Table (Dropped)

**Purpose:** Create the routing rules table for prompt-to-agent matching.

**Note:** This table is created in migration 005 and dropped in migration 008 (post-pivot). See migration 008 for the drop statement. The full creation SQL is retained here for migration rollback purposes.

```sql
-- src/mnemonic/migrations/postgres/005_create_routing_rules.up.sql
-- Creates the routing rules table
-- REMOVED: This table is dropped in migration 008 (pivot to knowledge sync)

create table if not exists routing_rules (
    id uuid primary key default gen_random_uuid(),
    name varchar(128) not null,
    priority integer not null,
    agent_name varchar(64) not null,
    match_type varchar(20) not null,
    match_config jsonb not null,
    enabled boolean not null default true,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),

    constraint fk_routing_rules_agent
        foreign key (agent_name) references agents(name) on delete restrict,
    constraint routing_rules_name_unique unique (name),
    constraint routing_rules_priority_range
        check (priority >= 0 and priority <= 1000),
    constraint routing_rules_match_type_valid
        check (match_type in ('keyword', 'regex', 'pattern', 'default')),
    constraint routing_rules_match_config_valid check (
        (match_type = 'keyword' and match_config ? 'keywords' and match_config ? 'match_mode') or
        (match_type = 'regex' and match_config ? 'pattern') or
        (match_type = 'pattern' and match_config ? 'pattern_ids') or
        (match_type = 'default')
    )
);

create index idx_routing_rules_agent on routing_rules(agent_name);
```

```sql
-- src/mnemonic/migrations/postgres/005_create_routing_rules.down.sql
drop index if exists idx_routing_rules_agent;
drop table if exists routing_rules;
```

### Migration 006: Enrichment Jobs Table

**Purpose:** Create the enrichment jobs queue table for background pattern processing.

```sql
-- src/mnemonic/migrations/postgres/006_create_enrichment_jobs.up.sql
-- Creates the enrichment jobs queue table
-- Part of Mnemonic MVP

create table if not exists enrichment_jobs (
    -- UUID primary key
    id uuid primary key default gen_random_uuid(),

    -- Reference to pattern being enriched
    pattern_id uuid not null,

    -- Job processing state
    status varchar(20) not null default 'pending',

    -- Retry tracking
    attempts integer not null default 0,
    max_attempts integer not null default 3,

    -- Error information
    last_error text,

    -- Scheduling and timing
    scheduled_for timestamptz not null default now(),
    started_at timestamptz,
    completed_at timestamptz,

    -- Audit timestamps
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),

    -- Foreign key with cascade delete
    constraint fk_enrichment_jobs_pattern
        foreign key (pattern_id) references patterns(id) on delete cascade,

    -- Constraints
    constraint enrichment_jobs_status_valid
        check (status in ('pending', 'processing', 'completed', 'failed')),
    constraint enrichment_jobs_attempts_valid
        check (attempts >= 0),
    constraint enrichment_jobs_max_attempts_valid
        check (max_attempts >= 1)
);

-- Index for pattern lookups
create index idx_enrichment_jobs_pattern
    on enrichment_jobs(pattern_id);

comment on table enrichment_jobs is 'Background processing queue for pattern enrichment';
comment on column enrichment_jobs.status is 'Job state: pending, processing, completed, or failed';
comment on column enrichment_jobs.scheduled_for is 'When the job should be processed (supports delayed retry)';
```

```sql
-- src/mnemonic/migrations/postgres/006_create_enrichment_jobs.down.sql
drop index if exists idx_enrichment_jobs_pattern;
drop table if exists enrichment_jobs;
```

### Migration 007: Performance Indexes

**Purpose:** Create performance-optimized indexes for common query patterns.

```sql
-- src/mnemonic/migrations/postgres/007_create_performance_indexes.up.sql
-- Creates performance indexes for common query patterns
-- Part of Mnemonic MVP

-- Patterns: enriched patterns only (for similarity search filtering)
create index idx_patterns_enriched
    on patterns(id)
    where enrichment_status = 'enriched';

-- Patterns: vector similarity search (IVFFlat for MVP scale)
-- lists = 100 suitable for 1,000-10,000 patterns
create index idx_patterns_embedding
    on patterns using ivfflat (embedding vector_cosine_ops)
    with (lists = 100);

-- Enrichment jobs: pending jobs by scheduled time (worker polling)
create index idx_enrichment_jobs_pending
    on enrichment_jobs(scheduled_for)
    where status = 'pending';

-- Enrichment jobs: processing jobs for timeout detection
create index idx_enrichment_jobs_processing
    on enrichment_jobs(started_at)
    where status = 'processing';

-- Patterns: GIN index for tag filtering
create index idx_patterns_tags
    on patterns using gin (tags);

-- Patterns: full-text search on name and description
create index idx_patterns_search
    on patterns using gin (
        to_tsvector('english', name || ' ' || coalesce(description, ''))
    );

-- Index documentation
comment on index idx_patterns_embedding is
    'IVFFlat index for vector similarity search (100 lists for MVP scale)';
comment on index idx_enrichment_jobs_pending is
    'Optimizes worker polling for pending jobs';
```

```sql
-- src/mnemonic/migrations/postgres/007_create_performance_indexes.down.sql
drop index if exists idx_patterns_search;
drop index if exists idx_patterns_tags;
drop index if exists idx_enrichment_jobs_processing;
drop index if exists idx_enrichment_jobs_pending;
drop index if exists idx_patterns_embedding;
drop index if exists idx_patterns_enriched;
```

### Migration 008: Drop Routing Rules

**Purpose:** Remove routing rules table (pivot to knowledge sync).

```sql
-- src/mnemonic/migrations/postgres/008_drop_routing_rules.up.sql
-- Drops the routing_rules table and related indexes
-- Part of Mnemonic pivot to knowledge sync (2026-02-15)

drop index if exists idx_routing_rules_agent;
drop index if exists idx_routing_rules_enabled_priority;
drop table if exists routing_rules;
```

```sql
-- src/mnemonic/migrations/postgres/008_drop_routing_rules.down.sql
-- Recreates the routing_rules table and indexes
-- WARNING: This recreates the schema but does not restore data

create table if not exists routing_rules (
    id uuid primary key default gen_random_uuid(),
    name varchar(128) not null unique,
    priority integer not null check (priority >= 0 and priority <= 1000),
    agent_name varchar(64) not null references agents(name) on delete restrict,
    match_type varchar(20) not null check (match_type in ('keyword', 'regex', 'pattern', 'default')),
    match_config jsonb not null,
    enabled boolean not null default true,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint routing_rules_match_config_valid check (
        (match_type = 'keyword' and match_config ? 'keywords' and match_config ? 'match_mode') or
        (match_type = 'regex' and match_config ? 'pattern') or
        (match_type = 'pattern' and match_config ? 'pattern_ids') or
        (match_type = 'default')
    )
);

create index idx_routing_rules_agent on routing_rules(agent_name);
create index idx_routing_rules_enabled_priority on routing_rules(priority desc, id) where enabled = true;
```

### Migration 009: Migrate Agents to JSONB Document Model

**Purpose:** Transform the agents table from decomposed columns to the JSONB document model with CRC64 change detection.

This migration:

1. Adds `id` (UUID), `definition` (JSONB), and `crc64` (BIGINT) columns
2. Populates `definition` from existing column data (excluding `routing_keywords`, which is deprecated)
3. Drops the individual content columns (`description`, `system_prompt`, `model`, `allowed_tools`, `routing_keywords`, `version`)
4. Drops associated CHECK constraints and triggers
5. Makes `name` a regular unique column instead of the primary key, with `id` as the new primary key

```sql
-- src/mnemonic/migrations/postgres/009_migrate_agents_to_jsonb.up.sql
-- Migrates agents table to JSONB document model with CRC64
-- Part of Mnemonic pivot to knowledge sync (2026-02-15)

-- Step 1: Add new columns
alter table agents add column id uuid default gen_random_uuid();
alter table agents add column definition jsonb;
alter table agents add column crc64 bigint;

-- Step 2: Populate definition from existing columns
-- NOTE: routing_keywords is intentionally excluded (deprecated)
update agents set definition = jsonb_build_object(
    'description', description,
    'system_prompt', system_prompt,
    'model', model,
    'allowed_tools', allowed_tools,
    'version', coalesce(version, '0.0.0')
);

-- Step 3: CRC64 must be set by the application after migration.
-- Set a placeholder value of 0 for existing rows. The application should
-- recompute CRC64 values on first access or via a one-time script.
update agents set crc64 = 0 where crc64 is null;

-- Step 4: Apply NOT NULL constraints now that data is populated
alter table agents alter column id set not null;
alter table agents alter column definition set not null;
alter table agents alter column crc64 set not null;

-- Step 5: Drop the trigger function reference (no more trigger-based updated_at)
drop trigger if exists trg_agents_updated_at on agents;

-- Step 6: Drop old constraints before dropping columns
alter table agents drop constraint if exists agents_model_valid;
alter table agents drop constraint if exists agents_system_prompt_length;
alter table agents drop constraint if exists agents_allowed_tools_array;
alter table agents drop constraint if exists agents_routing_keywords_array;

-- Step 7: Drop foreign key references from pattern_agent_associations
-- before changing the primary key. We will recreate them.
alter table pattern_agent_associations drop constraint if exists fk_pattern_agent_assoc_agent;

-- Step 8: Drop old primary key and individual columns
alter table agents drop constraint agents_pkey;
alter table agents drop column description;
alter table agents drop column system_prompt;
alter table agents drop column model;
alter table agents drop column allowed_tools;
alter table agents drop column routing_keywords;
alter table agents drop column version;

-- Step 9: Set new primary key and unique constraint
alter table agents add primary key (id);
alter table agents add constraint agents_name_unique unique (name);

-- Step 10: Recreate foreign key from pattern_agent_associations
alter table pattern_agent_associations
    add constraint fk_pattern_agent_assoc_agent
    foreign key (agent_name) references agents(name) on delete cascade;

-- Step 11: GIN index on definition for JSONB queries
create index idx_agents_definition on agents using gin (definition);

comment on table agents is 'Agent definitions stored as JSONB documents for team tooling synchronization';
comment on column agents.id is 'UUID primary key';
comment on column agents.name is 'Unique lookup key, lowercase-with-hyphens format';
comment on column agents.definition is 'Complete agent definition as JSONB document';
comment on column agents.crc64 is 'CRC-64 checksum of serialized definition for change detection';
```

```sql
-- src/mnemonic/migrations/postgres/009_migrate_agents_to_jsonb.down.sql
-- Reverses: migrate agents to JSONB document model
-- WARNING: This restores the schema but individual field data is extracted from JSONB

-- Drop the GIN index
drop index if exists idx_agents_definition;

-- Drop foreign key that references agents(name)
alter table pattern_agent_associations drop constraint if exists fk_pattern_agent_assoc_agent;

-- Drop new primary key and unique constraint
alter table agents drop constraint if exists agents_name_unique;
alter table agents drop constraint agents_pkey;

-- Re-add individual columns
alter table agents add column description varchar(500);
alter table agents add column system_prompt text;
alter table agents add column model varchar(20) default 'sonnet';
alter table agents add column allowed_tools jsonb default '[]'::jsonb;
alter table agents add column routing_keywords jsonb default '[]'::jsonb;
alter table agents add column version varchar(50);

-- Populate individual columns from JSONB definition
update agents set
    description = definition->>'description',
    system_prompt = definition->>'system_prompt',
    model = coalesce(definition->>'model', 'sonnet'),
    allowed_tools = coalesce(definition->'allowed_tools', '[]'::jsonb),
    routing_keywords = '[]'::jsonb,
    version = definition->>'version';

-- Apply NOT NULL constraints
alter table agents alter column description set not null;
alter table agents alter column system_prompt set not null;
alter table agents alter column model set not null;
alter table agents alter column allowed_tools set not null;
alter table agents alter column routing_keywords set not null;

-- Restore primary key on name
alter table agents add primary key (name);

-- Drop JSONB columns
alter table agents drop column id;
alter table agents drop column definition;
alter table agents drop column crc64;

-- Restore constraints
alter table agents add constraint agents_name_format
    check (name ~ '^[a-z][a-z0-9-]*$');
alter table agents add constraint agents_model_valid
    check (model in ('sonnet', 'opus', 'haiku'));
alter table agents add constraint agents_system_prompt_length
    check (length(system_prompt) <= 51200);
alter table agents add constraint agents_allowed_tools_array
    check (jsonb_typeof(allowed_tools) = 'array');
alter table agents add constraint agents_routing_keywords_array
    check (jsonb_typeof(routing_keywords) = 'array');

-- Restore foreign key from pattern_agent_associations
alter table pattern_agent_associations
    add constraint fk_pattern_agent_assoc_agent
    foreign key (agent_name) references agents(name) on delete cascade;
```

### Migration 010: Create Skills Table

**Purpose:** Create the skills table using the JSONB document model.

```sql
-- src/mnemonic/migrations/postgres/010_create_skills.up.sql
-- Creates the skills table with JSONB document model
-- Part of Mnemonic pivot to knowledge sync (2026-02-15)

create table if not exists skills (
    id          uuid primary key default gen_random_uuid(),
    name        varchar(64) not null,
    definition  jsonb not null,
    crc64       bigint not null,
    created_at  timestamptz not null default now(),
    updated_at  timestamptz not null default now(),

    constraint skills_name_unique unique (name),
    constraint skills_name_format
        check (name ~ '^[a-z][a-z0-9-]*$')
);

-- GIN index on definition for tag filtering and JSONB queries
create index idx_skills_definition on skills using gin (definition);

comment on table skills is 'Skill definitions stored as JSONB documents for team tooling synchronization';
comment on column skills.name is 'Unique lookup key, lowercase-with-hyphens (matches Claude Code skill directory name)';
comment on column skills.definition is 'Complete skill definition as JSONB document (Agent Skills spec aligned)';
comment on column skills.crc64 is 'CRC-64 checksum of serialized definition for change detection';
```

```sql
-- src/mnemonic/migrations/postgres/010_create_skills.down.sql
drop index if exists idx_skills_definition;
drop table if exists skills;
```

### Migration 011: Create Commands Table

**Purpose:** Create the commands table using the JSONB document model.

```sql
-- src/mnemonic/migrations/postgres/011_create_commands.up.sql
-- Creates the commands table with JSONB document model
-- Part of Mnemonic pivot to knowledge sync (2026-02-15)

create table if not exists commands (
    id          uuid primary key default gen_random_uuid(),
    name        varchar(255) not null,
    definition  jsonb not null,
    crc64       bigint not null,
    created_at  timestamptz not null default now(),
    updated_at  timestamptz not null default now(),

    constraint commands_name_unique unique (name)
);

-- GIN index on definition for tag filtering and JSONB queries
create index idx_commands_definition on commands using gin (definition);

comment on table commands is 'Command definitions stored as JSONB documents for team tooling synchronization';
comment on column commands.name is 'Unique lookup key (matches Claude Code command name)';
comment on column commands.definition is 'Complete command definition as JSONB document';
comment on column commands.crc64 is 'CRC-64 checksum of serialized definition for change detection';
```

```sql
-- src/mnemonic/migrations/postgres/011_create_commands.down.sql
drop index if exists idx_commands_definition;
drop table if exists commands;
```

### Migration 012: Create Skill Files Table

**Purpose:** Create the skill_files table for scripts, references, and assets associated with skills.

```sql
-- src/mnemonic/migrations/postgres/012_create_skill_files.up.sql
-- Creates the skill_files table with JSONB document model
-- Part of Mnemonic pivot to knowledge sync (2026-02-15)

create table if not exists skill_files (
    id          uuid primary key default gen_random_uuid(),
    skill_id    uuid not null references skills(id) on delete cascade,
    file_type   varchar(20) not null,
    filename    varchar(255) not null,
    document    jsonb not null,
    crc64       bigint not null,
    created_at  timestamptz not null default now(),
    updated_at  timestamptz not null default now(),

    constraint skill_files_unique_name unique (skill_id, file_type, filename),
    constraint skill_files_file_type_valid
        check (file_type in ('script', 'reference', 'asset'))
);

-- Index for skill_id lookups (foreign key)
create index idx_skill_files_skill_id on skill_files(skill_id);

comment on table skill_files is 'Child files (scripts, references, assets) for skill definitions';
comment on column skill_files.skill_id is 'Parent skill reference, cascade delete';
comment on column skill_files.file_type is 'File category: script, reference, or asset';
comment on column skill_files.filename is 'File name within the skill directory';
comment on column skill_files.document is 'File content and metadata as JSONB document';
comment on column skill_files.crc64 is 'CRC-64 checksum of serialized document for change detection';
```

```sql
-- src/mnemonic/migrations/postgres/012_create_skill_files.down.sql
drop index if exists idx_skill_files_skill_id;
drop table if exists skill_files;
```

## PGVector Configuration

[Table of Contents](#table-of-contents)

> **Architecture Reference:** [Data Architecture - PGVector Configuration](../../architecture/04-data-architecture.md#pgvector-configuration)

### Index Selection Guidelines

The vector index type should match the expected pattern count:

| Pattern Count | Index Type | Configuration | Query Performance |
|---------------|------------|---------------|-------------------|
| < 1,000 | None (exact search) | N/A | ~10ms |
| 1,000 - 100,000 | IVFFlat | `lists = sqrt(N)` | ~20-50ms |
| > 100,000 | HNSW | `m=16, ef_construction=64` | ~5-20ms |

**MVP Recommendation:** IVFFlat with 100 lists (suitable for 1,000-10,000 patterns).

**Upgrading to HNSW** (when pattern count exceeds 100,000):

```sql
-- Drop existing IVFFlat index
drop index if exists idx_patterns_embedding;

-- Create HNSW index (higher memory usage, better recall)
create index idx_patterns_embedding
    on patterns using hnsw (embedding vector_cosine_ops)
    with (m = 16, ef_construction = 64);
```

### Similarity Search Queries

**Basic Similarity Search:**

```sql
-- Find patterns similar to a given embedding
-- Returns patterns with similarity score (1.0 = identical)
select
    id,
    name,
    content,
    1 - (embedding <=> $1::vector) as similarity
from patterns
where enrichment_status = 'enriched'
  and embedding is not null
order by embedding <=> $1::vector
limit $2;  -- max_patterns parameter
```

**Similarity Search with Threshold:**

```sql
-- Find patterns above a similarity threshold
-- $1: query embedding, $2: minimum similarity (e.g., 0.7), $3: limit
select
    id,
    name,
    content,
    tags,
    1 - (embedding <=> $1::vector) as similarity
from patterns
where enrichment_status = 'enriched'
  and embedding is not null
  and (embedding <=> $1::vector) < (1 - $2)  -- convert similarity to distance
order by embedding <=> $1::vector
limit $3;
```

**Similarity Search with Tag Filter:**

```sql
-- Find similar patterns filtered by tags
-- $1: query embedding, $2: required tag, $3: limit
select
    id,
    name,
    content,
    1 - (embedding <=> $1::vector) as similarity
from patterns
where enrichment_status = 'enriched'
  and embedding is not null
  and tags @> $2::jsonb  -- contains the specified tag
order by embedding <=> $1::vector
limit $3;
```

### Index Maintenance

**Adjusting IVFFlat Probes:**

```sql
-- Increase probes for better recall (at cost of latency)
-- Default is 1, higher values check more lists
set ivfflat.probes = 10;

-- Query with increased probes
select id, name, 1 - (embedding <=> $1::vector) as similarity
from patterns
where enrichment_status = 'enriched'
order by embedding <=> $1::vector
limit 5;

-- Reset to default
reset ivfflat.probes;
```

**Reindexing After Bulk Updates:**

```sql
-- Reindex if significant portion of patterns updated
-- Use CONCURRENTLY to avoid blocking reads
reindex index concurrently idx_patterns_embedding;
```

**Analyzing for Query Planner:**

```sql
-- Update statistics after bulk operations
analyze patterns;
```

## Neo4j Setup

[Table of Contents](#table-of-contents)

> **Architecture Reference:** [Data Architecture - Neo4j Graph Model](../../architecture/04-data-architecture.md#neo4j-graph-model)

### Schema Constraints

Create these constraints during initial Neo4j setup.

**Uniqueness Constraints (Community Edition + Enterprise Edition):**

These constraints are compatible with all Neo4j editions and are created by `001_create_constraints.cypher`:

```cypher
// src/mnemonic/migrations/neo4j/001_create_constraints.cypher
// Creates uniqueness constraints for node labels
// Part of Mnemonic MVP

// Pattern nodes: UUID from Postgres
CREATE CONSTRAINT pattern_id_unique IF NOT EXISTS
FOR (p:Pattern) REQUIRE p.id IS UNIQUE;

// Agent nodes: name from Postgres
CREATE CONSTRAINT agent_name_unique IF NOT EXISTS
FOR (a:Agent) REQUIRE a.name IS UNIQUE;

// Concept nodes: normalized lowercase name
CREATE CONSTRAINT concept_name_unique IF NOT EXISTS
FOR (c:Concept) REQUIRE c.name IS UNIQUE;
```

**Existence Constraints (Enterprise Edition Only):**

These constraints require Neo4j Enterprise Edition and are created by `002_create_existence_constraints.cypher`. Community Edition users should skip this migration. The application layer enforces property completeness regardless of whether these database-level constraints are present.

```cypher
// src/mnemonic/migrations/neo4j/002_create_existence_constraints.cypher
// Requires Neo4j Enterprise Edition
// Part of Mnemonic MVP

// Existence constraints (properties must exist)
CREATE CONSTRAINT pattern_name_exists IF NOT EXISTS
FOR (p:Pattern) REQUIRE p.name IS NOT NULL;

CREATE CONSTRAINT agent_name_exists IF NOT EXISTS
FOR (a:Agent) REQUIRE a.name IS NOT NULL;

CREATE CONSTRAINT concept_name_exists IF NOT EXISTS
FOR (c:Concept) REQUIRE c.name IS NOT NULL;
```

### Startup Constraint Validation

Mnemonic validates Neo4j schema constraints at startup to ensure the database is properly configured. This validation is advisory only -- Mnemonic will not fail startup if constraints are missing, since Neo4j is used in a best-effort capacity.

**Validation Behavior:**

| Constraint Status | Mnemonic Behavior |
|-------------------|-------------------|
| All constraints exist | Log info message, continue startup |
| One or more missing | Log warning with missing constraint names, continue startup |
| Connection failure | Log warning, continue startup (Neo4j operations will fail gracefully) |

**Constraints Checked:**

The constraints checked depend on the Neo4j edition:

**Community Edition (3 constraints):**

- `pattern_id_unique` - Uniqueness on Pattern.id
- `agent_name_unique` - Uniqueness on Agent.name
- `concept_name_unique` - Uniqueness on Concept.name

**Enterprise Edition (6 constraints):**

- All Community Edition constraints above, plus:
- `pattern_name_exists` - Existence of Pattern.name
- `agent_name_exists` - Existence of Agent.name
- `concept_name_exists` - Existence of Concept.name

**Manual Constraint Creation:**

If constraints are missing, run the appropriate migration scripts manually:

```bash
# Using cypher-shell (Community + Enterprise)
cypher-shell -u neo4j -p <password> -f src/mnemonic/migrations/neo4j/001_create_constraints.cypher

# Enterprise Edition only (optional)
cypher-shell -u neo4j -p <password> -f src/mnemonic/migrations/neo4j/002_create_existence_constraints.cypher
```

### Index Configuration

```cypher
// src/mnemonic/migrations/neo4j/003_create_indexes.cypher
// Creates indexes for common query patterns
// Part of Mnemonic MVP

// Pattern lookup by name
CREATE INDEX pattern_name_index IF NOT EXISTS
FOR (p:Pattern) ON (p.name);

// Concept filtering by type
CREATE INDEX concept_type_index IF NOT EXISTS
FOR (c:Concept) ON (c.type);

// Full-text search on pattern content
CREATE FULLTEXT INDEX pattern_content_fulltext IF NOT EXISTS
FOR (p:Pattern) ON EACH [p.name, p.description];

// Full-text search on concept names
CREATE FULLTEXT INDEX concept_name_fulltext IF NOT EXISTS
FOR (c:Concept) ON EACH [c.name];
```

### Node Creation Patterns

**Agent Node:**

```cypher
// Create or update agent node (sync from Postgres)
MERGE (a:Agent {name: $name})
ON CREATE SET
    a.createdAt = datetime()
ON MATCH SET
    a.updatedAt = datetime()
RETURN a;
```

**Pattern Node:**

```cypher
// Create or update pattern node (sync from Postgres after enrichment)
MERGE (p:Pattern {id: $id})
ON CREATE SET
    p.name = $name,
    p.description = $description,
    p.createdAt = datetime()
ON MATCH SET
    p.name = $name,
    p.description = $description,
    p.updatedAt = datetime()
RETURN p;
```

**Concept Node:**

```cypher
// Create concept if not exists (extracted during enrichment)
MERGE (c:Concept {name: $name})
ON CREATE SET
    c.type = $type,
    c.createdAt = datetime()
RETURN c;
```

### Relationship Definitions

**RELEVANT_FOR (Pattern to Agent):**

```cypher
// Create pattern-agent relevance relationship
// Mirrors pattern_agent_associations table
MATCH (p:Pattern {id: $patternId})
MATCH (a:Agent {name: $agentName})
MERGE (p)-[r:RELEVANT_FOR]->(a)
SET r.relevance = $relevance,
    r.updatedAt = datetime()
RETURN r;
```

**MENTIONED_IN (Concept to Pattern):**

```cypher
// Create concept-pattern mention relationship
MATCH (c:Concept {name: $conceptName})
MATCH (p:Pattern {id: $patternId})
MERGE (c)-[r:MENTIONED_IN]->(p)
ON CREATE SET r.createdAt = datetime()
RETURN r;
```

**RELATES_TO (Pattern to Pattern):**

```cypher
// Create pattern-pattern similarity relationship
// Computed from shared concepts
MATCH (p1:Pattern {id: $patternId1})
MATCH (p2:Pattern {id: $patternId2})
WHERE p1 <> p2
MERGE (p1)-[r:RELATES_TO]->(p2)
SET r.similarity = $similarity,
    r.updatedAt = datetime()
RETURN r;
```

### Graph Synchronization Queries

**Sync Agent from Postgres:**

```cypher
// Called on agent create/update in Postgres
MERGE (a:Agent {name: $name})
SET a.updatedAt = datetime()
RETURN a;
```

**Sync Pattern with Associations:**

```cypher
// Called after pattern enrichment completes
// Transaction: create/update pattern, then set all associations

// Step 1: Create/update pattern node
MERGE (p:Pattern {id: $patternId})
SET p.name = $patternName,
    p.description = $patternDescription,
    p.updatedAt = datetime();

// Step 2: Remove old RELEVANT_FOR relationships
MATCH (p:Pattern {id: $patternId})-[r:RELEVANT_FOR]->()
DELETE r;

// Step 3: Create new RELEVANT_FOR relationships
UNWIND $associations AS assoc
MATCH (p:Pattern {id: $patternId})
MATCH (a:Agent {name: assoc.agentName})
CREATE (p)-[:RELEVANT_FOR {relevance: assoc.relevance}]->(a);
```

**Sync Concepts for Pattern:**

```cypher
// Called during enrichment to add extracted concepts
// $concepts: array of {name: string, type: string}

// Step 1: Remove old MENTIONED_IN relationships for this pattern
MATCH (:Concept)-[r:MENTIONED_IN]->(:Pattern {id: $patternId})
DELETE r;

// Step 2: Create concepts and relationships
UNWIND $concepts AS concept
MERGE (c:Concept {name: concept.name})
ON CREATE SET c.type = concept.type, c.createdAt = datetime()
WITH c
MATCH (p:Pattern {id: $patternId})
CREATE (c)-[:MENTIONED_IN]->(p);
```

**Find Related Patterns:**

```cypher
// Query: find patterns related to a given pattern through shared concepts
MATCH (p1:Pattern {id: $patternId})<-[:MENTIONED_IN]-(c:Concept)-[:MENTIONED_IN]->(p2:Pattern)
WHERE p1 <> p2
WITH p2, count(c) AS sharedConcepts
ORDER BY sharedConcepts DESC
LIMIT $limit
RETURN p2.id AS id, p2.name AS name, sharedConcepts;
```

**Find Patterns for Agent:**

```cypher
// Query: find patterns relevant to an agent, ordered by relevance
MATCH (p:Pattern)-[r:RELEVANT_FOR]->(a:Agent {name: $agentName})
RETURN p.id AS id, p.name AS name, r.relevance AS relevance
ORDER BY r.relevance DESC
LIMIT $limit;
```

**Cleanup Orphaned Concepts:**

```cypher
// Maintenance: remove concepts with no pattern relationships
MATCH (c:Concept)
WHERE NOT (c)-[:MENTIONED_IN]->()
DELETE c
RETURN count(c) AS deletedCount;
```

## Repository Interfaces

[Table of Contents](#table-of-contents)

> **Note:** These are interface definitions only. Implementation is handled by the go-software-engineer.

### AgentRepository

```go
// AgentRepository defines data access operations for agents.
// Agents are stored as JSONB documents with name as the unique lookup key.
// Implementation: internal/repository/agent/repository.go
type AgentRepository interface {
    // Create stores a new agent. The application computes crc64 from the
    // serialized definition before calling this method.
    // Returns ErrExists if name already exists.
    Create(ctx context.Context, agent *Agent) error

    // Get retrieves an agent by name. Returns ErrNotFound if not found.
    Get(ctx context.Context, name string) (*Agent, error)

    // GetByID retrieves an agent by UUID. Returns ErrNotFound if not found.
    GetByID(ctx context.Context, id uuid.UUID) (*Agent, error)

    // Update modifies an existing agent. The application computes crc64
    // from the serialized definition and sets updated_at before calling
    // this method. Returns ErrNotFound if not found.
    Update(ctx context.Context, agent *Agent) error

    // Delete removes an agent by name. Returns ErrNotFound if not found.
    Delete(ctx context.Context, name string) error

    // List retrieves all agents with optional pagination.
    List(ctx context.Context, opts ListOptions) ([]*Agent, int64, error)

    // Exists checks if an agent with the given name exists.
    Exists(ctx context.Context, name string) (bool, error)

    // GetManifest returns name and crc64 for all agents (used by sync protocol).
    GetManifest(ctx context.Context) ([]ManifestEntry, error)
}

// Agent represents an agent definition stored as a JSONB document.
type Agent struct {
    ID         uuid.UUID       `db:"id"`
    Name       string          `db:"name"`
    Definition json.RawMessage `db:"definition"` // JSONB document
    CRC64      int64           `db:"crc64"`       // CRC-64 checksum
    CreatedAt  time.Time       `db:"created_at"`
    UpdatedAt  time.Time       `db:"updated_at"`
}

// ManifestEntry represents a single entity in the sync manifest.
type ManifestEntry struct {
    Name  string `db:"name"`
    CRC64 int64  `db:"crc64"`
}
```

### PatternRepository

```go
// PatternRepository defines data access operations for patterns.
// Patterns use relational columns (NOT the JSONB document model).
// Implementation: internal/repository/pattern/repository.go
type PatternRepository interface {
    // Create stores a new pattern. Returns ErrNameExists if name exists.
    Create(ctx context.Context, pattern *Pattern) error

    // Get retrieves a pattern by ID. Returns ErrNotFound if not found.
    Get(ctx context.Context, id uuid.UUID) (*Pattern, error)

    // GetByName retrieves a pattern by name. Returns ErrNotFound if not found.
    GetByName(ctx context.Context, name string) (*Pattern, error)

    // Update modifies an existing pattern. The application sets updated_at
    // before calling this method. Returns ErrNotFound if not found.
    Update(ctx context.Context, pattern *Pattern) error

    // Delete removes a pattern by ID. Returns ErrNotFound if not found.
    Delete(ctx context.Context, id uuid.UUID) error

    // List retrieves patterns with filtering and pagination.
    List(ctx context.Context, filter Filter, opts ListOptions) ([]*Pattern, int64, error)

    // UpdateEmbedding stores the embedding vector for a pattern.
    UpdateEmbedding(ctx context.Context, id uuid.UUID, embedding []float32) error

    // UpdateEnrichmentStatus updates the enrichment state of a pattern.
    UpdateEnrichmentStatus(ctx context.Context, id uuid.UUID, status string, err error) error

    // FindSimilar finds patterns similar to the given embedding vector.
    FindSimilar(ctx context.Context, embedding []float32, opts SimilarityOptions) ([]*Match, error)

    // SetAgentAssociations replaces all agent associations for a pattern.
    SetAgentAssociations(ctx context.Context, patternID uuid.UUID, associations []AgentAssociation) error

    // GetAgentAssociations retrieves all agent associations for a pattern.
    GetAgentAssociations(ctx context.Context, patternID uuid.UUID) ([]AgentAssociation, error)
}

// Pattern represents a context pattern (relational columns, not JSONB).
type Pattern struct {
    ID               uuid.UUID  `db:"id"`
    Name             string     `db:"name"`
    Description      *string    `db:"description"`
    Content          string     `db:"content"`
    Tags             []string   `db:"-"` // Unmarshaled from JSONB
    Embedding        []float32  `db:"-"` // Vector type
    EnrichmentStatus string     `db:"enrichment_status"`
    EnrichmentError  *string    `db:"enrichment_error"`
    EnrichedAt       *time.Time `db:"enriched_at"`
    CreatedAt        time.Time  `db:"created_at"`
    UpdatedAt        time.Time  `db:"updated_at"`
}

// Filter defines filtering options for pattern queries.
type Filter struct {
    Tags             []string // Filter by any of these tags
    EnrichmentStatus string   // Filter by enrichment status
    SearchQuery      string   // Full-text search in name/description
}

// SimilarityOptions defines options for similarity search.
type SimilarityOptions struct {
    MinSimilarity float64     // Minimum similarity threshold (0.0-1.0)
    MaxResults    int         // Maximum number of results
    Tags          []string    // Optional tag filter
    PatternIDs    []uuid.UUID // Optional filter to specific pattern IDs
}

// Match represents a similarity search result.
type Match struct {
    Pattern    *Pattern
    Similarity float64
}

// AgentAssociation represents a pattern-agent relationship.
type AgentAssociation struct {
    AgentName string  `db:"agent_name"`
    Relevance float64 `db:"relevance"`
}
```

### SkillRepository

```go
// SkillRepository defines data access operations for skills.
// Skills are stored as JSONB documents with name as the unique lookup key.
// Implementation: internal/repository/skill/repository.go
type SkillRepository interface {
    // Create stores a new skill. The application computes crc64 from the
    // serialized definition before calling this method.
    // Returns ErrExists if name already exists.
    Create(ctx context.Context, skill *Skill) error

    // Get retrieves a skill by ID. Returns ErrNotFound if not found.
    Get(ctx context.Context, id uuid.UUID) (*Skill, error)

    // GetByName retrieves a skill by name. Returns ErrNotFound if not found.
    GetByName(ctx context.Context, name string) (*Skill, error)

    // Update modifies an existing skill. Returns ErrNotFound if not found.
    Update(ctx context.Context, skill *Skill) error

    // Delete removes a skill by ID. Returns ErrNotFound if not found.
    Delete(ctx context.Context, id uuid.UUID) error

    // List retrieves skills with optional pagination.
    List(ctx context.Context, opts ListOptions) ([]*Skill, int64, error)

    // Exists checks if a skill with the given ID exists.
    Exists(ctx context.Context, id uuid.UUID) (bool, error)

    // GetManifest returns name and crc64 for all skills (used by sync protocol).
    GetManifest(ctx context.Context) ([]ManifestEntry, error)
}

// Skill represents a skill definition stored as a JSONB document.
type Skill struct {
    ID         uuid.UUID       `db:"id"`
    Name       string          `db:"name"`
    Definition json.RawMessage `db:"definition"` // JSONB document
    CRC64      int64           `db:"crc64"`       // CRC-64 checksum
    CreatedAt  time.Time       `db:"created_at"`
    UpdatedAt  time.Time       `db:"updated_at"`
}
```

### CommandRepository

```go
// CommandRepository defines data access operations for commands.
// Commands are stored as JSONB documents with name as the unique lookup key.
// Implementation: internal/repository/command/repository.go
type CommandRepository interface {
    // Create stores a new command. The application computes crc64 from the
    // serialized definition before calling this method.
    // Returns ErrExists if name already exists.
    Create(ctx context.Context, command *Command) error

    // Get retrieves a command by ID. Returns ErrNotFound if not found.
    Get(ctx context.Context, id uuid.UUID) (*Command, error)

    // GetByName retrieves a command by name. Returns ErrNotFound if not found.
    GetByName(ctx context.Context, name string) (*Command, error)

    // Update modifies an existing command. Returns ErrNotFound if not found.
    Update(ctx context.Context, command *Command) error

    // Delete removes a command by ID. Returns ErrNotFound if not found.
    Delete(ctx context.Context, id uuid.UUID) error

    // List retrieves commands with optional pagination.
    List(ctx context.Context, opts ListOptions) ([]*Command, int64, error)

    // Exists checks if a command with the given ID exists.
    Exists(ctx context.Context, id uuid.UUID) (bool, error)

    // GetManifest returns name and crc64 for all commands (used by sync protocol).
    GetManifest(ctx context.Context) ([]ManifestEntry, error)
}

// Command represents a command definition stored as a JSONB document.
type Command struct {
    ID         uuid.UUID       `db:"id"`
    Name       string          `db:"name"`
    Definition json.RawMessage `db:"definition"` // JSONB document
    CRC64      int64           `db:"crc64"`       // CRC-64 checksum
    CreatedAt  time.Time       `db:"created_at"`
    UpdatedAt  time.Time       `db:"updated_at"`
}
```

### SkillFileRepository

```go
// SkillFileRepository defines data access operations for skill child files.
// Skill files are stored as JSONB documents keyed by (skill_id, file_type, filename).
// Implementation: internal/repository/skillfile/repository.go
type SkillFileRepository interface {
    // Create stores a new skill file. Returns ErrExists if the
    // (skill_id, file_type, filename) combination already exists.
    Create(ctx context.Context, file *SkillFile) error

    // Get retrieves a skill file by ID. Returns ErrNotFound if not found.
    Get(ctx context.Context, id uuid.UUID) (*SkillFile, error)

    // GetByKey retrieves a skill file by its composite key.
    // Returns ErrNotFound if not found.
    GetByKey(ctx context.Context, skillID uuid.UUID, fileType string, filename string) (*SkillFile, error)

    // Update modifies an existing skill file. Returns ErrNotFound if not found.
    Update(ctx context.Context, file *SkillFile) error

    // Delete removes a skill file by ID. Returns ErrNotFound if not found.
    Delete(ctx context.Context, id uuid.UUID) error

    // ListBySkill retrieves all files for a given skill, optionally filtered by file_type.
    ListBySkill(ctx context.Context, skillID uuid.UUID, fileType *string) ([]*SkillFile, error)

    // DeleteBySkill removes all files for a given skill.
    DeleteBySkill(ctx context.Context, skillID uuid.UUID) error
}

// SkillFile represents a child file (script, reference, asset) for a skill.
type SkillFile struct {
    ID        uuid.UUID       `db:"id"`
    SkillID   uuid.UUID       `db:"skill_id"`
    FileType  string          `db:"file_type"` // script, reference, asset
    Filename  string          `db:"filename"`
    Document  json.RawMessage `db:"document"`  // JSONB document (content, encoding, etc.)
    CRC64     int64           `db:"crc64"`      // CRC-64 checksum
    CreatedAt time.Time       `db:"created_at"`
    UpdatedAt time.Time       `db:"updated_at"`
}
```

### EnrichmentJob Repository

```go
// EnrichmentJobRepository defines data access operations for enrichment jobs.
// Implementation: internal/repository/enrichmentjob/repository.go
type EnrichmentJobRepository interface {
    // Create stores a new enrichment job.
    Create(ctx context.Context, job *Job) error

    // Get retrieves an enrichment job by ID. Returns ErrNotFound if not found.
    Get(ctx context.Context, id uuid.UUID) (*Job, error)

    // GetByPatternID retrieves the latest job for a pattern.
    GetByPatternID(ctx context.Context, patternID uuid.UUID) (*Job, error)

    // ClaimPending atomically claims a pending job for processing.
    // Uses FOR UPDATE SKIP LOCKED for safe concurrent processing.
    // Returns nil if no pending jobs are available.
    ClaimPending(ctx context.Context) (*Job, error)

    // MarkProcessing updates job status to processing with start time.
    MarkProcessing(ctx context.Context, id uuid.UUID) error

    // MarkCompleted updates job status to completed with completion time.
    MarkCompleted(ctx context.Context, id uuid.UUID) error

    // MarkFailed updates job status to failed with error message.
    // Increments attempt count and schedules retry if under max_attempts.
    MarkFailed(ctx context.Context, id uuid.UUID, err error, retryDelay time.Duration) error

    // ReclaimStale reclaims jobs stuck in processing state.
    // Jobs older than timeout are reset to pending for retry.
    ReclaimStale(ctx context.Context, timeout time.Duration) (int64, error)

    // DeleteCompleted removes completed jobs older than the retention period.
    DeleteCompleted(ctx context.Context, retention time.Duration) (int64, error)

    // DeleteFailed removes failed jobs older than the retention period.
    DeleteFailed(ctx context.Context, retention time.Duration) (int64, error)

    // List retrieves enrichment jobs with filtering and pagination.
    List(ctx context.Context, filter Filter, opts ListOptions) ([]*Job, int64, error)
}

// Job represents a background enrichment task.
type Job struct {
    ID           uuid.UUID  `db:"id"`
    PatternID    uuid.UUID  `db:"pattern_id"`
    Status       string     `db:"status"`
    Attempts     int        `db:"attempts"`
    MaxAttempts  int        `db:"max_attempts"`
    LastError    *string    `db:"last_error"`
    ScheduledFor time.Time  `db:"scheduled_for"`
    StartedAt    *time.Time `db:"started_at"`
    CompletedAt  *time.Time `db:"completed_at"`
    CreatedAt    time.Time  `db:"created_at"`
    UpdatedAt    time.Time  `db:"updated_at"`
}

// Filter defines filtering options for job queries.
type Filter struct {
    Status    *string    // Filter by job status (pending, processing, completed, failed)
    PatternID *uuid.UUID // Filter by the associated pattern ID
}
```

### GraphRepository

```go
// GraphRepository defines data access operations for the Neo4j knowledge graph.
// Implementation: internal/repository/graph/repository.go
type GraphRepository interface {
    // SyncAgent creates or updates an agent node in the graph.
    SyncAgent(ctx context.Context, agentName string) error

    // DeleteAgent removes an agent node and its relationships.
    DeleteAgent(ctx context.Context, agentName string) error

    // SyncPattern creates or updates a pattern node with its relationships.
    SyncPattern(ctx context.Context, pattern *GraphPattern) error

    // DeletePattern removes a pattern node and its relationships.
    DeletePattern(ctx context.Context, patternID uuid.UUID) error

    // SyncConcepts creates concepts and their relationships to a pattern.
    SyncConcepts(ctx context.Context, patternID uuid.UUID, concepts []Concept) error

    // SetPatternAgentRelevance sets the relevance relationships for a pattern.
    SetPatternAgentRelevance(ctx context.Context, patternID uuid.UUID, associations []AgentAssociation) error

    // FindRelatedPatterns finds patterns related through shared concepts.
    FindRelatedPatterns(ctx context.Context, patternID uuid.UUID, limit int) ([]RelatedPattern, error)

    // FindPatternsByAgent finds patterns relevant to an agent.
    FindPatternsByAgent(ctx context.Context, agentName string, limit int) ([]PatternRelevance, error)

    // CleanupOrphanedConcepts removes concepts with no pattern relationships.
    CleanupOrphanedConcepts(ctx context.Context) (int64, error)

    // HealthCheck verifies the graph database connection.
    HealthCheck(ctx context.Context) error
}

// GraphPattern represents pattern data for graph synchronization.
type GraphPattern struct {
    ID          uuid.UUID
    Name        string
    Description *string
}

// Concept represents an extracted concept entity.
type Concept struct {
    Name string // Normalized lowercase
    Type string // technology, practice, domain
}

// RelatedPattern represents a pattern found through graph traversal.
type RelatedPattern struct {
    ID             uuid.UUID
    Name           string
    SharedConcepts int
}

// PatternRelevance represents a pattern with its relevance to an agent.
type PatternRelevance struct {
    ID        uuid.UUID
    Name      string
    Relevance float64
}
```

### Common Types

```go
// ListOptions defines pagination parameters.
type ListOptions struct {
    Offset int // Number of items to skip
    Limit  int // Maximum items to return (0 = no limit)
}

// Repository errors (package-specific)

// agent package errors
var (
    ErrNotFound = errors.New("agent not found")
    ErrExists   = errors.New("agent already exists")
)

// pattern package errors
var (
    ErrNotFound   = errors.New("pattern not found")
    ErrNameExists = errors.New("pattern name already exists")
)

// skill package errors
var (
    ErrNotFound = errors.New("skill not found")
    ErrExists   = errors.New("skill already exists")
)

// command package errors
var (
    ErrNotFound = errors.New("command not found")
    ErrExists   = errors.New("command already exists")
)

// skillfile package errors
var (
    ErrNotFound = errors.New("skill file not found")
    ErrExists   = errors.New("skill file already exists")
)

// enrichmentjob package errors
var (
    ErrNotFound  = errors.New("enrichment job not found")
    ErrNoPending = errors.New("no pending enrichment jobs available")
)
```

## Connection Configuration

[Table of Contents](#table-of-contents)

> **Architecture Reference:** [Configuration - Database Connections](configuration.md#configuration-file)

### PostgreSQL Connection

**Configuration Structure:**

```go
// PostgresConfig defines PostgreSQL connection parameters.
type PostgresConfig struct {
    Host            string        `yaml:"host" env:"MNEMONIC_DATABASE_POSTGRES_HOST"`
    Port            int           `yaml:"port" env:"MNEMONIC_DATABASE_POSTGRES_PORT"`
    Database        string        `yaml:"database" env:"MNEMONIC_DATABASE_POSTGRES_DATABASE"`
    Username        string        `yaml:"username" env:"MNEMONIC_DATABASE_POSTGRES_USERNAME"`
    Password        string        `yaml:"password" env:"MNEMONIC_DATABASE_POSTGRES_PASSWORD"`
    SSLMode         string        `yaml:"ssl_mode" env:"MNEMONIC_DATABASE_POSTGRES_SSL_MODE"`
    MaxOpenConns    int           `yaml:"max_open_conns" env:"MNEMONIC_DATABASE_POSTGRES_MAX_OPEN_CONNS"`
    MaxIdleConns    int           `yaml:"max_idle_conns" env:"MNEMONIC_DATABASE_POSTGRES_MAX_IDLE_CONNS"`
    ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" env:"MNEMONIC_DATABASE_POSTGRES_CONN_MAX_LIFETIME"`
}
```

**Recommended Pool Settings:**

| Setting | Development | Production (Single Pod) | Production (Multi-Pod) |
|---------|-------------|-------------------------|------------------------|
| MaxOpenConns | 5 | 25 | 15 per pod |
| MaxIdleConns | 2 | 5 | 3 per pod |
| ConnMaxLifetime | 5m | 1h | 1h |

### Neo4j Connection

**Configuration Structure:**

```go
// Neo4jConfig defines Neo4j connection parameters.
type Neo4jConfig struct {
    URI                          string        `yaml:"uri" env:"MNEMONIC_DATABASE_NEO4J_URI"`
    Username                     string        `yaml:"username" env:"MNEMONIC_DATABASE_NEO4J_USERNAME"`
    Password                     string        `yaml:"password" env:"MNEMONIC_DATABASE_NEO4J_PASSWORD"`
    Database                     string        `yaml:"database" env:"MNEMONIC_DATABASE_NEO4J_DATABASE"`
    MaxConnectionPoolSize        int           `yaml:"max_connection_pool_size" env:"MNEMONIC_DATABASE_NEO4J_MAX_POOL_SIZE"`
    ConnectionAcquisitionTimeout time.Duration `yaml:"connection_acquisition_timeout" env:"MNEMONIC_DATABASE_NEO4J_ACQUISITION_TIMEOUT"`
}
```

**Recommended Pool Settings:**

| Setting | Development | Production |
|---------|-------------|------------|
| MaxConnectionPoolSize | 10 | 50 |
| ConnectionAcquisitionTimeout | 30s | 60s |

### Connection String Formats

**PostgreSQL DSN:**

```text
# Standard format
postgres://username:password@host:port/database?sslmode=prefer

# With all options
postgres://mnemonic:secret@localhost:5432/mnemonic?sslmode=require&connect_timeout=10

# pgx-style (recommended for Go)
host=localhost port=5432 dbname=mnemonic user=mnemonic password=secret sslmode=prefer pool_max_conns=25
```

**Neo4j URI:**

```text
# Bolt protocol (standard)
bolt://localhost:7687

# Bolt with encryption
bolt+s://neo4j.example.com:7687

# Neo4j AuraDB (cloud)
neo4j+s://xxxx.databases.neo4j.io
```

**Environment Variable Examples:**

```bash
# PostgreSQL
export MNEMONIC_DATABASE_POSTGRES_HOST="localhost"
export MNEMONIC_DATABASE_POSTGRES_PORT="5432"
export MNEMONIC_DATABASE_POSTGRES_DATABASE="mnemonic"
export MNEMONIC_DATABASE_POSTGRES_USERNAME="mnemonic"
export MNEMONIC_DATABASE_POSTGRES_PASSWORD="secure-password"
export MNEMONIC_DATABASE_POSTGRES_SSL_MODE="require"

# Neo4j
export MNEMONIC_DATABASE_NEO4J_URI="bolt://localhost:7687"
export MNEMONIC_DATABASE_NEO4J_USERNAME="neo4j"
export MNEMONIC_DATABASE_NEO4J_PASSWORD="secure-password"
export MNEMONIC_DATABASE_NEO4J_DATABASE="neo4j"
```

## References

[Table of Contents](#table-of-contents)

**Architecture Documents:**

- [Data Architecture](../../architecture/04-data-architecture.md) - Data model design and storage decisions
- [System Architecture](../../architecture/02-system-architecture.md) - Component overview
- [Deployment Architecture](../../architecture/06-deployment-architecture.md) - Deployment patterns

**Design Documents:**

- [Pivot API Specification](2026-02-15-pivot-api-specification.md) - REST and MCP API contracts
- [Configuration](configuration.md) - Server configuration including database settings
- [Pattern Processing](pattern-processing.md) - Enrichment pipeline using this data layer

**External References:**

- [golang-migrate](https://github.com/golang-migrate/migrate) - Migration tool
- [pgx](https://github.com/jackc/pgx) - PostgreSQL driver for Go
- [pgvector](https://github.com/pgvector/pgvector) - Vector similarity search extension
- [neo4j-go-driver](https://github.com/neo4j/neo4j-go-driver) - Neo4j driver for Go
- [Claude Code Agent Skills Spec](https://docs.anthropic.com/en/docs/agents-and-tools/claude-code/skills) - Skill definition format
