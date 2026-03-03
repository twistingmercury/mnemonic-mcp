# OpenAPI Spec Compliance Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Resolve all 15 discrepancies between the OpenAPI spec (`docs/openapi/mnemonic-v1.yaml`) and the Go handler implementations.

**Architecture:** Discrepancies fall into two categories: (1) the spec is missing status codes the implementation already correctly returns — fix by updating the YAML; (2) the implementation violates the spec contract — fix with TDD in the handler files.

**Tech Stack:** Go, Gin, Testify mocks, `net/http/httptest`

---

## Pre-work: Two False Positives (No Action Required)

Before starting, note these two reported discrepancies are **NOT real bugs**:

- **D12 — POST skillfile missing 409**: False positive. The skillfile service already wraps
  `skillfilerepo.ErrExists` as `service.ErrConflict` (see `service/skillfile/service.go:96-98`),
  which `RespondError` maps to 409. The code path is correct.

- **D10 — GET /patterns/{id}/chunks missing 503**: False positive. The handler routes errors
  through `RespondError`, which maps `service.ErrServiceUnavailable` to 503. The code path
  exists; whether the chunk repo ever returns that sentinel is a separate concern and does
  not change the handler contract.

---

## Task 1: Update OpenAPI Spec — Add Missing Status Codes

**Files:**
- Modify: `docs/openapi/mnemonic-v1.yaml`

The spec is missing status codes that the implementation correctly returns.
All changes are adding `$ref` lines to existing `responses:` blocks.

**Step 1: Add `400` to GET /v1/api/agents (list)**

Find `operationId: listAgents`. Its `responses:` block currently ends at `"500"`.
Add immediately before the `"500"` line:

```yaml
        "400":
          $ref: "#/components/responses/BadRequest"
```

**Step 2: Add `503` to POST /v1/api/agents (create)**

Find `operationId: createAgent`. Its `responses:` block currently ends at `"500"`.
Add immediately before the `"500"` line:

```yaml
        "503":
          $ref: "#/components/responses/ServiceUnavailable"
```

**Step 3: Add `409` to PUT /v1/api/agents/{name} (update)**

Find `operationId: updateAgent`. Its `responses:` block does not include 409.
Add after the `"404"` entry and before the `"500"` line:

```yaml
        "409":
          $ref: "#/components/responses/Conflict"
```

**Step 4: Add `400` to GET /v1/api/patterns (list)**

Find `operationId: listPatterns`. Its `responses:` block ends at `"500"`.
Add immediately before the `"500"` line:

```yaml
        "400":
          $ref: "#/components/responses/BadRequest"
```

**Step 5: Add `503` to POST /v1/api/patterns (create)**

Find `operationId: createPattern`. Its `responses:` block ends at `"500"`.
Add immediately before the `"500"` line:

```yaml
        "503":
          $ref: "#/components/responses/ServiceUnavailable"
```

**Step 6: Add `400` to GET /v1/api/patterns/{id} (get)**

Find `operationId: getPattern`. Its `responses:` block ends at `"500"`.
Add immediately before the `"500"` line:

```yaml
        "400":
          $ref: "#/components/responses/BadRequest"
```

**Step 7: Add `409` to PUT /v1/api/patterns/{id} (update)**

Find `operationId: updatePattern`. Its `responses:` block does not include 409.
Add after the `"404"` entry and before the `"500"` line:

```yaml
        "409":
          $ref: "#/components/responses/Conflict"
```

**Step 8: Add `400` to DELETE /v1/api/patterns/{id} (delete)**

Find `operationId: deletePattern`. Its `responses:` block ends at `"500"`.
Add immediately before the `"500"` line:

```yaml
        "400":
          $ref: "#/components/responses/BadRequest"
```

**Step 9: Add `400` to GET /v1/api/patterns/{id}/agents (get associations)**

Find `operationId: getPatternAgentAssociations`. Its `responses:` block ends at `"500"`.
Add immediately before the `"500"` line:

```yaml
        "400":
          $ref: "#/components/responses/BadRequest"
```

**Step 10: Add `400` to GET /v1/api/skills (list)**

Find `operationId: listSkills`. Its `responses:` block ends at `"500"`.
Add immediately before the `"500"` line:

```yaml
        "400":
          $ref: "#/components/responses/BadRequest"
```

**Step 11: Validate the YAML is well-formed**

