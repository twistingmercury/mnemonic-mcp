---
entity_name: pgvector Similarity Search Pattern
entity_type: database-pattern
language: agnostic
domain: backend
description: PostgreSQL pgvector query patterns for similarity search including cosine similarity, filtering, pagination, and hybrid search combining vector and relational queries.
tags:
  - PostgreSQL
  - pgvector
  - similarity-search
  - embeddings
  - semantic-search
version: pgvector 0.5+
related_patterns:
  - pgvector Setup Pattern
  - SQL Migration Pattern
---

# pgvector Similarity Search Pattern

This pattern covers query patterns for vector similarity search using pgvector.

## Basic Similarity Search

### Cosine Similarity (Most Common)

```sql
-- Find 10 most similar patterns to a query vector
-- <=> returns cosine distance, so 1 - distance = similarity

select
    id,
    name,
    content,
    1 - (embedding <=> $1::vector) as similarity
from patterns
where embedding is not null
order by embedding <=> $1::vector
limit 10;
```

### With Minimum Similarity Threshold

```sql
-- Only return results above similarity threshold
-- Cosine distance < 0.3 means similarity > 0.7

select
    id,
    name,
    content,
    1 - (embedding <=> $1::vector) as similarity
from patterns
where embedding is not null
  and (embedding <=> $1::vector) < 0.3  -- similarity > 0.7
order by embedding <=> $1::vector
limit 10;
```

### L2 (Euclidean) Distance

```sql
-- For image embeddings or when L2 is preferred
-- <-> returns L2 distance (lower is more similar)

select
    id,
    name,
    embedding <-> $1::vector as distance
from patterns
where embedding is not null
order by embedding <-> $1::vector
limit 10;
```

## Filtered Search

### Filter Then Search (Recommended)

```sql
-- Filter by relational columns, then vector search
-- More efficient when filter is selective

select
    id,
    name,
    content,
    1 - (embedding <=> $1::vector) as similarity
from patterns
where embedding is not null
  and is_active = true
  and category = 'technology'
order by embedding <=> $1::vector
limit 10;
```

### Search Then Filter (Less Efficient)

```sql
-- When you need exactly N results after filtering
-- Use subquery to get more candidates

select * from (
    select
        id,
        name,
        content,
        category,
        1 - (embedding <=> $1::vector) as similarity
    from patterns
    where embedding is not null
    order by embedding <=> $1::vector
    limit 100  -- Get more candidates
) as candidates
where category = 'technology'
limit 10;
```

## Hybrid Search

### Vector + Full-Text Search

```sql
-- Combine semantic similarity with keyword matching

select
    id,
    name,
    content,
    1 - (embedding <=> $1::vector) as vector_similarity,
    ts_rank(to_tsvector('english', content), plainto_tsquery('english', $2)) as text_rank,
    -- Combined score (adjust weights as needed)
    (0.7 * (1 - (embedding <=> $1::vector))) +
    (0.3 * ts_rank(to_tsvector('english', content), plainto_tsquery('english', $2))) as combined_score
from patterns
where embedding is not null
  and to_tsvector('english', content) @@ plainto_tsquery('english', $2)
order by combined_score desc
limit 10;
```

### Vector + Recency Boost

```sql
-- Boost recent content in similarity results

select
    id,
    name,
    content,
    created_at,
    1 - (embedding <=> $1::vector) as similarity,
    -- Recency score: 1.0 for today, decaying over 30 days
    greatest(0, 1 - extract(epoch from (now() - created_at)) / (30 * 86400)) as recency,
    -- Combined score
    (0.8 * (1 - (embedding <=> $1::vector))) +
    (0.2 * greatest(0, 1 - extract(epoch from (now() - created_at)) / (30 * 86400))) as final_score
from patterns
where embedding is not null
order by final_score desc
limit 10;
```

## Pagination

### Offset-Based (Simple but Slow for Deep Pages)

```sql
-- Page 3, 10 items per page
select
    id,
    name,
    1 - (embedding <=> $1::vector) as similarity
from patterns
where embedding is not null
order by embedding <=> $1::vector
limit 10 offset 20;
```

### Keyset Pagination (Better Performance)

```sql
-- First page
select
    id,
    name,
    embedding <=> $1::vector as distance
from patterns
where embedding is not null
order by embedding <=> $1::vector, id
limit 10;

-- Next page (using last distance and id from previous page)
select
    id,
    name,
    embedding <=> $1::vector as distance
from patterns
where embedding is not null
  and (embedding <=> $1::vector, id) > ($2, $3)  -- $2=last_distance, $3=last_id
order by embedding <=> $1::vector, id
limit 10;
```

## Batch Operations

