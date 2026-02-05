---
entity_name: SQL Migration Pattern
entity_type: database-pattern
language: agnostic
domain: backend
description: PostgreSQL migration pattern with up/down files, naming conventions, idempotency rules, and best practices for versioned schema changes.
tags:
  - PostgreSQL
  - migrations
  - schema
  - DDL
  - golang-migrate
version: PostgreSQL 14+
related_patterns:
  - Updated-at Trigger Pattern
  - Audit Columns Pattern
  - Soft Delete Pattern
---

# SQL Migration Pattern

This pattern defines conventions for creating versioned, reversible database migrations using golang-migrate or similar tools.

## File Structure

```
migrations/
├── 001_create_extensions.up.sql
├── 001_create_extensions.down.sql
├── 002_create_utility_functions.up.sql
├── 002_create_utility_functions.down.sql
├── 003_create_users_table.up.sql
├── 003_create_users_table.down.sql
├── 004_create_posts_table.up.sql
├── 004_create_posts_table.down.sql
└── ...
```

## Naming Convention

**Format**: `NNN_description.up.sql` / `NNN_description.down.sql`

| Component | Rules |
|-----------|-------|
| `NNN` | Three-digit sequence number (001, 002, etc.) |
| `description` | Snake_case, describes what the migration does |
| `.up.sql` | Forward migration (apply changes) |
| `.down.sql` | Reverse migration (rollback changes) |

**Good names:**
- `001_create_extensions.up.sql`
- `002_create_users_table.up.sql`
- `003_add_email_to_users.up.sql`
- `004_create_posts_with_fk_to_users.up.sql`

**Bad names:**
- `1_users.up.sql` (no padding, not descriptive)
- `create_table.up.sql` (no sequence number)
- `001_UpdateUsersTable.up.sql` (not snake_case)

## Migration Rules

### 1. Idempotent When Possible

Use `IF NOT EXISTS` and `IF EXISTS` to make migrations safe to re-run:

```sql
-- Good: Idempotent
create table if not exists users (
    id uuid primary key default gen_random_uuid(),
    email text not null unique
);

create index if not exists idx_users_email on users (email);

-- Bad: Will fail if run twice
create table users (
    id uuid primary key,
    email text not null
);
```

### 2. Always Provide Down Migrations

Every `up.sql` must have a corresponding `down.sql` that reverses it:

```sql
-- 003_create_users_table.up.sql
create table if not exists users (
    id uuid primary key default gen_random_uuid(),
    email text not null unique,
    created_at timestamptz not null default now()
);

create index if not exists idx_users_email on users (email);
```

```sql
-- 003_create_users_table.down.sql
drop index if exists idx_users_email;
drop table if exists users;
```

### 3. Use Transactions

Wrap DDL in transactions when supported (PostgreSQL supports transactional DDL):

```sql
-- 005_add_status_column.up.sql
begin;

alter table users add column status text not null default 'active';
create index idx_users_status on users (status);

commit;
```

### 4. Include Comments

Document the purpose of each migration:

```sql
-- 003_create_users_table.up.sql
-- Creates the users table for storing user account information.
-- Part of the authentication module.
--
-- Dependencies: 002_create_utility_functions (for update_updated_at function)

create table if not exists users (
    -- ... columns
);
```

### 5. Order by Dependencies

Create parent tables before children (FK dependencies):

```
001_create_extensions.up.sql      -- Extensions first
002_create_utility_functions.up.sql -- Shared functions
003_create_users_table.up.sql     -- Independent tables
004_create_posts_table.up.sql     -- Tables with FKs to users
005_create_comments_table.up.sql  -- Tables with FKs to posts
```

### 6. Test Rollbacks

Verify that down migrations actually reverse up migrations:

```bash
# Apply migration
migrate -path ./migrations -database "$DATABASE_URL" up 1

# Verify state
psql $DATABASE_URL -c "\dt"

# Rollback
migrate -path ./migrations -database "$DATABASE_URL" down 1

# Verify rollback worked
psql $DATABASE_URL -c "\dt"
```

## Migration Templates

### Create Table

