# ACE Architecture - The Big Picture

**Project:** ACE (Agentic Coding Engine)  
**Version:** 1.0  
**Last Updated:** December 22, 2025

## What We're Building

ACE is a production-grade system for deterministic agent routing with smart pattern querying. Think of it as a much smarter, more predictable version of Claude Code that actually knows what it's doing.

### The Problem We're Solving

Claude Code is great, but it has a frustrating limitation: it uses LLM interpretation for routing decisions. That means:

- Same prompt, different agent every time (non-deterministic)
- No guaranteed workflow execution
- Unpredictable behavior
- Can't reliably automate anything

We're fixing that with code-based routing (100% deterministic) plus a clever trick: instead of loading all our patterns into context (expensive!), we query them on-demand. That saves us 78% on costs and lets our pattern library grow without blowing up our context window.

## Core Value Proposition

**The whole point of this project:**

- **Deterministic routing** - Same input always routes to the same agent
- **Context efficiency** - Query patterns dynamically instead of pre-loading everything
- **Team collaboration** - Share patterns across your team via git
- **Independent scaling** - Scale different parts based on actual load

## Design Philosophy

We're following some key principles here:

### 1. Infrastructure vs Application Code

The infrastructure layer (Envoy, OPA) handles auth, rate limiting, and security. Our application code focuses purely on orchestration logic. No auth code in the app = cleaner, safer, easier to test.

### 2. REST for External, gRPC for Internal

External API uses REST because developers love it - easy debugging, familiar tools, works everywhere. Internal services use gRPC because performance matters when you're making lots of service-to-service calls.

### 3. Managed Infrastructure

We're using managed databases (RDS, Neo4j Aura) instead of running our own. The team should focus on building features, not babysitting PostgreSQL replicas.

### 4. Independent Scalability

Cognee (pattern search) is memory-heavy. The API server is CPU-bound. They scale differently, so we're deploying them as separate services from the start.

### 5. Service Mesh Patterns

Using Envoy sidecars for infrastructure concerns. We can adopt full Istio later if we need it, but starting simple with manual sidecar configs.

## High-Level Architecture

```mermaid
graph TB
    subgraph Clients["Client Applications"]
        CLI["CLI Client"]
        IDE["IDE Plugins"]
    end

    subgraph K8s["Kubernetes Cluster"]
        subgraph APIPod["API Server Pod"]
            Envoy["Envoy Sidecar<br/>Auth, Rate Limit, TLS"]
            OPA["OPA Sidecar<br/>Authorization"]
            API["API Server<br/>Business Logic"]
        end

        subgraph CogneePod["Cognee Service Pod"]
            CogneeAPI["Cognee MCP Server<br/>(Docker Image)"]
        end

        Redis["Redis<br/>Rate Limits, Cache"]
        AuthSvc["Auth Service<br/>API Key Validation"]
    end

    subgraph ManagedServices["Managed Services"]
        RDS["RDS PostgreSQL<br/>+ pgvector"]
        Neo4j["Neo4j Aura<br/>Knowledge Graph"]
    end

    Clients -->|REST/HTTPS| Envoy
    Envoy --> OPA
    OPA --> API
    Envoy -->|Rate Check| Redis
    Envoy -->|Auth Check| AuthSvc
    API -->|MCP Protocol| CogneeAPI
    CogneeAPI --> RDS
    CogneeAPI --> Neo4j

    style Clients fill:#E0E7FF
    style K8s fill:#FEF3C7
    style ManagedServices fill:#DBEAFE
```

## Key Architectural Decisions

Here's what we decided and why:

| Decision                   | Rationale                                | Where to Read More                                             |
| -------------------------- | ---------------------------------------- | -------------------------------------------------------------- |
| REST for external API      | Developer experience, tooling, debugging | [04-communication-patterns.md](04-communication-patterns.md)   |
| gRPC for internal comms    | Performance, type safety, streaming      | [04-communication-patterns.md](04-communication-patterns.md)   |
| Cognee as separate service | Independent scaling, resource isolation  | [03-system-architecture.md](03-system-architecture.md)         |
| Managed databases          | No ops overhead, built-in HA             | [05-data-architecture.md](05-data-architecture.md)             |
| Envoy + OPA sidecars       | Externalize infrastructure concerns      | [06-security-architecture.md](06-security-architecture.md)     |
| Service mesh patterns      | Uniform observability, zero-trust        | [08-deployment-architecture.md](08-deployment-architecture.md) |

