# Claude Code Setup

> **Maturity Level**: Emerging - Reference implementation for AI-assisted development with hierarchical agents supporting multiple languages. Not ready for wide-spread adoption!
>
> **Version**: 0.0.2 (2025-11-12)

---

Configuration resources for setting up Claude Code with specialized development agents for Go, Python, .NET, and shell scripts, integrated with MCP (Model Context Protocol) servers.

## Prerequisites

- Claude Code with MCP server support
- `yq` - YAML processor for pattern metadata parsing
- `jq` - JSON processor for pattern validation

## Quick Start

1. Configure Cognee MCP server following instructions in [../memory-mcp-server/README.md](../memory-mcp-server/README.md). This is a necessary first step, otherwise the agents will not function!
2. Run installation script:

   ```bash
   ../scripts/install-agents.sh
   ```

   This loads agent definitions into Claude Code directory and updates your global Claude configuration.

3. Restart Claude Code
4. Reference [definitions/ABOUT-THE-AGENTS.md](definitions/ABOUT-THE-AGENTS.md) for agent usage workflows

## Usage

This repository provides configuration and pattern examples for setting up an AI-assisted development environment supporting multiple languages. Use it to:

1. **Configure MCP servers** - Set up Cognee MCP and optional integration servers (Postgres, Azure DevOps, Atlassian, Microsoft Docs, Context7)
2. **Bootstrap agent patterns** - Load development patterns into Cognee MCP for agent use
3. **Reference agent workflows** - Learn how specialized agents coordinate across languages
4. **Adopt pattern templates** - Use example patterns for APIs, CLIs, testing, and DevOps

Main Claude Code acts as coordinator, delegating to specialized agents based on task complexity and type.

## How it works

**Hierarchical Agent Architecture**: Main Claude coordinates development by consulting architect agents for complex projects and delegating implementation to specialist agents. This reduces cognitive load and allows focused expertise per domain.

**Cognee MCP Integration**: Instead of embedding complete patterns in agent prompts (bloating token usage), agents query Cognee knowledge memory for patterns on-demand. This reduces agent prompt size by approximately 80% while maintaining access to comprehensive pattern libraries.

**Pattern-Driven Development**: Example patterns in `patterns/` cover common development scenarios: REST/GraphQL/gRPC APIs, CLI tools, E2E testing, and containerized deployments. Agents retrieve these patterns from Cognee MCP when needed.

**Agent Specialization**:

- **Architects** (`software architect agent`, `api architect agent`, language-architects for Go/Python/.NET) - Design system architecture and component structure
- **Engineers** (`software agent`, `devops agent` for Go/Python/.NET, `shell script agent`) - Implement designs and infrastructure
- **Test Engineers** (`e2e test agent` for Go/Python/.NET, `bats test agent` for shell scripts) - Create comprehensive test coverage
- **Documentation Agent** (`documentation agent`) - Maintain project documentation

See [definitions/ABOUT-THE-AGENTS.md](definitions/ABOUT-THE-AGENTS.md) for complete workflow details, decision trees, and examples.

## Pattern Examples

The `patterns/` directory contains pattern templates organized by domain:

### API Patterns

