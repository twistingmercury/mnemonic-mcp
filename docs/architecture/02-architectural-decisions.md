# Architectural Decisions

[Back to Overview](00-overview.md) | [Back to Project README](../../README.md)

## Table of Contents

- [Decision Record Format](#decision-record-format)
- [ADR-008: Architectural Pivot from Agent Routing to Team Knowledge Graph](#adr-008-architectural-pivot-from-agent-routing-to-team-knowledge-graph) (ACTIVE)
- [ADR-009: MCP Protocol for Claude Code Integration](#adr-009-mcp-protocol-for-claude-code-integration) (ACTIVE)
- [Decision Summary](#decision-summary)

## Decision Record Format

Each architectural decision follows this structure:

- **Context**: The situation and forces at play
- **Decision**: What we decided to do
- **Consequences**: The results of the decision, both positive and negative

## ADR-008: Architectural Pivot from Agent Routing to Team Knowledge Graph

**Date:** 2026-02-14
**Status:** Accepted
**Supersedes:** ADR-001, ADR-002, ADR-003, ADR-005, ADR-006

### Context

The original Mnemonic architecture focused on agent routing: accepting prompts, applying routing rules, and returning which agent to invoke. Development through Phase 13 revealed fundamental issues with this approach:

1. **User is the orchestrator**: Manual orchestration is valuable, not a problem to solve. Users know their workflows best.
2. **Routing is overkill**: The real problems are inconsistent tooling (different agent/skill versions) and knowledge silos.
3. **~6,300 lines of routing code**: Significant complexity for questionable value.
4. **Claude Code already has orchestration**: Skills provide workflow coordination; Mnemonic should support, not replace.

### Decision

**Mnemonic pivots from "agent routing orchestrator" to "team knowledge graph + tooling synchronization service."**

New core capabilities:

1. **Team knowledge graph**: Curated patterns with semantic search (PGVector) and knowledge relationships (Neo4j)
2. **Tooling synchronization**: Agents, skills, commands synchronized across team members via MCP protocol
3. **User is the orchestrator**: Mnemonic provides memory and consistent tools; user decides workflow

Removed capabilities:

- Agent routing (remove routing_rules table, routing engine code)
- ACE CLI (separate repository no longer needed)

### Consequences

**Positive:**

- Simpler architecture focused on real team pain points
- ~6,300 lines of routing code removed
- MCP protocol provides natural Claude Code integration
- Manual orchestration reframed as intentional, supported by team knowledge
- Single server with two listeners simplifies deployment

**Negative:**

- Existing routing work (Phases 1-13) becomes foundation but not primary feature
- Must educate users: Mnemonic provides knowledge/tools, not routing decisions
- Pattern_agent_associations table kept (supports "which agents use this pattern")

## ADR-009: MCP Protocol for Claude Code Integration

**Date:** 2026-02-15
**Status:** Accepted
**Supersedes:** ADR-003, ADR-004

### Context

The original architecture used REST API for CLI-to-Mnemonic communication, requiring a separate ACE CLI repository. The pivot to knowledge graph + tooling sync enables direct Claude Code integration.

Options considered:

1. **Continue REST + separate CLI**: Build ACE CLI consuming REST endpoints
2. **MCP protocol**: Leverage Claude Code's native Model Context Protocol support
3. **Claude Code skills + shell scripts**: Wrapper scripts calling REST API

Key considerations:

- Seamless Claude Code integration without separate CLI
- Native protocol support vs. custom integration
- Read-only access pattern for Claude Code
- Admin operations still need programmatic interface

### Decision

**Use MCP (Model Context Protocol) over Streamable HTTP for Claude Code integration.**

Architecture:

- **MCP Server** (`:8081`): Read-only access to patterns, agents, skills, commands
- **Admin REST API** (`:8080`): Write operations for data loading
- **Single server**: Two HTTP listeners on one Go server

MCP tools (11 total):

- `search_patterns`: Semantic search over team knowledge graph
- `find_related_patterns`: Find patterns related to a given pattern
- `get_pattern`: Retrieve specific pattern by ID
- `list_agents`: List all available agents
- `list_skills`: List all available skills
- `list_commands`: List all available commands
- `get_agent`: Get detailed agent information
- `get_skill`: Get detailed skill information
- `get_command`: Get detailed command information
- `get_sync_manifest`: Get synchronization manifest for tooling
- `get_skill_files`: Get skill child files (scripts, references, assets)

### Consequences

**Positive:**

- No separate CLI repository needed
- Native Claude Code integration via MCP
- Read-only MCP pattern keeps Claude Code safe
- Admin REST API for data loading via curl/scripts
- Single server simplifies deployment

**Negative:**

- Must implement MCP server in Go (new protocol support)
- Two protocol surfaces to maintain (REST admin + MCP read-only)
- MCP is less mature than REST for debugging/tooling

## Decision Summary

| Decision | Choice                    | Rationale                                            | Status |
| -------- | ------------------------- | ---------------------------------------------------- | ------ |
| ADR-008  | Pivot to knowledge graph  | User is orchestrator; solve real problems (tooling, knowledge) | ACTIVE |
| ADR-009  | MCP protocol integration  | Native Claude Code integration, dual protocol architecture | ACTIVE |

## Related Design Docs

- [Pivot API Specification](../design/2026-02-15-pivot-api-specification.md) - Current API design
- [Pattern Processing](../design/pattern-processing.md)
- [Configuration](../design/configuration.md)
- [Observability Implementation](../design/observability-implementation.md)

**Next:** [System Architecture](03-system-architecture.md)
