# System Architecture

[Back to Overview](00-overview.md) | [Back to Project README](../../README.md)

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Component Breakdown](#component-breakdown)
  - [ACE CLI](#ace-cli)
  - [Mnemonic](#mnemonic)
- [Data Flow](#data-flow)
- [CLI-Centric Model](#cli-centric-model)
- [Component Interactions](#component-interactions)
- [Boundary Definitions](#boundary-definitions)

## Architecture Overview

ACE follows a distributed architecture with clear separation between client-side execution and server-side orchestration.

```mermaid
graph TB
    subgraph "User Workstation"
        CLI[ACE CLI]
        CC[Claude Code]
        FS[Local Filesystem]
    end

    subgraph "Service Tier"
        MN[Mnemonic]
    end

    subgraph "Data Tier"
        PG[(Postgres + PGVector)]
        NEO[(Neo4j)]
    end

    CLI <-->|"REST"| MN
    MN <--> PG
    MN <--> NEO
    CLI <--> CC
    CC <--> FS
```

## Component Breakdown

### ACE CLI

The CLI is the primary user interface and orchestrates local execution.

**Responsibilities:**

- Accept user prompts and commands
- Request routing decisions from Mnemonic
- Construct enriched prompts with patterns and context
- Invoke Claude Code (Phase 1) or Anthropic API (Phase 2)
- Display results to the user

**Key Characteristics:**

- Runs on user workstation
- Stateless between invocations (state lives in Mnemonic)
- Handles authentication to external services
- Manages local execution environment

```mermaid
graph TB
    subgraph "ACE CLI Internal Structure"
        INPUT[Input Handler]
        AUTH[Auth Manager]
        ROUTE[Routing Client]
        PROMPT[Prompt Builder]
        EXEC[Execution Engine]
        OUTPUT[Output Handler]
    end

    INPUT --> AUTH
    AUTH --> ROUTE
    ROUTE --> PROMPT
    PROMPT --> EXEC
    EXEC --> OUTPUT
```

### Mnemonic

Mnemonic is the backend server that provides routing decisions and pattern retrieval via REST API. For MVP, Mnemonic serves only ACE (not a general-purpose memory service).

**Responsibilities:**

- Receive routing requests from CLI instances
- Apply deterministic routing logic to select the appropriate agent
- Retrieve relevant patterns for context enrichment
- Return routing decision and patterns to CLI

**Key Characteristics:**

- Lightweight service (no LLM calls)
- Deterministic routing (code-based logic)
- Stateless request handling
- REST API interface
- Full storage stack: Postgres + PGVector + Neo4j

```mermaid
graph TB
    subgraph "Mnemonic Internal Structure"
        REST[REST API Handler]
        VALID[Request Validator]
        ROUTER[Routing Engine]
        PATTERN[Pattern Enrichment Service]
        RESP[Response Builder]
    end

    subgraph "Storage Layer"
        PG[(Postgres)]
        PGV[(PGVector)]
        NEO[(Neo4j)]
    end

    REST --> VALID
    VALID --> ROUTER
    ROUTER --> PATTERN
    PATTERN --> RESP
    PATTERN <--> PG
    PATTERN <--> PGV
    PATTERN <--> NEO
```

See [Communication Patterns](04-communication-patterns.md#rest-endpoints) for REST endpoint details.

**Routing Rule Cache (MVP):**

- Routing rules are loaded once at startup from the database
- Service restart is required to reload rules if they change
- Background refresh with configurable TTL is planned for Post-MVP

**What Mnemonic Does NOT Do:**

- Make LLM API calls
- Store user credentials
- Execute tools or file operations
- Maintain session state

## Data Flow

The following diagram shows the complete data flow for a typical request.

```mermaid
sequenceDiagram
    participant U as User
    participant CLI as ACE CLI
    participant MN as Mnemonic
    participant CC as Claude Code
    participant FS as Filesystem

    U->>CLI: Submit prompt

    Note over CLI: Parse and validate input

    CLI->>MN: POST /v1/api/route

    Note over MN: Apply routing rules
    Note over MN: Fetch patterns from storage

    MN-->>CLI: {agent, patterns, metadata}

    Note over CLI: Construct enriched prompt

    CLI->>CC: Invoke with enriched prompt

    Note over CC: Process request

    CC->>FS: Read/write files
    FS-->>CC: File contents

    CC-->>CLI: Execution results

    Note over CLI: Format output

    CLI-->>U: Display results
```

## CLI-Centric Model

ACE follows a CLI-centric model where:

1. **CLI is the orchestrator**: The CLI coordinates between user, Mnemonic, and execution engine
2. **Mnemonic is advisory**: Mnemonic provides routing decisions but does not execute
3. **Execution is local**: All LLM interactions and tool execution happen on the workstation

```mermaid
graph TB
    subgraph "Control Flow"
        USER((User))
        CLI[ACE CLI<br/>Orchestrator]
        MN[Mnemonic<br/>Advisory]
        EXEC[Execution<br/>Local]
    end

    USER -->|"Commands"| CLI
    CLI -->|"POST /v1/api/route"| MN
    MN -->|"Agent + Patterns"| CLI
    CLI -->|"Execute"| EXEC
    EXEC -->|"Results"| CLI
    CLI -->|"Output"| USER
```

**Benefits of CLI-Centric Model:**

- No server-side LLM costs
- User data stays local
- Works offline after routing decision (with caching)
- Leverages existing Claude Code setup

## Component Interactions

### CLI to Mnemonic

| Aspect            | Detail                                       |
| ----------------- | -------------------------------------------- |
| Protocol          | REST (HTTP/HTTPS)                            |
| Authentication    | To be specified in design phase              |
| Request contains  | Full prompt, context hints, user preferences |
| Response contains | Agent identifier, patterns, execution hints  |

**Note:** Full prompts are sent to Mnemonic for routing but are not persisted. Mnemonic is organization-controlled infrastructure and requires the full prompt for accurate routing via keyword matching, regex, and semantic similarity.

See [Communication Patterns](04-communication-patterns.md#rest-endpoints) for REST endpoint details.

### CLI to Claude Code

| Aspect            | Detail                                                                 |
| ----------------- | ---------------------------------------------------------------------- |
| Invocation method | Direct subprocess invocation (see ADR-003 in Architectural Decisions) |
| Context passing   | Enriched prompt with routing decision and patterns from Mnemonic       |
| Result capture    | Standard output/error streams from Claude Code process                 |

See [ADR-003: Claude Code Integration Strategy](02-architectural-decisions.md#adr-003-claude-code-integration-strategy) for the full specification of the execution model.

## Boundary Definitions

Clear boundaries separate concerns between components.

```mermaid
graph TB
    subgraph "User Domain"
        UD1[User prompts]
        UD2[Local files]
        UD3[Credentials]
    end

    subgraph "CLI Domain"
        CD1[Input parsing]
        CD2[Prompt construction]
        CD3[Execution orchestration]
        CD4[Output formatting]
    end

    subgraph "Mnemonic Domain"
        MD1[Routing logic]
        MD2[Pattern retrieval]
        MD3[Request validation]
        MD4[Pattern storage]
    end

    UD1 --> CD1
    UD3 --> CD3
    CD1 --> MD3
    MD1 --> CD2
    MD2 --> CD2
    CD3 --> UD2
```

**Boundary Rules:**

- User credentials never leave the CLI
- Routing logic lives only in Mnemonic
- Pattern storage lives only in Mnemonic
- File operations happen only on the workstation

**Next:** [Communication Patterns](04-communication-patterns.md)
