# Future Work

[Back to MVP Scope](./mvp-scope.md) | [Back to MVP Implementation Plan](./mvp-implementation-plan.md)

This document tracks work items identified but deferred beyond the current MVP scope. These are potential enhancements, operational improvements, and features that have emerged during design and implementation but are not required for initial delivery.

## Table of Contents

- [Operational Concerns](#operational-concerns)
- [Skills and Orchestration Workflows](#skills-and-orchestration-workflows)

## Operational Concerns

### Admin Endpoints for Graph Maintenance

The API specification needs admin endpoints for graph maintenance. A key insight from the pattern-update cascade analysis is that **re-sync and re-enrich are distinct operations** because derived data lives in different places:

| Data                                 | Location                                  | Recovery method            |
| ------------------------------------ | ----------------------------------------- | -------------------------- |
| Pattern metadata (name, description) | Postgres + Neo4j                          | Re-sync from Postgres      |
| Concepts                             | Neo4j only (derived by enrichment)        | Re-enrich (rerun pipeline) |
| Relevance scores                     | Neo4j only (derived by enrichment)        | Re-enrich (rerun pipeline) |
| Embeddings                           | Postgres pgvector (derived by enrichment) | Re-enrich (rerun pipeline) |

This means granular concept/relevance sync endpoints are **internal service endpoints** (used by the enrichment worker), not admin-facing. An admin doing manual repair needs re-enrichment, not raw data push.

#### Recommended admin endpoints

Admin operations live under a distinct `/v1/api/admin/` path, cleanly separated from the primary API surface. This makes authorization rules simpler (middleware can enforce admin role on the entire `/v1/api/admin/` prefix) and prevents accidental exposure of maintenance operations.

| Method | Path                                              | Purpose                                                     |
| ------ | ------------------------------------------------- | ----------------------------------------------------------- |
| POST   | `/v1/api/admin/graph/sync/agents/{name}`          | Re-sync agent node from Postgres                            |
| POST   | `/v1/api/admin/graph/sync/patterns/{id}`          | Re-sync pattern metadata from Postgres                      |
| POST   | `/v1/api/admin/enrichment/patterns/{id}`          | Re-enrich pattern (new enrichment job, full pipeline rerun) |
| POST   | `/v1/api/admin/graph/cleanup/orphaned-concepts`   | Remove orphaned concept nodes                               |
| GET    | `/v1/api/admin/graph/health`                      | Graph health with node/relationship statistics              |
| GET    | `/v1/api/admin/graph/consistency`                 | Drift detection (Postgres vs Neo4j)                         |
| GET    | `/v1/api/admin/graph/patterns/{id}/related`       | Query related patterns via shared concepts                  |
| GET    | `/v1/api/admin/graph/agents/{name}/patterns`      | Query patterns relevant to an agent                         |

All require `admin` role. A detailed API design was produced by the api-architect-agent and needs review before incorporation into `api/openapi/mnemonic-v1.yaml`.

#### Design consideration: separate maintenance service

These admin operations could live in a **separate maintenance service** rather than being added to the primary Mnemonic API. Rationale:

- Keeps Mnemonic focused on its core responsibilities (routing, pattern retrieval)
- Different scaling profile — maintenance operations are infrequent, bursty, and potentially long-running
- Different access model — admin-only, possibly restricted to internal network or VPN
- Different deployment lifecycle — can be updated independently without redeploying Mnemonic
- Shares the same Postgres and Neo4j databases but has its own process and API surface

This would mean the `/v1/api/admin/` path above becomes its own service (e.g., `mnemonic-admin` or `mnemonic-maintenance`) rather than routes within the Mnemonic process.

#### Open questions

- Should there be a bulk re-enrich endpoint (e.g., re-enrich all patterns for a given agent)?
- Is a circuit-breaker or rate limit needed on re-enrich to prevent overloading the enrichment worker?
- If a separate service, should it share the same Go module or be a separate module with shared library dependencies?

**Context:** Emerged from Phase 8B code review and the pattern-update cascade analysis.

[Back to Table of Contents](#table-of-contents)

## Skills and Orchestration Workflows

### Incorporating Skill Definitions into Mnemonic

Skills are specialized orchestration workflows that coordinate multiple agents to accomplish complex tasks. Currently, skill definitions live in `agents/skills/`, are installed by the `02-install-skills.sh` script, and use the `project_skill` frontmatter marker to distinguish them from standard agent definitions.

A critical architectural constraint shapes this integration: skills are inherently a **client-side concept** — they execute locally within each developer's Claude Code installation (`~/.claude/skills/`). Mnemonic, as a server-side service, has no direct path to this local directory. This creates a distribution challenge: if Mnemonic becomes the source of truth for skill definitions, there must be a mechanism to push updated skills to developer workstations.

#### Integration benefits

Incorporating skills into Mnemonic's knowledge graph would provide four key capabilities:

- **Skill routing** — The routing engine could recommend skills (not just individual agents) when a task matches a multi-agent workflow pattern. For example, "write a shell script" would route to the shell-script-orchestration skill, which automatically chains `shell-script-agent` → `bats-test-agent`.
- **Skill storage** — Skill definitions would be stored in Postgres and Neo4j alongside agent definitions and patterns, making them queryable and versionable through the same infrastructure.
- **Skill-agent relationships** — The knowledge graph would model which agents a skill orchestrates, enabling queries like "what skills use this agent?" or "what agents does this skill coordinate?"
- **Skill pattern associations** — When a skill is invoked, Mnemonic could retrieve relevant patterns for all constituent agents in the workflow, ensuring each specialist has the knowledge it needs.

#### Open questions

- Should skills be first-class entities in the data model, or metadata on routing rules?
- Should Mnemonic validate skill definitions (e.g., ensure all referenced agents exist in the graph)?
- What is the sync strategy between source files (`agents/skills/`) and the database? Should skills follow the same update cascade pattern as agents?
- Should skill enrichment extract concepts from skill procedures to improve routing accuracy?
- How should skills be distributed from Mnemonic to developer workstations? Should Mnemonic expose an endpoint that a local agent or cron job polls for skill updates? Should the workbench install script pull from Mnemonic instead of the repository? What does the push/pull synchronization model look like?

**Context:** Emerged from code review and shell script workflow skill extraction.

[Back to Table of Contents](#table-of-contents)
