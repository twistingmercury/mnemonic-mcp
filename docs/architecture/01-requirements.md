# Mnemonic Requirements

[Back to Overview](00-overview.md) | [Back to Project README](../../README.md)

## Table of Contents

- [Problem Statement](#problem-statement)
- [How Mnemonic Addresses Original Problems](#how-mnemonic-addresses-original-problems)
- [Goals](#goals)
- [Non-Goals](#non-goals)
- [Success Criteria](#success-criteria)
- [Constraints](#constraints)
- [Assumptions](#assumptions)

## Problem Statement

Teams using Claude Code face several challenges when working at scale:

1. **Inconsistent tooling**: Without centralized agent, skill, and command definitions, each team member has different versions and capabilities
2. **Knowledge silos**: Patterns, prompts, and best practices remain isolated on individual workstations
3. **No shared memory**: Teams cannot leverage collective learnings or maintain organizational knowledge
4. **Manual orchestration**: Complex workflows require manual coordination between multiple Claude Code sessions

Mnemonic addresses these challenges by providing a team knowledge graph with semantic search and synchronized tooling across team members.

## How Mnemonic Addresses Original Problems

The February 2026 architectural pivot reframed how Mnemonic solves these problems:

### Problem 1: Inconsistent Tooling (Originally "Inconsistent Routing")

**Original approach**: Centralized routing logic to route prompts to appropriate agents.

**Pivot approach**: Synchronize agent, skill, and command definitions across team members via MCP protocol. Each team member has the same tooling capabilities. The user orchestrates which agent/skill to invoke based on team knowledge patterns.

**Why the change**: User is the orchestrator. Mnemonic provides consistent tools and knowledge, not routing decisions.

### Problem 2 & 3: Knowledge Silos and No Shared Memory

**Approach remains consistent**: Team knowledge graph with semantic search via PGVector and knowledge relationships via Neo4j. Patterns are curated and enriched, searchable by all team members through Claude Code's MCP integration.

### Problem 4: Manual Orchestration

**Original framing**: Manual orchestration is a problem to solve with automated routing.

**Pivot reframing**: Manual orchestration is intentional and valuable. The user knows their workflow best. Mnemonic provides:

1. **Workflow patterns** in the knowledge graph describing common coordination approaches
2. **Consistent tooling** so orchestration steps work the same for everyone
3. **Shared memory** so orchestration decisions benefit from team knowledge

Manual orchestration becomes informed orchestration supported by team knowledge and consistent tools.

## Goals

### Primary Goals

- **Team knowledge graph**: Provide semantic search over curated patterns with PGVector and knowledge relationships via Neo4j
- **Tooling synchronization**: Ensure all team members have consistent agents, skills, and commands via MCP protocol
- **Claude Code integration**: Seamless MCP integration without changing existing Claude Code workflows
- **Informed orchestration**: Support manual orchestration with team knowledge patterns and consistent tooling

### Secondary Goals

- **Gradual adoption**: Teams can start with pattern search and add tooling sync incrementally
- **Minimal infrastructure**: Keep server-side components lightweight and easy to deploy via Docker Compose
- **Production-ready path**: Clear path from MVP to production with Envoy + OPA security layer

## Non-Goals

The following are explicitly out of scope:

- **Automated routing**: Mnemonic does not route prompts to agents; the user orchestrates their workflow
- **Replacing Claude Code**: Mnemonic integrates with Claude Code via MCP; it does not replace functionality
- **Running LLM inference on server**: All LLM interactions happen locally via Claude Code
- **Managing user credentials**: Mnemonic does not store or manage Anthropic API keys
- **File synchronization**: Mnemonic does not sync files between workstations; file operations are strictly local
- **Real-time collaboration**: Mnemonic does not provide real-time collaborative editing or presence features

## Success Criteria

### Phase 1 (MVP Local Deployment)

| Criterion             | Measure                                                                   |
| --------------------- | ------------------------------------------------------------------------- |
| Pattern search        | Claude Code can search team knowledge graph via MCP                       |
| Tooling sync          | Agents, skills, commands accessible to all team members via MCP           |
| Data loading          | Patterns and tooling loadable via REST admin API                          |
| Docker Compose deploy | Single server with two listeners runs locally without authentication      |

### Phase 2 (Production Deployment)

| Criterion              | Measure                                                      |
| ---------------------- | ------------------------------------------------------------ |
| Admin API auth         | Envoy + OPA protect write operations on admin API           |
| TLS termination        | HTTPS for all external traffic                               |
| Production config      | Postgres and Neo4j configured for production workloads       |

### Quality Attributes

- **Reliability**: Pattern search and tooling sync work consistently
- **Performance**: MCP protocol overhead does not significantly impact Claude Code workflow
- **Maintainability**: Patterns and tooling can be updated via admin API without client changes
- **Observability**: Pattern searches and tooling requests are logged for analysis

## Constraints

### Technical Constraints

- **Claude Code dependency**: Mnemonic integrates with Claude Code via MCP protocol
- **Network connectivity**: Claude Code must reach Mnemonic MCP server for reads
- **Neo4j required**: Knowledge graph functionality requires Neo4j (not optional)

### Organizational Constraints

- **Existing workflows**: Must integrate seamlessly with how teams currently use Claude Code
- **Security requirements**: Patterns and tooling definitions may contain sensitive information
- **Operational capacity**: Server infrastructure should be minimal and easy to maintain

## Assumptions

1. **Claude Code availability**: Team members have Claude Code installed and configured with MCP support
2. **Network access**: Workstations can reach the Mnemonic MCP server endpoint
3. **Anthropic accounts**: Users have valid Anthropic API access via Claude Code
4. **Pattern quality**: Teams will maintain and curate patterns stored in Mnemonic
5. **Tooling governance**: Someone owns the responsibility for maintaining agent, skill, and command definitions

**Next:** [Architectural Decisions](02-architectural-decisions.md)
