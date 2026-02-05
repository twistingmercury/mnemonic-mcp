---
entity_name: Soft Delete Pattern
entity_type: database-pattern
language: agnostic
domain: backend
description: PostgreSQL pattern for soft deletes using deleted_at timestamp, partial indexes for active records, and views for common access patterns.
tags:
  - PostgreSQL
  - soft-delete
  - audit
  - data-retention
version: PostgreSQL 14+
related_patterns:
  - Audit Columns Pattern
  - SQL Migration Pattern
---

# Soft Delete Pattern

This pattern implements soft deletes where records are marked as deleted rather than physically removed, enabling data recovery and audit trails.

## Overview

Instead of `DELETE FROM users WHERE id = '123'`, soft delete sets a `deleted_at` timestamp:

```sql
UPDATE users SET deleted_at = now() WHERE id = '123';
```

The record remains in the database but is excluded from normal queries.

## Implementation

### Step 1: Add deleted_at Column

```sql
-- migrations/NNN_add_soft_delete_to_users.up.sql
-- Adds soft delete capability to users table.

alter table users
add column deleted_at timestamptz;

-- Index for filtering active records efficiently
create index idx_users_active on users (id) where deleted_at is null;

-- Index for finding deleted records (admin/audit queries)
create index idx_users_deleted on users (deleted_at) where deleted_at is not null;

comment on column users.deleted_at is
    'Timestamp when record was soft-deleted. NULL means active.';
```

```sql
-- migrations/NNN_add_soft_delete_to_users.down.sql
drop index if exists idx_users_deleted;
drop index if exists idx_users_active;
alter table users drop column if exists deleted_at;
```

### Step 2: Create View for Active Records

```sql
-- migrations/NNN_create_active_users_view.up.sql
-- Creates view for easily querying active (non-deleted) users.

create or replace view active_users as
select * from users where deleted_at is null;

comment on view active_users is
    'View showing only active (non-deleted) users';
```

```sql
-- migrations/NNN_create_active_users_view.down.sql
drop view if exists active_users;
```

## Table Structure

```sql
create table users (
    id uuid primary key default gen_random_uuid(),
    email text not null unique,
    name text not null,

    -- Audit columns
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    deleted_at timestamptz,  -- NULL = active, timestamp = deleted

    -- Optional: track who deleted
    deleted_by uuid references users(id)
);
```

## Query Patterns

### Select Active Records

```sql
-- Using WHERE clause
select * from users where deleted_at is null;

-- Using view (cleaner)
select * from active_users;
```

### Select Deleted Records

```sql
select * from users where deleted_at is not null;
```

### Select All Records (Including Deleted)

```sql
select * from users;  -- No filter
```

### Soft Delete a Record

```sql
update users
set deleted_at = now(),
    deleted_by = $current_user_id  -- optional
where id = $user_id
  and deleted_at is null;  -- prevent re-deleting
```

### Restore a Deleted Record

```sql
update users
set deleted_at = null,
    deleted_by = null
where id = $user_id;
```

### Permanently Delete (Purge)

```sql
-- Only for records deleted more than 90 days ago
delete from users
where deleted_at is not null
  and deleted_at < now() - interval '90 days';
```

## Index Strategy

### Partial Index for Active Records

```sql
-- Efficient for queries filtering on deleted_at is null
create index idx_users_active on users (id) where deleted_at is null;

-- This index is used by:
select * from users where deleted_at is null and id = '123';
```

### Partial Index for Deleted Records

```sql
-- For admin queries on deleted records
create index idx_users_deleted on users (deleted_at) where deleted_at is not null;

-- This index is used by:
select * from users where deleted_at is not null order by deleted_at desc;
```

### Compound Partial Index

```sql
-- For queries filtering on both email and active status
create index idx_users_email_active on users (email) where deleted_at is null;

-- This index is used by:
select * from users where email = 'test@example.com' and deleted_at is null;
```

## Unique Constraints with Soft Delete

### Problem

Standard unique constraint fails when you want to allow re-creating deleted records:

```sql
-- User deletes account with email 'test@example.com'
-- Later, same email tries to sign up - unique constraint fails!
```

### Solution: Partial Unique Index

```sql
-- Only enforce uniqueness on active records
create unique index idx_users_email_unique_active
on users (email) where deleted_at is null;

-- This allows:
-- 1. Active user with email 'test@example.com'
-- 2. Deleted user with email 'test@example.com' (different row)
-- But NOT:
-- 3. Two active users with same email
```

## Foreign Key Considerations

### Option 1: Allow References to Deleted Records

```sql
-- Posts can reference deleted users
create table posts (
    id uuid primary key,
    user_id uuid references users(id),  -- No cascade
    content text
);

-- Query shows author even if deleted
select p.*, u.name as author_name
from posts p
join users u on u.id = p.user_id;  -- Works even if user deleted
```

### Option 2: Cascade Soft Delete

```sql
-- When user is soft-deleted, also soft-delete their posts
create or replace function cascade_soft_delete_user()
returns trigger as $$
begin
    if new.deleted_at is not null and old.deleted_at is null then
        update posts set deleted_at = new.deleted_at where user_id = new.id;
    end if;
    return new;
end;
$$ language plpgsql;

create trigger trg_users_cascade_soft_delete
    after update on users
    for each row
    when (new.deleted_at is not null and old.deleted_at is null)
    execute function cascade_soft_delete_user();
```

## Application Integration

### Repository Pattern (Go)

```go
type UserRepository interface {
    // Default: only active records
    FindByID(ctx context.Context, id uuid.UUID) (*User, error)
    FindByEmail(ctx context.Context, email string) (*User, error)

    // Explicit: include deleted
    FindByIDIncludingDeleted(ctx context.Context, id uuid.UUID) (*User, error)

    // Soft delete
    Delete(ctx context.Context, id uuid.UUID) error

    // Restore
    Restore(ctx context.Context, id uuid.UUID) error

    // Hard delete (admin only)
    PermanentlyDelete(ctx context.Context, id uuid.UUID) error
}
```

### SQL in Go

```go
// Find active user
const findActiveUser = `
    SELECT * FROM users
    WHERE id = $1 AND deleted_at IS NULL
`

// Soft delete
const softDelete = `
    UPDATE users
    SET deleted_at = now(), updated_at = now()
    WHERE id = $1 AND deleted_at IS NULL
`

// Restore
const restore = `
    UPDATE users
    SET deleted_at = NULL, updated_at = now()
    WHERE id = $1
`
```

## Best Practices

1. **Use partial indexes** - Essential for performance on active record queries
2. **Create views** - Simplify application code with `active_*` views
3. **Partial unique constraints** - Allow re-use of unique values after deletion
4. **Consider cascade behavior** - Document how related records handle parent deletion
5. **Retention policy** - Define when soft-deleted records are purged
6. **Track deletion metadata** - Consider `deleted_by` for audit trails
7. **API design** - Decide if deleted records are returned with a flag or hidden entirely

## Anti-Patterns

- **Forgetting the filter** - Always check for `deleted_at is null` unless explicitly including deleted
- **No partial indexes** - Performance degrades as deleted records accumulate
- **Hard unique constraints** - Prevents re-registration after account deletion
- **No purge policy** - Database grows unbounded with deleted records
