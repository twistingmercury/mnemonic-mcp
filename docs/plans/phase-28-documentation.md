# Phase 28: Documentation

> Part of the [MVP Implementation Plan](mvp-implementation-plan.md)

**Goal:** Update all documentation to reflect the post-pivot architecture.

**Agent(s):** technical-writer

**Dependencies:** Phase 26 (all implementation complete)

---

## Step 1: Create ADR for pivot decision

- Add ADR-008 to `/Users/doublej/dev/mnemonic/docs/architecture/02-architectural-decisions.md`
- Document: context (routing engine complete but strategy shift), decision (pivot to knowledge graph + MCP), consequences
- Agent: `technical-writer`

## Step 2: Create ADR for MCP protocol choice

- Add ADR-009: why MCP over Streamable HTTP, why the official Go SDK, alternatives considered
- Agent: `technical-writer`

## Step 3: Update system architecture

- Modify: `/Users/doublej/dev/mnemonic/docs/architecture/03-system-architecture.md`
- Replace routing architecture diagram with two-listener diagram
- Agent: `technical-writer`

## Step 4: Update data architecture

- Modify: `/Users/doublej/dev/mnemonic/docs/architecture/08-data-architecture.md`
- Add skills and commands schemas, note routing_rules removal, document agents.version addition
- Agent: `technical-writer`

## Step 5: Archive routing engine design

- Move (do not delete): `/Users/doublej/dev/mnemonic/docs/design/routing-engine.md` to `/Users/doublej/dev/mnemonic/docs/archive/routing-engine.md`
- Agent: `technical-writer`

## Step 6: Update CHANGELOG

- Add pivot entry to `/Users/doublej/dev/mnemonic/CHANGELOG.md`
- Agent: `technical-writer`

## Step 7: Commit

```bash
git add docs/ CHANGELOG.md
git commit -m "docs(pivot): ADRs for pivot + MCP, update architecture docs"
```
