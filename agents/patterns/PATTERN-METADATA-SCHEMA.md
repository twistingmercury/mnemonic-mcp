# Pattern Metadata Schema

This document defines the required metadata fields for all patterns loaded into Cognee's knowledge memory.

## Purpose

The metadata schema ensures:

1. **Discoverability** - Agents can find relevant patterns through precise queries
2. **Language separation** - Avoid mixing Go examples in Python agent results
3. **Domain organization** - Clear categorization by technical domain
4. **Consistency** - All patterns follow the same structure

## Required Fields

All patterns MUST include these fields in YAML frontmatter:

### `entity_name`

- **Type**: String
- **Description**: Human-readable name for the pattern
- **Example**: `"REST API Specification Pattern"`
- **Rules**:
  - Use title case
  - Be descriptive but concise
  - Include the word "Pattern" at the end

### `entity_type`

- **Type**: String
- **Description**: Specific category of pattern
- **Example**: `"api-design"`, `"testing-pattern"`, `"devops-pattern"`
- **Rules**:
  - Use kebab-case
  - Should be specific enough to distinguish from other types

### `language`

- **Type**: String
- **Description**: Programming language or stack this pattern applies to
- **Allowed values**:
  - `agnostic` - Not tied to any language (API specs, architecture patterns)
  - `go` - Go/Golang specific
  - `python` - Python specific
  - `dotnet` - .NET/C# specific
  - `shell` - Shell script (bash, sh)
  - `typescript` - TypeScript/JavaScript
  - `react` - React specific (extends typescript)
- **Example**: `"go"`, `"agnostic"`
- **Rules**:
  - Use lowercase
  - Choose `agnostic` if the pattern applies to multiple languages
  - If showing implementation in multiple languages, split into separate files

### `domain`

- **Type**: String
- **Description**: Technical domain or area of concern
- **Allowed values**:
  - `api-design` - API specifications (OpenAPI, GraphQL schemas, gRPC protos)
  - `backend` - Backend implementation patterns
  - `frontend` - Frontend implementation patterns
  - `testing` - Testing patterns (unit, integration, e2e)
  - `devops` - Infrastructure, CI/CD, containerization
  - `cli` - Command-line interface patterns
  - `documentation` - Documentation patterns
- **Example**: `"api-design"`, `"backend"`
- **Rules**:
  - Use kebab-case
  - Choose the primary domain if pattern spans multiple

### `description`

- **Type**: String (multiline supported)
- **Description**: Detailed explanation of what the pattern provides
- **Example**: `"Comprehensive OpenAPI 3.1 specification pattern for RESTful APIs with standard CRUD operations, pagination, filtering, and versioning"`
- **Rules**:
  - Be specific and actionable
  - Mention key features or capabilities
  - Keep under 200 characters if possible

## Optional Fields

Patterns MAY include these fields for enhanced searchability:

### `tags`

- **Type**: Array of strings
- **Description**: Additional keywords for search
- **Example**: `["REST", "CRUD", "pagination", "OpenAPI"]`
- **Rules**:
  - Use relevant technical terms
  - Include framework names if applicable (gin, gonic, cobra, fastapi, etc.)
  - Don't duplicate information already in other fields

### `version`

- **Type**: String
- **Description**: Version of the spec/framework this pattern targets
- **Example**: `"OpenAPI 3.1"`, `"Gin 1.9+"`, `"Go 1.21+"`
- **Rules**:
  - Include when pattern is version-specific
  - Use semver or conventional version format

### `related_patterns`

- **Type**: Array of strings
- **Description**: Names of related patterns users might also need
- **Example**: `["REST API Implementation Pattern (Go)", "API Authentication Patterns"]`
- **Rules**:
  - Reference by entity_name
  - Only include directly related patterns

## Complete Example

### API Design Pattern (Language-Agnostic)

```yaml
---
entity_name: REST API Specification Pattern
entity_type: api-specification
language: agnostic
domain: api-design
description: Comprehensive OpenAPI 3.1 specification pattern for RESTful APIs with standard CRUD operations, pagination, filtering, and versioning
tags:
  - REST
  - OpenAPI
  - CRUD
  - pagination
  - versioning
version: OpenAPI 3.1
related_patterns:
  - REST API Implementation Pattern (Go)
  - API Authentication Patterns
---
```

### Implementation Pattern (Language-Specific - Go with Gin)

