# Phase 14: Rule Cache Configuration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Wire the existing `routing.cache.startup_timeout` configuration into `NewRuleCache` so the initial rule load is bounded by a configurable timeout, preventing indefinite hangs if the database is unreachable at startup.

**Architecture:** The in-memory `RuleCache` was built in Phase 9 with a one-shot startup load and no timeout. The `RoutingCacheConfig` struct (with `StartupTimeout` and `RefreshTTL` fields) was defined in Phase 2 but never wired. Phase 14 connects `StartupTimeout` to the cache constructor. `RefreshTTL` remains unused (post-MVP: background refresh requires a ticker and `ReloadRules` method not yet designed).

**Tech Stack:** Go 1.25, testify (assert/require), github.com/google/uuid

---

## Non-Negotiables (from MVP Implementation Plan)

Before this phase is complete:

- Unit tests written and passing; existing tests unbroken
- Static analysis clean: `goimports`, `golangci-lint`, `govulncheck`, `gosec`
- Code review workflow ran; findings dispositioned
- CI build passes locally
- Docs updated (CHANGELOG, design docs, function-documentation-map)
- Committed, pushed, PR created and merged to develop

---

## Summary of Changes

| File                                 | Action | What                                                             |
| ------------------------------------ | ------ | ---------------------------------------------------------------- |
| `internal/routing/cache.go`          | Modify | Add `startupTimeout time.Duration` parameter to `NewRuleCache`   |
| `internal/routing/cache_test.go`     | Modify | Add timeout tests; update existing calls with new parameter      |
| `internal/routing/engine_test.go`    | Modify | Update `newTestEngine` helper for new `NewRuleCache` signature   |
| `docs/design/routing-engine.md`      | Modify | Update code example and startup behavior section                 |
| `docs/design/configuration.md`       | Modify | Mark `startup_timeout` as active; add to reference table         |
| `docs/function-documentation-map.md` | Modify | No change needed (function name unchanged, signature not in map) |

---

## Task 1: Write failing tests for startup timeout

**Files:**

- Modify: `src/mnemonic/internal/routing/cache_test.go`

**Step 1: Add the timeout test function**

Append this test after `TestRuleCache_RuleCount` (line 165):

```go
func TestNewRuleCache_StartupTimeout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		timeout        time.Duration
		loaderDelay    time.Duration
		wantErr        bool
		wantErrContain string
		wantCount      int
	}{
		{
			name:        "load completes within timeout",
			timeout:     1 * time.Second,
			loaderDelay: 0,
			wantErr:     false,
			wantCount:   1,
		},
		{
			name:           "load exceeds timeout",
			timeout:        10 * time.Millisecond,
			loaderDelay:    200 * time.Millisecond,
			wantErr:        true,
			wantErrContain: "failed to load rules at startup",
		},
		{
			name:        "zero timeout means no limit",
			timeout:     0,
			loaderDelay: 10 * time.Millisecond,
			wantErr:     false,
			wantCount:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			loader := &mockRuleLoader{
				loadFn: func(ctx context.Context) ([]*routingrule.Rule, error) {
					if tt.loaderDelay > 0 {
						select {
						case <-time.After(tt.loaderDelay):
						case <-ctx.Done():
							return nil, ctx.Err()
						}
					}
					return []*routingrule.Rule{
						{ID: uuid.New(), Name: "rule-1", Priority: 100, Enabled: true},
					}, nil
				},
			}

			cache, err := routing.NewRuleCache(context.Background(), loader, tt.timeout)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrContain)
				assert.Nil(t, cache)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, cache)
			assert.Equal(t, tt.wantCount, cache.RuleCount())
		})
	}
}
```

Add `"time"` to the import block in `cache_test.go`.

**Step 2: Run the test to verify it fails**

Run: `cd src/mnemonic && go test ./internal/routing/ -run TestNewRuleCache_StartupTimeout -v`

Expected: **Compilation error** — `NewRuleCache` does not accept a third argument yet. This confirms the test is correctly calling the new API.

**Step 3: Commit**

```
git add src/mnemonic/internal/routing/cache_test.go
git commit -m "test: add startup timeout tests for NewRuleCache (Phase 14)

Red phase: tests fail because NewRuleCache does not yet accept
startupTimeout parameter."
```

---

## Task 2: Implement startup timeout in NewRuleCache

**Files:**

- Modify: `src/mnemonic/internal/routing/cache.go`

**Step 1: Update the function signature and implementation**

Replace the existing `NewRuleCache` function (lines 27-45) with:

