-- src/migrations/postgres/000004_create_pattern_agent_associations.down.sql
-- Reverses: Creates the pattern-agent association table.
--
-- Copyright 2025, Mnemonic Authors

drop index if exists idx_pattern_agent_assoc_agent;
drop table if exists pattern_agent_associations;
