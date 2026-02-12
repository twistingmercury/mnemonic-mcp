# Architecture Review: Phase 13 -- Remove Default Fallback, Implement Explicit No-Match Signaling

**Review Date:** 2026-02-12
**Reviewer:** solutions-architect
**Scope:** Uncommitted changes on `phase/13` branch
**Inputs:** `git diff`, design change assessment, implementation plan, OpenAPI spec, Go source

---

## Verdict

The design change is architecturally sound. The "no match = no match, client decides" principle is applied consistently across domain, API contract, and documentation layers. Six findings remain, none of which block the change from proceeding.

---

## Findings

### F1 -- MEDIUM -- E2E: `routing_rules_test.go` stale default references

**Area:** Test consistency
**Files:** `src/mnemonic/tests/e2e/routing_rules_test.go`

The E2E routing test file (`routing_test.go`) was updated, but the routing *rules* test file was not. Lines 16, 24, 180-194 still reference `match_type: default`, describe default fallback behavior, and contain a `TestCreateRoutingRule_DefaultMatch` stub that instructs the implementer to "POST with default match_type." Since `default` is no longer a valid `match_type` in the API, this stub is misleading.

**Recommendation:** Update the header comment (lines 16, 24) to remove `default` from the match type list. Rewrite or remove `TestCreateRoutingRule_DefaultMatch` (lines 180-195) -- either delete it entirely or convert it to a test that verifies the API rejects `match_type: default` with a 400 response.

---

### F2 -- MEDIUM -- Design doc: `routing-engine.md` RuleMatcher class diagram retains DefaultMatcher

**Area:** Doc consistency
**Files:** `docs/design/routing-engine.md` lines 177-185

The "Matcher Implementations" class diagram (lines ~170-186) still includes the `DefaultMatcher` class and its `implements` relationship to `RuleMatcher`. The diff removed the Default Matcher *section* (lines 896-927) and several other references, but this earlier diagram was missed.

**Recommendation:** Delete the `DefaultMatcher` class block (lines 177-180) and the `RuleMatcher <|.. DefaultMatcher : implements` relationship (line 185) from the Mermaid diagram.

---

### F3 -- LOW -- `function-documentation-map.md` retains DefaultMatchConfig entry

**Area:** Doc consistency
**Files:** `docs/function-documentation-map.md` line 243

The function documentation map still has an entry mapping `(DefaultMatchConfig) Type()` to the deleted Default Matcher design doc section. This will produce a broken cross-reference.

**Recommendation:** Remove the line. The implementation plan (Step 8z) already calls for this, so it appears to have been missed during execution.

---

### F4 -- LOW -- E2E: `TestRoute_NoMatchNoDefault` is now semantically ambiguous

**Area:** Test design
**Files:** `src/mnemonic/tests/e2e/routing_test.go` lines 216-228

`TestRoute_NoMatchNoDefault` was written to test the case "404 when no agent matches and no default configured." With the default fallback removed, the name and description are confusing. The updated `TestRoute_DefaultMatch` (lines 200-214) already covers the no-match scenario (200 OK with `matched: false`). Meanwhile, `NoMatchNoDefault` describes a 404 for "strict routing mode" that does not exist in the current design.

**Recommendation:** Either remove `TestRoute_NoMatchNoDefault` entirely (since there is no strict routing mode in the design) or rename it to document the 404 as a data-integrity error (matched agent not found in DB), which aligns with the updated OpenAPI 404 description.

---

### F5 -- LOW -- Metrics: `recordNoMatch` omits `RecordRoutingDecision`

**Area:** Observability consistency
**Files:** `src/mnemonic/internal/routing/engine.go` lines 142-148

When a rule matches, `recordMetrics` calls both `RecordRoutingDecision(agentName)` and `RecordRuleMatch(matchType)`. When no rule matches, `recordNoMatch` calls only `RecordRuleMatch("no_match")`. This means `mnemonic.routing.decisions` undercounts total routing evaluations -- it only tracks decisions that produced a match. Operators who want "total routing requests" must sum two different metrics.

This is not necessarily wrong -- the counter is named "decisions" and an explicit no-match could be considered "no decision was made." But it is worth confirming this is the intended semantic. If operators want a single counter for "total route evaluations," `recordNoMatch` should also increment `routingDecisions` (perhaps with `agent=none`).

**Recommendation:** Confirm intent. If total-evaluation counting matters, add `RecordRoutingDecision(ctx, "none")` to `recordNoMatch`. If "decisions" should only count matches, document this in the metrics doc or a code comment.

---

### F6 -- LOW -- OpenAPI: `no_match` example omits `metadata.routing_duration_ms` type

**Area:** API contract clarity
**Files:** `api/openapi/mnemonic-v1.yaml` lines 1484-1490

