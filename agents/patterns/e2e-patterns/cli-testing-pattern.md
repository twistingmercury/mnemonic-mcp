---
entity_name: CLI Testing Pattern
entity_type: e2e-testing-pattern
language: go
domain: testing
description: End-to-end testing pattern for CLI applications with command execution, output validation, error handling, and test fixtures using Go testing framework
tags:
  - E2E
  - CLI
  - testing
  - Go
  - command-testing
  - separate-module
  - docker
  - CI
---

# CLI Testing Pattern

## Philosophy

Execute the actual compiled binary as a subprocess, exactly as end users would. Never import internal CLI packages. This ensures true black-box testing from the user's perspective.

## Core Approach

1. **Binary Execution via `os/exec`**:

   - Execute the CLI binary from tests
   - Capture stdout, stderr, and exit codes
   - Test various flag combinations and arguments
   - Validate output format and content

2. **Test with Real Infrastructure**:

   - Use Docker Compose to provide real API services
   - Include actual databases for state verification
   - Test against deployed services, not mocks

3. **Test Isolation**:
   - Each test cleans up before and after
   - Use `t.Cleanup()` for guaranteed teardown
   - Tests must run in any order
   - No shared state between tests

## Separate Go Module for E2E Tests

E2E tests should be a separate Go module with their own `go.mod` to isolate test dependencies from the main application. This keeps testify and other test-only dependencies out of the main module.

Example structure:

```text
project/
├── go.mod              # Main application module
├── go.sum
├── cmd/
├── tests/
│   └── e2e/
│       ├── go.mod      # Separate E2E test module
│       ├── go.sum
│       └── cli_test.go
```

Example go.mod for E2E tests:

```text
module github.com/org/project/tests/e2e

go 1.24

require github.com/stretchr/testify v1.10.0
```

## Binary Discovery for Docker/CI

Tests must support finding the binary via environment variable when running in Docker or CI, with fallback to relative paths for local development.

```go
// getBinaryPath returns the path to the CLI binary.
// Checks CLI_BINARY_PATH env var first (for Docker/CI),
// then falls back to relative paths for local development.
func getBinaryPath(t *testing.T) string {
    t.Helper()

    // Check env var first (used in Docker/CI)
    if envPath := os.Getenv("CLI_BINARY_PATH"); envPath != "" {
        if _, err := os.Stat(envPath); err == nil {
            return envPath
        }
    }

    // Fall back to relative paths for local development
    wd, err := os.Getwd()
    require.NoError(t, err)

    possiblePaths := []string{
        filepath.Join(wd, ".bin", "cli-tool"),
        filepath.Join(wd, "..", "..", ".bin", "cli-tool"),
    }

    for _, path := range possiblePaths {
        if _, err := os.Stat(path); err == nil {
            return path
        }
    }

    t.Skip("CLI binary not found. Run 'make build' first or set CLI_BINARY_PATH")
    return ""
}
```

## Example Test Structure

```go
package integration

import (
    "os/exec"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

// TestUserAdd_ValidYAMLWithAllFields tests the happy path for user creation.
func TestUserAdd_ValidYAMLWithAllFields(t *testing.T) {
    const testEmail = "jane.smith@company.com"

    // Setup: Clean up any existing test user
    deleteUser(t, testEmail)

    // Cleanup: Ensure user is deleted after test
    t.Cleanup(func() {
        deleteUser(t, testEmail)
    })

    // Execute: Run CLI command
    output, err := execCommand(t,
        "--skip-auth",
        "user", "add",
        "--yaml", "../testdata/users/valid-user-complete.yaml",
    )

    // Assert: Command should succeed
    require.NoError(t, err, "command should succeed\nOutput: %s", output)
    assert.Contains(t, output, "Successfully",
        "output should indicate success")

    // Verify: User should exist in database
    assert.True(t, userExists(t, testEmail),
        "user %s should exist in database", testEmail)
}

// TestUserFind_InvalidUUIDFormat tests validation error handling.
func TestUserFind_InvalidUUIDFormat(t *testing.T) {
    // Execute: Run command with invalid UUID
    output, err := execCommand(t,
        "--skip-auth",
        "user", "find",
        "--id", "not-a-uuid",
    )

    // Assert: Command should fail with validation error
    require.Error(t, err, "command should fail")
    assert.Contains(t, output, "invalid UUID format")
    assert.Contains(t, output, "Usage:") // Help should be shown
}

// TestUserDisable_ConfirmationPrompt tests interactive command.
func TestUserDisable_ConfirmationPrompt(t *testing.T) {
    userID := "550e8400-e29b-41d4-a716-446655440000"

    // Execute: Run command with "yes" stdin input
    output, err := execCommandWithStdin(t, "yes\n",
        "--skip-auth",
        "user", "disable",
        "--id", userID,
    )

    // Assert: Command should succeed
    require.NoError(t, err)
    assert.Contains(t, output, "Successfully disabled")
}
```

