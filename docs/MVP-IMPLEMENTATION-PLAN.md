# ACE (Agentic Coding Engine) MVP Implementation Plan

**Version:** 1.0
**Created:** January 8, 2026
**Status:** Ready for Implementation

## Executive Summary

This plan defines a minimal viable product for ACE that proves **deterministic agent routing with dynamic pattern querying**. The MVP focuses on local deployment, manual builds, and simplified security.

### MVP Constraints

- Local deployment only - no production deployment
- Focus on proving core functionality
- Security deferred to future iterations (no Envoy/OPA, no auth)
- No CI/CD pipeline - manual builds only

---

## 1. MVP Scope Definition

### IN Scope (Must Have for MVP)

| Component                  | Description                          | Rationale                                |
| -------------------------- | ------------------------------------ | ---------------------------------------- |
| **API Server**             | REST API for agent execution         | Core entry point for client interactions |
| **Deterministic Router**   | Code-based routing logic             | Predictable agent selection              |
| **Cognee Integration**     | MCP client to existing Cognee server | Dynamic pattern querying                 |
| **Claude API Integration** | Agent execution via Claude API       | Core execution engine                    |
| **Single Agent Type**      | go-software-agent only               | Proves routing and pattern querying work |
| **Local Docker Compose**   | Full stack runs locally              | Local development and testing            |
| **Basic Usage Logging**    | Log executions to file/stdout        | Debugging and validation                 |
| **Health Endpoint**        | /health via heartbeat                | Operational visibility                   |
| **CLI Client**             | Simple command-line interface        | Manual testing and validation            |

### OUT of Scope (Deferred)

| Component                    | Deferral Reason                        |
| ---------------------------- | -------------------------------------- |
| Envoy/OPA sidecars           | Security deferred                      |
| Authentication/Authorization | Security deferred                      |
| Rate limiting                | Not needed locally                     |
| PostgreSQL for ACE app data  | Use file-based storage initially       |
| Usage billing/cost tracking  | Not needed for MVP                     |
| Web UI                       | CLI sufficient                         |
| Multi-agent routing          | Prove single agent first               |
| Kubernetes deployment        | Docker Compose sufficient locally      |
| CI/CD pipeline               | Manual builds                          |
| Agent chaining               | Prove single execution first           |
| Streaming responses          | Add after basic request-response works |
| Redis caching                | Optimize later                         |

### Success Criteria

The MVP is complete when:

1. **Core Flow Works**: User can submit a prompt via CLI, get routed to go-software-agent, have patterns queried from Cognee, and receive a response from Claude
2. **Deterministic Routing Verified**: Same prompt always routes to same agent (log verification)
3. **Pattern Querying Works**: Agent successfully queries Cognee for relevant patterns during execution
4. **End-to-End Latency**: Total execution completes within 60 seconds (excluding Claude thinking time)
5. **Local Stack Runs**: Full system starts with `docker compose up`
6. **Basic Health Checks Pass**: All services report healthy

**Validation Test**:

```bash
# Should work consistently
ace execute --agent go-software-agent --prompt "Write a function to reverse a string"
```

---

## 2. Technical Stack Decisions

### API Server: Go with Gin

- Existing agent definitions are Go-focused
- Low memory footprint, fast startup
- Strong concurrency model for Claude API calls

### Health Checks: twistingmercury/heartbeat

- https://github.com/twistingmercury/heartbeat
- Parallel dependency checking with timeouts
- Kubernetes-ready (liveness/readiness probes)
- Returns 503 when dependencies are unhealthy

### Local Infrastructure

**Docker Compose Stack**:

| Service    | Image                        | Purpose         | Port       |
| ---------- | ---------------------------- | --------------- | ---------- |
| ace-api    | Custom Go image              | API server      | 3000       |
| cognee-mcp | cognee/cognee-mcp:local-main | Pattern queries | 8000       |
| postgres   | pgvector/pgvector:pg16       | Cognee data     | 5432       |
| neo4j      | neo4j:5-community            | Knowledge graph | 7474, 7687 |

**Note**: We reuse the existing Cognee stack from `memory-mcp-server/docker-compose.yaml`

### Reusable Existing Assets

