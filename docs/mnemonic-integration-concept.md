# Mnemonic Client Integration Concept

[Back to Architecture Overview](architecture/00-overview.md) |
[Back to Project README](../README.md)

## Unified Architecture: Mnemonic as the Backend

In this model, Mnemonic serves as the single backend service providing routing,
pattern retrieval, and agent management capabilities. Client applications
orchestrate calls to Mnemonic and AI services like Claude Code or the
Anthropic API directly.

> **What is a "Client"?** Any application that consumes Mnemonic's API: a CLI
> tool (like ACE CLI), a web application, an IDE extension, an automated
> business process, or a custom integration script.

### Phase 1: Claude Code Custom Commands

In Phase 1, Claude Code is the client. Users invoke custom commands (skills) like
`/review`, `/document`, `/design`, `/test`, etc. These commands query Mnemonic
for routing and patterns, then execute with enriched context.

```mermaid
sequenceDiagram
    participant User
    participant CC as Claude Code
    participant MN as Mnemonic Server

    User->>CC: /review (or /document, /design, /test, etc.)

    Note over CC,MN: Step 1: Get routing decision + patterns
    CC->>MN: POST /v1/api/route
    MN->>MN: Evaluate routing rules
    MN->>MN: Query knowledge graph for patterns
    MN-->>CC: { agent: "code-review-agent",<br/>patterns: [...] }

    Note over CC: Step 2: Execute with enriched context
    CC->>CC: Apply patterns to system prompt
    CC->>CC: Execute with Anthropic API
    CC->>CC: Tool calls as needed

    CC-->>User: Review complete!
```

### Phase 2: Other Clients (Web Apps, CLI Tools, etc.)

Other clients (web applications, the ACE CLI, business processes) can integrate
with Mnemonic and call the Anthropic API directly for custom workflows.

```mermaid
sequenceDiagram
    participant User
    participant Client as Client (Web/CLI/Async Process, etc.)
    participant MN as Mnemonic Server
    participant ANT as Anthropic API

    User->>Client: "Write a Go function to sum numbers"

    Note over Client,MN: Step 1: Get routing decision + patterns
    Client->>MN: POST /v1/api/route
    MN-->>Client: { agent: "go-software-agent",<br/>patterns: [...] }

    Note over Client,ANT: Step 2: Direct API call
    Client->>ANT: POST /v1/messages<br/>{ model, system, messages }
    ANT-->>Client: { tool_use: "write_file",<br/>path: "math.go" }

    Note over Client: Step 3: Local tool execution
    Client->>Client: Execute write_file("math.go", content)

    Client-->>User: Done! Created math.go
```

## Mnemonic API Endpoints

### Client Integration Endpoints

| Endpoint                         | Purpose                                |
| -------------------------------- | -------------------------------------- |
| `POST /v1/api/route`             | Determine which agent handles a prompt |
| `GET /v1/api/patterns`           | Retrieve patterns for agent + context  |
| `GET /v1/api/agents`             | List available agents and capabilities |
| `PUT /v1/api/routing-rules/{id}` | Update routing rules (admin)           |

## What Lives Where

| Component                 | Location            | Responsibility                   |
| ------------------------- | ------------------- | -------------------------------- |
| **Routing rules**         | Mnemonic            | Queryable knowledge storage      |
| **Patterns**              | Mnemonic            | Knowledge graph storage          |
| **Agent definitions**     | Mnemonic            | Structured data storage          |
| **Routing logic**         | Mnemonic            | Code-based evaluation            |
| **Prompt assembly**       | Client              | Combines route, patterns, prompt |
| **AI service invocation** | Client              | Builds and executes commands     |
| **Tool execution**        | Client / AI Service | Local filesystem operations      |

## Benefits of This Model

1. **Single backend**: Only Mnemonic to deploy and manage
2. **Lightweight clients**: Just orchestration, no server logic required
3. **Clean separation**: Knowledge storage (Mnemonic) vs orchestration
4. **Routing as data**: Rules stored alongside patterns, version controlled
5. **Flexible integration**: Clients can use Claude Code or Anthropic API
6. **Reusable patterns**: Multiple clients share the same knowledge base
