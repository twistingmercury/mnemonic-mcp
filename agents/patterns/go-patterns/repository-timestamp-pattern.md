---
entity_name: Repository Timestamp Pattern
entity_type: go-pattern
language: go
domain: backend
description: Application-layer timestamp management pattern for created_at and updated_at columns in repositories, following storage-only database philosophy
tags:
  - Go
  - repositories
  - timestamps
  - database
  - PostgreSQL
  - audit-columns
version: Go 1.21+
related_patterns:
  - SQL Migration Pattern
  - Audit Columns Pattern
---

# Repository Timestamp Pattern

## Philosophy

**Databases are storage-only** - No triggers, functions, or stored procedures. All business logic, including timestamp management, is handled in the application layer (Go repositories).

This pattern defines how Go repositories manage `created_at` and `updated_at` timestamp columns consistently across the codebase.

## Core Principles

1. **created_at** - Relies on database DEFAULT, handled automatically
2. **updated_at** - MUST be explicitly set in every UPDATE statement
3. **Use SQL NOW()** - Not Go's `time.Now()` (keeps time consistent with database)
4. **Explicit is better** - Make timestamp handling visible in SQL

## created_at Handling

### Database Schema

Tables include a `created_at` column with a DEFAULT value:

```sql
create table users (
    id uuid primary key default gen_random_uuid(),
    email text not null unique,
    name text not null,

    -- Audit columns
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);
```

### Repository INSERT Pattern

**Option 1: Let database handle it (Recommended)**

```go
func (r *UserRepository) Create(ctx context.Context, user *User) error {
    query := `
        insert into users (email, name, updated_at)
        values ($1, $2, now())
        returning id, created_at, updated_at
    `

    err := r.db.QueryRowContext(
        ctx,
        query,
        user.Email,
        user.Name,
    ).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

    return err
}
```

**Option 2: Explicitly set it**

```go
func (r *UserRepository) Create(ctx context.Context, user *User) error {
    query := `
        insert into users (email, name, created_at, updated_at)
        values ($1, $2, now(), now())
        returning id, created_at, updated_at
    `

    err := r.db.QueryRowContext(
        ctx,
        query,
        user.Email,
        user.Name,
    ).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

    return err
}
```

### Key Points

- **Let the database set the timestamp** using DEFAULT or SQL `now()`
- **Do NOT pass `time.Now()` from Go** - Let the database handle it
- **Always RETURNING the timestamp** - Populate the struct with the actual value
- **Consistency** - Use SQL `now()`, not Go `time.Now()`

## updated_at Handling

### Repository UPDATE Pattern

**ALWAYS explicitly set `updated_at` in UPDATE statements:**

```go
func (r *UserRepository) Update(ctx context.Context, user *User) error {
    query := `
        update users
        set email = $1,
            name = $2,
            updated_at = now()
        where id = $3
        returning updated_at
    `

    err := r.db.QueryRowContext(
        ctx,
        query,
        user.Email,
        user.Name,
        user.ID,
    ).Scan(&user.UpdatedAt)

    return err
}
```

### Partial Update Pattern

For partial updates (PATCH operations):

```go
func (r *UserRepository) UpdateEmail(ctx context.Context, id uuid.UUID, email string) error {
    query := `
        update users
        set email = $1,
            updated_at = now()
        where id = $2
    `

    _, err := r.db.ExecContext(ctx, query, email, id)
    return err
}
```

### Batch Update Pattern

Even batch updates must include `updated_at`:

```go
func (r *UserRepository) DeactivateUsers(ctx context.Context, ids []uuid.UUID) error {
    query := `
        update users
        set active = false,
            updated_at = now()
        where id = any($1)
    `

    _, err := r.db.ExecContext(ctx, query, pq.Array(ids))
    return err
}
```

### Key Points

- **NEVER forget `updated_at`** - Include in every UPDATE statement
- **Use SQL `now()`** - Not Go's `time.Now()`
- **RETURNING updated_at** - If you need the exact timestamp value
- **No triggers** - Application explicitly manages this, not the database

## Complete Repository Example

```go
package repository

import (
    "context"
    "database/sql"
    "time"

    "github.com/google/uuid"
)

type User struct {
    ID        uuid.UUID  `db:"id"`
    Email     string     `db:"email"`
    Name      string     `db:"name"`
    Active    bool       `db:"active"`
    CreatedAt time.Time  `db:"created_at"`
    UpdatedAt time.Time  `db:"updated_at"`
}

type UserRepository struct {
    db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
    return &UserRepository{db: db}
}

// Create inserts a new user. Database sets created_at, we set updated_at.
func (r *UserRepository) Create(ctx context.Context, user *User) error {
    query := `
        insert into users (email, name, active, updated_at)
        values ($1, $2, $3, now())
        returning id, created_at, updated_at
    `

    err := r.db.QueryRowContext(
        ctx,
        query,
        user.Email,
        user.Name,
        user.Active,
    ).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

    return err
}

// Update modifies an existing user. MUST set updated_at.
func (r *UserRepository) Update(ctx context.Context, user *User) error {
    query := `
        update users
        set email = $1,
            name = $2,
            active = $3,
            updated_at = now()
        where id = $4
        returning updated_at
    `

    err := r.db.QueryRowContext(
        ctx,
        query,
        user.Email,
        user.Name,
        user.Active,
        user.ID,
    ).Scan(&user.UpdatedAt)

    return err
}

// UpdatePartial demonstrates a partial update (single field).
func (r *UserRepository) UpdateEmail(ctx context.Context, id uuid.UUID, email string) error {
    query := `
        update users
        set email = $1,
            updated_at = now()
        where id = $2
    `

    result, err := r.db.ExecContext(ctx, query, email, id)
    if err != nil {
        return err
    }

    rows, err := result.RowsAffected()
    if err != nil {
        return err
    }

    if rows == 0 {
        return sql.ErrNoRows
    }

    return nil
}

// GetByID retrieves a user by ID.
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
    query := `
        select id, email, name, active, created_at, updated_at
        from users
        where id = $1
    `

    user := &User{}
    err := r.db.QueryRowContext(ctx, query, id).Scan(
        &user.ID,
        &user.Email,
        &user.Name,
        &user.Active,
        &user.CreatedAt,
        &user.UpdatedAt,
    )

    if err != nil {
        return nil, err
    }

    return user, nil
}

// Delete removes a user (or use soft delete pattern instead).
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
    query := `delete from users where id = $1`

    result, err := r.db.ExecContext(ctx, query, id)
    if err != nil {
        return err
    }

    rows, err := result.RowsAffected()
    if err != nil {
        return err
    }

    if rows == 0 {
        return sql.ErrNoRows
    }

    return nil
}
```

