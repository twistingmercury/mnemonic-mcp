# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.0.3] - 2025-11-14

### BREAKING CHANGES

#### Directory Structure Refactoring

- Renamed `claude-agents/` directory to `agents/`
- Renamed `cognee/` directory to `memory-mcp/`
- Created new `agents/definitions/agnostic/` directory for language-agnostic agents

#### Agent Naming Convention Changes

All agent definition files have been renamed:

- Removed `-engineer` suffix from engineer agents
- Added `-agent` suffix to all agent files
- Updated agent names in frontmatter to use spaces (e.g., "api architect agent")

**Specific agent renames:**

- `api-architect` → `api-architect-agent`
- `documentation-engineer` → `documentation-agent`
- `software-architect` → `software-architect-agent`
- `go-architect` → `go-architect-agent`
- `go-devops-engineer` → `go-devops-agent`
- `go-e2e-test-engineer` → `go-e2e-test-agent`
- `go-software-engineer` → `go-software-agent`
- `bats-test-engineer` → `bats-test-agent`
- `shell-script-engineer` → `shell-script-agent`

**Migration Required**: Users must:

1. Re-run `./scripts/install-agents.sh` to get updated agent definitions
2. Re-run `./scripts/install-global-agent-rules.sh` to update delegation table in `~/.claude/CLAUDE.md`
3. Update any custom references to old agent names

See full list of renamed agents in `agents/definitions/ABOUT-THE-AGENTS.md`

### Changed

- Updated `install-global-agent-rules.sh` to be date-aware and avoid bloating `~/.claude/CLAUDE.md`
- Enhanced agent installation script to preserve user-created agents
- Improved Cognee server configuration and Docker Compose setup
- Switched to pgvector for vector database
- Renamed `claude-agents/examples` to `agents/patterns`

### Added

- Documentation page for the repository (docs/about.md)
- No-report-files instruction added to agent definitions to prevent random report file creation

### Removed

- `update-agents` command removed from available commands

[Unreleased]: https://github.com/twistingmercury/team-agentic-setup/compare/0.0.3...HEAD
[0.0.3]: https://github.com/twistingmercury/team-agentic-setup/compare/0.0.2...0.0.3
[0.0.2]: https://github.com/twistingmercury/team-agentic-setup/compare/0.0.1...0.0.2
[0.0.1]: https://github.com/twistingmercury/team-agentic-setup/releases/tag/0.0.1
