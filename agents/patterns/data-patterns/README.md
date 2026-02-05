# Data Architecture Patterns

This directory contains comprehensive patterns for data modeling, schema design, and database operations for PostgreSQL and Neo4j (the Mnemonic stack).

## Overview

These patterns provide production-ready templates that Claude Code agents can reference when designing schemas, writing migrations, and implementing database operations.

## Pattern Categories

### PostgreSQL Patterns (`postgresql/`)

Core PostgreSQL patterns for relational data:

1. **SQL Migration Pattern** (`sql-migration-pattern.md`)
   - Up/down migration file structure
   - Naming conventions (`NNN_description.up.sql`)
   - Idempotent migration patterns
   - Transaction wrapping
   - golang-migrate compatibility

2. **Updated-At Trigger Pattern** (`updated-at-trigger-pattern.md`)
   - Reusable `update_updated_at()` function
   - Trigger creation per table
   - Timestamp handling best practices

3. **Soft Delete Pattern** (`soft-delete-pattern.md`)
   - `deleted_at` column approach
   - Partial indexes for active records
   - Views for hiding deleted records
   - Cascade considerations

4. **Audit Columns Pattern** (`audit-columns-pattern.md`)
   - Standard `created_at`, `updated_at` columns
   - Optional `created_by`, `updated_by` for user tracking
   - Default value strategies

5. **JSONB Validation Pattern** (`jsonb-validation-pattern.md`)
   - CHECK constraints for JSONB structure
   - Required key validation
   - Type validation within JSONB
   - GIN indexes for JSONB queries

**When to use these:**
- Schema migrations for any PostgreSQL database
- Implementing standard audit trails
- Soft-delete requirements
- Flexible metadata storage with JSONB

### pgvector Patterns (`pgvector/`)

Vector embedding patterns for AI/ML applications:

1. **pgvector Setup Pattern** (`pgvector-setup-pattern.md`)
   - Extension installation
   - Vector column definitions (1536, 3072 dimensions)
   - Index strategy decision matrix (IVFFlat vs HNSW)
   - Index tuning parameters
   - Storage estimates

2. **Similarity Search Pattern** (`similarity-search-pattern.md`)
   - Cosine similarity queries
   - L2 (Euclidean) distance queries
   - Filtered vector search
   - Hybrid search (vector + full-text)
   - Pagination patterns
   - Batch operations
   - Go implementation examples

**When to use these:**
- Semantic search features
- RAG (Retrieval Augmented Generation)
- Recommendation systems
- Content similarity matching

### Neo4j Patterns (`neo4j/`)

Graph database patterns for relationship-heavy data:

1. **Cypher Schema Pattern** (`cypher-schema-pattern.md`)
   - Naming conventions (labels, relationships, properties)
   - Uniqueness constraints
   - Existence constraints
   - Node keys (composite uniqueness)
   - Property indexes
   - Full-text indexes
   - Schema migration versioning

2. **Cypher Query Pattern** (`cypher-query-pattern.md`)
   - Node CRUD operations
   - Relationship operations
   - Variable-length path traversal
   - Shortest path queries
   - Aggregation and statistics
   - Full-text search
   - Go implementation with neo4j-go-driver

**When to use these:**
- Knowledge graphs
- Pattern relationships and similarity
- Multi-hop traversal queries
- Entity extraction and linking

### Design Patterns (`design/`)

Schema design documentation patterns:

1. **Schema Design Output Pattern** (`schema-design-output-pattern.md`)
   - Standardized output format for data-architect agent
   - Entity documentation template
   - Relationship documentation
   - Table specification format
   - Index specification format
   - Graph schema format
   - Hand-off instructions for data-engineer

**When to use these:**
- Documenting new schema designs
- Communicating between data-architect and data-engineer
- Creating migration sequences

## Cognee Integration

All patterns are loaded into Cognee's knowledge graph for agent retrieval:

```bash
# PostgreSQL patterns
# search(search_query="SQL migration pattern PostgreSQL", search_type="GRAPH_COMPLETION")
# search(search_query="soft delete pattern", search_type="GRAPH_COMPLETION")

# pgvector patterns
# search(search_query="pgvector setup embedding", search_type="GRAPH_COMPLETION")
# search(search_query="similarity search cosine", search_type="GRAPH_COMPLETION")

# Neo4j patterns
# search(search_query="Cypher schema constraints", search_type="GRAPH_COMPLETION")
# search(search_query="graph traversal query", search_type="GRAPH_COMPLETION")
```

## Related Agents

These Claude Code agents use these patterns:

- [data-architect-agent](../../agents/definitions/agnostic/data-architect-agent.md) - Schema design and data modeling
- [data-engineer-agent](../../agents/definitions/agnostic/data-engineer-agent.md) - SQL/Cypher implementation
- [go-software-agent](../../agents/definitions/go/go-software-agent.md) - Go repository implementation

