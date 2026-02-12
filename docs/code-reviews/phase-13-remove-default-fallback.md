# Code Review: Phase 13 - Remove Default Fallback

**Review Date:** 2026-02-12
**Reviewers:** code-review-agent, software-architect-agent, go-architect-agent
**Phase:** 13 (Remove default fallback, implement explicit no-match signaling)

## Design Change

Phase 13 was originally scoped as "Implement default matcher (fallback routing)." The DefaultMatcher was implemented and passed code review, but finding H1 in that review (see [`phase-13-default-matcher.md`](phase-13-default-matcher.md)) identified a dual-path default fallback that prompted a design discussion. The conclusion: "A default match is no match." Phase 13 was redefined to remove the default fallback entirely and implement explicit no-match signaling.

- **Design change assessment:** [`docs/architecture/design-change-remove-default-matcher.md`](../architecture/design-change-remove-default-matcher.md)
- **Implementation plan:** [`docs/plans/phase-13-implementation-plan.md`](../plans/phase-13-implementation-plan.md)
- **ADR:** [ADR-007: Explicit No-Match Signaling](../architecture/02-architectural-decisions.md#adr-007-explicit-no-match-signaling)

## Files Reviewed

### Source Files

**Modified:**

- `src/mnemonic/internal/routing/routing.go` - Added Matched bool to Decision, removed MatchTypeDefault re-export
- `src/mnemonic/internal/routing/engine.go` - Removed defaultAgent, replaced fallback with no-match, added recordNoMatch
- `src/mnemonic/internal/routing/matcher.go` - Doc comment update
- `src/mnemonic/internal/routing/cache.go` - Comment update
- `src/mnemonic/internal/repository/routingrule/routingrule.go` - Removed MatchTypeDefault, DefaultMatchConfig
- `src/mnemonic/internal/config/config.go` - Removed DefaultAgent field and validation
- `src/mnemonic/internal/config/defaults.go` - Removed DefaultRoutingDefaultAgent constant
- `api/openapi/mnemonic-v1.yaml` - Added matched field, removed DefaultMatchConfig, MatchMethodDefault

**Deleted:**

- `src/mnemonic/internal/routing/default_matcher.go`
- `src/mnemonic/internal/routing/default_matcher_test.go`

### Test Files

- `src/mnemonic/internal/routing/engine_test.go`
- `src/mnemonic/internal/routing/routing_test.go`
- `src/mnemonic/internal/routing/matcher_test.go`
- `src/mnemonic/internal/repository/routingrule/repository_test.go`
- `src/mnemonic/internal/config/config_test.go`
- `src/mnemonic/internal/telemetry/telemetry_test.go`
- `src/mnemonic/tests/e2e/routing_test.go`

### Documentation Files

- `docs/design/routing-engine.md`
- `docs/design/configuration.md`
- `docs/architecture/02-architectural-decisions.md`
- `docs/architecture/08-data-architecture.md`
- `docs/plans/mvp-implementation-plan.md`
- `docs/architecture/design-change-remove-default-matcher.md` - NEW
- `docs/plans/phase-13-implementation-plan.md` - NEW

## Validation Results

| Tool          | Result             |
| ------------- | ------------------ |
| go build      | PASS               |
| go test       | PASS               |
| go test -race | PASS               |
| go vet        | PASS               |
| goimports     | PASS               |
| golangci-lint | PASS (0 issues)    |
| govulncheck   | no vulnerabilities |
| gosec         | 0 issues           |
| OpenAPI lint  | valid              |
| Test coverage | 94.8% (routing)    |

## Design Compliance

Implementation satisfies the design change assessment at `docs/architecture/design-change-remove-default-matcher.md`. The "no match = no match, client decides" principle is consistently applied across domain, API contract, and documentation.

### Behavioral Requirements Verified

- Engine returns Decision{Matched: false}, nil when no rules match ✓
- Decision struct has Matched bool as first field ✓
- Zero-value Decision{} signals no-match ✓
- MatchTypeDefault, DefaultMatchConfig removed from type system ✓
- defaultAgent removed from Engine and NewEngine ✓
- routing.default_agent config removed ✓
- OpenAPI matched field is the only required field in RoutingDecision ✓
- HTTP 200 OK returned for both match and no-match ✓
- ADR-007 documents the design decision ✓

### Design Doc Divergences (Post-Review)

None intentional. The findings below identify residual references that were missed during doc updates.

## Findings

### HIGH Priority

| ID  | Source                                                        | Finding                                                                                                                                               | Resolution                                                                                                                                   |
| --- | ------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| H1  | code-review-agent                                             | `docs/design/routing-engine.md:63` - Design principle still says "Fail-safe defaults: If no rules match, a default agent handles the request..."     | FIX: Replace with "Explicit no-match signaling: If no rules match, the engine returns Decision{Matched: false} and lets client decide"      |
| H2  | code-review-agent                                             | `docs/design/routing-engine.md:119` - Evaluator description says "returns a default routing decision using the configured default agent"             | FIX: Replace with "returns Decision{Matched: false}, nil to signal no routing rule matched"                                                 |
| H3  | software-architect-agent, code-review-agent                   | `docs/design/routing-engine.md:177-186` - RuleMatcher class diagram still includes DefaultMatcher class and implements relationship                  | FIX: Remove DefaultMatcher from class diagram                                                                                                |
| H4  | code-review-agent                                             | `docs/design/configuration.md:571` - Mermaid diagram shows `+string DefaultAgent` field in MnemonicRoutingConfig                                      | FIX: Remove DefaultAgent field from diagram                                                                                                  |

### MEDIUM Priority

| ID  | Source                                                 | Finding                                                                                                                                                                                    | Resolution                                                                                                                                                                                                           |
| --- | ------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| M1  | software-architect-agent                               | `src/mnemonic/tests/e2e/routing_rules_test.go` lines 16, 24, 180-194 - Still references `match_type: default` and contains `TestCreateRoutingRule_DefaultMatch` stub                      | FIX: Update stub comments; remove default match_type references                                                                                                                                                      |
| M2  | go-architect-agent, software-architect-agent, code-review-agent | `src/mnemonic/tests/e2e/routing_test.go` - `TestRoute_DefaultMatch` function name misleading; semantically overlaps with `TestRoute_NoMatchNoDefault`                                     | FIX: Rename TestRoute_DefaultMatch to TestRoute_NoMatch; rename TestRoute_NoMatchNoDefault to TestRoute_UnmatchedPrompt or remove if redundant                                                                      |
| M3  | go-architect-agent, software-architect-agent           | `src/mnemonic/internal/routing/engine.go:143-148` - `recordNoMatch` reuses `RecordRuleMatch(ctx, "no_match")`, conflating matches and non-matches in the same metric; doesn't increment routing decisions counter | DEFER to Phase 15: Add dedicated RecordNoMatch method to metrics.Routing with its own counter. Document whether routing.decisions counts all evaluations or only matches.                                          |

### LOW Priority

| ID  | Source                   | Finding                                                                                                                                  | Resolution                                                                                                     |
| --- | ------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------- |
| L1  | software-architect-agent | `docs/function-documentation-map.md:243` - Still maps `(DefaultMatchConfig) Type()` to deleted Default Matcher section                  | FIX: Remove entry                                                                                              |
| L2  | code-review-agent        | `docs/design/routing-engine.md:123` - "keyword, regex, pattern, and default matching"                                                   | FIX: Remove "and default"                                                                                      |
| L3  | go-architect-agent       | `src/mnemonic/internal/routing/engine.go` - Span attribute `routing.matched` only set on no-match path, not match path                  | FIX: Add `attribute.Bool("routing.matched", true)` to match-path span attributes                              |
| L4  | go-architect-agent       | `src/mnemonic/internal/routing/engine_test.go:558` - `TestEngine_Route_ContextCancellation` uses full-struct equality `Decision{}`      | DEFER: Acceptable for now; could switch to field-level assertions in Phase 15                                 |
| L5  | code-review-agent        | `src/mnemonic/internal/routing/engine.go:147` - `"no_match"` string literal could be a constant                                         | DEFER: Minor; will be addressed when M3 introduces dedicated RecordNoMatch method                             |

## Patterns to Document

Patterns identified that should be added to the patterns and examples for Claude Code's sub agents.

1. **Explicit State Signaling with Bool Fields** - Using `Matched bool` as the first field on result structs to explicitly signal success vs. no-match, superior to zero-value sentinels or sentinel errors. Zero-value semantics (`Decision{}` has `Matched: false`) make it safe by default.

## Notes for Future Phases

**Phase 15** (Routing Unit Tests): Add dedicated `RecordNoMatch` metric method (M3). Consider switching context cancellation test to field-level assertions (L4).

**Phase 16** (Route Endpoint): Handler maps `Decision.Matched == false` to 200 OK with `matched: false`. No `defaultAgent` wiring needed.

**Post-MVP**: Database migration to remove `'default'` from routing_rules CHECK constraint. Go validation already rejects new default-type rules.
