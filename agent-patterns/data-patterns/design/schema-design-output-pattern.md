---
entity_name: Schema Design Output Pattern
entity_type: documentation-pattern
language: agnostic
domain: data-design
description: Standardized output format for data-architect agent schema designs, ensuring consistent handoff to data-engineer for implementation.
tags:
  - schema-design
  - data-modeling
  - ERD
  - documentation
  - architecture
version: "1.0"
related_patterns:
  - SQL Migration Pattern
  - Cypher Schema Pattern
---

# Schema Design Output Pattern

This pattern defines the standardized output format for data-architect schema designs, ensuring clear communication and consistent handoff to data-engineer for implementation.

## Overview

When designing a database schema, the data-architect agent should produce output in this format to ensure:
- Clear entity and relationship documentation
- Unambiguous column specifications
- Explicit index requirements
- Graph schema (if applicable)
- Clear implementation instructions

## Output Template

```markdown
# Schema Design: [Name]

## Overview

[Brief description of what this schema supports, the domain, and key decisions]

## Entities

| Entity | Description |
|--------|-------------|
| EntityName | What it represents, key characteristics |

## Relationships

| From | Relationship | To | Cardinality | Notes |
|------|--------------|-----|-------------|-------|
| Entity1 | verb_phrase | Entity2 | 1:N | Optional context |

## Tables

### table_name

[Description of the table's purpose]

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | uuid | PK, DEFAULT gen_random_uuid() | Primary key |
| name | text | NOT NULL, UNIQUE | Human-readable name |
| foreign_id | uuid | FK -> other_table(id) | Reference to other entity |
| status | text | NOT NULL, CHECK(...) | Status enum |
| data | jsonb | DEFAULT '{}' | Flexible metadata |
| created_at | timestamptz | NOT NULL, DEFAULT now() | Creation timestamp |
| updated_at | timestamptz | NOT NULL, DEFAULT now() | Last modification |

**Triggers:**
- `trg_table_name_updated_at` - Auto-update updated_at on modification

**Notes:**
- [Any special considerations for this table]

## Indexes

| Index Name | Table | Columns | Type | Rationale |
|------------|-------|---------|------|-----------|
| idx_table_col | table_name | (column) | btree | Query pattern X |
| idx_table_multi | table_name | (col1, col2) | btree | Join optimization |
| idx_table_partial | table_name | (column) WHERE condition | btree | Filter on active records |
| idx_table_vector | table_name | (embedding) | ivfflat | Similarity search |

## Graph Schema (Neo4j)

### Node Labels

| Label | Description | Key Properties |
|-------|-------------|----------------|
| :Pattern | Knowledge pattern | id, name, content |
| :Concept | Extracted concept | name, type |

### Relationship Types

| Type | From | To | Properties | Description |
|------|------|-----|------------|-------------|
| :CONTAINS | Pattern | Concept | weight | Pattern contains concept |
| :RELATES_TO | Pattern | Pattern | weight, reason | Similar patterns |

### Constraints

```cypher
// Uniqueness
CREATE CONSTRAINT pattern_id_unique FOR (p:Pattern) REQUIRE p.id IS UNIQUE;

// Existence
CREATE CONSTRAINT pattern_name_exists FOR (p:Pattern) REQUIRE p.name IS NOT NULL;
```

### Indexes

```cypher
CREATE INDEX pattern_name FOR (p:Pattern) ON (p.name);
CREATE FULLTEXT INDEX pattern_content FOR (p:Pattern) ON EACH [p.name, p.content];
```

## Hand-off to data-engineer

### Migration Sequence

1. **Migration 001: Extensions and utilities**
   - Enable pgvector extension
   - Create update_updated_at() function

2. **Migration 002: Create table_name**
   - Table definition as specified above
   - Trigger for updated_at
   - Primary indexes

3. **Migration 003: Create other_table**
   - Table definition
   - Foreign key to table_name
   - Additional indexes

4. **Migration 004: Neo4j schema setup**
   - Constraints as specified
   - Indexes as specified

### Implementation Notes

- [Special considerations]
- [Data type rationale]
- [Index sizing estimates]
- [Expected query patterns]

### Compatibility

- **Minimum PostgreSQL version:** 14
- **Required extensions:** pgvector
- **Neo4j version:** 5.0+
```

## Section Details

### Entities Section

Document each entity with:
- **Name:** PascalCase, singular (User, not Users)
- **Description:** What it represents in the domain
- **Key characteristics:** Important constraints or behaviors

Example:
```markdown
| Entity | Description |
|--------|-------------|
| Agent | A specialized AI agent with defined capabilities and system prompt |
| Pattern | A reusable knowledge pattern for prompt enrichment |
| RoutingRule | A rule that matches prompts to agents by keyword, regex, or pattern |
```

### Relationships Section

Document each relationship with:
- **From/To:** Entity names (not table names)
- **Relationship:** Verb phrase describing the relationship
- **Cardinality:** 1:1, 1:N, N:M
- **Notes:** Important constraints or behaviors

Example:
```markdown
| From | Relationship | To | Cardinality | Notes |
|------|--------------|-----|-------------|-------|
| Pattern | relevant_for | Agent | N:M | Many-to-many via junction table |
| RoutingRule | routes_to | Agent | N:1 | Each rule points to one agent |
| Pattern | contains | Concept | 1:N | Patterns have multiple concepts |
```

### Tables Section

For each table, include:
- **Purpose description**
- **Column table with all columns**
- **Triggers** (especially updated_at)
- **Notes** for special considerations

