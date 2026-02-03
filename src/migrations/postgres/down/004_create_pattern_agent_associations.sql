-- src/migrations/postgres/down/004_create_pattern_agent_associations.sql
-- Reverses: Creates the pattern-agent association table.
--
-- This migration drops the association table and its indexes.
-- No other tables depend on this table, so it can be safely dropped.

drop index if exists idx_pattern_agent_assoc_agent;
drop index if exists idx_pattern_agent_assoc_pattern;
drop table if exists pattern_agent_associations;
