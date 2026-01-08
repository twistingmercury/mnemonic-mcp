---
entity_name: BATS Test Isolation Pattern
entity_type: bats-testing-pattern
language: bash
domain: testing
description: Test isolation patterns for BATS with temporary directories, cleanup strategies, and independent test execution
tags:
  - BATS
  - test-isolation
  - cleanup
  - bash
  - testing
---

# BATS Test Isolation Pattern

## Philosophy

Test isolation ensures that each test runs independently without interference from other tests or system state. Proper isolation prevents flaky tests, enables parallel execution, and ensures tests can run in any order with consistent results.

## Quality Standards

All BATS tests MUST:
- **Pass shellcheck** with the project's `.shellcheckrc` configuration
- **Be POSIX compliant** - use `printf` not `echo`, avoid bash-isms
- **Be readable** - extract variables, avoid terse one-liners
- **Assume tools available** - jq, yq, docker, shellcheck (no availability checks needed)

## Core Approach

1. **Temporary Directories**:
   - Always use `$BATS_TEST_TMPDIR` for temporary files
   - Create unique subdirectories per test with `$$` suffix
   - Clean up in `teardown()` function

2. **Environment Variable Isolation**:
   - Override environment variables in `setup()`
   - Restore original values in `teardown()`
   - Never modify global system environment permanently

3. **Resource Isolation**:
   - Use unique names for all resources (files, Docker volumes, etc.)
   - Clean up all resources after each test
   - Handle cleanup failures gracefully

4. **State Independence**:
   - Each test starts with clean state
   - Tests don't depend on execution order
   - Tests don't share data or resources

## Using $BATS_TEST_TMPDIR

```bash
#!/usr/bin/env bats

SCRIPT_PATH="${BATS_TEST_DIRNAME}/../../scripts/process-files.sh"

setup() {
    # Create unique test directory under BATS_TEST_TMPDIR
    # The $$ suffix ensures uniqueness even for parallel tests
    export TEST_DIR="${BATS_TEST_TMPDIR}/test-$$"
    mkdir -p "$TEST_DIR"

    # Create subdirectories for test isolation
    export INPUT_DIR="$TEST_DIR/input"
    export OUTPUT_DIR="$TEST_DIR/output"
    export CONFIG_DIR="$TEST_DIR/config"

    mkdir -p "$INPUT_DIR" "$OUTPUT_DIR" "$CONFIG_DIR"

    # Create test fixtures
    printf '%s\n' "test data" > "$INPUT_DIR/test.txt"
}

teardown() {
    # Clean up test directory
    # BATS automatically cleans up $BATS_TEST_TMPDIR after all tests
    # but explicit cleanup is good practice
    rm -rf "$TEST_DIR"
}

@test "script processes files in isolated directory" {
    # Act: Run script with test directories
    run "$SCRIPT_PATH" --input "$INPUT_DIR" --output "$OUTPUT_DIR"

    # Assert
    assert_success
    assert [ -f "$OUTPUT_DIR/test.txt" ]
}
```

## Environment Variable Isolation

### Pattern 1: Override and Restore

```bash
setup() {
    # Save original environment variables
    ORIGINAL_HOME="${HOME:-}"
    ORIGINAL_PATH="${PATH:-}"
    ORIGINAL_CONFIG="${CONFIG_FILE:-}"

    # Create test environment
    export TEST_DIR="${BATS_TEST_TMPDIR}/test-$$"
    mkdir -p "$TEST_DIR/home"

    # Override environment variables
    export HOME="$TEST_DIR/home"
    export PATH="$TEST_DIR/bin:$PATH"
    export CONFIG_FILE="$TEST_DIR/config.yaml"
}

teardown() {
    # Restore original environment
    if [ -n "$ORIGINAL_HOME" ]; then
        export HOME="$ORIGINAL_HOME"
    fi

    if [ -n "$ORIGINAL_PATH" ]; then
        export PATH="$ORIGINAL_PATH"
    fi

    if [ -n "$ORIGINAL_CONFIG" ]; then
        export CONFIG_FILE="$ORIGINAL_CONFIG"
    else
        unset CONFIG_FILE
    fi

    # Clean up test directory
    rm -rf "$TEST_DIR"
}
```

### Pattern 2: Using Subshells for Temporary Changes

```bash
@test "script respects custom environment variable" {
    # Run in subshell to avoid affecting other tests
    (
        export CUSTOM_VAR="test-value"
        run "$SCRIPT_PATH"
        assert_success
        assert_output --partial "test-value"
    )

    # CUSTOM_VAR is not set in parent shell
}
```

