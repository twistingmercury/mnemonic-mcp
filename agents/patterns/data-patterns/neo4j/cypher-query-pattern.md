---
entity_name: Cypher Query Pattern
entity_type: database-pattern
language: agnostic
domain: backend
description: Neo4j Cypher query patterns for common graph operations including node CRUD, relationship traversal, path finding, aggregation, and pattern matching.
tags:
  - Neo4j
  - Cypher
  - graph-database
  - graph-queries
  - traversal
version: Neo4j 5.0+
related_patterns:
  - Cypher Schema Pattern
---

# Cypher Query Pattern

This pattern covers common Cypher query patterns for Neo4j graph operations.

## Node Operations

### Create Node

```cypher
// Simple create
CREATE (p:Pattern {
    id: $id,
    name: $name,
    content: $content,
    createdAt: datetime()
})
RETURN p;

// Create with multiple labels
CREATE (d:Document:Pattern {
    id: $id,
    name: $name
})
RETURN d;
```

### Merge (Create or Match)

```cypher
// Create if not exists, otherwise match
MERGE (a:Agent {name: $name})
ON CREATE SET
    a.createdAt = datetime(),
    a.description = $description
ON MATCH SET
    a.updatedAt = datetime()
RETURN a;
```

### Find Node

```cypher
// By unique property
MATCH (p:Pattern {id: $id})
RETURN p;

// By multiple conditions
MATCH (p:Pattern)
WHERE p.name CONTAINS $searchTerm
  AND p.createdAt > datetime() - duration('P7D')
RETURN p
ORDER BY p.createdAt DESC
LIMIT 10;
```

### Update Node

```cypher
// Update properties
MATCH (p:Pattern {id: $id})
SET p.name = $name,
    p.content = $content,
    p.updatedAt = datetime()
RETURN p;

// Add/remove labels
MATCH (p:Pattern {id: $id})
SET p:Featured
REMOVE p:Draft
RETURN p;
```

### Delete Node

```cypher
// Delete node (only if no relationships)
MATCH (p:Pattern {id: $id})
DELETE p;

// Delete node and all its relationships
MATCH (p:Pattern {id: $id})
DETACH DELETE p;
```

## Relationship Operations

### Create Relationship

```cypher
// Create relationship between existing nodes
MATCH (p:Pattern {id: $patternId})
MATCH (c:Concept {name: $conceptName})
CREATE (p)-[:CONTAINS {weight: $weight}]->(c)
RETURN p, c;

// Merge relationship (create if not exists)
MATCH (p:Pattern {id: $patternId})
MATCH (c:Concept {name: $conceptName})
MERGE (p)-[r:CONTAINS]->(c)
ON CREATE SET r.weight = $weight, r.createdAt = datetime()
RETURN p, r, c;
```

### Create Node and Relationship Together

```cypher
// Create concept and link to pattern in one query
MATCH (p:Pattern {id: $patternId})
MERGE (c:Concept {name: $conceptName})
ON CREATE SET c.type = $type, c.createdAt = datetime()
MERGE (p)-[r:CONTAINS]->(c)
ON CREATE SET r.weight = $weight
RETURN p, r, c;
```

### Delete Relationship

```cypher
// Delete specific relationship
MATCH (p:Pattern {id: $patternId})-[r:CONTAINS]->(c:Concept {name: $conceptName})
DELETE r;

// Delete all relationships of a type from a node
MATCH (p:Pattern {id: $patternId})-[r:CONTAINS]->()
DELETE r;
```

## Traversal Patterns

### Direct Neighbors

```cypher
// Get all concepts contained in a pattern
MATCH (p:Pattern {id: $patternId})-[:CONTAINS]->(c:Concept)
RETURN c;

// Get all patterns relevant for an agent
MATCH (p:Pattern)-[:RELEVANT_FOR]->(a:Agent {name: $agentName})
RETURN p
ORDER BY p.name;
```

### Variable-Length Paths

