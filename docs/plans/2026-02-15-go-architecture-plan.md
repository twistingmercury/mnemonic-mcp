# Go Architecture Plan: Mnemonic Pivot to Knowledge Sync (Revised)

**Date:** 2026-02-15 (Revised)
**Author:** Go Software Architect
**Status:** Proposed
**Revision:** 3 -- simplifications per owner review (single server, no CLI, Neo4j required)
**Inputs:**

- Pivot proposal: `docs/plans/2026-02-14-mnemonic-pivot-knowledge-sync.md`
- Solutions architect review: `docs/architecture/2026-02-15-pivot-review.md`
- Owner's architectural direction: single server with REST admin + MCP endpoint
- Current MVP plan: `docs/plans/mvp-implementation-plan.md` (Phases 1-14 complete)

---

## Table of Contents

1. [Architecture Overview](#1-architecture-overview)
2. [Codebase Inventory](#2-codebase-inventory)
3. [Package Disposition: What Stays, Changes, Goes](#3-package-disposition)
4. [Neo4j Assessment](#4-neo4j-assessment)
5. [MCP Server Design](#5-mcp-server-design)
6. [Package Structure](#6-package-structure)
7. [Migration Ordering](#7-migration-ordering)
8. [Schema Changes for Existing Tables](#8-schema-changes-for-existing-tables)
9. [New Package Design](#9-new-package-design)
10. [Configuration Changes](#10-configuration-changes)
11. [Phased Implementation Plan](#11-phased-implementation-plan)
12. [Open Decisions](#12-open-decisions)

---

## 1. Architecture Overview

Mnemonic is a single server that exposes two protocols: a REST API for data management and an MCP endpoint for Claude Code integration. Data is loaded via `curl` against the REST API. Claude Code discovers and invokes read-only tools via the MCP endpoint.

### Single Server, Two Protocols

```
                                           +---------------------+
                                           |   PostgreSQL        |
                                           |   + PGVector        |
                                           |                     |
                                           |  READ + WRITE       |
                                           +----------+----------+
                                                      ^
                                                      |
                                           +----------+----------+
                                           |   Neo4j             |
                                           |   (Knowledge Graph) |
                                           +----------+----------+
                                                      ^
                                                      |
                            +-------------------------+-------------------------+
                            |                                                   |
                  +---------+---------+                              +----------+---------+
                  | REST Admin API    |                              | MCP Endpoint       |
                  | (Gin)             |                              | (Streamable HTTP)  |
                  | Port :8080        |                              | Port :8081         |
                  |                   |                              |                    |
                  | READ + WRITE      |                              | READ ONLY          |
                  +---------+---------+                              +----------+---------+
                            ^                                                   ^
                            |                                                   |
                  +---------+---------+                              +----------+---------+
                  | curl / httpie     |                              | Claude Code        |
                  | (data loading)    |                              | (MCP client)       |
                  +-------------------+                              +--------------------+
```

**REST Admin API:**

- Protocol: REST over HTTP (Gin)
- Client: `curl` or any HTTP client
- Access: READ + WRITE to database
- Purpose: Managing patterns, agents, skills, commands -- data ingestion
- Post-MVP: Envoy + OPA sidecar for auth

**MCP Endpoint:**

- Protocol: MCP over Streamable HTTP (official Go SDK)
- Client: Claude Code (as MCP server connection)
- Access: READ ONLY to database
- Purpose: Knowledge retrieval, semantic search, tooling sync
- Post-MVP: Envoy + OPA sidecar for auth

### Key Differences from Previous Plan

| Aspect                       | Previous Plan                              | Revised Plan                                  |
| ---------------------------- | ------------------------------------------ | --------------------------------------------- |
| Claude Code integration      | REST API + shell script wrappers in skills | MCP server -- Claude calls tools directly     |
| Service topology             | Single REST API                            | Single server with two listeners (REST + MCP) |
| Data loading                 | `cmd/seed/` standalone binary              | `curl` against the REST Admin API             |
| `pattern_agent_associations` | Dropped (migration 009)                    | **Kept** -- needed for agent-scoped filtering |
| Skills like `/recall`        | REST call + parse response                 | MCP tool call -- native to Claude Code        |
| Neo4j                        | Optional with config toggle                | **Required** -- always enabled                |

---

## 2. Codebase Inventory

Before making architectural decisions, here is an honest accounting of what exists today.

### Lines of Code by Package (production + test)

| Package                              | Production LOC                                    | Test LOC | Total   | Status                             |
| ------------------------------------ | ------------------------------------------------- | -------- | ------- | ---------------------------------- |
| `internal/routing/`                  | ~825                                              | ~3,389   | 4,214   | **REMOVE**                         |
| `internal/repository/routingrule/`   | ~634                                              | ~1,139   | 1,773   | **REMOVE**                         |
| `internal/repository/agent/`         | ~330                                              | ~tests   | ~600+   | **MODIFY**                         |
| `internal/repository/pattern/`       | ~765                                              | ~tests   | ~1,200+ | **MODIFY** (keep agent assoc)      |
| `internal/repository/graph/`         | ~647                                              | ~tests   | ~1,000+ | **KEEP**                           |
| `internal/repository/enrichmentjob/` | ~113                                              | ~tests   | ~400+   | **KEEP**                           |
| `internal/config/`                   | ~400+                                             | ~tests   | ~600+   | **MODIFY** (two listeners)         |
| `internal/metrics/`                  | routing: 93, patterns: ~80, db: ~80, registry: 41 | ~tests   | ~500+   | **MODIFY**                         |
| `internal/middleware/`               | ~100+                                             | ~tests   | ~200+   | **KEEP**                           |
| `internal/server/`                   | ~140                                              | -        | ~140    | **REWRITE** (two listeners)        |
| `internal/telemetry/`                | ~100+                                             | ~tests   | ~200+   | **KEEP**                           |
| `internal/handlers/`                 | stubs only                                        | -        | ~200    | **REPLACE**                        |
| `internal/health/`                   | ~50                                               | -        | ~50     | **KEEP**                           |
| `internal/version/`                  | ~30                                               | -        | ~30     | **KEEP**                           |

### What Is Actually Built vs. Designed

| Component           | Schema      | Repository         | Handler     | Service Logic | Pipeline  |
| ------------------- | ----------- | ------------------ | ----------- | ------------- | --------- |
| Agents              | Yes (002)   | Yes, tested        | Stub (501)  | No            | N/A       |
| Patterns            | Yes (003)   | Yes, tested        | Stub (501)  | No            | N/A       |
| Pattern-Agent Assoc | Yes (004)   | In pattern repo    | N/A         | No            | N/A       |
| Routing Rules       | Yes (005)   | Yes, tested        | Stub (501)  | No            | N/A       |
| Enrichment Jobs     | Yes (006)   | Yes, tested        | N/A         | No            | Not built |
| Performance Indexes | Yes (007)   | N/A                | N/A         | N/A           | N/A       |
| Routing Engine      | N/A         | N/A                | N/A         | Yes, tested   | N/A       |
| Neo4j Graph         | Constraints | Yes, tested        | N/A         | No            | Not built |
| Semantic Search     | N/A         | `FindSimilar` impl | No endpoint | No            | Not built |

**Key observation:** The handler layer is entirely stubs returning 501. All business logic resides in the routing engine and repository layers. The pivot discards the routing engine and builds the knowledge features that were always planned.

---

## 3. Package Disposition

### REMOVE -- Delete Entirely

| Package / File                                        | Reason                                              | LOC Removed |
| ----------------------------------------------------- | --------------------------------------------------- | ----------- |
| `internal/routing/` (entire directory)                | Routing engine, matchers, cache -- no longer needed | 4,214       |
| `internal/repository/routingrule/` (entire directory) | Routing rule data model and repository              | 1,773       |
| `internal/handlers/routes/` (entire directory)        | Route handler stubs (`POST /api/route`)             | ~20         |
| `internal/handlers/routes/rules/` (entire directory)  | Routing rules CRUD handler stubs                    | ~48         |
| `internal/metrics/routing.go`                         | Routing-specific metrics                            | 93          |
| `internal/metrics/routing_test.go`                    | Tests for routing metrics                           | ~120        |
| `tests/e2e/routing_test.go`                           | E2E test scenarios for routing                      | TBD         |
| `tests/e2e/routing_rules_test.go`                     | E2E test scenarios for routing rules                | TBD         |

**Total removed: ~6,300+ lines** (the majority of application logic built in Phases 9-14).

### KEEP -- No Changes Required

| Package                              | Reason                                                                  |
| ------------------------------------ | ----------------------------------------------------------------------- |
| `internal/middleware/`               | Request metrics and tracing middleware are API-agnostic                 |
| `internal/telemetry/`                | Telemetry initialization is infrastructure, not routing-specific        |
| `internal/health/`                   | Health check framework stays; dependency list will evolve               |
| `internal/version/`                  | Build version reporting is unchanged                                    |
| `internal/repository/enrichmentjob/` | Enrichment job queue supports the pattern enrichment pipeline           |
| `internal/repository/graph/`         | Neo4j graph repository -- all operations remain valid                   |
| `internal/repository/repository.go`  | `DBTX`, `ListOptions`, `TxBeginner` interfaces -- shared infrastructure |
| `internal/repository/pgerrors.go`    | Postgres error code constants -- shared infrastructure                  |
| `cmd/version/`                       | Version subcommand                                                      |

### MODIFY -- Changes Required

| Package                        | What Changes                                                                                                | Scope                   |
| ------------------------------ | ----------------------------------------------------------------------------------------------------------- | ----------------------- |
| `internal/repository/agent/`   | Add `version` column support to model and repository; deprecate `RoutingKeywords` in struct                 | Small                   |
| `internal/repository/pattern/` | **Keep** `SetAgentAssociations` / `GetAgentAssociations` methods -- still needed for agent-scoped filtering | Small (no code removal) |
| `internal/config/`             | Remove `RoutingConfig`; add two-listener config (`server.admin`, `server.mcp`)                              | Medium                  |
| `internal/metrics/registry.go` | Remove `Routing` field from `Registry`; add new metric domains                                              | Small                   |
| `internal/server/`             | Rewrite: two HTTP listeners in one process                                                                  | Full rewrite            |
| `internal/handlers/agents/`    | Rewrite with dependency injection                                                                           | Full rewrite            |
| `internal/handlers/patterns/`  | Rewrite with dependency injection                                                                           | Full rewrite            |
| `tests/e2e/`                   | Rewrite for new API surface; remove routing tests                                                           | Full rewrite            |

### CREATE -- New Packages

| Package                        | Purpose                                                  | Details   |
| ------------------------------ | -------------------------------------------------------- | --------- |
| `cmd/mnemonic/`                | Main binary entrypoint (runs both listeners)             | Section 6 |
| `internal/handlers/skills/`    | Skill CRUD handlers                                      | Section 9 |
| `internal/handlers/commands/`  | Command CRUD handlers                                    | Section 9 |
| `internal/handlers/search/`    | Search endpoint handler                                  | Section 9 |
| `internal/mcpserver/`          | MCP endpoint tools and setup (read-only operations)      | Section 5 |
| `internal/repository/skill/`   | Skill storage and retrieval (Postgres)                   | Section 9 |
| `internal/repository/command/` | Command storage and retrieval (Postgres)                 | Section 9 |
| `internal/service/search/`     | Search orchestration (PGVector + Neo4j graph context)    | Section 9 |

---

## 4. Neo4j Assessment

### Current Neo4j Usage

The graph repository (`/Users/doublej/dev/mnemonic/src/mnemonic/internal/repository/graph/repository.go`) implements these operations:

| Operation                  | Purpose                                        | Post-Pivot Value                           |
| -------------------------- | ---------------------------------------------- | ------------------------------------------ |
| `SyncAgent`                | Create/update Agent node                       | Agent-pattern graph                        |
| `DeleteAgent`              | Remove Agent node                              | Same                                       |
| `SyncPattern`              | Create/update Pattern node                     | **Core value** -- pattern graph navigation |
| `DeletePattern`            | Remove Pattern node                            | Same                                       |
| `SyncConcepts`             | Link Concept nodes to Pattern via MENTIONED_IN | **Core value** -- concept extraction       |
| `SetPatternAgentRelevance` | Link Pattern to Agent via RELEVANT_FOR         | Agent-scoped search                        |
| `FindRelatedPatterns`      | Traverse shared concepts between patterns      | **Core value** -- graph context            |
| `FindPatternsByAgent`      | Find patterns relevant to an agent             | Agent-scoped search                        |
| `CleanupOrphanedConcepts`  | Garbage collection                             | Operational                                |
| `HealthCheck`              | Connectivity check                             | Operational                                |

### Neo4j Is Required

Neo4j is a required dependency alongside Postgres. The concept-to-concept traversals that power search graph context are the differentiator between "search a database" and "navigate a knowledge graph." The pastor-sermon-archive use case (cited by the owner) demonstrates Neo4j's value for discovering non-obvious relationships between concepts.

Neo4j is initialized at startup alongside Postgres. If Neo4j is unreachable, the server fails to start -- the same behavior as a Postgres connection failure.

---

## 5. MCP Server Design

### What Is MCP over Streamable HTTP

The Model Context Protocol (MCP) is a JSON-RPC 2.0 based protocol that allows LLM applications (like Claude Code) to discover and invoke **tools**, **resources**, and **prompts** exposed by an MCP server. The "Streamable HTTP" transport sends JSON-RPC messages over HTTP POST with optional SSE streaming for server-initiated messages.

The official Go SDK (`github.com/modelcontextprotocol/go-sdk`) provides `mcp.StreamableHTTPHandler`, which is a standard `http.Handler`. This means the MCP endpoint can be mounted on its own `http.Server` listening on a dedicated port, completely independent of the Gin-based REST admin API.

### SDK Choice

**Official SDK: `github.com/modelcontextprotocol/go-sdk/mcp`** (3,854+ stars, maintained by Google + Anthropic)

This is the canonical choice. It supports:

- Streamable HTTP transport (`mcp.StreamableHTTPHandler`)
- Typed tool handlers with input validation (`mcp.AddTool[In, Out]`)
- Resources and prompts
- Session management with configurable timeouts
- MCP spec versions 2024-11-05 through 2025-06-18

The alternative `mark3labs/mcp-go` (8,171+ stars) is more mature in the community but the official SDK is the correct long-term choice given it is maintained by the protocol authors.

### MCP Server Architecture

```go
// internal/mcpserver/server.go

package mcpserver

import (
    "net/http"
    "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Deps holds the read-only dependencies for all MCP tools.
type Deps struct {
    Patterns     pattern.Repository
    Agents       agent.Repository
    Skills       skill.Repository
    Commands     command.Repository
    Search       search.Service
    Graph        graph.Repository
}

// NewHandler creates the MCP HTTP handler with all tools registered.
func NewHandler(deps *Deps) http.Handler {
    server := mcp.NewServer(
        &mcp.Implementation{Name: "mnemonic", Version: version.Version},
        nil,
    )

    // Register all tools (11 total)
    registerSearchTools(server, deps)       // search_patterns, find_related_patterns
    registerPatternTools(server, deps)       // get_pattern
    registerAgentTools(server, deps)         // list_agents, get_agent
    registerSkillTools(server, deps)         // list_skills, get_skill
    registerCommandTools(server, deps)       // list_commands, get_command
    registerSyncTools(server, deps)          // get_sync_manifest
    registerSkillFileTools(server, deps)    // get_skill_files

    return mcp.NewStreamableHTTPHandler(
        func(r *http.Request) *mcp.Server { return server },
        &mcp.StreamableHTTPOptions{
            SessionTimeout: 30 * time.Minute,
        },
    )
}
```

### MCP Tools Inventory

These tools replace what was previously envisioned as REST endpoints + shell script wrappers in skills. Claude Code calls these directly as an MCP client.

#### Search Tools

| Tool Name               | Description                                         | Parameters                                                                                               | Maps To                                  |
| ----------------------- | --------------------------------------------------- | -------------------------------------------------------------------------------------------------------- | ---------------------------------------- |
| `search_patterns`       | Semantic search across patterns                     | `query: string`, `max_results?: int`, `min_similarity?: float`, `tags?: []string`, `agent_name?: string` | `search.Service.Search()`                |
| `find_related_patterns` | Find patterns sharing concepts with a given pattern | `pattern_id: string`, `limit?: int`                                                                      | `graph.Repository.FindRelatedPatterns()` |

#### Pattern Tools (read-only)

| Tool Name       | Description                  | Parameters                                                          | Maps To                                    |
| --------------- | ---------------------------- | ------------------------------------------------------------------- | ------------------------------------------ |
| `get_pattern`   | Get a pattern by ID or name  | `id?: string`, `name?: string`                                      | `pattern.Repository.Get()` / `GetByName()` |

#### Agent Tools (read-only)

| Tool Name     | Description             | Parameters                    | Maps To                   |
| ------------- | ----------------------- | ----------------------------- | ------------------------- |
| `get_agent`   | Get an agent definition | `name: string`                | `agent.Repository.Get()`  |
| `list_agents` | List all agents         | `limit?: int`, `offset?: int` | `agent.Repository.List()` |

#### Skill Tools (read-only)

| Tool Name     | Description         | Parameters                                       | Maps To                        |
| ------------- | ------------------- | ------------------------------------------------ | ------------------------------ |
| `get_skill`   | Get a skill by name | `name: string`                                   | `skill.Repository.GetByName()` |
| `list_skills` | List all skills     | `tags?: []string`, `limit?: int`, `offset?: int` | `skill.Repository.List()`      |

#### Command Tools (read-only)

| Tool Name       | Description           | Parameters                                       | Maps To                          |
| --------------- | --------------------- | ------------------------------------------------ | -------------------------------- |
| `get_command`   | Get a command by name | `name: string`                                   | `command.Repository.GetByName()` |
| `list_commands` | List all commands     | `tags?: []string`, `limit?: int`, `offset?: int` | `command.Repository.List()`      |

#### Sync Tools (read-only)

| Tool Name            | Description                            | Parameters | Maps To                                    |
| -------------------- | -------------------------------------- | ---------- | ------------------------------------------ |
| `get_sync_manifest`  | Get collection versions for sync check | --         | Collection version calculation across repos |

### How MCP Replaces REST + Skills

Previously, Claude Code would use skills like `/recall` which internally ran a shell script that called `curl` against the REST API. With MCP:

| Before (REST + Skills)                                                                                                                                         | After (MCP)                                                                                                  |
| -------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------ |
| `/recall "error handling"` triggers skill, skill runs `scripts/search.sh`, script calls `GET /v1/api/search?q=...`, script parses JSON, skill presents results | Claude Code calls `search_patterns(query: "error handling")` directly -- no skill, no script, no HTTP client |
| `/mnemonic-sync` triggers skill, skill runs `scripts/sync.sh`, script calls multiple REST endpoints, script writes files                                       | Claude Code calls `list_agents()`, `list_skills()`, `list_commands()` directly to read definitions           |
| Adding a pattern requires the `/remember` skill with a shell script that POSTs to REST                                                                         | Adding a pattern is done via `curl` against the Admin API                                                    |

**The `/recall` and `/remember` skills become unnecessary.** Claude Code natively discovers and invokes the MCP tools. The MCP tool descriptions serve as the "skill instructions" -- they tell Claude what the tool does and what parameters it accepts.

**The `/mnemonic-sync` skill is simplified.** It can still exist as a convenience skill, but it calls MCP tools directly instead of shell scripts.

### Tool Handler Example

```go
// internal/mcpserver/tools_search.go

package mcpserver

import (
    "context"
    "github.com/modelcontextprotocol/go-sdk/mcp"
    "github.com/twistingmercury/mnemonic/internal/service/search"
)

type SearchInput struct {
    Query         string   `json:"query" jsonschema:"description=Natural language search query,required"`
    MaxResults    int      `json:"max_results,omitempty" jsonschema:"description=Maximum results to return,minimum=1,maximum=50"`
    MinSimilarity float64  `json:"min_similarity,omitempty" jsonschema:"description=Minimum similarity threshold (0.0-1.0)"`
    Tags          []string `json:"tags,omitempty" jsonschema:"description=Filter by tags"`
    AgentName     string   `json:"agent_name,omitempty" jsonschema:"description=Filter to patterns relevant to this agent"`
}

type SearchOutput struct {
    Results []SearchResult `json:"results"`
    Total   int            `json:"total"`
}

type SearchResult struct {
    PatternID   string   `json:"pattern_id"`
    Name        string   `json:"name"`
    Description string   `json:"description,omitempty"`
    Content     string   `json:"content"`
    Similarity  float64  `json:"similarity"`
    Tags        []string `json:"tags"`
}

func registerSearchTools(server *mcp.Server, deps *Deps) {
    searchHandler := makeSearchHandler(deps)
    mcp.AddTool(server, &mcp.Tool{
        Name:        "search_patterns",
        Description: "Search the team knowledge graph for patterns matching a natural language query. Returns ranked results by semantic similarity.",
    }, searchHandler)
}

func makeSearchHandler(deps *Deps) mcp.ToolHandlerFor[SearchInput, SearchOutput] {
    return func(ctx context.Context, req *mcp.CallToolRequest, input SearchInput) (*mcp.CallToolResult, SearchOutput, error) {
        maxResults := input.MaxResults
        if maxResults == 0 {
            maxResults = 10
        }
        minSim := input.MinSimilarity
        if minSim == 0 {
            minSim = 0.7
        }

        results, err := deps.Search.Search(ctx, input.Query, search.SearchOptions{
            MaxResults:    maxResults,
            MinSimilarity: minSim,
            Tags:          input.Tags,
            AgentName:     input.AgentName,
            IncludeGraph:  true,
        })
        if err != nil {
            return nil, SearchOutput{}, err
        }

        out := SearchOutput{Total: len(results)}
        for _, r := range results {
            desc := ""
            if r.Pattern.Description != nil {
                desc = *r.Pattern.Description
            }
            out.Results = append(out.Results, SearchResult{
                PatternID:   r.Pattern.ID.String(),
                Name:        r.Pattern.Name,
                Description: desc,
                Content:     r.Pattern.Content,
                Similarity:  r.Similarity,
                Tags:        r.Pattern.Tags,
            })
        }

        return nil, out, nil
    }
}
```

### Claude Code Configuration

To connect Claude Code to Mnemonic's MCP server, the user adds this to their Claude Code MCP settings:

```json
{
  "mcpServers": {
    "mnemonic": {
      "type": "streamable-http",
      "url": "http://localhost:8081/mcp"
    }
  }
}
```

Claude Code then automatically discovers all available tools via the MCP `tools/list` method and can invoke them as needed during conversations.

---

## 6. Package Structure

```
src/mnemonic/
+-- cmd/
|   +-- mnemonic/                    # Main binary entrypoint
|       +-- main.go                  # Initializes server, runs it
+-- internal/
|   +-- handlers/                    # REST Admin API handlers
|   |   +-- agents/                  # Agent CRUD handlers
|   |   |   +-- agents.go
|   |   +-- patterns/                # Pattern CRUD handlers
|   |   |   +-- patterns.go
|   |   +-- skills/                  # Skill CRUD handlers
|   |   |   +-- skills.go
|   |   +-- commands/                # Command CRUD handlers
|   |   |   +-- commands.go
|   |   +-- search/                  # Search endpoint handler
|   |   |   +-- search.go
|   |   +-- router.go                # Gin router setup
|   +-- mcpserver/                   # MCP endpoint (read-only tools)
|   |   +-- server.go                # MCP server setup + handler creation
|   |   +-- tools_search.go          # search_patterns, find_related_patterns
|   |   +-- tools_patterns.go        # get_pattern
|   |   +-- tools_agents.go          # get_agent, list_agents
|   |   +-- tools_skills.go          # get_skill, list_skills
|   |   +-- tools_commands.go        # get_command, list_commands
|   |   +-- tools_sync.go            # get_sync_manifest
|   +-- repository/                  # Data access layer
|   |   +-- agent/                   # Agent repository (existing, modified)
|   |   +-- pattern/                 # Pattern repository (existing, keep assoc methods)
|   |   +-- skill/                   # Skill repository (new)
|   |   +-- command/                 # Command repository (new)
|   |   +-- graph/                   # Neo4j graph repository (existing, kept)
|   |   +-- enrichmentjob/           # Enrichment jobs (existing, kept)
|   |   +-- repository.go            # Shared interfaces (DBTX, ListOptions)
|   |   +-- pgerrors.go              # PG error constants
|   +-- service/
|   |   +-- search/                  # Search orchestration service
|   |       +-- search.go
|   +-- config/                      # Configuration (two listeners)
|   +-- server/                      # Server lifecycle (start both listeners, graceful shutdown)
|   +-- middleware/                   # HTTP middleware
|   +-- telemetry/                   # OTel setup
|   +-- metrics/                     # Metrics registry
|   +-- health/                      # Health checks
|   +-- version/                     # Build version
+-- tests/
    +-- e2e/                         # End-to-end tests
```

The package structure is organized by concern: handlers for REST, mcpserver for MCP tools, repository for data access, service for business logic, and infrastructure packages (config, server, middleware, telemetry, metrics, health, version) at the top level of `internal/`.

### Router Setup (`internal/handlers/router.go`)

```go
// internal/handlers/router.go
package handlers

func NewRouter(repos *Repositories, searchSvc search.Service, tel *telemetry.Telemetry) *gin.Engine {
    router := gin.New()
    router.Use(gin.Recovery())
    // ... middleware setup (tracing, metrics, logging)

    // Operations endpoints (no version prefix)
    operations.SetupHandlers(router)

    // Admin API group (with version prefix)
    v1 := router.Group("/v1/api")

    agents.NewHandler(repos.Agents).SetupRoutes(v1)
    patterns.NewHandler(repos.Patterns).SetupRoutes(v1)
    skills.NewHandler(repos.Skills).SetupRoutes(v1)
    commands.NewHandler(repos.Commands).SetupRoutes(v1)
    searchHandler.NewHandler(searchSvc).SetupRoutes(v1)

    return router
}
```

### Server Lifecycle (`internal/server/server.go`)

```go
// internal/server/server.go

package server

import (
    "context"
    "fmt"
    "net/http"
    "os/signal"
    "syscall"

    "github.com/twistingmercury/mnemonic/internal/config"
    "github.com/twistingmercury/mnemonic/internal/handlers"
    "github.com/twistingmercury/mnemonic/internal/mcpserver"
    "github.com/twistingmercury/mnemonic/internal/telemetry"
)

// Run starts the Mnemonic server (REST + MCP listeners), blocking until shutdown.
func Run(cfg *config.MnemonicConfig) error {
    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    // Initialize telemetry
    tel, err := telemetry.Initialize(ctx, cfg)
    if err != nil {
        return fmt.Errorf("telemetry init: %w", err)
    }
    defer tel.Shutdown(context.Background())

    // Initialize database connections
    pgPool, err := initPostgres(ctx, cfg)
    if err != nil {
        return fmt.Errorf("postgres init: %w", err)
    }
    defer pgPool.Close()

    graphRepo, err := initNeo4j(ctx, cfg)
    if err != nil {
        return fmt.Errorf("neo4j init: %w", err)
    }

    // Create repositories
    repos := initRepositories(pgPool)

    // Create search service
    searchSvc := search.NewService(repos.Patterns, graphRepo)

    // Start REST Admin API listener
    adminRouter := handlers.NewRouter(repos, searchSvc, tel)
    adminServer := &http.Server{
        Addr:    cfg.Server.Admin.Address(),
        Handler: adminRouter,
        // ... timeouts from config
    }

    // Start MCP listener
    mcpHandler := mcpserver.NewHandler(&mcpserver.Deps{
        Patterns: repos.Patterns,
        Agents:   repos.Agents,
        Skills:   repos.Skills,
        Commands: repos.Commands,
        Search:   searchSvc,
        Graph:    graphRepo,
    })
    mcpHTTPServer := &http.Server{
        Addr:    cfg.Server.MCP.Address(),
        Handler: mcpHandler,
        // ... timeouts from config
    }

    // Run both listeners concurrently
    errChan := make(chan error, 2)
    go func() { errChan <- adminServer.ListenAndServe() }()
    go func() { errChan <- mcpHTTPServer.ListenAndServe() }()

    tel.Logger().Info().
        Str("admin_addr", cfg.Server.Admin.Address()).
        Str("mcp_addr", cfg.Server.MCP.Address()).
        Msg("mnemonic started")

    // Wait for shutdown signal or server error
    select {
    case err := <-errChan:
        if err != nil && err != http.ErrServerClosed {
            return err
        }
    case <-ctx.Done():
    }

    // Graceful shutdown of both listeners
    shutCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.Admin.ShutdownTimeout)
    defer cancel()

    adminServer.Shutdown(shutCtx)
    mcpHTTPServer.Shutdown(shutCtx)

    return nil
}
```

---

## 7. Migration Ordering

### Existing Migrations (001-007)

These have been applied in development environments and are part of the migration history. They must NOT be modified.

```
001_extensions_and_functions.sql     -- pgcrypto, pgvector, update_updated_at()
002_create_agents.sql                -- agents table (PK: name)
003_create_patterns.sql              -- patterns table with embedding column
004_create_pattern_agent_associations.sql -- FK: pattern_id -> patterns, agent_name -> agents
005_create_routing_rules.sql         -- FK: agent_name -> agents (ON DELETE RESTRICT)
006_create_enrichment_jobs.sql       -- FK: pattern_id -> patterns
007_create_performance_indexes.sql   -- Indexes for routing_rules, patterns, enrichment_jobs
```

### Foreign Key Dependency Chain

```
agents.name <-- routing_rules.agent_name (ON DELETE RESTRICT)
agents.name <-- pattern_agent_associations.agent_name (ON DELETE CASCADE)
patterns.id <-- pattern_agent_associations.pattern_id (ON DELETE CASCADE)
patterns.id <-- enrichment_jobs.pattern_id (ON DELETE CASCADE)
```

**Critical constraint:** `routing_rules.agent_name` has `ON DELETE RESTRICT`. The `routing_rules` table must be dropped before any structural changes to `agents`.

### Revised Migration Sequence

**Change from previous plan:** Migration 009 (`drop_pattern_agent_associations`) is **removed**. The `pattern_agent_associations` table is kept because it supports agent-scoped pattern filtering, which is needed by both the Admin API and MCP search tools.

```
008_drop_routing_rules.sql
    -- Drop routing_rules table (removes FK constraint on agents.name)
    -- Drop idx_routing_rules_enabled_priority (from 007)
    -- Drop idx_routing_rules_agent (from 005)

009_alter_agents_add_version.sql
    -- ALTER TABLE agents ADD COLUMN version VARCHAR(50) NULL;
    -- COMMENT ON COLUMN agents.routing_keywords IS 'DEPRECATED...';
    -- COMMENT ON COLUMN agents.version IS 'Semantic version...';

010_create_skills.sql
    -- Create skills table with proper constraints
    -- See Section 9 for full schema

011_create_commands.sql
    -- Create commands table with proper constraints
    -- See Section 9 for full schema
```

### Why This Order

1. **008 first:** `routing_rules` has `ON DELETE RESTRICT` to `agents`. Must be dropped before altering `agents`.

2. **009 after 008:** With `routing_rules` gone, only `pattern_agent_associations` references `agents.name` (with `ON DELETE CASCADE`, which is safe). The `agents` table can be altered.

3. **010 and 011 after 009:** Skills and commands are independent tables. They can be created in any order after the agents alteration.

### Down Migration Strategy

- `008_down`: Recreates `routing_rules` with full schema, constraints, and indexes (no data migration)
- `009_down`: `ALTER TABLE agents DROP COLUMN version;` and restore routing_keywords comment
- `010_down`: `DROP TABLE IF EXISTS skills;`
- `011_down`: `DROP TABLE IF EXISTS commands;`

---

## 8. Schema Changes for Existing Tables

### Agents Table Modification

**Migration 009 -- ALTER TABLE agents:**

```sql
-- Add version column as nullable (backward compatible)
ALTER TABLE agents ADD COLUMN IF NOT EXISTS version VARCHAR(50) NULL;

-- Update column comments to reflect new purpose
COMMENT ON COLUMN agents.routing_keywords IS
    'DEPRECATED (pivot 2026-02-15): No longer used for routing. '
    'Retained for backward compatibility. Will be removed in a future migration. '
    'New code should not read or write this column.';

COMMENT ON COLUMN agents.version IS
    'Semantic version of the agent definition (e.g., 1.2.0). '
    'NULL for agents created before version tracking was added.';

COMMENT ON TABLE agents IS
    'Agent definitions for team tooling synchronization '
    '(formerly: routing system targets)';
```

**Go struct changes (`/Users/doublej/dev/mnemonic/src/mnemonic/internal/repository/agent/agent.go`):**

```go
type Agent struct {
    Name            string    `db:"name"`
    Description     string    `db:"description"`
    SystemPrompt    string    `db:"system_prompt"`
    Model           string    `db:"model"`
    AllowedTools    []string  `db:"-"`
    RoutingKeywords []string  `db:"-"` // Deprecated: retained for DB compatibility
    Version         *string   `db:"version"` // Nullable -- nil for pre-pivot agents
    CreatedAt       time.Time `db:"created_at"`
    UpdatedAt       time.Time `db:"updated_at"`
}
```

### Pattern Repository -- Keep Agent Associations

**Unlike the previous plan, these methods are NOT removed from `/Users/doublej/dev/mnemonic/src/mnemonic/internal/repository/pattern/repository.go`:**

- `SetAgentAssociations` -- used by Admin API when creating/updating patterns with agent associations
- `GetAgentAssociations` -- used by MCP `get_pattern` tool to return associated agents
- `validateAgentNames` -- input validation

The `AgentAssociation` type in `/Users/doublej/dev/mnemonic/src/mnemonic/internal/repository/pattern/pattern.go` (lines 94-101) also stays.

---

## 9. New Package Design

### 9.1 Skill Repository (`internal/repository/skill/`)

**Schema (Migration 010):**

```sql
CREATE TABLE IF NOT EXISTS skills (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    content TEXT NOT NULL,
    version VARCHAR(50) NOT NULL DEFAULT '0.0.0',
    tags JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT skills_name_unique UNIQUE (name),
    CONSTRAINT skills_name_format CHECK (name ~ '^[a-z][a-z0-9-]*$'),
    CONSTRAINT skills_content_length CHECK (length(content) <= 524288),
    CONSTRAINT skills_tags_array CHECK (jsonb_typeof(tags) = 'array')
);

CREATE TRIGGER skills_updated_at
    BEFORE UPDATE ON skills
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

COMMENT ON TABLE skills IS 'Skill definitions for team tooling synchronization';
COMMENT ON COLUMN skills.name IS 'Unique identifier, lowercase-with-hyphens (matches Claude Code skill directory name)';
COMMENT ON COLUMN skills.content IS 'Skill content (markdown). Max 512KB. Multi-file skills are post-MVP.';
```

**Go model (`internal/repository/skill/skill.go`):**

```go
package skill

type Skill struct {
    ID          uuid.UUID  `db:"id"`
    Name        string     `db:"name"`
    Description *string    `db:"description"`
    Content     string     `db:"content"`
    Version     string     `db:"version"`
    Tags        []string   `db:"-"`
    CreatedAt   time.Time  `db:"created_at"`
    UpdatedAt   time.Time  `db:"updated_at"`
}
```

**Repository interface (`internal/repository/skill/repository.go`):**

```go
type Repository interface {
    Create(ctx context.Context, skill *Skill) error
    Get(ctx context.Context, id uuid.UUID) (*Skill, error)
    GetByName(ctx context.Context, name string) (*Skill, error)
    Update(ctx context.Context, skill *Skill) error
    Delete(ctx context.Context, id uuid.UUID) error
    List(ctx context.Context, filter Filter, opts repository.ListOptions) ([]*Skill, int64, error)
    Exists(ctx context.Context, id uuid.UUID) (bool, error)
}
```

### 9.2 Command Repository (`internal/repository/command/`)

**Schema (Migration 011):**

```sql
CREATE TABLE IF NOT EXISTS commands (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    content TEXT NOT NULL,
    version VARCHAR(50) NOT NULL DEFAULT '0.0.0',
    tags JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT commands_name_unique UNIQUE (name),
    CONSTRAINT commands_name_format CHECK (name ~ '^[a-z][a-z0-9-]*$'),
    CONSTRAINT commands_content_length CHECK (length(content) <= 51200),
    CONSTRAINT commands_tags_array CHECK (jsonb_typeof(tags) = 'array')
);

CREATE TRIGGER commands_updated_at
    BEFORE UPDATE ON commands
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();
```

The Go model and repository interface mirror the skill package exactly. The only differences are table name, content length constraint, and package name.

### 9.3 Handler Packages (`internal/handlers/`)

All handlers follow the same dependency-injected pattern. The REST API is the write side of the system -- data is loaded via `curl` or any HTTP client.

**Handler Architecture:**

```go
// internal/handlers/agents/agents.go
package agents

import (
    "github.com/gin-gonic/gin"
    agentrepo "github.com/twistingmercury/mnemonic/internal/repository/agent"
)

type Handler struct {
    repo agentrepo.Repository
}

func NewHandler(repo agentrepo.Repository) *Handler {
    return &Handler{repo: repo}
}

func (h *Handler) SetupRoutes(rg *gin.RouterGroup) {
    rg.GET("/agents", h.List)
    rg.POST("/agents", h.Create)
    rg.GET("/agents/:name", h.Get)
    rg.PUT("/agents/:name", h.Update)
    rg.DELETE("/agents/:name", h.Delete)
}
```

**Admin API Routes:**

| Route                  | Method | Handler              | Purpose                 |
| ---------------------- | ------ | -------------------- | ----------------------- |
| `/v1/api/agents`       | GET    | `agents.List`        | List agents (paginated) |
| `/v1/api/agents`       | POST   | `agents.Create`      | Create agent            |
| `/v1/api/agents/:name` | GET    | `agents.Get`         | Get agent by name       |
| `/v1/api/agents/:name` | PUT    | `agents.Update`      | Update agent            |
| `/v1/api/agents/:name` | DELETE | `agents.Delete`      | Delete agent            |
| `/v1/api/patterns`     | GET    | `patterns.List`      | List patterns           |
| `/v1/api/patterns`     | POST   | `patterns.Create`    | Create pattern          |
| `/v1/api/patterns/:id` | GET    | `patterns.Get`       | Get pattern             |
| `/v1/api/patterns/:id` | PUT    | `patterns.Update`    | Update pattern          |
| `/v1/api/patterns/:id` | DELETE | `patterns.Delete`    | Delete pattern          |
| `/v1/api/skills`       | GET    | `skills.List`        | List skills             |
| `/v1/api/skills`       | POST   | `skills.Create`      | Create skill            |
| `/v1/api/skills/:id`   | GET    | `skills.Get`         | Get skill               |
| `/v1/api/skills/:id`   | PUT    | `skills.Update`      | Update skill            |
| `/v1/api/skills/:id`   | DELETE | `skills.Delete`      | Delete skill            |
| `/v1/api/commands`     | GET    | `commands.List`      | List commands           |
| `/v1/api/commands`     | POST   | `commands.Create`    | Create command          |
| `/v1/api/commands/:id` | GET    | `commands.Get`       | Get command             |
| `/v1/api/commands/:id` | PUT    | `commands.Update`    | Update command          |
| `/v1/api/commands/:id` | DELETE | `commands.Delete`    | Delete command          |
| `/v1/api/search`       | GET    | `search.Search`      | Semantic search         |
| `/ops/health`          | GET    | `operations.Health`  | Health check            |
| `/ops/version`         | GET    | `operations.Version` | Version info            |

### 9.4 Search Service (`internal/service/search/`)

```go
package search

type Result struct {
    Pattern         *patternrepo.Pattern
    Similarity      float64
    RelatedPatterns []graphrepo.RelatedPattern
}

type Service interface {
    Search(ctx context.Context, query string, opts SearchOptions) ([]Result, error)
}

type SearchOptions struct {
    MaxResults    int
    MinSimilarity float64
    Tags          []string
    AgentName     string   // Filter to patterns relevant to this agent
    IncludeGraph  bool     // Enrich with graph context
}
```

The service coordinates:

1. Generate embedding for the search query (requires OpenAI integration)
2. If `AgentName` is set, call `pattern.Repository.GetAgentAssociations()` to get relevant pattern IDs
3. Call `pattern.Repository.FindSimilar()` with the embedding (optionally filtered to pattern IDs)
4. If `IncludeGraph` is true, call `graph.Repository.FindRelatedPatterns()`
5. Return merged results

**Note:** This service cannot function until the OpenAI embedding integration is implemented. Until then, the search endpoint and MCP `search_patterns` tool return an appropriate "search not yet available" response, and the pattern list endpoint's `SearchQuery` filter (full-text search on name/description) serves as a basic search mechanism.

---

## 10. Configuration Changes

### Remove

```go
// Remove from MnemonicConfig:
Routing RoutingConfig `mapstructure:"routing" yaml:"routing"`

// Remove RoutingConfig struct entirely
// Remove RoutingCacheConfig struct entirely
// Remove routing defaults from SetDefaults()
// Remove routing validation from Validate()
```

### Restructure Server Config

The current single `ServerConfig` becomes two listener configs.

```go
// MnemonicConfig -- revised top level
type MnemonicConfig struct {
    Server        ServerConfigs       `mapstructure:"server" yaml:"server"`
    Database      DatabaseConfig      `mapstructure:"database" yaml:"database"`
    OpenAI        OpenAIConfig        `mapstructure:"openai" yaml:"openai"`
    RateLimit     RateLimitConfig     `mapstructure:"rate_limit" yaml:"rate_limit"`
    Enrichment    EnrichmentConfig    `mapstructure:"enrichment" yaml:"enrichment"`
    Logging       LoggingConfig       `mapstructure:"logging" yaml:"logging"`
    Observability ObservabilityConfig `mapstructure:"observability" yaml:"observability"`
}

// ServerConfigs holds configuration for both listeners.
type ServerConfigs struct {
    Admin AdminServerConfig `mapstructure:"admin" yaml:"admin"`
    MCP   MCPServerConfig   `mapstructure:"mcp" yaml:"mcp"`
}

// AdminServerConfig is the REST admin API listener configuration.
type AdminServerConfig struct {
    Host            string        `mapstructure:"host" yaml:"host"`
    Port            int           `mapstructure:"port" yaml:"port"`
    ReadTimeout     time.Duration `mapstructure:"read_timeout" yaml:"read_timeout"`
    WriteTimeout    time.Duration `mapstructure:"write_timeout" yaml:"write_timeout"`
    IdleTimeout     time.Duration `mapstructure:"idle_timeout" yaml:"idle_timeout"`
    ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout" yaml:"shutdown_timeout"`
    TLS             TLSConfig     `mapstructure:"tls" yaml:"tls"`
}

// MCPServerConfig is the MCP Streamable HTTP listener configuration.
type MCPServerConfig struct {
    Host            string        `mapstructure:"host" yaml:"host"`
    Port            int           `mapstructure:"port" yaml:"port"`
    ReadTimeout     time.Duration `mapstructure:"read_timeout" yaml:"read_timeout"`
    WriteTimeout    time.Duration `mapstructure:"write_timeout" yaml:"write_timeout"`
    IdleTimeout     time.Duration `mapstructure:"idle_timeout" yaml:"idle_timeout"`
    ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout" yaml:"shutdown_timeout"`
    SessionTimeout  time.Duration `mapstructure:"session_timeout" yaml:"session_timeout"`
    TLS             TLSConfig     `mapstructure:"tls" yaml:"tls"`
}

func (c *AdminServerConfig) Address() string {
    return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func (c *MCPServerConfig) Address() string {
    return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
```

### Neo4j Config (No Toggle)

```go
type Neo4jConfig struct {
    URI                              string        `mapstructure:"uri" yaml:"uri"`
    Username                         string        `mapstructure:"username" yaml:"username"`
    Password                         string        `mapstructure:"password" yaml:"password"`
    Database                         string        `mapstructure:"database" yaml:"database"`
    MaxConnectionPoolSize            int           `mapstructure:"max_connection_pool_size" yaml:"max_connection_pool_size"`
    ConnectionAcquisitionTimeout     time.Duration `mapstructure:"connection_acquisition_timeout" yaml:"connection_acquisition_timeout"`
}
```

Neo4j config is always validated. There is no `Enabled` field -- Neo4j is a required dependency.

### Validation Changes

```go
func (c *DatabaseConfig) validate() ValidationErrors {
    var errs ValidationErrors
    errs = append(errs, c.Postgres.validate()...)
    errs = append(errs, c.Neo4j.validate()...)
    return errs
}
```

The admin and MCP listener configs both validate independently. Cross-validation ensures they are on different ports.

### New Defaults

```go
// Server defaults
const (
    DefaultAdminHost            = "0.0.0.0"
    DefaultAdminPort            = 8080
    DefaultAdminReadTimeout     = 30 * time.Second
    DefaultAdminWriteTimeout    = 30 * time.Second
    DefaultAdminIdleTimeout     = 120 * time.Second
    DefaultAdminShutdownTimeout = 5 * time.Second

    DefaultMCPHost              = "0.0.0.0"
    DefaultMCPPort              = 8081
    DefaultMCPReadTimeout       = 30 * time.Second
    DefaultMCPWriteTimeout      = 120 * time.Second  // Longer for SSE streaming
    DefaultMCPIdleTimeout       = 120 * time.Second
    DefaultMCPShutdownTimeout   = 5 * time.Second
    DefaultMCPSessionTimeout    = 30 * time.Minute
)
```

### Example Post-Pivot Config File

```yaml
server:
  admin:
    host: "0.0.0.0"
    port: 8080
  mcp:
    host: "0.0.0.0"
    port: 8081
    session_timeout: "30m"

database:
  postgres:
    host: "localhost"
    port: 5433
    database: "mnemonic"
  neo4j:
    uri: "bolt://localhost:7688"

openai:
  api_key: ""

enrichment:
  worker_count: 2
  poll_interval: "5s"
```

---

## 11. Phased Implementation Plan

### Prerequisites

Before any pivot work begins:

1. **Tag the pre-pivot codebase:** `git tag pre-pivot-v0.14` on the current `develop` branch head
2. **Create the pivot branch:** `feature/pivot` (already exists per git status)

---

### Phase P1: Database Migration -- Drop Routing Tables

**Goal:** Remove routing-related tables from the database schema.

**Migrations:**

- `008_drop_routing_rules.sql` (up + down)

**Note:** Unlike the previous plan, there is only one migration in this phase. We no longer drop `pattern_agent_associations`.

**Delegated to:** `data-architect` (schema design), `data-engineer` (migration SQL)

**Deliverables:**

- `/Users/doublej/dev/mnemonic/src/migrations/postgres/up/008_drop_routing_rules.sql`
- `/Users/doublej/dev/mnemonic/src/migrations/postgres/down/008_drop_routing_rules.sql`

**Acceptance criteria:**

- Migration applies cleanly against a database with 001-007 already applied
- Down migration recreates `routing_rules` with identical schema, constraints, and indexes
- No data migration needed

**Dependencies:** None (first phase).

---

### Phase P2: Database Migration -- Modify Agents, Create Skills/Commands

**Goal:** Add `version` column to agents; create `skills` and `commands` tables.

**Migrations:**

- `009_alter_agents_add_version.sql` (up + down)
- `010_create_skills.sql` (up + down)
- `011_create_commands.sql` (up + down)

**Delegated to:** `data-architect` (schema design), `data-engineer` (migration SQL)

**Deliverables:**

- Six migration files (up + down for each)
- Schema follows conventions from existing migrations

**Acceptance criteria:**

- 009 applies after 008; adds `version` column as nullable
- 010 and 011 create tables matching the schema in Section 9
- Down migrations cleanly reverse all changes

**Dependencies:** Phase P1.

---

### Phase P3: Remove Routing Code + Restructure Packages

**Goal:** Delete all routing-related Go packages, restructure handler packages.

**Files to remove:**

- `internal/routing/` (entire directory -- 4,214 lines)
- `internal/repository/routingrule/` (entire directory -- 1,773 lines)
- `internal/handlers/routes/` (entire directory)
- `internal/handlers/routes/rules/` (entire directory)
- `internal/metrics/routing.go`
- `internal/metrics/routing_test.go`
- `tests/e2e/routing_test.go`
- `tests/e2e/routing_rules_test.go`

**Files to modify:**

- `internal/metrics/registry.go` -- Remove `Routing *Routing` field from `Registry`
- `internal/config/config.go` -- Remove `RoutingConfig`, `RoutingCacheConfig`, routing validation
- `internal/config/defaults.go` -- Remove routing default constants
- `cmd/main/main.go` -- Verify no routing imports remain

**Delegated to:** `go-software-engineer`

**Acceptance criteria:**

- All unit tests pass (`go test ./...`)
- `go vet ./...` passes
- `golangci-lint run` passes
- Build succeeds
- The binary starts and serves `/ops/health` and `/ops/version`

**Dependencies:** Phase P1 and P2.

---

### Phase P4: Configuration Overhaul

**Goal:** Restructure configuration for two-listener architecture.

**Files to modify:**

- `/Users/doublej/dev/mnemonic/src/mnemonic/internal/config/config.go` -- Two listener configs, remove routing, remove Neo4j toggle
- `/Users/doublej/dev/mnemonic/src/mnemonic/internal/config/defaults.go` -- New defaults for admin/MCP ports
- `/Users/doublej/dev/mnemonic/src/mnemonic/internal/config/config_test.go` -- Update all tests

**Delegated to:** `go-software-engineer`

**Acceptance criteria:**

- `server.admin` and `server.mcp` config sections work
- Neo4j config is always validated (no `enabled` toggle)
- Cross-validation: admin port != MCP port
- All config tests pass
- Backward compatibility: old `server.host`/`server.port` does not break (migration path)

**Dependencies:** Phase P3.

---

### Phase P5: Skill and Command Repositories

**Goal:** Implement skill and command repositories following established patterns.

**New packages:**

- `internal/repository/skill/` -- `skill.go`, `errors.go`, `repository.go`, `repository_test.go`, `doc.go`
- `internal/repository/command/` -- `command.go`, `errors.go`, `repository.go`, `repository_test.go`, `doc.go`

**Delegated to:** `go-software-engineer`

**Pattern to follow:** The `internal/repository/agent/` package is the template. Each new repository:

1. Defines a model struct with `db` tags
2. Defines a `Repository` interface with CRUD + List + Exists + GetByName
3. Implements `pgxRepository` using `repository.DBTX` injection
4. Handles JSONB marshaling/unmarshaling for `tags`
5. Uses `pgxmock` for unit tests
6. Defines package-specific sentinel errors (`ErrNotFound`, `ErrExists`)

**Acceptance criteria:**

- Full CRUD operations with unit tests
- JSONB tag handling consistent with pattern repository
- Error handling consistent with existing repositories
- Test coverage for edge cases

**Dependencies:** Phase P2 (tables must exist).

---

### Phase P6: Modify Agent Repository

**Goal:** Update agent repository to support `version` column.

**Files to modify:**

- `/Users/doublej/dev/mnemonic/src/mnemonic/internal/repository/agent/agent.go` -- Add `Version *string` field
- `/Users/doublej/dev/mnemonic/src/mnemonic/internal/repository/agent/repository.go` -- Update queries
- `/Users/doublej/dev/mnemonic/src/mnemonic/internal/repository/agent/repository_test.go` -- Update tests

**Delegated to:** `go-software-engineer`

**Acceptance criteria:**

- `Version` field is `*string` (nullable)
- Create/Get/Update/List include `version` column
- All existing tests pass (backward compatible)
- New tests cover version field scenarios

**Dependencies:** Phase P2 (migration 009 must be applied).

---

### Phase P7: REST Admin API Handlers

**Goal:** Implement working REST handlers for all admin operations.

**New/rewritten packages:**

- `internal/handlers/agents/` -- Working CRUD with `agent.Repository`
- `internal/handlers/patterns/` -- Working CRUD with `pattern.Repository`
- `internal/handlers/skills/` -- CRUD with `skill.Repository`
- `internal/handlers/commands/` -- CRUD with `command.Repository`
- `internal/handlers/search/` -- Search with `search.Service`
- `internal/handlers/router.go` -- Gin router wiring

**Delegated to:** `go-software-engineer`

**Acceptance criteria:**

- All CRUD endpoints return proper HTTP status codes (200, 201, 400, 404, 409, 500)
- Request/response JSON matches the schema
- Repository errors map to appropriate HTTP errors
- Pagination support (limit/offset) on list endpoints
- Unit tests for each handler using mocked repositories

**Dependencies:** Phases P3, P5, P6.

---

### Phase P8: MCP Server Implementation

**Goal:** Implement the MCP server with all read-only tools.

**New packages:**

- `internal/mcpserver/server.go` -- MCP server setup, handler creation
- `internal/mcpserver/tools_search.go` -- `search_patterns`, `find_related_patterns`
- `internal/mcpserver/tools_patterns.go` -- `get_pattern`, `list_patterns`
- `internal/mcpserver/tools_agents.go` -- `get_agent`, `list_agents`
- `internal/mcpserver/tools_skills.go` -- `get_skill`, `list_skills`
- `internal/mcpserver/tools_commands.go` -- `get_command`, `list_commands`

**New dependency in `go.mod`:**

- `github.com/modelcontextprotocol/go-sdk` (official MCP Go SDK)

**Delegated to:** `go-software-engineer`

**Acceptance criteria:**

- MCP server starts on configured port
- Claude Code can connect and discover all tools via `tools/list`
- All tool handlers return correct data from repositories
- Tool input schemas are auto-generated from struct tags
- Search tools return "not yet available" until embedding integration is built
- Session management works (create, timeout, cleanup)
- Unit tests for each tool handler

**Dependencies:** Phases P5, P6, P7 (repositories and admin API must exist for data to query).

---

### Phase P9: Server Lifecycle

**Goal:** Rewrite `internal/server/` to start both REST and MCP listeners in one process.

**Files to rewrite:**

- `/Users/doublej/dev/mnemonic/src/mnemonic/internal/server/server.go` -- Full rewrite per Section 6 design

**Files to create/modify:**

- `cmd/mnemonic/main.go` -- New main entrypoint

**Delegated to:** `go-software-engineer`

**Acceptance criteria:**

- Both listeners start on their configured ports
- Graceful shutdown stops both listeners
- Health checks cover both Postgres and Neo4j
- TLS configurable independently for each listener
- Process exits cleanly on SIGINT/SIGTERM
- Startup logs show both listener addresses

**Dependencies:** Phases P7, P8.

---

### Phase P10: API Specification Update

**Goal:** Update the OpenAPI specification for the Admin API. Document MCP tool schemas.

**Delegated to:** `api-architect`

**Deliverables:**

- `api/openapi/mnemonic-admin-v1.yaml` -- Admin REST API spec
- `api/mcp/tool-schemas.md` -- MCP tool documentation (tool names, descriptions, input/output schemas)

**Changes:**

- Remove all routing-related paths and schemas
- Update agents paths to `/v1/api/agents` with `version` field
- Add skills paths with full CRUD
- Add commands paths with full CRUD
- Add search path
- Document MCP tool schemas separately (MCP tools are self-documenting via the protocol, but human-readable docs are needed)

**Dependencies:** Phases P7, P8 (implementation must be stable before finalizing specs).

---

### Phase P11: E2E Tests

**Goal:** End-to-end tests for both the Admin API and MCP server.

**Delegated to:** `go-e2e-test-engineer`

**Test suites:**

**Admin API E2E:**

- Agent CRUD lifecycle
- Pattern CRUD lifecycle (including agent associations)
- Skill CRUD lifecycle
- Command CRUD lifecycle
- List with pagination
- Error cases: duplicate names, not found, invalid input

**MCP Server E2E:**

- Connect Claude-like client, discover tools
- Call each tool, verify results
- Verify read-only behavior (no write tools exposed)
- Session lifecycle (create, timeout)

**Acceptance criteria:**

- All E2E tests pass against fresh Docker Compose environment
- Tests are independent (no ordering dependencies)
- Tests clean up after themselves
- Both protocols are tested

**Dependencies:** Phase P9.

---

### Phase P12: Documentation Update

**Goal:** Update all documentation to reflect the post-pivot architecture.

**Delegated to:** `technical-writer`

**Files to create/update:**

- ADR-008 in `docs/architecture/00-architectural-decisions.md` -- Pivot decision record
- ADR-009 -- MCP protocol choice for Claude Code integration
- `docs/architecture/02-system-architecture.md` -- Architecture diagrams
- `docs/architecture/04-data-architecture.md` -- Updated schema documentation
- `docs/design/2026-02-15-pivot-api-specification.md` -- Admin API docs
- `docs/design/mcp-server.md` -- MCP server documentation
- `CHANGELOG.md` -- Document the pivot

**Dependencies:** Phase P11.

---

### Phase P13: Deployment Update

**Goal:** Update Docker and deployment configurations.

**Delegated to:** `devops-engineer`

**Changes:**

- Update Dockerfile to build `cmd/mnemonic/` (new entrypoint)
- Expose both ports in Docker (8080 for admin, 8081 for MCP)
- Update `docker-compose.yaml` with both port mappings
- Verify CI/CD pipeline works with post-pivot codebase

**Dependencies:** Phase P11.

---

### Phase Summary and Dependency Graph

```
P1 (Drop routing_rules)
  +-> P2 (Add version, create skills/commands)
        +-> P3 (Remove routing Go code + restructure)
        |     +-> P4 (Config overhaul: two listeners)
        |     +-> P6 (Modify agent repo)
        +-> P5 (Skill + command repos)
              +-> P7 (Admin API handlers) <-- also depends on P3, P4, P6
              |     +-> P8 (MCP server)
              |     |     +-> P9 (Server lifecycle) <-- depends on P7, P8
              |     |           +-> P11 (E2E tests)
              |     +-> P10 (API spec update) <-- depends on P7, P8
              +-> P11 (E2E tests)
                    +-> P12 (Documentation)
                    +-> P13 (Deployment update)
```

### Estimated Scope

| Phase | LOC Added         | LOC Removed       | Complexity                   |
| ----- | ----------------- | ----------------- | ---------------------------- |
| P1    | ~40 (migration)   | 0                 | Low                          |
| P2    | ~120 (migrations) | 0                 | Low                          |
| P3    | 0                 | ~6,300            | Low (deletion + restructure) |
| P4    | ~150              | ~80               | Medium                       |
| P5    | ~1,200            | 0                 | Medium (follows template)    |
| P6    | ~80               | ~30               | Low                          |
| P7    | ~2,000            | ~200 (stubs)      | High (core feature)          |
| P8    | ~1,500            | 0                 | High (new protocol)          |
| P9    | ~300              | ~140              | Medium                       |
| P10   | ~500 (YAML/docs)  | ~2,000 (old spec) | Medium                       |
| P11   | ~1,500            | ~500 (old tests)  | High                         |
| P12   | ~1,000 (docs)     | ~500 (old docs)   | Medium                       |
| P13   | ~100              | ~50               | Low                          |

---

## 12. Open Decisions

### 12.1 MCP Server Path Prefix

The MCP `StreamableHTTPHandler` is a standard `http.Handler`. It can be mounted at the root (`/`) of the MCP HTTP server, or at a path prefix like `/mcp`.

**Recommendation:** Mount at `/mcp` for clarity. This means Claude Code connects to `http://localhost:8081/mcp`. This leaves room for future endpoints on the MCP listener (e.g., `/health` for MCP-specific health checks).

### 12.2 Search Tool Behavior Before Embeddings

The `search_patterns` MCP tool and the `/v1/api/search` admin endpoint require OpenAI embedding generation to function. Until the enrichment pipeline is built:

**Option A:** Return an error ("semantic search not yet available, use list_patterns with search filter instead")

**Option B:** Fall back to full-text search on name/description (the existing `SearchQuery` filter in `pattern.Repository.List()`)

**Recommendation:** Option B. Full-text fallback provides value immediately. The tool description should note that results improve when embeddings are available.

### 12.3 MCP Tool Naming Convention

MCP tools are identified by name strings. Naming conventions:

**Option A:** `snake_case` -- `search_patterns`, `get_agent`, `list_skills`
**Option B:** `kebab-case` -- `search-patterns`, `get-agent`, `list-skills`
**Option C:** Namespaced -- `mnemonic.search_patterns`, `mnemonic.get_agent`

**Recommendation:** Option A (`snake_case`). This matches the JSON-RPC convention and the official MCP SDK examples. No namespace prefix is needed since the MCP server is already identified as "mnemonic" in the implementation info.

### 12.4 Session Timeout for MCP

The `StreamableHTTPOptions.SessionTimeout` controls how long an idle MCP session lives before being cleaned up. This affects how long Claude Code can maintain a connection without activity.

**Recommendation:** 30 minutes default, configurable via `server.mcp.session_timeout`. Claude Code sessions are typically active for the duration of a conversation, which can vary widely.

### 12.5 API Response Format for Agents

The pivot proposal suggests a nested `definition` object in the agent API response. The solutions architect recommends against restructuring. I continue to agree -- keep the flat structure at the API level:

```json
{
  "name": "go-software-engineer",
  "description": "Implements Go code...",
  "system_prompt": "You are an expert...",
  "model": "opus",
  "allowed_tools": ["Read", "Write", "Edit", "Bash"],
  "version": "1.2.0",
  "created_at": "2026-02-15T10:00:00Z",
  "updated_at": "2026-02-15T10:00:00Z"
}
```

The `routing_keywords` field is omitted from the response (deprecated column).

### 12.6 Skill Content Model

Skills in Claude Code can be multi-file (e.g., `instructions.md` plus `scripts/sync.sh`).

**MVP approach:** Store only the `instructions.md` content. Multi-file skills are not yet supported.

**Post-MVP approach:** Either JSONB column for a file manifest, or a `skill_files` table.

### 12.7 Sync Direction

With MCP replacing REST+skills for Claude Code integration, the sync workflow changes:

**Writing data into Mnemonic:** `curl` against the Admin API (POST/PUT requests)

**Reading data from Mnemonic into Claude Code:** Claude Code calls MCP tools directly. No file-writing needed.

**Reading data from Mnemonic onto local filesystem:** A future CLI tool or script could pull data and write to `~/.claude/`. This is a post-MVP feature.

---

## Summary

This revised plan transforms Mnemonic from a routing orchestrator into a knowledge graph and tooling synchronization server.

**Key architectural decisions:**

1. **Single server, two listeners (MVP):** REST Admin API on port 8080, MCP Streamable HTTP on port 8081
2. **Official MCP Go SDK** (`github.com/modelcontextprotocol/go-sdk/mcp`) for Claude Code integration
3. **MCP replaces REST+skills:** Claude Code calls MCP tools directly; no shell script wrappers needed
4. **`pattern_agent_associations` kept:** Needed for agent-scoped pattern filtering
5. **Neo4j required:** Always initialized alongside Postgres; no optional toggle
6. **Data loading via `curl`:** No CLI binary for MVP; `curl` or `httpie` against the REST Admin API
7. **Organized by concern:** Handlers, repositories, services -- not by service boundary
8. **No Python anywhere:** All tooling in Go

**What gets discarded:** ~6,300 lines of routing engine and routing rule code.

**What gets built:**

- MCP server with 10+ read-only tools (~1,500 LOC)
- Working admin API handlers (~2,000 LOC)
- Skill/command repositories (~1,200 LOC)
- Server lifecycle management (~300 LOC)
- E2E tests for both protocols (~1,500 LOC)

**What remains on the roadmap (unbuilt):** Enrichment pipeline (OpenAI embedding generation, concept extraction, Neo4j graph sync). These are prerequisites for the full semantic search experience via the `search_patterns` MCP tool.
