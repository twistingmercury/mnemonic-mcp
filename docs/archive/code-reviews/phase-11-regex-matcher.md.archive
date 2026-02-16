# Code Review: Phase 11 - Regex Matcher

**Review Date:** 2026-02-10
**Reviewers:** code-review-agent, software-architect-agent, go-architect-agent, Codex
**Phase:** 11 (Implement Regex Matcher with Compiled Pattern Caching)

## Files Reviewed

### Source Files

- `src/mnemonic/internal/routing/regex.go` - RegexMatcher struct, Match, getOrCompile, applyRegexFlags, cache cleanup
- `src/mnemonic/internal/routing/export_test.go` - Added regex matcher test exports (ExportNewRegexMatcherForTest, ExportCleanExpiredRegexPatterns, ExportRegexCacheLen, ExportRegexCacheLastUsed)
- `src/mnemonic/internal/routing/keyword.go` - Added nolint directive for false positive unusedfunc lint warning

### Test Files

- `src/mnemonic/internal/routing/regex_test.go` - 16 test functions, 22+ subtests
- `src/mnemonic/internal/routing/regex_benchmark_test.go` - 7 benchmarks

### Other Files

- `docs/plans/mvp-implementation-plan.md` - Cleared erroneous COMPLETE status for Phase 11

## Validation Results

| Tool | Result |
|------|--------|
| goimports | Clean |
| go vet | No issues |
| go test -race ./internal/routing/... | 48 tests, all PASS, no races (1.036s) |
| go test -cover ./internal/routing/... | 93.9% total; regex.go functions at 100% |
| golangci-lint run ./internal/routing/... | 0 issues |
| govulncheck | No vulnerabilities |
| gosec | 0 issues |

## Design Compliance

Implementation satisfies all Phase 11 behavioral requirements from the routing engine design doc (docs/design/routing-engine.md), Regex Matcher section.

### Behavioral Requirements Verified

- Compiled regex patterns cached for performance (map[string]*cachedRegex with sliding TTL) ✓
- Standard Go regex syntax via regexp.Compile (RE2) ✓
- Optional flag "i" for case-insensitive matching via (?i) prefix ✓
- Matching is unanchored (matches anywhere in prompt) ✓
- Confidence always 1.0 for matches (deterministic binary match) ✓
- Cache key format "flags:pattern" ✓
- Invalid regex pattern returns error (engine skips rule, logs warning) ✓
- getOrCompile(pattern, flags) signature matches design doc class diagram ✓
- Reasoning format "Matched regex pattern: <pattern>" matches design doc table ✓

### Design Doc Divergences

Implementation adds capabilities beyond the minimal design doc spec, all mirroring the keyword matcher's architecture:

| Divergence | Design Doc | Implementation | Assessment |
| --- | --- | --- | --- |
| Cache eviction | Not specified | Sliding TTL with background cleanup goroutine (reuses patternTTL/patternCleanupInterval from keyword.go) | Justified: consistent with keyword matcher; prevents unbounded cache growth |
| Lifecycle management | Not specified | Close() method for goroutine shutdown, injectable clock for testing | Justified: consistent with keyword matcher; enables deterministic testing |
| Test constructors | Not specified | newRegexMatcherForTest with custom TTL and clock, export_test.go helpers | Justified: consistent with keyword matcher test infrastructure |

## Findings

### HIGH Priority

| Source | Finding | Resolution |
| ------ | ------- | ---------- |
| Codex | Case-sensitive regex broken by prompt normalization (engine.go:58, engine.go:85) - Engine calls `NormalizePrompt()` which lowercases the prompt before passing it to all matchers. The regex matcher receives a pre-lowered prompt, so case-sensitive patterns (e.g., `\bGo\b` with no `i` flag) can never match uppercase characters. The design doc's `i` flag for case-insensitive matching becomes effectively meaningless since all input is already lowercase. This is a silent correctness bug — patterns fail without error or warning. Root cause is a Phase 9 design decision (normalize once for all matchers) that conflicts with the Phase 11 regex matcher's case-sensitivity feature. | FIXED: Moved lowercasing out of `NormalizePrompt()` (now trim-only) and into `KeywordMatcher.Match()`. Regex matcher receives original-case prompt; `(?i)` flag now works correctly. Updated routing.go, keyword.go, regex.go (comments), routing_test.go, engine_test.go, regex_test.go (added case-sensitive test cases). |

### MEDIUM Priority