```cypher
// Find patterns connected through shared concepts (2 hops)
MATCH (p1:Pattern {id: $patternId})-[:CONTAINS]->(c:Concept)<-[:CONTAINS]-(p2:Pattern)
WHERE p1 <> p2
RETURN p2, collect(c.name) AS sharedConcepts
ORDER BY size(sharedConcepts) DESC
LIMIT 10;

// Find all patterns within 3 relationship hops
MATCH (p1:Pattern {id: $patternId})-[*1..3]-(p2:Pattern)
WHERE p1 <> p2
RETURN DISTINCT p2;
```

### Shortest Path

```cypher
// Find shortest path between two patterns
MATCH path = shortestPath(
    (p1:Pattern {id: $id1})-[*]-(p2:Pattern {id: $id2})
)
RETURN path, length(path) AS hops;

// All shortest paths (may be multiple)
MATCH path = allShortestPaths(
    (p1:Pattern {id: $id1})-[*]-(p2:Pattern {id: $id2})
)
RETURN path;
```

## Aggregation

### Count by Label

```cypher
// Count nodes by label
MATCH (p:Pattern) RETURN count(p) AS patternCount;
MATCH (c:Concept) RETURN count(c) AS conceptCount;
MATCH (a:Agent) RETURN count(a) AS agentCount;

// Multiple counts in one query
MATCH (p:Pattern) WITH count(p) AS patterns
MATCH (c:Concept) WITH patterns, count(c) AS concepts
MATCH (a:Agent)
RETURN patterns, concepts, count(a) AS agents;
```

### Group By

```cypher
// Count concepts by type
MATCH (c:Concept)
RETURN c.type AS type, count(c) AS count
ORDER BY count DESC;

// Count patterns by number of concepts
MATCH (p:Pattern)
OPTIONAL MATCH (p)-[:CONTAINS]->(c:Concept)
WITH p, count(c) AS conceptCount
RETURN conceptCount, count(p) AS patternCount
ORDER BY conceptCount;
```

### Statistics

```cypher
// Average, min, max relationship weights
MATCH (p:Pattern)-[r:RELATES_TO]->(other:Pattern)
RETURN
    avg(r.weight) AS avgWeight,
    min(r.weight) AS minWeight,
    max(r.weight) AS maxWeight,
    count(r) AS relationshipCount;
```

## Full-Text Search

```cypher
// Search patterns by content (requires full-text index)
CALL db.index.fulltext.queryNodes('pattern_content_fulltext', $searchTerm)
YIELD node, score
RETURN node.id AS id, node.name AS name, score
ORDER BY score DESC
LIMIT 10;

// Full-text search with additional filters
CALL db.index.fulltext.queryNodes('pattern_content_fulltext', $searchTerm)
YIELD node, score
WHERE node.createdAt > datetime() - duration('P30D')
RETURN node, score
ORDER BY score DESC
LIMIT 10;
```

## Common Query Patterns

### Find Related Patterns via Shared Concepts

```cypher
// Most similar patterns based on shared concepts
MATCH (p1:Pattern {id: $patternId})-[:CONTAINS]->(c:Concept)<-[:CONTAINS]-(p2:Pattern)
WHERE p1 <> p2
WITH p2, count(c) AS sharedConceptCount, collect(c.name) AS sharedConcepts
RETURN p2.id, p2.name, sharedConceptCount, sharedConcepts
ORDER BY sharedConceptCount DESC
LIMIT 10;
```

### Find Patterns for Agent with Concepts

```cypher
// Get patterns relevant to agent, with their concepts
MATCH (p:Pattern)-[rel:RELEVANT_FOR]->(a:Agent {name: $agentName})
OPTIONAL MATCH (p)-[:CONTAINS]->(c:Concept)
WITH p, rel, collect(c.name) AS concepts
RETURN p.id, p.name, rel.score AS relevanceScore, concepts
ORDER BY rel.score DESC
LIMIT 20;
```

### Build Pattern-to-Pattern Similarity

