# System Architecture

[Back to Overview](00-overview.md) | [Back to Project README](../../README.md)

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Component Breakdown](#component-breakdown)
  - [Mnemonic Server](#mnemonic-server)
  - [Claude Code Integration](#claude-code-integration)
- [Data Flow](#data-flow)
- [Component Interactions](#component-interactions)
- [Boundary Definitions](#boundary-definitions)

## Architecture Overview

Mnemonic follows a single-server architecture with dual protocol interfaces: REST for administration and MCP for Claude Code integration.

```mermaid
graph TB
    subgraph "Claude Code"
        CC[Claude Code MCP Client]
    end

    subgraph "Admin Tools"
        CURL[curl/scripts]
    end

    subgraph "Mnemonic Server"
        subgraph "Two HTTP Listeners"
            ADMIN[Admin REST API :8080]
            MCP[MCP Server :8081]
        end
        PATTERN[Pattern Enrichment Service]
        TOOLING[Tooling Repository]
    end

    subgraph "Data Tier"
        PG[(Postgres + PGVector)]
        NEO[(Neo4j)]
    end

    CURL -->|"POST/PUT/DELETE"| ADMIN
    CC -->|"MCP: Read-only"| MCP
    ADMIN --> PATTERN
    ADMIN --> TOOLING
    MCP --> PATTERN
    MCP --> TOOLING
    PATTERN <--> PG
    PATTERN <--> NEO
    TOOLING <--> PG
```

## Component Breakdown

### Mnemonic Server

Mnemonic is a single Go server with two HTTP listeners providing team knowledge graph and tooling synchronization.

**Responsibilities:**

- **Admin REST API** (`:8080`): CRUD operations for patterns, agents, skills, commands
- **MCP Server** (`:8081`): Read-only access to knowledge graph and tooling for Claude Code
- **Pattern enrichment**: Semantic search via PGVector, knowledge graph via Neo4j
- **Tooling synchronization**: Shared agent, skill, command definitions across team

**Key Characteristics:**

- Single server process, two HTTP listeners
- Lightweight service (no LLM calls)
- Stateless request handling
- Dual protocol architecture (REST admin + MCP read-only)
- Full storage stack: Postgres + PGVector + Neo4j

```mermaid
graph TB
    subgraph "Mnemonic Internal Structure"
        subgraph "Admin REST API :8080"
            REST_HANDLER[HTTP Handler]
            REST_VALID[Request Validator]
            ADMIN_SVC[Admin Service]
        end

        subgraph "MCP Server :8081"
            MCP_HANDLER[MCP Handler]
            MCP_TOOLS[Tool Implementations]
        end

        subgraph "Core Services"
            PATTERN[Pattern Service]
            AGENT[Agent Service]
            SKILL[Skill Service]
            COMMAND[Command Service]
        end

        subgraph "Storage Layer"
            PG[(Postgres)]
            PGV[(PGVector)]
            NEO[(Neo4j)]
        end
    end

    REST_HANDLER --> REST_VALID
    REST_VALID --> ADMIN_SVC
    ADMIN_SVC --> PATTERN
    ADMIN_SVC --> AGENT
    ADMIN_SVC --> SKILL
    ADMIN_SVC --> COMMAND

    MCP_HANDLER --> MCP_TOOLS
    MCP_TOOLS --> PATTERN
    MCP_TOOLS --> AGENT
    MCP_TOOLS --> SKILL
    MCP_TOOLS --> COMMAND

    PATTERN <--> PG
    PATTERN <--> PGV
    PATTERN <--> NEO
    AGENT <--> PG
    SKILL <--> PG
    COMMAND <--> PG
```

**What Mnemonic Does NOT Do:**

- Make LLM API calls
- Store user credentials
- Route prompts to agents (user orchestrates)
- Execute tools or file operations
- Maintain session state

### Claude Code Integration

Claude Code integrates with Mnemonic via the MCP (Model Context Protocol) interface.

**MCP Tools Provided (11 total):**

| Tool | Purpose |
| ---- | ------- |
| `search_patterns` | Semantic search over team knowledge graph |
| `find_related_patterns` | Find patterns related to a given pattern |
| `get_pattern` | Retrieve specific pattern by ID |
| `list_agents` | List all available agents |
| `get_agent` | Get detailed agent information |
| `list_skills` | List all available skills |
| `get_skill` | Get detailed skill information |
| `get_skill_files` | Get skill child files (scripts, references, assets) |
| `list_commands` | List all available commands |
| `get_command` | Get detailed command information |
| `get_sync_manifest` | Get synchronization manifest for tooling |

**Integration Characteristics:**

- Read-only access via MCP
- Runs in trusted environment (local network)
- No authentication for MVP (Phase 1)
- Searches team knowledge for workflow patterns
- Discovers consistent tooling across team

## Data Flow

The following diagrams show data flow for the two primary use cases.

### Pattern Search via MCP

```mermaid
sequenceDiagram
    participant User
    participant CC as Claude Code
    participant MCP as MCP Server
    participant PG as Postgres + PGVector
    participant NEO as Neo4j

    User->>CC: Ask question
    CC->>CC: Determine need for team knowledge

    CC->>MCP: search_patterns(query, limit)

    Note over MCP: Generate embedding
    MCP->>PG: Semantic search (PGVector)
    PG-->>MCP: Top N patterns

    MCP->>NEO: Fetch related patterns
    NEO-->>MCP: Knowledge graph relationships

    MCP-->>CC: {patterns with context}

    CC->>CC: Incorporate team knowledge
    CC-->>User: Answer with team context
```

### Data Loading via Admin API

```mermaid
sequenceDiagram
    participant Admin
    participant REST as Admin REST API
    participant SVC as Pattern Service
    participant PG as Postgres
    participant NEO as Neo4j

    Admin->>REST: POST /v1/api/patterns (JSON)

    REST->>REST: Validate request

    REST->>SVC: Create pattern

    SVC->>SVC: Generate embedding
    SVC->>PG: Store pattern + embedding
    PG-->>SVC: Pattern ID

    SVC->>NEO: Create knowledge graph nodes/relationships
    NEO-->>SVC: Success

    SVC-->>REST: Pattern created
    REST-->>Admin: 201 Created
```

## Component Interactions

### Claude Code to MCP Server

| Aspect            | Detail                                       |
| ----------------- | -------------------------------------------- |
| Protocol          | MCP over Streamable HTTP                     |
| Authentication    | None (MVP, trusted environment)              |
| Request contains  | MCP tool name, parameters                    |
| Response contains | Search results, pattern details, tooling lists |

### Admin Tools to REST API

| Aspect            | Detail                                       |
| ----------------- | -------------------------------------------- |
| Protocol          | REST (HTTP/HTTPS)                            |
| Authentication    | None (MVP), Envoy + OPA (Phase 2)            |
| Request contains  | JSON payloads for CRUD operations            |
| Response contains | Created/updated resources, success/error status |

### Mnemonic to Storage Layer

| Aspect            | Detail                                       |
| ----------------- | -------------------------------------------- |
| Postgres          | Relational data, pattern storage             |
| PGVector          | Semantic search via embeddings               |
| Neo4j             | Knowledge graph relationships                |

## Boundary Definitions

Clear boundaries separate concerns between components.

```mermaid
graph TB
    subgraph "Claude Code Domain"
        CD1[User prompts]
        CD2[Workflow orchestration]
        CD3[Tool execution]
        CD4[File operations]
    end

    subgraph "Mnemonic Domain"
        MD1[Team knowledge graph]
        MD2[Pattern semantic search]
        MD3[Tooling synchronization]
        MD4[Knowledge graph relationships]
    end

    subgraph "Admin Domain"
        AD1[Pattern CRUD]
        AD2[Agent CRUD]
        AD3[Skill CRUD]
        AD4[Command CRUD]
    end

    CD2 -->|"search_patterns"| MD2
    CD2 -->|"list_agents"| MD3
    AD1 --> MD1
    AD2 --> MD3
```

**Boundary Rules:**

- User credentials never leave Claude Code
- Pattern storage and search live only in Mnemonic
- Tooling definitions (agents, skills, commands) managed via admin API
- File operations happen only on the workstation
- User orchestrates workflow; Mnemonic provides knowledge and consistent tooling

**Next:** [Communication Patterns](04-communication-patterns.md)
