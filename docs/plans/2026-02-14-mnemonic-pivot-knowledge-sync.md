# Mnemonic Architectural Pivot: Team Knowledge Graph & Tooling Synchronization

**Date:** 2026-02-14
**Status:** Proposal
**Impact:** Major architectural change - simplifies Mnemonic's scope and value proposition

## Executive Summary

This document proposes a fundamental architectural pivot for Mnemonic, shifting from **agent routing orchestration** to **team knowledge graph and tooling synchronization**. This change simplifies the system, removes the complex routing engine, and focuses on solving real collaboration problems: shared institutional knowledge and synchronized Claude Code configurations across teams.

## Problem Statement

### Original Vision (Agent Routing)

The original Mnemonic architecture (see [ADR-001](../architecture/02-architectural-decisions.md#adr-001-orchestrator-model), [ADR-002](../architecture/02-architectural-decisions.md#adr-002-routing-location)) envisioned deterministic agent routing:

- User sends prompt → Mnemonic routes to appropriate agent
- Complex routing engine with keyword/regex/pattern matchers
- Rule management system for routing decisions
- Separate ACE CLI to orchestrate Claude Code execution

### Issues Discovered

The MVP implementation plan (`docs/plans/mvp-implementation-plan.md`) demonstrates the core problem: every phase was human-routed. The "Agent(s)" column across 28 phases shows manual delegation. Even the routing engine itself was built through 6 phases (9-14) of human-routed work. If routing automation were necessary, the project would have struggled before it existed. It did not.

During design exploration, several fundamental issues emerged:

1. **Workflow vs. Single Agent**: Real problems need workflows (solutions-architect → go-architect → go-engineer), not single agent routing
2. **User is the Orchestrator**: In spec-based development, the user decides which agents to use and when - they don't hand everything off to an automated system
3. **Routing Adds No Value**: The user already knows "I need architecture work" vs. "I need a simple script"
4. **Complexity Without Benefit**: The routing engine is complex (matchers, rules, caching) but solves a problem users don't actually have

### In reality, the user *is* the orchestrator!

Working on the current implementation and MVP plan had me realize that I was the coordinator. I was the one deciding what agent would work on what, or what set of agents would act as a team. Why do I need to give control over to software for that? I don't.

### The Original Problems Are Solved Differently

The original requirements document identified four problems. This pivot does not abandon problems 1 and 4 - it solves them differently:

**Problem 1: Inconsistent routing** → Solved by consistent tooling via sync (everyone has the same agents, skills, quality gates) PLUS shared workflow patterns in the knowledge graph. Running `/recall "how to build a new service"` returns a documented workflow pattern describing the architect → designer → engineer sequence. The knowledge graph replaces deterministic routing with retrievable, transparent guidance.

**Problem 4: Manual orchestration** → Reframed as **intentional orchestration**. The user-as-orchestrator is the intended operating model, not a workaround. Spec-based development inherently requires human judgment about sequencing. This is a feature, not a bug.

For team scale and new members: workflow patterns in the knowledge graph serve as guardrails. A new developer runs `/recall` and gets the team's documented workflow. This is more flexible and transparent than a routing engine - it teaches the workflow rather than hiding it behind a black box. 

### Real Problems to Solve

Through brainstorming, two genuine collaboration problems emerged:

1. **Institutional Knowledge Loss**: Teams lose patterns, decisions, and conventions when they're not systematically captured and retrieved
2. **Configuration Drift**: Team members have different agent definitions, skills, and commands, leading to inconsistent behavior and collaboration friction

## Proposed Solution

### New Vision: Team Memory & Tooling Sync

Mnemonic becomes a **curated team knowledge graph** with **synchronized agentic tooling**, not an agent orchestrator.

**Core Capabilities:**

#### 1. Team Knowledge Graph (Curated RAG)

A carefully curated knowledge system storing:
- **Patterns**: Engineering patterns, best practices, conventions (e.g., "Go error handling patterns")
- **Decisions**: Architectural decisions with rationale (e.g., "Why we chose Postgres over MongoDB")
- **Guidelines**: Team standards and coding conventions (e.g., "API versioning guidelines")
- **Context**: Project-specific knowledge that aids agent effectiveness

**Interaction model:**
```bash
# Store institutional knowledge
/remember "We use structured logging with zerolog for all Go services"

# Retrieve relevant knowledge
/recall "logging patterns for Go"
→ Returns: zerolog patterns, examples, team conventions

# Search patterns semantically
/patterns "error handling"
→ Returns: Related patterns with graph context
```

**Technical foundation** (already built):
- Postgres + PGVector for semantic search
- Neo4j for knowledge graph relationships
- Pattern enrichment pipeline (embeddings, concept extraction)
- REST API for storage and retrieval

#### 2. Agentic Tooling Synchronization

A **single source of truth** for Claude Code configurations across the team:

**What gets synchronized:**
- Agent definitions (system prompts, capabilities, tools)
- Skills (reusable workflows, scripts)
- Commands (shortcuts, automations)

**How it works:**
```bash
# User runs sync skill
/mnemonic-sync

# Behind the scenes:
# 1. Calls GET /v1/api/agents → downloads agent definitions
# 2. Calls GET /v1/api/skills → downloads skills
# 3. Calls GET /v1/api/commands → downloads commands
# 4. Updates local ~/.claude/{agents,skills,commands}/
# 5. Reports: "Synced 12 agents, 8 skills, 5 commands"
```

**Benefits:**
- New team members → run `/mnemonic-sync` → instantly have team setup
- Team updates error handling pattern → everyone syncs → agents have new context
- No more "works on my machine" due to config drift
- Centralized management, distributed execution

### Architecture Changes

**What stays from current implementation:**
- ✅ Pattern repository (Postgres + PGVector)
- ✅ Graph repository (Neo4j)
- ✅ Pattern enrichment pipeline
- ✅ Semantic search capabilities
- ✅ REST API infrastructure
- ✅ Observability (metrics, tracing, logging)
- ✅ Docker Compose local deployment

**What gets removed:**
- ❌ Entire `internal/routing/` package (engine, matchers, rule cache)
- ❌ `routing_rules` table
- ❌ Rule management endpoints
- ❌ Agent routing logic
- ❌ ACE CLI (no longer needed - skills handle integration)

**What gets added:**
```
mnemonic/
├── internal/
│   ├── storage/
│   │   ├── agents/          # Agent definition storage
│   │   ├── skills/          # Skill storage
│   │   └── commands/        # Command storage
│   └── sync/
│       └── sync_service.go  # Tooling sync orchestration
└── api/
    └── endpoints/
        ├── /v1/api/patterns      # ✅ Already planned
        ├── /v1/api/agents        # ✅ Already planned, repurposed
        ├── /v1/api/skills        # 🆕 New endpoint
        ├── /v1/api/commands      # 🆕 New endpoint
        └── /v1/api/search        # 🆕 Semantic search endpoint
```

**Database schema changes:**
```sql
-- Keep existing
agents (id, name, summary, full_definition, ...)
patterns (id, content, embedding, ...)
enrichment_jobs (...)

-- Remove
routing_rules                -- No longer needed
pattern_agent_associations   -- No longer needed

-- NOTE: The line above was OVERRIDDEN (2026-02-15).
-- pattern_agent_associations is KEPT per the Go Architecture Plan.
-- It is needed for agent-scoped pattern filtering in both the
-- Admin API and MCP search tools.

-- Add new
skills (
  id UUID PRIMARY KEY,
  name VARCHAR(255) UNIQUE NOT NULL,
  description TEXT,
  content TEXT NOT NULL,        -- Skill markdown content
  version VARCHAR(50),
  created_at TIMESTAMP,
  updated_at TIMESTAMP
);

commands (
  id UUID PRIMARY KEY,
  name VARCHAR(255) UNIQUE NOT NULL,
  description TEXT,
  content TEXT NOT NULL,        -- Command definition
  version VARCHAR(50),
  created_at TIMESTAMP,
  updated_at TIMESTAMP
);
```

### API Design

#### Pattern Search & Retrieval

```bash
# Semantic search across patterns
GET /v1/api/patterns/search?q=error+handling&limit=10
Response:
{
  "patterns": [
    {
      "id": "uuid",
      "content": "Go error handling pattern...",
      "similarity": 0.92,
      "related_patterns": ["uuid1", "uuid2"]
    }
  ]
}

# Get pattern by ID with graph context
GET /v1/api/patterns/{id}?include_graph=true
Response:
{
  "pattern": {...},
  "related_patterns": [...],
  "concepts": [...],
  "agents": [...]
}

# Store new pattern
POST /v1/api/patterns
{
  "content": "Pattern description...",
  "tags": ["go", "error-handling"],
  "agent_names": ["go-software-engineer"]
}
```

#### Tooling Synchronization

```bash
# Get all agent definitions (for sync)
GET /v1/api/agents
Response:
{
  "agents": [
    {
      "name": "go-software-engineer",
      "summary": "Implements Go code: functions, packages, tests",
      "definition": {
        "system_prompt": "You are an expert Go engineer...",
        "tools": ["Read", "Write", "Edit", "Bash"],
        "temperature": 0.7
      },
      "version": "1.2.0"
    }
  ]
}

# Get all skills
GET /v1/api/skills
Response:
{
  "skills": [
    {
      "name": "update-agents",
      "description": "Sync agent definitions from Mnemonic",
      "content": "# Skill markdown content...",
      "version": "1.0.0"
    }
  ]
}

# Get all commands
GET /v1/api/commands
Response:
{
  "commands": [
    {
      "name": "prime",
      "description": "Load project context",
      "content": "# Command markdown content...",
      "version": "1.0.0"
    }
  ]
}

# Update agent definition (admin operation)
PUT /v1/api/agents/{name}
{
  "summary": "Updated summary...",
  "definition": {...},
  "version": "1.3.0"
}
```

### Integration via Skills

Skills provide the glue between Claude Code and Mnemonic:

**Example: `/mnemonic-sync` skill**
```markdown
---
name: mnemonic-sync
description: Synchronize agents, skills, and commands from Mnemonic
---

You are synchronizing Claude Code configurations from the team's Mnemonic server.

Steps:
1. Call the sync script: `~/.claude/skills/mnemonic-sync/scripts/sync.py`
2. The script will:
   - Fetch agents from GET /v1/api/agents
   - Fetch skills from GET /v1/api/skills
   - Fetch commands from GET /v1/api/commands
   - Update local ~/.claude/ directories
3. Report what was updated to the user

The user should run this periodically to stay in sync with the team.
```

**Example: `/recall` skill**
```markdown
---
name: recall
description: Search Mnemonic for relevant patterns and team knowledge
---

You are searching the team's institutional knowledge for relevant patterns.

Given the user's query, call the search script:
`~/.claude/skills/recall/scripts/search.py "{query}"`

The script calls GET /v1/api/patterns/search and formats results.

Present the patterns to the user with:
- Pattern content
- Relevance score
- Related patterns (if helpful)
```

**Example: `/remember` skill**
```markdown
---
name: remember
description: Store a pattern or decision in Mnemonic for team knowledge
---

You are storing institutional knowledge in Mnemonic.

Given the user's description:
1. Help them formulate a clear pattern description
2. Identify relevant tags (language, domain, type)
3. Identify which agents should reference this pattern
4. Call the storage script: `~/.claude/skills/remember/scripts/store.py`

The script calls POST /v1/api/patterns to store the knowledge.
```

### User Workflows

#### New Team Member Onboarding

```bash
# Day 1: New developer joins team
$ claude

> /mnemonic-sync
✓ Synced 15 agents from Mnemonic
✓ Synced 12 skills from Mnemonic
✓ Synced 8 commands from Mnemonic
✓ You're now using the team's standard configuration

# They immediately have the same setup as everyone else
```

#### Working on a Feature

```bash
# Developer needs to implement error handling
> /recall "Go error handling patterns"

Found 3 relevant patterns:
1. "Structured error wrapping with context" (92% match)
   - Use fmt.Errorf with %w for error chains
   - Add context at each layer
   - Example: [code sample]

2. "Logging errors vs returning errors" (87% match)
   - Log at boundaries (HTTP handlers, main)
   - Return errors in domain logic
   - Avoid double-logging

3. "Custom error types for domain errors" (85% match)
   [...]

# Developer uses patterns while coding
# When they create a new pattern:
> /remember "We use zerolog for all structured logging in Go services"
✓ Stored pattern in Mnemonic
✓ Tagged: go, logging, conventions
✓ Associated with: go-software-engineer
```

#### Spec-Based Development (User as Orchestrator)

```bash
# User works through specification phases
> I need to build a new microservice for user notifications

# User decides: "I need architecture first"
> /task solutions-architect
"Design architecture for notification service..."

# Architect produces docs, user reviews
# User decides: "Now I need Go-specific design"
> /task go-software-architect
"Based on the architecture in docs/arch.md, design the Go implementation..."

# User approves design
# User decides: "Now implement"
> /task go-software-engineer
"Implement the notification service per docs/design.md..."

# Throughout, agents have access to team patterns via /recall
# User stores new patterns via /remember
# No routing engine needed - user orchestrates
```

### Migration Path

**Phase 1: Assessment & Planning**
1. Review current MVP implementation progress (phases 1-14 complete)
2. Identify what to keep vs. remove
3. Design new API endpoints for sync
4. Create implementation plan with phases

**Phase 2: API & Storage (New Capabilities)**
1. Implement skills storage (table, repository, API)
2. Implement commands storage (table, repository, API)
3. Extend agents API for sync use case
4. Add semantic search endpoint

**Phase 3: Skills Development**
1. Create `/mnemonic-sync` skill with sync script
2. Create `/recall` skill with search script
3. Create `/remember` skill with storage script
4. Test integration with Claude Code

**Phase 4: Cleanup (Remove Routing)**
1. Remove routing engine code
2. Drop routing_rules table
3. Update documentation
4. Simplify deployment

**Phase 5: Documentation & Rollout**
1. Update architecture docs to reflect new vision
2. Update ADRs with pivot decision
3. Create user guide for new workflows
4. Team adoption and feedback

### Benefits of This Pivot

**Simplification:**
- Remove entire routing subsystem (matchers, rules, cache)
- Simpler database schema (drop routing_rules)
- Clearer value proposition: "team knowledge + config sync"
- No custom ACE CLI needed

**Solves Real Problems:**
- ✅ Institutional knowledge loss → curated knowledge graph
- ✅ Configuration drift → synchronized tooling
- ✅ Onboarding friction → instant team setup
- ✅ Pattern discovery → semantic search

**Preserves Investment:**
- Existing pattern repository infrastructure works perfectly
- Graph capabilities (Neo4j) add value for knowledge relationships
- Enrichment pipeline enhances search quality
- Observability, deployment, testing infrastructure all reusable

**Better Alignment:**
- User is the orchestrator (spec-based development model)
- Agents are team members, not automated black boxes
- Mnemonic is a tool, not a replacement for human judgment
- Focuses on augmentation, not automation

### Trade-offs

**What we lose:**
- Deterministic routing (but users didn't need it)
- Automated agent selection (but users prefer manual control)
- Complex rule system (but it was complexity without value)

**What we gain:**
- Focused product vision
- Simpler architecture
- Solves real collaboration problems
- Better team workflow support

**Net result:** Better product, simpler implementation, clearer value.

### Open Questions

1. **Agent definition format:** What should the canonical format be for agent definitions stored in Mnemonic? (JSON schema, YAML, custom format?)

2. **Versioning strategy:** How do we handle version conflicts when syncing? (semver, force-latest, user choice?)

3. **Conflict resolution:** What happens if user has local modifications to agents/skills when syncing? (overwrite, merge, prompt?)

4. **Search ranking:** How do we rank pattern search results beyond cosine similarity? (recency, usage, graph centrality?)

5. **Offline mode:** Can skills work without Mnemonic connection? (local cache, degraded mode, hard dependency?)

6. **Pattern lifecycle:** When do patterns become stale? (deprecation markers, archival process, version history?)

### Post-MVP Deferrals

The following concerns are explicitly deferred to post-MVP:

1. **Pattern governance**: Quality gates, review processes, and conflict resolution for patterns stored in the knowledge graph. MVP assumes a single trusted user as curator.

2. **Admin access control**: Write access to agents/skills/commands/patterns will be scoped to a small set of administrators. MVP runs in a trusted single-user environment, consistent with the existing security architecture deferral (see `docs/architecture/06-security-architecture.md`).

3. **Workflow pattern curation**: Distinguishing workflow patterns (e.g., "sequence for building a new service") from engineering patterns (e.g., "use zerolog for logging") and ensuring workflow patterns - which replace routing for team consistency - are treated as a first-class concern with maintenance ownership.

## Next Steps

1. **Review this proposal** with stakeholders
2. **Create detailed design documents** for:
   - Skills/commands storage schema
   - Sync protocol specification
   - Pattern search ranking algorithm
3. **Build implementation plan** with phases and dependencies
4. **Prototype** `/mnemonic-sync` skill to validate approach
5. **Update architectural decisions** with pivot rationale

## References

- [Original ADR-001: Orchestrator Model](../architecture/02-architectural-decisions.md#adr-001-orchestrator-model)
- [Original ADR-002: Routing Location](../architecture/02-architectural-decisions.md#adr-002-routing-location)
- [Pattern Processing](../design/pattern-processing.md) - Core to new vision
- [MVP Implementation Plan](./mvp-implementation-plan.md) - Phases 1-14 complete

---

**Conclusion:** This pivot simplifies Mnemonic's architecture, focuses on genuine collaboration problems, and better aligns with how users actually work with Claude Code. The transition preserves valuable existing work (knowledge graph, enrichment pipeline) while removing unnecessary complexity (routing engine).
