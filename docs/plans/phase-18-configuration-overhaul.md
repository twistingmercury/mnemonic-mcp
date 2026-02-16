# Phase 18: Configuration Overhaul

> Part of the [MVP Implementation Plan](mvp-implementation-plan.md)

**Goal:** Restructure `MnemonicConfig` for two listener configs (`server.admin` + `server.mcp`), remove routing config, and make Neo4j always-required (no `Enabled` toggle).

**Agent(s):** go-software-engineer

**Dependencies:** Phase 17 (routing code removed)

---

## Step 1: Add new config structs to config.go

- Modify file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/config/config.go`
- Add `ServerConfigs` struct with `Admin AdminServerConfig` and `MCP MCPServerConfig` fields
- Add `AdminServerConfig` struct (same fields as current `ServerConfig`: Host, Port, ReadTimeout, WriteTimeout, IdleTimeout, ShutdownTimeout, TLS)
- Add `MCPServerConfig` struct (same fields plus `SessionTimeout time.Duration`)
- Add `Address()` method to both `AdminServerConfig` and `MCPServerConfig`
- Agent: `go-software-engineer`
- Design reference: [Go Architecture Plan - Section 10](2026-02-15-go-architecture-plan.md#10-configuration-changes)

## Step 2: Update MnemonicConfig to use new server struct

- Modify file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/config/config.go`
- Change `Server ServerConfig` to `Server ServerConfigs`
- Remove `Routing RoutingConfig` field from `MnemonicConfig`
- Remove `RoutingConfig` struct
- Remove `RoutingCacheConfig` struct
- Remove `RoutingConfig.validate()` method
- Remove `c.Routing.validate()` call from `MnemonicConfig.Validate()`
- Agent: `go-software-engineer`

## Step 3: Update server validation

- Modify file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/config/config.go`
- Replace `c.Server.validate()` call with `c.Server.Admin.validate("server.admin")` and `c.Server.MCP.validate("server.mcp")`
- Refactor `ServerConfig.validate()` into a shared validation function that takes a field prefix string parameter, or create `validate()` methods on both `AdminServerConfig` and `MCPServerConfig`
- Add cross-validation: admin port must not equal MCP port (when both use the same host)
- Update the metrics port conflict check to reference `c.Server.Admin.Port` instead of `c.Server.Port`
- Remove old `ServerConfig.validate()` method
- Remove old `ServerConfig.Address()` method
- Agent: `go-software-engineer`

## Step 4: Add MCP session timeout validation

- Modify file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/config/config.go`
- In `MCPServerConfig.validate()`, validate that `SessionTimeout > 0`
- Agent: `go-software-engineer`

## Step 5: Update defaults

- Modify file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/config/defaults.go`
- Replace single `DefaultServer*` constants with:
  - `DefaultAdminHost = "0.0.0.0"`, `DefaultAdminPort = 8080`, `DefaultAdminReadTimeout = 30s`, `DefaultAdminWriteTimeout = 30s`, `DefaultAdminIdleTimeout = 120s`, `DefaultAdminShutdownTimeout = 5s`
  - `DefaultMCPHost = "0.0.0.0"`, `DefaultMCPPort = 8081`, `DefaultMCPReadTimeout = 30s`, `DefaultMCPWriteTimeout = 120s` (longer for SSE), `DefaultMCPIdleTimeout = 120s`, `DefaultMCPShutdownTimeout = 5s`, `DefaultMCPSessionTimeout = 30 * time.Minute`
- Remove `DefaultServerHost`, `DefaultServerPort`, `DefaultServerReadTimeout`, `DefaultServerWriteTimeout`, `DefaultServerIdleTimeout`, `DefaultServerShutdownTimeout`, `DefaultServerTLSEnabled`
- Remove `DefaultRoutingCacheRefreshTTL`, `DefaultRoutingCacheStartupTimeout`
- Agent: `go-software-engineer`

## Step 6: Update SetDefaults

- Modify file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/config/config.go`
- In `SetDefaults()`, replace `server.*` defaults with `server.admin.*` and `server.mcp.*` defaults
- Remove `routing.cache.*` defaults
- Agent: `go-software-engineer`

