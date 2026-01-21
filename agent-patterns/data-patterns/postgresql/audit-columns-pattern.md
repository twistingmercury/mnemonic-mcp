---
entity_name: Audit Columns Pattern
entity_type: database-pattern
language: agnostic
domain: backend
description: PostgreSQL pattern for standard audit columns (created_at, updated_at) with optional user tracking (created_by, updated_by).
tags:
  - PostgreSQL
  - audit
  - timestamps
  - tracking
version: PostgreSQL 14+
related_patterns:
  - Updated-at Trigger Pattern
  - Soft Delete Pattern
  - SQL Migration Pattern
---

# Audit Columns Pattern

This pattern defines standard columns for tracking when and optionally by whom records were created and modified.

## Basic Audit Columns

Every table should include at minimum:

```sql
create table example (
    id uuid primary key default gen_random_uuid(),

    -- Domain columns
    name text not null,

    -- Audit columns (required)
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

-- Trigger for automatic updated_at
create trigger trg_example_updated_at
    before update on example
    for each row execute function update_updated_at();
```

## Column Definitions

| Column | Type | Constraints | Purpose |
|--------|------|-------------|---------|
| `created_at` | `timestamptz` | `NOT NULL DEFAULT now()` | When record was inserted |
| `updated_at` | `timestamptz` | `NOT NULL DEFAULT now()` | When record was last modified |
| `created_by` | `uuid` | `REFERENCES users(id)` | Who created (optional) |
| `updated_by` | `uuid` | `REFERENCES users(id)` | Who last modified (optional) |

## Implementation Levels

### Level 1: Timestamps Only (Minimum)

```sql
create table products (
    id uuid primary key default gen_random_uuid(),
    name text not null,
    price numeric(10,2) not null,

    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);
```

### Level 2: Timestamps + User Tracking

```sql
create table orders (
    id uuid primary key default gen_random_uuid(),
    product_id uuid not null references products(id),
    quantity int not null,

    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    created_by uuid references users(id),
    updated_by uuid references users(id)
);
```

### Level 3: Full Audit Trail (Separate Table)

For compliance requirements, use a separate audit table:

```sql
create table audit_log (
    id uuid primary key default gen_random_uuid(),
    table_name text not null,
    record_id uuid not null,
    action text not null check (action in ('INSERT', 'UPDATE', 'DELETE')),
    old_data jsonb,
    new_data jsonb,
    changed_by uuid references users(id),
    changed_at timestamptz not null default now()
);

create index idx_audit_log_table_record on audit_log (table_name, record_id);
create index idx_audit_log_changed_at on audit_log (changed_at);
```

## Automatic User Tracking

### Using Session Variables

```sql
-- Set user in application before queries
set local app.current_user_id = 'user-uuid-here';

-- Trigger function that reads session variable
create or replace function update_audit_columns()
returns trigger as $$
declare
    current_user_id uuid;
begin
    -- Get user from session variable (NULL if not set)
    current_user_id := nullif(current_setting('app.current_user_id', true), '')::uuid;

    new.updated_at = now();
    new.updated_by = current_user_id;

    if tg_op = 'INSERT' then
        new.created_by = coalesce(new.created_by, current_user_id);
    end if;

    return new;
end;
$$ language plpgsql;
```

### Go Application Integration

```go
func (r *Repository) withUserContext(ctx context.Context, tx pgx.Tx) error {
    userID := auth.GetUserID(ctx)
    if userID != uuid.Nil {
        _, err := tx.Exec(ctx, "SET LOCAL app.current_user_id = $1", userID.String())
        return err
    }
    return nil
}

func (r *Repository) CreateOrder(ctx context.Context, order *Order) error {
    return r.pool.BeginTxFunc(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
        if err := r.withUserContext(ctx, tx); err != nil {
            return err
        }

        _, err := tx.Exec(ctx, `
            INSERT INTO orders (product_id, quantity)
            VALUES ($1, $2)
        `, order.ProductID, order.Quantity)
        return err
    })
}
```

## Migration Templates

### Adding Audit Columns to New Table

```sql
-- migrations/NNN_create_products_table.up.sql
create table if not exists products (
    id uuid primary key default gen_random_uuid(),

    -- Domain columns
    name text not null,
    description text,
    price numeric(10,2) not null,
    is_active boolean not null default true,

    -- Audit columns
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create trigger trg_products_updated_at
    before update on products
    for each row execute function update_updated_at();
```

### Adding Audit Columns to Existing Table

```sql
-- migrations/NNN_add_audit_columns_to_legacy.up.sql
-- Adds standard audit columns to legacy_table.

-- Add columns
alter table legacy_table
add column created_at timestamptz,
add column updated_at timestamptz;

-- Backfill with reasonable defaults
update legacy_table set
    created_at = coalesce(some_existing_date_column, now()),
    updated_at = coalesce(some_existing_date_column, now());

-- Add NOT NULL constraints
alter table legacy_table
alter column created_at set not null,
alter column updated_at set not null;

-- Add defaults for future inserts
alter table legacy_table
alter column created_at set default now(),
alter column updated_at set default now();

-- Add trigger
create trigger trg_legacy_table_updated_at
    before update on legacy_table
    for each row execute function update_updated_at();
```

## Query Patterns

### Find Recently Created

```sql
select * from products
where created_at > now() - interval '7 days'
order by created_at desc;
```

### Find Recently Modified

```sql
select * from products
where updated_at > now() - interval '24 hours'
order by updated_at desc;
```

### Find Unmodified Since Creation

```sql
select * from products
where created_at = updated_at;
```

### Find Records Modified by Specific User

```sql
select * from orders
where updated_by = $user_id
order by updated_at desc;
```

## Indexing Recommendations

```sql
-- For "recent records" queries
create index idx_products_created_at on products (created_at desc);

-- For "recently modified" queries
create index idx_products_updated_at on products (updated_at desc);

-- For "modified by user" queries (if using user tracking)
create index idx_orders_updated_by on orders (updated_by);
```

## Best Practices

1. **Always use timestamptz** - Store with timezone information
2. **Always NOT NULL** - Audit columns should never be null
3. **Always default now()** - Automatic timestamp on insert
4. **Use triggers for updated_at** - Don't rely on application code
5. **Consider user tracking** - For multi-user systems
6. **Index if querying** - Only add indexes if you query by these columns

## Type Choices

### Why timestamptz?

```sql
-- timestamptz stores in UTC, displays in session timezone
-- This is almost always what you want

-- BAD: timestamp without timezone
created_at timestamp  -- Ambiguous, server timezone dependent

-- GOOD: timestamp with timezone
created_at timestamptz  -- Unambiguous, always UTC internally
```

### Why UUID for user references?

```sql
-- Consistent with other IDs in the system
-- Works across distributed systems
-- No sequential information leakage
created_by uuid references users(id)
```

## Anti-Patterns

- **Nullable audit columns** - Makes queries and reasoning harder
- **Using timestamp instead of timestamptz** - Timezone confusion
- **Relying on app for updated_at** - Easy to forget, inconsistent
- **Over-indexing** - Don't index unless you query by these columns
- **Exposing internal timestamps** - Consider what to show in API responses
