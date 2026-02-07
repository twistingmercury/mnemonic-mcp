-- src/migrations/postgres/up/001_extensions_and_functions.sql
-- Enables required PostgreSQL extensions.
-- Part of Mnemonic MVP
--
-- This migration must be applied first as it provides:
-- - UUID generation extension
-- - Vector operations extension for embeddings

-- Enable UUID generation (gen_random_uuid is built-in to PG 13+, but uuid-ossp provides uuid_generate_v4)
create extension if not exists "uuid-ossp";

-- Enable vector operations for embeddings (pgvector extension)
create extension if not exists vector;