## Using sqlx for Cleaner Code

If using `jmoiron/sqlx`, the pattern is similar but with named parameters:

```go
import "github.com/jmoiron/sqlx"

type UserRepository struct {
    db *sqlx.DB
}

func (r *UserRepository) Create(ctx context.Context, user *User) error {
    query := `
        insert into users (email, name, active, updated_at)
        values (:email, :name, :active, now())
        returning id, created_at, updated_at
    `

    stmt, err := r.db.PrepareNamedContext(ctx, query)
    if err != nil {
        return err
    }
    defer stmt.Close()

    return stmt.QueryRowxContext(ctx, user).StructScan(user)
}

func (r *UserRepository) Update(ctx context.Context, user *User) error {
    query := `
        update users
        set email = :email,
            name = :name,
            active = :active,
            updated_at = now()
        where id = :id
        returning updated_at
    `

    stmt, err := r.db.PrepareNamedContext(ctx, query)
    if err != nil {
        return err
    }
    defer stmt.Close()

    return stmt.QueryRowxContext(ctx, user).Scan(&user.UpdatedAt)
}
```

## Anti-Patterns

### Don't: Forget updated_at in UPDATE

```go
// BAD: Missing updated_at
query := `
    update users
    set email = $1,
        name = $2
    where id = $3
`
```

### Don't: Pass time.Now() from Go

```go
// BAD: Using Go's time.Now() instead of SQL now()
query := `
    update users
    set email = $1,
        updated_at = $2
    where id = $3
`

_, err := r.db.ExecContext(ctx, query, user.Email, time.Now(), user.ID)
```

**Why?** The database timestamp and Go timestamp may differ slightly. Using SQL `now()` ensures consistency with database time.

### Don't: Rely on triggers

```go
// BAD: Assuming a trigger will handle updated_at
query := `
    update users
    set email = $1,
        name = $2
    where id = $3
`
// This only works if you have triggers - we DON'T use triggers!
```

### Don't: Manually set created_at on INSERT

```go
// BAD: Manually setting created_at from Go
query := `
    insert into users (email, name, created_at, updated_at)
    values ($1, $2, $3, $4)
`

_, err := r.db.ExecContext(ctx, query,
    user.Email,
    user.Name,
    time.Now(),  // Let the database handle this!
    time.Now(),
)
```

## Best Practices

1. **Let database set created_at** - Use DEFAULT in schema, omit from INSERT
2. **Always set updated_at** - Explicitly include `updated_at = now()` in every UPDATE
3. **Use SQL now()** - Not Go's `time.Now()`
4. **RETURNING timestamps** - Populate struct fields with actual database values
5. **Consistency** - Follow the same pattern across all repositories
6. **No triggers** - Application layer handles all timestamp logic
7. **Explicit over implicit** - Make timestamp management visible in SQL

## Why This Pattern?

### Storage-Only Database Philosophy

- **No hidden logic** - All timestamp management is visible in Go code
- **Easier debugging** - No mysterious trigger behavior
- **Portable** - Works with any SQL database, not just PostgreSQL
- **Testable** - Mock time using test fixtures, not database functions
- **Clear intent** - Developers see exactly what's happening

### Database Time vs Go Time

Using SQL `now()` instead of Go's `time.Now()`:

- **Consistency** - All timestamps use database server time
- **Cluster-safe** - Works correctly in multi-server deployments
- **Transaction-safe** - Timestamp set within transaction boundary
- **Time zone handling** - Database handles TZ conversion (timestamptz)

## Testing Timestamps

```go
func TestUserRepository_Update_SetsUpdatedAt(t *testing.T) {
    // Setup
    db := setupTestDB(t)
    repo := NewUserRepository(db)

    // Create initial user
    user := &User{
        Email:  "test@example.com",
        Name:   "Test User",
        Active: true,
    }

    err := repo.Create(context.Background(), user)
    require.NoError(t, err)

    originalUpdatedAt := user.UpdatedAt

    // Wait a moment to ensure timestamp changes
    time.Sleep(10 * time.Millisecond)

    // Update user
    user.Name = "Updated Name"
    err = repo.Update(context.Background(), user)
    require.NoError(t, err)

    // Assert: updated_at should be newer
    assert.True(t, user.UpdatedAt.After(originalUpdatedAt),
        "updated_at should be set to a newer timestamp")
}
```

## Summary

- **created_at**: Let database DEFAULT handle it (or use SQL `now()`)
- **updated_at**: Explicitly set to SQL `now()` in every UPDATE
- **No triggers**: Application manages all timestamps
- **SQL now() > Go time.Now()**: Use database time for consistency
- **Always RETURNING**: Populate struct with actual database values
