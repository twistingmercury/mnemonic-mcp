# Code Review: Phase 2 Doc Fixes

**Review Date:** 2026-02-24
**Reviewers:** code-reviewer, solutions-architect, go-software-architect
**Phase:** Phase 2 (Doc Fixes — Groups B and C)

## Files Reviewed

### Source Files (Group B — Command Removal + Architecture)

- `docs/mnemonic-concept.md` — Command references removed
- `docs/mnemonic-requirements.md` — TM-3 deleted, command refs removed from TM-4, TS-1, TS-4, AD-1
- `docs/requirements-traceability.md` — TM-3 row removed, related rows updated
- `docs/architecture/00-architectural-decisions.md` — ADR-001/002 MCP wording fixed, commands removed
- `docs/architecture/01-security-architecture.md` — Command refs removed
- `docs/architecture/02-system-architecture.md` — 3-service diagram (#11), MCP to Pattern only, commands removed
- `docs/architecture/03-communication-patterns.md` — Command removal (Group B scope)
- `docs/architecture/05-database-integration-flow.md` — MCP sequence calls service layer (#15)
- `docs/architecture/06-deployment-architecture.md` — Command refs + "tooling requests" removed
- `docs/architecture/07-observability-architecture.md` — Command refs removed
- `docs/design/service-layer.md` — CommandService deleted, diagrams updated (Group B)
- `docs/design/observability-implementation.md` — Command refs removed
- `docs/design/design-changelog.md` — "/commands" removed
- `docs/design/mcp-server.md` — ToolDependencies forward ref + constructor injection (#17)
- `docs/design/configuration.md` — Pending implementation callout (#18)

### Source Files (Group C — OpenAPI + Auth)

- `docs/api/openapi/mnemonic-v1.yaml` — id/crc64 added (#4/#9), commands removed (#7), auth removed (#8), JSONB mapping documented
- `docs/architecture/03-communication-patterns.md` — Auth fix: "No auth in MVP" (#8)
- `docs/design/service-layer.md` — Agent name-to-UUID resolution note (#6)

## Validation Results

| Tool | Result |
| ---- | ------ |
| 3-agent parallel code review | Completed |
| Command removal grep check | PASS — no orphaned Command entity refs |
| OpenAPI structural check | PASS — all refs resolve, no orphaned schemas |
| Cross-edit conflict check | PASS — Groups B and C edits coexist cleanly |

## Design Compliance

Phase 2 fixes resolve issues #4, #6, #7, #8, #9, #11, #15, #17, #18 from `_reviews/docs-contradictions-and-gaps.md`.

### Behavioral Requirements Verified

- Command entity completely removed from all docs (merged into Skills)
- System architecture shows 3 services: Pattern, Agent, Skill
- MCP connects only to Pattern service, not Agent/Skill
- MCP sequence diagram calls SearchService directly, not REST
- ToolDependencies has 3 search methods only; middleware uses constructor injection
- OpenAPI Agent/Skill responses include id (UUID, readOnly) and crc64 (string, readOnly)
- JSONB flat-field mapping documented on Agent and Skill schemas
- AgentAssociation documents name-to-UUID resolution
- MVP has no authentication (bearerApiKey removed, global security removed)
- Pending implementation callout added for 4 config fields

### Fixes Applied During Review (3 corrections)

| File | Change | Reason |
| ---- | ------ | ------ |
| `05-database-integration-flow.md` | `SearchPatterns(query, limit, threshold)` → `SearchPatterns(SearchOptions{...})` | Match actual Go interface signature |
| `service-layer.md` | Comment "Get retrieves" → "GetByName retrieves" | Comment must match method name |
| `mnemonic-v1.yaml` | Removed orphaned `Unauthorized` and `Forbidden` response definitions | MVP has no auth; these contradicted the stated policy |

## Findings

### HIGH Priority

None. All 9 Phase 2 issues resolved.

### MEDIUM Priority

| ID | Source | Finding | Resolution |
| -- | ------ | ------- | ---------- |
| M1 | code-reviewer, go-architect | Orphaned Unauthorized/Forbidden response schemas in OpenAPI | Fixed during review |
| M2 | go-architect | Sequence diagram signature mismatch (positional args vs SearchOptions) | Fixed during review |
| M3 | go-architect | ListOptions duplicated across 3 service packages | Acknowledged; consolidation deferred to implementation |
| M4 | go-architect | SkillService comment/method name mismatch | Fixed during review |
| M5 | go-architect | MCP server wiring for observability deps not shown | Noted; will be addressed during implementation |
| M6 | solutions-architect | ObservabilityConfig struct incomplete in observability-implementation.md vs configuration.md | Phase 3 scope (Group E touches this file) |

### LOW Priority

| ID | Source | Finding | Resolution |
| -- | ------ | ------- | ---------- |
| L1 | code-reviewer | design-changelog ADR-008 reference is dangling | Tracked; not in scope |
| L2 | go-architect | syncNeo4j captures ctx from closure scope instead of parameter | Style preference; functional as-is |
| L3 | go-architect | OpenAPI skill name missing explicit maxLength vs DB VARCHAR(255) | Tracked for implementation |

## Patterns to Document

Patterns identified that should be added to the patterns and examples for Claude Code's sub agents.

1. **Constructor injection for middleware**: Logger/metrics injected at construction, not via service interface. Keeps ToolDependencies focused on business operations.
2. **JSONB flat-field mapping**: Wire format uses flat fields; server marshals to/from JSONB definition column. Documented in OpenAPI schema descriptions.

## Notes for Future Phases

**Phase 3** (Group E): Health path standardization (#13), package path fix (#14), and Neo4j best-effort clarification (#12) in service-layer.md and observability-implementation.md.

**Implementation**: ListOptions consolidation, MCP server wiring example, skill name maxLength validation.

---

Copyright 2025 Mnemonic Contributors
