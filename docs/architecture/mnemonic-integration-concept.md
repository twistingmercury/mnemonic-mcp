# ACE + Mnemonic Integration Concept

[Back to Overview](00-overview.md)

## Unified Architecture: Mnemonic as the Backend

In this model, Mnemonic serves as the single backend service with ACE-specific endpoints.
ACE CLI orchestrates calls to Mnemonic and Claude Code.

### Phase 1: With Claude Code

```mermaid
sequenceDiagram
    participant User
    participant CLI as ACE CLI
    participant MN as Mnemonic Server
    participant CC as Claude Code

    User->>CLI: "Write a Go function to sum numbers"

    Note over CLI,MN: Step 1: Get routing decision
    CLI->>MN: POST /ace/route
    MN->>MN: Evaluate routing rules (code-based)
    MN-->>CLI: { agent: "go-software-agent", confidence: 1.0 }

    Note over CLI,MN: Step 2: Get relevant patterns
    CLI->>MN: GET /ace/patterns?agent="go-software-agent"&context="sum function"
    MN->>MN: Query knowledge graph
    MN-->>CLI: { patterns: [...], system_prompt: "..." }

    Note over CLI,CC: Step 3: Execute via Claude Code
    CLI->>CLI: Assemble Claude Code invocation
    CLI->>CC: claude --agent go-software-agent<br/>--agents '{patterns...}'<br/>-p "Write a Go function to sum numbers"
    CC->>CC: Execute with Anthropic API
    CC->>CC: Tool calls: Write("math.go", content)
    CC-->>CLI: { success: true, files_modified: ["math.go"] }

    CLI-->>User: Done! Created math.go
```

### Phase 2: Direct Anthropic API (Future)

```mermaid
sequenceDiagram
    participant User
    participant CLI as ACE CLI
    participant MN as Mnemonic Server
    participant ANT as Anthropic API

    User->>CLI: "Write a Go function to sum numbers"

    Note over CLI,MN: Step 1: Get routing decision
    CLI->>MN: POST /ace/route
    MN-->>CLI: { agent: "go-software-agent" }

    Note over CLI,MN: Step 2: Get relevant patterns
    CLI->>MN: GET /ace/patterns?agent="go-software-agent"&context="sum function"
    MN-->>CLI: { patterns: [...], system_prompt: "..." }

    Note over CLI,ANT: Step 3: Direct API call
    CLI->>ANT: POST /v1/messages<br/>{ model, system, messages, tools }
    ANT-->>CLI: { tool_use: "write_file", path: "math.go", content: "..." }

    Note over CLI: Step 4: Local tool execution
    CLI->>CLI: Execute write_file("math.go", content)

    CLI-->>User: Done! Created math.go
```

## Mnemonic API Endpoints

### ACE-Specific Endpoints

| Endpoint | Purpose |
|----------|---------|
| `POST /ace/route` | Determine which agent handles a prompt |
| `GET /ace/patterns` | Retrieve patterns for a specific agent + context |
| `GET /ace/agents` | List available agents and their capabilities |
| `PUT /ace/rules` | Update routing rules (admin) |

### General Memory Endpoints (for other tools)

| Endpoint | Purpose |
|----------|---------|
| `POST /memory/store` | Store knowledge/patterns |
| `GET /memory/search` | Semantic search across knowledge |
| `GET /memory/graph` | Query knowledge graph relationships |

## What Lives Where

| Component | Location | Responsibility |
|-----------|----------|----------------|
| **Routing rules** | Mnemonic | Stored as queryable knowledge |
| **Patterns** | Mnemonic | Stored in knowledge graph |
| **Agent definitions** | Mnemonic | Stored as structured data |
| **Routing logic** | Mnemonic | Code-based evaluation |
| **Prompt assembly** | ACE CLI | Combines route + patterns + user prompt |
| **Claude Code invocation** | ACE CLI | Builds and executes command |
| **Tool execution** | ACE CLI / Claude Code | Local filesystem operations |

## Benefits of This Model

1. **Single backend**: Only Mnemonic to deploy/manage
2. **ACE CLI is lightweight**: Just orchestration, no server logic
3. **Mnemonic is reusable**: Other tools can use memory endpoints
4. **Clean separation**: Knowledge storage (Mnemonic) vs orchestration (CLI)
5. **Routing as data**: Rules stored alongside patterns, version controlled
