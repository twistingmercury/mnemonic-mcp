# Deployment Architecture

[Back to Overview](00-overview.md) | [Back to Project README](../../README.md)

## Table of Contents

- [Deployment Overview](#deployment-overview)
- [Deployment Topology](#deployment-topology)
- [Component Deployment](#component-deployment)
  - [ACE CLI](#ace-cli)
  - [Mnemonic](#mnemonic)
- [Infrastructure Requirements](#infrastructure-requirements)
- [Operational Considerations](#operational-considerations)
- [Scaling Considerations](#scaling-considerations)

## Deployment Overview

ACE uses a lightweight deployment model with minimal server-side infrastructure. The heavy lifting (LLM inference, tool execution) happens on user workstations.

```mermaid
graph TB
    subgraph "User Workstations"
        WS1[Workstation 1<br/>ACE CLI + Claude Code]
        WS2[Workstation 2<br/>ACE CLI + Claude Code]
        WS3[Workstation N<br/>ACE CLI + Claude Code]
    end

    subgraph "Server Infrastructure"
        MN[Mnemonic]
        PG[(Postgres + PGVector)]
        NEO[(Neo4j)]
    end

    WS1 -->|"REST"| MN
    WS2 -->|"REST"| MN
    WS3 -->|"REST"| MN
    MN --> PG
    MN --> NEO
```

## Deployment Topology

### Logical View

```mermaid
graph TB
    subgraph "Client Tier"
        CLI[ACE CLI instances]
    end

    subgraph "Service Tier"
        MN[Mnemonic<br/>Routing + Patterns]
    end

    subgraph "Data Tier"
        PG[(Postgres + PGVector)]
        NEO[(Neo4j)]
    end

    CLI -->|"REST"| MN
    MN --> PG
    MN --> NEO
```

### Physical View

The physical deployment is intentionally simple.

```mermaid
graph TB
    subgraph "Developer Machines"
        D1[Dev 1]
        D2[Dev 2]
        DN[Dev N]
    end

    subgraph "Server Environment"
        subgraph "Container Runtime"
            MN_C[Mnemonic Container]
        end
        PG[(Postgres + PGVector)]
        NEO[(Neo4j)]
    end

    D1 -->|"REST"| MN_C
    D2 -->|"REST"| MN_C
    DN -->|"REST"| MN_C
    MN_C --> PG
    MN_C --> NEO
```

## Component Deployment

### ACE CLI

**Deployment Location:** User workstations

**Characteristics:**

- Installed per-user or per-machine
- No persistent state (stateless between invocations)
- Requires network access to Mnemonic
- Requires Claude Code installation (Phase 1)

**Distribution:**

Post-MVP: Distribution mechanism to be designed after initial release.

| Aspect        | Detail                                         |
| ------------- | ---------------------------------------------- |
| Method        | Post-MVP: To be designed after initial release |
| Updates       | Post-MVP: To be designed after initial release |
| Configuration | Post-MVP: To be designed after initial release |

### Mnemonic

**Deployment Location:** Server infrastructure

**Characteristics:**

- Stateless service (routing rules and patterns from storage)
- Lightweight (no LLM inference)
- Horizontally scalable (if needed)
- Single point of contact for all CLI instances
- REST API for external communication

**Resource Requirements:**

| Resource | Expectation                                 |
| -------- | ------------------------------------------- |
| CPU      | Low to moderate (routing + pattern queries) |
| Memory   | Moderate (caching, pattern indexing)        |
| Storage  | Via external databases (Postgres, Neo4j)    |
| Network  | Moderate (all CLI traffic)                  |

**Storage Stack:**

- **Postgres** - Relational data (agents, routing rules, metadata)
- **PGVector** - Vector embeddings for semantic search
- **Neo4j** - Knowledge graph for pattern relationships

## Infrastructure Requirements

### Server Infrastructure

The server-side footprint is intentionally minimal.

```mermaid
graph TB
    subgraph "Minimal Deployment"
        SINGLE[Single Container<br/>Mnemonic]
        PG1[(Postgres + PGVector)]
        NEO1[(Neo4j)]
        SINGLE --> PG1
        SINGLE --> NEO1
    end

    subgraph "Scaled Deployment"
        LB[Load Balancer]
        MN1[Mnemonic 1]
        MN2[Mnemonic 2]
        PG2[(Postgres + PGVector)]
        NEO2[(Neo4j)]
    end

    LB --> MN1
    LB --> MN2
    MN1 --> PG2
    MN1 --> NEO2
    MN2 --> PG2
    MN2 --> NEO2
```

**Minimal Deployment:**

- Single Mnemonic container
- External Postgres and Neo4j databases
- Suitable for small teams

**Scaled Deployment:**

- Multiple Mnemonic instances behind load balancer
- Shared database backends
- Suitable for larger teams or high availability requirements

### Client Requirements

| Requirement       | Phase 1         | Phase 2                 |
| ----------------- | --------------- | ----------------------- |
| ACE CLI           | Required        | Required                |
| Claude Code       | Required        | Optional                |
| Anthropic API key | Via Claude Code | Direct                  |
| Network access    | To Mnemonic     | To Mnemonic + Anthropic |

## Operational Considerations

### Monitoring

Key metrics to monitor:

| Component        | Metrics                                            |
| ---------------- | -------------------------------------------------- |
| Mnemonic         | Request rate, latency, error rate, pattern queries |
| Postgres         | Connection count, query latency, storage usage     |
| Neo4j            | Query latency, memory usage, connection count      |
| CLI (aggregated) | Usage patterns, version distribution               |

### Logging

| Component | Log Focus                                      |
| --------- | ---------------------------------------------- |
| Mnemonic  | Routing decisions, pattern queries, errors     |
| CLI       | Post-MVP: Logging configuration to be designed |

### Backup and Recovery

| Component         | Strategy                                   |
| ----------------- | ------------------------------------------ |
| Routing rules     | Post-MVP: Backup procedures to be designed |
| Postgres data     | Post-MVP: Backup procedures to be designed |
| Neo4j data        | Post-MVP: Backup procedures to be designed |
| CLI configuration | Post-MVP: Backup procedures to be designed |

### Updates and Maintenance

```mermaid
graph TB
    subgraph "Update Strategy"
        MN_UP[Mnemonic Updates<br/>Rolling deployment]
        DB_UP[Database Updates<br/>Careful migration]
        CLI_UP[CLI Updates<br/>User-controlled]
    end
```

| Component | Update Approach                              |
| --------- | -------------------------------------------- |
| Mnemonic  | Rolling deployment, backward compatible      |
| Databases | Migration-aware, data preservation           |
| CLI       | User-initiated, version compatibility checks |

### Independent Deployment Pipelines

**CRITICAL PRINCIPLE:** Database migrations and application code are versioned and deployed independently.

```mermaid
graph TB
    subgraph "Code Changes"
        APP_CODE[internal/, cmd/**]
        DB_CODE[migrations/**]
    end

    subgraph "CI/CD Pipelines"
        APP_CI[mnemonic-app-ci.yaml<br/>Build, Test, Deploy Container]
        DB_CI[mnemonic-db-ci.yaml<br/>Validate, Test, Apply Migrations]
    end

    subgraph "Deployments"
        APP_DEPLOY[Application Container<br/>Version: v1.2.3]
        DB_DEPLOY[Database Schema<br/>Version: migration 005]
    end

    APP_CODE -->|triggers| APP_CI
    DB_CODE -->|triggers| DB_CI
    APP_CI --> APP_DEPLOY
    DB_CI --> DB_DEPLOY
```

**Why Separate Pipelines?**

| Scenario | Without Separation | With Separation |
| -------- | ------------------ | --------------- |
| Go logic bug fix | Rebuilds app AND runs migrations | App deploys only |
| Add new index | Rebuilds app container | Migrations run only |
| Add column + code | Single coupled deploy | Migration first, then app |

**Pipeline Triggers:**

| Pipeline | Triggers On | Does NOT Trigger On |
| -------- | ----------- | ------------------- |
| `mnemonic-app-ci.yaml` | `internal/**`, `cmd/**`, `go.mod` | `migrations/**` |
| `mnemonic-db-ci.yaml` | `migrations/**` | `internal/**`, `cmd/**` |

**Version Compatibility:**

- Application version: Git tag (e.g., `v1.2.3`)
- Database version: Highest applied migration (e.g., `005`)
- Compatibility matrix documented in release notes

**Deployment Order for Breaking Changes:**

```
1. Deploy migration (forward-compatible: nullable/default values)
2. Verify migration succeeded in production
3. Deploy application (uses new schema)
4. (Optional) Deploy tightening migration (add NOT NULL, remove old columns)
```

This separation ensures:
- Faster deployments (only deploy what changed)
- Safer rollbacks (can rollback app without touching DB)
- Clear audit trail (which pipeline changed what)

## Scaling Considerations

### Horizontal Scaling

```mermaid
graph LR
    subgraph "Scaling Points"
        MN[Mnemonic<br/>Stateless, easy to scale]
        PG[Postgres<br/>Read replicas]
        NEO[Neo4j<br/>Query scaling]
    end
```

| Component | Scaling Approach                          |
| --------- | ----------------------------------------- |
| Mnemonic  | Add instances behind load balancer        |
| Postgres  | Read replicas, connection pooling         |
| Neo4j     | Post-MVP: Scaling approach to be designed |

### Performance Considerations

- Mnemonic latency should be minimal (routing is fast)
- Pattern queries should be cached where possible
- CLI-side caching can reduce Mnemonic calls
- Claude Code execution is the primary latency source (not ACE)

### Capacity Planning

| Factor         | Consideration                  |
| -------------- | ------------------------------ |
| Team size      | Number of concurrent CLI users |
| Request rate   | Queries per minute to Mnemonic |
| Pattern volume | Total patterns in storage      |
| Pattern size   | Average pattern complexity     |

**Next:** Return to [Architecture Overview](00-overview.md)
