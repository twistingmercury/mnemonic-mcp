---
entity_name: README Structure Guideline
entity_type: engineering-guideline
language: agnostic
domain: documentation
description: Required README sections and formatting standards
tags:
  - documentation
  - readme
  - project-setup
---

# README Structure Guideline

## Overview

Every project MUST include a README.md file that follows this exact structure. No exceptions.

## Required Structure Template

```markdown
# Project Name

> **Maturity Level**: [Emerging|Basic|Mature] - (a short sentence fragment for context)
>
> - **Emerging**: Prototype, not production-ready, expect breaking changes
> - **Basic**: Production-ready but actively evolving, expect minor version changes
> - **Mature**: Stable, battle-tested, changes are rare

***

A sentence describing the project. Two at most.

## Usage

## How it works

## Key Considerations

## Development Considerations

### Quick Start

### Building & running

### Testing

### Versioning
```

## Section Requirements

### Maturity Level (Required)

**Purpose**: Immediately communicate the project's stability and readiness.

**Format**:

```markdown
> **Maturity Level**: [Level] - (brief context)
```

**Levels**:

- **Emerging**: Prototype, not production-ready, expect breaking changes
- **Basic**: Production-ready but actively evolving, expect minor version changes
- **Mature**: Stable, battle-tested, changes are rare

**Example**:

```markdown
> **Maturity Level**: Basic - Core features work but API may evolve based on feedback
```

### Project Description (Required)

**Purpose**: Provide a clear, concise overview of what the project does.

**Rules**:

- Maximum two sentences
- Focus on the value, not implementation details
- Answer: "What does this do and why does it exist?"

**Example**:

```markdown
A CLI tool for managing database migrations across multiple environments. Supports PostgreSQL, MySQL, and SQLite with automatic rollback capabilities.
```

### Usage (Required)

**Purpose**: Show users how to actually use the project.

**Content**:

- Command-line examples for CLI tools
- API calls for services
- Import/usage examples for libraries
- Common use cases
- Configuration options

**Example**:

```markdown
## Usage

Run migrations:

```bash
dbmigrate up --env production
```

Rollback last migration:

```bash
dbmigrate down --steps 1
```

Check migration status:

```bash
dbmigrate status
```
```

### How it works (Required)

**Purpose**: Explain the high-level approach, architecture, or design philosophy.

**Content**:

- Core concepts
- Architecture overview (not implementation details)
- Design decisions and trade-offs
- How components interact

**Rules**:

- Focus on concepts, not code details
- Explain the "why" behind the approach
- Keep it understandable for someone not familiar with the codebase

**Example**:

```markdown
## How it works

The tool tracks migrations using a version table in your database. Each migration is executed in a transaction, allowing automatic rollback if errors occur. Migration files follow a numbered naming convention (001_initial.sql, 002_add_users.sql) to maintain ordering.
```

### Key Considerations (Required)

**Purpose**: Highlight important information users need to know before using the project.

**Content**:

- Limitations or known issues
- Assumptions or prerequisites
- Security considerations
- Performance considerations
- Breaking changes or migration notes

**Example**:

```markdown
## Key Considerations

- Requires database user with DDL permissions (CREATE/ALTER/DROP)
- Migration transactions may lock tables during execution
- Rollback support requires all migrations to include down scripts
- SQLite support is limited to local file databases
```

### Development Considerations (Required)

**Purpose**: Help contributors and developers work on the project.

This section MUST include four subsections:

#### Quick Start (Required Subsection)

**Purpose**: Get developers running the project as quickly as possible.

**Content**:

