# Code Review: Phase 9 - Routing Engine

**Review Date:** 2026-02-09
**Reviewers:** code-review-agent, software-architect-agent, user
**Phase:** 9 (Implement Routing Engine Core Package)

## Files Reviewed

### Source Files

- `src/mnemonic/internal/routing/routing.go` - Core types, Evaluator interface, NormalizePrompt, NormalizeConfidence
- `src/mnemonic/internal/routing/matcher.go` - RuleMatcher interface, MatcherRegistry
- `src/mnemonic/internal/routing/cache.go` - RuleLoader interface, RuleCache
- `src/mnemonic/internal/routing/engine.go` - Engine struct implementing Evaluator, Route method, buildReasoning
- `src/mnemonic/internal/routing/errors.go` - Sentinel errors (deleted during review)

### Test Files

- `src/mnemonic/internal/routing/routing_test.go`
- `src/mnemonic/internal/routing/matcher_test.go`
- `src/mnemonic/internal/routing/cache_test.go`
- `src/mnemonic/internal/routing/engine_test.go`
- `src/mnemonic/internal/routing/mock_test.go`

## Validation Results

| Tool | Result |
|------|--------|
| goimports | Clean |
| go vet | No issues |
| go test -race -v ./internal/routing/... | 15 tests, all PASS, no races |
| go test ./... | All module tests pass (no regressions) |
| govulncheck | No vulnerabilities |
| gosec | 0 issues |

## Design Compliance

Implementation satisfies all Phase 9 behavioral requirements from the routing engine design doc (docs/design/routing-engine.md).

### Behavioral Requirements Verified

- Evaluator interface with Route(ctx, req) (decision, error) ✓
- Request with Prompt, RequestContext, Options ✓
- Decision with AgentName, Confidence, MatchType, Keywords, Reasoning ✓
- RuleMatcher interface with Match and Type ✓
- MatcherRegistry for type-to-matcher mapping ✓
- RuleCache with RWMutex, pre-sorted, copy on read ✓
- Priority DESC, ID ASC sorting ✓
- Fail-fast on startup load failure ✓
- Short-circuit on first match ✓
- Skip disabled rules, unknown matcher types, matcher errors (log warning) ✓
- Default decision when no rules match ✓
- Confidence normalization [0.0, 1.0] ✓
- Reasoning generation per match type ✓
- OpenTelemetry tracing ✓
- Nil-safe metrics recording ✓
- Prompt normalization (lowercase + trim) ✓

### Design Doc Divergences (Post-Review)

Code review fixes introduced divergences between the implementation and the design/architecture docs. A compliance audit identified the following. All divergences are justified improvements — the design docs are being updated to match the implementation.

#### Naming Divergences (7 renames applied to code, docs updated to match)

| Old Name (in design docs) | New Name (in implementation) | Reason |
| --- | --- | --- |
| `RoutingDecision` | `Decision` | Go stutter fix |
| `RouteRequest` | `Request` | Go stutter fix |
| `RouteContext` | `RequestContext` | Go stutter fix |
| `RouteOptions` | `Options` | Go stutter fix |
| `Router` (interface) | `Evaluator` | Go stutter fix |
| `RecordPatternMatch` / `mnemonic.routing.pattern_matches` | `RecordRuleMatch` / `mnemonic.routing.rule_matches` | Misleading name fix |

#### Structural Divergences (justified improvements over design doc)

| Divergence | Design Doc | Implementation | Assessment |
| --- | --- | --- | --- |
| MatchConfig type | Union struct with pointer fields | Interface with concrete types implementing `Type() string` | Better Go idiom; supports type-safe dispatch |
| RuleCache storage | `[]Rule` (value type) | `[]*routingrule.RoutingRule` (pointer type) | Reuses repository type directly; avoids translation layer |
| GetRules() return | Direct slice reference | Shallow copy via `make + copy` | Prevents external mutation of cache internals |
| RuleLoader interface | Implicit `Load()` with internal `context.Background()` | `LoadRules(ctx context.Context)` with caller-provided ctx | Better context propagation for cancellation/tracing |
| NewRuleCache signature | `NewRuleCache(loader)` | `NewRuleCache(ctx, loader)` | Explicit context passing |
| Engine dependencies | RuleCache + MatcherRegistry only | Adds defaultAgent, metrics, logger, tracer | Operational concerns for production use |
| RuleRepository | Referenced in design diagrams | Replaced by RuleLoader interface | Simpler abstraction for rule loading |

