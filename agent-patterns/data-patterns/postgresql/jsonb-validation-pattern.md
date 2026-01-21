---
entity_name: JSONB Validation Pattern
entity_type: database-pattern
language: agnostic
domain: backend
description: PostgreSQL pattern for JSONB columns with CHECK constraints, ensuring data integrity for semi-structured data.
tags:
  - PostgreSQL
  - JSONB
  - validation
  - constraints
  - semi-structured
version: PostgreSQL 14+
related_patterns:
  - SQL Migration Pattern
  - Audit Columns Pattern
---

# JSONB Validation Pattern

This pattern provides techniques for validating JSONB data at the database level using CHECK constraints.

## Overview

JSONB columns offer flexibility but can become a data quality risk. Use CHECK constraints to enforce structure while maintaining flexibility.

## Basic Structure Validation

### Ensure Object Type

```sql
create table settings (
    id uuid primary key default gen_random_uuid(),
    user_id uuid not null references users(id),

    -- JSONB must be an object, not array or scalar
    preferences jsonb not null default '{}',
    constraint preferences_is_object check (jsonb_typeof(preferences) = 'object')
);
```

### Ensure Array Type

```sql
create table products (
    id uuid primary key default gen_random_uuid(),
    name text not null,

    -- Tags must be an array
    tags jsonb not null default '[]',
    constraint tags_is_array check (jsonb_typeof(tags) = 'array')
);
```

## Required Fields Validation

### Single Required Field

```sql
create table events (
    id uuid primary key default gen_random_uuid(),

    -- Payload must have 'type' field
    payload jsonb not null,
    constraint payload_has_type check (payload ? 'type')
);
```

### Multiple Required Fields

```sql
create table audit_events (
    id uuid primary key default gen_random_uuid(),

    -- Data must have action and timestamp
    data jsonb not null,
    constraint data_has_required_fields check (
        data ? 'action' and data ? 'timestamp'
    )
);
```

## Field Type Validation

### Validate Field Is String

```sql
create table configs (
    id uuid primary key default gen_random_uuid(),

    metadata jsonb not null default '{}',
    constraint metadata_version_is_string check (
        metadata->>'version' is null
        or jsonb_typeof(metadata->'version') = 'string'
    )
);
```

### Validate Field Is Number

```sql
create table metrics (
    id uuid primary key default gen_random_uuid(),

    data jsonb not null,
    constraint data_value_is_number check (
        data ? 'value' and jsonb_typeof(data->'value') = 'number'
    )
);
```

### Validate Field Is Integer (Specific Range)

```sql
create table versioned_docs (
    id uuid primary key default gen_random_uuid(),

    metadata jsonb not null default '{"version": 1}',
    constraint metadata_version_valid check (
        (metadata->>'version')::int >= 1
        and (metadata->>'version')::int <= 1000
    )
);
```

## Enum-like Validation

### Validate Field Against Allowed Values

```sql
create table notifications (
    id uuid primary key default gen_random_uuid(),

    config jsonb not null,
    constraint config_channel_valid check (
        config->>'channel' in ('email', 'sms', 'push', 'webhook')
    )
);
```

### Validate Array Elements

```sql
create table subscriptions (
    id uuid primary key default gen_random_uuid(),

    -- All topics must be from allowed list
    topics jsonb not null default '[]',
    constraint topics_are_valid check (
        topics <@ '["news", "updates", "alerts", "marketing"]'::jsonb
    )
);
```

## Complex Validation Examples

### Conditional Validation

```sql
create table integrations (
    id uuid primary key default gen_random_uuid(),

    config jsonb not null,
    -- If type is 'webhook', url is required
    constraint config_webhook_has_url check (
        config->>'type' != 'webhook'
        or (config->>'type' = 'webhook' and config ? 'url')
    )
);
```

### Nested Object Validation

```sql
create table orders (
    id uuid primary key default gen_random_uuid(),

    shipping jsonb not null,
    constraint shipping_address_valid check (
        jsonb_typeof(shipping->'address') = 'object'
        and shipping->'address' ? 'street'
        and shipping->'address' ? 'city'
        and shipping->'address' ? 'postal_code'
    )
);
```

### Array Length Validation

```sql
create table surveys (
    id uuid primary key default gen_random_uuid(),

    -- Must have between 1 and 10 questions
    questions jsonb not null,
    constraint questions_count_valid check (
        jsonb_array_length(questions) >= 1
        and jsonb_array_length(questions) <= 10
    )
);
```

## Migration Example

```sql
-- migrations/NNN_create_user_preferences.up.sql
-- Creates user_preferences table with validated JSONB settings.

create table if not exists user_preferences (
    id uuid primary key default gen_random_uuid(),
    user_id uuid not null references users(id) unique,

    -- Main settings object
    settings jsonb not null default '{
        "theme": "light",
        "notifications": {
            "email": true,
            "push": false
        },
        "language": "en"
    }',

    -- Constraints
    constraint settings_is_object check (
        jsonb_typeof(settings) = 'object'
    ),
    constraint settings_theme_valid check (
        settings->>'theme' is null
        or settings->>'theme' in ('light', 'dark', 'system')
    ),
    constraint settings_language_valid check (
        settings->>'language' is null
        or settings->>'language' ~ '^[a-z]{2}(-[A-Z]{2})?$'
    ),
    constraint settings_notifications_valid check (
        settings->'notifications' is null
        or jsonb_typeof(settings->'notifications') = 'object'
    ),

    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create trigger trg_user_preferences_updated_at
    before update on user_preferences
    for each row execute function update_updated_at();
```

## Indexing JSONB

### GIN Index for Containment Queries

```sql
-- For queries using @>, ?, ?|, ?&
create index idx_settings_gin on user_preferences using gin (settings);

-- Query that uses this index:
select * from user_preferences
where settings @> '{"theme": "dark"}';
```

### Expression Index for Specific Fields

```sql
-- For queries on specific JSON path
create index idx_settings_theme on user_preferences ((settings->>'theme'));

-- Query that uses this index:
select * from user_preferences
where settings->>'theme' = 'dark';
```

## Best Practices

1. **Default to empty object/array** - `default '{}'` or `default '[]'`
2. **Use NOT NULL** - Prefer empty object over NULL
3. **Validate at boundary** - Check structure, not every possible value
4. **Document the schema** - Add comments explaining expected structure
5. **Consider migration** - How will you evolve the JSON structure?
6. **Index what you query** - GIN for containment, expression for specific paths

## When NOT to Use JSONB

JSONB is great for:
- Configuration/preferences
- Variable schema data
- Audit/event payloads
- External API responses

Consider relational columns for:
- Frequently queried fields
- Fields needing foreign keys
- Fields needing unique constraints
- Fields with complex validation

## Application Integration

### Go Struct with JSONB

```go
type UserPreferences struct {
    ID       uuid.UUID `db:"id"`
    UserID   uuid.UUID `db:"user_id"`
    Settings Settings  `db:"settings"`
}

type Settings struct {
    Theme         string        `json:"theme"`
    Language      string        `json:"language"`
    Notifications Notifications `json:"notifications"`
}

// Scan implements sql.Scanner for JSONB
func (s *Settings) Scan(src interface{}) error {
    return json.Unmarshal(src.([]byte), s)
}

// Value implements driver.Valuer for JSONB
func (s Settings) Value() (driver.Value, error) {
    return json.Marshal(s)
}
```

## Error Messages

When constraints fail, errors look like:

```
ERROR: new row for relation "user_preferences" violates check constraint "settings_theme_valid"
DETAIL: Failing row contains (...).
```

Consider using custom error handling in your application to provide user-friendly messages.
