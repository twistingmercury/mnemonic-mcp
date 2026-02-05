---
entity_name: pgvector Setup Pattern
entity_type: database-pattern
language: agnostic
domain: backend
description: PostgreSQL pgvector extension setup pattern including extension installation, vector columns, index strategies (IVFFlat, HNSW), and configuration for different scale requirements.
tags:
  - PostgreSQL
  - pgvector
  - embeddings
  - vector-search
  - AI
  - machine-learning
version: pgvector 0.5+
related_patterns:
  - pgvector Similarity Search Pattern
  - SQL Migration Pattern
---

# pgvector Setup Pattern

This pattern covers setting up PostgreSQL's pgvector extension for storing and querying vector embeddings.

## Overview

pgvector enables:
- Storage of high-dimensional vectors (embeddings from OpenAI, etc.)
- Similarity search using cosine, L2, or inner product distance
- Approximate nearest neighbor (ANN) search with indexes

## Extension Installation

### Migration to Enable Extension

```sql
-- migrations/001_enable_vector_extension.up.sql
-- Enables pgvector extension for vector embeddings.
-- Requires superuser or rds_superuser role.

create extension if not exists vector;

comment on extension vector is 'pgvector: vector similarity search';
```

```sql
-- migrations/001_enable_vector_extension.down.sql
-- WARNING: Drops all vector columns and indexes!
drop extension if exists vector cascade;
```

### Verify Installation

```sql
-- Check extension is installed
select * from pg_extension where extname = 'vector';

-- Check available operators
select opfname from pg_opfamily where opfname like '%vector%';
```

## Vector Column Types

### Common Embedding Dimensions

| Model | Dimensions | Column Definition |
|-------|------------|-------------------|
| OpenAI text-embedding-ada-002 | 1536 | `vector(1536)` |
| OpenAI text-embedding-3-small | 1536 | `vector(1536)` |
| OpenAI text-embedding-3-large | 3072 | `vector(3072)` |
| Cohere embed-english-v3.0 | 1024 | `vector(1024)` |
| Sentence Transformers (all-MiniLM) | 384 | `vector(384)` |

### Table with Vector Column

```sql
-- migrations/002_create_patterns_table.up.sql
create table if not exists patterns (
    id uuid primary key default gen_random_uuid(),
    name text not null,
    content text not null,

    -- Vector embedding (nullable until enriched)
    embedding vector(1536),

    -- Enrichment status
    enrichment_status text not null default 'pending'
        check (enrichment_status in ('pending', 'processing', 'enriched', 'failed')),

    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create trigger trg_patterns_updated_at
    before update on patterns
    for each row execute function update_updated_at();

comment on column patterns.embedding is
    'OpenAI text-embedding-3-small vector (1536 dimensions)';
```

## Index Strategies

### Decision Matrix

| Row Count | Index Type | Build Time | Query Speed | Recall | Memory |
|-----------|------------|------------|-------------|--------|--------|
| < 1,000 | None (exact) | N/A | Fast enough | 100% | Low |
| 1,000 - 100,000 | IVFFlat | Fast | Good | ~95% | Medium |
| 100,000+ | HNSW | Slow | Excellent | ~99% | High |

### No Index (Exact Search)

For small datasets, exact search is fast enough:

```sql
-- No index needed for < 1,000 rows
-- Queries do sequential scan with exact distance calculation

select id, name, 1 - (embedding <=> $1::vector) as similarity
from patterns
where embedding is not null
order by embedding <=> $1::vector
limit 10;
```

### IVFFlat Index

Inverted File Flat index - good balance of build speed and recall:

```sql
-- migrations/003_create_patterns_ivfflat_index.up.sql
-- Creates IVFFlat index for medium-scale vector search.
-- lists = sqrt(row_count) is a good starting point

create index if not exists idx_patterns_embedding_ivfflat
on patterns using ivfflat (embedding vector_cosine_ops)
with (lists = 100);

comment on index idx_patterns_embedding_ivfflat is
    'IVFFlat index for cosine similarity search, optimized for ~10K patterns';
```

**Tuning IVFFlat:**

| Parameter | Description | Rule of Thumb |
|-----------|-------------|---------------|
| `lists` | Number of clusters | `sqrt(row_count)` |
| `probes` (query time) | Clusters to search | Higher = better recall, slower |

```sql
-- Increase probes for better recall (at query time)
set ivfflat.probes = 10;  -- Default is 1
```

### HNSW Index

Hierarchical Navigable Small World - best recall, slower to build:

