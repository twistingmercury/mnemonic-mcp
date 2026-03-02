-- src/migrations/postgres/000009_pattern_schema_chunks.down.sql
-- Reverses: pattern_chunks table, metadata columns, and enrichment_jobs changes.
--
-- Copyright 2025, Mnemonic Authors

-- =============================================================================
-- ENRICHMENT_JOBS — remove chunk_id, restore pattern_id NOT NULL
-- =============================================================================

-- Chunk-based jobs have pattern_id = NULL; purge them before restoring NOT NULL.
ALTER TABLE enrichment_jobs DROP CONSTRAINT IF EXISTS enrichment_jobs_target_exclusive;
DROP INDEX IF EXISTS idx_enrichment_jobs_unique_pending_chunk;

delete from enrichment_jobs where pattern_id is null;

alter table enrichment_jobs
    drop column if exists chunk_id,
    alter column pattern_id set not null;

-- =============================================================================
-- PATTERN_CHUNKS — drop table (cascades its indexes and constraints)
-- =============================================================================

drop table if exists pattern_chunks;

-- =============================================================================
-- PATTERNS TABLE — drop new indexes, columns; restore embedding and constraint
-- =============================================================================

drop index if exists idx_patterns_entity_type;
drop index if exists idx_patterns_domain;
drop index if exists idx_patterns_language;

alter table patterns
    drop constraint if exists patterns_related_patterns_array,
    drop column if exists entity_type,
    drop column if exists language,
    drop column if exists domain,
    drop column if exists version,
    drop column if exists related_patterns;

alter table patterns
    add column embedding vector(1536);

alter table patterns
    add constraint patterns_content_length
        check (length(content) <= 10240);
