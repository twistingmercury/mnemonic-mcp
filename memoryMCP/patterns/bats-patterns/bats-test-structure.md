---
entity_name: BATS Test Structure Pattern
entity_type: bats-testing-pattern
language: bash
domain: testing
description: Recommended test structure and organization patterns for BATS test suites with setup, teardown, and helper functions
tags:
  - BATS
  - test-structure
  - organization
  - bash
  - testing
---

# BATS Test Structure Pattern

## Philosophy

BATS (Bash Automated Testing System) tests should follow a consistent structure with proper setup, teardown, and clear test organization. Each test should be independent, well-named, and follow the AAA (Arrange, Act, Assert) pattern.

## Quality Standards

All BATS tests MUST:

- **Pass shellcheck** with the project's `.shellcheckrc` configuration
- **Be POSIX compliant** - use `printf` not `echo`, avoid bash-isms
- **Be readable** - extract variables, avoid terse one-liners
- **Assume tools available** - jq, yq, docker, shellcheck (no availability checks needed)

## Core Approach

1. **File Structure**:

   - One BATS file per shell script being tested
   - Name format: `test-{script-name}.bats`
   - Place in `tests/bats/` directory
   - Use shebang: `#!/usr/bin/env bats`

2. **Setup and Teardown**:

   - `setup()` runs before each test
   - `teardown()` runs after each test
   - Both are optional but highly recommended
   - Use for test isolation and cleanup

3. **Test Naming**:
   - Clear, descriptive names in quotes
   - Describe what behavior is being validated
   - Format: `@test "component action - expected behavior"`

## Example Test File Structure

```bash
#!/usr/bin/env bats

# Load helper libraries (optional but recommended)
load 'test_helper/bats-support/load'
load 'test_helper/bats-assert/load'

# Global variables (if needed)
SCRIPT_PATH="${BATS_TEST_DIRNAME}/../../scripts/example.sh"
TEST_CONFIG_DIR=""

# setup() runs before each test
setup() {
    # Create isolated test environment
    export TEST_DIR="${BATS_TEST_TMPDIR}/test-$$"
    mkdir -p "$TEST_DIR"

    # Override environment variables
    export CONFIG_DIR="$TEST_DIR/config"
    export OUTPUT_DIR="$TEST_DIR/output"

    # Create required directories
    mkdir -p "$CONFIG_DIR" "$OUTPUT_DIR"

    # Set up test fixtures
    printf '%s\n' '{"test": "data"}' > "$CONFIG_DIR/test.json"

    # Store original directory
    ORIGINAL_DIR="$PWD"
}

# teardown() runs after each test
teardown() {
    # Return to original directory
    cd "$ORIGINAL_DIR" || true

    # Clean up test directory
    rm -rf "$TEST_DIR"

    # Clean up any other resources
    # (Docker volumes, containers, etc.)
}

@test "script succeeds with valid input" {
    # Arrange: Set up test preconditions
    local input_file="$TEST_DIR/input.txt"
    printf '%s\n' "test data" > "$input_file"

    # Act: Execute the script
    run "$SCRIPT_PATH" --input "$input_file"

    # Assert: Verify expected behavior
    assert_success
    assert_output --partial "Success"
}

@test "script fails with missing input file" {
    # Act: Execute script with non-existent file
    run "$SCRIPT_PATH" --input "/nonexistent/file.txt"

    # Assert: Should fail with appropriate error
    assert_failure
    assert_output --partial "not found"
}

@test "script creates expected output files" {
    # Arrange
    local input_file="$TEST_DIR/input.txt"
    printf '%s\n' "test" > "$input_file"

    # Act
    run "$SCRIPT_PATH" --input "$input_file" --output "$OUTPUT_DIR"

    # Assert
    assert_success
    assert [ -f "$OUTPUT_DIR/result.txt" ]
}
```

## Helper Functions

Define reusable helper functions before tests:

```bash
#!/usr/bin/env bats

# Helper: Create test config file
create_test_config() {
    local config_file="${1:-$TEST_DIR/config.yaml}"
    cat > "$config_file" <<EOF
settings:
  enabled: true
  timeout: 30
EOF
    echo "$config_file"
}

# Helper: Wait for file to exist (with timeout)
wait_for_file() {
    local file="$1"
    local timeout="${2:-5}"
    local elapsed=0

    while [ ! -f "$file" ] && [ $elapsed -lt $timeout ]; do
        sleep 1
        elapsed=$((elapsed + 1))
    done

    [ -f "$file" ]
}

# Helper: Count lines in file
count_lines() {
    local file="$1"
    wc -l < "$file" | tr -d ' '
}

@test "example using helpers" {
    # Use helper to create config
    local config_file
    config_file=$(create_test_config)

    # Execute script
    run "$SCRIPT_PATH" --config "$config_file"

    assert_success
}
```

## Test Organization Patterns

### Grouping Related Tests

```bash
# Happy path tests
@test "script succeeds with minimal input" {
    run "$SCRIPT_PATH" --input "minimal"
    assert_success
}

@test "script succeeds with complete input" {
    run "$SCRIPT_PATH" --input "complete" --verbose --output "$TEST_DIR"
    assert_success
}

# Validation error tests
@test "script fails with invalid format" {
    run "$SCRIPT_PATH" --input "invalid-format"
    assert_failure
}

@test "script fails with missing required flag" {
    run "$SCRIPT_PATH"
    assert_failure
}

# Edge case tests
@test "script handles empty input file" {
    local empty_file="$TEST_DIR/empty.txt"
    touch "$empty_file"

    run "$SCRIPT_PATH" --input "$empty_file"
    assert_success
}
```