| Asset                      | Location                                | Reuse Strategy                |
| -------------------------- | --------------------------------------- | ----------------------------- |
| Cognee Docker Compose      | `memory-mcp-server/docker-compose.yaml` | Extend with ace-api service   |
| Agent Patterns             | `agent-patterns/`                       | Load into Cognee for querying |
| Agent Definitions          | `agents/definitions/go/`                | Reference for agent behavior  |
| Pattern Loading Scripts    | `scripts/load-patterns.sh`              | Use to populate Cognee        |
| Pattern Processing Scripts | `scripts/cognify-patterns.sh`           | Use to build knowledge graph  |

---

## 3. Implementation Phases

### Phase 1: Foundation (Week 1)

**Goal**: Project scaffolding, contracts, and local environment

**Deliverables**:

1. **Go Project Structure**

   ```
   src/
   ├── api/                      # API server (Go module)
   │   ├── cmd/
   │   │   └── ace-api/
   │   │       └── main.go
   │   ├── internal/
   │   │   ├── api/              # HTTP handlers
   │   │   ├── router/           # Deterministic routing logic
   │   │   ├── mcp/              # MCP client for Cognee
   │   │   ├── claude/           # Claude API client
   │   │   └── config/           # Configuration
   │   ├── pkg/
   │   │   └── models/           # Shared data models
   │   ├── go.mod
   │   ├── go.sum
   │   └── Dockerfile
   │
   └── cli/                      # CLI client (separate Go module)
       ├── cmd/
       │   └── ace/
       │       └── main.go
       ├── internal/
       │   ├── client/           # HTTP client for ace-api
       │   └── output/           # Response formatting
       ├── go.mod
       └── go.sum
   ```

2. **API Contract** (OpenAPI spec)

   - `POST /api/v1/execute` - Execute agent
   - `GET /health` - Health check (via heartbeat)

3. **Extended Docker Compose**

   - Add ace-api service to existing Cognee stack
   - Configure networking between services

4. **Environment Configuration**
   - `.env.example` with required variables
   - Configuration loading in Go

**Dependencies**: None (first phase)

**Assigned Agent**: go-architect-agent, devops-agent

### Phase 2: Core API Server (Week 2)

**Goal**: Working REST API with stubbed dependencies

**Deliverables**:

1. **HTTP Server with Gin**

   - `/api/v1/execute` endpoint
   - Request validation
   - Error handling
   - JSON request/response

2. **Configuration Management**

   - Environment variable loading
   - Sensible defaults for local development

3. **Health Endpoint** (using twistingmercury/heartbeat)

   - `/health` - Liveness/readiness with dependency status
   - Register Cognee as dependency with timeout

4. **Logging**

   - Structured JSON logging
   - Request/response logging
   - Execution logging

5. **Stubbed Dependencies**
   - Mock MCP client (returns static patterns)
   - Mock Claude client (returns static response)

**Dependencies**: Phase 1 complete

**Assigned Agent**: go-software-agent

### Phase 3: Cognee MCP Integration (Week 3)

**Goal**: Live pattern querying from Cognee

**Deliverables**:

1. **MCP Client Implementation**

   - SSE transport connection to Cognee
   - `search` tool call
   - Response parsing

2. **Pattern Query Logic**

   - Extract keywords from prompt
   - Build search query
   - Parse and format pattern results

3. **Connection Management**

   - Retry logic
   - Timeout handling
   - Health check integration

4. **Pattern Loading Verification**
   - Verify existing patterns load correctly
   - Test search queries return expected patterns

**Dependencies**: Phase 2 complete, Cognee stack running with patterns loaded

**Assigned Agent**: go-software-agent

### Phase 4: Claude API Integration (Week 4)

**Goal**: Live agent execution with Claude

**Deliverables**:

1. **Claude API Client**

   - Messages API integration
   - Tool definitions for pattern search
   - Response handling

2. **Agent Prompt Assembly**

   - Load agent definition (go-software-agent.md)
   - Combine with user prompt
   - Include retrieved patterns

3. **Tool Call Handling**

   - Detect pattern search tool calls
   - Route to MCP client
   - Return results to Claude

4. **Response Processing**
   - Extract final response
   - Format for API response
   - Log execution details

**Dependencies**: Phase 3 complete

**Assigned Agent**: go-software-agent

### Phase 5: Deterministic Router (Week 5)

**Goal**: Code-based agent selection

**Deliverables**:

1. **Router Implementation**

   - Keyword-based routing rules
   - Agent registry
   - Default agent handling

2. **Routing Configuration**

   - YAML-based routing rules
   - Runtime loading

3. **Routing Logging**

   - Log routing decisions
   - Enable verification of determinism

