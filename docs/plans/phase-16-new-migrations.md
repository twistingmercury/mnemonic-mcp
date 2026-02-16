# Phase 16: New Migrations

> Part of the [MVP Implementation Plan](mvp-implementation-plan.md)

**Goal:** Create migrations to migrate agents to JSONB document model, and create `skills`, `commands`, and `skill_files` tables using JSONB document model.

**Agent(s):** data-engineer

**Dependencies:** Phase 15 (migration cleanup)

---

## Step 1: Create migration 008 (deprecated)

- Migration 008 was removed during the pivot to JSONB document model
- Agent: `data-engineer`

## Step 2: Create migration 009 (up) -- migrate agents to JSONB document model

- Create file: `/Users/doublej/dev/mnemonic/src/migrations/postgres/up/009_migrate_agents_to_jsonb.sql`
- Content: Migrate agents table to JSONB document model
  - Add `definition JSONB` column
  - Add `crc64 BIGINT` column
  - Migrate existing data from columns to JSONB structure
  - Drop old columns: `system_prompt`, `model_name`, `model_temperature`, `routing_keywords`, `allowed_tools`
  - Drop trigger `agents_updated_at` (application handles `updated_at` now)
  - Update table and column comments
- Agent: `data-engineer`
- Design reference: [Go Architecture Plan - Section 8](2026-02-15-go-architecture-plan.md#8-schema-changes-for-existing-tables)

## Step 3: Create migration 009 (down)

- Create file: `/Users/doublej/dev/mnemonic/src/migrations/postgres/down/009_migrate_agents_to_jsonb.sql`
- Content: Reverse JSONB migration
  - Restore individual columns from JSONB
  - Drop `definition` and `crc64` columns
  - Recreate `agents_updated_at` trigger
- Agent: `data-engineer`

## Step 4: Create migration 010 (up) -- create skills

- Create file: `/Users/doublej/dev/mnemonic/src/migrations/postgres/up/010_create_skills.sql`
- Content: Full `CREATE TABLE skills` with PK `id UUID DEFAULT gen_random_uuid()`, columns: `name VARCHAR(64) UNIQUE NOT NULL`, `definition JSONB NOT NULL`, `crc64 BIGINT NOT NULL`, `created_at`, `updated_at`. Constraints: `skills_name_format`. No trigger (application handles `updated_at`). Comments on table and columns.
- Agent: `data-engineer`
- Design reference: [API Specification - Skills](../design/2026-02-15-pivot-api-specification.md#26-skills), [Go Architecture Plan - Section 9.1](2026-02-15-go-architecture-plan.md#91-skill-repository)

## Step 5: Create migration 010 (down)

- Create file: `/Users/doublej/dev/mnemonic/src/migrations/postgres/down/010_create_skills.sql`
- Content: `DROP TABLE IF EXISTS skills;`
- Agent: `data-engineer`

## Step 6: Create migration 011 (up) -- create commands

- Create file: `/Users/doublej/dev/mnemonic/src/migrations/postgres/up/011_create_commands.sql`
- Content: Full `CREATE TABLE commands` with PK `id UUID DEFAULT gen_random_uuid()`, columns: `name VARCHAR(255) UNIQUE NOT NULL`, `definition JSONB NOT NULL`, `crc64 BIGINT NOT NULL`, `created_at`, `updated_at`. Constraints: `commands_name_format`. No trigger (application handles `updated_at`). Comments.
- Agent: `data-engineer`
- Design reference: [API Specification - Commands](../design/2026-02-15-pivot-api-specification.md#27-commands), [Go Architecture Plan - Section 9.2](2026-02-15-go-architecture-plan.md#92-command-repository)

## Step 7: Create migration 011 (down)

- Create file: `/Users/doublej/dev/mnemonic/src/migrations/postgres/down/011_create_commands.sql`
- Content: `DROP TABLE IF EXISTS commands;`
- Agent: `data-engineer`

## Step 8: Create migration 012 (up) -- create skill_files

- Create file: `/Users/doublej/dev/mnemonic/src/migrations/postgres/up/012_create_skill_files.sql`
- Content: Full `CREATE TABLE skill_files` with PK `id UUID DEFAULT gen_random_uuid()`, columns: `skill_id UUID NOT NULL REFERENCES skills(id) ON DELETE CASCADE`, `file_type VARCHAR(50) NOT NULL`, `filename VARCHAR(255) NOT NULL`, `document JSONB NOT NULL`, `crc64 BIGINT NOT NULL`, `created_at`, `updated_at`. Constraints: `skill_files_unique_filename` (unique on skill_id + filename). No trigger (application handles `updated_at`). Comments.
- Agent: `data-engineer`

## Step 9: Create migration 012 (down)

- Create file: `/Users/doublej/dev/mnemonic/src/migrations/postgres/down/012_create_skill_files.sql`
- Content: `DROP TABLE IF EXISTS skill_files;`
- Agent: `data-engineer`

## Step 10: Verify all migration files exist

- Run: `ls -1 /Users/doublej/dev/mnemonic/src/migrations/postgres/up/` -- should show 001-004, 006-007, 009-012 (no 005, no 008)
- Run: `ls -1 /Users/doublej/dev/mnemonic/src/migrations/postgres/down/` -- should show 001-004, 006-007, 009-012 (no 005, no 008)
- Agent: `data-engineer`

## Step 11: Run migration tests

- Start local Postgres via docker-compose
- Apply all up migrations in order (001-004, 006-007, 009-012) against a fresh database
- Verify all tables exist: `agents`, `patterns`, `pattern_agent_associations`, `enrichment_jobs`, `skills`, `commands`, `skill_files`
- Verify `agents.definition` and `agents.crc64` columns exist
- Verify old `agents` columns are gone: `system_prompt`, `model_name`, `model_temperature`, `routing_keywords`, `allowed_tools`
- Verify `skills` has `definition JSONB`, `crc64 BIGINT`, `name VARCHAR(64)`
- Verify `commands` has `definition JSONB`, `crc64 BIGINT`, `name VARCHAR(255)`
- Verify `skill_files` has `document JSONB`, `crc64 BIGINT`
- Verify `routing_rules` table does NOT exist
- Verify no `update_updated_at()` triggers on `agents`, `skills`, `commands`, `skill_files` tables
- Run all down migrations in reverse order (012-009, 007-006, 004-001)
- Verify clean teardown
- Agent: `data-engineer`

## Step 12: Commit

```bash
git add src/migrations/postgres/
git commit -m "feat(pivot): add migrations 009-012 (agents JSONB, skills, commands, skill_files with JSONB document model)"
```