### Pattern 3: Script-Specific Environment Override

```bash
setup() {
    export TEST_DIR="${BATS_TEST_TMPDIR}/test-$$"
    mkdir -p "$TEST_DIR"

    # Override script-specific variables
    export VOLUME_NAME="test-volume-$$"
    export BACKUP_DIR="$TEST_DIR/backups"
    export MAX_BACKUPS=5
}

teardown() {
    # No need to restore - these are script-specific
    # and won't affect other tests
    rm -rf "$TEST_DIR"
}
```

## Working Directory Isolation

```bash
setup() {
    export TEST_DIR="${BATS_TEST_TMPDIR}/test-$$"
    mkdir -p "$TEST_DIR"

    # Save current directory
    ORIGINAL_PWD="$PWD"

    # Change to test directory
    cd "$TEST_DIR" || exit 1
}

teardown() {
    # Return to original directory
    cd "$ORIGINAL_PWD" || true

    # Clean up test directory
    rm -rf "$TEST_DIR"
}

@test "script creates files in current directory" {
    # We're already in TEST_DIR from setup()
    run "$SCRIPT_PATH" --create-config

    assert_success
    assert [ -f "config.yaml" ]
}
```

## File System Isolation

```bash
setup() {
    export TEST_DIR="${BATS_TEST_TMPDIR}/test-$$"

    # Create isolated file structure
    mkdir -p "$TEST_DIR"/{input,output,logs,temp}

    # Create test config that points to test directories
    cat > "$TEST_DIR/config.yaml" <<EOF
directories:
  input: $TEST_DIR/input
  output: $TEST_DIR/output
  logs: $TEST_DIR/logs
  temp: $TEST_DIR/temp
EOF

    export CONFIG_FILE="$TEST_DIR/config.yaml"
}

teardown() {
    # Clean up entire test directory tree
    rm -rf "$TEST_DIR"
}
```

## Docker Resource Isolation

```bash
setup() {
    export TEST_DIR="${BATS_TEST_TMPDIR}/test-$$"
    mkdir -p "$TEST_DIR"

    # Use unique Docker resource names
    export VOLUME_NAME="test-volume-$$"
    export CONTAINER_NAME="test-container-$$"
    export NETWORK_NAME="test-network-$$"

    # Create Docker resources
    docker volume create "$VOLUME_NAME"
}

teardown() {
    # Clean up Docker resources (suppress errors)
    docker volume rm "$VOLUME_NAME" 2>/dev/null || true
    docker rm -f "$CONTAINER_NAME" 2>/dev/null || true
    docker network rm "$NETWORK_NAME" 2>/dev/null || true

    # Clean up test directory
    rm -rf "$TEST_DIR"
}
```

## Parallel Test Execution Safety

```bash
#!/usr/bin/env bats

# These tests can run in parallel because they are properly isolated

setup() {
    # Use process ID for uniqueness
    export TEST_ID="$$"
    export TEST_DIR="${BATS_TEST_TMPDIR}/test-${TEST_ID}"
    export VOLUME_NAME="test-volume-${TEST_ID}"
    export LOCK_FILE="$TEST_DIR/script.lock"

    mkdir -p "$TEST_DIR"
    docker volume create "$VOLUME_NAME"
}

teardown() {
    # Clean up unique resources
    docker volume rm "$VOLUME_NAME" 2>/dev/null || true
    rm -rf "$TEST_DIR"
}

@test "test 1 - isolated from other tests" {
    # Uses TEST_DIR and VOLUME_NAME unique to this process
    printf '%s\n' "data-1" > "$TEST_DIR/data.txt"
    run "$SCRIPT_PATH" --input "$TEST_DIR/data.txt"
    assert_success
}

@test "test 2 - also isolated from other tests" {
    # Uses different TEST_DIR and VOLUME_NAME (different $$)
    printf '%s\n' "data-2" > "$TEST_DIR/data.txt"
    run "$SCRIPT_PATH" --input "$TEST_DIR/data.txt"
    assert_success
}

# Run tests in parallel:
# bats --jobs 4 tests/bats/test-script.bats
```

## Fixture Isolation