| Source | Finding | Resolution |
| ------ | ------- | ---------- |
| all 3 agents | TOCTOU in getOrCompile (regex.go:148-156) - No re-check under write lock before insert. Two concurrent goroutines with the same uncached pattern both compile it; second overwrites first. Functionally safe (idempotent overwrite, no data race) but wastes CPU on duplicate compilation. Exact same pattern exists in keyword.go:209-218 (getOrCompilePattern). | DISMISSED: Not a bug, consistent with existing patterns in MatcherRegistry, RuleCache, and KeywordMatcher (dismissed in Phase 10 review). |
| code-review-agent, go-architect-agent | Shared constants coupling (keyword.go:14-21 used by regex.go:45,49) - patternTTL and patternCleanupInterval defined in keyword.go and reused by regex.go. Creates implicit dependency: modifying these constants affects both matchers. Acceptable because both files are in the same package with identical caching semantics. | DISMISSED: All matchers use them for the same purpose. Same package, same caching behavior by design. |

### LOW Priority

| Source | Finding | Resolution |
| ------ | ------- | ---------- |
| code-review-agent | nolint directive linter name (regex.go:57, keyword.go:61) - `//nolint:unusedfunc` may not match actual golangci-lint linter name. The standard name is `unused` from staticcheck. golangci-lint silently ignores unknown linter names, so the directive may not suppress anything. Functions are referenced via export_test.go so may not actually need suppression. | DISMISSED: No harm — golangci-lint silently ignores unknown linter names. Directive is a no-op. |
| code-review-agent | Missing test for unsupported flag character - applyRegexFlags silently ignores unknown flags (e.g., flags: "x"). No test documents this expected behavior. Could lead to hard-to-diagnose issues. | FIXED: Changed `applyRegexFlags` to return `(string, error)`, rejecting unknown flags with clear error message (e.g., `unsupported regex flag "x"`). Added 4 test cases covering unsupported flags in various positions. |
| go-architect-agent | Mixed benchmark iteration styles (regex_benchmark_test.go) - File uses both b.Loop() (lines 28, 68, 89, 144) and range b.N (lines 43, 120). Split is justified: range b.N used where loop index is needed for unique pattern generation. | DISMISSED: Functionally correct, justified by index usage requirements. |
| software-architect-agent | Close() not in RuleMatcher interface - Both KeywordMatcher and RegexMatcher have Close() for goroutine shutdown, but RuleMatcher interface does not include it. MatcherRegistry has no CloseAll(). Callers must know concrete type to call Close(). | FIXED: Added `Close()` to `RuleMatcher` interface, `CloseAll()` to `MatcherRegistry`, and `Close()` to mock. Tests added for CloseAll (including idempotency and empty registry). |
| go-architect-agent | cleanExpiredRegexPatterns method name includes redundant "Regex" qualifier since receiver is already *RegexMatcher. However, name is used in export_test.go (ExportCleanExpiredRegexPatterns) where the qualifier aids readability. | DISMISSED: Acceptable trade-off for export naming clarity. |

## Good Patterns Observed

- **Structural symmetry with KeywordMatcher** - Both matchers are structurally identical, making the codebase predictable and maintainable
- **Injectable clock (nowFunc)** enables deterministic TTL testing without time.Sleep
- **Comprehensive test coverage** - 100% on all regex.go functions, including context cancellation, concurrent access, cache eviction, sliding TTL refresh, and design doc example pattern
- **Proper export_test.go pattern** for white-box testing from external test package
- **Seamless engine integration** - Zero changes needed to engine, registry, or type system
- **Background cleanup goroutine** reuses constants from keyword.go, ensuring consistent cache behavior
- **Idempotent Close()** via select on done channel prevents double-close panics
- **Benchmark suite** covers cache hit, cache miss, simple/complex patterns, multiple patterns, no-match, and case sensitivity comparison

## Patterns to Document

1. **RuleMatcher implementation pattern** - Both keyword.go and regex.go demonstrate a repeatable pattern: struct with sync.RWMutex, compiled-pattern cache with sliding TTL, background cleanup goroutine, injectable clock, idempotent Close, test helper via export_test.go. Should be documented in `agents/patterns/go-patterns/rule-matcher-implementation-pattern.md` before Phase 12 (pattern matcher) and Phase 13 (default matcher).

## Notes for Future Phases

**Phase 12** (Pattern Matcher): Will implement PatternMatcher with vector similarity. Should follow the RuleMatcher implementation pattern. May not need TTL cache (embeddings are stored, not compiled). Will consume RequestContext and Options fields.

**Phase 13** (Default Matcher): Simple always-match implementation. Should follow RuleMatcher interface but likely does not need caching or background goroutine.

**Phase 14** (Rule Cache): In-memory cache already exists from Phase 9. Phase 14 adds configuration for cache behavior per routing.cache config section.

**Phase 16** (Route Endpoint): Will wire RegexMatcher into MatcherRegistry during server initialization via registry.Register(routing.NewRegexMatcher()).
