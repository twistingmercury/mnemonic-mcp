-- src/migrations/postgres/down/002_create_agents.sql
-- Reverses: Creates the agents table for storing agent definitions.
--
-- Note: This will fail if other tables have foreign keys referencing agents.
-- Those tables (e.g., routing_rules, pattern_agent_associations) must be
-- dropped first.

drop table if exists agents;