#### Documents Updated

| Document | Scope | Status |
| --- | --- | --- |
| `docs/design/routing-engine.md` | Major — 50+ naming occurrences, structural snippets, Mermaid diagrams | Updated |
| `docs/design/observability-implementation.md` | Medium — ~9 metric/method name occurrences | Updated |
| `docs/plans/mvp-implementation-plan.md` | Minor — broken `#routing-algorithm` anchor link | Updated |

## Findings

### HIGH Priority

No HIGH priority findings.

### MEDIUM Priority

| Source | Finding | Resolution |
| ------ | ------- | ---------- |
| both agents | Unused Sentinel Errors - ErrNoRulesLoaded and ErrUnknownMatcher declared but never referenced. Engine handles these cases by logging warnings and falling through to default, which is correct per "never fail to route" principle. | FIXED: Deleted errors.go. Dead code removed; can be reintroduced if needed. |
| both agents | Shallow Copy Fragility in GetRules() - Returns shallow copy via copy(result, c.rules). Safe today because all MatchConfig types are value types. Would break if any MatchConfig changes to pointer receiver or stores pointer fields. | DISMISSED: Safe as-is. All MatchConfig implementations are value types (structs without pointer fields). |
| both agents | Context Cancellation Falls Through to Default - Route method doesn't check ctx.Err() between rule evaluations. Cancelled requests get "valid" routing decision rather than error. Low impact for MVP keyword/regex matchers (sub-millisecond). Could waste resources when pattern matcher (with embedding API calls) is added. | FIXED: Added ctx.Err() check at top of rule evaluation loop in engine.go. Route now returns zero Decision and context error on cancellation. Test updated to verify. |
| both agents | MatchType Duplication Between Packages - routing package defines MatchType as typed string; routingrule package defines ValidMatchTypes as []string. Engine bridges with unchecked string cast. No compile-time linkage between parallel definitions. | FIXED: MatchType type and constants now defined once in routingrule package (source of truth). routing package uses type alias (type MatchType = routingrule.MatchType) and re-exports constants. ValidMatchTypes derived from typed constants. |
| software-architect-agent | RuleLoader Pointer/Value Mismatch with Repository - RuleLoader returns []routingrule.RoutingRule (values); routingrule.Repository.ListEnabled() returns []*RoutingRule (pointers). Adapter needed at Phase 16 server wiring. RuleLoaderFunc adapter type makes this straightforward. | FIXED: RuleLoader interface, RuleCache, and all tests updated to use []*routingrule.RoutingRule, matching Repository.ListEnabled() directly. No adapter needed at Phase 16. |
| **user** | RuleLoaderFunc YAGNI - `RuleLoaderFunc` adapter in `src/mnemonic/internal/routing/cache.go` adds 8 lines solving a problem that hasn't materialized. Tests use `mockRuleLoader` structs instead. Phase 16 wiring is the only consumer and a direct adapter struct is equally clear. YAGNI score: 3/10. Remove and add back if/when needed. | FIXED: Removed RuleLoaderFunc type and its two associated tests from cache_test.go. |

### LOW Priority