### Using Skip Directive

```bash
@test "feature not yet implemented" {
    skip "Waiting for feature X to be implemented"

    run "$SCRIPT_PATH" --new-feature
    assert_success
}

@test "slow test - skip by default" {
    if [ -z "$RUN_SLOW_TESTS" ]; then
        skip "Set RUN_SLOW_TESTS=1 to run"
    fi

    run "$SCRIPT_PATH" --slow-operation
    assert_success
}
```

## Environment Variable Management

```bash
setup() {
    # Save original environment variables
    ORIGINAL_PATH="$PATH"
    ORIGINAL_HOME="$HOME"

    # Override for testing
    export HOME="$TEST_DIR/home"
    export PATH="$TEST_DIR/bin:$PATH"
    export CONFIG_FILE="$TEST_DIR/config.yaml"

    # Create test directories
    mkdir -p "$HOME" "$TEST_DIR/bin"
}

teardown() {
    # Restore original environment
    export PATH="$ORIGINAL_PATH"
    export HOME="$ORIGINAL_HOME"

    # Clean up
    rm -rf "$TEST_DIR"
}
```

## Test Data and Fixtures

```bash
setup() {
    export TEST_DIR="${BATS_TEST_TMPDIR}/test-$$"
    export FIXTURES_DIR="${BATS_TEST_DIRNAME}/fixtures"

    mkdir -p "$TEST_DIR"

    # Copy fixtures if they exist
    if [ -d "$FIXTURES_DIR" ]; then
        cp -r "$FIXTURES_DIR"/* "$TEST_DIR/"
    fi
}

@test "script processes test fixture correctly" {
    # Use fixture file
    local fixture="$TEST_DIR/sample-data.json"

    run "$SCRIPT_PATH" --input "$fixture"

    assert_success
    assert_output --partial "Processed 10 records"
}
```

## Required Tools

The following tools are assumed to be available in the development environment:

- `bats` - BATS test framework
- `bats-support` - BATS helper library
- `bats-assert` - BATS assertion library
- `shellcheck` - Shell script linter
- `jq` - JSON processor
- `yq` - YAML processor
- `docker` - Container runtime

**Installation** (for reference only - tests assume these are already installed):

```bash
# Install BATS
brew install bats-core

# Install helper libraries
git clone https://github.com/bats-core/bats-support tests/bats/test_helper/bats-support
git clone https://github.com/bats-core/bats-assert tests/bats/test_helper/bats-assert

# Install other tools
brew install shellcheck jq yq docker
```

## Shellcheck Configuration

Create `tests/bats/.shellcheckrc`:

```bash
# Shellcheck configuration
disable=SC2034,SC2086
external-sources=true
format=gcc
severity=warning
```

## Running Tests

```bash
# Run all tests in a file
bats tests/bats/test-script.bats

# Run all BATS tests
bats tests/bats/*.bats

# Run with tap output
bats --tap tests/bats/test-script.bats

# Run with pretty output
bats --pretty tests/bats/test-script.bats

# Run specific test by line number
bats tests/bats/test-script.bats:42

# Set environment for tests
RUN_SLOW_TESTS=1 bats tests/bats/*.bats
```

## Key Patterns

1. **setup/teardown**: Always use for test isolation
2. **$BATS_TEST_TMPDIR**: Use for all temporary files
3. **Helper functions**: Define once, reuse across tests
4. **Clear test names**: Describe what is being validated
5. **AAA pattern**: Arrange, Act, Assert structure
6. **run command**: Always use `run` to capture output and exit code

## Common Pitfalls

- **Using `echo` instead of `printf`**: Not POSIX compliant, behaves differently across platforms
- **Not passing shellcheck**: Always run `shellcheck tests/bats/*.bats` before delivering
- **Checking for jq/docker availability**: These tools are assumed available, don't check
- **Terse one-liners**: Prefer readable code with extracted variables
- **Not using $BATS_TEST_TMPDIR**: Creates files in unpredictable locations
- **Forgetting teardown()**: Leaves test artifacts behind
- **Not using `run` command**: Can't capture output or test exit codes
- **Shared state between tests**: Tests must be independent
- **Hardcoded paths**: Use variables and $BATS_TEST_DIRNAME
- **Not loading helper libraries**: Miss out on useful assertions
- **Vague test names**: Should be descriptive and specific
- **Using grep -P**: Not portable (BSD doesn't support), use `-E` instead
- **Platform-specific stat**: Use portable version with fallback

## Test File Template

```bash
#!/usr/bin/env bats

# Load helpers
load 'test_helper/bats-support/load'
load 'test_helper/bats-assert/load'

# Script under test
SCRIPT_PATH="${BATS_TEST_DIRNAME}/../../scripts/SCRIPT_NAME.sh"

setup() {
    export TEST_DIR="${BATS_TEST_TMPDIR}/test-$$"
    mkdir -p "$TEST_DIR"

    # Add setup logic here
}

teardown() {
    rm -rf "$TEST_DIR"

    # Add cleanup logic here
}

# Helper functions
# (Define reusable helpers here)

# Tests
@test "describe the happy path behavior" {
    run "$SCRIPT_PATH" --valid-input

    assert_success
    assert_output --partial "expected output"
}

@test "describe the error case behavior" {
    run "$SCRIPT_PATH" --invalid-input

    assert_failure
    assert_output --partial "error message"
}
```