4. **MVP Agent Support**
   - go-software-agent only initially
   - Extensible for future agents

**Dependencies**: Phase 4 complete

**Assigned Agent**: go-software-agent

### Phase 6: CLI Client (Week 6)

**Goal**: Command-line interface for testing (separate Go module at `src/cli/`)

**Deliverables**:

1. **CLI Project Setup**

   - Separate Go module at `src/cli/`
   - Independent `go.mod` (not shared with API server)
   - Binary name: `ace`

2. **CLI Structure (Cobra)**

   - `ace execute` command
   - Configuration flags

3. **API Client**

   - HTTP client for ace-api
   - Response formatting

4. **User Experience**
   - Progress indication
   - Error messages
   - Output formatting

**Dependencies**: Phase 5 complete

**Assigned Agent**: go-software-agent

### Phase 7: Integration & Testing (Week 7)

**Goal**: End-to-end validation

**Deliverables**:

1. **Integration Tests**

   - Full flow tests
   - Error scenario tests
   - Timeout tests

2. **Manual Test Suite**

   - Test scripts for common scenarios
   - Validation checklist

3. **Documentation**
   - Local setup guide
   - Troubleshooting guide

**Dependencies**: All previous phases complete

**Assigned Agents**: go-e2e-test-agent, bats-test-agent

### Phase 8: Polish & Handoff (Week 8)

**Goal**: MVP ready for demonstration

**Deliverables**:

1. **Bug Fixes**

   - Address issues found in Phase 7

2. **Performance Tuning**

   - Optimize critical paths
   - Reduce startup time

3. **Final Documentation**
   - Architecture overview
   - Extension guide

**Dependencies**: Phase 7 complete

**Assigned Agents**: go-software-agent, devops-agent

---

## 4. Component Specifications

### 4.1 API Server (`src/api/`)

**Purpose**: REST API gateway for agent execution requests

**Inputs**:

```json
{
  "agent": "go-software-agent",      // Optional: explicit agent selection
  "prompt": "Write a function...",    // Required: user prompt
  "context": { ... }                  // Optional: additional context
}
```

**Outputs**:

```json
{
  "execution_id": "exec_abc123",
  "status": "completed",
  "agent": "go-software-agent",
  "result": {
    "output": "Here is the function...",
    "patterns_used": ["error-handling-pattern", "..."]
  },
  "timing": {
    "total_ms": 5000,
    "routing_ms": 10,
    "pattern_query_ms": 200,
    "claude_ms": 4790
  }
}
```

**Key Interfaces**:

- `Router` - Determines target agent
- `MCPClient` - Queries patterns from Cognee
- `ClaudeClient` - Executes agent via Claude API

**Dependencies**:

- Cognee MCP Server (pattern queries)
- Claude API (agent execution)

### 4.2 Deterministic Router (`src/api/internal/router/`)

**Purpose**: Code-based agent selection based on prompt analysis

**Behavior**:

- Accept prompt and optional explicit agent override
- Match keywords to determine agent (e.g., "go", "golang" → go-software-agent)
- Return selected agent with confidence score and reason
- MVP default: go-software-agent

**Dependencies**: None (standalone logic)

### 4.3 MCP Client (`src/api/internal/mcp/`)

**Purpose**: Communicate with Cognee MCP server for pattern queries

**Behavior**:

- Connect to Cognee via SSE transport
- Send search queries with optional domain filters
- Parse pattern results (ID, title, content, relevance score)
- Handle connection lifecycle, retries, and health checks

**Dependencies**: Cognee MCP Server (network)

### 4.4 Claude Client (`src/api/internal/claude/`)

**Purpose**: Execute agents via Claude Messages API

**Behavior**:

- Assemble prompt from agent definition, user prompt, and retrieved patterns
- Send to Claude Messages API with tool definitions
- Handle tool calls (route pattern searches back to MCP client)
- Extract final response and token usage

**Dependencies**: Claude API (Anthropic)

### 4.5 CLI Client (`src/cli/`)

**Purpose**: Command-line interface for interacting with ace-api

**Location**: Separate Go module at `src/cli/`

**Commands**:

```
ace execute [--agent <agent>] --prompt <prompt>
ace version
```

**Inputs**: Command-line arguments and flags

**Outputs**: Formatted console output

**Dependencies**: ace-api (HTTP)

---

## 5. Data Models

### MVP Simplified Models

