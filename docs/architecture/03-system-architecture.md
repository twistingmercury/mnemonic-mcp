# How the System Fits Together

**Document:** System Architecture  
**Version:** 1.0  
**Last Updated:** December 22, 2025

Let's talk about how all the pieces work together. This is the "how" to complement the "why" from the ADRs.

## The Big Picture

Here's the full system architecture:

```mermaid
graph TB
    subgraph Clients["Clients"]
        CLI[CLI]
        Web[Web UI]
        IDE[IDE Plugins]
    end

    subgraph K8s["Kubernetes Cluster"]
        LB[Load Balancer]

        subgraph APIPod["API Server Pod"]
            Envoy[Envoy Sidecar<br/>TLS, Auth, Rate Limit]
            OPA[OPA Sidecar<br/>Authorization]
            API[API Server<br/>Business Logic]
        end

        subgraph CogneePod["Cognee Pod"]
            CogneeApp["Cognee MCP Server"]
        end

        Auth[Auth Service]
        Usage[Usage Tracker]
        RateLimit[Rate Limit Service]
    end

    subgraph Data["Data Layer"]
        Redis[(Redis<br/>Cache)]
        RDS[(PostgreSQL<br/>+ pgvector)]
        Neo4j[(Neo4j Aura<br/>Knowledge Graph)]
    end

    Clients -->|HTTPS| LB
    LB --> Envoy
    Envoy -->|ext_authz| Auth
    Envoy -->|rate check| RateLimit
    Envoy --> OPA
    OPA --> API
    API -->|MCP Protocol| CogneeApp
    API -->|async| Usage

    Auth --> Redis
    Auth --> RDS
    RateLimit --> Redis
    Usage --> RDS
    CogneeApp --> Neo4j
    CogneeApp --> RDS

    style Clients fill:#E0E7FF
    style K8s fill:#FEF3C7
    style Data fill:#DBEAFE
```

## Component Breakdown

Let's walk through each piece and what it does.

### API Server Pod

This is where the main application logic lives. It's actually three containers running together:

#### Envoy Sidecar (Infrastructure)

- Terminates TLS from the load balancer
- Checks API keys via external auth service
- Enforces rate limits (calls rate limit service)
- Adds authentication headers for the app
- Emits access logs and metrics
- Handles circuit breaking and retries

#### OPA Sidecar (Policy Engine)

- Evaluates authorization policies (written in Rego)
- Checks if user's plan allows the requested agent
- Verifies feature access
- Injects plan metadata into headers
- Updates policies without code deployment

#### API Server (Application)

- Handles the REST API endpoints
- Routes requests to appropriate agents
- Orchestrates agent execution
- Calls Cognee for pattern queries
- Records usage metrics
- Returns responses to clients

The application code doesn't do auth at all - it trusts the headers Envoy injects. This keeps the app code clean and focused on business logic.

### Cognee Service Pod

Pattern search and knowledge graph queries happen here:

#### Cognee MCP Server

- Official Cognee Docker image (MCP server)
- Provides pattern search via MCP protocol
- Manages knowledge graph queries
- Handles vector embeddings
- Stores patterns in Neo4j

We're using the existing Cognee MCP server directly via its native protocol. This keeps things simple and leverages the standard MCP interface.

### Supporting Services

#### Auth Service

- Validates API keys against PostgreSQL
- Returns user and team context
- Caches frequently accessed keys in Redis
- Issues JWT tokens for web UI
- Handles API key lifecycle

This is called by Envoy's ext_authz filter on every request. It's fast (< 50ms) because of Redis caching.

#### Usage Tracker

- Records agent executions
- Tracks token usage
- Calculates costs
- Emits billing events
- Updates counters

This runs asynchronously - the API doesn't wait for it. If it's down, we queue events and process them later.

#### Rate Limit Service

- Implements token bucket algorithm
- Tracks limits per user, team, and globally
- Uses Redis for shared state
- Returns allow/deny + remaining quota
- Configurable limits per plan

## Data Flow: Agent Execution

Here's what happens when a user executes an agent:

```mermaid
sequenceDiagram
    participant User
    participant Envoy
    participant Auth as Auth Service
    participant OPA
    participant API as API Server
    participant Cognee
    participant Claude as Claude API
    participant Usage

    User->>Envoy: POST /execute agent=go, prompt="..."
    Envoy->>Auth: Check API key
    Auth->>Auth: Validate key (Redis cache)
    Auth-->>Envoy: User context
    Envoy->>OPA: Authorize request + context
    OPA->>OPA: Check plan allows "go" agent
    OPA-->>Envoy: Allowed + headers
    Envoy->>API: Request + auth headers

    API->>API: Route to go-software-agent
    API->>Cognee: Search patterns (MCP)
    Cognee->>Neo4j: Graph query
    Neo4j-->>Cognee: Matching patterns
    Cognee-->>API: Pattern results

    API->>Claude: Execute agent (prompt + patterns)
    Note over Claude: Agent runs, may call<br/>more pattern queries
    Claude-->>API: Response

    API->>Usage: Record execution (async)
    Usage->>RDS: Insert usage record

    API-->>User: Agent response
```

### Key Points

**Authentication happens at the edge** - By the time the request reaches our app, it's already been authenticated. The app trusts the headers.

