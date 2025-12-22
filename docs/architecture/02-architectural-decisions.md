# Major Architectural Decisions

**Document:** Architectural Decision Records (ADRs)  
**Version:** 1.0  
**Last Updated:** December 22, 2025

This doc captures the big architectural decisions we made, why we made them, what we considered, and what we're accepting as trade-offs.

## ADR-001: API-First Architecture

**Status:** Accepted  
**Date:** December 2024

### Context

We could build this as just a CLI tool, as an API with clients, or as some hybrid approach. Each has different implications for how users interact with the system and how we evolve it.

### Decision

We're building an API server with the CLI as a client, not a standalone CLI binary.

**The architecture:**

```text
User → CLI Client → REST API → Backend Services
```

### Rationale

**Why API-first wins:**

1. **Multiple client types** - We can add a web UI, IDE plugins, or integrations without changing the backend. CLI is just the first client.

2. **Team collaboration** - Centralized execution means shared pattern library, usage tracking, and consistent behavior across the team. Everyone sees the same thing.

3. **Better observability** - Server-side execution gives us metrics, logs, and traces. We can see what's actually happening in production.

4. **Portfolio value** - Demonstrates production architecture skills. Shows thinking about scale, operations, and team dynamics. Makes for better blog content.

5. **Future monetization** - Usage tracking and billing are built-in from day one. Can't do that with a standalone CLI.

**What we're giving up:**