For MVP, we minimize data persistence complexity:

| Model           | Storage     | Notes                     |
| --------------- | ----------- | ------------------------- |
| ExecutionLog    | File/stdout | JSON lines format         |
| AgentDefinition | Filesystem  | Load from markdown files  |
| RoutingRules    | YAML config | Static configuration      |
| Patterns        | Cognee      | Already managed by Cognee |

### ExecutionLog Schema (JSON Lines)

```json
{
  "execution_id": "exec_abc123",
  "timestamp": "2025-01-08T10:30:00Z",
  "agent": "go-software-agent",
  "prompt_hash": "sha256:abc...",
  "routing_decision": {
    "reason": "keyword match: golang",
    "confidence": 0.95
  },
  "patterns_queried": 3,
  "patterns_used": ["pattern-123", "pattern-456"],
  "tokens": {
    "input": 1500,
    "output": 800
  },
  "duration_ms": 5000,
  "status": "success"
}
```

### AgentDefinition (From Existing Files)

Located at `agents/definitions/go/go-software-agent.md`

The markdown frontmatter provides:

```yaml
name: go software agent
description: Expert Go engineer...
model: inherit
allowed_tools: [...]
```

### What's Deferred from Full Architecture

| Full Architecture         | MVP Simplification              |
| ------------------------- | ------------------------------- |
| PostgreSQL users table    | Not needed (no auth)            |
| PostgreSQL teams table    | Not needed (single user)        |
| PostgreSQL api_keys table | Not needed (no auth)            |
| PostgreSQL usage_records  | File-based logging              |
| Redis cache               | Direct queries (optimize later) |

---

## 6. Integration Points

### 6.1 ACE API <-> Cognee MCP

**Protocol**: MCP over SSE (Server-Sent Events)

**Endpoint**: `http://cognee-mcp:8000/sse` (within Docker network)