```yaml
---
entity_name: REST API Implementation Pattern (Go)
entity_type: backend-implementation
language: go
domain: backend
description: Go implementation of RESTful API using Gin framework with middleware, error handling, and OpenAPI integration
tags:
  - REST
  - Gin
  - gin-gonic
  - middleware
  - error-handling
version: Go 1.21+
related_patterns:
  - REST API Specification Pattern
  - REST API Testing Pattern (Go)
---
```

### Testing Pattern

```yaml
---
entity_name: REST API Testing Pattern (Go)
entity_type: e2e-testing
language: go
domain: testing
description: End-to-end black-box testing pattern for REST APIs using standard Go http client and testify
tags:
  - E2E
  - integration-testing
  - testify
  - net/http
  - black-box-testing
version: Go 1.21+
related_patterns:
  - REST API Implementation Pattern (Go)
  - REST API Specification Pattern
---
```

## Framework Conventions

When creating implementation patterns, use these preferred frameworks:

### Go

- **REST APIs**: Gin (github.com/gin-gonic/gin)
- **GraphQL**: gqlgen
- **gRPC**: google.golang.org/grpc
- **CLI**: Cobra

### Python

- **REST APIs**: FastAPI
- **GraphQL**: Strawberry or Graphene
- **CLI**: Click or Typer

### .NET

- **REST APIs**: ASP.NET Core Web API
- **GraphQL**: Hot Chocolate
- **gRPC**: Grpc.AspNetCore

## Migration Guide

For existing patterns:

1. **API design patterns** (OpenAPI specs, GraphQL schemas, gRPC protos):

   - Set `language: agnostic`
   - Set `domain: api-design`
   - Remove language-specific mentions from description

2. **Implementation patterns with Go code**:

   - Create language-specific version: append "(Go)" to entity_name
   - Set `language: go`
   - Set `domain: backend` (or appropriate domain)
   - Use Gin framework for REST API examples
   - Move to language-specific directory structure

3. **Testing patterns**:

   - Append language to entity_name if contains language-specific code
   - Set `language` field appropriately
   - Set `domain: testing`

4. **DevOps patterns**:
   - If Dockerfile/CI is language-specific, note in language field
   - If generic containerization, use `agnostic`
   - Set `domain: devops`

## Directory Structure

Organize patterns to match metadata:

```text
examples/
  api-patterns/
    design/                    # language: agnostic, domain: api-design
      rest-api-specification-pattern.md
      graphql-schema-pattern.md
      grpc-service-definition-pattern.md
    go/                        # language: go, domain: backend
      rest-api-implementation-go.md
      graphql-implementation-go.md
      grpc-implementation-go.md
    python/                    # language: python, domain: backend
      rest-api-implementation-python.md
      graphql-implementation-python.md

  testing-patterns/
    go/                        # language: go, domain: testing
      rest-api-testing-go.md
      graphql-testing-go.md
    python/                    # language: python, domain: testing
      rest-api-testing-python.md
    shell/                     # language: shell, domain: testing
      bats-test-structure.md
      bats-assertions.md

  cli-patterns/
    design/                    # language: agnostic, domain: cli
      cli-architecture-pattern.md
    go/                        # language: go, domain: cli
      cobra-root-command-pattern.md
      cobra-subcommand-pattern.md

  devops-patterns/
    go/                        # language: go, domain: devops
      service-dockerfile-go.md
      service-build-script-go.md
    python/                    # language: python, domain: devops
      service-dockerfile-python.md
    agnostic/                  # language: agnostic, domain: devops
      azure-devops-pipeline-pattern.md
```

## Query Examples

How agents should query for patterns:

```javascript
// Go software engineer looking for REST API implementation
"REST API implementation pattern for Go backend using Gin";

// Python engineer looking for GraphQL implementation
"GraphQL implementation pattern for Python backend";

// Any engineer looking at API design
"REST API specification pattern OpenAPI";

// Go engineer looking for testing patterns
"REST API testing pattern for Go";

// DevOps engineer looking for containerization
"Dockerfile pattern for Go services";
```

## Validation

Before loading patterns into Cognee, validate:

1. All required fields present
2. `language` is one of allowed values
3. `domain` is one of allowed values
4. `entity_type` uses kebab-case
5. `description` is non-empty
6. File location matches metadata (language/domain)

The loading script will perform these validations automatically.