Run:
```bash
cd /Users/doublej/dev/mnemonic
npx --yes @redocly/cli lint docs/openapi/mnemonic-v1.yaml
```

Expected: zero errors. If a linter is not available, at minimum verify YAML parses:
```bash
python3 -c "import yaml; yaml.safe_load(open('docs/openapi/mnemonic-v1.yaml'))" && echo "OK"
```

**Step 12: Commit**

```bash
git add docs/openapi/mnemonic-v1.yaml
git commit -m "docs(openapi): add missing 400/409/503 status codes to spec

Endpoints that correctly return 400 (invalid limit/cursor/UUID), 409
(conflict on update), and 503 (service unavailable) were missing these
codes from their response definitions.

Resolves D1-D9 and D11 from openapi-spec-compliance audit."
```

---

## Task 2: Fix Agent Create — Require `description` and `version`

**Files:**
- Modify: `src/mnemonic/internal/handlers/agents/agents.go`
- Modify: `src/mnemonic/internal/handlers/agents/agents_test.go`

The spec marks both `description` and `version` as required in `AgentCreate`.
`validateAgentFields` currently treats both as optional.

**Step 1: Write failing tests**

In `src/mnemonic/internal/handlers/agents/agents_test.go`, add two test cases to the
existing Create test table (find the test function for `POST /v1/api/agents`, likely
`TestCreate` or similar):

```go
{
    name:       "missing description returns 400",
    body:       `{"name":"test-agent","system_prompt":"You are a test agent.","model":"sonnet","version":"1.0.0"}`,
    wantStatus: http.StatusBadRequest,
},
{
    name:       "missing version returns 400",
    body:       `{"name":"test-agent","description":"A test agent.","system_prompt":"You are a test agent.","model":"sonnet"}`,
    wantStatus: http.StatusBadRequest,
},
```

**Step 2: Run tests to verify they fail**

```bash
cd src/mnemonic
go test ./internal/handlers/agents/... -run TestCreate -v
```

Expected: the two new cases FAIL (currently return 201, want 400).

**Step 3: Fix `validateAgentFields` in agents.go**

In `src/mnemonic/internal/handlers/agents/agents.go`, the function
`validateAgentFields` starts at line 121. Its signature is:
```go
func validateAgentFields(name, systemPrompt, model, description string) []handlers.FieldError {
```

Add `version` as a parameter and add required checks for both `description` and `version`:

```go
func validateAgentFields(name, systemPrompt, model, description, version string) []handlers.FieldError {
	var errs []handlers.FieldError

	// name: required, regex, max 64
	if name == "" {
		errs = append(errs, handlers.FieldError{Field: "name", Code: "REQUIRED", Message: "name is required"})
	} else if utf8.RuneCountInString(name) > 64 {
		errs = append(errs, handlers.FieldError{Field: "name", Code: "MAX_LENGTH", Message: "name must be 64 characters or fewer"})
	} else if !agentNameRe.MatchString(name) {
		errs = append(errs, handlers.FieldError{Field: "name", Code: "INVALID_FORMAT", Message: "name must match ^[a-z]([a-z0-9](-[a-z0-9])*)*$"})
	}

	// system_prompt: required, max 2048
	if systemPrompt == "" {
		errs = append(errs, handlers.FieldError{Field: "system_prompt", Code: "REQUIRED", Message: "system_prompt is required"})
	} else if utf8.RuneCountInString(systemPrompt) > 2048 {
		errs = append(errs, handlers.FieldError{Field: "system_prompt", Code: "MAX_LENGTH", Message: "system_prompt must be 2048 characters or fewer"})
	}

	// model: required
	if model == "" {
		errs = append(errs, handlers.FieldError{Field: "model", Code: "REQUIRED", Message: "model is required"})
	}

	// description: required, max 500
	if description == "" {
		errs = append(errs, handlers.FieldError{Field: "description", Code: "REQUIRED", Message: "description is required"})
	} else if utf8.RuneCountInString(description) > 500 {
		errs = append(errs, handlers.FieldError{Field: "description", Code: "MAX_LENGTH", Message: "description must be 500 characters or fewer"})
	}

	// version: required, max 50
	if version == "" {
		errs = append(errs, handlers.FieldError{Field: "version", Code: "REQUIRED", Message: "version is required"})
	} else if utf8.RuneCountInString(version) > 50 {
		errs = append(errs, handlers.FieldError{Field: "version", Code: "MAX_LENGTH", Message: "version must be 50 characters or fewer"})
	}

	return errs
}
```

