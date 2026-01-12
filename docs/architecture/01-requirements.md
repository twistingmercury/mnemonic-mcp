# What We're Building and Why

**Document:** Requirements  
**Version:** 1.0  
**Last Updated:** December 22, 2025

## The Problem

Let's talk about what's broken and what we're fixing.

### What's Wrong with Current Agent Systems

**Non-Deterministic Routing**  
Claude Code uses an LLM to interpret delegation rules. That sounds smart until you realize: same prompt, different routing, every single time. You can't build reliable automation on top of that. It's like having a GPS that takes you different routes home every day - technically it works, but you can't predict when you'll arrive.

**Context Bloat**  
To give agents proper guidance, we currently pre-load ALL patterns into context. That's ~758KB of text. Every. Single. Execution. At $0.90 per run, that adds up fast. Plus you're eating up context that could be used for actual work.

**No Team Collaboration**  
Everyone maintains their own pattern library. Someone figures out the perfect way to structure a BATS test? Too bad, that knowledge stays siloed. No sharing, no improvements, no team learning.

**Can't Scale Different Parts Differently**  
Everything's coupled together. Pattern search is memory-intensive, but it scales with the same settings as your request handling which is CPU-bound. You end up over-provisioning everything.

## What We're Building

Here's what ACE fixes:

### 1. Deterministic Routing

**Requirement:** Same prompt must ALWAYS route to the same agent(s).

**What success looks like:**

- 100% consistency across runs
- Fully verifiable through logging
- No LLM interpretation in routing path
- Predictable workflow chaining

**Why this matters:**  
Users need to trust the system. Automated workflows need reliability. Debugging is impossible when behavior changes randomly. Costs need to be predictable.

**How we're doing it:**  
Code-based routing with keyword matching and explicit chaining rules. The LLM doesn't decide where things go - our code does.

### 2. Context Efficiency

**Requirement:** Agents query patterns dynamically instead of pre-loading everything.

**What success looks like:**

- Context usage < 100KB per execution (vs ~758KB with pre-loading)
- 75%+ cost reduction vs pre-loading
- 3-5 pattern queries per execution
- No drop in output quality

**Why this matters:**
This is literally the whole point of the project. Pre-loading patterns is expensive and doesn't scale. We need to query what we need, when we need it.

**How we're doing it:**
Tool calling protocol. Agent calls a `search()` tool, we query Cognee's knowledge graph, return just the relevant patterns. See [ADR-006](02-architectural-decisions.md#adr-006-dynamic-pattern-querying) for the full rationale and cost analysis.

### 3. Team Collaboration

**Requirement:** Multiple developers share the same pattern library with version control.

**What success looks like:**

- Patterns stored in git
- Updates propagate to whole team
- Pattern improvements reviewable via PR
- Consistent access across everyone

**Why this matters:**  
Knowledge compounds. Your team gets smarter together instead of separately. Best practices spread naturally.

**How we're doing it:**  
Patterns in git, loaded into shared Cognee instance. Update a pattern, commit, everyone gets it. Simple.

### 4. Independent Scalability

**Requirement:** Different components scale based on their actual workload.

**What success looks like:**

- API server scales for request load
- Cognee scales for pattern query load
- Database resources allocated separately
- No forced coupling

**Why this matters:**  
API requests and pattern searches have completely different resource profiles. API is CPU-bound and bursty. Cognee is memory-bound and steady. They shouldn't scale together.

**How we're doing it:**  
Separate services with separate scaling configs. Start in the same pod (simple), move apart when needed (flexible).

## What We're NOT Building (Initially)

Let's be clear about scope:

**Not building:**

- Web UI (just API + CLI for MVP)
- Advanced caching strategies (simple caching is fine)
- Multi-region deployment (single region to start)
- Custom agent uploads (use git for now)
- Billing automation (track usage, bill manually)

**Will build later:**

- Web UI (v0.5+)
- Advanced caching (when we see actual patterns)
- Multi-region (when we have 10K+ users)
- Custom uploads (v0.6+)
- Billing automation (v0.7+)

## Success Metrics

How do we know if this works?

### Technical Metrics

- **Routing accuracy:** 100% deterministic (it better be!)
- **Context efficiency:** < 100KB per execution
- **Cost reduction:** > 75% vs pre-loading
- **Availability:** > 99.9% uptime
- **Performance:** < 100ms API response time

### Business Metrics

- **Team adoption:** 10+ teams using shared patterns
- **Pattern growth:** 1000+ patterns in library
- **Execution volume:** 100K+ requests/month
- **Cost per execution:** < $0.15 average

### User Experience Metrics

- **Consistency:** Zero reports of routing variation
- **Reliability:** < 0.1% error rate
- **Speed:** 90% of executions < 20 seconds
- **Satisfaction:** Positive feedback on predictability

## Critical Requirements

These are non-negotiable:

**Must have (P0):**

- Deterministic routing (code-based, not LLM)
- Dynamic pattern querying (tool calling)
- Basic authentication (API keys)
- Agent execution with chaining
- Usage logging
- Health checks

**Should have (P1):**

- Advanced authorization (OPA policies)
- Rate limiting per plan
- Cost tracking and metrics
- Managed database integration
- Kubernetes deployment
- OpenTelemetry observability

**Nice to have (P2):**

- Web UI for management
- Advanced caching strategies
- Multi-region deployment
- Custom agent uploading
- Pattern versioning
- Billing automation

## Constraints We're Working With

### Technical Constraints

**Kubernetes Deployment**  
Everything runs in Kubernetes. We're using K8s native features (ConfigMaps, Secrets, Services). Service mesh compatible but not required initially.

**Managed Databases**  
No self-hosted database management. Using cloud provider managed services (RDS, Neo4j Aura, ElastiCache). Team focuses on application, not database ops.

**Cognee Integration**
Using existing Cognee MCP server Docker images. Communication via standard MCP protocol. No modifications to Cognee itself. Support Cognee API evolution through versioning.

**Protocol Standards**  
REST for external APIs (OpenAPI spec). gRPC for internal services (protobuf). OpenTelemetry for observability. Standard HTTP/2.

### Operational Constraints

**Development Workflow**  
Support local development with Docker Compose. Production environment should closely match local. Enable parallel development across teams.

**Cost Management**  
Optimize for cloud cost efficiency. Pay-as-you-grow model. Right-size resources. Use spot instances where appropriate.

### Business Constraints

**Time to Market**  
Phased rollout. MVP in 8 weeks. Production-ready in 16 weeks. Incremental value delivery. This is for portfolio/job search, not a hard deadline.

**Monetization Support**  
Usage-based billing ready from day one. Multiple plan tiers. Cost tracking per team. Billing integration hooks (even if we don't bill immediately).

**Portfolio Showcase**  
Architecture demonstrates production thinking. Shows cloud-native patterns. Exhibits scalability design. Provides blog-worthy technical depth.

## Open Questions

Things we'll figure out during implementation:

**Cognee multi-tenancy?**  
Should we run one Cognee instance for all teams or separate instances? Impact on resource isolation vs efficiency. We'll decide before production.

**Pattern cache TTL?**  
How long do we cache patterns? Impact on freshness vs performance. We'll tune based on actual usage patterns.

**Custom agent uploads?**  
If we allow custom agents, how do we validate them? Security implications? Probably post-MVP based on demand.

**Multi-region strategy?**  
If we go multi-region, how do we handle data consistency? What's the latency/consistency trade-off? Cross that bridge at 10K+ users.

That's what we're building and why. Next up: how we're building it.

---

Copyright © 2025 Jeremy K. Johnson. All rights reserved.
