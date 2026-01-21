---
entity_name: Cypher Schema Pattern
entity_type: database-pattern
language: agnostic
domain: backend
description: Neo4j Cypher pattern for schema setup including node labels, relationship types, uniqueness constraints, property indexes, and full-text indexes.
tags:
  - Neo4j
  - Cypher
  - graph-database
  - constraints
  - indexes
version: Neo4j 5.0+
related_patterns:
  - Cypher Query Pattern
  - SQL Migration Pattern
---

# Cypher Schema Pattern

This pattern covers Neo4j schema setup using Cypher, including constraints, indexes, and schema conventions.

## Naming Conventions

| Element | Convention | Example |
|---------|------------|---------|
| Node Labels | PascalCase | `:Pattern`, `:Agent`, `:Concept` |
| Relationship Types | UPPER_SNAKE_CASE | `:CONTAINS`, `:RELATES_TO`, `:MENTIONED_IN` |
| Properties | camelCase | `id`, `name`, `createdAt` |
| Constraints | snake_case with suffix | `pattern_id_unique`, `agent_name_exists` |
| Indexes | snake_case | `pattern_name`, `concept_type_name` |

## Constraints

### Uniqueness Constraint

Ensures a property value is unique across all nodes with a label:

```cypher
// Unique ID for each Pattern
CREATE CONSTRAINT pattern_id_unique IF NOT EXISTS
FOR (p:Pattern) REQUIRE p.id IS UNIQUE;

// Unique name for each Agent
CREATE CONSTRAINT agent_name_unique IF NOT EXISTS
FOR (a:Agent) REQUIRE a.name IS UNIQUE;

// Unique name for each Concept
CREATE CONSTRAINT concept_name_unique IF NOT EXISTS
FOR (c:Concept) REQUIRE c.name IS UNIQUE;
```

### Existence Constraint (Property Required)

Ensures a property exists on all nodes with a label:

```cypher
// Pattern must have id
CREATE CONSTRAINT pattern_id_exists IF NOT EXISTS
FOR (p:Pattern) REQUIRE p.id IS NOT NULL;

// Pattern must have name
CREATE CONSTRAINT pattern_name_exists IF NOT EXISTS
FOR (p:Pattern) REQUIRE p.name IS NOT NULL;

// Agent must have name
CREATE CONSTRAINT agent_name_exists IF NOT EXISTS
FOR (a:Agent) REQUIRE a.name IS NOT NULL;
```

### Node Key (Composite Uniqueness)

Ensures combination of properties is unique:

```cypher
// Entity type + name combination must be unique
CREATE CONSTRAINT entity_type_name_key IF NOT EXISTS
FOR (e:Entity) REQUIRE (e.type, e.name) IS NODE KEY;
```

### Relationship Property Existence

```cypher
// RELATES_TO must have weight
CREATE CONSTRAINT relates_to_weight_exists IF NOT EXISTS
FOR ()-[r:RELATES_TO]-() REQUIRE r.weight IS NOT NULL;
```

## Indexes

### Property Index (Single Property)

```cypher
// Index for Pattern name lookups
CREATE INDEX pattern_name IF NOT EXISTS
FOR (p:Pattern) ON (p.name);

// Index for Concept type filtering
CREATE INDEX concept_type IF NOT EXISTS
FOR (c:Concept) ON (c.type);

// Index for timestamp queries
CREATE INDEX pattern_created_at IF NOT EXISTS
FOR (p:Pattern) ON (p.createdAt);
```

### Composite Index (Multiple Properties)

```cypher
// Index for filtering by type and name together
CREATE INDEX entity_type_name IF NOT EXISTS
FOR (e:Entity) ON (e.type, e.name);
```

### Full-Text Index

For text search across properties:

```cypher
// Full-text index on Pattern content
CREATE FULLTEXT INDEX pattern_content_fulltext IF NOT EXISTS
FOR (p:Pattern) ON EACH [p.name, p.content];

// Full-text index on Concept descriptions
CREATE FULLTEXT INDEX concept_description_fulltext IF NOT EXISTS
FOR (c:Concept) ON EACH [c.name, c.description];
```

### Range Index (Neo4j 5.0+)

For range queries on numeric/temporal properties:

```cypher
// Range index for numeric comparisons
CREATE RANGE INDEX pattern_priority_range IF NOT EXISTS
FOR (p:Pattern) ON (p.priority);
```

## Complete Schema Setup Script

