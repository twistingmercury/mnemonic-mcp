-- src/migrations/postgres/down/003_create_patterns.sql
-- Reverses: Creates the patterns table with PGVector embedding support.
--
-- Note: This will fail if other tables have foreign keys referencing patterns.
-- The pattern_agent_associations table (migration 004) must be dropped first.

drop table if exists patterns;
