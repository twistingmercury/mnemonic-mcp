# Phase 17: Remove Routing Code

> Part of the [MVP Implementation Plan](mvp-implementation-plan.md)

**Goal:** Delete all routing-related Go packages and references. The build must pass with all remaining tests green.

**Agent(s):** go-software-engineer

**Dependencies:** Phase 15 (migration cleanup)

---

## Step 1: Delete routing engine package

- Delete directory: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/routing/` (entire directory, ~19 files, ~4,214 lines)
- Agent: `go-software-engineer`
- Verify: `ls /Users/doublej/dev/mnemonic/src/mnemonic/internal/routing/` returns "No such file or directory"

## Step 2: Delete routing rule repository package

- Delete directory: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/repository/routingrule/` (entire directory, 5 files, ~1,773 lines)
- Agent: `go-software-engineer`
- Verify: `ls /Users/doublej/dev/mnemonic/src/mnemonic/internal/repository/routingrule/` returns "No such file or directory"

## Step 3: Delete routing handler stubs

- Delete directory: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/handlers/routes/` (entire directory, including `rules/` subdirectory)
- Agent: `go-software-engineer`
- Verify: `ls /Users/doublej/dev/mnemonic/src/mnemonic/internal/handlers/routes/` returns "No such file or directory"

## Step 4: Delete routing metrics

- Delete file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/metrics/routing.go`
- Delete file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/metrics/routing_test.go`
- Agent: `go-software-engineer`
- Verify: `ls /Users/doublej/dev/mnemonic/src/mnemonic/internal/metrics/routing*.go` returns no files

## Step 5: Delete routing E2E tests

- Delete file: `/Users/doublej/dev/mnemonic/src/mnemonic/tests/e2e/routing_test.go`
- Delete file: `/Users/doublej/dev/mnemonic/src/mnemonic/tests/e2e/routing_rules_test.go`
- Agent: `go-software-engineer`
- Verify: `ls /Users/doublej/dev/mnemonic/src/mnemonic/tests/e2e/routing*.go` returns no files

## Step 6: Update metrics registry -- remove routing field

- Modify file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/metrics/registry.go`
- Remove `Routing *Routing` field from the `Registry` struct
- Remove `NewRouting(meter)` call and `routing` variable from `NewRegistry()`
- Remove `Routing: routing` from the `Registry` struct literal
- Remove the `"fmt"` import if no longer needed (check other usages)
- Agent: `go-software-engineer`
- Verify: `grep -c Routing /Users/doublej/dev/mnemonic/src/mnemonic/internal/metrics/registry.go` returns `0`

## Step 7: Update metrics registry test -- remove routing assertions

- Modify file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/metrics/registry_test.go`
- Remove any assertions referencing `registry.Routing`
- Agent: `go-software-engineer`

## Step 8: Fix any remaining import references

- Search all `.go` files for imports of deleted packages:
  - `"github.com/twistingmercury/mnemonic/internal/routing"`
  - `"github.com/twistingmercury/mnemonic/internal/repository/routingrule"`
  - `"github.com/twistingmercury/mnemonic/internal/handlers/routes"`
- Remove any found references
- Agent: `go-software-engineer`
- Verify: `cd /Users/doublej/dev/mnemonic/src/mnemonic && grep -r "internal/routing" --include="*.go" .` returns nothing
- Verify: `cd /Users/doublej/dev/mnemonic/src/mnemonic && grep -r "repository/routingrule" --include="*.go" .` returns nothing

## Step 9: Verify build and tests pass

- Run: `cd /Users/doublej/dev/mnemonic/src/mnemonic && go build ./...`
- Run: `cd /Users/doublej/dev/mnemonic/src/mnemonic && go vet ./...`
- Run: `cd /Users/doublej/dev/mnemonic/src/mnemonic && go test ./...` -- all remaining tests must pass
- Agent: `go-software-engineer`

## Step 10: Commit

```bash
git add -A src/mnemonic/internal/routing/ src/mnemonic/internal/repository/routingrule/ \
  src/mnemonic/internal/handlers/routes/ src/mnemonic/internal/metrics/routing.go \
  src/mnemonic/internal/metrics/routing_test.go src/mnemonic/internal/metrics/registry.go \
  src/mnemonic/internal/metrics/registry_test.go src/mnemonic/tests/e2e/routing_test.go \
  src/mnemonic/tests/e2e/routing_rules_test.go
git commit -m "feat(pivot): remove routing engine, routing rule repo, routing handlers, routing metrics (~6300 LOC)"
```
