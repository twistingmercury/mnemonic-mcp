# Phase 25: Server Lifecycle

> Part of the [MVP Implementation Plan](mvp-implementation-plan.md)

**Goal:** Rewrite `internal/server/` to start both REST Admin and MCP listeners in one process. Create new `cmd/mnemonic/main.go` entrypoint.

**Agent(s):** go-software-engineer

**Dependencies:** Phase 18 (config overhaul), Phase 23 (all handlers), Phase 24 (MCP server)

---

## Step 1: Rewrite server.go for two listeners

- Rewrite file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/server/server.go`
- `Run(cfg *config.MnemonicConfig) error`:
  - Initialize telemetry
  - Initialize Postgres pool (fail if unreachable)
  - Initialize Neo4j driver (fail if unreachable)
  - Create all repositories
  - Create router via `handlers.NewRouter()`
  - Create MCP handler via `mcpserver.NewHandler()`
  - Start admin `http.Server` on `cfg.Server.Admin.Address()`
  - Start MCP `http.Server` on `cfg.Server.MCP.Address()` (write timeout from config, default 120s)
  - Wait for SIGINT/SIGTERM or server error
  - Graceful shutdown of both listeners
  - Close database connections
- Agent: `go-software-engineer`
- Design reference: [Go Architecture Plan - Server Lifecycle](2026-02-15-go-architecture-plan.md#6-package-structure)

## Step 2: Create new main entrypoint

- Create file: `/Users/doublej/dev/mnemonic/src/mnemonic/cmd/mnemonic/main.go`
- Same structure as current `cmd/main/main.go` but calls `server.Run(cfg)` (new function name)
- Keep `--version` and `--health` flags
- Agent: `go-software-engineer`

## Step 3: Update old main.go (preserve for backward compat during transition)

- Modify file: `/Users/doublej/dev/mnemonic/src/mnemonic/cmd/main/main.go`
- Either: forward to the new `server.Run()`, or mark as deprecated
- Agent: `go-software-engineer`

## Step 4: Update health check initialization

- Modify file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/health/health.go` (if changes needed)
- Health checks must cover: Postgres connectivity, Neo4j connectivity, PGVector extension presence
- Both listeners share the same health state
- Agent: `go-software-engineer`

## Step 5: Verify build and tests

- Run: `cd /Users/doublej/dev/mnemonic/src/mnemonic && go build ./cmd/mnemonic/...`
- Run: `cd /Users/doublej/dev/mnemonic/src/mnemonic && go build ./...`
- Run: `cd /Users/doublej/dev/mnemonic/src/mnemonic && go test ./...`
- Agent: `go-software-engineer`

## Step 6: Manual smoke test

- Start local infrastructure: `docker-compose up -d postgres neo4j`
- Run: `cd /Users/doublej/dev/mnemonic/src/mnemonic && go run ./cmd/mnemonic/`
- Verify admin API: `curl http://localhost:8080/ops/health` returns 200
- Verify admin API: `curl http://localhost:8080/ops/version` returns version JSON
- Verify admin API: `curl http://localhost:8080/api/agents` returns empty list
- Verify MCP endpoint: `curl -X POST http://localhost:8081/mcp -H 'Content-Type: application/json' -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"0.1"}}}'` returns MCP initialize response
- Send SIGINT (Ctrl+C), verify graceful shutdown log messages
- Agent: `go-software-engineer`

## Step 7: Commit

```bash
git add src/mnemonic/internal/server/ src/mnemonic/cmd/mnemonic/ src/mnemonic/cmd/main/ src/mnemonic/internal/health/
git commit -m "feat(pivot): rewrite server lifecycle for dual listeners (admin :8080 + mcp :8081)"
```
