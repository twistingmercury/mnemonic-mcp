-- src/migrations/postgres/000007_create_skills.down.sql
-- Reverses: Creates the skills table.
--
-- Copyright 2025, Mnemonic Authors

drop index if exists idx_skills_definition;
drop table if exists skills;