The `no_match` response example is well-formed and includes `metadata.routing_duration_ms: 8`, which correctly shows that metadata is returned even on no-match. This is good. However, the `RouteMetadata` schema should be verified to ensure `routing_duration_ms` is required while `pattern_retrieval_duration_ms` and `total_patterns_considered` are optional (since pattern retrieval would not occur on a no-match). This is a pre-existing schema concern, not introduced by this change.

**Recommendation:** No action needed for this change. Note for Phase 16 handler implementation: ensure `RouteMetadata` fields that depend on pattern retrieval are omitted from no-match responses.

---

## Assessment by Review Focus Area

### 1. Consistency of "no match = no match, client decides"

**Rating: Consistent across all layers.**

- **Domain layer:** `Decision{Matched: false}` with zero-valued fields. The engine returns `nil` error. Clean separation.
- **API layer:** `RoutingDecision.matched` is the only required field. All other fields documented as "present only when matched is true." `agent` moved from required to optional in `RouteResponse`.
- **Documentation:** ADR-007 records the decision. Design doc updated. Configuration doc updated. Data architecture doc updated with deprecation notes.
- **Tests:** Unit tests assert `Matched: true/false` explicitly. E2E stubs updated (with the exception noted in F1).

### 2. Separation of concerns -- engine evaluates rules, not policy

**Rating: Achieved.**

The engine no longer makes a policy decision about what to do when nothing matches. It returns a fact: "nothing matched." The `defaultAgent` field, the `MatchTypeDefault` constant, and the `DefaultMatcher` are all removed. The engine's only job is: iterate rules, return first match or no-match.

The `recordNoMatch` metric gives operators visibility without introducing policy. The `buildReasoning` function is only called on the match path, so no-match produces no reasoning string -- correct, since there is nothing to explain.

### 3. API contract: 200 OK with `matched: false`

**Rating: Sound.**

The design correctly treats "no match" as a successful request with a specific business outcome, not an error. The 200 response with `matched: false` is the right choice. The 404 has been redefined to mean "data integrity error: matched agent not found in DB," which is the only remaining scenario where a 404 makes sense for this endpoint.

The `no_match` response example in the OpenAPI spec is well-structured: it includes `routing.matched: false` and `metadata` but omits `agent` and `patterns`.

### 4. Internal consistency of docs, ADR-007, and implementation plan

**Rating: Consistent, with two gaps (F2, F3).**

ADR-007 aligns with the design change assessment. The implementation plan's nine steps match the executed changes. The MVP plan row was updated. The code review doc has a superseded notice.

The two gaps are the RuleMatcher class diagram (F2) and the function documentation map (F3), both of which retain deleted references.

### 5. Architectural gaps or edge cases

**Rating: No gaps found.**

The design change assessment covers five edge cases (empty cache, all disabled, matcher error, context cancellation, stale DB rules) and all are handled correctly. The zero-value safety of `Decision{Matched: false}` is a strong property.

One question worth tracking for Phase 16: the handler implementation will need to conditionally omit `agent` and `patterns` from the JSON response when `matched: false`. This is straightforward but should be explicit in the Phase 16 plan.

### 6. Database deprecation strategy

**Rating: Sound for MVP.**

Leaving `'default'` in the CHECK constraint and rejecting at the application layer is the pragmatic choice. The deprecation is documented in three places: the data architecture ER diagram note, the SQL CREATE TABLE comment, and the match_config validation constraint comment. The `ValidMatchTypes` slice in Go no longer includes `"default"`, so the API validation layer (once built in Phase 16) will reject new default-type rules.

The only risk is that an operator manually inserts a `default`-type rule via SQL. The engine handles this safely: it logs a warning ("no matcher registered for match type") and skips the rule. This degradation path is documented in the design change assessment's edge case 5.

---

## Summary Table

| ID | Severity | Area                | Description                                                      |
| -- | -------- | ------------------- | ---------------------------------------------------------------- |
| F1 | MEDIUM   | E2E tests           | `routing_rules_test.go` stale default references                 |
| F2 | MEDIUM   | Design doc          | `routing-engine.md` RuleMatcher diagram retains DefaultMatcher   |
| F3 | LOW      | Function doc map    | `function-documentation-map.md` retains DefaultMatchConfig entry |
| F4 | LOW      | E2E tests           | `TestRoute_NoMatchNoDefault` semantically ambiguous              |
| F5 | LOW      | Observability       | `recordNoMatch` omits `RecordRoutingDecision` -- confirm intent  |
| F6 | LOW      | API contract        | `RouteMetadata` optional fields on no-match (Phase 16 concern)   |

No HIGH-severity findings. The design is architecturally coherent and ready to proceed after addressing F1-F3.
