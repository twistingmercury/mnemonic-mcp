# Phase 26: E2E Tests

> Part of the [MVP Implementation Plan](mvp-implementation-plan.md)

**Goal:** End-to-end tests for both the Admin API and MCP server running against a live Docker Compose environment.

**Agent(s):** go-e2e-test-engineer

**Dependencies:** Phase 25 (full server running with both listeners)

---

## Step 1: Update E2E test helpers

- Modify file: `/Users/doublej/dev/mnemonic/src/mnemonic/tests/e2e/helpers.go`
- Update base URL to use admin port (8080) for REST tests
- Add MCP client helper for MCP tool calls
- Update any references to old routing endpoints
- Agent: `go-e2e-test-engineer`

## Step 2: Update E2E test types

- Modify file: `/Users/doublej/dev/mnemonic/src/mnemonic/tests/e2e/types.go`
- Add types for skill, command, and search response structures
- Add MCP JSON-RPC request/response types
- Agent: `go-e2e-test-engineer`

## Step 3: Update agent E2E tests

- Modify file: `/Users/doublej/dev/mnemonic/src/mnemonic/tests/e2e/agents_test.go`
- Update to use `/api/agents` (not `/v1/api/agents`)
- Add test for `version` field in create/get/update
- Test CRUD lifecycle: create -> get -> list -> update -> delete -> get (404)
- Test duplicate name (409)
- Agent: `go-e2e-test-engineer`

## Step 4: Update pattern E2E tests

- Modify file: `/Users/doublej/dev/mnemonic/src/mnemonic/tests/e2e/patterns_test.go`
- Update to use `/api/patterns`
- Test CRUD lifecycle including agent associations
- Test list with tag filter and search query
- Agent: `go-e2e-test-engineer`

## Step 5: Write skill E2E tests

- Create file: `/Users/doublej/dev/mnemonic/src/mnemonic/tests/e2e/skills_test.go`
- CRUD lifecycle, duplicate name, list with tag filter
- Agent: `go-e2e-test-engineer`

## Step 6: Write command E2E tests

- Create file: `/Users/doublej/dev/mnemonic/src/mnemonic/tests/e2e/commands_test.go`
- Same structure as skill E2E tests
- Agent: `go-e2e-test-engineer`

## Step 7: Write search E2E tests

- Create file: `/Users/doublej/dev/mnemonic/src/mnemonic/tests/e2e/search_test.go`
- Create patterns, then search via `GET /api/patterns/search?q=...`
- Verify results include matching patterns
- Agent: `go-e2e-test-engineer`

## Step 8: Write MCP E2E tests

- Create file: `/Users/doublej/dev/mnemonic/src/mnemonic/tests/e2e/mcp_test.go`
- Send `initialize` request, verify response
- Send `tools/list`, verify all expected tools are present (search_patterns, get_pattern, list_agents, list_skills, list_commands, get_agent, get_skill, get_command, get_sync_manifest, find_related_patterns)
- Create data via REST Admin API, then call MCP tools to verify read access
- Call `list_agents` after creating agents, verify data
- Call `get_agent` with valid/invalid name
- Call `get_sync_manifest`, verify collection version hashes
- Agent: `go-e2e-test-engineer`

## Step 9: Write 410 Gone E2E tests

- Add to existing test file or create new: verify `POST /v1/api/route` returns 410
- Verify `GET /v1/api/routing-rules` returns 410
- Agent: `go-e2e-test-engineer`

## Step 10: Run E2E tests

- Start full stack: `docker-compose up -d`
- Run: `cd /Users/doublej/dev/mnemonic/src/mnemonic && go test -v -tags=e2e ./tests/e2e/...`
- All tests must pass
- Agent: `go-e2e-test-engineer`

## Step 11: Commit

```bash
git add src/mnemonic/tests/e2e/
git commit -m "feat(pivot): E2E tests for admin API and MCP server"
```
