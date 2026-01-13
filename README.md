# ACE (Agentic Coding Engine)

> **Maturity Level**: Emerging - Architecture defined, MVP implementation planned

ACE is a system for deterministic agent routing with dynamic pattern querying, designed to replace LLM-based routing interpretation with predictable, code-based routing logic.

## Usage

ACE is currently in the design and planning phase. Once implemented, users will interact with the system through:

- A CLI client for executing agent tasks
- A REST API for programmatic integration

## How it works

ACE addresses key limitations in current agent systems:

- **Deterministic routing** - Same input always routes to the same agent, enabling reliable automation and predictable workflows
- **Dynamic pattern querying** - Agents query relevant patterns on-demand via Cognee, reducing context size by approximately 78% compared to pre-loading all patterns
- **Team collaboration** - Shared pattern libraries stored in git, accessible across teams
- **Independent scaling** - API server and pattern search scale separately based on their workload profiles

The system uses the Model Context Protocol (MCP) for communication with Cognee and Claude's tool calling feature for dynamic pattern retrieval.

## Key Considerations

- This project is in early development; architecture is defined but implementation has not started
- The MVP focuses on local deployment with a single agent type (go-software-agent)
- Production features such as authentication, rate limiting, and multi-region deployment are planned for later phases

## Development Considerations

### Quick Start

The project is not yet ready for development setup. See the architecture documentation for design details.

### Building and running

Implementation has not started. The MVP plan outlines an 8-week phased approach for building the core system.

### Testing

Testing strategy is defined in the MVP plan and includes unit tests, integration tests, and end-to-end validation scripts.

### Versioning

This project uses git tag-based semantic versioning. No releases have been published yet.

## Documentation

Detailed documentation is available in the `docs/` directory:

- [Architecture Overview](docs/architecture/00-overview.md) - High-level system design and goals
- [Requirements](docs/architecture/01-requirements.md) - Problem statement and success criteria
- [Architectural Decisions](docs/architecture/02-architectural-decisions.md) - Major decisions with rationale
- [MVP Implementation Plan](docs/MVP-IMPLEMENTATION-PLAN.md) - Phased approach for initial development