**Pattern queries are on-demand** - Agent calls `search()` tool, we query Cognee, return just what's needed. This is the core efficiency gain.

**Usage tracking is async** - We don't block the response waiting for usage to be recorded. Fire and forget.

**Multiple pattern queries** - A single agent execution might query patterns 3-5 times. That's fine - each query is cheap and targeted.

## Data Flow: Pattern Updates

When someone updates patterns in git:

```mermaid
sequenceDiagram
    participant Dev as Developer
    participant Git
    participant CI as CI Pipeline
    participant Cognee
    participant Neo4j

    Dev->>Git: Commit pattern changes
    Git->>CI: Webhook triggers build
    CI->>CI: Validate pattern format
    CI->>Cognee: POST /patterns/batch
    Cognee->>Cognee: Generate embeddings
    Cognee->>Neo4j: Update knowledge graph
    Neo4j-->>Cognee: Success
    Cognee-->>CI: Patterns updated
    CI->>Dev: Notify (Slack)
```

Patterns are version controlled in git. When you commit, CI validates and loads them into Cognee. Next query gets the updated patterns. Simple.

## Scaling Characteristics

Different components scale differently:

### API Server (CPU-bound)

- Horizontal scaling: 3-10 replicas
- Autoscale on: CPU > 70%, request rate > 100/s
- Resource profile: 1-2 CPU cores, 2-4GB RAM
- Bottleneck: Request processing, JSON serialization

### Cognee Service (Memory-bound)

- Horizontal scaling: 2-5 replicas
- Autoscale on: Memory > 75%, query latency > 1s
- Resource profile: 1-2 CPU cores, 4-8GB RAM
- Bottleneck: Embedding storage, graph queries

### Databases (Managed)

- Vertical scaling: Upsize instance type
- Horizontal scaling: Read replicas for PostgreSQL
- Autoscale: Managed service handles this
- Bottleneck: Connections, storage IOPS

## Failure Modes

What breaks and how do we handle it?

### Component Failures

#### API Server Pod Dies

- Impact: That pod stops serving traffic
- Detection: Kubernetes health checks fail
- Recovery: K8s restarts pod, traffic routes to healthy pods
- User impact: None (other pods handle requests)
- Time to recover: 30 seconds

#### Cognee Service Dies

- Impact: Pattern queries fail
- Detection: gRPC health check fails
- Recovery: K8s restarts pod
- User impact: Agent executions fail with pattern query error
- Time to recover: 60 seconds
- Mitigation: Cache patterns temporarily, fall back to cached

#### Auth Service Dies

- Impact: New requests can't authenticate
- Detection: ext_authz fails
- Recovery: K8s restarts pod
- User impact: 503 errors on new requests
- Time to recover: 30 seconds
- Mitigation: Short-lived cache in Envoy

#### Redis Dies

- Impact: No caching, rate limiting breaks
- Detection: Connection failures
- Recovery: ElastiCache automatic failover
- User impact: Slower auth, degraded rate limiting
- Time to recover: < 60 seconds

#### PostgreSQL Dies

- Impact: Can't read user data, can't record usage
- Detection: Connection failures
- Recovery: RDS automatic failover to standby
- User impact: 503 errors during failover
- Time to recover: < 5 minutes (RDS SLA)
- Mitigation: Read from replica if available

#### Neo4j Dies

- Impact: Pattern queries fail
- Detection: Connection errors
- Recovery: Neo4j Aura handles failover
- User impact: Agent executions fail
- Time to recover: Aura SLA dependent
- Mitigation: Cache patterns with longer TTL

### Cascading Failure Prevention

We prevent small failures from cascading:

#### Circuit Breakers

- If Cognee has 50% error rate → Open circuit
- Fast-fail requests instead of waiting
- Periodically test if service recovered
- Close circuit when healthy

#### Timeouts

- Every external call has a timeout
- API → Cognee: 30 seconds
- API → Claude: 120 seconds
- Auth check: 500ms
- Fail fast, don't wait forever

#### Rate Limiting

- Prevents thundering herd
- Protects backend services
- Graceful degradation under load

#### Bulkheads

- Separate connection pools per service
- Cognee failures don't exhaust API connections
- Resource isolation

## Security Boundaries

Five trust zones with different security requirements:

### Zone 1: Internet (Untrusted)

- Anyone can send requests
- No implicit trust
- All traffic encrypted (TLS)

### Zone 2: Load Balancer (DMZ)

- Terminates external TLS
- DDoS protection
- Certificate validation

### Zone 3: Application Pods (Authenticated)

- Only authenticated requests
- Envoy enforces auth
- mTLS between pods

### Zone 4: Supporting Services (Internal)

- Service-to-service only
- mTLS required
- No external access

### Zone 5: Data Layer (Encrypted)

- TLS for all connections
- Encryption at rest
- Managed service security

## That's the Architecture

Key takeaways:

- **Sidecars handle infrastructure** - Envoy and OPA keep app code clean
- **Clear service boundaries** - Each service has a specific job
- **Independent scaling** - API and Cognee scale separately
- **Resilient by design** - Circuit breakers, timeouts, automatic failover
- **Zero-trust security** - Nothing is implicitly trusted

Next doc covers why we chose REST external and gRPC internal.

---

Copyright © 2025 Jeremy K. Johnson. All rights reserved.