- **design/** - API specification patterns (OpenAPI, GraphQL, gRPC, AsyncAPI)
- **go/** - Go implementation patterns for REST, GraphQL, gRPC
- **graphql/** - GraphQL schema design patterns
- **grpc/** - gRPC streaming and service patterns
- **openapi/** - REST API authentication and design patterns

### Shell Script Patterns

Located in `patterns/shell-script-patterns/`:

- `cross-platform-pattern.md` - Writing portable shell scripts across different Unix systems
- `never-nester-pattern.md` - Avoiding deep nesting and improving code flow
- `posix-compliance-pattern.md` - POSIX-compliant scripting practices
- `readability-pattern.md` - Code clarity and maintainability
- `solid-principles-shell-pattern.md` - Applying SOLID principles to shell scripts
- `variable-naming-quoting-pattern.md` - Variable conventions and quote safety
- `library-script-pattern.md` - Creating reusable shell script libraries
- `shell-script-pattern.md` - General shell scripting best practices

**Common Patterns** (in `common-patterns/` subdirectory):

- `file-locking-pattern.md` - Safe concurrent file access
- `retry-logic-pattern.md` - Robust retry mechanisms
- `temp-directory-pattern.md` - Temporary file and directory management

### Testing Patterns

- **bats-patterns/** - BATS test framework patterns (Docker testing, test isolation)
- **e2e-patterns/** - End-to-end testing patterns for services

### Other Patterns

- **cli-patterns/** - CLI tool design using Cobra (configuration, domain architecture)
- **devops-patterns/** - Docker and containerization patterns
- **mcp-servers/** - MCP server configuration examples

All patterns include YAML frontmatter metadata for categorization and Cognee MCP ingestion.

## Key Considerations

**Prerequisites**: Requires Claude Code with MCP server support, `yq`, and `jq`. Cognee MCP server is essential; other MCP servers are optional depending on your integration needs.

**Cognee MCP is required**: The agent system depends on Cognee MCP for pattern storage and retrieval. Without it, agents cannot access the pattern library that defines best practices and templates.

**Experimental architecture**: This is an emerging reference implementation. The hierarchical agent approach and Cognee MCP integration patterns are subject to change as we learn more about effective AI-assisted development workflows.

**Pattern maintenance**: Patterns in `patterns/` should be kept synchronized with Cognee MCP. Patterns are loaded using the Cognee MCP tools during agent operations.

**Not a framework**: This is a configuration reference, not a distributable framework or library. Adapt the patterns and agent definitions to your specific project needs.

## Development Considerations

### Building & running

This repository contains configuration and documentation, not executable code. Key scripts:

- `../scripts/install-agents.sh` - Install agent definitions to Claude Code and update global configuration
- `../scripts/validate-metadata.sh` - Validate pattern metadata (YAML frontmatter)
- `../scripts/load-patterns.sh` - Load pattern files into Cognee dataset
- `../scripts/cognify-patterns.sh` - Process datasets into knowledge graphs
- `../scripts/setup-cognee.sh` - Configure Cognee MCP server
- `../scripts/install-global-agent-rules.sh` - Install global agent coordination rules

Scripts require standard Unix tools: `bash`, `jq`, `yq`

To load and process patterns into Cognee:

```bash
# Step 1: Load patterns into dataset
../scripts/load-patterns.sh

# Step 2: Process into knowledge graph (choose one)
cat ../memory-mcp-server/logs/datasets-loaded.txt | ../scripts/cognify-patterns.sh  # Process loaded datasets
../scripts/cognify-patterns.sh                                                # Process ALL datasets
```

### Shell script quality

All shell scripts are validated with shellcheck. Configuration is in `../.shellcheckrc`:

```bash
# Run shellcheck on all scripts
shellcheck ../scripts/*.sh
```

### Testing

BATS test suite validates scripts and configuration:

```bash
cd ../tests
bats install-agents.bats
```

Or run tests from project root:

```bash
bats tests/install-agents.bats
```

Tests cover:

- Agent installation and configuration
- Script validation and error handling
- Configuration file integrity
- Agent definition structure

Test files are located in `../tests/` directory. Each `.bats` file contains test cases for specific functionality.

### Versioning

This project uses git tag-based versioning with semantic version numbers (e.g., `v0.1.0`, `v1.0.0`). Changes are tracked in [CHANGELOG.md](../CHANGELOG.md) following [Keep a Changelog](https://keepachangelog.com/en/1.0.0/) format.

Releases are created by tagging commits (commits must be signed!):

```bash
git tag -a -s v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0
```

No formal releases yet (Emerging maturity level). Current work tracked in CHANGELOG.md Unreleased section.
