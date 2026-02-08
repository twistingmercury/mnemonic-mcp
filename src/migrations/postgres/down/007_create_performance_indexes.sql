-- src/mnemonic/migrations/postgres/down/007_create_performance_indexes.sql
-- Reverses: Creates performance-optimized indexes for common query patterns.
--
-- This migration drops only the indexes created in 007_create_performance_indexes.up.sql.
-- Indexes created in earlier migrations (003) are not affected.

-- Drop enrichment jobs indexes
drop index if exists idx_enrichment_jobs_processing;
drop index if exists idx_enrichment_jobs_pending;

-- Drop patterns indexes (only those created in this migration)
drop index if exists idx_patterns_tags;
drop index if exists idx_patterns_enriched;

-- Drop routing rules indexes
drop index if exists idx_routing_rules_enabled_priority;
