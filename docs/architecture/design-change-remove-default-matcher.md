# Design Change Assessment: Remove DefaultMatcher and Engine Hardcoded Fallback

**Date:** 2026-02-12
**Status:** Proposed
**Scope:** Routing engine fallback behavior
**Triggered by:** Phase 13 code review finding H1 (dual-path default fallback)

## Table of Contents

- [Design Decision](#design-decision)
- [Decision Struct Signaling Analysis](#decision-struct-signaling-analysis)
- [Ripple Effect Analysis](#ripple-effect-analysis)
  - [Code Changes](#code-changes)
  - [Configuration Changes](#configuration-changes)
  - [Database Schema Impact](#database-schema-impact)
  - [API Contract Impact](#api-contract-impact)
  - [Design Document Updates](#design-document-updates)
  - [MVP Plan Impact](#mvp-plan-impact)
- [Edge Cases](#edge-cases)
- [Migration Strategy](#migration-strategy)

---

## Design Decision

**"A default match is no match."**

When no routing rules match a prompt, the engine should return an explicit "no match" signal rather than silently routing to a hardcoded default agent. The client decides what to do with an unmatched prompt: route to a general agent, ask the user to rephrase, or take any other action.

This removes two fallback paths that exist today:

1. **Path A (engine hardcoded fallback):** `engine.go` lines 122-142 construct a `Decision` with `e.defaultAgent` when no rules match.
2. **Path B (DefaultMatcher):** A `RuleMatcher` implementation that always returns `Matched: true` with confidence 0.5, registered as match type `"default"`.

Both paths are eliminated. The engine's responsibility ends at evaluating rules. The client's responsibility begins at interpreting the result.

---

## Decision Struct Signaling Analysis

Four options were evaluated for communicating "no match" to the caller.

### Option 1: `Matched bool` field on Decision (Recommended)

Add a `Matched bool` field to the `Decision` struct. When no rules match, the engine returns `Decision{Matched: false}, nil`. When a rule matches, `Matched` is `true` and all other fields are populated.

**Pros:**

- Explicit and unambiguous. Callers must check `Matched` before using `AgentName`.
- Consistent with `MatchResult.Matched` already used by matchers. Same pattern at both levels.
- Zero-value of `Decision{}` has `Matched: false`, so an uninitialized Decision is safe by default.
- No sentinel values, no special error types, no magic strings.
- Backward-compatible in Go: new field with zero-value `false` does not break existing struct literals that use field names.

**Cons:**

- Callers who forget to check `Matched` will see empty `AgentName`, `Confidence: 0`, etc. This is detectable but not enforced at compile time.

### Option 2: Sentinel error (e.g., `ErrNoMatch`)

Return a defined error like `var ErrNoMatch = errors.New("no routing rules matched")`.

**Pros:**

- Callers must handle the error path, making it harder to ignore.
- Standard Go error-handling pattern.

**Cons:**

- Conflates "nothing matched" (a valid, expected outcome) with actual errors (context cancellation, broken matchers). No-match is not an error; it is one of two normal outcomes.
- Callers must use `errors.Is(err, ErrNoMatch)` to distinguish from real errors. Forgetting this produces incorrect error handling.
- Breaks the current contract where `(Decision{}, error)` means "something went wrong" and `(Decision, nil)` means "here is your result."

### Option 3: Zero-value Decision as sentinel

Return `Decision{}, nil` and let callers infer "no match" from the empty `AgentName`.

**Pros:**

- No struct changes needed.

**Cons:**

- Implicit. An empty `AgentName` could also indicate a bug (misconfigured rule with empty agent name).
- Relies on callers knowing that `AgentName == ""` means no match. Not self-documenting.
- Fragile: if any field defaults change, the sentinel semantics break.

### Option 4: HTTP status code differentiation

Return 200 for match, 404 (or 204) for no-match, with the Decision struct unchanged.

**Pros:**

- Standard REST semantics. Clients already handle status codes.

**Cons:**

- Couples the routing engine's internal contract to HTTP. The engine is a domain service; it should not know about HTTP.
- The handler layer can derive the HTTP status from `Decision.Matched`, so this is better handled one layer up.
- Does not solve the Go-level API problem. Internal callers (tests, service layer) still need a way to distinguish match from no-match.

### Recommendation

**Option 1: `Matched bool` field.** It is explicit, consistent with the existing `MatchResult` pattern, safe by zero-value, and keeps "no match" out of the error channel where it does not belong. The HTTP handler can trivially map `Matched: false` to the appropriate status code.

---

## Ripple Effect Analysis

### Code Changes

#### Files to Delete

| File                                                    | Reason                                |
| ------------------------------------------------------- | ------------------------------------- |
| `src/mnemonic/internal/routing/default_matcher.go`      | DefaultMatcher implementation removed |
| `src/mnemonic/internal/routing/default_matcher_test.go` | Tests for removed implementation      |

#### Files to Modify

**1. `src/mnemonic/internal/routing/routing.go`**

| Change                                          | Lines                     | Detail                                                                |
| ----------------------------------------------- | ------------------------- | --------------------------------------------------------------------- |
| Add `Matched bool` to `Decision`                | 77-93                     | New field, first in struct (prominent position)                       |
| Remove `MatchTypeDefault` re-export             | 20                        | Remove the `MatchTypeDefault = routingrule.MatchTypeDefault` constant |
| Remove `MatchTypeDefault` from `buildReasoning` | (in engine.go, see below) | Case removed                                                          |
| Update `Evaluator` doc comment                  | 96-101                    | Remove "or a default decision" language                               |

Proposed `Decision` struct:

```go
type Decision struct {
    // Matched indicates whether a routing rule matched the prompt.
    // When false, all other fields are zero-valued and should not be used.
    Matched bool

    // AgentName is the identifier of the selected agent.
    AgentName string

    // Confidence is the routing confidence from 0.0 to 1.0.
    Confidence float64

    // MatchType indicates which type of matching triggered the route.
    MatchType MatchType

    // MatchedKeywords contains keywords that triggered the route.
    // Only populated for MatchTypeKeyword.
    MatchedKeywords []string

    // Reasoning is a human-readable explanation of why this agent was selected.
    Reasoning string
}
```

**2. `src/mnemonic/internal/routing/engine.go`**

| Change                                                | Lines        | Detail                                                                              |
| ----------------------------------------------------- | ------------ | ----------------------------------------------------------------------------------- |
| Remove `defaultAgent` field from `Engine` struct      | 23           | Field deleted                                                                       |
| Remove `defaultAgent` parameter from `NewEngine`      | 31-46        | Parameter and assignment removed                                                    |
| Replace hardcoded fallback block with no-match return | 122-142      | Return `Decision{Matched: false}, nil` with appropriate logging and span attributes |
| Set `Matched: true` on successful match               | 95-101       | Add `Matched: true` to the Decision literal                                         |
| Remove `MatchTypeDefault` case from `buildReasoning`  | 169-170      | Case deleted                                                                        |
| Update Engine and Route doc comments                  | 17-19, 48-49 | Remove "or a default decision" language                                             |

Proposed no-match block (replaces lines 122-142):

```go
// No rules matched.
span.SetAttributes(
    attribute.Bool("routing.matched", false),
)

e.recordNoMatch(ctx)

e.logger.Debug().
    Msg("no rules matched")

return Decision{}, nil
```

Note: `Decision{}` has `Matched: false` by zero-value, which is the desired signal.

**3. `src/mnemonic/internal/routing/engine_test.go`**

| Change                                                             | Detail                                                          |
| ------------------------------------------------------------------ | --------------------------------------------------------------- |
| Update `newTestEngine` helper                                      | Remove `defaultAgent` parameter, update `NewEngine` call        |
| Update "falls through to default" test case                        | Assert `Matched == false`, empty `AgentName`, zero `Confidence` |
| Update "no rules at all" test case                                 | Same as above                                                   |
| Update all matching test cases                                     | Assert `Matched == true`                                        |
| Remove `wantMatchType: routing.MatchTypeDefault` assertions        | No more default match type in decisions                         |
| Remove `wantReasonContain: "No specific rules matched"` assertions | No reasoning for no-match                                       |

**4. `src/mnemonic/internal/routing/routing_test.go`**

| Change                              | Lines | Detail                                                                                    |
| ----------------------------------- | ----- | ----------------------------------------------------------------------------------------- |
| Remove `MatchTypeDefault` assertion | 67    | Remove the line `assert.Equal(t, routing.MatchType("default"), routing.MatchTypeDefault)` |

**5. `src/mnemonic/internal/routing/matcher_test.go`**

| Change                                        | Lines | Detail                                                                                                                                                                                                                              |
| --------------------------------------------- | ----- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Remove or update `MatchTypeDefault` assertion | 90    | Line `assert.Nil(t, registry.GetMatcher(routing.MatchTypeDefault))` can be removed or left as-is (it tests GetMatcher returns nil for unregistered types, which remains valid behavior if the constant still exists in routingrule) |

**6. `src/mnemonic/internal/routing/matcher.go`**

| Change                           | Detail                                                                                                       |
| -------------------------------- | ------------------------------------------------------------------------------------------------------------ |
| Update `RuleMatcher` doc comment | Remove "default" from the list: "Each concrete matcher (keyword, regex, pattern) implements this interface." |

**7. `src/mnemonic/internal/repository/routingrule/routingrule.go`**

| Change                                              | Lines   | Detail                                          |
| --------------------------------------------------- | ------- | ----------------------------------------------- |
| Remove `MatchTypeDefault` constant                  | 21      | Delete `MatchTypeDefault MatchType = "default"` |
| Remove `"default"` from `ValidMatchTypes`           | 61      | Remove `string(MatchTypeDefault)` entry         |
| Remove `DefaultMatchConfig` struct and method       | 131-136 | Delete struct and `Type()` method               |
| Remove `"default"` case from `UnmarshalMatchConfig` | 162-163 | Delete case                                     |

**8. `src/mnemonic/internal/config/config.go`**

| Change                                           | Lines   | Detail                     |
| ------------------------------------------------ | ------- | -------------------------- |
| Remove `DefaultAgent` field from `RoutingConfig` | 100     | Delete field               |
| Remove `routing.default_agent` default value     | 299     | Delete `v.SetDefault(...)` |
| Remove `DefaultAgent` validation                 | 634-639 | Delete validation block    |

**9. `src/mnemonic/internal/config/defaults.go`**

| Change                                       | Lines | Detail          |
| -------------------------------------------- | ----- | --------------- |
| Remove `DefaultRoutingDefaultAgent` constant | 60    | Delete constant |

**10. `src/mnemonic/internal/config/config_test.go`**

| Change                                                 | Detail                                                                                 |
| ------------------------------------------------------ | -------------------------------------------------------------------------------------- |
| Remove all `DefaultAgent` / `default_agent` assertions | Multiple locations (lines 73, 140, 182, 208, 243, 666-670, 964, 972, 1246, 1252, 1449) |
| Update `validConfig()` helper                          | Remove `DefaultAgent` field                                                            |
| Remove "empty default_agent" validation test case      | Lines 666-671                                                                          |

**11. `src/mnemonic/internal/routing/engine.go` `recordMetrics` method**

| Change                                               | Detail                                                                                                                                                                       |
| ---------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Add `recordNoMatch` method or update `recordMetrics` | Record a "no_match" metric so operations can track unmatched requests. The metric name `routing.no_match` (counter) gives visibility into how often the system cannot route. |

**12. `src/mnemonic/tests/e2e/routing_test.go`**

| Change                               | Detail                                                                                                     |
| ------------------------------------ | ---------------------------------------------------------------------------------------------------------- |
| Rewrite `TestRoute_DefaultMatch`     | Should assert 200 with `matched: false` (or the appropriate HTTP status the team chooses, see API section) |
| Update `TestRoute_DisabledRules`     | Comment at line 259 references "routes to default instead" -- update expectation                           |
| Update `TestRoute_ReasoningExplains` | Remove "default: indicates fallback" comment                                                               |
| Update `TestRoute_ConfidenceScores`  | Remove "default match" confidence test case                                                                |

### Configuration Changes

The `routing.default_agent` configuration key, its environment variable `MNEMONIC_ROUTING_DEFAULT_AGENT`, and its default value `"general-agent"` are all removed.

**Affected files:**

- `src/mnemonic/internal/config/config.go` -- struct, defaults, validation
- `src/mnemonic/internal/config/defaults.go` -- constant
- `src/mnemonic/internal/config/config_test.go` -- test assertions
- `docs/design/configuration.md` -- documentation table (line 301) and YAML example (line 163)

### Database Schema Impact

The `routing_rules` table allows `match_type = 'default'` in its CHECK constraint (migration `005_create_routing_rules.sql`, line 71). Two options:

**Option A (Recommended for MVP): Leave the schema as-is.** The routing engine will never register a `"default"` matcher, so any existing `default`-type rules will be skipped with a "no matcher registered" warning. A future migration can tighten the constraint. This avoids a migration in the critical path of this change.

**Option B: New migration (008) to remove 'default' from CHECK constraint.** This is cleaner but adds migration complexity. Defer to post-MVP unless the team prefers immediate cleanup.

For MVP, document that `match_type = 'default'` is deprecated in routing rules. The `ValidMatchTypes` slice in Go code will no longer include `"default"`, so the API validation layer will reject new default-type rules once Phase 16 (API endpoints) is built.

### API Contract Impact

The `RoutingDecision` schema in the OpenAPI spec (`api/openapi/mnemonic-v1.yaml`, lines 834-878) needs changes:

**1. Add `matched` field to `RoutingDecision`:**

```yaml
RoutingDecision:
  type: object
  description: Routing decision details
  required:
    - matched
  properties:
    matched:
      type: boolean
      description: Whether a routing rule matched the prompt
    agent_name:
      type: string
      description: Selected agent identifier (present only when matched is true)
    confidence:
      type: number
      format: double
      description: Routing confidence score (0-1, present only when matched is true)
      minimum: 0
      maximum: 1
    method:
      type: string
      description: Routing method used (present only when matched is true)
      enum:
        - MatchMethodKeyword
        - MatchMethodRegex
        - MatchMethodPattern
    matched_keywords:
      type: array
      description: Keywords that triggered the route (for MatchMethodKeyword)
      items:
        type: string
    reasoning:
      type: string
      description: Human-readable routing explanation (present only when matched is true)
```

Key changes:

- `matched` is the only required field.
- `agent_name`, `confidence`, `method`, and `reasoning` move from required to optional (present only when `matched: true`).
- `MatchMethodDefault` is removed from the `method` enum.

**2. Update `RouteResponse`:**

The `agent` field (currently required) should become optional, since there is no agent to return when `matched` is `false`.

```yaml
RouteResponse:
  type: object
  required:
    - routing
  properties:
    routing:
      $ref: "#/components/schemas/RoutingDecision"
    agent:
      $ref: "#/components/schemas/Agent"
      description: Agent details (present only when routing.matched is true)
```

**3. Remove `DefaultMatchConfig` schema** (lines 1025-1028).

**4. Remove `default` from `MatchType` enum** (line 964).

**5. Remove `DefaultMatchConfig` from `RoutingRuleCreate` and `RoutingRuleUpdate` oneOf lists** (lines 1077, 1129).

**6. HTTP status code for no-match:**

The handler should return **200 OK** with `matched: false`. The request was processed successfully; the outcome is "no match." This aligns with the principle that HTTP status codes indicate transport/protocol outcomes, not business logic results. The client inspects `routing.matched` to determine next steps.

Alternative: 204 No Content. This is semantically defensible ("we processed your request but have no agent to offer"), but it prevents returning a response body in some HTTP client libraries, which makes it harder for the client to extract metadata (request ID, timing info). 200 with a clear `matched: false` is simpler and more practical.

### Design Document Updates

**1. `docs/design/routing-engine.md`**

| Section                                             | Lines    | Change                                                                                                                                               |
| --------------------------------------------------- | -------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| Table of Contents                                   | 27       | Remove "Default Matcher" entry                                                                                                                       |
| Default Matcher section                             | 896-927  | Delete entire section                                                                                                                                |
| Confidence Scoring table                            | 942      | Remove `default \| 0.5 \| Baseline for fallback` row                                                                                                 |
| Confidence score diagram                            | 956      | Remove `D[Default<br/>0.5]` node                                                                                                                     |
| Reasoning generation table                          | 990      | Remove `default` row                                                                                                                                 |
| Latency targets table                               | 1014     | Remove `Default \| <1ms \| <5ms \| <10ms` row                                                                                                        |
| Operation-level targets                             | 1020     | Remove "Default" from "Keyword, Regex, Default"                                                                                                      |
| Optimization priority table                         | 1082     | Remove `0 \| default \| Fallback only` row                                                                                                           |
| Error handling table                                | 1108     | Change "All rules fail \| Return default agent" to "All rules fail \| Return no-match decision"                                                      |
| Error handling diagram                              | 1136     | Change `ReturnDefaultDecision` state to `ReturnNoMatch`                                                                                              |
| Error handling key principle                        | 1142     | Rewrite: "The routing engine returns a no-match decision when no rules match or all rules fail. The client decides how to handle unmatched prompts." |
| Interface Definitions - Decision Type               | ~358-364 | Add `Matched bool` to diagram                                                                                                                        |
| Interface Definitions - Complete Type Relationships | 316-319  | Remove `defaultAgent` from Engine class diagram                                                                                                      |
| Match type implementations intro                    | ~300     | Remove `"default"` from the list of MatchConfig implementations                                                                                      |
| Supporting types - MatchConfig semantics            | 296-299  | Remove `MatchType: default -> DefaultMatchConfig` line                                                                                               |
| Class diagram for MatchConfig                       | 280-288  | Remove DefaultMatchConfig class and its implements relationship                                                                                      |

**2. `docs/design/configuration.md`**

| Section             | Lines   | Change                                                |
| ------------------- | ------- | ----------------------------------------------------- |
| Configuration table | 301     | Remove `routing.default_agent` row                    |
| YAML example        | 162-163 | Remove `default_agent: general-agent` and its comment |

**3. `docs/architecture/04-communication-patterns.md`**

| Section                 | Change                                                                                                                            |
| ----------------------- | --------------------------------------------------------------------------------------------------------------------------------- |
| Fallback behavior table | Update if it references default agent routing (current table is about Mnemonic being unreachable, not no-match; no change needed) |

**4. `docs/architecture/08-data-architecture.md`**

| Section                            | Lines   | Change                                                                          |
| ---------------------------------- | ------- | ------------------------------------------------------------------------------- |
| ER diagram `match_type` enum       | 213     | Remove `default` from `"keyword\|regex\|pattern\|default"`                      |
| match_config examples              | 331-332 | Remove `# Default match (fallback)` example                                     |
| SQL CREATE TABLE                   | 408     | Remove `'default'` from CHECK constraint (or add comment that it is deprecated) |
| match_config validation constraint | 929     | Remove `(match_type = 'default')` condition                                     |

**5. `docs/code-reviews/phase-13-default-matcher.md`**

| Change                                                                                                                                                                              |
| ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Add a note at the top: "Superseded: The DefaultMatcher was removed before merge. See `docs/architecture/design-change-remove-default-matcher.md` for the design change assessment." |

**6. `docs/architecture/02-architectural-decisions.md`**

| Change                                                                                       |
| -------------------------------------------------------------------------------------------- |
| Add ADR-007 documenting this design decision (see [Migration Strategy](#migration-strategy)) |

### MVP Plan Impact

**Phase 13 is redefined.** Instead of "Implement default matcher (fallback routing)," Phase 13 becomes:

> **Phase 13: Remove default fallback, implement explicit no-match signaling.**
>
> - Delete DefaultMatcher and its tests.
> - Add `Matched bool` to `Decision` struct.
> - Remove engine hardcoded fallback; return `Decision{Matched: false}, nil`.
> - Remove `defaultAgent` from Engine, `NewEngine`, and configuration.
> - Remove `MatchTypeDefault` constant and `DefaultMatchConfig` from routingrule package.
> - Add `recordNoMatch` metrics path.
> - Update OpenAPI spec: add `matched` field, remove `MatchMethodDefault`, adjust required fields.
> - Update all affected tests.

**Downstream phase impacts:**

| Phase                             | Impact   | Change                                                                                                                                                                   |
| --------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 14 (Rule Cache)                   | None     | Rule cache does not depend on match types                                                                                                                                |
| 15 (Routing Unit Tests)           | Minor    | Tests should cover `Matched: false` no-match scenarios instead of default-agent scenarios. The deferred items from Phase 13 review (H1, L1) are resolved by this change. |
| 16 (Route Endpoint)               | Moderate | Handler must translate `Decision.Matched == false` to the chosen HTTP response (200 with `matched: false`). No need to pass `defaultAgent` to the engine during wiring.  |
| 17-18 (Patterns/Agents Endpoints) | None     | No dependency on default matching                                                                                                                                        |
| 25-28 (E2E and Final Validation)  | Minor    | E2E test `TestRoute_DefaultMatch` rewritten. Default-match confidence/reasoning tests removed.                                                                           |

**Dependencies table update:** Phase 13's dependency on Phase 9 remains unchanged. No new dependencies are introduced.

---

## Edge Cases

### 1. Rule cache is empty (no rules loaded)

**Current behavior:** Returns hardcoded default decision with `defaultAgent`.
**New behavior:** Returns `Decision{Matched: false}, nil`. The `for _, rule := range rules` loop body never executes, and the function falls through to the no-match return.

This is correct. An empty rule cache means the operator has not configured any routing rules. The client should handle this, not the engine.

### 2. All rules are disabled

**Current behavior:** Same as empty cache -- falls through to hardcoded default.
**New behavior:** Same as empty cache -- `Decision{Matched: false}, nil`. Each rule is skipped by the `if !rule.Enabled` guard, and the loop completes without a match.

Correct. Disabled rules should not produce matches.

### 3. The only matching rule's matcher returns an error

**Current behavior:** The erroring rule is skipped (logged as warning), and if no subsequent rules match, the hardcoded default is returned.
**New behavior:** The erroring rule is skipped (logged as warning), and if no subsequent rules match, `Decision{Matched: false}, nil` is returned.

Correct. A matcher error on the only candidate rule means no rules matched. The client can inspect the response and decide how to proceed. Observability (warning logs, error metrics) provides operator visibility.

### 4. Context cancellation during rule evaluation

**Current behavior:** Returns `Decision{}, err` where err wraps `context.Canceled` or `context.DeadlineExceeded`.
**New behavior:** No change. Context cancellation returns an error, not a no-match decision. This remains correct.

### 5. Existing `default`-type rules in the database

If any rules with `match_type = 'default'` exist in the database, the engine will log a warning ("no matcher registered for match type") and skip them. This is safe degradation. Operators should remove these rules, but the system does not break if they exist.

### 6. Client receives `matched: false` -- what does it do?

This is explicitly outside the routing engine's scope. The design principle is that the engine evaluates rules; the client decides policy. Reasonable client behaviors include:

- Route to a general-purpose agent (the client's own configuration, not the engine's).
- Ask the user to rephrase.
- Log and alert the operator that routing rules may need updating.
- Fall back to a local default.

The ACE CLI (the primary client) will need to implement its own fallback logic. This should be documented in the CLI design but does not affect the Mnemonic backend.

---

## Migration Strategy

This change should be implemented as a single phase (the redefined Phase 13) in this order:

1. **Add `Matched bool` to `Decision` struct** and set `Matched: true` in the engine's match path. Existing default fallback still works. All existing tests still pass.
2. **Update engine tests** to assert `Matched == true` on match cases and `Matched == false` on no-match cases.
3. **Remove `defaultAgent` from Engine** and `NewEngine`. Remove the hardcoded fallback block. Update the no-match path to return `Decision{}, nil`.
4. **Remove `MatchTypeDefault`** constant, `DefaultMatchConfig`, and their references in routingrule package.
5. **Delete `default_matcher.go` and `default_matcher_test.go`**.
6. **Remove `routing.default_agent`** from config, defaults, and config tests.
7. **Update OpenAPI spec**.
8. **Update design and architecture docs**.
9. **Run full test suite** and static analysis.

Steps 1-2 are backward-compatible. Steps 3-6 are the breaking changes. This ordering minimizes the window where tests are red.

A new ADR (ADR-007: Explicit No-Match Signaling) should be added to `docs/architecture/02-architectural-decisions.md` documenting the decision, context (dual-path fallback from Phase 13 review), and consequences.
