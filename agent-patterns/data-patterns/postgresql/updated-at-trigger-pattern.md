---
entity_name: Updated-at Trigger Pattern
entity_type: database-pattern
language: agnostic
domain: backend
description: PostgreSQL trigger pattern for automatically updating the updated_at timestamp column on row modifications.
tags:
  - PostgreSQL
  - triggers
  - plpgsql
  - audit
  - timestamps
version: PostgreSQL 14+
related_patterns:
  - SQL Migration Pattern
  - Audit Columns Pattern
---

# Updated-at Trigger Pattern

This pattern provides a reusable trigger function that automatically updates the `updated_at` column whenever a row is modified.

## Overview

Instead of relying on application code to set `updated_at`, this trigger ensures the timestamp is always updated at the database level, providing consistency across all update paths (application, manual SQL, migrations).

## Implementation

### Step 1: Create the Function (Once Per Database)

```sql
-- migrations/001_create_utility_functions.up.sql
-- Creates reusable trigger function for automatic updated_at timestamps.
-- This function should be created once and reused by all tables.

create or replace function update_updated_at()
returns trigger as $$
begin
    new.updated_at = now();
    return new;
end;
$$ language plpgsql;

comment on function update_updated_at() is
    'Trigger function that sets updated_at to current timestamp on row update';
```

```sql
-- migrations/001_create_utility_functions.down.sql
drop function if exists update_updated_at();
```

### Step 2: Apply to Each Table

```sql
-- migrations/003_create_users_table.up.sql
create table if not exists users (
    id uuid primary key default gen_random_uuid(),
    email text not null unique,
    name text not null,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

-- Apply the trigger
create trigger trg_users_updated_at
    before update on users
    for each row execute function update_updated_at();
```

```sql
-- migrations/003_create_users_table.down.sql
drop trigger if exists trg_users_updated_at on users;
drop table if exists users;
```

## Naming Convention

| Component | Convention | Example |
|-----------|------------|---------|
| Function | `update_updated_at` | Single shared function |
| Trigger | `trg_<table>_updated_at` | `trg_users_updated_at` |
| Column | `updated_at` | Consistent across all tables |

## Usage Examples

### Multiple Tables

```sql
-- All tables use the same function
create trigger trg_users_updated_at
    before update on users
    for each row execute function update_updated_at();

create trigger trg_posts_updated_at
    before update on posts
    for each row execute function update_updated_at();

create trigger trg_comments_updated_at
    before update on comments
    for each row execute function update_updated_at();
```

### Adding to Existing Table

```sql
-- migrations/010_add_updated_at_trigger_to_legacy_table.up.sql
-- Adds automatic updated_at to legacy_table that was missing it.

-- First ensure column exists with default
alter table legacy_table
add column if not exists updated_at timestamptz not null default now();

-- Backfill existing rows (optional - set to created_at if available)
update legacy_table set updated_at = coalesce(created_at, now())
where updated_at = now();

-- Add trigger
create trigger trg_legacy_table_updated_at
    before update on legacy_table
    for each row execute function update_updated_at();
```

## Behavior

### What Gets Updated

The trigger fires `BEFORE UPDATE` for each row:

```sql
-- This update will automatically set updated_at = now()
update users set name = 'New Name' where id = '123';

-- Even bulk updates get individual timestamps
update users set status = 'inactive' where last_login < now() - interval '1 year';
```

### What Does NOT Trigger

- `INSERT` - Column default handles initial value
- `DELETE` - Row is removed, no update needed
- Updates that don't change any values - Trigger still fires

## Advanced Variations

### Conditional Update (Only When Data Changes)

```sql
create or replace function update_updated_at_if_changed()
returns trigger as $$
begin
    -- Only update if something actually changed
    if row(new.*) is distinct from row(old.*) then
        new.updated_at = now();
    end if;
    return new;
end;
$$ language plpgsql;
```

### With User Tracking

```sql
create or replace function update_audit_columns()
returns trigger as $$
begin
    new.updated_at = now();
    -- Assumes current_user_id() is a custom function or session variable
    new.updated_by = current_setting('app.current_user_id', true)::uuid;
    return new;
end;
$$ language plpgsql;
```

### Timezone-Aware

```sql
create or replace function update_updated_at()
returns trigger as $$
begin
    -- Explicitly use UTC
    new.updated_at = now() at time zone 'utc';
    return new;
end;
$$ language plpgsql;
```

## Testing

```sql
-- Insert a row
insert into users (email, name) values ('test@example.com', 'Test User');

-- Check timestamps are equal
select email, created_at, updated_at,
       created_at = updated_at as timestamps_equal
from users where email = 'test@example.com';

-- Wait a moment, then update
select pg_sleep(1);
update users set name = 'Updated Name' where email = 'test@example.com';

-- Check updated_at changed
select email, created_at, updated_at,
       created_at < updated_at as updated_at_changed
from users where email = 'test@example.com';
```

## Best Practices

1. **Create function once** - Don't duplicate the function per table
2. **Use BEFORE UPDATE** - Modify `new` record before it's written
3. **Use FOR EACH ROW** - Trigger fires for each affected row
4. **Name consistently** - `trg_<table>_updated_at` pattern
5. **Include in down migration** - Drop trigger before dropping table
6. **Use timestamptz** - Always timezone-aware timestamps

## Common Issues

### Trigger Not Firing

```sql
-- Check if trigger exists
select trigger_name, event_manipulation, action_timing
from information_schema.triggers
where event_object_table = 'users';

-- Ensure function exists
select proname from pg_proc where proname = 'update_updated_at';
```

### Circular Updates

If your trigger causes additional updates, you may get infinite loops:

```sql
-- BAD: This trigger updates another table which triggers more updates
create or replace function bad_trigger()
returns trigger as $$
begin
    update other_table set related_updated = now() where id = new.other_id;
    new.updated_at = now();
    return new;
end;
$$ language plpgsql;
```

Solution: Use `pg_trigger_depth()` or separate the concerns.
