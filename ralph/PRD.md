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

- [x] **Cycle 4 — Update OpenAPI spec: remove enums**
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

- [x] **Cycle 5 — PUT handlers return 204**
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

- [x] **Cycle 6 — Fix unit tests: PUT returns 204**
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

- [x] **Cycle 7 — Fix E2E tests: PUT returns 204**
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

- [x] **Cycle 8 — Add VocabularyConfig to config package**
  - Agent: `go software engineer`
  - Files:
    - `src/mnemonic/internal/config/config.go`
    - `src/mnemonic/internal/config/config_test.go`
    - `src/mnemonic/config.yaml` (create if absent)
  - Steps:
    - Add `VocabularyConfig` struct to `config.go`:
      ```go
      // VocabularyConfig holds the allowed values for pattern language and domain fields.
      type VocabularyConfig struct {
          Languages []string `mapstructure:"languages"`
          Domains   []string `mapstructure:"domains"`
      }
      ```
    - Add `Vocabulary VocabularyConfig `mapstructure:"vocabulary"`` field to `MnemonicConfig`.
    - Add a `validate()` method on `VocabularyConfig` that returns a `ValidationError` if `Languages` is empty and another if `Domains` is empty. No defaults — fail fast.
    - Call `c.Vocabulary.validate()` inside `MnemonicConfig.Validate()`.
    - Do NOT add `SetDefaults` entries for vocabulary — the lists must be explicitly configured.
    - Create `src/mnemonic/config.yaml` with a `vocabulary:` block containing the canonical lists:
      ```yaml
      vocabulary:
        languages:
          - agnostic
          - go
          - python
          - dotnet
          - shell
          - typescript
          - react
          - sql
          - cypher
        domains:
          - api-design
          - backend
          - frontend
          - testing
          - devops
          - cli
          - data-design
          - documentation
      ```
    - Add unit tests in `config_test.go` covering: (a) empty Languages fails validation, (b) empty Domains fails validation, (c) both populated passes validation.
  - Verify: `cd src/mnemonic && go build ./... && go vet ./... && go test ./internal/config/... && make analyze`
  - Done: All commands exit 0.

- [x] **Cycle 9 — Wire vocabulary into pattern handler**
  - Agent: `go software engineer`
  - Files:
    - `src/mnemonic/internal/handlers/patterns/patterns.go`
    - `src/mnemonic/internal/server/routes.go`
  - Steps:
    - In `patterns.go`:
      - Add `allowedLanguages []string` and `allowedDomains []string` fields to the `Handler` struct.
      - Update `New` signature to accept a `config.VocabularyConfig` as a third parameter: `func New(patternSvc patternsvc.Service, searchSvc searchsvc.Service, vocab config.VocabularyConfig) *Handler`. Store `vocab.Languages` and `vocab.Domains` on the handler.
      - In `validatePatternFields`, after the kebab-case format check for `language`, add a membership check: if the value is not in `h.allowedLanguages`, append `FieldError{Field: "language", Code: "INVALID_VALUE"}`. Same for `domain` against `h.allowedDomains`.
      - Also fix the `len()` → `utf8.RuneCountInString()` bug on the language and domain max-length checks (HIGH finding H1 from the code review).
      - Add the `unicode/utf8` import if not already present.
    - In `routes.go`:
      - Update `RegisterAPIRoutes` signature to accept `cfg config.VocabularyConfig` as a third parameter.
      - Pass `cfg` to `patternhandler.New`.
  - Verify: `cd src/mnemonic && go build ./... && go vet ./... && go test ./internal/handlers/patterns/... && make analyze`
  - Done: All commands exit 0.

- [x] **Cycle 10 — Wire vocabulary through server startup**
  - Agent: `go software engineer`
  - Files:
    - `src/mnemonic/internal/server/server.go`
  - Steps:
    - Find the call to `RegisterAPIRoutes` in `server.go`. Update the call to pass `cfg.Vocabulary` as the third argument (where `cfg` is the `*config.MnemonicConfig` already available in the server setup).
    - If `cfg` is not directly accessible at the call site, trace the call chain and thread it through — do not introduce a global variable.
  - Verify: `cd src/mnemonic && go build ./... && go vet ./... && make analyze`
  - Done: All commands exit 0.

- [x] **Cycle 11 — Fix unit tests: vocabulary validation**
  - Agent: `go software engineer`
  - Files:
    - `src/mnemonic/internal/handlers/patterns/patterns_test.go`
  - Steps:
    - The `New` function now requires a `config.VocabularyConfig` argument. Find all calls to `patternhandler.New` (or the handler constructor) in `patterns_test.go` and add a `config.VocabularyConfig` with `Languages` and `Domains` populated with reasonable test values (include the values used in existing happy-path tests).
    - Find `TestPatternCreate_InvalidLanguage` and `TestPatternUpdate_InvalidLanguage` (or equivalent). The test currently sends a non-kebab value and expects `INVALID_FORMAT`. That test remains correct. Add a new sub-case (or separate test) that sends a valid kebab-case value that is NOT in the allowed list and expects `INVALID_VALUE`.
    - Do the same for domain: add a test case that sends a kebab-case domain not in the allowed list and expects `INVALID_VALUE`.
  - Verify: `cd src/mnemonic && go test ./internal/handlers/patterns/... -v 2>&1 | tail -30`
  - Done: `go test ./internal/handlers/patterns/...` exits 0.

- [ ] **Cycle 12 — Run E2E tests until all pass**
  - Agent: `go software engineer`
  - Files: `src/mnemonic/tests/e2e/patterns_test.go` (fix as needed)
  - Steps:
    - Run `cd src/mnemonic && bash tests/run-e2e.sh`.
    - If any test fails because the running server now rejects a language/domain value that was valid under the old open-vocabulary rules, update the E2E test to use a value in the allowed vocabulary list.
    - Do not change business logic — only fix test data.
    - Iterate until `run-e2e.sh` exits 0.
  - Verify: `cd src/mnemonic && bash tests/run-e2e.sh`
  - Done: `run-e2e.sh` exits 0 with all tests passing.
