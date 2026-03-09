# PRD: Dynamic Language and Domain Fields

## Goal

Remove hardcoded enum constraints from `language` and `domain` on the pattern API. Accept any kebab-case value. Vocabulary governance moves to `mnemonic-patterns/config/validate.yaml`. Change all PUT endpoints from `200 OK` to `204 No Content`.

## Not in scope

- DB migrations (columns are `varchar(50)`, no CHECK constraints)
- Service, repository, MCP, or enrichment changes

## Implementation Plan

- [x] **Cycle 1 — Remove enum slices from handler**
  - Agent: `go software engineer`
  - Files: `src/mnemonic/internal/handlers/patterns/patterns.go`
  - Steps:
    - Delete `var allowedLanguages` and `var allowedDomains` (~lines 30-31)
    - Remove the unused `"slices"` import
    - In `validatePatternFields`, replace each `slices.Contains` enum check with `kebabCaseRe.MatchString` + `len() > 64` guard (reuse the existing `kebabCaseRe`). Keep the existing empty-string required-field checks.
  - Verify: `cd src/mnemonic && go build ./... && go vet ./... && go test ./internal/handlers/patterns/... && make analyze`
  - Done: All commands exit 0. No matches for `grep "allowedLanguages\|allowedDomains\|slices.Contains" src/mnemonic/internal/handlers/patterns/patterns.go`.

- [x] **Cycle 2 — Fix unit tests: language/domain format**
  - Agent: `go software engineer`
  - Files: `src/mnemonic/internal/handlers/patterns/patterns_test.go`
  - Steps:
    - Find `TestPatternCreate_InvalidLanguage` (~line 266). Its test value is now valid kebab-case. Change it to a non-kebab value (e.g. `"Not A Language"`) so it still expects 400.
    - Search for any other test asserting 400 for an enum-invalid (but kebab-valid) language or domain value; fix those the same way.
  - Verify: `cd src/mnemonic && go test ./internal/handlers/patterns/... -v 2>&1 | tail -20`
  - Done: `go test ./internal/handlers/patterns/...` exits 0.

- [x] **Cycle 3 — Fix E2E tests: language/domain format**
  - Agent: `go software engineer`
  - Files: `src/mnemonic/tests/e2e/patterns_test.go`
  - Steps:
    - In `TestCreatePattern_ValidationErrors` (~line 728) and `TestUpdatePattern_ValidationErrors` (~line 1667), find the cases for `"invalid language value"` (sends `"cobol"`) and `"invalid domain value"` (sends `"not-a-valid-domain"`).
    - Both values are valid kebab-case now. Change them to non-kebab values (e.g. `"COBOL"`, `"Not A Domain"`).
  - Verify: `cd src/mnemonic && go build ./tests/... && go vet ./tests/...`
  - Done: Both commands exit 0.

- [ ] **Cycle 4 — Update OpenAPI spec: remove enums**
  - Agent: `api architect`
  - Files: `docs/openapi/mnemonic-v1.yaml`
  - Steps:
    - Remove `enum:` blocks from `language` and `domain` in all 10 locations:
      - `Pattern` schema (~lines 555, 562)
      - `PatternCreate` schema (~lines 672, 681)
      - `PatternUpdate` schema (~lines 747, 757)
      - `GET /v1/api/patterns` query params (~lines 1756, 1768)
      - `GET /v1/api/patterns/search` query params (~lines 1938, 1950)
    - Replace each removed `enum:` block with `pattern: '^[a-z][a-z0-9-]*$'` and `maxLength: 64`. Append `Allowed values defined in mnemonic-patterns/config/validate.yaml.` to each `description:`.
    - `PatternSummary` and `PatternSearchResult` have no enum — leave them unchanged.
  - Verify: `grep -c "enum:" docs/openapi/mnemonic-v1.yaml` is unchanged (other fields use enum); confirm `grep -A5 "language:\|domain:" docs/openapi/mnemonic-v1.yaml` shows no enum on those fields.
  - Done: No `enum:` blocks remain on any `language` or `domain` field.

- [ ] **Cycle 5 — PUT handlers return 204**
  - Agent: `go software engineer`
  - Files:
    - `src/mnemonic/internal/handlers/patterns/patterns.go`
    - `src/mnemonic/internal/handlers/agents/agents.go`
    - `src/mnemonic/internal/handlers/skills/skills.go`
    - `src/mnemonic/internal/handlers/skillfiles/skillfiles.go`
    - `docs/openapi/mnemonic-v1.yaml`
  - Steps:
    - In each handler file, change every PUT handler from `c.JSON(http.StatusOK, ...)` to `c.Status(http.StatusNoContent)`. Affected handlers:
      - `patterns.go`: `h.Update` (~line 724), `h.SetAgentAssociations` (~line 845)
      - `agents.go`: `h.Update` (~line 378)
      - `skills.go`: `h.Update` (~line 479)
      - `skillfiles.go`: `updateFile` (~line 376)
    - Update each handler's `// @Success 200` swaggo annotation to `// @Success 204`.
    - Remove any now-unused response type references.
    - In `mnemonic-v1.yaml`: for each `put:` operation, change the `200:` success response to `204:` with description `No Content` and no response body.
  - Verify: `cd src/mnemonic && go build ./... && go vet ./... && make analyze`
  - Done: All commands exit 0.

- [ ] **Cycle 6 — Fix unit tests: PUT returns 204**
  - Agent: `go software engineer`
  - Files:
    - `src/mnemonic/internal/handlers/patterns/patterns_test.go`
    - `src/mnemonic/internal/handlers/agents/agents_test.go`
    - `src/mnemonic/internal/handlers/skills/skills_test.go`
    - `src/mnemonic/internal/handlers/skillfiles/skillfiles_test.go`
  - Steps:
    - In each file, find tests that call a PUT handler and assert `http.StatusOK` (e.g. `TestPatternUpdate_Success` ~line 448 in `patterns_test.go`).
    - Change those assertions to `http.StatusNoContent`. Remove any response body assertions that follow — 204 has no body.
    - Do not change GET, POST, or DELETE assertions.
  - Verify: `cd src/mnemonic && go test ./internal/handlers/...`
  - Done: `go test ./internal/handlers/...` exits 0.

- [ ] **Cycle 7 — Fix E2E tests: PUT returns 204**
  - Agent: `go software engineer`
  - Files:
    - `src/mnemonic/tests/e2e/agents_test.go`
    - `src/mnemonic/tests/e2e/patterns_test.go`
    - `src/mnemonic/tests/e2e/skillfiles_test.go`
  - Steps:
    - `agents_test.go`: rename `TestUpdateAgent_HappyPathReturns200` (~line 698) to `TestUpdateAgent_HappyPathReturns204`. Change all `AssertStatusCode(..., http.StatusOK)` in that function to `http.StatusNoContent`. Remove body assertions after PUT calls.
    - `patterns_test.go`: find the PUT assertion at ~line 2562; change to `http.StatusNoContent`. Remove body assertions.
    - `skillfiles_test.go`: find the PUT assertion at ~line 818; change to `http.StatusNoContent`. Remove body assertions.
    - Search all three files for any remaining `StatusOK` on PUT calls and update them.
  - Verify: `cd src/mnemonic && bash tests/run-e2e.sh`
  - Done: `run-e2e.sh` exits 0 with all tests passing. Fix any failures and re-run until clean.
