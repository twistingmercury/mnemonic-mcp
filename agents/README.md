# Agents

Specialized development agents and reusable patterns for AI-assisted software
development.

## Quick Start

1. **Configure Cognee MCP** - Follow setup in
   [workbench/README.md](workbench/README.md)
2. **Run installation** - From `workbench/` directory, run `./install.sh`
3. **Restart Claude Code**
4. **Learn workflows** - See [ABOUT-THE-AGENTS.md](ABOUT-THE-AGENTS.md) for
   agent coordination patterns

## What's Inside

- **Agent definitions** (`definitions/`) - Specialized agents for architecture,
  implementation, testing, and deployment across Go, Python, .NET, and shell
- **Development patterns** (`patterns/`) - Reusable templates for APIs, CLIs,
  testing, DevOps, and data engineering
- **Workbench** (`workbench/`) - Local Cognee MCP setup for pattern storage
  and retrieval

## How It Works

**Main Claude coordinates**, delegating to specialized agents:

- **Architects** design systems and create implementation plans
- **Engineers** implement services, APIs, CLIs, and infrastructure
- **Test Engineers** create comprehensive test coverage
- **Documentation** maintains project docs

**Patterns stored in Cognee** instead of agent prompts, reducing agent size by
~80% while providing comprehensive pattern libraries on-demand.

See [ABOUT-THE-AGENTS.md](ABOUT-THE-AGENTS.md) for complete workflows, decision
trees, and examples.

## Pattern Library

The `patterns/` directory organizes reusable templates by domain:

- **api-patterns/** - OpenAPI, GraphQL, gRPC, AsyncAPI specs and Go
  implementations
- **bats-patterns/** - BATS test framework for shell scripts
- **cli-patterns/** - Cobra-based CLI tool design
- **data-patterns/** - PostgreSQL, Neo4j, pgvector, schema design
- **devops-patterns/** - Docker, containerization, deployments
- **e2e-patterns/** - End-to-end testing for services
- **engineering-guidelines/** - Configuration, documentation, observability,
  source management, testing standards
- **go-patterns/** - Go language patterns and best practices
- **shell-script-patterns/** - POSIX compliance, readability, SOLID principles,
  common patterns

All patterns include YAML frontmatter metadata for Cognee ingestion.

## Key Considerations

**Cognee MCP required** - Agents depend on Cognee MCP for pattern retrieval.
Setup instructions in [workbench/README.md](workbench/README.md).

**Experimental architecture** - Hierarchical agent coordination and Cognee
integration patterns are evolving based on real-world usage.

**Not a framework** - This is a reference implementation. Adapt patterns and
agents to your project needs.

## Development

### Prerequisites

- Claude Code with MCP support
- Docker 27+ and Docker Compose 2.32+ (for Cognee infrastructure)
- `yq` - Pattern metadata parsing
- `jq` - Pattern validation
- OpenAI API key (for Cognee)

See [workbench/README.md](workbench/README.md) for complete setup requirements.

### Scripts

Key scripts in `workbench/scripts/`:

- `00-start-memory-infra.sh` - Start Cognee services (Docker Compose)
- `01-install-agents.sh` - Install agent definitions to `~/.claude/agents/`
- `02-install-global-agent-rules.sh` - Install global coordination rules
- `03-validate-metadata.sh` - Validate pattern metadata
- `04-load-patterns.sh` - Load patterns into Cognee datasets
- `05-enrich-patterns.sh` - Process patterns into knowledge graphs

Run `./install.sh` from `workbench/` to orchestrate complete setup.

### Pattern Maintenance

Keep patterns synchronized with Cognee:

```bash
cd workbench

# Load patterns
./scripts/04-load-patterns.sh

# Process into knowledge graph
./scripts/05-enrich-patterns.sh
```

### Shell Script Quality

All scripts validated with shellcheck using `../.shellcheckrc`:

```bash
shellcheck workbench/scripts/*.sh
```
