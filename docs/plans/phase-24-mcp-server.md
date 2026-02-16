# Phase 24: MCP Server Implementation

> Part of the [MVP Implementation Plan](mvp-implementation-plan.md)

**Goal:** Implement the MCP server using the official Go SDK with all read-only tools for Claude Code integration.

**Agent(s):** go-software-engineer

**Dependencies:** Phase 19 (agent repo), Phase 20 (skill repo), Phase 21 (command repo)

---

## Step 1: Add MCP SDK dependency

- Run: `cd /Users/doublej/dev/mnemonic/src/mnemonic && go get github.com/modelcontextprotocol/go-sdk`
- Verify it appears in `go.mod`
- Agent: `go-software-engineer`

## Step 2: Create MCP server setup

- Create file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/mcpserver/server.go`
- Define `Deps` struct holding all read-only dependencies: `Patterns pattern.Repository`, `Agents agent.Repository`, `Skills skill.Repository`, `Commands command.Repository`, `Graph graph.Repository`
- Define `NewHandler(deps *Deps) http.Handler` that creates an `mcp.Server`, registers all tools, and returns `mcp.NewStreamableHTTPHandler()`
- Use `mcp.Implementation{Name: "mnemonic", Version: version.Version}`
- Session timeout configurable (pass from config)
- Agent: `go-software-engineer`
- Design reference: [Go Architecture Plan - MCP Server Design](2026-02-15-go-architecture-plan.md#5-mcp-server-design)

## Step 3: Create search tools

- Create file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/mcpserver/tools_search.go`
- Register `search_patterns` tool: accepts query, limit, threshold, tags, agent params. MVP: falls back to full-text search via `pattern.Repository.List()`.
- Register `find_related_patterns` tool: accepts pattern_id, limit. Calls `graph.Repository.FindRelatedPatterns()`.
- Agent: `go-software-engineer`
- Design reference: [API Specification - search_patterns](../design/2026-02-15-pivot-api-specification.md#33-tool-search_patterns)

## Step 4: Create pattern tools

- Create file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/mcpserver/tools_patterns.go`
- Register `get_pattern` (by ID, includes graph context)
- Agent: `go-software-engineer`
- Design reference: [API Specification - get_pattern](../design/2026-02-15-pivot-api-specification.md#34-tool-get_pattern)

## Step 5: Create agent tools

- Create file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/mcpserver/tools_agents.go`
- Register `get_agent` (by name) and `list_agents` (returns all agents with collection_version hash)
- Agent: `go-software-engineer`
- Design reference: [API Specification - list_agents](../design/2026-02-15-pivot-api-specification.md#35-tool-list_agents)

## Step 6: Create skill tools

- Create file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/mcpserver/tools_skills.go`
- Register `get_skill` (by name) -- unmarshals JSONB `definition` to return full skill data
- Register `list_skills` (all skills with collection_version)
- Register `get_skill_files` (by skill name) -- queries `skill_files` table and unmarshals JSONB `document` field
- Tool responses transform JSONB data to expected API response format
- Agent: `go-software-engineer`
- Design reference: [API Specification - list_skills](../design/2026-02-15-pivot-api-specification.md#36-tool-list_skills)

## Step 7: Create command tools

- Create file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/mcpserver/tools_commands.go`
- Register `get_command` (by name) -- unmarshals JSONB `definition` to return full command data
- Register `list_commands` (all commands with collection_version)
- Tool responses transform JSONB data to expected API response format
- Agent: `go-software-engineer`
- Design reference: [API Specification - list_commands](../design/2026-02-15-pivot-api-specification.md#37-tool-list_commands)

## Step 8: Create sync manifest tool

- Add to `/Users/doublej/dev/mnemonic/src/mnemonic/internal/mcpserver/tools_search.go` (or a new `tools_sync.go`)
- Register `get_sync_manifest` tool: returns collection version hashes for agents, skills, commands
- Can optionally use CRC64 values from entity tables for efficient change detection
- Compute hash: `SHA-256(count || max(updated_at) || min(created_at))` truncated to 12 hex chars
- Agent: `go-software-engineer`
- Design reference: [API Specification - get_sync_manifest](../design/2026-02-15-pivot-api-specification.md#311-tool-get_sync_manifest)

## Step 9: Verify build compiles

- Run: `cd /Users/doublej/dev/mnemonic/src/mnemonic && go build ./internal/mcpserver/...`
- Agent: `go-software-engineer`

## Step 10: Write unit tests

- Create file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/mcpserver/server_test.go`
- Test that `NewHandler` creates a valid `http.Handler`
- Test each tool handler with mocked repository dependencies
- Verify tool responses match expected format (text content, isError for not-found cases)
- Agent: `go-software-engineer`

## Step 11: Run tests

- Run: `cd /Users/doublej/dev/mnemonic/src/mnemonic && go test -v ./internal/mcpserver/...`
- Run: `cd /Users/doublej/dev/mnemonic/src/mnemonic && go test ./...` -- no regressions
- Agent: `go-software-engineer`

## Step 12: Commit

```bash
git add src/mnemonic/internal/mcpserver/ src/mnemonic/go.mod src/mnemonic/go.sum
git commit -m "feat(pivot): implement MCP server with read-only tools (search, patterns, agents, skills, commands, sync)"
```
