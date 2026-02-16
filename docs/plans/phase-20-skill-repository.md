# Phase 20: Skill Repository

> Part of the [MVP Implementation Plan](mvp-implementation-plan.md)

**Goal:** Implement the skill repository following the established agent repository pattern with JSONB document model. PK is `id UUID`, with unique `name VARCHAR(64)` constraint. Skills use `definition JSONB` column and `crc64 BIGINT` for change detection.

**Agent(s):** go-software-engineer

**Dependencies:** Phase 16 (migration 010 for skills table)

---

## Step 1: Create skill model

- Create file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/repository/skill/skill.go`
- Define `Skill` struct with fields: `ID uuid.UUID`, `Name string`, `Definition json.RawMessage` (db tag: "definition"), `CRC64 int64` (db tag: "crc64"), `CreatedAt time.Time`, `UpdatedAt time.Time`
- The `Definition` field stores JSONB containing: `description`, `content`, `version`, `tags`
- Follow the same `db` tag convention as agent model
- Agent: `go-software-engineer`
- Design reference: [Go Architecture Plan - Section 9.1](2026-02-15-go-architecture-plan.md#91-skill-repository), [API Specification - Skill schema](../design/2026-02-15-pivot-api-specification.md#6-request-and-response-schemas)

## Step 2: Create skill errors

- Create file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/repository/skill/errors.go`
- Define: `ErrExists = errors.New("skill already exists")`, `ErrNotFound = errors.New("skill not found")`
- Agent: `go-software-engineer`

## Step 3: Create skill repository interface and pgx implementation

- Create file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/repository/skill/repository.go`
- Define `Filter` struct: `Tags []string`, `SearchQuery string`
- Define `Repository` interface: `Create`, `Get` (by name), `Update`, `Delete`, `List` (with `Filter` + `repository.ListOptions`), `Exists`
- Implement `pgxRepository` struct with `db repository.DBTX`
- Implement `NewRepository(db repository.DBTX) Repository`
- Implement all CRUD methods following agent repository pattern
- Marshal/unmarshal `Definition` JSONB field to/from skill struct
- CRC64 computed on write operations using Go's `hash/crc64` with ISO polynomial
- Application sets `updated_at` on update operations (no database trigger)
- SQL queries use `id` (UUID) as primary key, with `name` as unique lookup key for Get/Update/Delete operations
- `List` supports tag filtering (`WHERE definition->'tags' @> $N::jsonb`) and text search (`WHERE name ILIKE $N OR definition->>'description' ILIKE $N`)
- Agent: `go-software-engineer`

## Step 4: Create doc.go

- Create file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/repository/skill/doc.go`
- Package doc comment following the agent package convention
- Agent: `go-software-engineer`

## Step 5: Verify build compiles

- Run: `cd /Users/doublej/dev/mnemonic/src/mnemonic && go build ./internal/repository/skill/...`
- Agent: `go-software-engineer`

## Step 6: Write unit tests

- Create file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/repository/skill/repository_test.go`
- Follow agent test pattern: table-driven tests with `pgxmock`
- Test cases for each method:
  - `Create`: successful, duplicate name (ErrExists), CRC64 computed correctly
  - `Get`: successful, not found (ErrNotFound), corrupted definition JSONB
  - `Update`: successful, not found, CRC64 updated, `updated_at` set by application
  - `Delete`: successful, not found
  - `List`: all skills, with limit/offset, empty result, with tag filter (JSONB query), with search query (JSONB field access)
  - `Exists`: exists, does not exist, database error
  - JSONB marshaling: definition field marshal/unmarshal, nested tags array
- Agent: `go-software-engineer`

## Step 7: Run tests

- Run: `cd /Users/doublej/dev/mnemonic/src/mnemonic && go test -v ./internal/repository/skill/...`
- All tests must pass
- Agent: `go-software-engineer`

## Step 8: Run full test suite

- Run: `cd /Users/doublej/dev/mnemonic/src/mnemonic && go test ./...`
- All tests must pass (no regressions)
- Agent: `go-software-engineer`

## Step 9: Commit

```bash
git add src/mnemonic/internal/repository/skill/
git commit -m "feat(pivot): implement skill repository (model, interface, pgx impl, tests)"
```
