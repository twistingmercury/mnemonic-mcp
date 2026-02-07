-- src/mnemonic/migrations/postgres/up/007_create_performance_indexes.sql
-- Creates performance-optimized indexes for common query patterns.
-- Part of Mnemonic MVP
--
-- Dependencies:
--   - 003_create_patterns (for patterns table)
--   - 005_create_routing_rules (for routing_rules table)
--   - 006_create_enrichment_jobs (for enrichment_jobs table)
--
-- These indexes optimize the most common query patterns:
-- 1. Routing engine: find enabled rules in priority order
-- 2. Pattern search: find enriched patterns for similarity search
-- 3. Enrichment worker: claim pending jobs for processing
-- 4. Pattern listing: filter by tags, full-text search
--
-- Note: Some indexes use partial index predicates (WHERE clause) to
-- minimize index size and improve performance for specific queries.

-- =============================================================================
-- ROUTING RULES INDEXES
-- =============================================================================

-- Optimizes the primary routing query: get enabled rules by priority
-- This is the most frequent query in the routing engine
-- Partial index excludes disabled rules from the index
create index if not exists idx_routing_rules_enabled_priority
    on routing_rules(priority desc, id)
    where enabled = true;

comment on index idx_routing_rules_enabled_priority is
    'Optimizes routing rule lookup by priority order (enabled rules only)';

-- =============================================================================
-- PATTERNS INDEXES
-- =============================================================================

-- Partial index for filtering to only enriched patterns
-- Used when selecting patterns eligible for similarity search
create index if not exists idx_patterns_enriched
    on patterns(id)
    where enrichment_status = 'enriched';

comment on index idx_patterns_enriched is
    'Partial index for filtering enriched patterns in similarity queries';

-- Note: idx_patterns_embedding_cosine already exists from migration 003
-- It provides IVFFlat vector similarity search with lists=100
-- This is suitable for 1,000-10,000 patterns (MVP scale)
-- If you need to recreate it or adjust parameters:
--
-- create index idx_patterns_embedding
--     on patterns using ivfflat (embedding vector_cosine_ops)
--     with (lists = 100);

-- GIN index for tag filtering using JSONB containment operator (@>)
-- Optimizes queries like: WHERE tags @> '["golang"]'::jsonb
create index if not exists idx_patterns_tags
    on patterns using gin (tags);

comment on index idx_patterns_tags is
    'GIN index for efficient tag filtering using JSONB containment';

-- Note: idx_patterns_search already exists from migration 003
-- It provides full-text search on name and description
-- If you need to recreate it:
--
-- create index idx_patterns_search
--     on patterns using gin (
--         to_tsvector('english', name || ' ' || coalesce(description, ''))
--     );

-- =============================================================================
-- ENRICHMENT JOBS INDEXES
-- =============================================================================

-- Partial index for pending jobs ordered by scheduled time
-- Used by enrichment workers to claim the next job to process
-- Only includes pending jobs to minimize index size
create index if not exists idx_enrichment_jobs_pending
    on enrichment_jobs(scheduled_for)
    where status = 'pending';

comment on index idx_enrichment_jobs_pending is
    'Optimizes worker polling for pending jobs by scheduled time';

-- Partial index for processing jobs ordered by start time
-- Used to detect stale/stuck jobs that need to be reclaimed
-- Jobs processing for too long may indicate worker failure
create index if not exists idx_enrichment_jobs_processing
    on enrichment_jobs(started_at)
    where status = 'processing';

comment on index idx_enrichment_jobs_processing is
    'Optimizes detection of stale processing jobs for timeout handling';
