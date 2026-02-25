# Code Review: Phase 1 Doc Fixes

**Review Date:** 2026-02-24
**Reviewers:** code-reviewer, solutions-architect, go-software-architect
**Phase:** Phase 1 (Doc Fixes — Groups A and D)

## Files Reviewed

### Source Files (Group A — Doc Fixes)

- `docs/design/mcp-server.md` — MCP tool count fix (#1), dangling YAML ref removal
- `docs/architecture/08-mcp-tools.md` — Dangling YAML ref removal (#1)
- `docs/mnemonic-concept.md` — MCP scope correction (#2)
- `docs/design/configuration.md` — Single/dual server note (#5)
- `docs/architecture/02-system-architecture.md` — External migration wording (#10)

### Source Files (Group D — Migration Restructure)

- `src/migrations/postgres/000001_extensions.up.sql` through `000008_create_skill_files.down.sql` (16 files)
- `docs/design/data-storage.md` — Migration section alignment

## Validation Results

| Tool | Result |
| ---- | ------ |
| 3-agent parallel code review | Completed |
| Cross-reference check | 3 issues found (all in Phase 2 scope) |
| SQL consistency check | 3 fixes applied during review |

## Design Compliance

Phase 1 fixes resolve issues #1, #2, #3, #5, #10 from `_reviews/docs-contradictions-and-gaps.md`.

### Behavioral Requirements Verified

- MCP tool count consistently says "3 tools" across `mcp-server.md` and `08-mcp-tools.md`
- Concept doc MCP scope limited to pattern search only
- Dangling `mnemonic-mcp-tools-v1.yaml` references removed
- Configuration doc notes MVP-1 single deployable with dual-server config retained
- Migration management described as external golang-migrate CLI
- Migration files use flat directory, 6-digit numbering, golang-migrate convention
- Routing rules migration deleted (feature removed)
- Agent/skill tables use JSONB definition pattern with UUID PK

### Design Doc Divergences (Post-Review)

#### SQL Fixes Applied During Review (3 corrections)

| Migration | Change | Reason |
| --------- | ------ | ------ |
| `000004...up.sql` | Removed redundant `idx_pattern_agent_assoc_pattern` | PK composite index already covers pattern_id leading-column lookups |
| `000006...up.sql` | Added IVFFlat empty-table warning comment | IVFFlat requires at least one row; operator must be aware |
| `000003...up.sql` + `data-storage.md` | Changed pattern name from `varchar(128)` to `varchar(255)` | Consistency with agents/skills name length |

## Findings

### HIGH Priority

| ID | Source | Finding | Resolution |
| -- | ------ | ------- | ---------- |
| H1 | code-reviewer | `02-system-architecture.md` diagrams still show MCP→TOOLING connections | Phase 2 scope (Group B #11) |
| H2 | code-reviewer | `02-system-architecture.md` component text says MCP serves tooling | Phase 2 scope (Group B #11) |
| H3 | go-architect | Redundant index on PK leading column in migration 004 | Fixed during review |
| H4 | go-architect | Code-level: `config.go` has flat ServerConfig, stale RoutingConfig | Out of scope (code, not docs) |
| H5 | go-architect | Code-level: EnrichmentConfig missing 3 fields | Out of scope (code, not docs); noted via #18 |
| H6 | go-architect | `data-storage.md` AgentRepository.Delete(name) vs SkillRepository.Delete(id) asymmetry | Acknowledged; intentional (agents historically keyed by name) |

### MEDIUM Priority

| ID | Source | Finding | Resolution |
| -- | ------ | ------- | ---------- |
| M1 | code-reviewer | `00-architectural-decisions.md` ADR-002 still says MCP serves tooling | Phase 2 scope (Group B #7) |
| M2 | code-reviewer | `00-architectural-decisions.md` ADR-001 says tooling sync via MCP | Phase 2 scope (Group B #7) |
| M3 | code-reviewer | `06-deployment-architecture.md` says MCP handles "tooling requests" | Phase 2 scope (Group B #7) |
| M4 | go-architect | Pattern name varchar(128) inconsistent with 255 | Fixed during review |
| M5 | go-architect | Different status vocabulary (enriched vs completed) between patterns and jobs | Intentional; documented in `data-storage.md` |
| M6 | solutions-architect | Auth contradiction in `03-communication-patterns` | Phase 2 scope (Group C #8) |
| M7 | solutions-architect | Commands entity still referenced everywhere | Phase 2 scope (Group B #7) |
| M8 | solutions-architect | `05-database-integration-flow` MCP calls REST | Phase 2 scope (Group B #15) |

### LOW Priority

| ID | Source | Finding | Resolution |
| -- | ------ | ------- | ---------- |
| L1 | code-reviewer | Patterns down migration lacks explicit DROP INDEX | Postgres cascades; cosmetic only |
| L2 | go-architect | IVFFlat empty-table issue | Comment added during review |
| L3 | go-architect | No CHECK on `skill_files.content` length | Intentional; app-layer validation |
| L4 | solutions-architect | ER diagram stale (skill_files shape, name length, crc64 type) | Known; ER diagram update deferred |

## Patterns to Document

Patterns identified that should be added to the patterns and examples for Claude Code's sub agents.

1. **MCP vs REST scope boundary**: "3 MCP tools for pattern search. Tooling sync via REST API." — canonical statement from `mcp-server.md` should be a Mnemonic pattern.
2. **Migration down-file convention**: Explicit DROP INDEX before DROP TABLE for consistency, even though Postgres cascades.

## Notes for Future Phases

**Phase 2** (Groups B and C): All HIGH/MEDIUM cross-reference issues (MCP→tooling in diagrams, command removal, auth fix) are scheduled for resolution. The ER diagram staleness (L4) should be tracked for a future cleanup pass.

**Code cleanup** (not in this plan): `config.go` needs dual-server update, RoutingConfig removal, enrichment fields, and routing rule repository deletion. These are code changes, not doc fixes.

---

Copyright 2025 Mnemonic Contributors
