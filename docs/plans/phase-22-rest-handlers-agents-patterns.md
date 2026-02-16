# Phase 22: REST Admin Handlers -- Agents and Patterns

> Part of the [MVP Implementation Plan](mvp-implementation-plan.md)

**Goal:** Rewrite the agent and pattern handler stubs as working CRUD handlers with dependency injection. Each handler struct receives a repository interface, making it testable with mocks.

**Agent(s):** go-software-engineer

**Dependencies:** Phase 18 (config overhaul), Phase 19 (agent repo with version)

---

## Step 1: Rewrite agent handlers

- Rewrite file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/handlers/agents/agents.go`
- Define `Handler struct { repo agent.Repository }`
- Define `NewHandler(repo agent.Repository) *Handler`
- Define `SetupRoutes(rg *gin.RouterGroup)` method: `rg.GET("/agents", h.List)`, `rg.POST("/agents", h.Create)`, etc.
- Implement: `List` (pagination via query params, returns JSON array with total count), `Create` (bind JSON, validate, return 201), `Get` (path param `:name`, return 200 or 404), `Update` (path param `:name`, bind JSON, return 200 or 404), `Delete` (path param `:name`, return 204 or 404)
- Map repository errors: `agent.ErrNotFound` -> 404, `agent.ErrExists` -> 409, `agent.ErrInUse` -> 409
- Omit `routing_keywords` from JSON response (deprecated field)
- Include `version` field in request/response JSON
- Agent: `go-software-engineer`
- Design reference: [API Specification - Agents](../design/2026-02-15-pivot-api-specification.md#24-agents)

## Step 2: Write agent handler tests

- Create file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/handlers/agents/agents_test.go`
- Use `httptest.NewRecorder()` and `gin.CreateTestContext()`
- Mock the `agent.Repository` interface (either hand-written mock or `mockgen`)
- Test cases: List (200, empty), Create (201, 400 bad JSON, 409 duplicate), Get (200, 404), Update (200, 404, 400), Delete (204, 404)
- Agent: `go-software-engineer`

## Step 3: Rewrite pattern handlers

- Rewrite file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/handlers/patterns/patterns.go`
- Define `Handler struct { repo pattern.Repository, graphRepo graph.Repository }`
- Implement: `List` (pagination, tag filter, search query), `Create` (bind JSON, return 202 Accepted with enrichment_status: pending), `Get` (UUID path param, include graph context from Neo4j), `Update`, `Delete`
- Implement pattern-agent association sub-routes: `SetAgentAssociations` (PUT `/patterns/:id/agents`), `GetAgentAssociations` (GET `/patterns/:id/agents`)
- Agent: `go-software-engineer`
- Design reference: [API Specification - Patterns](../design/2026-02-15-pivot-api-specification.md#25-patterns)

## Step 4: Write pattern handler tests

- Create file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/handlers/patterns/patterns_test.go`
- Mock both `pattern.Repository` and `graph.Repository`
- Test CRUD operations plus agent association endpoints
- Agent: `go-software-engineer`

## Step 5: Update doc.go files

- Update `/Users/doublej/dev/mnemonic/src/mnemonic/internal/handlers/agents/doc.go` -- new description mentioning dependency injection
- Update `/Users/doublej/dev/mnemonic/src/mnemonic/internal/handlers/patterns/doc.go`
- Agent: `go-software-engineer`

## Step 6: Run tests

- Run: `cd /Users/doublej/dev/mnemonic/src/mnemonic && go test -v ./internal/handlers/agents/...`
- Run: `cd /Users/doublej/dev/mnemonic/src/mnemonic && go test -v ./internal/handlers/patterns/...`
- Run: `cd /Users/doublej/dev/mnemonic/src/mnemonic && go test ./...` -- no regressions
- Agent: `go-software-engineer`

## Step 7: Commit

```bash
git add src/mnemonic/internal/handlers/agents/ src/mnemonic/internal/handlers/patterns/
git commit -m "feat(pivot): rewrite agent and pattern handlers with dependency injection"
```
