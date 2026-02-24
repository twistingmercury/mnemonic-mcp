-- src/migrations/postgres/000008_create_skill_files.down.sql
-- Reverses: Creates the skill_files table.
--
-- Copyright 2025, Mnemonic Authors

drop index if exists idx_skill_files_skill_id;
drop table if exists skill_files;
