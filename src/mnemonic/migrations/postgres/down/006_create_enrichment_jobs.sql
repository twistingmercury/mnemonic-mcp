-- src/mnemonic/migrations/postgres/down/006_create_enrichment_jobs.sql
-- Reverses: Creates the enrichment_jobs table for background pattern processing.
--
-- This migration drops the enrichment_jobs table and its indexes.
-- No other tables depend on enrichment_jobs, so it can be safely dropped.

drop index if exists idx_enrichment_jobs_pattern;
drop table if exists enrichment_jobs;
