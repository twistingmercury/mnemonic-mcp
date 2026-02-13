# Code Review: Phase 14 - Rule Cache Startup Timeout

**Review Date:** 2026-02-12
**Reviewers:** code-review-agent, software-architect-agent, go-architect-agent
**Phase:** 14 (Rule cache startup timeout)

## Files Reviewed

### Source Files

- `src/mnemonic/internal/routing/cache.go` - Added `startupTimeout time.Duration` parameter to `NewRuleCache`

### Test Files

- `src/mnemonic/internal/routing/cache_test.go` - Added 4 timeout test cases, updated 3 existing call sites
- `src/mnemonic/internal/routing/engine_test.go` - Updated 1 call site in `newTestEngine` helper

### Documentation Files

- `docs/design/routing-engine.md` - Updated code example, startup behavior section, post-MVP features
- `docs/design/configuration.md` - Changed YAML comment from "IGNORED IN MVP" to active, added reference table row
- `CHANGELOG.md` - Added entry under Unreleased > Added
- `docs/plans/mvp-implementation-plan.md` - Phase 14 marked COMPLETE

## Validation Results

| Tool          | Result             |
| ------------- | ------------------ |
| go build      | PASS               |
| go test       | PASS (11 packages) |
| go test -race | PASS               |
| go vet        | PASS               |
| goimports     | PASS               |
| golangci-lint | PASS (0 issues)    |
| govulncheck   | no vulnerabilities |
| gosec         | 0 issues           |

## Design Compliance

Implementation satisfies Phase 14 plan at `docs/plans/2026-02-12-phase-14-rule-cache.md`. The `startupTimeout` parameter is wired into `NewRuleCache`, bounded by `context.WithTimeout` when positive, with zero/negative meaning no timeout.

### Behavioral Requirements Verified

- NewRuleCache accepts startupTimeout time.Duration as third parameter ✓
- Positive timeout creates context.WithTimeout ✓
- Zero timeout means no limit applied ✓
- Negative timeout means no limit applied ✓
- Timeout error wrapped as "failed to load rules at startup: %w" ✓
- All existing call sites updated to pass 0 (backward compatible) ✓
- Configuration reference table updated ✓
- YAML comment changed from "IGNORED IN MVP" to active ✓

### Design Doc Divergences (Post-Review)

None. No naming or structural divergences introduced.

## Findings

### HIGH Priority

| ID  | Source                   | Finding                                                                                                                                                                                                                                                                                                                                           | Resolution                                                                |
| --- | ------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------- |
| H1  | software-architect-agent | `docs/design/routing-engine.md:125` - Residual Phase 13 issue: text says "MatchConfig is an interface implemented by concrete types (KeywordMatchConfig, RegexMatchConfig, PatternMatchConfig, DefaultMatchConfig)". `DefaultMatchConfig` was deleted in Phase 13 but the reference was never removed. This is in the same file Phase 14 updated. | FIX: Remove `DefaultMatchConfig` from the parenthetical list on line 125. |

### MEDIUM Priority

| ID  | Source                                       | Finding                                                                                                                                                                                                                                                                                                                                                                                                                                 | Resolution                                                                                                                                                                                                      |
| --- | -------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| M1  | code-review-agent                            | `docs/function-documentation-map.md:250` - The `NewRuleCache` signature changed from `(ctx, loader)` to `(ctx, loader, startupTimeout)`. The function-documentation-map should reflect the new signature.                                                                                                                                                                                                                               | FIX: Update the function signature in the documentation map.                                                                                                                                                    |
| M2  | go-architect-agent, software-architect-agent | `src/mnemonic/internal/routing/cache_test.go:187-189,214-225` - The "load exceeds timeout" test case uses `10ms` timeout with `200ms` delay (wall-clock race). While the 20x ratio is generous, CI environments under load can exhibit scheduling jitter. The `select` pattern in the mock loader is correct, but replacing the time-based mock with a channel-block-until-cancel pattern would eliminate the wall-clock race entirely. | FIX: Replace `time.After(tt.loaderDelay)` mock with `<-ctx.Done(); return nil, ctx.Err()` pattern for the timeout test case. The test still runs in exactly `timeout` duration with zero scheduling dependence. |
| M3  | software-architect-agent                     | `src/mnemonic/tests/e2e/routing_rules_test.go:183,190` - Lines still reference `match_type: "default"` in skipped e2e test stubs. These are `t.Skip("not implemented")` stubs with no runtime impact, but reference a deleted concept from Phase 13.                                                                                                                                                                                    | DEFER to Phase 16: Stubs will be implemented when the route endpoint is wired. Update comments at that time.                                                                                                    |

### LOW Priority

| ID  | Source                   | Finding                                                                                                                                                                                                                                                                                                                                                                                           | Resolution                                                                            |
| --- | ------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------- |
| L1  | go-architect-agent       | `src/mnemonic/internal/routing/cache_test.go:232` - Missing `assert.ErrorIs(t, err, context.DeadlineExceeded)` assertion in the "load exceeds timeout" test case. The error wrapping with `%w` is correct, but the test only checks string containment, not the unwrap chain. Adding `errors.Is` would serve as a contract test ensuring callers can programmatically distinguish timeout errors. | FIX: Add `assert.ErrorIs(t, err, context.DeadlineExceeded)` to the timeout test case. |
| L2  | software-architect-agent | `docs/plans/mvp-implementation-plan.md:43` - Phase 14 description reads "Implement in-memory rule cache" which is misleading (cache was implemented in Phase 9; Phase 14 wires timeout config). Status correctly shows COMPLETE.                                                                                                                                                                  | ACCEPT: Historical description. No change needed.                                     |

## Patterns to Document

No new patterns discovered. Phase 14 correctly applies existing Go context and timeout patterns.

## Notes for Future Phases

**Phase 15** (Routing Unit Tests): Phase 13 finding M3 (dedicated `RecordNoMatch` metric) remains deferred to Phase 15.

**Phase 16** (Route Endpoint): Will pass `cfg.Routing.Cache.StartupTimeout` as third argument to `NewRuleCache`. Fix M3 e2e stub comments at that time.