```cypher
// Create RELATES_TO relationships based on shared concepts
MATCH (p1:Pattern)-[:CONTAINS]->(c:Concept)<-[:CONTAINS]-(p2:Pattern)
WHERE p1.id < p2.id  // Avoid duplicates
WITH p1, p2, count(c) AS sharedCount

// Calculate Jaccard similarity
MATCH (p1)-[:CONTAINS]->(c1:Concept)
WITH p1, p2, sharedCount, count(c1) AS p1Count
MATCH (p2)-[:CONTAINS]->(c2:Concept)
WITH p1, p2, sharedCount, p1Count, count(c2) AS p2Count

// Only create relationship if similarity above threshold
WITH p1, p2, sharedCount,
     toFloat(sharedCount) / (p1Count + p2Count - sharedCount) AS jaccard
WHERE jaccard > 0.3

MERGE (p1)-[r:RELATES_TO]->(p2)
SET r.weight = jaccard,
    r.sharedConcepts = sharedCount,
    r.updatedAt = datetime()
RETURN count(r) AS relationshipsCreated;
```

### Delete Orphan Concepts

```cypher
// Find and delete concepts not linked to any pattern
MATCH (c:Concept)
WHERE NOT (c)<-[:CONTAINS]-(:Pattern)
DELETE c
RETURN count(c) AS deletedCount;
```

## Parameterized Queries (Go)

### Using neo4j-go-driver

```go
import "github.com/neo4j/neo4j-go-driver/v5/neo4j"

func (r *GraphRepository) FindRelatedPatterns(ctx context.Context, patternID uuid.UUID, limit int) ([]Pattern, error) {
    session := r.driver.NewSession(ctx, neo4j.SessionConfig{
        AccessMode: neo4j.AccessModeRead,
    })
    defer session.Close(ctx)

    result, err := session.Run(ctx, `
        MATCH (p1:Pattern {id: $id})-[:CONTAINS]->(c:Concept)<-[:CONTAINS]-(p2:Pattern)
        WHERE p1 <> p2
        WITH p2, count(c) AS sharedCount
        RETURN p2.id AS id, p2.name AS name, sharedCount
        ORDER BY sharedCount DESC
        LIMIT $limit
    `, map[string]interface{}{
        "id":    patternID.String(),
        "limit": limit,
    })
    if err != nil {
        return nil, err
    }

    var patterns []Pattern
    for result.Next(ctx) {
        record := result.Record()
        // ... extract values
    }
    return patterns, result.Err()
}
```

## Performance Tips

### Use EXPLAIN/PROFILE

```cypher
// Show query plan
EXPLAIN MATCH (p:Pattern {id: $id})-[:CONTAINS]->(c:Concept) RETURN c;

// Run and show actual metrics
PROFILE MATCH (p:Pattern {id: $id})-[:CONTAINS]->(c:Concept) RETURN c;
```

### Index Hints

```cypher
// Force use of specific index
MATCH (p:Pattern)
USING INDEX p:Pattern(name)
WHERE p.name = $name
RETURN p;
```

### Limit Early

```cypher
// BAD: Collect all then limit
MATCH (p:Pattern)-[:CONTAINS]->(c:Concept)
WITH p, collect(c) AS concepts
RETURN p, concepts
LIMIT 10;

// GOOD: Limit patterns first
MATCH (p:Pattern)
WITH p LIMIT 10
MATCH (p)-[:CONTAINS]->(c:Concept)
RETURN p, collect(c) AS concepts;
```

## Best Practices

1. **Always parameterize** - Never concatenate strings into queries
2. **Use MERGE for idempotency** - Prefer MERGE over CREATE for upserts
3. **Index lookup first** - Start MATCH with indexed property
4. **DETACH DELETE for nodes** - Prevents orphan relationships
5. **Use WITH for staging** - Break complex queries into stages
6. **EXPLAIN before PROFILE** - Check plan before running
7. **Limit traversal depth** - Always bound variable-length paths