```bash
setup() {
    export TEST_DIR="${BATS_TEST_TMPDIR}/test-$$"
    export FIXTURES_DIR="${BATS_TEST_DIRNAME}/fixtures"

    mkdir -p "$TEST_DIR"

    # Copy fixtures to isolated test directory
    # Each test gets its own copy, preventing interference
    if [ -d "$FIXTURES_DIR" ]; then
        cp -r "$FIXTURES_DIR"/* "$TEST_DIR/"
    fi
}

teardown() {
    # Remove test directory with modified fixtures
    rm -rf "$TEST_DIR"
}

@test "test can modify fixture without affecting other tests" {
    # Modify fixture file
    printf '%s\n' "modified" >> "$TEST_DIR/fixture.txt"

    run "$SCRIPT_PATH" --input "$TEST_DIR/fixture.txt"
    assert_success
}

@test "test gets clean fixture copy" {
    # This test gets original fixture, not modified one
    run grep "modified" "$TEST_DIR/fixture.txt"
    assert_failure
}
```

## State File Isolation

```bash
setup() {
    export TEST_DIR="${BATS_TEST_TMPDIR}/test-$$"
    mkdir -p "$TEST_DIR"

    # Override state file location
    export STATE_FILE="$TEST_DIR/state.json"
    export CACHE_DIR="$TEST_DIR/cache"
    export LOG_FILE="$TEST_DIR/script.log"

    mkdir -p "$CACHE_DIR"
}

teardown() {
    rm -rf "$TEST_DIR"
}

@test "script creates isolated state file" {
    run "$SCRIPT_PATH" --initialize

    assert_success
    assert [ -f "$STATE_FILE" ]

    # State file is in test directory, not system location
    assert_equal "$(dirname "$STATE_FILE")" "$TEST_DIR"
}
```

## Complete Isolation Example

```bash
#!/usr/bin/env bats

load 'test_helper/bats-support/load'
load 'test_helper/bats-assert/load'

SCRIPT_PATH="${BATS_TEST_DIRNAME}/../../scripts/backup-memory-mcp.sh"

setup() {
    # 1. Create isolated file system
    export TEST_DIR="${BATS_TEST_TMPDIR}/test-$$"
    export BACKUP_DIR="$TEST_DIR/backups"
    mkdir -p "$BACKUP_DIR"

    # 2. Create isolated Docker resources
    export VOLUME_NAME="test-memory-$$"
    docker volume create "$VOLUME_NAME"

    # Populate with test data
    printf '%s\n' '{"test": "data"}' | \
        docker run --rm -i -v "${VOLUME_NAME}:/data" \
        alpine sh -c 'cat > /data/memory.json'

    # 3. Override environment variables
    export MAX_BACKUPS=3

    # 4. Save working directory
    ORIGINAL_PWD="$PWD"
}

teardown() {
    # 1. Return to original directory
    cd "$ORIGINAL_PWD" || true

    # 2. Clean up Docker resources
    docker volume rm "$VOLUME_NAME" 2>/dev/null || true

    # 3. Clean up file system
    rm -rf "$TEST_DIR"

    # 4. Environment variables automatically cleaned by BATS
}

@test "fully isolated test execution" {
    # This test is completely isolated:
    # - Uses unique Docker volume (test-memory-$$)
    # - Uses isolated backup directory ($TEST_DIR/backups)
    # - Has custom MAX_BACKUPS setting
    # - Won't interfere with other tests or system

    run "$SCRIPT_PATH"

    assert_success
    assert_output --partial "Backup created"

    # Verify backup in isolated directory
    assert [ "$(find "$BACKUP_DIR" -name "*.json" | wc -l)" -eq 1 ]
}
```

## Key Patterns

1. **$BATS_TEST_TMPDIR**: Always use for temporary files
2. **Unique Names**: Add `$$` suffix to all resource names
3. **setup/teardown**: Create and destroy isolated environment
4. **Save and Restore**: Preserve original environment state
5. **Suppress Errors**: Use `|| true` for cleanup commands
6. **Process ID**: Use `$$` for uniqueness in parallel execution

## Common Pitfalls

- **Using `echo` instead of `printf`**: Not POSIX compliant, behaves differently across platforms
- **Using system directories**: Tests interfere with real system
- **Hardcoded paths**: Tests fail on different systems
- **Shared resource names**: Parallel tests conflict
- **Not cleaning up**: Resources accumulate over time
- **Forgetting to restore**: Environment pollution affects other tests
- **Cleanup order**: Clean Docker before directories (volume might be mounted)
- **Missing error suppression**: Cleanup failures cause test failures
- **Terse one-liners**: Prefer readable code with extracted variables

## Verification Checklist

- Each test uses `$BATS_TEST_TMPDIR`
- Resource names include `$$` suffix
- `setup()` creates isolated environment
- `teardown()` cleans up all resources
- Tests don't modify system state
- Tests can run in any order
- Tests can run in parallel
- Environment variables are isolated
- Cleanup handles errors gracefully
- Tests pass when run multiple times
