-- src/migrations/postgres/000006_create_performance_indexes.down.sql
-- Reverses: Creates performance-optimized indexes.
--
-- Copyright 2025, Mnemonic Authors

drop index if exists idx_enrichment_jobs_processing;
drop index if exists idx_enrichment_jobs_pending;
drop index if exists idx_patterns_search;
drop index if exists idx_patterns_tags;
drop index if exists idx_patterns_embedding;
drop index if exists idx_patterns_enriched;