| Source | Finding | Resolution |
| ------ | ------- | ---------- |
| code-review-agent | RequestContext and Options Unused - Defined in Request but not consumed by Engine.Route(). Expected for Phase 9. Pattern matching (Phase 12) and per-request options (Phase 16 handler) will consume these fields. Establishes API contract early. | DISMISSED: By-design for Phase 9 scope. Fields establish API contract for Phases 12 and 16. |
| code-review-agent | RecordPatternMatch Naming Misleading - Method called for every routing decision regardless of match type. Metric name "mnemonic.routing.pattern_matches" with description "by rule type" is workable but method name suggests pattern-only recording. Naming concern in existing metrics package, not routing engine. | FIXED: Renamed to `RecordRuleMatch` with metric `mnemonic.routing.rule_matches`. Updated in metrics/routing.go, metrics/routing_test.go, routing/engine.go, and telemetry/telemetry_test.go. |
| code-review-agent | No Benchmark Tests for Latency SLOs - Design doc specifies benchmark targets (100 rules keyword match < 1ms, full scan < 5ms, pattern match < 500ms). No Benchmark* functions exist. | DISMISSED: Benchmarks are meaningful only after matchers exist. Add alongside Phases 10-12 when real workloads can be measured. |
| software-architect-agent | No Routing Latency Histogram Metric - RoutingMetrics provides counters but no histogram for routing latency. Tiered latency SLOs cannot be measured via Prometheus metrics. OTel spans provide some data but not Prometheus-style SLO dashboards. | DISMISSED: Latency histograms are meaningless without real matchers. Add alongside Phases 10-12 when latency SLOs become measurable. |
| software-architect-agent | MatcherRegistry Comment Understates Thread Safety - Comment says "safe for concurrent read access after all matchers registered" but implementation uses sync.RWMutex for both Register and GetMatcher, making it safe for concurrent registration. Code more correct than comment. | FIXED: Updated comment to "safe for concurrent registration and lookup via sync.RWMutex". |
| **user** | Go Naming Stutter Violations - Full stutter audit of all exported identifiers. `RoutingDecision` → `routing.RoutingDecision`, `RouteRequest` → `routing.RouteRequest`, `RouteContext` → `routing.RouteContext`, `RouteOptions` → `routing.RouteOptions`, `Router` → `routing.Router` all stutter when package-qualified. Go naming convention recommends avoiding package name repetition in type names. | FIXED: Renamed `RoutingDecision` → `Decision`, `RouteRequest` → `Request`, `RouteContext` → `RequestContext`, `RouteOptions` → `Options`, `Router` → `Evaluator`. All references updated across routing.go, engine.go, and engine_test.go. |

## Good Patterns Observed

- **"Never fail to route" principle** faithfully implemented - unknown matcher types, matcher errors, and empty rule sets gracefully fall through to default
- **Nil-safe metrics** via guard function prevents nil pointer panics when metrics not configured
- **Short-circuit evaluation** implemented correctly, verified with atomic.Bool test proving second matcher never called
- **Defensive disabled-rule skip** adds safety net even though cache should only contain enabled rules
- **Pre-sorted cache** avoids sorting per request, consistent with design doc optimization strategies
- **Thread-safe MatcherRegistry** with RWMutex - write lock for registration, read lock for lookups
- **Hand-rolled mocks** - minimal, focused, no framework dependencies, consistent with project testing patterns
- **OTel tracing span** created at Route entry with attributes for both match and default paths

## Patterns to Document

1. **Nil-safe optional dependencies** - Accept pointer dependency (like *metrics.RoutingMetrics) that can be nil, with nil-check guard. Reusable pattern for optional metrics/loggers across the project.

## Notes for Future Phases

**Phase 10** (Keyword Matcher): Will implement KeywordMatcher referenced in design doc. Register with MatcherRegistry during initialization.

**Phase 11** (Regex Matcher): Will implement RegexMatcher with compiled pattern caching. Register with MatcherRegistry during initialization.

**Phase 12** (Pattern Matcher): Will consume RequestContext and Options fields. Context cancellation check (ctx.Err()) already in place at top of rule evaluation loop.

**Phase 16** (HTTP Handlers): Will wire RuleCache to routingrule.Repository. RuleLoader interface already accepts []*RoutingRule matching Repository.ListEnabled() directly.
