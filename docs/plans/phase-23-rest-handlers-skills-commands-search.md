# Phase 23: REST Admin Handlers -- Skills, Commands, Search, Router

> Part of the [MVP Implementation Plan](mvp-implementation-plan.md)

**Goal:** Implement the remaining REST handlers for skills, commands, and search. Wire all handlers together in a router.

**Agent(s):** go-software-engineer

**Dependencies:** Phase 20 (skill repo), Phase 21 (command repo), Phase 22 (handler pattern established)

---

## Step 1: Create skill handlers

- Create file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/handlers/skills/skills.go`
- Define `Handler struct { repo skill.Repository }`
- Implement: `List`, `Create` (201), `Get` (by name), `Update`, `Delete` (204)
- Routes: GET/POST `/skills`, GET/PUT/DELETE `/skills/:name`
- Handlers marshal/unmarshal JSONB `definition` field to/from API response shape (description, content, version, tags)
- Validation is application-layer (checking JSONB field values, not relying on DB column constraints)
- API wire format remains unchanged from specification
- Agent: `go-software-engineer`
- Design reference: [API Specification - Skills](../design/2026-02-15-pivot-api-specification.md#26-skills)

## Step 2: Create skill handler tests

- Create file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/handlers/skills/skills_test.go`
- Agent: `go-software-engineer`

## Step 3: Create skill handler doc.go

- Create file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/handlers/skills/doc.go`
- Agent: `go-software-engineer`

## Step 4: Create command handlers

- Create file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/handlers/commands/commands.go`
- Same structure as skill handlers, targeting `command.Repository`
- Handlers marshal/unmarshal JSONB `definition` field to/from API response shape
- Application-layer validation of JSONB field values
- API wire format remains unchanged from specification
- Agent: `go-software-engineer`
- Design reference: [API Specification - Commands](../design/2026-02-15-pivot-api-specification.md#27-commands)

## Step 5: Create command handler tests

- Create file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/handlers/commands/commands_test.go`
- Agent: `go-software-engineer`

## Step 6: Create command handler doc.go

- Create file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/handlers/commands/doc.go`
- Agent: `go-software-engineer`

## Step 7: Create search handler

- Create file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/handlers/search/search.go`
- Define `Handler struct { patternRepo pattern.Repository, graphRepo graph.Repository }`
- Implement: `Search` (GET `/search`) -- accepts query params `q`, `limit`, `threshold`, `tags`, `agent`
- MVP behavior: since OpenAI embedding integration is not yet built, fall back to `pattern.Repository.List()` with `SearchQuery` filter (full-text search on name/description). Return results in the search response format.
- Agent: `go-software-engineer`
- Design reference: [API Specification - Search](../design/2026-02-15-pivot-api-specification.md#25-patterns) (search endpoint)

## Step 8: Create search handler tests

- Create file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/handlers/search/search_test.go`
- Agent: `go-software-engineer`

## Step 9: Create search handler doc.go

- Create file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/handlers/search/doc.go`
- Agent: `go-software-engineer`

## Step 10: Create 410 Gone handler for removed routing endpoints

- Create file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/handlers/gone/gone.go`
- Implement a handler that returns 410 Gone with RFC 7807 body for `POST /v1/api/route` and all `/v1/api/routing-rules/*` paths
- Agent: `go-software-engineer`
- Design reference: [API Specification - Removed Endpoints](../design/2026-02-15-pivot-api-specification.md#29-removed-endpoints)

## Step 11: Create router

- Create file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/handlers/router.go`
- Define `Repositories` struct holding all repository interfaces
- Define `NewRouter(repos *Repositories, graphRepo graph.Repository, tel *telemetry.Telemetry) *gin.Engine`
- Wire: recovery middleware, tracing, logging, request metrics
- Register operations handlers (health, version) at `/ops/*`
- Create versioned API group at `/v1/api/`
- Register agent, pattern, skill, command, search handlers via `SetupRoutes`
- Register 410 Gone handlers for old routing paths under `/v1/api/`
- Agent: `go-software-engineer`
- Design reference: [Go Architecture Plan - Router Setup](2026-02-15-go-architecture-plan.md#6-package-structure)

## Step 12: Run tests

- Run: `cd /Users/doublej/dev/mnemonic/src/mnemonic && go test -v ./internal/handlers/...`
- Run: `cd /Users/doublej/dev/mnemonic/src/mnemonic && go test ./...` -- no regressions
- Agent: `go-software-engineer`

## Step 13: Commit

```bash
git add src/mnemonic/internal/handlers/
git commit -m "feat(pivot): implement skill, command, search handlers; create router with all admin endpoints"
```