- More infrastructure to manage (API server, databases, deployment)
- Network dependency (can't work offline)
- Longer time to MVP (8 weeks instead of 2)
- Operational overhead

### Alternatives Considered

**Option A: CLI-Only**  
Just ship a binary, no server infrastructure.

Pros: Simple, fast to build, works offline  
Cons: No team coordination, no centralized tracking, hard to add web UI later

**Option C: Hybrid (CLI + Optional Server)**  
CLI works standalone OR connects to server.

Pros: Flexibility, gradual migration  
Cons: Two codepaths to maintain, confusing UX, testing nightmare

### Consequences

**Positive:**

- Clean separation between client and backend
- Easy to add new clients
- Production-grade architecture from day one
- Natural scaling path

**Negative:**

- Need to deploy and operate infrastructure
- Local development more complex
- Network latency in critical path

**Neutral:**

- Demonstrates real-world architecture patterns
- More moving parts but clearer boundaries

---

## ADR-002: REST External, gRPC Internal

**Status:** Accepted  
**Date:** December 2024

### Context

We need to decide on communication protocols. Do we use the same protocol everywhere, or different protocols for different use cases?

### Decision

REST/HTTP for external API, gRPC for internal service-to-service communication.

**The split:**

```text
External:  Client → REST API → Business Logic
Internal:  Service A → gRPC → Service B
```

### Rationale

**Why REST externally:**

1. **Developer experience** - Everyone knows REST. Curl works. Postman works. Browser works. Zero friction for API users.

2. **Debugging** - Easy to inspect requests. Browser dev tools work. No special tooling needed.

3. **Documentation** - OpenAPI spec gives us automatic documentation. Interactive API explorer out of the box.

**Why gRPC internally:**

1. **Performance** - Binary protocol is 2-5x faster than JSON. Matters when you're making lots of internal calls.

2. **Type safety** - Protobuf definitions give us compile-time type checking. Catch bugs before runtime.

3. **Streaming** - Built-in support for streaming responses. Useful for long-running agent executions.

4. **Better for high-frequency calls** - API → Cognee might happen 5-10 times per request. gRPC shines here.

### Alternatives Considered

**Option A: REST Everywhere**  
All communication via HTTP/JSON.

Pros: Simple, single protocol, familiar  
Cons: Performance overhead for internal calls, larger payloads, no streaming

**Option B: gRPC Everywhere**  
All communication via gRPC.

Pros: Consistent, fast, type safe  
Cons: Terrible external developer experience, browser limitations, harder debugging

**Option D: GraphQL**  
GraphQL for external API.

Pros: Flexible queries, strong typing  
Cons: Overkill for our simple API, caching complexity, N+1 query problems

### Consequences

**Positive:**

- Right protocol for each use case
- External API is approachable
- Internal services are fast
- Can support streaming when needed

**Negative:**

- Two protocols to maintain
- Translation layer needed at boundaries
- Team needs to know both protocols

**Neutral:**

- Industry-standard pattern (Google, Netflix, etc. do this)
- More complex but justified by benefits

---

## ADR-003: Cognee as Separate Service

**Status:** Accepted  
**Date:** December 2024

### Context

Cognee handles our pattern search and knowledge graph queries. Should we embed it in the API server, run it as a separate service, or use an external SaaS?

### Decision

Deploy Cognee as a separate service in its own pod.

**The architecture:**

```text
API Server Pod:
  - Envoy Sidecar
  - OPA Sidecar
  - API Server Container

Cognee Service Pod:
  - Cognee MCP Server Container
```

### Rationale

**Why separate service:**

1. **Independent lifecycle** - Update Cognee without restarting the API server. Deploy new Cognee versions independently.

2. **Resource isolation** - Cognee is memory-intensive (4-8GB). API server is CPU-bound (2-4GB). They have completely different resource profiles.

3. **Independent scaling** - Scale API (3-10 pods) and Cognee (2-5 pods) based on their actual workload. API scales with request volume, Cognee scales with pattern query load.

4. **Use existing MCP server** - Cognee ships as an MCP server Docker image. We communicate via standard MCP protocol.

5. **Clear service boundaries** - Each service has a single responsibility. Makes troubleshooting and monitoring easier.

### Alternatives Considered

**Option A: Embedded in API:**  
Import Cognee as a library.

Pros: Simple deployment, no network hop  
Cons: Coupled scaling, resource contention, harder to update

**Option B: Same Pod, Different Containers:**  
API and Cognee containers in one pod.

Pros: Localhost communication, simpler networking  
Cons: Can't scale independently, different resource profiles clash, pod becomes large

**Option D: External SaaS (Cognee Cloud):**  
Use hosted Cognee.

Pros: Zero management overhead  
Cons: Data sovereignty issues, network latency, vendor lock-in, cost at scale

### Consequences

**Positive:**

- Clean separation of concerns
- Can use official Cognee MCP server
- Independent scaling from day one
- Independent updates and deployments
- Standard MCP protocol communication
- Clear resource allocation per service

**Negative:**

- Service discovery needed (Kubernetes DNS)
- Network latency between services (minimal, within cluster)
- Slightly more complex deployment manifests

**Neutral:**

- Standard Kubernetes pattern (separate services = separate pods)
- Production-ready architecture from the start

---

## ADR-004: Managed Databases

**Status:** Accepted  
**Date:** December 2024

### Context

We need PostgreSQL, Neo4j, and Redis. Should we self-host them in Kubernetes or use managed services?

### Decision

Use managed database services: RDS PostgreSQL, Neo4j Aura, ElastiCache Redis.

### Rationale

**Why managed services:**

1. **No ops overhead** - No backups to manage. No replicas to configure. No failover to worry about. Just use it.

2. **Built-in HA** - Multi-AZ by default. Automatic failover. We get 99.95% SLA without lifting a finger.

3. **Professional support** - Database breaks at 2 AM? That's their problem, not ours.

4. **Team focuses on application** - We're building an agent orchestrator, not a database company. Use our time on value-add work.

5. **Cost-effective at our scale** - Until we hit 100K+ users, managed services are cheaper than hiring database experts.

**What we're giving up:**

- Some vendor lock-in (mitigated by using standard protocols)
- Less control over tuning
- Higher cost at very large scale

### Alternatives Considered

**Option A: Self-Hosted in Kubernetes**  
StatefulSets for PostgreSQL, Neo4j, Redis.

Pros: Full control, cost savings at very large scale  
Cons: Operations overhead, need database expertise, higher risk of data loss

**Option C: Hybrid**  
PostgreSQL managed, Neo4j self-hosted.

Pros: Balanced approach  
Cons: Inconsistent operations, still need some database expertise

### Consequences

**Positive:**

- Reliable from day one
- Automatic backups and disaster recovery
- Can start small and grow
- Professional-grade infrastructure

**Negative:**

- Higher cost than self-hosting (at scale)
- Some vendor-specific features

**Neutral:**

- Standard protocols minimize lock-in
- Can migrate to self-hosted later if economics change

---

## ADR-005: Envoy + OPA for Auth/Authz

**Status:** Accepted  
**Date:** December 2024

### Context

Every API request needs authentication and authorization. Where should that logic live?

### Decision

Handle authentication and authorization at the infrastructure layer using Envoy sidecars and OPA policy engines.

**The flow:**

```text
Request → Envoy → ext_authz (Auth Service) → OPA (Policy) → Application
```

### Rationale

**Why infrastructure layer:**

1. **Zero auth code in application** - Business logic stays pure. No auth concerns bleeding into the app.

2. **Language agnostic** - Same pattern works for Go, Python, Node, whatever. Makes it easy to add services.

3. **Centralized policy** - Update authorization rules without deploying code. Policy as configuration.

4. **Easier security audits** - All auth logic in one place. Clear boundaries.

5. **Service mesh compatible** - When we adopt Istio, this pattern just works.

**How it works:**

Envoy intercepts every request and calls an external auth service. That service validates the API key and returns user context. Then OPA evaluates policies to decide if the request is allowed. Only then does the request reach our application.

The application trusts the headers Envoy injects. It doesn't do any auth itself.

### Alternatives Considered

**Option A: Application Middleware**  
Auth code in the application.

Pros: Simple, full control, familiar pattern  
Cons: Auth code in every service, security bugs in application layer, language-specific

**Option B: API Gateway Only**  
Single gateway handles all auth.

Pros: Centralized, simple  
Cons: Single point of failure, no service-to-service auth, not zero-trust

### Consequences

**Positive:**

- Application code is simpler and safer
- Uniform auth across all services
- Update auth logic without code deployment
- Built-in audit trail

**Negative:**

- More infrastructure to manage
- Learning curve (Envoy, OPA)
- Debugging across boundaries

**Neutral:**

- Production-standard pattern
- Demonstrates modern security practices

---

## ADR-006: Dynamic Pattern Querying

**Status:** Accepted  
**Date:** December 2024

### Context

Agents need access to patterns (examples, templates, guidelines). Should we pre-load all patterns into context or query them dynamically?

### Decision

Query patterns dynamically using Claude's tool calling feature.

**The approach:**

```text
Agent needs pattern → Calls search() tool → Query Cognee → Return relevant patterns
```

### Rationale

**Why dynamic querying:**

1. **This IS the project** - Solving context bloat is literally the point. Pre-loading defeats the purpose.

2. **78% cost reduction** - $0.13 per execution vs $0.90. That's massive at scale.

3. **Scales forever** - Pattern library can grow to 10,000+ patterns without impacting context size.

4. **Always fresh** - Update a pattern, next query gets the new version. No cache invalidation needed.

5. **Better context utilization** - Use context for actual work instead of patterns.

**The numbers:**

- Pre-loading: ~758KB context, $0.90 per execution
- Dynamic: ~75KB context, $0.13 per execution
- Savings: 78% reduction in cost

### Alternatives Considered

**Option A: Pre-load Everything**  
Load all patterns into context.

Pros: Simple, single API call, all patterns available  
Cons: Massive context (defeats project purpose), expensive, doesn't scale

**Option C: Pattern Embeddings**  
Include embeddings, let Claude query semantically.

Pros: Semantic access  
Cons: Claude doesn't process embeddings directly, still loads everything

**Option D: Pre-load Summaries, Expand on Demand**  
Two-phase approach.

Pros: Refinement possible  
Cons: Still significant overhead, complexity without benefit

### Consequences

**Positive:**

- Achieves core project goal
- Massive cost savings
- Unlimited pattern library growth
- Demonstrates advanced Claude usage

**Negative:**

- Multiple API calls add latency
- Tool calling complexity
- Requires Cognee service

**Neutral:**

- Latency is acceptable (pattern query < 500ms)
- Complexity is justified by savings

---

## ADR-007: Phased Development

**Status:** Accepted  
**Date:** December 2024

### Context

How should we structure the implementation to enable parallel work and incremental delivery?

### Decision

Four-phase approach with clear contracts defined upfront.

**Phase 1 (Weeks 1-2):** Foundation  
Define all contracts: protobuf schemas, API specs, database schemas.

**Phase 2 (Weeks 3-4):** Core Services  
Implement API server, Cognee wrapper, CLI in parallel.

**Phase 3 (Weeks 5-6):** Infrastructure  
Add Envoy sidecars, OPA policies, observability.

**Phase 4 (Weeks 7-8):** Integration  
Wire everything together, end-to-end testing.

### Rationale

**Why phased:**

1. **Parallel development** - Different teams can work independently once contracts are defined.

2. **Incremental value** - Each phase delivers something usable.

3. **Reduced integration risk** - Clear contracts prevent "big bang" integration nightmares.

4. **Flexible timeline** - Can pause after any phase if needed.

**Contract-first approach:**

Spending 2 weeks on contracts might seem slow, but it unlocks 3 teams working in parallel for the next 4 weeks. That's 12 team-weeks of work in 4 calendar weeks. Math works out.

### Consequences

**Positive:**

- Enables parallel work
- Clear milestones
- Lower integration risk
- Incremental delivery

**Negative:**

- Requires upfront design
- Contract changes are expensive
- More coordination needed

**Neutral:**

- Standard for distributed systems
- Well-understood pattern

---

## Key Principles Applied

Looking across all these decisions, some patterns emerge:

### 1. Separation of Concerns

Infrastructure handles infrastructure things. Application handles business logic. Never the twain shall meet.

Examples:

- Envoy/OPA for auth, not application middleware
- Managed databases, not StatefulSets
- MCP protocol for Cognee, not custom wrapper

### 2. Right Tool for the Job

Use the best tool for each specific use case, even if it means more tools overall.

Examples:

- REST for external (developer experience)
- gRPC for internal (performance)
- Managed DBs at this scale (operations)

### 3. Start Simple, Evolve Complexity

Begin with the simplest thing that could work. Add complexity when justified by data.

Examples:

- Cognee in same pod, separate later
- Manual Envoy configs, Istio later
- Single region, multi-region at scale

### 4. Production Patterns from Day One

Build like it's going to production, because it is.

Examples:

- API-first architecture
- Zero-trust security
- Observability built-in
- Managed infrastructure

### 5. Portfolio Value Matters

Architecture decisions demonstrate skills and thinking.

Examples:

- Service mesh patterns
- Policy as code
- Contract-first development
- Clear trade-off analysis

---

That's the major architectural decisions and why we made them. Next doc covers how these decisions shape the system architecture.

---

Copyright © 2025 Jeremy K. Johnson. All rights reserved.
