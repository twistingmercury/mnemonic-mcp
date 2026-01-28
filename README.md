# ACE (Agent Coordination Engine)

> **Maturity Level**: Emerging - CI/CD pipeline implemented, MVP foundation in progress

ACE is an orchestration layer that provides deterministic agent routing and dynamic pattern retrieval for Claude Code. It replaces LLM-based routing with predictable, code-based logic.

## Usage

ACE is currently in the design phase. Once implemented, users will interact through the ACE CLI, which orchestrates routing decisions and Claude Code execution.

## How it works

ACE is a monorepo containing two binaries built from a single Go module:

| Binary       | Purpose                                                             |
| ------------ | ------------------------------------------------------------------- |
| **mnemonic** | Backend server providing routing and pattern retrieval via REST API |
| **ace**      | CLI client that orchestrates routing decisions and execution        |

The system provides:

- **Deterministic routing** - Code-based logic ensures the same input always routes to the same agent
- **Dynamic patterns** - Patterns retrieved from Mnemonic's knowledge graph (Postgres + PGVector + Neo4j)
- **Local execution** - All LLM interactions and file operations happen on the user's workstation

### Phased Approach

- **Phase 1**: Claude Code integration - CLI invokes Claude Code for execution
- **Phase 2**: Direct API integration - CLI calls Anthropic API directly, removing Claude Code dependency

## Key Considerations

- This project is in early development; architecture is defined but implementation has not started
- The MVP focuses on local deployment with a single agent type (go-software-agent)
- Mnemonic serves only ACE for MVP (not a general-purpose memory service)
- Production features such as authentication, rate limiting, and multi-region deployment are planned for later phases

## Development Considerations

### Quick Start

The project is not yet ready for development setup. See the architecture documentation for design details.

### Building and running

CI/CD automation is now operational for the Mnemonic service. The MVP plan outlines an 8-week phased approach for building the core system.

### Testing

Testing strategy is defined in the MVP plan and includes unit tests, integration tests, and end-to-end validation scripts.

### Versioning

This project uses git tag-based semantic versioning. No releases have been published yet.

## Documentation

### Architecture

- [Architecture Overview](docs/architecture/00-overview.md) - System model, phased approach, key principles
- [Requirements](docs/architecture/01-requirements.md) - Problem statement and success criteria
- [Architectural Decisions](docs/architecture/02-architectural-decisions.md) - Major decisions with rationale
- [System Architecture](docs/architecture/03-system-architecture.md) - Component breakdown and data flow

### Design

- [API Specification](docs/design/mnemonic_service/api-specification.md) - OpenAPI spec for Mnemonic REST API
- [Pattern Processing](docs/design/mnemonic_service/pattern-processing.md) - Pattern enrichment and search pipeline
- [ACE CLI Configuration](docs/design/ace_cli/configuration.md) - CLI configuration reference
- [Mnemonic Configuration](docs/design/mnemonic_service/configuration.md) - Server configuration reference
