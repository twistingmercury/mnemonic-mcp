# Communication Patterns

[Back to Overview](00-overview.md) | [Back to Project README](../../README.md)

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

| Endpoint                | Method | Purpose                               |
| ----------------------- | ------ | ------------------------------------- |
| `/v1/ace/route`         | POST   | Deterministic routing based on prompt |
| `/v1/ace/patterns`      | GET    | Pattern retrieval for agent + context |
| `/v1/ace/agents`        | GET    | List available agents                 |
| `/v1/ace/agents/{name}` | GET    | Get agent details                     |

> **Note:** This table shows primary endpoints. See the [API Specification](../design/api-specification.md) for the complete endpoint reference including patterns and routing-rules CRUD operations.

### Request Flow

```mermaid
sequenceDiagram
    participant CLI as ACE CLI
    participant MN as Mnemonic

    CLI->>MN: POST /v1/ace/route
    Note right of MN: Validate request
    Note right of MN: Apply routing rules
    Note right of MN: Fetch patterns from storage
    MN-->>CLI: {agent_name, patterns, metadata}

    alt Error Case
        MN-->>CLI: HTTP Error Response
        Note left of CLI: Handle gracefully
    end
```

**Request Characteristics:**

- Synchronous request-response
- Contains full prompt for accurate routing decisions
- Includes context hints for better routing
- Authenticated per team/user

**Request Body for `/v1/ace/route`:**

| Field     | Purpose                          |
| --------- | -------------------------------- |
| `prompt`  | Full prompt for routing decision |
| `context` | Domain, task type, preferences   |
| `options` | Optional routing configuration   |

### Response Structure

The response provides everything the CLI needs for local execution.

**Response Fields:**

| Field        | Purpose                                   |
| ------------ | ----------------------------------------- |
| `agent_name` | Which agent to invoke                     |
| `patterns`   | Retrieved patterns for context enrichment |
| `hints`      | Suggested parameters for Claude Code      |
| `metadata`   | Routing rationale for logging/debugging   |

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

| HTTP Status   | Meaning      | CLI Behavior                               |
| ------------- | ------------ | ------------------------------------------ |
| 400           | Bad Request  | Display validation errors                  |
| 401           | Unauthorized | Prompt for re-authentication               |
| 404           | Not Found    | Agent or pattern not found                 |
| 500           | Server Error | Retry with backoff, then fail gracefully   |
| Network Error | Unreachable  | Post-MVP: Fallback behavior to be designed |

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

| Aspect           | Detail                           |
| ---------------- | -------------------------------- |
| Method           | Subprocess spawn                 |
| Prompt passing   | Post-MVP: Details to be designed |
| Output capture   | Post-MVP: Details to be designed |
| Timeout handling | Post-MVP: Details to be designed |

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

| Scenario             | Fallback                                   |
| -------------------- | ------------------------------------------ |
| Mnemonic unreachable | Post-MVP: Fallback behavior to be designed |
| Claude Code fails    | Display error, suggest retry               |

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
        PROMPT[User Prompts<br/>Sent for routing]
        PATTERNS[Patterns<br/>Team-shared]
        ROUTES[Routes<br/>Configuration]
        CREDS[Credentials<br/>CLI only]
    end

    PROMPT -->|"Sent for routing<br/>(not persisted)"| MN[Mnemonic]
    PATTERNS -->|"Stored in Mnemonic"| MN
    ROUTES -->|"Managed in Mnemonic"| MN
    CREDS -->|"Stays local"| CLI[ACE CLI]
```

**Key Principles:**

- Full prompts sent to Mnemonic for routing (not persisted)
- Mnemonic is organization-controlled infrastructure (not a third-party service)
- Routing accuracy requires full prompt for keyword matching, regex, and semantic similarity
- Patterns are team-shared (access controlled)
- Credentials never leave CLI
- Actual LLM calls go directly from CLI to Anthropic API (not through Mnemonic)

**Next:** [Deployment Architecture](05-deployment-architecture.md)