```cypher
// schema/neo4j-schema.cypher
// Neo4j schema setup for Mnemonic knowledge graph

// ============================================
// CONSTRAINTS
// ============================================

// Pattern constraints
CREATE CONSTRAINT pattern_id_unique IF NOT EXISTS
FOR (p:Pattern) REQUIRE p.id IS UNIQUE;

CREATE CONSTRAINT pattern_id_exists IF NOT EXISTS
FOR (p:Pattern) REQUIRE p.id IS NOT NULL;

CREATE CONSTRAINT pattern_name_exists IF NOT EXISTS
FOR (p:Pattern) REQUIRE p.name IS NOT NULL;

// Agent constraints
CREATE CONSTRAINT agent_name_unique IF NOT EXISTS
FOR (a:Agent) REQUIRE a.name IS UNIQUE;

CREATE CONSTRAINT agent_name_exists IF NOT EXISTS
FOR (a:Agent) REQUIRE a.name IS NOT NULL;

// Concept constraints
CREATE CONSTRAINT concept_name_unique IF NOT EXISTS
FOR (c:Concept) REQUIRE c.name IS UNIQUE;

CREATE CONSTRAINT concept_name_exists IF NOT EXISTS
FOR (c:Concept) REQUIRE c.name IS NOT NULL;

// ============================================
// INDEXES
// ============================================

// Pattern indexes
CREATE INDEX pattern_name IF NOT EXISTS
FOR (p:Pattern) ON (p.name);

CREATE INDEX pattern_created_at IF NOT EXISTS
FOR (p:Pattern) ON (p.createdAt);

// Concept indexes
CREATE INDEX concept_type IF NOT EXISTS
FOR (c:Concept) ON (c.type);

CREATE INDEX concept_type_name IF NOT EXISTS
FOR (c:Concept) ON (c.type, c.name);

// Full-text indexes
CREATE FULLTEXT INDEX pattern_content_fulltext IF NOT EXISTS
FOR (p:Pattern) ON EACH [p.name, p.content];

// ============================================
// VERIFICATION
// ============================================

// Show all constraints
SHOW CONSTRAINTS;

// Show all indexes
SHOW INDEXES;
```

## Node Label Design

### When to Use Multiple Labels

```cypher
// Single label (common)
CREATE (p:Pattern {id: $id, name: $name})

// Multiple labels (for categorization)
CREATE (d:Document:Pattern {id: $id, name: $name})
CREATE (c:Code:Pattern {id: $id, name: $name})

// Query by specific label
MATCH (p:Code) RETURN p
```

### Label Inheritance Pattern

```cypher
// Base pattern with type property instead of multiple labels
CREATE (p:Pattern {
    id: $id,
    type: 'code',  // or 'document', 'config', etc.
    name: $name
})

// Index the type for efficient filtering
CREATE INDEX pattern_type IF NOT EXISTS
FOR (p:Pattern) ON (p.type);
```

## Relationship Design

### Relationship Types

```cypher
// Pattern contains concepts
(p:Pattern)-[:CONTAINS]->(c:Concept)

// Pattern relates to other patterns (similarity)
(p1:Pattern)-[:RELATES_TO {weight: 0.85}]->(p2:Pattern)

// Pattern is relevant for agent
(p:Pattern)-[:RELEVANT_FOR {score: 0.9}]->(a:Agent)

// Concept mentioned in pattern (inverse of CONTAINS)
(c:Concept)-[:MENTIONED_IN]->(p:Pattern)
```

### Relationship Properties

```cypher
// Store metadata on relationships
CREATE (p1)-[:RELATES_TO {
    weight: 0.85,
    reason: 'shared_concepts',
    createdAt: datetime()
}]->(p2)
```

## Schema Migration Pattern

### Version Tracking

```cypher
// Store schema version in database
MERGE (v:SchemaVersion {name: 'mnemonic'})
SET v.version = 3,
    v.migratedAt = datetime()
```

### Migration Script Structure

```cypher
// migrations/neo4j/003_add_concept_type_index.cypher
// Migration 003: Add index on Concept type property
//
// Prerequisites: Migration 002 completed
// Rollback: DROP INDEX concept_type IF EXISTS

// Check current version
MATCH (v:SchemaVersion {name: 'mnemonic'})
WHERE v.version < 3
WITH v

// Apply migration
CREATE INDEX concept_type IF NOT EXISTS
FOR (c:Concept) ON (c.type);

// Update version
MERGE (v:SchemaVersion {name: 'mnemonic'})
SET v.version = 3, v.migratedAt = datetime();
```

## Verification Queries

### List All Constraints

```cypher
SHOW CONSTRAINTS
YIELD name, type, entityType, labelsOrTypes, properties
RETURN name, type, entityType, labelsOrTypes, properties
ORDER BY name;
```

### List All Indexes

```cypher
SHOW INDEXES
YIELD name, type, entityType, labelsOrTypes, properties, state
RETURN name, type, entityType, labelsOrTypes, properties, state
ORDER BY name;
```

### Check Constraint Violations

```cypher
// Find Patterns without required id
MATCH (p:Pattern)
WHERE p.id IS NULL
RETURN p LIMIT 10;

// Find duplicate Pattern ids
MATCH (p:Pattern)
WITH p.id AS id, count(*) AS count
WHERE count > 1
RETURN id, count;
```

## Best Practices

1. **Use IF NOT EXISTS** - Makes scripts idempotent
2. **Constraint before index** - Unique constraints create indexes automatically
3. **Index what you query** - Only add indexes for frequent query patterns
4. **Full-text for search** - Use full-text indexes for text search
5. **Document relationships** - Include comments explaining relationship meaning
6. **Version your schema** - Track schema changes for migrations

## Common Issues

### Constraint Creation Fails

```
Neo.ClientError.Schema.ConstraintValidationFailed
```

Existing data violates the constraint. Fix data first:

```cypher
// Find and fix duplicates before adding unique constraint
MATCH (p:Pattern)
WITH p.name AS name, collect(p) AS patterns
WHERE size(patterns) > 1
UNWIND patterns[1..] AS duplicate
DELETE duplicate;

// Then add constraint
CREATE CONSTRAINT pattern_name_unique IF NOT EXISTS
FOR (p:Pattern) REQUIRE p.name IS UNIQUE;
```

### Index Not Used

Check with EXPLAIN:

```cypher
EXPLAIN MATCH (p:Pattern {name: 'test'}) RETURN p
```

Look for "NodeIndexSeek" instead of "NodeByLabelScan".
