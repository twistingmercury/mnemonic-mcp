-- src/migrations/postgres/000009_pattern_schema_chunks.up.sql
-- Adds structured metadata columns to patterns, introduces chunk-based
-- semantic search via pattern_chunks, and updates enrichment_jobs to
-- support chunk-level enrichment jobs.
-- Part of Mnemonic MVP
--
-- Copyright 2025, Mnemonic Authors
--
-- Dependencies:
--   - 000003_create_patterns (for patterns table)
--   - 000005_create_enrichment_jobs (for enrichment_jobs table)
--   - 000006_create_performance_indexes (for idx_patterns_embedding, which is dropped)

-- =============================================================================
-- PATTERNS TABLE — drop embedding, remove content limit, add metadata columns
-- =============================================================================

-- Drop the 10KB content constraint (pattern files can exceed this)
alter table patterns drop constraint if exists patterns_content_length;

-- Drop embedding from patterns (embeddings move to per-chunk rows)
alter table patterns drop column if exists embedding;

-- Add structured metadata columns populated from pattern file YAML frontmatter
alter table patterns
    add column entity_type      varchar(100) not null default '',
    add column language         varchar(50)  not null default '',
    add column domain           varchar(50)  not null default '',
    add column version          varchar(50),
    add column related_patterns jsonb        not null default '[]'::jsonb;

-- Enforce that related_patterns is always a JSON array
alter table patterns
    add constraint patterns_related_patterns_array
        check (jsonb_typeof(related_patterns) = 'array');

-- Indexes for common filter query params (language, domain, entity_type)
create index idx_patterns_language    on patterns (language);
create index idx_patterns_domain      on patterns (domain);
create index idx_patterns_entity_type on patterns (entity_type);

-- =============================================================================
-- PATTERN_CHUNKS TABLE — one row per H2 section of a parent pattern
-- =============================================================================

create table if not exists pattern_chunks (
    id                uuid         primary key default gen_random_uuid(),
    pattern_id        uuid         not null references patterns(id) on delete cascade,
    section_title     varchar(255) not null,
    chunk_index       int          not null,
    content           text         not null,
    embedding         vector(1536),
    enrichment_status varchar(20)  not null default 'pending',
    enrichment_error  text,
    enriched_at       timestamptz,
    created_at        timestamptz  not null default now(),
    updated_at        timestamptz  not null default now(),
    constraint pattern_chunks_enrichment_status_valid
        check (enrichment_status in ('pending', 'enriched', 'failed'))
);

create index idx_pattern_chunks_pattern_id on pattern_chunks (pattern_id);

-- Vector similarity search on chunks (IVFFlat for MVP scale)
-- lists = 100 suitable for 1,000-10,000 chunks
-- NOTE: IVFFlat requires at least one row with a non-null embedding to build
-- centroids. If applied to an empty database, create this index after seeding
-- initial data, or use HNSW as an alternative that builds on empty tables.
create index idx_pattern_chunks_embedding
    on pattern_chunks using ivfflat (embedding vector_cosine_ops)
    with (lists = 100);

comment on table pattern_chunks is 'H2-bounded chunks of patterns with per-chunk embeddings for semantic search';
comment on column pattern_chunks.chunk_index is 'Zero-based order within parent pattern';
comment on index idx_pattern_chunks_embedding is
    'IVFFlat index for chunk vector similarity search (100 lists for MVP scale)';

-- =============================================================================
-- ENRICHMENT_JOBS — add chunk_id, make pattern_id nullable
-- =============================================================================

-- chunk_id is set for chunk-based enrichment jobs; pattern_id is set for legacy jobs
alter table enrichment_jobs
    add column chunk_id uuid references pattern_chunks(id) on delete cascade;
alter table enrichment_jobs
    alter column pattern_id drop not null;