Update both call sites of `validateAgentFields` to pass `version`:

In `Create` (around line 161):
```go
if fieldErrs := validateAgentFields(req.Name, req.SystemPrompt, req.Model, req.Description, req.Version); len(fieldErrs) > 0 {
```

In `Update` (around line 296):
```go
if fieldErrs := validateAgentFields(effectiveName, req.SystemPrompt, req.Model, req.Description, req.Version); len(fieldErrs) > 0 {
```

**Step 4: Run tests to verify they pass**

```bash
cd src/mnemonic
go test ./internal/handlers/agents/... -v
```

Expected: all tests PASS, including the two new cases.

**Step 5: Commit**

```bash
git add src/mnemonic/internal/handlers/agents/agents.go \
        src/mnemonic/internal/handlers/agents/agents_test.go
git commit -m "fix(agents): enforce description and version as required fields

The OpenAPI spec marks both description and version as required in
AgentCreate. The handler was treating them as optional.

Resolves D13 from openapi-spec-compliance audit."
```

---

## Task 3: Fix Skill `description` Max Length (1024 → 500)

**Files:**
- Modify: `src/mnemonic/internal/handlers/skills/skills.go`
- Modify: `src/mnemonic/internal/handlers/skills/skills_test.go`

The spec constrains `description` to 1–500 chars. The implementation validates max 1024.
The fix is in two places: the `Create` handler (line 179) and the `Update` handler (line 362).

**Step 1: Write failing tests**

In `src/mnemonic/internal/handlers/skills/skills_test.go`, add test cases:

```go
{
    name: "description over 500 chars on create returns 400",
    body: func() string {
        long := strings.Repeat("x", 501)
        b, _ := json.Marshal(map[string]any{
            "name":        "test-skill",
            "description": long,
            "content":     "# Test\n\nContent here.",
            "version":     "1.0.0",
        })
        return string(b)
    }(),
    wantStatus: http.StatusBadRequest,
},
```

Add a parallel case for Update.

**Step 2: Run tests to verify they fail**

```bash
cd src/mnemonic
go test ./internal/handlers/skills/... -run "Test.*description" -v
```

Expected: FAIL (currently 201/200, want 400).

**Step 3: Change the validation limit in skills.go**

There are two identical blocks to update (one in `Create`, one in `Update`).
Find this comment and validation block (appears twice):

```go
	// Description length validation (max 1024 characters).
	if utf8.RuneCountInString(req.Description) > 1024 {
		fieldErrs = append(fieldErrs, handlers.FieldError{
			Field:   "description",
			Code:    "MAX_LENGTH",
			Message: "description must be 1024 characters or fewer",
		})
	}
```

Change **both occurrences** to:

```go
	// Description length validation (max 500 characters).
	if utf8.RuneCountInString(req.Description) > 500 {
		fieldErrs = append(fieldErrs, handlers.FieldError{
			Field:   "description",
			Code:    "MAX_LENGTH",
			Message: "description must be 500 characters or fewer",
		})
	}
```

**Step 4: Run tests to verify they pass**

```bash
cd src/mnemonic
go test ./internal/handlers/skills/... -v
```

Expected: all tests PASS.

**Step 5: Commit**

```bash
git add src/mnemonic/internal/handlers/skills/skills.go \
        src/mnemonic/internal/handlers/skills/skills_test.go
git commit -m "fix(skills): align description max length with spec (500 chars)

The OpenAPI spec constrains description to 500 chars. The handler was
validating against 1024. Applied to both Create and Update handlers.

Resolves D14 from openapi-spec-compliance audit."
```

---

## Task 4: Fix Skillfile Update — Make `content_type` Optional

**Files:**
- Modify: `src/mnemonic/internal/handlers/skillfiles/skillfiles.go`
- Modify: `src/mnemonic/internal/handlers/skillfiles/skillfiles_test.go`
- Modify: `src/mnemonic/internal/service/skillfile/service.go`

The spec's `SkillFileUpdate` schema marks `content_type` as **optional** (only `content` is
required). The `fileUpdateRequest` struct uses `binding:"required"` on `content_type`,
which causes Gin to reject requests without it before the handler even runs.