## Helper Functions

### Command Execution

```go
// execCommand executes the CLI binary with given arguments.
func execCommand(t *testing.T, args ...string) (string, error) {
    t.Helper()

    cmd := exec.Command(cliBinary, args...)
    output, err := cmd.CombinedOutput()

    return string(output), err
}

// execCommandWithStdin executes CLI with stdin input (for interactive commands).
func execCommandWithStdin(t *testing.T, stdin string, args ...string) (string, error) {
    t.Helper()

    cmd := exec.Command(cliBinary, args...)

    stdinPipe, err := cmd.StdinPipe()
    if err != nil {
        t.Fatalf("failed to create stdin pipe: %v", err)
    }

    if err := cmd.Start(); err != nil {
        return "", err
    }

    if stdin != "" {
        stdinPipe.Write([]byte(stdin))
    }
    stdinPipe.Close()

    output, err := cmd.CombinedOutput()
    return string(output), err
}

// execCommandWithEnv executes CLI with custom environment variables.
func execCommandWithEnv(t *testing.T, env []string, args ...string) (string, error) {
    t.Helper()

    cmd := exec.Command(cliBinary, args...)
    cmd.Env = append(os.Environ(), env...)
    output, err := cmd.CombinedOutput()

    return string(output), err
}

// createTempYAML creates a temporary YAML file with given content.
func createTempYAML(t *testing.T, content string) string {
    t.Helper()

    tmpFile, err := os.CreateTemp("", "test-*.yaml")
    require.NoError(t, err)

    _, err = tmpFile.WriteString(content)
    require.NoError(t, err)

    require.NoError(t, tmpFile.Close())

    t.Cleanup(func() {
        os.Remove(tmpFile.Name())
    })

    return tmpFile.Name()
}
```

### Database Verification

```go
// userExists checks if a user with given email exists in the database.
func userExists(t *testing.T, email string) bool {
    t.Helper()

    var count int
    err := db.QueryRow(
        "SELECT COUNT(*) FROM Users WHERE Email = ?",
        email,
    ).Scan(&count)

    if err != nil {
        t.Logf("error checking user existence: %v", err)
        return false
    }

    return count > 0
}

// deleteUser removes a user by email for test cleanup.
func deleteUser(t *testing.T, email string) {
    t.Helper()

    _, err := db.Exec("DELETE FROM Users WHERE Email = ?", email)
    if err != nil {
        t.Logf("warning: failed to delete user: %v", err)
    }
}

// companyExists checks if a company exists by domain.
func companyExists(t *testing.T, domain string) bool {
    t.Helper()

    var count int
    err := db.QueryRow(
        "SELECT COUNT(*) FROM Companies WHERE Domain = ?",
        domain,
    ).Scan(&count)

    return err == nil && count > 0
}
```

## Required Packages

```go
import (
    "os"
    "os/exec"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    // Database drivers
    "database/sql"
    _ "github.com/lib/pq"                    // PostgreSQL
    _ "github.com/denisenkom/go-mssqldb"     // SQL Server
    _ "github.com/go-sql-driver/mysql"       // MySQL
)
```

## Key Patterns

1. **AAA Pattern**: Arrange (setup), Act (execute), Assert (verify)
2. **Binary Execution**: Always use `os/exec` to run the compiled binary
3. **Test Isolation**: Use `t.Cleanup()` for guaranteed cleanup
4. **Database Verification**: Verify state via direct SQL queries (external verification)
5. **No Internal Imports**: Never import CLI internal packages

## Common Pitfalls

- **Forgetting t.Cleanup()**: Always register cleanup functions
- **Not cleaning up before tests**: Tests should clean up at start AND end
- **Importing internal packages**: Breaks black-box testing philosophy
- **Assuming output format**: Always test actual output, don't assume
- **Not testing error cases**: Test validation errors, missing files, etc.
- **Shared state**: Each test must be completely independent

## Test Coverage Requirements

All CLI commands must be tested for:

### Happy Path

- Valid input with minimal required fields
- Valid input with all optional fields
- Multiple output formats (JSON, YAML, table)

### Validation Errors

- Invalid format (UUID, email, domain, etc.)
- Missing required fields
- Empty values
- String length limits
- Invalid YAML/JSON structure
- Non-existent files

### CLI-Specific Scenarios

- Network timeouts/connection errors
- Configuration precedence (env vars, flags, config files)
- Interactive prompts (confirmation, input)
- Help text display
- Version information
- Exit codes (0 for success, 1 for errors)
