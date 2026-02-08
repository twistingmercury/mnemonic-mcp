# Future Work

[Back to MVP Scope](./mvp-scope.md) | [Back to MVP Implementation Plan](./mvp-implementation-plan.md)

This document tracks work items identified but deferred beyond the current MVP scope. These are potential enhancements, operational improvements, and features that have emerged during design and implementation but are not required for initial delivery.

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

Admin operations live under a distinct `/api/admin/` path, cleanly separated from the primary API surface. This makes authorization rules simpler (middleware can enforce admin role on the entire `/api/admin/` prefix) and prevents accidental exposure of maintenance operations.

| Method | Path                                          | Purpose                                                     |
| ------ | --------------------------------------------- | ----------------------------------------------------------- |
| POST   | `/api/admin/graph/sync/agents/{name}`         | Re-sync agent node from Postgres                            |
| POST   | `/api/admin/graph/sync/patterns/{id}`         | Re-sync pattern metadata from Postgres                      |
| POST   | `/api/admin/enrichment/patterns/{id}`         | Re-enrich pattern (new enrichment job, full pipeline rerun) |
| POST   | `/api/admin/graph/cleanup/orphaned-concepts`  | Remove orphaned concept nodes                               |
| GET    | `/api/admin/graph/health`                     | Graph health with node/relationship statistics              |
| GET    | `/api/admin/graph/consistency`                | Drift detection (Postgres vs Neo4j)                         |
| GET    | `/api/admin/graph/patterns/{id}/related`      | Query related patterns via shared concepts                  |
| GET    | `/api/admin/graph/agents/{name}/patterns`     | Query patterns relevant to an agent                         |

All require `admin` role. A detailed API design was produced by the api-architect-agent and needs review before incorporation into `api/openapi/mnemonic-v1.yaml`.

#### Design consideration: separate maintenance service

These admin operations could live in a **separate maintenance service** rather than being added to the primary Mnemonic API. Rationale:

- Keeps Mnemonic focused on its core responsibilities (routing, pattern retrieval)
- Different scaling profile — maintenance operations are infrequent, bursty, and potentially long-running
- Different access model — admin-only, possibly restricted to internal network or VPN
- Different deployment lifecycle — can be updated independently without redeploying Mnemonic
- Shares the same Postgres and Neo4j databases but has its own process and API surface

This would mean the `/api/admin/` path above becomes its own service (e.g., `mnemonic-admin` or `mnemonic-maintenance`) rather than routes within the Mnemonic process.

#### Open questions

- Should there be a bulk re-enrich endpoint (e.g., re-enrich all patterns for a given agent)?
- Is a circuit-breaker or rate limit needed on re-enrich to prevent overloading the enrichment worker?
- If a separate service, should it share the same Go module or be a separate module with shared library dependencies?

**Context:** Emerged from Phase 8B code review and the pattern-update cascade analysis.