Additionally, the `UpdateInput` service struct includes `ContentType` — when the client
omits `content_type`, the service should preserve the existing file's content type rather
than overwrite it with an empty string.

**Step 1: Write failing tests**

In `src/mnemonic/internal/handlers/skillfiles/skillfiles_test.go`, add:

```go
{
    name:       "update script without content_type succeeds",
    method:     http.MethodPut,
    path:       "/v1/api/skills/my-skill/scripts/run.sh",
    body:       `{"content":"#!/bin/bash\necho hello"}`,
    wantStatus: http.StatusOK,
},
```

**Step 2: Run tests to verify they fail**

```bash
cd src/mnemonic
go test ./internal/handlers/skillfiles/... -run "Test.*update.*content_type" -v
```

Expected: FAIL (currently returns 400 because `content_type` binding fails).

**Step 3: Fix `fileUpdateRequest` in skillfiles.go**

In `src/mnemonic/internal/handlers/skillfiles/skillfiles.go`, find the struct at line 89:

```go
type fileUpdateRequest struct {
	ContentType string `json:"content_type" binding:"required"`
	Content     string `json:"content" binding:"required"`
	Encoding    string `json:"encoding"`
}
```

Remove `binding:"required"` from `ContentType`:

```go
type fileUpdateRequest struct {
	ContentType string `json:"content_type"`
	Content     string `json:"content" binding:"required"`
	Encoding    string `json:"encoding"`
}
```

**Step 4: Fix the `updateFile` handler to preserve existing content_type when omitted**

In the `updateFile` factory function (around line 318), the service call currently uses
`req.ContentType` directly. When the client omits `content_type`, this is empty string,
which would overwrite the stored content type.

Change the handler to fall back to the existing file's content type. This requires
fetching the existing file before updating. However, the service's `Update` method
already fetches the existing file internally (see `service/skillfile/service.go:135`).
The fix is to pass the empty `content_type` down and let the service preserve the
existing value when it is empty.

Update `src/mnemonic/internal/service/skillfile/service.go` in the `Update` method.
Find (around line 143):

```go
	existing.Content = input.Content
	existing.CRC64 = computeCRC64(input.Content)
```

Add a conditional content type update after those lines:

```go
	existing.Content = input.Content
	existing.CRC64 = computeCRC64(input.Content)
	if input.ContentType != "" {
		existing.ContentType = input.ContentType
	}
```

> **Note:** This requires that `skillfilerepo.SkillFile` has a `ContentType` field.
> If the content type is not stored in the repository (currently the handler uses
> `inferContentType` to derive it from the filename), then skip the service change
> and simply ensure the handler uses `inferContentType(filename)` as the fallback
> when `req.ContentType` is empty. In that case, update the handler:

```go
		contentType := req.ContentType
		if contentType == "" {
			contentType = inferContentType(filename)
		}

		file, err := h.svc.Update(c.Request.Context(), skillName, fileType, filename, skillfilesvc.UpdateInput{
			ContentType: contentType,
			Content:     req.Content,
			Encoding:    encoding,
		})
```

Use whichever approach matches the actual `SkillFile` struct fields.

**Step 5: Run tests to verify they pass**

```bash
cd src/mnemonic
go test ./internal/handlers/skillfiles/... -v
go test ./internal/service/skillfile/... -v
```

Expected: all tests PASS.

**Step 6: Run full unit test suite to catch regressions**

```bash
cd src/mnemonic
go test ./...
```

Expected: all tests PASS, zero failures.

**Step 7: Commit**

```bash
git add src/mnemonic/internal/handlers/skillfiles/skillfiles.go \
        src/mnemonic/internal/handlers/skillfiles/skillfiles_test.go \
        src/mnemonic/internal/service/skillfile/service.go
git commit -m "fix(skillfiles): make content_type optional on file update

The OpenAPI spec SkillFileUpdate schema marks content_type as optional.
The handler was using binding:\"required\" which rejected requests that
omitted it. When omitted, the existing or inferred content type is used.

Resolves D15 from openapi-spec-compliance audit."
```

---

## Final Verification

After all four tasks are complete:

```bash
cd src/mnemonic
go test ./...
```

Expected: all tests PASS. Then optionally run the E2E suite:

```bash
cd /Users/doublej/dev/mnemonic
src/mnemonic/tests/run-e2e.sh
```

Expected: exit 0.
