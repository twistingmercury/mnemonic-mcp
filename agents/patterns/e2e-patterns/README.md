# E2E Testing Patterns for Cognee

This directory contains comprehensive end-to-end testing patterns that are loaded into the Cognee knowledge graph. These patterns guide the `go-e2e-test-engineer` agent when creating black-box tests for Go applications.

## Pattern Files

Each pattern file contains:

- **Frontmatter**: Entity metadata for Cognee (entity_name, entity_type)
- **Philosophy**: Core testing approach and principles
- **Code Examples**: Complete, runnable test examples
- **Helper Functions**: Reusable utility functions
- **Required Packages**: Go imports needed
- **Common Pitfalls**: Things to avoid
- **Test Coverage Requirements**: What scenarios must be tested

### Available Patterns

1. **cli-testing-pattern.md** - CLI tool testing via binary execution
2. **rest-api-testing-pattern.md** - REST API testing via HTTP client
3. **graphql-testing-pattern.md** - GraphQL API testing via GraphQL client
4. **grpc-testing-pattern.md** - gRPC service testing via gRPC client

## Loading Patterns into Cognee

Patterns are automatically loaded into Cognee during agent operations when agents query for testing patterns. The agents use the Cognee MCP tools to create entities and relations as needed.

### Verification

To verify patterns are loaded:

1. **Search for patterns** (via Claude Code):

   ```text
   Search for "E2E testing patterns"
   ```

2. **Retrieve specific pattern**:

   ```text
   Show me the REST API testing pattern
   ```

3. **Use with agent**:

   ```text
   Use go-e2e-test-engineer agent to create tests for my API
   ```

### Re-running Bootstrap

The bootstrap script is **idempotent** - it's safe to run multiple times. If patterns are updated, simply re-run the script to refresh Cognee.

## Usage with go-e2e-test-engineer Agent

The `go-e2e-test-engineer` agent automatically queries Cognee for patterns when creating tests:

1. **Agent identifies test type** (CLI, REST, GraphQL, gRPC)
2. **Agent searches Cognee**: `search(search_query="REST API testing pattern", search_type="GRAPH_COMPLETION")`
4. **Agent applies pattern**: Uses retrieved code examples and practices

### Example Workflow

```text
User: "I've completed the POST /api/users endpoint. Create E2E tests?"
Agent (go-e2e-test-engineer):
  1. Queries Cognee: search(search_query="REST API testing pattern", search_type="GRAPH_COMPLETION")
  3. Creates comprehensive E2E tests following the pattern

Result: Complete test suite covering happy path, validation errors, authentication,
        not found scenarios, and pagination - all without importing internal code.
```

## Pattern Structure

Each pattern file follows this structure:

````markdown
---
entity_name: Pattern Name
entity_type: e2e-testing-pattern
---

# Pattern Name

## Philosophy

Core testing approach...

## Example Test Structure

```go
func TestFeature_Scenario(t *testing.T) {
    // AAA pattern: Arrange, Act, Assert
}
```
````

## Helper Functions

```go
func helperFunction(t *testing.T, ...) {
    t.Helper()
    // implementation
}
```

## Required Packages

```go
import (
    "testing"
    // other imports
)
```

## Key Patterns

1. Pattern 1
2. Pattern 2

## Common Pitfalls

- Pitfall 1
- Pitfall 2

## Test Coverage Requirements

- Scenario type 1
- Scenario type 2

````

## Benefits

### Context Efficiency
- **Before**: 1,465-line agent definition (~20K-25K tokens, 10-12% of context)
- **After**: 295-line agent + on-demand pattern retrieval (~2-3K tokens)
- **Savings**: ~85% context reduction, leaving more room for codebase

### Knowledge Centralization
- Single source of truth for testing patterns
- Easy to update (edit pattern files, re-run bootstrap)
- Consistent patterns across all projects
- Team members share same knowledge base

### Maintainability
- Patterns versioned in git
- Changes tracked in version control
- Easy to review and approve pattern updates
- Clear audit trail of testing standards evolution

## Troubleshooting

### Bootstrap Script Issues

**Problem**: "jq: command not found"
```bash
# Solution: Install jq
brew install jq
````

**Problem**: "Cognee MCP configuration not found"

```bash
# Solution: Configure Cognee MCP server
# See memory-mcp-server/README.md for setup instructions
```

### Cognee Query Issues

**Problem**: Agent can't find patterns

Solution 1: Verify patterns are loaded:

```
Search Cognee for "e2e-testing-pattern"
```

Solution 2: Check agent definition includes Cognee queries

Solution 3: Patterns are loaded automatically during agent operations - ensure Cognee MCP server is configured and running

## Future Enhancements

### Phase 1 (Current)

- Local Cognee with automatic pattern loading
- CLI, REST, GraphQL, gRPC patterns
- go-e2e-test-engineer agent integration

### Phase 2 (Planned)

- Shared Cognee server deployment
- Team-wide knowledge graph
- Enhanced pattern management

### Pattern Additions

- Integration testing patterns
- Performance testing patterns
- Security testing patterns
- Contract testing patterns
- Chaos engineering patterns

## Contributing

To add or update patterns:

1. **Edit pattern file** in this directory
2. **Follow frontmatter structure**:
   ```markdown
   ---
   entity_name: Your Pattern Name
   entity_type: e2e-testing-pattern
   ---
   ```
3. **Include complete examples** with comments
4. **Add helper functions** following `t.Helper()` pattern
5. **Document pitfalls** from real experience
6. **Load patterns into Cognee** using Cognee MCP tools during agent operations
7. **Commit to version control**:
   ```bash
   git add agents/patterns/e2e-patterns/
   git commit -m "Add/update E2E testing pattern"
   ```

## Related Documentation

- [go-e2e-test-engineer agent](../../definitions/go/go-e2e-test-agent.md)
- [go-software-engineer agent](../../definitions/go/go-software-agent.md)
- [go-devops-engineer agent](../../definitions/go/go-devops-agent.md)
- [Cognee Documentation](https://github.com/topoteretes/cognee)
