# Phase 19: Agent Repository Modification

> Part of the [MVP Implementation Plan](mvp-implementation-plan.md)

**Goal:** Add `Version *string` field to the agent model and update all repository queries and tests.

**Agent(s):** go-software-engineer

**Dependencies:** Phase 16 (migration 008 for version column), Phase 17 (routing code removed)

---

## Step 1: Add Version field to agent model

- Modify file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/repository/agent/agent.go`
- Add field: `Version *string \`db:"version"\`` between `RoutingKeywords` and `CreatedAt`
- Add comment: `// Version is the semantic version of the agent definition. Nullable for pre-pivot agents.`
- Add deprecation comment to `RoutingKeywords`: `// Deprecated: retained for DB compatibility. New code should not use this field.`
- Agent: `go-software-engineer`
- Design reference: [Go Architecture Plan - Section 8](2026-02-15-go-architecture-plan.md#8-schema-changes-for-existing-tables)

## Step 2: Update Create query to include version

- Modify file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/repository/agent/repository.go`
- In `Create()`: add `version` to the INSERT column list and parameter list (9th parameter)
- Pass `agent.Version` as the 9th argument to `r.db.Exec()`
- Agent: `go-software-engineer`

## Step 3: Update Get query to include version

- Modify file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/repository/agent/repository.go`
- In `Get()`: add `version` to SELECT column list
- Add `&agent.Version` to `Scan()` call (between `routingKeywordsJSON` and `agent.CreatedAt`)
- Agent: `go-software-engineer`

## Step 4: Update Update query to include version

- Modify file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/repository/agent/repository.go`
- In `Update()`: add `version = $8` to SET clause
- Add `agent.Version` as 8th argument (shift `now` to 9th position, update WHERE clause parameter number if needed)
- Agent: `go-software-engineer`

## Step 5: Update List query to include version

- Modify file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/repository/agent/repository.go`
- In `List()`: add `version` to SELECT column list
- Add `&agent.Version` to `rows.Scan()` call
- Agent: `go-software-engineer`

## Step 6: Verify build compiles

- Run: `cd /Users/doublej/dev/mnemonic/src/mnemonic && go build ./internal/repository/agent/...`
- Agent: `go-software-engineer`

## Step 7: Update test helper

- Modify file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/repository/agent/repository_test.go`
- Update `testAgent()` to include `Version: ptr("1.0.0")` (add a local `ptr` helper: `func ptr(s string) *string { return &s }`)
- Agent: `go-software-engineer`

## Step 8: Update Create test mock expectations

- Modify file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/repository/agent/repository_test.go`
- In `TestRepository_Create`: update all `ExpectExec("INSERT INTO agents")` calls to expect 9 `WithArgs` instead of 8 (add `pgxmock.AnyArg()` for the version parameter)
- Agent: `go-software-engineer`

## Step 9: Update Get test mock expectations

- Modify file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/repository/agent/repository_test.go`
- In `TestRepository_Get`: update `pgxmock.NewRows` column lists to include `"version"`
- Update `AddRow` calls to include a version value (e.g., `ptr("1.0.0")` or `(*string)(nil)`)
- Update `wantAgent` expectations to include `Version`
- Agent: `go-software-engineer`

## Step 10: Update Update test mock expectations

- Modify file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/repository/agent/repository_test.go`
- In `TestRepository_Update`: update `ExpectExec("UPDATE agents SET")` calls to expect one additional `WithArgs` parameter for version
- Agent: `go-software-engineer`

## Step 11: Update List test mock expectations

- Modify file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/repository/agent/repository_test.go`
- In `TestRepository_List`: update `pgxmock.NewRows` column lists to include `"version"`
- Update all `AddRow` calls to include a version value
- Agent: `go-software-engineer`

## Step 12: Update remaining tests

- Update `TestRepository_Create_CheckConstraintViolation`, `TestRepository_Update_CheckConstraintViolation`, `TestRepository_JSONBMarshaling` with the new argument count
- Add a new test case: "Create with nil version succeeds" (pre-pivot agent with no version)
- Add a new test case: "Get agent with nil version" (returns agent where `Version` is `nil`)
- Agent: `go-software-engineer`

## Step 13: Run tests

- Run: `cd /Users/doublej/dev/mnemonic/src/mnemonic && go test -v ./internal/repository/agent/...`
- All tests must pass
- Agent: `go-software-engineer`

## Step 14: Commit

```bash
git add src/mnemonic/internal/repository/agent/
git commit -m "feat(pivot): add version field to agent model and repository"
```
