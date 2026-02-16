# Phase 21: Command Repository

> Part of the [MVP Implementation Plan](mvp-implementation-plan.md)

**Goal:** Implement the command repository. Mirrors the skill repository pattern exactly -- same PK convention (UUID `id` with unique `name VARCHAR(255)` constraint), same interface shape, same JSONB document model with `definition JSONB` and `crc64 BIGINT`, different table name.

**Agent(s):** go-software-engineer

**Dependencies:** Phase 16 (migration 011 for commands table)

---

## Step 1: Create command model

- Create file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/repository/command/command.go`
- Define `Command` struct with fields: `ID uuid.UUID`, `Name string`, `Definition json.RawMessage` (db tag: "definition"), `CRC64 int64` (db tag: "crc64"), `CreatedAt time.Time`, `UpdatedAt time.Time`
- The `Definition` field stores JSONB containing: `description`, `content`, `version`, `tags`
- Agent: `go-software-engineer`
- Design reference: [Go Architecture Plan - Section 9.2](2026-02-15-go-architecture-plan.md#92-command-repository)

## Step 2: Create command errors

- Create file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/repository/command/errors.go`
- Define: `ErrExists`, `ErrNotFound`
- Agent: `go-software-engineer`

## Step 3: Create command repository interface and pgx implementation

- Create file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/repository/command/repository.go`
- Define `Filter` struct, `Repository` interface, `pgxRepository` -- mirrors skill package exactly, targeting `commands` table
- Marshal/unmarshal `Definition` JSONB field
- CRC64 computed on write operations using Go's `hash/crc64` with ISO polynomial
- Application sets `updated_at` on update operations (no database trigger)
- Tag filtering uses `WHERE definition->'tags' @> $N::jsonb`
- Text search uses `WHERE name ILIKE $N OR definition->>'description' ILIKE $N`
- Agent: `go-software-engineer`

## Step 4: Create doc.go

- Create file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/repository/command/doc.go`
- Agent: `go-software-engineer`

## Step 5: Verify build compiles

- Run: `cd /Users/doublej/dev/mnemonic/src/mnemonic && go build ./internal/repository/command/...`
- Agent: `go-software-engineer`

## Step 6: Write unit tests

- Create file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/repository/command/repository_test.go`
- Same test structure as skill repository tests, targeting `commands` table
- Test CRC64 computation, JSONB definition marshaling, application-managed `updated_at`
- Agent: `go-software-engineer`

## Step 7: Run tests

- Run: `cd /Users/doublej/dev/mnemonic/src/mnemonic && go test -v ./internal/repository/command/...`
- Run: `cd /Users/doublej/dev/mnemonic/src/mnemonic && go test ./...` -- no regressions
- Agent: `go-software-engineer`

## Step 8: Commit

```bash
git add src/mnemonic/internal/repository/command/
git commit -m "feat(pivot): implement command repository (model, interface, pgx impl, tests)"
```
