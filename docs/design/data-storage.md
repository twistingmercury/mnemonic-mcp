# Data Storage Implementation

[Back to Architecture Overview](../../architecture/00-overview.md) | [Back to Project README](../../../README.md)

## Table of Contents

- [Overview](#overview)
- [PostgreSQL Migrations](#postgresql-migrations)
  - [Migration File Structure](#migration-file-structure)
  - [Migration 001: Extensions and Functions](#migration-001-extensions-and-functions)
  - [Migration 002: Agents Table](#migration-002-agents-table)
  - [Migration 003: Patterns Table](#migration-003-patterns-table)
  - [Migration 004: Pattern-Agent Associations](#migration-004-pattern-agent-associations)
  - [Migration 005: Routing Rules Table](#migration-005-routing-rules-table)
  - [Migration 006: Enrichment Jobs Table](#migration-006-enrichment-jobs-table)
  - [Migration 007: Performance Indexes](#migration-007-performance-indexes)
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
  - [RoutingRule Repository](#routingrule-repository)
  - [EnrichmentJob Repository](#enrichmentjob-repository)
  - [GraphRepository](#graphrepository)
- [Connection Configuration](#connection-configuration)
  - [PostgreSQL Connection](#postgresql-connection)
  - [Neo4j Connection](#neo4j-connection)
  - [Connection String Formats](#connection-string-formats)
- [References](#references)

## Overview

[Table of Contents](#table-of-contents)

> **Architecture Reference:** [Data Architecture](../../architecture/08-data-architecture.md) | [System Architecture - Mnemonic](../../architecture/03-system-architecture.md#mnemonic)

This document provides implementation details for the Mnemonic data storage layer. It translates the architectural specifications from [Data Architecture](../../architecture/08-data-architecture.md) into concrete SQL migrations, Cypher queries, and Go interface definitions.

**Implementation Scope:**

| Component | Technology | Purpose |
|-----------|------------|---------|
| Relational Storage | PostgreSQL 15+ | Agents, patterns, rules, jobs |
| Vector Storage | PGVector extension | Pattern embeddings for semantic search |
| Graph Storage | Neo4j 5.x | Knowledge graph relationships |
| Migrations | golang-migrate | Schema versioning and deployment |

**Deployment Independence:**

Database migrations and application code are versioned and deployed independently. Migrations have their own CI/CD pipeline that triggers only on changes to the `migrations/` directory. This enables:

- Logic bug fixes in Go without database deployment
- Schema changes without rebuilding application containers
- Forward-compatible migrations for zero-downtime deployments

## PostgreSQL Migrations

[Table of Contents](#table-of-contents)

> **Architecture Reference:** [Data Architecture - Migration Strategy](../../architecture/08-data-architecture.md#migration-strategy)

### Migration File Structure

All PostgreSQL migrations follow the golang-migrate convention:

```text
src/mnemonic/
└── migrations/
    └── postgres/
        ├── 001_extensions_and_functions.up.sql
        ├── 001_extensions_and_functions.down.sql
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
        └── 007_create_performance_indexes.down.sql
```

**Running Migrations:**

```bash
# Apply all pending migrations
migrate -path src/mnemonic/migrations/postgres -database "$DATABASE_URL" up

# Rollback last migration
migrate -path src/mnemonic/migrations/postgres -database "$DATABASE_URL" down 1

# Check current version
migrate -path src/mnemonic/migrations/postgres -database "$DATABASE_URL" version
```

### Migration 001: Extensions and Functions

**Purpose:** Enable required PostgreSQL extensions and create reusable utility functions.

```sql
-- src/mnemonic/migrations/postgres/001_extensions_and_functions.up.sql
-- Enables required extensions and creates utility functions
-- Part of Mnemonic MVP

-- Enable UUID generation (if not using gen_random_uuid)
create extension if not exists "uuid-ossp";

-- Enable vector operations for embeddings
create extension if not exists vector;

-- Reusable trigger function for automatic updated_at timestamps
create or replace function update_updated_at()
returns trigger as $$
begin
    new.updated_at = now();
    return new;
end;
$$ language plpgsql;

comment on function update_updated_at() is
    'Trigger function to automatically update updated_at timestamp on row modification';
```

```sql
-- src/mnemonic/migrations/postgres/001_extensions_and_functions.down.sql
-- Reverses: Enables required extensions and creates utility functions
-- WARNING: Dropping extensions may fail if objects depend on them

drop function if exists update_updated_at() cascade;

-- Extensions are not dropped to avoid breaking other schemas
-- drop extension if exists vector;
-- drop extension if exists "uuid-ossp";
```

### Migration 002: Agents Table

**Purpose:** Create the agents table for storing agent definitions.

```sql
-- src/mnemonic/migrations/postgres/002_create_agents.up.sql
-- Creates the agents table for storing agent definitions
-- Part of Mnemonic MVP

create table if not exists agents (
    -- Primary key: lowercase-with-hyphens format, URL-safe
    name varchar(64) primary key,

    -- Agent metadata
    description varchar(500) not null,

    -- System prompt content (up to 50KB)
    system_prompt text not null,

    -- Model preference: sonnet, opus, haiku, or inherit from caller
    model varchar(20) not null default 'inherit',

    -- Allowed MCP tools (JSON array of tool names)
    allowed_tools jsonb not null default '[]'::jsonb,

    -- Keywords for fast routing (denormalized from routing_rules)
    routing_keywords jsonb not null default '[]'::jsonb,

    -- Audit timestamps
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),

    -- Constraints
    constraint agents_name_format
        check (name ~ '^[a-z][a-z0-9-]*$'),
    constraint agents_model_valid
        check (model in ('sonnet', 'opus', 'haiku', 'inherit')),
    constraint agents_system_prompt_length
        check (length(system_prompt) <= 51200),
    constraint agents_allowed_tools_array
        check (jsonb_typeof(allowed_tools) = 'array'),
    constraint agents_routing_keywords_array
        check (jsonb_typeof(routing_keywords) = 'array')
);

-- Trigger for automatic updated_at
create trigger trg_agents_updated_at
    before update on agents
    for each row execute function update_updated_at();

-- Table documentation
comment on table agents is 'Agent definitions for the routing system';
comment on column agents.name is 'Unique identifier, lowercase-with-hyphens format';
comment on column agents.model is 'Claude model preference: sonnet, opus, haiku, or inherit';
comment on column agents.allowed_tools is 'JSON array of MCP tool names this agent can use';
comment on column agents.routing_keywords is 'Denormalized keywords for fast routing lookups';
```

```sql
-- src/mnemonic/migrations/postgres/002_create_agents.down.sql
-- Reverses: Creates the agents table for storing agent definitions

drop trigger if exists trg_agents_updated_at on agents;
drop table if exists agents;
```

### Migration 003: Patterns Table

**Purpose:** Create the patterns table with PGVector embedding column for semantic search.

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

-- Trigger for automatic updated_at
create trigger trg_patterns_updated_at
    before update on patterns
    for each row execute function update_updated_at();

-- Table documentation
comment on table patterns is 'Reusable context patterns for prompt enrichment';
comment on column patterns.embedding is 'Vector embedding (1536d) for semantic similarity search';
comment on column patterns.enrichment_status is 'Processing state: pending, enriched, or failed';
comment on column patterns.tags is 'JSON array of categorization tags';
```

```sql
-- src/mnemonic/migrations/postgres/003_create_patterns.down.sql
-- Reverses: Creates the patterns table with vector embeddings

drop trigger if exists trg_patterns_updated_at on patterns;
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

-- Table documentation
comment on table pattern_agent_associations is
    'Many-to-many relationship between patterns and agents with relevance scores';
comment on column pattern_agent_associations.relevance is
    'Relevance score from 0.0 (not relevant) to 1.0 (highly relevant)';
```

```sql
-- src/mnemonic/migrations/postgres/004_create_pattern_agent_associations.down.sql
-- Reverses: Creates the pattern-agent association table

drop index if exists idx_pattern_agent_assoc_agent;
drop index if exists idx_pattern_agent_assoc_pattern;
drop table if exists pattern_agent_associations;
```

### Migration 005: Routing Rules Table

**Purpose:** Create the routing rules table for prompt-to-agent matching.

```sql
-- src/mnemonic/migrations/postgres/005_create_routing_rules.up.sql
-- Creates the routing rules table
-- Part of Mnemonic MVP

create table if not exists routing_rules (
    -- UUID primary key (rules may be renamed)
    id uuid primary key default gen_random_uuid(),

    -- Rule metadata
    name varchar(128) not null,

    -- Priority for evaluation order (0-1000, higher evaluated first)
    priority integer not null,

    -- Target agent for this rule
    agent_name varchar(64) not null,

    -- Match type determines match_config interpretation
    match_type varchar(20) not null,

    -- Type-specific match configuration (JSONB)
    match_config jsonb not null,

    -- Rule enabled/disabled state
    enabled boolean not null default true,

    -- Audit timestamps
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),

    -- Foreign key to agents
    constraint fk_routing_rules_agent
        foreign key (agent_name) references agents(name) on delete restrict,

    -- Constraints
    constraint routing_rules_name_unique unique (name),
    constraint routing_rules_priority_range
        check (priority >= 0 and priority <= 1000),
    constraint routing_rules_match_type_valid
        check (match_type in ('keyword', 'regex', 'pattern', 'default')),

    -- Match config validation based on match_type
    constraint routing_rules_match_config_valid check (
        (match_type = 'keyword' and
            match_config ? 'keywords' and
            match_config ? 'match_mode') or
        (match_type = 'regex' and
            match_config ? 'pattern') or
        (match_type = 'pattern' and
            match_config ? 'pattern_ids') or
        (match_type = 'default')
    )
);

-- Trigger for automatic updated_at
create trigger trg_routing_rules_updated_at
    before update on routing_rules
    for each row execute function update_updated_at();

-- Index for agent lookups
create index idx_routing_rules_agent
    on routing_rules(agent_name);

-- Table documentation
comment on table routing_rules is 'Rules for matching prompts to agents';
comment on column routing_rules.priority is 'Evaluation priority (0-1000), higher values evaluated first';
comment on column routing_rules.match_type is 'Match algorithm: keyword, regex, pattern (semantic), or default';
comment on column routing_rules.match_config is 'Type-specific configuration as JSONB';
```

```sql
-- src/mnemonic/migrations/postgres/005_create_routing_rules.down.sql
-- Reverses: Creates the routing rules table

drop trigger if exists trg_routing_rules_updated_at on routing_rules;
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

-- Trigger for automatic updated_at
create trigger trg_enrichment_jobs_updated_at
    before update on enrichment_jobs
    for each row execute function update_updated_at();

-- Index for pattern lookups
create index idx_enrichment_jobs_pattern
    on enrichment_jobs(pattern_id);

-- Table documentation
comment on table enrichment_jobs is 'Background processing queue for pattern enrichment';
comment on column enrichment_jobs.status is 'Job state: pending, processing, completed, or failed';
comment on column enrichment_jobs.scheduled_for is 'When the job should be processed (supports delayed retry)';
```

```sql
-- src/mnemonic/migrations/postgres/006_create_enrichment_jobs.down.sql
-- Reverses: Creates the enrichment jobs queue table

drop trigger if exists trg_enrichment_jobs_updated_at on enrichment_jobs;
drop index if exists idx_enrichment_jobs_pattern;
drop table if exists enrichment_jobs;
```

### Migration 007: Performance Indexes

**Purpose:** Create performance-optimized indexes for common query patterns.

```sql
-- src/mnemonic/migrations/postgres/007_create_performance_indexes.up.sql
-- Creates performance indexes for common query patterns
-- Part of Mnemonic MVP

-- Routing rules: enabled rules by priority (most common query)
create index idx_routing_rules_enabled_priority
    on routing_rules(priority desc, id)
    where enabled = true;

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
comment on index idx_routing_rules_enabled_priority is
    'Optimizes routing rule lookup by priority order';
comment on index idx_patterns_embedding is
    'IVFFlat index for vector similarity search (100 lists for MVP scale)';
comment on index idx_enrichment_jobs_pending is
    'Optimizes worker polling for pending jobs';
```

```sql
-- src/mnemonic/migrations/postgres/007_create_performance_indexes.down.sql
-- Reverses: Creates performance indexes for common query patterns

drop index if exists idx_patterns_search;
drop index if exists idx_patterns_tags;
drop index if exists idx_enrichment_jobs_processing;
drop index if exists idx_enrichment_jobs_pending;
drop index if exists idx_patterns_embedding;
drop index if exists idx_patterns_enriched;
drop index if exists idx_routing_rules_enabled_priority;
```

## PGVector Configuration

[Table of Contents](#table-of-contents)

> **Architecture Reference:** [Data Architecture - PGVector Configuration](../../architecture/08-data-architecture.md#pgvector-configuration)

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

> **Architecture Reference:** [Data Architecture - Neo4j Graph Model](../../architecture/08-data-architecture.md#neo4j-graph-model)

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

Mnemonic validates Neo4j schema constraints at startup to ensure the database is properly configured. This validation is advisory only - Mnemonic will not fail startup if constraints are missing, since Neo4j is used in a best-effort capacity.

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

**Example Log Output (Community Edition):**

```text
# All constraints present (Community Edition)
INFO  neo4j schema validation complete: all 3 constraints present

# Missing constraints
WARN  neo4j schema validation: missing constraints: pattern_id_unique, concept_name_unique
WARN  create missing constraints using: src/mnemonic/migrations/neo4j/001_create_constraints.cypher
```

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

> **Note:** These are interface definitions only. Implementation is handled by the go-software-agent.

### AgentRepository

```go
// AgentRepository defines data access operations for agents.
// Implementation: internal/repository/agent_repository.go
type AgentRepository interface {
    // Create stores a new agent. Returns ErrExists if name already exists.
    Create(ctx context.Context, agent *Agent) error

    // Get retrieves an agent by name. Returns ErrNotFound if not found.
    Get(ctx context.Context, name string) (*Agent, error)

    // Update modifies an existing agent. Returns ErrNotFound if not found.
    Update(ctx context.Context, agent *Agent) error

    // Delete removes an agent by name. Returns ErrInUse if referenced by rules.
    Delete(ctx context.Context, name string) error

    // List retrieves all agents with optional pagination.
    List(ctx context.Context, opts ListOptions) ([]*Agent, int64, error)

    // Exists checks if an agent with the given name exists.
    Exists(ctx context.Context, name string) (bool, error)
}

// Agent represents an agent definition.
type Agent struct {
    Name            string    `db:"name"`
    Description     string    `db:"description"`
    SystemPrompt    string    `db:"system_prompt"`
    Model           string    `db:"model"`
    AllowedTools    []string  `db:"-"` // Unmarshaled from JSONB
    RoutingKeywords []string  `db:"-"` // Unmarshaled from JSONB
    CreatedAt       time.Time `db:"created_at"`
    UpdatedAt       time.Time `db:"updated_at"`
}
```

### PatternRepository

```go
// PatternRepository defines data access operations for patterns.
// Implementation: internal/repository/pattern_repository.go
type PatternRepository interface {
    // Create stores a new pattern. Returns ErrNameExists if name exists.
    Create(ctx context.Context, pattern *Pattern) error

    // Get retrieves a pattern by ID. Returns ErrNotFound if not found.
    Get(ctx context.Context, id uuid.UUID) (*Pattern, error)

    // GetByName retrieves a pattern by name. Returns ErrNotFound if not found.
    GetByName(ctx context.Context, name string) (*Pattern, error)

    // Update modifies an existing pattern. Returns ErrNotFound if not found.
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

// Pattern represents a context pattern.
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
    Tags            []string // Filter by any of these tags
    EnrichmentStatus string  // Filter by enrichment status
    SearchQuery     string   // Full-text search in name/description
}

// SimilarityOptions defines options for similarity search.
type SimilarityOptions struct {
    MinSimilarity float64 // Minimum similarity threshold (0.0-1.0)
    MaxResults    int     // Maximum number of results
    Tags          []string // Optional tag filter
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

### RoutingRule Repository

```go
// RoutingRuleRepository defines data access operations for routing rules.
// Implementation: internal/repository/routing_rule_repository.go
type RoutingRuleRepository interface {
    // Create stores a new routing rule. Returns ErrNameExists if name exists.
    Create(ctx context.Context, rule *Rule) error

    // Get retrieves a routing rule by ID. Returns ErrNotFound if not found.
    Get(ctx context.Context, id uuid.UUID) (*Rule, error)

    // GetByName retrieves a routing rule by name. Returns ErrNotFound if not found.
    GetByName(ctx context.Context, name string) (*Rule, error)

    // Update modifies an existing routing rule. Returns ErrNotFound if not found.
    Update(ctx context.Context, rule *Rule) error

    // Delete removes a routing rule by ID. Returns ErrNotFound if not found.
    Delete(ctx context.Context, id uuid.UUID) error

    // List retrieves routing rules with filtering and pagination.
    List(ctx context.Context, filter Filter, opts ListOptions) ([]*Rule, int64, error)

    // ListEnabled retrieves all enabled rules ordered by priority (descending).
    // This is the primary method used by the routing engine.
    ListEnabled(ctx context.Context) ([]*Rule, error)

    // SetEnabled updates the enabled state of a rule.
    SetEnabled(ctx context.Context, id uuid.UUID, enabled bool) error

    // Exists checks if a routing rule with the given ID exists.
    Exists(ctx context.Context, id uuid.UUID) (bool, error)
}

// Rule represents a routing rule definition.
type Rule struct {
    ID          uuid.UUID   `db:"id"`
    Name        string      `db:"name"`
    Priority    int         `db:"priority"`
    AgentName   string      `db:"agent_name"`
    MatchType   string      `db:"match_type"`
    MatchConfig MatchConfig `db:"-"` // Unmarshaled from JSONB
    Enabled     bool        `db:"enabled"`
    CreatedAt   time.Time   `db:"created_at"`
    UpdatedAt   time.Time   `db:"updated_at"`
}

// MatchConfig represents type-specific match configuration.
// Use type assertion to access type-specific fields.
type MatchConfig interface {
    Type() string
}

// KeywordMatchConfig for match_type = 'keyword'
type KeywordMatchConfig struct {
    Keywords  []string `json:"keywords"`
    MatchMode string   `json:"match_mode"` // "any" or "all"
}

func (k KeywordMatchConfig) Type() string { return "keyword" }

// RegexMatchConfig for match_type = 'regex'
type RegexMatchConfig struct {
    Pattern string `json:"pattern"`
    Flags   string `json:"flags,omitempty"` // e.g., "i" for case-insensitive
}

func (r RegexMatchConfig) Type() string { return "regex" }

// PatternMatchConfig for match_type = 'pattern'
type PatternMatchConfig struct {
    PatternIDs []uuid.UUID `json:"pattern_ids"`
}

func (p PatternMatchConfig) Type() string { return "pattern" }

// DefaultMatchConfig for match_type = 'default'
type DefaultMatchConfig struct{}

func (d DefaultMatchConfig) Type() string { return "default" }

// Filter defines filtering options for rule queries.
type Filter struct {
    AgentName *string // Filter by target agent
    MatchType *string // Filter by match type
    Enabled   *bool   // Filter by enabled state
}
```

### EnrichmentJob Repository

```go
// EnrichmentJobRepository defines data access operations for enrichment jobs.
// Implementation: internal/repository/enrichment_job_repository.go
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
    // Returns the jobs, total count, and any error.
    List(ctx context.Context, filter Filter, opts ListOptions) ([]*Job, int64, error)
}

// Job represents a background enrichment task.
type Job struct {
    ID          uuid.UUID  `db:"id"`
    PatternID   uuid.UUID  `db:"pattern_id"`
    Status      string     `db:"status"`
    Attempts    int        `db:"attempts"`
    MaxAttempts int        `db:"max_attempts"`
    LastError   *string    `db:"last_error"`
    ScheduledFor time.Time `db:"scheduled_for"`
    StartedAt   *time.Time `db:"started_at"`
    CompletedAt *time.Time `db:"completed_at"`
    CreatedAt   time.Time  `db:"created_at"`
    UpdatedAt   time.Time  `db:"updated_at"`
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
// Implementation: internal/repository/graph_repository.go
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
    ErrInUse    = errors.New("agent is referenced by routing rules")
)

// pattern package errors
var (
    ErrNotFound   = errors.New("pattern not found")
    ErrNameExists = errors.New("pattern name already exists")
)

// routingrule package errors
var (
    ErrNotFound   = errors.New("routing rule not found")
    ErrNameExists = errors.New("routing rule name already exists")
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

- [Data Architecture](../../architecture/08-data-architecture.md) - Data model design and storage decisions
- [System Architecture](../../architecture/03-system-architecture.md) - Component overview
- [Deployment Architecture](../../architecture/05-deployment-architecture.md) - Deployment patterns

**Design Documents:**

- [Configuration](configuration.md) - Server configuration including database settings
- [Pattern Processing](pattern-processing.md) - Enrichment pipeline using this data layer
- [Routing Engine](routing-engine.md) - Routing using rules from this data layer

**External References:**

- [golang-migrate](https://github.com/golang-migrate/migrate) - Migration tool
- [pgx](https://github.com/jackc/pgx) - PostgreSQL driver for Go
- [pgvector](https://github.com/pgvector/pgvector) - Vector similarity search extension
- [neo4j-go-driver](https://github.com/neo4j/neo4j-go-driver) - Neo4j driver for Go