```go
// NewRuleCache creates a new RuleCache by loading rules from the provided loader.
// Rules are sorted by priority DESC, then by ID ASC (lexicographic) for deterministic
// tie-breaking. Returns an error if loading fails (fail-fast on startup).
//
// If startupTimeout is positive, the initial load is bounded by that duration.
// A zero or negative startupTimeout means no timeout is applied.
func NewRuleCache(ctx context.Context, loader RuleLoader, startupTimeout time.Duration) (*RuleCache, error) {
	if startupTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, startupTimeout)
		defer cancel()
	}

	rules, err := loader.LoadRules(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load rules at startup: %w", err)
	}

	// Sort rules: priority descending, then ID ascending for tie-breaking.
	sort.Slice(rules, func(i, j int) bool {
		if rules[i].Priority != rules[j].Priority {
			return rules[i].Priority > rules[j].Priority
		}
		return rules[i].ID.String() < rules[j].ID.String()
	})

	return &RuleCache{rules: rules}, nil
}
```

Add `"time"` to the import block in `cache.go`.

**Step 2: Run the new timeout tests**

Run: `cd src/mnemonic && go test ./internal/routing/ -run TestNewRuleCache_StartupTimeout -v -count=1`

Expected: **Compilation error** — existing call sites in `cache_test.go` and `engine_test.go` still use the old 2-argument signature. The new tests themselves would pass, but the package won't compile.

---

## Task 3: Update existing call sites for new signature

**Files:**

- Modify: `src/mnemonic/internal/routing/cache_test.go` (3 call sites)
- Modify: `src/mnemonic/internal/routing/engine_test.go` (1 call site)

**Step 1: Update cache_test.go**

In `TestNewRuleCache` (line 80), change:

```go
cache, err := routing.NewRuleCache(context.Background(), tt.loader)
```

to:

```go
cache, err := routing.NewRuleCache(context.Background(), tt.loader, 0)
```

In `TestRuleCache_GetRules_ReturnsCopy` (line 115), change:

```go
cache, err := routing.NewRuleCache(context.Background(), loader)
```

to:

```go
cache, err := routing.NewRuleCache(context.Background(), loader, 0)
```

In `TestRuleCache_RuleCount` (line 159), change:

```go
cache, err := routing.NewRuleCache(context.Background(), loader)
```

to:

```go
cache, err := routing.NewRuleCache(context.Background(), loader, 0)
```

**Step 2: Update engine_test.go**

In `newTestEngine` helper (line 27), change:

```go
cache, err := routing.NewRuleCache(context.Background(), loader)
```

to:

```go
cache, err := routing.NewRuleCache(context.Background(), loader, 0)
```

**Step 3: Run the full routing test suite**

Run: `cd src/mnemonic && go test ./internal/routing/ -v -count=1`

Expected: **ALL PASS** — existing tests pass with `0` timeout (no limit applied), new timeout tests pass with the implementation.

**Step 4: Run the full module test suite for regressions**

Run: `cd src/mnemonic && go test ./... -count=1`

Expected: **ALL PASS** — no other packages call `NewRuleCache` directly.

**Step 5: Commit**

```
git add src/mnemonic/internal/routing/cache.go src/mnemonic/internal/routing/cache_test.go src/mnemonic/internal/routing/engine_test.go
git commit -m "feat: add startup timeout to NewRuleCache (Phase 14)

NewRuleCache accepts a startupTimeout parameter that bounds the
initial rule load. A zero value means no timeout. This wires the
routing.cache.startup_timeout configuration into the cache layer."
```

---

## Task 4: Run static analysis

**Step 1: Run goimports**

Run: `cd src/mnemonic && goimports -w internal/routing/cache.go internal/routing/cache_test.go internal/routing/engine_test.go`

Expected: Files reformatted (or no changes if imports are already correct).

**Step 2: Run golangci-lint**

Run: `cd src/mnemonic && golangci-lint run ./internal/routing/...`

Expected: 0 issues.

**Step 3: Run govulncheck**

Run: `cd src/mnemonic && govulncheck ./...`

Expected: No vulnerabilities.

**Step 4: Run gosec**

Run: `cd src/mnemonic && gosec ./internal/routing/...`

Expected: 0 issues.

**Step 5: Fix any issues found, re-run tests, commit if needed**

If any tool reports issues, fix them and re-run:

```
cd src/mnemonic && go test ./internal/routing/ -v -count=1
```

---

## Task 5: Update documentation

**Files:**

- Modify: `docs/design/routing-engine.md` (lines 415-437, 489-506)
- Modify: `docs/design/configuration.md` (lines 159-160, after line 297)

### Step 1: Update routing-engine.md — Cache Architecture code example

Replace the `NewRuleCache` code block (lines 423-437) with:

```go
func NewRuleCache(ctx context.Context, loader RuleLoader, startupTimeout time.Duration) (*RuleCache, error) {
    if startupTimeout > 0 {
        var cancel context.CancelFunc
        ctx, cancel = context.WithTimeout(ctx, startupTimeout)
        defer cancel()
    }

    rules, err := loader.LoadRules(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to load rules at startup: %w", err)
    }

    // Sort rules: priority descending, then ID ascending for tie-breaking.
    sort.Slice(rules, func(i, j int) bool {
        if rules[i].Priority != rules[j].Priority {
            return rules[i].Priority > rules[j].Priority
        }
        return rules[i].ID.String() < rules[j].ID.String()
    })

    return &RuleCache{rules: rules}, nil
}
```

### Step 2: Update routing-engine.md — Startup behavior section

In the "Startup failure behavior (MVP)" section (lines 478-482), add a bullet:

```markdown
- The initial load is bounded by `startup_timeout` (default 30s); a hung database
  triggers a timeout error rather than blocking indefinitely
```

### Step 3: Update routing-engine.md — Move startup_timeout to MVP

In the "Rule Reloading (Post-MVP)" section (lines 489-506), move `startup_timeout` out of the planned features list and note it is now implemented:

Change the planned features list to:

```markdown
**Planned features:**

- Background refresh via ticker with configurable `refresh_interval`
- Explicit cache invalidation when rules are modified via admin API
- Graceful degradation for refresh failures (use stale cache)
- `Engine.ReloadRules(ctx context.Context) error` method for on-demand refresh
```

And update the planned configuration table to remove `startup_timeout`:

```markdown
**Planned configuration:**

| Setting            | Default | Description                 |
| ------------------ | ------- | --------------------------- |
| `refresh_interval` | 5m      | Background refresh interval |
```

### Step 4: Update configuration.md — YAML example

Change line 160 from:

```yaml
startup_timeout: 30s # IGNORED IN MVP: Timeout for initial cache load
```

to:

```yaml
startup_timeout: 30s # Timeout for initial rule cache load at startup
```

### Step 5: Update configuration.md — Reference table

After the `routing.cache.refresh_ttl` row (line 297), add a new row:

```markdown
| `routing.cache.startup_timeout` | duration | `30s` | `MNEMONIC_ROUTING_CACHE_STARTUP_TIMEOUT` | Timeout for initial rule cache load at startup; 0 disables |
```

### Step 6: Commit

```
git add docs/design/routing-engine.md docs/design/configuration.md
git commit -m "docs: update cache architecture and config for Phase 14

Mark startup_timeout as active (no longer ignored in MVP).
Update NewRuleCache code example with timeout parameter.
Add startup_timeout to configuration reference table."
```

---

## Task 6: Update CHANGELOG

**Files:**

- Modify: `CHANGELOG.md`

**Step 1: Add entry under [Unreleased] > Added**

Add to the `### Added` section:

```markdown
- Startup timeout for rule cache (`routing.cache.startup_timeout`) bounds
  initial rule load to prevent indefinite hangs (default 30s)
```

**Step 2: Commit**

```
git add CHANGELOG.md
git commit -m "docs: add Phase 14 rule cache entry to CHANGELOG"
```

---

## Task 7: Final validation

**Step 1: Run full test suite with race detector**

Run: `cd src/mnemonic && go test -race ./... -count=1`

Expected: ALL PASS, no data races.

**Step 2: Run full static analysis suite**

Run: `cd src/mnemonic && goimports -l ./internal/routing/ && golangci-lint run ./... && govulncheck ./... && gosec ./...`

Expected: Clean output, 0 issues.

**Step 3: Verify CI build locally**

Run: `cd src/mnemonic && ./build/build.sh`

Expected: Build succeeds, all tests pass, Docker image built.

---

## What Phase 14 Does NOT Include

These are explicitly out of scope:

- **Background refresh** (`RefreshTTL` / `refresh_interval`): Post-MVP feature requiring a ticker goroutine and `ReloadRules` method. Config field exists for forward compatibility but remains unused.
- **Cache hit/miss metrics wiring**: `RecordCacheHit`/`RecordCacheMiss` counters exist in `metrics.Routing` but are semantically meaningful only with post-MVP refresh logic. In MVP, every `GetRules()` is a "hit" by definition. Wiring them now would add noise without insight.
- **`Engine.ReloadRules()`**: Post-MVP method for on-demand cache refresh.
- **Admin API cache invalidation**: Post-MVP feature.

---

## Dependency Context

- **Depends on**: Phase 9 (routing engine) — complete
- **Blocks**: Phase 15 (routing unit tests), Phase 16 (route endpoint wiring)
- **Phase 16 wiring note**: When Phase 16 wires the server, it will pass `cfg.Routing.Cache.StartupTimeout` as the third argument to `NewRuleCache`