## What This Gets Us

### For Development

**Parallel Work Streams**  
Clear service boundaries mean teams can work independently. Well-defined contracts (protobuf) let you mock dependencies. Nobody's blocked waiting for someone else.

**Better Testing**  
Business logic is testable without spinning up infrastructure. Infrastructure is testable without touching application code. Contract testing at service boundaries keeps everything honest.

### For Operations

**Independent Deployment**  
Update auth policies without redeploying the app. Scale Cognee independently. Run database migrations without blocking releases. Canary deploy per service.

**Great Observability**  
Uniform telemetry across services via OpenTelemetry. Request tracing across boundaries. Centralized logging and metrics. Clear failure domains.

**Security by Default**  
Zero-trust architecture - nothing is implicitly trusted. Centralized auth/authz policies. Defense in depth. Built-in audit trail.

### For Product

**Feature Velocity**  
Add new agents without infrastructure changes. Update patterns without deployment. New rate limit tier? Config change. Fast experimentation.

**Reliability**  
Deterministic routing = predictable behavior. Automatic retries at infrastructure layer. Circuit breakers prevent cascading failures. Graceful degradation when things break.

**Cost Management**  
Dynamic pattern querying reduces token usage by 78%. Independent scaling prevents over-provisioning. Managed services reduce ops overhead. Usage tracking for cost allocation.

## Document Structure

The architecture is documented across 10 focused docs:

1. **[Requirements](01-requirements.md)** - What we're solving and why
2. **[Architectural Decisions](02-architectural-decisions.md)** - Major decisions with justification (ADR format)
3. **[System Architecture](03-system-architecture.md)** - How components fit together
4. **[Communication Patterns](04-communication-patterns.md)** - Why REST external, gRPC internal
5. **[Data Architecture](05-data-architecture.md)** - Database strategy and why managed services
6. **[Security Architecture](06-security-architecture.md)** - Auth, authz, zero-trust approach
7. **[Observability Architecture](07-observability-architecture.md)** - Monitoring, logging, tracing
8. **[Deployment Architecture](08-deployment-architecture.md)** - Kubernetes, service mesh, GitOps
9. **[Scalability](09-scalability.md)** - How we scale and when
10. **[Trade-offs](10-trade-offs.md)** - Alternatives we considered and why we chose what we did

## Reading Guide

**If you're implementing this:**

1. Start with Requirements (01)
2. Read System Architecture (03)
3. Check Communication Patterns (04)
4. Review deployment strategy (08)

**If you're doing security review:**

1. Security Architecture (06)
2. Data Architecture (05)
3. Observability (07)

**If you're a stakeholder wanting the overview:**

1. This document (00)
2. Requirements (01)
3. Trade-offs (10)

**If you're planning operations:**

1. Deployment Architecture (08)
2. Scalability (09)
3. Observability Architecture (07)

## What's Next

After architecture approval, we'll:

1. **Phase 1:** Define contracts (protobuf, API schemas, DB schemas)
2. **Phase 2:** Implement core services (parallel development streams)
3. **Phase 3:** Add infrastructure services (sidecars, observability)
4. **Phase 4:** Integration and production readiness

See [Architectural Decisions](02-architectural-decisions.md) for the phased implementation approach and how teams can work in parallel.

## System Characteristics at a Glance

### Performance Targets

- API response time: < 100ms (excluding agent execution)
- Agent execution: 10-30 seconds (Claude API bound)
- Pattern query: < 500ms
- Throughput: 100+ requests/second per API instance

### Scalability Targets

- API server: 3-10 replicas
- Cognee service: 2-5 replicas
- Support 10-1000 users per deployment
- Handle 1000+ patterns efficiently
- 50+ concurrent executions

### Availability Targets

- Uptime: 99.9% (3 nines)
- Recovery time: < 5 minutes
- Data durability: 99.999999999% (managed DB SLA)

### Cost Efficiency

- Token usage: ~75KB context (vs 758KB pre-loading)
- Cost per execution: ~$0.13 (Sonnet 4)
- Infrastructure: $500-2000/month depending on scale

That's the big picture. Dive into individual docs for the details on specific areas.

---

Copyright © 2025 Jeremy K. Johnson. All rights reserved.
