-- src/migrations/postgres/000002_create_agents.down.sql
-- Reverses: Creates the agents table with JSONB document model.
--
-- Copyright 2025, Mnemonic Authors

drop index if exists idx_agents_definition;
drop table if exists agents;
