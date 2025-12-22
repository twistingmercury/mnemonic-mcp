# Team Agentic Setup

> **Maturity Level**: Emerging - Reference implementation for team-wide AI-assisted development with shared memory! <u>Not ready for general adoption!</u>
>
> **Version**: 0.0.3

---

This is a hierarchical agent setup for Claude Code that uses Cognee MCP to give agents on-demand access to development patterns. The goal is to cut down on context usage while keeping things consistent across your team.

## Prerequisites

- Claude Code with MCP server support
- Docker and Docker Compose - [Installation instructions](https://docs.docker.com/get-docker/)
- `yq` - [Installation instructions](https://github.com/mikefarah/yq#install)
- `jq` - [Installation instructions](https://stedolan.github.io/jq/download/)
- `bats` - [Installation instructions](https://bats-core.readthedocs.io/en/stable/installation.html)
- `markdownlint` - [Installation instructions](https://github.com/igorshubovych/markdownlint-cli#installation)
- `shellcheck` - [Installation instructions](https://github.com/koalaman/shellcheck#installing)

## Quick Start

1. **Start Cognee Services** - See [memory-mcp/README.md](./memory-mcp/README.md)
2. **Configure MCP Connection** - `claude mcp add --scope user --transport sse cognee http://YOUR_SERVER_IP:4000/sse`
3. **Install Agent Definitions** - Run `./scripts/install-agents.sh`
4. **Load and Process Patterns** - Run `./scripts/load-patterns.sh` then `cat memory-mcp/logs/datasets-loaded.txt | ./scripts/cognify-patterns.sh`
5. **Start Using Agents** - See [agents/definitions/ABOUT-THE-AGENTS.md](./agents/definitions/ABOUT-THE-AGENTS.md)

## Usage

Main Claude coordinates your development work by handing off tasks to specialized agents depending on what you're building. Instead of loading everything upfront, agents pull patterns from Cognee MCP only when they need them.

**Available agents**: software architect agent, api architect agent, go software agent, go e2e test agent, shell script agent, bats test agent, documentation agent

**Pattern library**: API patterns (REST, GraphQL, gRPC, AsyncAPI), CLI patterns (Cobra), Shell script patterns (POSIX, cross-platform, readability), Testing patterns (E2E, BATS), DevOps patterns (Docker, CI/CD, Azure DevOps)

Check out [agents/README.md](./agents/README.md) for complete agent details and [agents/definitions/ABOUT-THE-AGENTS.md](./agents/definitions/ABOUT-THE-AGENTS.md) for workflows.

## How it works

**Hierarchical agents**: Main Claude talks to architects when you need design work, hands off to engineers for implementation, and brings in test engineers for validation. Each agent focuses on what it does best.

**Cognee MCP integration**: Instead of stuffing patterns into prompts, agents query Cognee's knowledge memory when they need something. This cuts agent size by about 80% while still giving them access to comprehensive pattern libraries.

**Shared memory**: Everyone on your team connects to the same Cognee server. When someone updates a pattern, it's immediately available to everyone - keeps things consistent across the team.

## Key Considerations

**Cognee MCP is required**: The agents need Cognee to retrieve patterns. Head over to [memory-mcp/README.md](./memory-mcp/README.md) to get it set up.

**Network access needed**: Your team needs to be able to reach the Cognee server on port 4000.

**Resource requirements**: Cognee needs around 8-12GB RAM depending on which LLM provider you're using. Check [memory-mcp/README.md](./memory-mcp/README.md) for configuration details.

**Experimental architecture**: This is an emerging reference implementation - expect things to change as we learn what works.

**Pattern maintenance**: Keep the patterns in [agents/patterns/](./agents/patterns/) in sync with Cognee using the `/load_patterns` command.

**Not a framework**: Think of this as a configuration reference you can adapt to your needs, not a ready-to-go library.

## Development Considerations

### Building & running

**Cognee server**: See [memory-mcp/README.md](./memory-mcp/README.md)

**Agent installation**: Run `./scripts/install-agents.sh`

**Pattern loading and processing**:

```bash
# Step 1: Load patterns into Cognee dataset
./scripts/load-patterns.sh

# Step 2: Process into knowledge graph (choose one):
cat memory-mcp/logs/datasets-loaded.txt | ./scripts/cognify-patterns.sh  # Process loaded datasets
./scripts/cognify-patterns.sh                                             # Process ALL datasets

# Monitor async processing
docker compose -f memory-mcp/docker-compose.yml logs -f cognee-api
```

**Pattern validation**: Run `./scripts/validate-metadata.sh`

### Testing

There's a BATS test suite in the `tests/` directory. Run tests with:

```bash
bats tests/install-agents.bats
```

### Shell script quality

We validate all scripts with shellcheck:

```bash
shellcheck scripts/*.sh
```

### Versioning

We use git tag-based versioning with semantic version numbers. Changes are tracked in [CHANGELOG.md](./CHANGELOG.md) following [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

To create a release, tag the commit:

```bash
git tag -a -s v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0
```

## Documentation

- [docs/about.md](./docs/about.md) - Project backstory and motivation
- [agents/README.md](./agents/README.md) - Agent system details
- [agents/definitions/ABOUT-THE-AGENTS.md](./agents/definitions/ABOUT-THE-AGENTS.md) - Agent workflows and delegation patterns
- [memory-mcp/README.md](./memory-mcp/README.md) - Cognee server setup
- [docs/PATTERN-METADATA-SCHEMA.md](./docs/PATTERN-METADATA-SCHEMA.md) - Pattern metadata specification