```sql
-- migrations/003_create_patterns_hnsw_index.up.sql
-- Creates HNSW index for large-scale vector search.
-- Better recall than IVFFlat but uses more memory.

create index if not exists idx_patterns_embedding_hnsw
on patterns using hnsw (embedding vector_cosine_ops)
with (m = 16, ef_construction = 64);

comment on index idx_patterns_embedding_hnsw is
    'HNSW index for cosine similarity search, optimized for 100K+ patterns';
```

**Tuning HNSW:**

| Parameter | Description | Trade-off |
|-----------|-------------|-----------|
| `m` | Max connections per node | Higher = better recall, more memory |
| `ef_construction` | Build-time search width | Higher = better index, slower build |
| `ef_search` (query time) | Query-time search width | Higher = better recall, slower query |

```sql
-- Increase ef_search for better recall (at query time)
set hnsw.ef_search = 100;  -- Default is 40
```

## Distance Functions

### Available Operators

| Operator | Function | Use Case |
|----------|----------|----------|
| `<=>` | Cosine distance | Text embeddings (most common) |
| `<->` | L2 (Euclidean) distance | Image embeddings |
| `<#>` | Inner product (negative) | When vectors are normalized |

### Index Operator Classes

| Distance | Operator Class | Index Definition |
|----------|----------------|------------------|
| Cosine | `vector_cosine_ops` | `using ivfflat (col vector_cosine_ops)` |
| L2 | `vector_l2_ops` | `using ivfflat (col vector_l2_ops)` |
| Inner Product | `vector_ip_ops` | `using ivfflat (col vector_ip_ops)` |

## Migration Example: Complete Setup

```sql
-- migrations/001_setup_vector_infrastructure.up.sql
-- Complete pgvector setup for pattern embeddings.

-- 1. Enable extension
create extension if not exists vector;

-- 2. Create utility function for updated_at (if not exists)
create or replace function update_updated_at()
returns trigger as $$
begin
    new.updated_at = now();
    return new;
end;
$$ language plpgsql;

-- 3. Create patterns table with vector column
create table if not exists patterns (
    id uuid primary key default gen_random_uuid(),
    name text not null unique,
    content text not null,
    embedding vector(1536),
    enrichment_status text not null default 'pending'
        check (enrichment_status in ('pending', 'processing', 'enriched', 'failed')),
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create trigger trg_patterns_updated_at
    before update on patterns
    for each row execute function update_updated_at();

-- 4. Create index (start with IVFFlat, migrate to HNSW at scale)
create index if not exists idx_patterns_embedding
on patterns using ivfflat (embedding vector_cosine_ops)
with (lists = 100);

-- 5. Index for enrichment queue
create index if not exists idx_patterns_enrichment_status
on patterns (enrichment_status)
where enrichment_status in ('pending', 'processing');
```

## Maintenance

### Rebuilding Indexes

After significant data changes, rebuild for optimal performance:

```sql
-- Rebuild IVFFlat index
reindex index idx_patterns_embedding;

-- Or drop and recreate with new parameters
drop index idx_patterns_embedding;
create index idx_patterns_embedding
on patterns using ivfflat (embedding vector_cosine_ops)
with (lists = 200);  -- Adjusted for new row count
```

### Monitoring Index Usage

```sql
-- Check if index is being used
explain analyze
select id from patterns
order by embedding <=> '[0.1, 0.2, ...]'::vector
limit 10;

-- Look for "Index Scan using idx_patterns_embedding"
```

### Storage Estimates

```sql
-- Estimate storage for vectors
select
    count(*) as rows,
    pg_size_pretty(count(*) * 1536 * 4) as vector_data_size,
    pg_size_pretty(pg_relation_size('patterns')) as table_size,
    pg_size_pretty(pg_indexes_size('patterns')) as index_size
from patterns
where embedding is not null;
```

## Best Practices

1. **Start without index** - For < 1,000 rows, exact search is fine
2. **Use IVFFlat first** - Faster to build, good for iteration
3. **Migrate to HNSW at scale** - When you need better recall
4. **Match index to distance function** - Use `vector_cosine_ops` with `<=>`
5. **Tune at query time** - Adjust probes/ef_search for recall vs speed
6. **Rebuild after bulk loads** - IVFFlat benefits from reindexing
7. **Monitor query plans** - Ensure indexes are actually used

## Common Issues

### Extension Not Found

```
ERROR: could not open extension control file ".../vector.control": No such file or directory
```

Solution: Install pgvector on the server or use a managed service that supports it.

### Index Not Used

```sql
-- Force index usage for testing
set enable_seqscan = off;

-- Check query plan
explain select * from patterns order by embedding <=> $1 limit 10;
```

### Out of Memory During Index Build

For HNSW with large datasets:

```sql
-- Increase maintenance_work_mem
set maintenance_work_mem = '2GB';

-- Then create index
create index ...;
```