**Tool Used**: `search` (Cognee's built-in search tool)

**Request Flow**:

```
1. ace-api establishes SSE connection to Cognee MCP
2. ace-api sends tool call: search(query="golang error handling", search_type="GRAPH_COMPLETION")
3. Cognee processes query against knowledge graph
4. Cognee returns pattern results via SSE
5. ace-api parses results and includes in Claude prompt
```

**Error Handling**:

- Connection timeout: 5 seconds
- Query timeout: 10 seconds
- Retry on connection failure: 3 attempts with exponential backoff
- Fallback: Execute without patterns if Cognee unavailable

### 6.2 ACE API <-> Claude API

**Protocol**: HTTPS REST (Claude Messages API)

**Endpoint**: `https://api.anthropic.com/v1/messages`

**Request Flow**:

```
1. ace-api assembles prompt:
   - System: Agent definition (from go-software-agent.md)
   - System: Retrieved patterns
   - User: Original prompt
2. ace-api sends to Claude API with tool definitions
3. Claude executes, may call pattern search tool
4. ace-api handles tool calls (queries Cognee)
5. ace-api returns tool results to Claude
6. Claude completes response
7. ace-api extracts and returns final output
```

**Tool Definition for Claude**:

```json
{
  "name": "search_patterns",
  "description": "Search for relevant coding patterns and best practices",
  "input_schema": {
    "type": "object",
    "properties": {
      "query": {
        "type": "string",
        "description": "Search query for patterns"
      },
      "domains": {
        "type": "array",
        "items": { "type": "string" },
        "description": "Filter by domains (e.g., golang, error-handling)"
      }
    },
    "required": ["query"]
  }
}
```

### 6.3 Docker Network Communication

```
                    +------------------------------------------+
                    |           ace_network                    |
                    |                                          |
   Host:3000  ------|-> ace-api ----+-> cognee-mcp:8000        |
                    |               |                          |
                    |               +-> cognee-api:8000        |
                    |                      |                   |
                    |              +-------+-------+           |
                    |              v               v           |
                    |         postgres:5432    neo4j:7687      |
                    +------------------------------------------+
```

---

## 7. Local Development Setup

Devops agent responsibility. Requirements:

- Extend existing Cognee stack (`memory-mcp-server/docker-compose.yaml`)
- Add ace-api service (build from `src/api/`)
- Configure networking between ace-api and Cognee services
- Create `.env.example` with required variables (ANTHROPIC_API_KEY, LLM_API_KEY)
- Create Dockerfile for Go API server
- Full stack starts with `docker compose up`

---

## 8. Testing Strategy

### 8.1 What Must Be Tested for MVP

| Test Type          | Scope                            | Automation                     |
| ------------------ | -------------------------------- | ------------------------------ |
| Unit Tests         | Router logic, request parsing    | Automated (go test)            |
| Integration Tests  | MCP client <-> Cognee            | Automated with test containers |
| Integration Tests  | Claude client <-> Claude API     | Manual (API costs)             |
| E2E Tests          | Full flow CLI -> API -> Response | Manual test scripts            |
| Health Check Tests | Service startup and readiness    | Automated                      |

### 8.2 Unit Test Coverage (Automated)

**Router Tests** (`src/api/internal/router/router_test.go`):

- Keyword matching returns correct agent
- Multiple keyword matches use priority
- Unknown prompts use default agent
- Same prompt always returns same result (determinism)

**API Handler Tests** (`src/api/internal/api/handlers_test.go`):

- Valid request parses correctly
- Missing prompt returns 400
- Response format matches spec

**MCP Client Tests** (`src/api/internal/mcp/client_test.go`):

- Connection establishment
- Query formatting
- Response parsing
- Error handling

### 8.3 E2E Test Script

Go e2e test agent responsibility. Must validate:

- Health check endpoint
- Simple execution flow
- Routing determinism (same prompt → same agent)
- Pattern querying (patterns returned in response)

---

## 9. Risks and Mitigations

### 9.1 Technical Risks

| Risk                            | Probability | Impact | Mitigation                                                                     |
| ------------------------------- | ----------- | ------ | ------------------------------------------------------------------------------ |
| **MCP protocol complexity**     | Medium      | High   | Start with Cognee's existing MCP examples; use SSE which is simpler than stdio |
| **Claude tool calling latency** | Medium      | Medium | Set appropriate timeouts; optimize pattern queries to minimize round-trips     |
| **Cognee query performance**    | Low         | Medium | Pre-load patterns before MVP demo; monitor query times                         |
| **Docker networking issues**    | Low         | Medium | Use well-documented Docker Compose patterns; test networking early             |
| **Claude API rate limits**      | Low         | Low    | Local development won't hit limits; implement retry logic                      |

### 9.2 Potential Blockers

| Blocker                                     | Detection                               | Resolution                                                                     |
| ------------------------------------------- | --------------------------------------- | ------------------------------------------------------------------------------ |
| **MCP SSE connection unstable**             | Connection drops during testing         | Implement reconnection logic; consider fallback to REST API                    |
| **Patterns not returning relevant results** | Test queries return empty or irrelevant | Review pattern content; adjust Cognee search type (GRAPH_COMPLETION vs CHUNKS) |
| **Claude not using patterns effectively**   | Agent outputs don't reflect patterns    | Refine system prompt to better instruct pattern usage                          |
| **Memory issues with Cognee**               | Container OOM kills                     | Increase Docker resource limits; reduce batch sizes                            |

### 9.3 Risk Mitigation Actions

1. **Week 1**: Validate MCP connection works with simple test before building full client
2. **Week 2**: Create integration test for Cognee queries; verify patterns return expected results
3. **Week 3**: Test Claude tool calling separately before integrating with full flow
4. **Ongoing**: Maintain escape hatches (e.g., bypass Cognee if unavailable, use static patterns)

---

## 10. Agent Task Breakdown

### Recommended Agent Assignments

| Phase                           | Primary Agent      | Supporting Agents | Tasks                                                  |
| ------------------------------- | ------------------ | ----------------- | ------------------------------------------------------ |
| **Phase 1: Foundation**         | go-architect-agent | devops-agent      | Project structure, API contract design, Docker Compose |
| **Phase 2: Core API**           | go-software-agent  | -                 | HTTP server, handlers, configuration, logging          |
| **Phase 3: MCP Integration**    | go-software-agent  | -                 | MCP client, SSE connection, pattern queries            |
| **Phase 4: Claude Integration** | go-software-agent  | -                 | Claude client, tool handling, prompt assembly          |
| **Phase 5: Router**             | go-software-agent  | -                 | Routing logic, keyword matching, configuration         |
| **Phase 6: CLI**                | go-software-agent  | -                 | Cobra CLI, API client, formatting                      |
| **Phase 7: Testing**            | go-e2e-test-agent  | bats-test-agent   | Integration tests, manual test scripts                 |
| **Phase 8: Polish**             | go-software-agent  | devops-agent      | Bug fixes, performance, documentation                  |

### Suggested Order of Agent Involvement

```
Week 1:
├── go-architect-agent: Design project structure and API contract
└── devops-agent: Create Docker Compose configuration

Week 2-3:
└── go-software-agent: Implement core API server

Week 3-4:
└── go-software-agent: Implement MCP and Claude clients

Week 5-6:
└── go-software-agent: Implement router and CLI

Week 7:
├── go-e2e-test-agent: Create integration tests
└── bats-test-agent: Create shell-based test scripts

Week 8:
├── go-software-agent: Bug fixes and polish
└── devops-agent: Final Docker configuration
```

### Agent Instructions Summary

**For go-architect-agent (Phase 1)**:

```
Design the API server and CLI project structure for ACE MVP.
- API server at src/api/ using Gin framework
- CLI client at src/cli/ using Cobra (separate Go module)
- Create OpenAPI spec for /api/v1/execute, /health, /ready
- Define internal package structure for both modules
- Output: Project scaffold and API contract
```

**For go-software-agent (Phases 2-6)**:

```
Implement ACE API server components:
- Phase 2: HTTP server with Gin, configuration, logging (src/api/)
- Phase 3: MCP client for Cognee SSE connection (src/api/)
- Phase 4: Claude Messages API client with tool calling (src/api/)
- Phase 5: Keyword-based deterministic router (src/api/)
- Phase 6: Cobra CLI client (src/cli/ - separate Go module)

Key files:
- API server source: src/api/
- CLI client source: src/cli/
- Agent definition: agents/definitions/go/go-software-agent.md
- Existing patterns: agent-patterns/
- Cognee compose: memory-mcp-server/docker-compose.yaml
```

**For devops-agent (Phases 1, 8)**:

```
Configure local Docker development environment:
- Extend existing Cognee Docker Compose
- Add ace-api service (build context: src/api/)
- Configure networking between services
- Create Dockerfile at src/api/Dockerfile
```

**For go-e2e-test-agent (Phase 7)**:

```
Create integration tests for ACE MVP:
- Test full execution flow
- Test routing determinism
- Test pattern querying
- Use testcontainers for Cognee dependencies
```

---

## Appendix A: Key File Locations

| Purpose                         | Path                                    |
| ------------------------------- | --------------------------------------- |
| Architecture Docs               | `docs/architecture/`                    |
| Existing Cognee Compose         | `memory-mcp-server/docker-compose.yaml` |
| Agent Definitions               | `agents/definitions/`                   |
| Agent Patterns                  | `agent-patterns/`                       |
| Pattern Load Script             | `scripts/load-patterns.sh`              |
| Pattern Process Script          | `scripts/cognify-patterns.sh`           |
| API Server (to create)          | `src/api/`                              |
| CLI Client (to create)          | `src/cli/`                              |
| Root Docker Compose (to create) | `docker-compose.yaml`                   |

---

## Appendix B: API Contract

API architect agent designs the OpenAPI YAML spec. Go software agent implements it.

### POST /api/v1/execute

**Request**:

```json
{
  "agent": "go-software-agent",
  "prompt": "Write a function to reverse a string in Go"
}
```

**Response (200 OK)**:

```json
{
  "execution_id": "exec_20250108_abc123",
  "status": "completed",
  "agent": "go-software-agent",
  "result": {
    "output": "Here's a function to reverse a string in Go...",
    "patterns_used": ["string-manipulation-pattern"]
  },
  "timing": {
    "total_ms": 5234,
    "routing_ms": 12,
    "pattern_query_ms": 187,
    "claude_ms": 5035
  }
}
```

**Response (400 Bad Request)**:

```json
{
  "error": {
    "code": "invalid_request",
    "message": "prompt is required"
  }
}
```

### GET /health

Defined by `twistingmercury/heartbeat` package. No custom spec needed.

---

## Appendix C: Glossary

| Term                      | Definition                                                 |
| ------------------------- | ---------------------------------------------------------- |
| **ACE**                   | Agentic Coding Engine - this project                       |
| **Cognee**                | Knowledge graph system for pattern storage and retrieval   |
| **MCP**                   | Model Context Protocol - standard for LLM tool integration |
| **Pattern**               | Reusable coding best practice stored in Cognee             |
| **Deterministic Routing** | Code-based agent selection (same input = same output)      |
| **Agent**                 | Specialized Claude prompt for a specific task domain       |