## Pattern Structure

Each pattern file contains:

1. **Frontmatter**
   - `entity_name`: Pattern name for Cognee
   - `entity_type`: Pattern category (`database-pattern`, `documentation-pattern`)
   - `language`: Language specificity (`sql`, `cypher`, `agnostic`)
   - `domain`: Domain category (`backend`, `data-design`)
   - `description`: Pattern purpose
   - `tags`: Searchable keywords
   - `version`: Database version requirements
   - `related_patterns`: Links to related patterns

2. **Pattern Documentation**
   - Purpose and when to use
   - Complete SQL/Cypher examples
   - Naming conventions
   - Best practices

3. **Implementation Examples**
   - Working code snippets
   - Go integration examples
   - Query patterns

4. **Best Practices**
   - Performance considerations
   - Common issues and solutions
   - Migration strategies

## Database Stack (Mnemonic)

These patterns are designed for the Mnemonic stack:

| Database | Purpose | Patterns |
|----------|---------|----------|
| PostgreSQL | Source of truth, relational data | `postgresql/`, `pgvector/` |
| Neo4j | Knowledge graph, relationships | `neo4j/` |

### Data Flow

```
                    ┌─────────────────┐
                    │   PostgreSQL    │
                    │  (Source of     │
                    │    Truth)       │
                    └────────┬────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
              ▼              ▼              ▼
         Patterns      Embeddings     Enrichment
           CRUD        (pgvector)       Queue
              │              │              │
              └──────────────┼──────────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │     Neo4j       │
                    │  (Derived       │
                    │   Graph)        │
                    └─────────────────┘
```

## Migration Tool

These patterns assume [golang-migrate](https://github.com/golang-migrate/migrate):

```bash
# Install
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Create migration
migrate create -ext sql -dir migrations -seq create_users_table

# Run migrations
migrate -database "postgres://localhost/mydb?sslmode=disable" -path migrations up

# Rollback
migrate -database "postgres://localhost/mydb?sslmode=disable" -path migrations down 1
```

## Neo4j Tools

For Neo4j schema and queries:

```bash
# cypher-shell (bundled with Neo4j)
cypher-shell -u neo4j -p password < schema/neo4j-schema.cypher

# Go driver
go get github.com/neo4j/neo4j-go-driver/v5
```

## Deployment Independence

Database and application deployments are independent:

```
migrations/**     → Database CI/CD pipeline → Database deployment
internal/**       → Application CI/CD pipeline → Application deployment
```

Key principles:
- Migrations versioned separately from application
- Database changes can deploy without app changes
- Application can deploy without database changes
- Migrations run as init containers or standalone jobs

## Index Strategy Guide

### PostgreSQL

| Data Size | Index Type | Use Case |
|-----------|------------|----------|
| < 1,000 rows | btree | General purpose |
| Text search | GIN (pg_trgm) | LIKE queries |
| JSONB | GIN | Key/value queries |
| Arrays | GIN | Contains queries |

### pgvector

| Vector Count | Index Type | Trade-off |
|--------------|------------|-----------|
| < 1,000 | None (exact) | Perfect recall |
| 1K - 100K | IVFFlat | Fast build, good recall |
| 100K+ | HNSW | Slow build, best recall |

### Neo4j

| Use Case | Index Type |
|----------|------------|
| Unique lookup | Uniqueness constraint |
| Property filter | Property index |
| Text search | Full-text index |
| Composite | Composite index |

## Testing

### PostgreSQL Migrations

```bash
# Test migrations up and down
migrate -database "postgres://localhost/testdb?sslmode=disable" -path migrations up
migrate -database "postgres://localhost/testdb?sslmode=disable" -path migrations down

# Verify schema
psql -d testdb -c "\dt"
psql -d testdb -c "\di"
```

### Neo4j Schema

```bash
# Verify constraints
SHOW CONSTRAINTS;

# Verify indexes
SHOW INDEXES;

# Test query plan
EXPLAIN MATCH (p:Pattern {id: $id}) RETURN p;
```

## Contributing

When adding new patterns:

1. Use the standard frontmatter format (see `PATTERN-METADATA-SCHEMA.md`)
2. Include complete, working examples
3. Add version requirements
4. Document common issues
5. Provide Go integration examples where applicable
6. Update this README

## Additional Resources

- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [pgvector GitHub](https://github.com/pgvector/pgvector)
- [Neo4j Cypher Manual](https://neo4j.com/docs/cypher-manual/current/)
- [golang-migrate](https://github.com/golang-migrate/migrate)
- [neo4j-go-driver](https://github.com/neo4j/neo4j-go-driver)
- [pgvector-go](https://github.com/pgvector/pgvector-go)