- Minimal steps to get a development environment running
- Assume tools are already installed (link to requirements, don't explain installation)
- Focus on commands that get the project running

**Example**:

```markdown
### Quick Start

Clone and run:

```bash
git clone https://github.com/org/dbmigrate.git
cd dbmigrate
make dev
```

Requires Go 1.21+ ([installation instructions](https://go.dev/doc/install))
```

#### Building & running (Required Subsection)

**Purpose**: Explain how to build the project for development and testing.

**Content**:

- Build commands
- How to run locally
- Development mode vs. production mode
- Build artifacts and outputs

**Example**:

```markdown
### Building & running

Build the binary:

```bash
make build
```

Run in development mode with hot reload:

```bash
make dev
```

Build for production:

```bash
make release
```
```

#### Testing (Required Subsection)

**Purpose**: Show how to run tests.

**Content**:

- Commands to run different test types
- Unit tests, integration tests, E2E tests (if applicable)
- Test prerequisites or setup
- Coverage requirements

**Example**:

```markdown
### Testing

Run all tests:

```bash
make test
```

Run unit tests only:

```bash
go test ./...
```

Run integration tests (requires Docker):

```bash
make test-integration
```
```

#### Versioning (Required Subsection)

**Purpose**: Explain how the project is versioned and released.

**Content**:

- Versioning strategy (git tags, semantic versioning)
- How versions are determined
- How to create releases
- Link to CHANGELOG.md if applicable

**Example**:

```markdown
### Versioning

This project follows [Semantic Versioning 2.0.0](https://semver.org/).

Versions are determined by git tags:

```bash
git describe --tags --always
```

Create a new release by tagging:

```bash
git tag -a v1.2.3 -m "Release v1.2.3"
git push origin v1.2.3
```

See [CHANGELOG.md](CHANGELOG.md) for version history.
```

## README Writing Rules

Follow these rules to keep READMEs useful and professional:

### Content Rules

1. **No emojis**: They look unprofessional and add zero value (checkboxes are fine)
2. **Working links only**: Test every link before committing. Broken links destroy credibility
3. **Never duplicate docs**: Reference other docs, don't copy-paste them. DRY applies to documentation too
4. **No file tree diagrams**: They're useless and become outdated instantly
5. **Explain versioning**: Document how the project is versioned (usually git tags)
6. **No workstation setup instructions**: Include version requirements with links to official installation docs, not "how to install Node.js" steps
7. **Write for the reader**: What do they actually need to know?

### Writing Style Rules

1. **Most important stuff first**: Lead with usage, bury the implementation details
2. **Be specific**: "Configure the database" is worthless. "Set the DATABASE_URL environment variable" is helpful
3. **Show, don't just tell**: Include examples. A code snippet is worth a thousand words

### Version Requirements Format

When specifying tool or dependency versions:

**Bad**:

```markdown
Install Node.js:
1. Download from nodejs.org
2. Run the installer
3. Verify with node --version
```

**Good**:

```markdown
Requires Node.js 18+ ([installation instructions](https://nodejs.org/en/download/))
```

**Format**: `[Tool] [Version Range]+ ([link to official installation docs])`

## Examples

### Minimal CLI Tool README

```markdown
# deploy-script

> **Maturity Level**: Emerging - Initial prototype for automating deployments

A shell script that automates deployment of containerized applications to Kubernetes clusters. Handles rollback on failure and sends notifications to Slack.

## Usage

Deploy to staging:

```bash
./deploy.sh staging my-app:v1.2.3
```

Deploy to production with approval:

```bash
./deploy.sh production my-app:v1.2.3 --require-approval
```

## How it works

The script uses kubectl to perform rolling updates. It monitors pod health during deployment and automatically rolls back if any pods fail health checks. Deployment progress is sent to Slack via webhook.

## Key Considerations

- Requires kubectl access to target cluster
- Slack webhook URL must be set in SLACK_WEBHOOK_URL environment variable
- Only supports deployments with health check endpoints
- Rollback may take 2-3 minutes depending on cluster size

## Development Considerations

### Quick Start

Run locally:

```bash
export KUBECONFIG=~/.kube/config
export SLACK_WEBHOOK_URL=https://hooks.slack.com/...
./deploy.sh staging test-app:latest
```

Requires kubectl 1.25+ ([installation](https://kubernetes.io/docs/tasks/tools/))

### Building & running

No build step required. The script runs directly:

```bash
chmod +x deploy.sh
./deploy.sh --help
```

### Testing

Run BATS tests:

```bash
bats tests/
```

Requires BATS 1.8+ ([installation](https://bats-core.readthedocs.io/))

### Versioning

This project uses git tags for versioning (v1.0.0, v1.1.0, etc.). Check current version:

```bash
git describe --tags
```
```

### Go Service README

```markdown
# user-service

> **Maturity Level**: Basic - Core CRUD operations stable, advanced features in development

A RESTful API service for managing user accounts and authentication. Provides JWT-based authentication and role-based access control.

## Usage

Start the service:

```bash
user-service --config config.yaml
```

Create a user:

```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"secret123"}'
```

Authenticate:

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"secret123"}'
```

## How it works

The service uses PostgreSQL for user data storage and Redis for session management. Passwords are hashed using bcrypt. JWT tokens expire after 24 hours and are validated on each API request. The service implements rate limiting per IP address to prevent brute force attacks.

## Key Considerations

- Requires PostgreSQL 14+ with users table schema
- Redis is required for session storage (in-memory sessions are not supported)
- JWT secret must be at least 32 characters and set in JWT_SECRET environment variable
- Rate limiting is set to 100 requests per minute per IP
- Does not support OAuth or SSO integration (planned for v2.0)

## Development Considerations

### Quick Start

Clone and run:

```bash
git clone https://github.com/org/user-service.git
cd user-service
make dev
```

Requires Go 1.21+, Docker 24+, and Docker Compose 2.20+ ([Go install](https://go.dev/doc/install), [Docker install](https://docs.docker.com/get-docker/))

### Building & running

Build the binary:

```bash
make build
```

Run locally with development database:

```bash
docker-compose up -d postgres redis
make run
```

Build Docker image:

```bash
make docker-build
```

### Testing

Run all tests:

```bash
make test
```

Run unit tests only:

```bash
go test -short ./...
```

Run integration tests (requires Docker):

```bash
make test-integration
```

Run E2E tests:

```bash
make test-e2e
```

### Versioning

This project follows [Semantic Versioning 2.0.0](https://semver.org/).

Version is determined from the latest git tag:

```bash
git describe --tags --always
```

Create a new release:

```bash
git tag -a v1.2.0 -m "Release v1.2.0"
git push origin v1.2.0
```

See [CHANGELOG.md](CHANGELOG.md) for version history and release notes.
```

## Validation Checklist

Before committing a README, verify:

- [ ] Maturity level is specified with context
- [ ] Project description is 1-2 sentences maximum
- [ ] All required sections present (Usage, How it works, Key Considerations, Development Considerations)
- [ ] All required subsections present (Quick Start, Building & running, Testing, Versioning)
- [ ] No emojis used
- [ ] All markdown links work and reference existing files
- [ ] No content duplication from other documentation
- [ ] No file tree diagrams
- [ ] Versioning strategy explained
- [ ] Version requirements use format: `[Tool] [Range]+ ([link])`
- [ ] Examples show actual commands/code
- [ ] Writing is specific and actionable
- [ ] Most important information (usage) comes first

## Related Guidelines

- [Keep a Changelog](https://keepachangelog.com/) - CHANGELOG.md format
- [Semantic Versioning](https://semver.org/) - Version numbering scheme
- Engineering Handbook: Documentation section
