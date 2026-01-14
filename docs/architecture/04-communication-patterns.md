# Communication Patterns

[Back to Overview](00-overview.md) | [Back to Documentation Index](../README.md)

## Table of Contents

- [Overview](#overview)
- [CLI to Mnemonic Communication](#cli-to-mnemonic-communication)
  - [REST Endpoints](#rest-endpoints)
  - [Request Flow](#request-flow)
  - [Response Structure](#response-structure)
  - [Error Handling](#error-handling)
- [CLI to Claude Code Communication](#cli-to-claude-code-communication)
- [Resilience Patterns](#resilience-patterns)
- [Security Considerations](#security-considerations)

## Overview

ACE uses distinct communication patterns for each component boundary.

```mermaid
graph LR
    subgraph "Communication Channels"
        CLI[ACE CLI]
        MN[Mnemonic]
        CC[Claude Code]
    end

    CLI -->|"REST<br/>Sync"| MN
    CLI -->|"Local<br/>Subprocess"| CC
```

## CLI to Mnemonic Communication

The CLI communicates with Mnemonic via REST for routing decisions and pattern retrieval.

### REST Endpoints

Mnemonic exposes the following REST endpoints for ACE:

| Endpoint             | Method | Purpose                               |
| -------------------- | ------ | ------------------------------------- |
| `/ace/route`         | GET    | Deterministic routing based on prompt |
| `/ace/patterns`      | GET    | Pattern retrieval for agent + context |
| `/ace/agents`        | GET    | List available agents                 |
| `/ace/agents/{name}` | GET    | Get agent details                     |

### Request Flow

```mermaid
sequenceDiagram
    participant CLI as ACE CLI
    participant MN as Mnemonic

    CLI->>MN: GET /ace/route?prompt=...
    Note right of MN: Validate request
    Note right of MN: Apply routing rules
    Note right of MN: Fetch patterns from storage
    MN-->>CLI: {agent, patterns, metadata}

    alt Error Case
        MN-->>CLI: HTTP Error Response
        Note left of CLI: Handle gracefully
    end
```

**Request Characteristics:**

- Synchronous request-response
- Contains prompt summary (not full prompt for privacy)
- Includes context hints for better routing
- Authenticated per team/user

**Query Parameters for `/ace/route`:**

| Parameter | Purpose                                 |
| --------- | --------------------------------------- |
| `prompt`  | Prompt summary for routing decision     |
| `context` | Domain, task type, preferences          |
| `user`    | User/team identifier for access control |

### Response Structure

The response provides everything the CLI needs for local execution.

**Response Fields:**

| Field      | Purpose                                   |
| ---------- | ----------------------------------------- |
| `agent`    | Which agent to invoke                     |
| `patterns` | Retrieved patterns for context enrichment |
| `hints`    | Suggested parameters for Claude Code      |
| `metadata` | Routing rationale for logging/debugging   |

```mermaid
graph TB
    subgraph "Response Structure"
        AGENT[Agent Information]
        PATTERNS[Pattern Collection]
        HINTS[Execution Hints]
        META[Metadata]
    end

    AGENT --> |"What to invoke"| CLI[CLI Processing]
    PATTERNS --> |"Context enrichment"| CLI
    HINTS --> |"How to invoke"| CLI
    META --> |"Logging/debugging"| CLI
```

### Error Handling

The CLI must handle Mnemonic errors gracefully.

| HTTP Status   | Meaning      | CLI Behavior                             |
| ------------- | ------------ | ---------------------------------------- |
| 400           | Bad Request  | Display validation errors                |
| 401           | Unauthorized | Prompt for re-authentication             |
| 404           | Not Found    | Agent or pattern not found               |
| 500           | Server Error | Retry with backoff, then fail gracefully |
| Network Error | Unreachable  | To be specified in design phase          |

## CLI to Claude Code Communication

The CLI invokes Claude Code as a local subprocess for execution.

```mermaid
sequenceDiagram
    participant CLI as ACE CLI
    participant CC as Claude Code
    participant FS as Filesystem

    CLI->>CLI: Build enriched prompt<br/>(route + patterns)
    CLI->>CC: Invoke with prompt

    loop Execution
        CC->>FS: Tool operations
        FS-->>CC: Results
    end

    CC-->>CLI: Final output
```

**Invocation Characteristics:**

To be specified in [Configuration](../design/configuration.md).

| Aspect           | Detail                        |
| ---------------- | ----------------------------- |
| Method           | Subprocess spawn              |
| Prompt passing   | To be specified in design     |
| Output capture   | To be specified in design     |
| Timeout handling | To be specified in design     |

**Context Enrichment:**

The CLI constructs an enriched prompt by combining:

1. Original user prompt
2. Routing context from Mnemonic
3. Retrieved patterns
4. Execution hints

```mermaid
graph LR
    USER[User Prompt] --> BUILDER[Prompt Builder]
    ROUTE[Route Context] --> BUILDER
    PATTERNS[Patterns] --> BUILDER
    HINTS[Execution Hints] --> BUILDER
    BUILDER --> ENRICHED[Enriched Prompt]
    ENRICHED --> CC[Claude Code]
```

## Resilience Patterns

### Timeout Handling

Each communication channel has timeout considerations.

```mermaid
graph TB
    subgraph "Timeout Strategy"
        CLI_MN[CLI to Mnemonic<br/>Short timeout]
        CLI_CC[CLI to Claude Code<br/>Long timeout]
    end
```

| Channel            | Timeout Strategy                  |
| ------------------ | --------------------------------- |
| CLI to Mnemonic    | Short timeout, fail fast          |
| CLI to Claude Code | Long timeout, progress indication |

### Retry Logic

```mermaid
graph TB
    REQUEST[Request] --> ATTEMPT[Attempt]
    ATTEMPT -->|Success| DONE[Done]
    ATTEMPT -->|Transient Failure| BACKOFF[Exponential Backoff]
    BACKOFF --> ATTEMPT
    ATTEMPT -->|Permanent Failure| FAIL[Fail Gracefully]
    BACKOFF -->|Max Retries| FAIL
```

**Retry Considerations:**

- Idempotent operations only
- Exponential backoff
- Maximum retry limits
- Clear failure messaging

### Fallback Behavior

When components are unavailable:

| Scenario             | Fallback                        |
| -------------------- | ------------------------------- |
| Mnemonic unreachable | To be specified in design phase |
| Claude Code fails    | Display error, suggest retry    |

## Security Considerations

### Data in Transit

| Channel            | Security Requirement               |
| ------------------ | ---------------------------------- |
| CLI to Mnemonic    | TLS required; auth to be specified |
| CLI to Claude Code | Local only, no network             |

### Sensitive Data Handling

```mermaid
graph TB
    subgraph "Data Classification"
        PROMPT[User Prompts<br/>Sensitive]
        SUMMARY[Prompt Summary<br/>Minimal]
        PATTERNS[Patterns<br/>Team-shared]
        ROUTES[Routes<br/>Configuration]
    end

    PROMPT -->|"Stays in CLI"| CLI[ACE CLI]
    SUMMARY -->|"Sent to Mnemonic"| MN[Mnemonic]
    PATTERNS -->|"Stored in Mnemonic"| MN
    ROUTES -->|"Managed in Mnemonic"| MN
```

**Key Principles:**

- Full prompts stay local (CLI only)
- Only summaries sent to Mnemonic for routing
- Patterns are team-shared (access controlled)
- Credentials never leave CLI

**Next:** [Deployment Architecture](05-deployment-architecture.md)
