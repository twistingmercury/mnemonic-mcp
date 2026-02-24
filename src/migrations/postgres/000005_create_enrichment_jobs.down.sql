-- src/migrations/postgres/000005_create_enrichment_jobs.down.sql
-- Reverses: Creates the enrichment jobs queue table.
--
-- Copyright 2025, Mnemonic Authors

drop index if exists idx_enrichment_jobs_unique_pending;
drop index if exists idx_enrichment_jobs_pattern;
drop table if exists enrichment_jobs;