```sql
-- migrations/NNN_create_tablename.up.sql
-- Creates the tablename table for [purpose].

create table if not exists tablename (
    id uuid primary key default gen_random_uuid(),

    -- Domain columns
    name text not null,
    description text,

    -- Audit columns
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

-- Trigger for updated_at (requires utility function)
create trigger trg_tablename_updated_at
    before update on tablename
    for each row execute function update_updated_at();

-- Indexes for common queries
create index if not exists idx_tablename_name on tablename (name);

-- Table comment
comment on table tablename is 'Description of what this table stores';
```

```sql
-- migrations/NNN_create_tablename.down.sql
-- Reverses: Creates the tablename table

drop trigger if exists trg_tablename_updated_at on tablename;
drop table if exists tablename;
```

### Add Column

```sql
-- migrations/NNN_add_column_to_table.up.sql
-- Adds column_name to table_name for [purpose].

alter table table_name
add column if not exists column_name text;

-- If adding NOT NULL column, provide default first, then optionally remove
alter table table_name
add column if not exists status text not null default 'active';
```

```sql
-- migrations/NNN_add_column_to_table.down.sql
alter table table_name
drop column if exists column_name;
```

### Add Index

```sql
-- migrations/NNN_add_index_to_table.up.sql
-- Adds index on (column1, column2) for [query pattern].

create index concurrently if not exists idx_table_column1_column2
on table_name (column1, column2);
```

```sql
-- migrations/NNN_add_index_to_table.down.sql
drop index concurrently if exists idx_table_column1_column2;
```

### Add Foreign Key

```sql
-- migrations/NNN_add_fk_to_table.up.sql
-- Adds foreign key from child_table to parent_table.

alter table child_table
add constraint fk_child_parent
foreign key (parent_id) references parent_table(id)
on delete cascade;

-- Index on FK column for join performance
create index if not exists idx_child_parent_id on child_table (parent_id);
```

```sql
-- migrations/NNN_add_fk_to_table.down.sql
drop index if exists idx_child_parent_id;
alter table child_table drop constraint if exists fk_child_parent;
```

## Forward Compatibility

When migrations and application code are deployed independently:

### Adding Columns

```sql
-- Step 1: Add nullable or with default (migration deploys first)
alter table users add column phone text;  -- nullable
-- OR
alter table users add column status text not null default 'active';

-- Step 2: App deploys, starts using column

-- Step 3 (optional): Tighten constraint later
alter table users alter column phone set not null;
```

### Removing Columns

```sql
-- Step 1: App stops using column (deploy app first)
-- Step 2: Remove column (migration deploys second)
alter table users drop column old_column;
```

### Renaming Columns

```sql
-- Step 1: Add new column
alter table users add column new_name text;

-- Step 2: Copy data
update users set new_name = old_name;

-- Step 3: App deploys, uses new_name

-- Step 4: Drop old column
alter table users drop column old_name;
```

## Tools

### golang-migrate

```bash
# Install
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Create new migration
migrate create -ext sql -dir migrations -seq create_users_table

# Apply all migrations
migrate -path ./migrations -database "$DATABASE_URL" up

# Apply N migrations
migrate -path ./migrations -database "$DATABASE_URL" up 3

# Rollback last migration
migrate -path ./migrations -database "$DATABASE_URL" down 1

# Rollback all
migrate -path ./migrations -database "$DATABASE_URL" down

# Check current version
migrate -path ./migrations -database "$DATABASE_URL" version

# Force version (use with caution)
migrate -path ./migrations -database "$DATABASE_URL" force 5
```

### Connection String Format

```
postgres://username:password@host:5432/database?sslmode=disable
```

## Best Practices

1. **Never edit applied migrations** - Create new migrations to fix issues
2. **Test migrations in staging first** - Always test before production
3. **Backup before migrating** - Especially for destructive changes
4. **Use transactions** - PostgreSQL supports transactional DDL
5. **Keep migrations small** - One logical change per migration
6. **Document complex migrations** - Explain the "why" in comments
7. **Consider deployment order** - Migrations may deploy separately from app

## Anti-Patterns

- **Editing applied migrations** - Creates inconsistent environments
- **Skipping down migrations** - Makes rollback impossible
- **Large monolithic migrations** - Hard to debug and rollback
- **Data migrations in schema migrations** - Separate concerns
- **Missing IF EXISTS/IF NOT EXISTS** - Breaks re-runnability