### Find Similar for Multiple Queries

```sql
-- Using LATERAL join for multiple query vectors

with query_vectors as (
    select unnest($1::vector[]) as query_embedding,
           generate_series(1, array_length($1::vector[], 1)) as query_index
)
select
    qv.query_index,
    p.id,
    p.name,
    1 - (p.embedding <=> qv.query_embedding) as similarity
from query_vectors qv
cross join lateral (
    select id, name, embedding
    from patterns
    where embedding is not null
    order by embedding <=> qv.query_embedding
    limit 5
) p
order by qv.query_index, similarity desc;
```

### Bulk Similarity Matrix

```sql
-- Find similarity between all pairs (expensive!)
-- Only for small datasets

select
    p1.id as id1,
    p2.id as id2,
    1 - (p1.embedding <=> p2.embedding) as similarity
from patterns p1
cross join patterns p2
where p1.id < p2.id  -- Avoid duplicates and self-comparison
  and p1.embedding is not null
  and p2.embedding is not null
  and (p1.embedding <=> p2.embedding) < 0.3  -- Only high similarity
order by similarity desc
limit 100;
```

## Performance Optimization

### Ensure Index Usage

```sql
-- Check query plan
explain (analyze, buffers)
select id from patterns
where embedding is not null
order by embedding <=> '[0.1, 0.2, ...]'::vector
limit 10;

-- Should show "Index Scan using idx_patterns_embedding"
```

### Tune IVFFlat Probes

```sql
-- Increase probes for better recall (slower)
set ivfflat.probes = 10;

-- Default is 1, higher = better recall, more computation
-- Good values: 1-5 for speed, 10-50 for recall
```

### Tune HNSW ef_search

```sql
-- Increase ef_search for better recall (slower)
set hnsw.ef_search = 100;

-- Default is 40, higher = better recall, more computation
-- Good values: 40-100 for balance, 200+ for high recall
```

## Go Implementation

### Basic Search

```go
func (r *PatternRepository) FindSimilar(ctx context.Context, embedding []float32, limit int) ([]PatternWithSimilarity, error) {
    query := `
        SELECT id, name, content, 1 - (embedding <=> $1::vector) as similarity
        FROM patterns
        WHERE embedding IS NOT NULL
        ORDER BY embedding <=> $1::vector
        LIMIT $2
    `

    // Convert []float32 to pgvector format
    vectorStr := fmt.Sprintf("[%s]", floatsToString(embedding))

    rows, err := r.pool.Query(ctx, query, vectorStr, limit)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var results []PatternWithSimilarity
    for rows.Next() {
        var p PatternWithSimilarity
        if err := rows.Scan(&p.ID, &p.Name, &p.Content, &p.Similarity); err != nil {
            return nil, err
        }
        results = append(results, p)
    }
    return results, rows.Err()
}
```

### Using pgvector-go

```go
import "github.com/pgvector/pgvector-go"

func (r *PatternRepository) FindSimilar(ctx context.Context, embedding pgvector.Vector, limit int) ([]PatternWithSimilarity, error) {
    query := `
        SELECT id, name, content, 1 - (embedding <=> $1) as similarity
        FROM patterns
        WHERE embedding IS NOT NULL
        ORDER BY embedding <=> $1
        LIMIT $2
    `

    rows, err := r.pool.Query(ctx, query, embedding, limit)
    // ... handle results
}

func (r *PatternRepository) UpdateEmbedding(ctx context.Context, id uuid.UUID, embedding pgvector.Vector) error {
    _, err := r.pool.Exec(ctx, `
        UPDATE patterns SET embedding = $1, enrichment_status = 'enriched'
        WHERE id = $2
    `, embedding, id)
    return err
}
```

## Query Patterns Summary

| Use Case | Query Pattern |
|----------|---------------|
| Basic similarity | `ORDER BY embedding <=> $1 LIMIT n` |
| With threshold | `WHERE distance < threshold ORDER BY ...` |
| Filtered search | `WHERE filter_col = val ORDER BY embedding <=> $1` |
| Hybrid (vector + text) | Combine scores with weights |
| Pagination | Keyset with `(distance, id)` |
| Multiple queries | LATERAL join |

## Best Practices

1. **Always LIMIT** - Vector search without limit is expensive
2. **Filter first** - Apply relational filters before vector search when selective
3. **Tune recall vs speed** - Adjust probes/ef_search based on requirements
4. **Use appropriate distance** - Cosine for text, L2 for images
5. **Check query plans** - Ensure indexes are used
6. **Handle NULL embeddings** - Always filter `WHERE embedding IS NOT NULL`
7. **Normalize similarity** - Convert distance to 0-1 similarity for API responses
