-- src/migrations/postgres/000003_create_patterns.down.sql
-- Reverses: Creates the patterns table with PGVector embedding support.
--
-- Copyright 2025, Mnemonic Authors

drop table if exists patterns;