Column specifications:
```markdown
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
```

Use these constraint formats:
- `PK` - Primary key
- `FK -> table(column)` - Foreign key
- `NOT NULL` - Required
- `UNIQUE` - Unique constraint
- `DEFAULT value` - Default value
- `CHECK (condition)` - Check constraint

### Indexes Section

Include all indexes with:
- **Name:** Following `idx_table_columns` convention
- **Type:** btree, hash, gin, ivfflat, hnsw
- **Rationale:** Why this index exists (query pattern it supports)

### Graph Schema Section

Only include if the design involves Neo4j:
- **Node Labels:** With descriptions and key properties
- **Relationship Types:** With directions and properties
- **Constraints:** Cypher statements
- **Indexes:** Cypher statements

### Hand-off Section

Critical for data-engineer:
1. **Migration sequence:** Numbered order of migrations
2. **Implementation notes:** Special considerations
3. **Compatibility:** Version requirements

## Example: Complete Schema Design

```markdown
# Schema Design: Mnemonic Core

## Overview

Core schema for the Mnemonic routing service. Stores agents, patterns, and routing rules.
PostgreSQL is the source of truth; Neo4j stores derived graph relationships.

## Entities

| Entity | Description |
|--------|-------------|
| Agent | AI agent with system prompt and model preferences |
| Pattern | Knowledge pattern for semantic matching |
| RoutingRule | Rule matching prompts to agents |
| Concept | Entity extracted from pattern content (Neo4j only) |

## Relationships

| From | Relationship | To | Cardinality | Notes |
|------|--------------|-----|-------------|-------|
| RoutingRule | routes_to | Agent | N:1 | FK constraint |
| Pattern | relevant_for | Agent | N:M | Via junction table |
| Pattern | contains | Concept | 1:N | Graph only |
| Pattern | relates_to | Pattern | N:M | Graph only, similarity |

## Tables

### agents

Core agent definitions.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| name | text | PK | Unique agent identifier |
| description | text | NOT NULL | Human-readable description |
| system_prompt | text | NOT NULL | Agent's system instructions |
| model_preference | text | NOT NULL, DEFAULT 'default' | Preferred model |
| is_active | boolean | NOT NULL, DEFAULT true | Whether agent is enabled |
| created_at | timestamptz | NOT NULL, DEFAULT now() | Creation time |
| updated_at | timestamptz | NOT NULL, DEFAULT now() | Last update |

**Triggers:**
- `trg_agents_updated_at`

### patterns

Knowledge patterns for semantic enrichment.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | uuid | PK, DEFAULT gen_random_uuid() | Pattern identifier |
| name | text | NOT NULL, UNIQUE | Pattern name |
| content | text | NOT NULL | Pattern content |
| embedding | vector(1536) | | OpenAI embedding |
| enrichment_status | text | NOT NULL, DEFAULT 'pending', CHECK | Status |
| created_at | timestamptz | NOT NULL, DEFAULT now() | Creation time |
| updated_at | timestamptz | NOT NULL, DEFAULT now() | Last update |

**Triggers:**
- `trg_patterns_updated_at`

**Notes:**
- embedding is nullable until enrichment completes
- enrichment_status CHECK: ('pending', 'processing', 'enriched', 'failed')

## Indexes

| Index Name | Table | Columns | Type | Rationale |
|------------|-------|---------|------|-----------|
| idx_patterns_name | patterns | (name) | btree | Name lookups |
| idx_patterns_embedding | patterns | (embedding) | ivfflat(100) | Similarity search |
| idx_patterns_status | patterns | (enrichment_status) | btree | Enrichment queue |

## Graph Schema (Neo4j)

### Node Labels

| Label | Description | Key Properties |
|-------|-------------|----------------|
| :Pattern | Knowledge pattern | id (UUID string), name |
| :Agent | AI agent | name |
| :Concept | Extracted concept | name, type |

### Relationship Types

| Type | From | To | Properties | Description |
|------|------|-----|------------|-------------|
| :RELEVANT_FOR | Pattern | Agent | score | Pattern relevance |
| :CONTAINS | Pattern | Concept | weight | Pattern has concept |
| :RELATES_TO | Pattern | Pattern | weight, reason | Similar patterns |

### Constraints

```cypher
CREATE CONSTRAINT pattern_id_unique FOR (p:Pattern) REQUIRE p.id IS UNIQUE;
CREATE CONSTRAINT agent_name_unique FOR (a:Agent) REQUIRE a.name IS UNIQUE;
CREATE CONSTRAINT concept_name_unique FOR (c:Concept) REQUIRE c.name IS UNIQUE;
```

## Hand-off to data-engineer

### Migration Sequence

1. **001_extensions**: Enable pgvector
2. **002_utility_functions**: Create update_updated_at()
3. **003_create_agents**: agents table with trigger
4. **004_create_patterns**: patterns table with vector, trigger, indexes
5. **005_create_routing_rules**: routing_rules table with FK
6. **006_neo4j_schema**: Neo4j constraints and indexes

### Implementation Notes

- Start with IVFFlat index (lists=100), migrate to HNSW at 100K+ patterns
- enrichment_status index is partial: only pending/processing states

### Compatibility

- PostgreSQL 14+
- pgvector 0.5+
- Neo4j 5.0+
```

## Using This Pattern

1. **data-architect** produces output in this format
2. **Main Claude** reviews and gets user approval
3. **data-engineer** receives the approved design and creates migrations
4. **go-software-agent** implements repositories after migrations exist
