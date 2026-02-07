---
entity_name: Neo4j CE/EE Tiered Migration Pattern
entity_type: database-pattern
language: agnostic
domain: backend
description: Neo4j migration pattern for separating Community Edition and Enterprise Edition features into tiered migration files with edition-aware test infrastructure.
tags:
  - Neo4j
  - Cypher
  - graph-database
  - migrations
  - Community-Edition
  - Enterprise-Edition
version: Neo4j 4.3+
related_patterns:
  - Cypher Schema Pattern
  - Cypher Query Pattern
---

# Neo4j CE/EE Tiered Migration Pattern

This pattern handles Neo4j schema migrations when features differ between Community Edition (CE) and Enterprise Edition (EE), preventing deployment failures in CE environments.

## References

- Neo4j Community vs Enterprise: https://neo4j.com/docs/operations-manual/current/installation/
- Relationship property indexes (4.3+): https://neo4j.com/docs/cypher-manual/4.3/indexes/

## Problem

Some Neo4j features are only available in Enterprise Edition:

- Existence constraints (`IS NOT NULL`)
- Node key constraints (composite uniqueness)
- Property type constraints

Mixing CE and EE features in the same migration file causes failures in CE environments (local development, CI pipelines) where Enterprise Edition is not available.

## Solution

Separate migrations into edition-specific tiers:

- **CE-compatible migrations**: Applied in all environments (dev, CI, staging, prod)
- **EE-only migrations**: Skipped in CE environments, applied only in Enterprise Edition

## Edition Feature Matrix

| Feature                                      | Community | Enterprise |
| -------------------------------------------- | --------- | ---------- |
| Uniqueness constraints                       | Yes       | Yes        |
| Existence constraints (IS NOT NULL)          | No        | Yes        |
| Node key constraints                         | No        | Yes        |
| Property type constraints                    | No        | Yes        |
| Node property indexes                        | Yes       | Yes        |
| Relationship property indexes (4.3+)         | Yes       | Yes        |
| Full-text indexes                            | Yes       | Yes        |
| Composite indexes                            | Yes       | Yes        |
| Vector indexes (5.11+)                       | Yes       | Yes        |

**Important**: Relationship property indexes became CE-compatible in Neo4j 4.3 (2021). All index types are CE-compatible.

## Migration File Structure

### Directory Organization (Phase 8A Example)

```text
migrations/neo4j/
├── 001_create_constraints.cypher           # Uniqueness constraints (CE + EE)
├── 002_create_existence_constraints.cypher # Existence constraints (EE only)
└── 003_create_indexes.cypher              # All indexes (CE + EE)
```

### CE-Compatible Migration (001)

```cypher
// Migration 001: Create uniqueness constraints
// Edition: Community + Enterprise
// Prerequisites: None
// Rollback: DROP CONSTRAINT IF EXISTS (CE: v0, EE: v0)

// Pattern uniqueness constraint
CREATE CONSTRAINT unique_pattern_id IF NOT EXISTS
FOR (p:Pattern) REQUIRE p.id IS UNIQUE;

// Agent uniqueness constraint
CREATE CONSTRAINT unique_agent_name IF NOT EXISTS
FOR (a:Agent) REQUIRE a.name IS UNIQUE;

// Concept uniqueness constraint
CREATE CONSTRAINT unique_concept_name IF NOT EXISTS
FOR (c:Concept) REQUIRE c.name IS UNIQUE;

// Update schema version
MERGE (v:SchemaVersion {name: 'mnemonic'})
SET v.version = 1,
    v.migratedAt = datetime()
RETURN v.version AS version;
```

### EE-Only Migration (002)

```cypher
// Migration 002: Create existence constraints
// Edition: ENTERPRISE ONLY - Skip in Community Edition
// Prerequisites: Migration 001 completed
// Rollback: DROP CONSTRAINT IF EXISTS (CE: v1, EE: v1)
//
// WARNING: This migration will fail in Community Edition.
//          Test runners must explicitly skip this file in CE environments.

// Pattern property existence
CREATE CONSTRAINT pattern_id_exists IF NOT EXISTS
FOR (p:Pattern) REQUIRE p.id IS NOT NULL;

CREATE CONSTRAINT pattern_name_exists IF NOT EXISTS
FOR (p:Pattern) REQUIRE p.name IS NOT NULL;

// Agent property existence
CREATE CONSTRAINT agent_name_exists IF NOT EXISTS
FOR (a:Agent) REQUIRE a.name IS NOT NULL;

// Concept property existence
CREATE CONSTRAINT concept_name_exists IF NOT EXISTS
FOR (c:Concept) REQUIRE c.name IS NOT NULL;

// Update schema version
MERGE (v:SchemaVersion {name: 'mnemonic'})
SET v.version = 2,
    v.migratedAt = datetime()
RETURN v.version AS version;
```