## Step 7: Update config tests -- test helper

- Modify file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/config/config_test.go`
- Update `validConfig()` helper: replace `Server: config.ServerConfig{...}` with `Server: config.ServerConfigs{Admin: config.AdminServerConfig{...}, MCP: config.MCPServerConfig{...}}`
- Remove `Routing: config.RoutingConfig{...}` from `validConfig()`
- Agent: `go-software-engineer`

## Step 8: Update config tests -- default values

- Modify file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/config/config_test.go`
- Update `TestDefaultValues` to assert against `cfg.Server.Admin.*` and `cfg.Server.MCP.*` instead of `cfg.Server.*`
- Remove routing default assertions (`cfg.Routing.Cache.RefreshTTL`, etc.)
- Add assertion for `cfg.Server.MCP.SessionTimeout`
- Agent: `go-software-engineer`

## Step 9: Update config tests -- validation tests

- Modify file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/config/config_test.go`
- Update `TestValidation_ServerConfig` tests to modify `cfg.Server.Admin.*` fields (update field paths in `expectError` strings to `server.admin.*`)
- Add new test cases for MCP server validation (port, timeouts, session_timeout)
- Add test for admin port == MCP port conflict
- Remove `TestValidation_RoutingConfig` entirely
- Update `TestValidation_PortConflict` to check admin port vs metrics port (not `cfg.Server.Port`)
- Agent: `go-software-engineer`

## Step 10: Update config tests -- YAML loading

- Modify file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/config/config_test.go`
- Update `TestYAMLFileLoading` config content to use `server.admin.*` and `server.mcp.*` structure
- Update assertions accordingly
- Agent: `go-software-engineer`

## Step 11: Update config tests -- env var tests

- Modify file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/config/config_test.go`
- Update `TestEnvironmentVariableOverrides` to use `MNEMONIC_SERVER_ADMIN_HOST` instead of `MNEMONIC_SERVER_HOST`
- Update `TestEnvironmentVariableNaming` test cases (e.g., `MNEMONIC_SERVER_ADMIN_PORT`, `MNEMONIC_SERVER_MCP_PORT`)
- Update `TestServerAddress` to use `AdminServerConfig` instead of `ServerConfig`
- Add test for `MCPServerConfig.Address()`
- Update `TestDurationEnvironmentVariables` to reference `cfg.Server.Admin.ReadTimeout`
- Agent: `go-software-engineer`

## Step 12: Update server.go -- fix imports

- Modify file: `/Users/doublej/dev/mnemonic/src/mnemonic/internal/server/server.go`
- Update `ListenAndServe` to use `cfg.Server.Admin.Address()` instead of `cfg.Server.Address()`
- Update `CreateHTTPServer` to use `cfg.Server.Admin.*` timeout fields
- Update startup log to reference `cfg.Server.Admin.Host` and `cfg.Server.Admin.Port`
- Note: Full server rewrite happens in Phase 25; this step only fixes compilation against the new config types
- Agent: `go-software-engineer`

## Step 13: Verify build and tests pass

- Run: `cd /Users/doublej/dev/mnemonic/src/mnemonic && go build ./...`
- Run: `cd /Users/doublej/dev/mnemonic/src/mnemonic && go vet ./...`
- Run: `cd /Users/doublej/dev/mnemonic/src/mnemonic && go test ./internal/config/...` -- all config tests pass
- Run: `cd /Users/doublej/dev/mnemonic/src/mnemonic && go test ./...` -- all tests pass
- Agent: `go-software-engineer`

## Step 14: Commit

```bash
git add src/mnemonic/internal/config/ src/mnemonic/internal/server/server.go
git commit -m "feat(pivot): restructure config for two listeners (admin+mcp), remove routing config"
```
