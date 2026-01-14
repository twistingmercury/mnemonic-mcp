# ACE Project Structure

## Overview

ACE consists of two separate repositories:

1. **mnemonic** - Backend server providing routing and pattern retrieval via REST API
2. **ace** - CLI client that orchestrates routing decisions and Claude Code execution

Separate repositories allow independent release cycles - Mnemonic can be updated without rebuilding the CLI.

## Repository Layout

```mermaid
graph TB
    subgraph "mnemonic repository"
        MN_API[REST API]
        MN_ROUTE[Routing Engine]
        MN_PATTERN[Pattern Service]
        MN_STORAGE[Storage Layer]
    end

    subgraph "ace repository"
        ACE_CLI[CLI]
        ACE_CLIENT[Mnemonic Client]
        ACE_EXEC[Execution Engine]
    end

    subgraph "Storage"
        PG[(Postgres)]
        PGV[(PGVector)]
        NEO[(Neo4j)]
    end

    ACE_CLI --> ACE_CLIENT
    ACE_CLIENT -->|"REST"| MN_API
    ACE_CLI --> ACE_EXEC
    MN_API --> MN_ROUTE
    MN_ROUTE --> MN_PATTERN
    MN_PATTERN --> MN_STORAGE
    MN_STORAGE --> PG
    MN_STORAGE --> PGV
    MN_STORAGE --> NEO
```

## mnemonic Repository

The Mnemonic server provides routing and pattern retrieval for ACE. For MVP, Mnemonic serves only ACE (not a general-purpose memory service).

See [Communication Patterns](04-communication-patterns.md#rest-endpoints) for REST endpoint details.

**Storage Stack:**

- **Postgres** - Relational data (agents, routing rules, metadata)
- **PGVector** - Vector embeddings for semantic search
- **Neo4j** - Knowledge graph for pattern relationships

## ace Repository

The ACE CLI orchestrates routing decisions and executes prompts via Claude Code.

**Responsibilities:**

- Connect to Mnemonic via REST
- Get routing decisions and patterns
- Invoke Claude Code (Phase 1) or Anthropic API (Phase 2)
- Handle local tool execution (Phase 2)

## Data Flow

```mermaid
sequenceDiagram
    participant User
    participant CLI as ACE CLI
    participant MN as Mnemonic
    participant CC as Claude Code

    User->>CLI: Submit prompt
    CLI->>MN: GET /ace/route?prompt=...
    MN-->>CLI: {agent, patterns}
    CLI->>CC: Invoke with enriched prompt
    CC-->>CLI: Results
    CLI-->>User: Display output
```

## Benefits of Separate Repositories

| Benefit | Description |
|---------|-------------|
| **Independent releases** | Update Mnemonic without rebuilding CLI |
| **Clear boundaries** | Each repo has focused responsibility |
| **Flexible deployment** | Deploy Mnemonic centrally, distribute CLI independently |
| **Separate CI/CD** | Each repo has its own pipeline |
| **Team autonomy** | Different teams can own different repos |