### CE-Compatible Migration (003)

```cypher
// Migration 003: Create indexes
// Edition: Community + Enterprise
// Prerequisites: Migration 001 completed (CE), Migration 002 completed (EE)
// Rollback: DROP INDEX IF EXISTS (CE: v1, EE: v2)

// Node property indexes
CREATE INDEX idx_pattern_name IF NOT EXISTS
FOR (p:Pattern) ON (p.name);

CREATE INDEX idx_agent_name IF NOT EXISTS
FOR (a:Agent) ON (a.name);

CREATE INDEX idx_concept_type IF NOT EXISTS
FOR (c:Concept) ON (c.type);

// Relationship property indexes (Neo4j 4.3+, CE-compatible)
CREATE INDEX rel_relevant_for_relevance IF NOT EXISTS
FOR ()-[r:RELEVANT_FOR]-() ON (r.relevance);

CREATE INDEX rel_contains_weight IF NOT EXISTS
FOR ()-[r:CONTAINS]-() ON (r.weight);

// Composite index
CREATE INDEX idx_concept_type_name IF NOT EXISTS
FOR (c:Concept) ON (c.type, c.name);

// Full-text indexes
CREATE FULLTEXT INDEX pattern_content_fulltext IF NOT EXISTS
FOR (p:Pattern) ON EACH [p.name, p.content];

// Update schema version
MERGE (v:SchemaVersion {name: 'mnemonic'})
SET v.version = 3,
    v.migratedAt = datetime()
RETURN v.version AS version;
```

## Test Runner Implementation

### Edition Detection

```bash
#!/usr/bin/env bash
# neo4j-test-runner.sh

detect_neo4j_edition() {
    local edition
    edition=$(docker exec neo4j-test cypher-shell \
        "CALL dbms.components() YIELD edition RETURN edition;" 2>/dev/null | \
        grep -oE 'community|enterprise' || echo "community")
    echo "$edition"
}

NEO4J_EDITION=$(detect_neo4j_edition)
echo "Detected Neo4j Edition: $NEO4J_EDITION"
```

### CE-Safe Migration Runner

```bash
run_migrations() {
    local edition="$1"

    for file in migrations/neo4j/*.cypher; do
        local filename=$(basename "$file")

        # Skip EE-only files in CE environments
        if [[ "$edition" == "community" ]] && [[ "$filename" == *"existence"* ]]; then
            echo "Skipping EE-only migration: $filename (Community Edition detected)"
            continue
        fi

        echo "Running migration: $filename"
        docker exec neo4j-test cypher-shell < "$file" || {
            echo "Migration failed: $filename"
            return 1
        }
    done
}

run_migrations "$NEO4J_EDITION"
```

## Test Assertions

### Edition-Aware Object Counts

```bash
# test-migrations.bats

@test "CE: Verify constraint count (uniqueness only)" {
  skip_if_not_ce

  run docker exec neo4j-test cypher-shell \
    "SHOW CONSTRAINTS YIELD name RETURN count(*) AS count;"

  # Only uniqueness constraints (migration 001)
  assert_output --partial "| 3    |"
}

@test "EE: Verify constraint count (uniqueness + existence)" {
  skip_if_not_ee

  run docker exec neo4j-test cypher-shell \
    "SHOW CONSTRAINTS YIELD name RETURN count(*) AS count;"

  # Uniqueness + existence constraints (migrations 001 + 002)
  assert_output --partial "| 7    |"
}

@test "Verify index count (all editions)" {
  run docker exec neo4j-test cypher-shell \
    "SHOW INDEXES YIELD name RETURN count(*) AS count;"

  # All indexes (migration 003)
  assert_output --partial "| 8    |"
}

# Helper functions
skip_if_not_ce() {
  [[ "$NEO4J_EDITION" == "community" ]] || skip "Test requires Community Edition"
}

skip_if_not_ee() {
  [[ "$NEO4J_EDITION" == "enterprise" ]] || skip "Test requires Enterprise Edition"
}
```

### SchemaVersion Verification

```bash
@test "CE: Verify final schema version" {
  skip_if_not_ce

  run docker exec neo4j-test cypher-shell \
    "MATCH (v:SchemaVersion {name: 'mnemonic'}) RETURN v.version AS version;"

  # CE skips migration 002, final version is 3
  assert_output --partial "| 3    |"
}

@test "EE: Verify final schema version" {
  skip_if_not_ee

  run docker exec neo4j-test cypher-shell \
    "MATCH (v:SchemaVersion {name: 'mnemonic'}) RETURN v.version AS version;"

  # EE runs all migrations, final version is 3
  assert_output --partial "| 3    |"
}
```

## Rollback Strategy

### SchemaVersion Branching

```cypher
// Rollback migration 003
// Edition-aware rollback targets:
// - CE: Roll back to version 1 (001 only)
// - EE: Roll back to version 2 (001 + 002)

// Drop indexes
DROP INDEX idx_pattern_name IF EXISTS;
DROP INDEX idx_agent_name IF EXISTS;
DROP INDEX idx_concept_type IF EXISTS;
DROP INDEX rel_relevant_for_relevance IF EXISTS;
DROP INDEX rel_contains_weight IF EXISTS;
DROP INDEX idx_concept_type_name IF EXISTS;
DROP INDEX pattern_content_fulltext IF EXISTS;

// Update version (branches by edition)
CALL dbms.components() YIELD edition
WITH CASE edition
    WHEN 'community' THEN 1    // CE: back to 001
    WHEN 'enterprise' THEN 2   // EE: back to 002
    END AS targetVersion
MERGE (v:SchemaVersion {name: 'mnemonic'})
SET v.version = targetVersion,
    v.rolledBackAt = datetime()
RETURN v.version AS version;
```

## Key Rules

1. **EE-only files contain ONLY EE-only features** - Never mix CE-compatible features into EE-only files
2. **All indexes are CE-compatible** - Never put indexes in EE-only files (node property, relationship property, full-text, composite, vector)
3. **EE-only files must have clear header documentation** - State the edition requirement explicitly with WARNING
4. **Test runners must explicitly skip EE-only migrations** - When running against CE
5. **BATS test counts must account for reduced object count** - Use edition-aware assertions
6. **SchemaVersion rollback must branch by edition** - Different version targets for CE vs EE
7. **All statements use IF NOT EXISTS / IF EXISTS** - For idempotent execution

## Best Practices

1. **Detect edition at runtime** - Use `CALL dbms.components() YIELD edition` for dynamic behavior
2. **Fail fast in test runners** - Detect CE before attempting EE-only operations
3. **Document rollback targets** - Include CE and EE version targets in migration headers
4. **Use descriptive filenames** - Make edition requirements obvious (e.g., `*_existence_constraints.cypher`)
5. **Test both editions in CI** - Run separate CE and EE test jobs
6. **Keep EE files minimal** - Only include features that require Enterprise Edition
7. **Group by feature type** - Separate files for constraints, indexes, etc.

## Common Pitfalls

### Mixing Indexes with EE Constraints

```cypher
// BAD: Index in EE-only file
// Migration 002: Create existence constraints (EE only)

CREATE CONSTRAINT pattern_id_exists IF NOT EXISTS
FOR (p:Pattern) REQUIRE p.id IS NOT NULL;

// WRONG: This index is CE-compatible but in EE-only file
CREATE INDEX idx_pattern_name IF NOT EXISTS
FOR (p:Pattern) ON (p.name);
```

```cypher
// GOOD: Separate indexes into CE-compatible file
// Migration 002: Create existence constraints (EE only)

CREATE CONSTRAINT pattern_id_exists IF NOT EXISTS
FOR (p:Pattern) REQUIRE p.id IS NOT NULL;

// Migration 003: Create indexes (CE + EE)

CREATE INDEX idx_pattern_name IF NOT EXISTS
FOR (p:Pattern) ON (p.name);
```

### Incorrect Test Counts

```bash
# BAD: Same count for all editions
@test "Verify constraint count" {
  run docker exec neo4j-test cypher-shell \
    "SHOW CONSTRAINTS YIELD name RETURN count(*) AS count;"
  assert_output --partial "| 7    |"  # Fails in CE
}

# GOOD: Edition-aware counts
@test "CE: Verify constraint count" {
  skip_if_not_ce
  assert_output --partial "| 3    |"  # Uniqueness only
}

@test "EE: Verify constraint count" {
  skip_if_not_ee
  assert_output --partial "| 7    |"  # Uniqueness + existence
}
```

