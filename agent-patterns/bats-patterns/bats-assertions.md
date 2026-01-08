---
entity_name: BATS Assertion Patterns
entity_type: bats-testing-pattern
language: bash
domain: testing
description: Comprehensive assertion patterns for BATS testing using bats-assert library with clear, expressive test failures and error messages
tags:
  - BATS
  - testing
  - assertions
  - bash
  - bats-assert
---

# BATS Assertion Patterns

## Philosophy

Clear, expressive assertions are the foundation of maintainable tests. Use the bats-assert library for readable test failures and descriptive error messages. Each assertion should clearly communicate what is being validated and why it matters.

## Quality Standards

All BATS tests MUST:

- **Pass shellcheck** with the project's `.shellcheckrc` configuration
- **Be POSIX compliant** - use `printf` not `echo`, avoid bash-isms
- **Be readable** - extract variables, avoid terse one-liners
- **Assume tools available** - jq, yq, docker, shellcheck (no availability checks needed)

## Core Approach

1. **Use bats-assert Library**:

   - Provides clear assertion functions
   - Generates helpful failure messages
   - Standard across BATS community

2. **Assert Intent, Not Implementation**:

   - Test behavior, not internal details
   - Validate user-facing results
   - Focus on observable outcomes

3. **Provide Context**:
   - Use descriptive test names
   - Add custom error messages when needed
   - Make failures easy to diagnose

## Installing bats-assert

```bash
# Clone into test helper directory
git clone https://github.com/bats-core/bats-assert \
    tests/bats/test_helper/bats-assert

git clone https://github.com/bats-core/bats-support \
    tests/bats/test_helper/bats-support

# Load in test files
load 'test_helper/bats-support/load'
load 'test_helper/bats-assert/load'
```

## Exit Code Assertions

### Basic Success/Failure

```bash
@test "script succeeds with valid input" {
    run "$SCRIPT_PATH" --input valid.txt

    # Assert command succeeded (exit code 0)
    assert_success
}

@test "script fails with invalid input" {
    run "$SCRIPT_PATH" --input invalid.txt

    # Assert command failed (exit code non-zero)
    assert_failure
}
```

### Specific Exit Codes

```bash
@test "script exits with code 2 for usage errors" {
    run "$SCRIPT_PATH" --invalid-flag

    # Assert specific exit code
    assert_failure 2
}

@test "script exits with code 1 for runtime errors" {
    run "$SCRIPT_PATH" --missing-file /nonexistent

    assert_failure 1
}
```

### Without bats-assert (Built-in)

```bash
@test "script succeeds - built-in assertion" {
    run "$SCRIPT_PATH"

    # Check exit code directly
    [ "$status" -eq 0 ]
}

@test "script fails - built-in assertion" {
    run "$SCRIPT_PATH" --invalid

    [ "$status" -ne 0 ]
}
```

## Output Assertions

### Exact Match

```bash
@test "script outputs exact message" {
    run printf '%s\n' "Hello, World!"

    # Assert output matches exactly
    assert_output "Hello, World!"
}
```

### Partial Match (Substring)

```bash
@test "script output contains success message" {
    run "$SCRIPT_PATH" --process data.txt

    # Assert output contains substring
    assert_output --partial "Successfully processed"
    assert_output --partial "data.txt"
}
```

### Regular Expression Match

```bash
@test "script outputs timestamp format" {
    run "$SCRIPT_PATH" --timestamp

    # Assert output matches regex pattern
    assert_output --regexp '^[0-9]{4}-[0-9]{2}-[0-9]{2}'
}

@test "script outputs valid UUID" {
    run "$SCRIPT_PATH" --generate-id

    assert_output --regexp '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
}
```

### Multi-line Output

```bash
@test "script outputs multiple lines" {
    run "$SCRIPT_PATH" --list

    # Check each line contains expected text
    assert_output --partial "Item 1"
    assert_output --partial "Item 2"
    assert_output --partial "Item 3"
}
```

## Line-Specific Assertions

### Assert Specific Line

```bash
@test "script outputs header on first line" {
    run "$SCRIPT_PATH" --format table

    # Assert first line (index 0)
    assert_line --index 0 "NAME    STATUS    AGE"
}

@test "script outputs expected line anywhere" {
    run "$SCRIPT_PATH" --list

    # Assert line exists somewhere in output
    assert_line "Total: 42 items"
}
```

### Partial Line Match

```bash
@test "script output includes error line" {
    run "$SCRIPT_PATH" --invalid

    # Assert any line contains substring
    assert_line --partial "ERROR:"
    assert_line --partial "invalid"
}
```

### Regular Expression on Lines

```bash
@test "script outputs lines matching pattern" {
    run "$SCRIPT_PATH" --verbose

    # Assert at least one line matches pattern
    assert_line --regexp '^DEBUG: .*'
}
```

### Negative Assertions

```bash
@test "script output does not contain sensitive data" {
    run "$SCRIPT_PATH" --sanitize

    # Assert no line contains pattern
    refute_line --partial "password"
    refute_line --partial "secret"
    refute_line --partial "token"
}
```

## File and Directory Assertions

### File Existence

```bash
@test "script creates output file" {
    run "$SCRIPT_PATH" --output "$TEST_DIR/result.txt"

    assert_success

    # Assert file exists
    assert [ -f "$TEST_DIR/result.txt" ]
}

@test "script removes temporary file" {
    local temp_file="$TEST_DIR/temp.txt"
    touch "$temp_file"

    run "$SCRIPT_PATH" --cleanup

    # Assert file does not exist
    assert [ ! -f "$temp_file" ]
}
```

### Directory Existence

```bash
@test "script creates output directory" {
    run "$SCRIPT_PATH" --init "$TEST_DIR/output"

    assert_success
    assert [ -d "$TEST_DIR/output" ]
}
```

### File Content

```bash
@test "script writes expected content to file" {
    local output_file="$TEST_DIR/output.txt"

    run "$SCRIPT_PATH" --write "$output_file"

    assert_success
    assert [ -f "$output_file" ]

    # Assert file contains expected content
    run cat "$output_file"
    assert_output --partial "expected content"
}
```

### File Properties

```bash
@test "script creates executable file" {
    local script_file="$TEST_DIR/script.sh"

    run "$SCRIPT_PATH" --generate "$script_file"

    assert_success
    assert [ -x "$script_file" ]
}

@test "script creates non-empty file" {
    local output_file="$TEST_DIR/output.txt"

    run "$SCRIPT_PATH" --write "$output_file"

    assert_success
    assert [ -s "$output_file" ]  # -s checks file is not empty
}
```

## Variable and Value Assertions

### Equal Assertions

```bash
@test "script sets expected variable value" {
    run "$SCRIPT_PATH" --get-count

    assert_success

    # Parse count from output
    local count
    count=$(printf '%s\n' "$output" | grep -oE '[0-9]+')

    # Assert equal
    assert_equal "$count" "42"
}
```

### Not Equal Assertions

```bash
@test "script generates unique ID" {
    run "$SCRIPT_PATH" --generate-id
    local id1="$output"

    run "$SCRIPT_PATH" --generate-id
    local id2="$output"

    # Assert not equal
    refute_equal "$id1" "$id2"
}
```

## Custom Assertions

### Helper Function Assertions

```bash
# Custom assertion for JSON validity
assert_valid_json() {
    local file="$1"

    if ! jq empty "$file" 2>/dev/null; then
        echo "File is not valid JSON: $file"
        return 1
    fi
}

# Custom assertion for file count
assert_file_count() {
    local directory="$1"
    local expected_count="$2"

    local actual_count
    actual_count=$(find "$directory" -type f | wc -l | tr -d ' ')

    if [ "$actual_count" -ne "$expected_count" ]; then
        echo "Expected $expected_count files, found $actual_count"
        return 1
    fi
}

@test "script creates valid JSON backup" {
    run "$SCRIPT_PATH" --backup "$BACKUP_DIR"

    assert_success

    local backup_file
    backup_file=$(find "$BACKUP_DIR" -name "*.json" | head -1)

    assert_valid_json "$backup_file"
}

@test "script creates expected number of backups" {
    # Create multiple backups
    for i in {1..3}; do
        run "$SCRIPT_PATH" --backup "$BACKUP_DIR"
        assert_success
    done

    assert_file_count "$BACKUP_DIR" 3
}
```

## Docker State Assertions

### Container Assertions

```bash
@test "script starts container" {
    run "$SCRIPT_PATH" --start "$CONTAINER_NAME"

    assert_success

    # Assert container is running
    run docker ps --filter "name=$CONTAINER_NAME" --format "{{.Names}}"
    assert_output "$CONTAINER_NAME"
}

@test "container has expected status" {
    run "$SCRIPT_PATH" --start "$CONTAINER_NAME"

    assert_success

    # Assert container status
    local status
    status=$(docker inspect -f '{{.State.Status}}' "$CONTAINER_NAME")
    assert_equal "$status" "running"
}
```

### Volume Assertions

```bash
@test "script creates Docker volume" {
    run "$SCRIPT_PATH" --create-volume "$VOLUME_NAME"

    assert_success

    # Assert volume exists
    run docker volume inspect "$VOLUME_NAME"
    assert_success
}

@test "volume contains expected data" {
    run "$SCRIPT_PATH" --populate-volume "$VOLUME_NAME"

    assert_success

    # Assert data exists in volume
    run docker run --rm -v "${VOLUME_NAME}:/data" alpine test -f /data/memory.json
    assert_success
}
```

## JSON and YAML Assertions

### JSON Structure

```bash
@test "script outputs valid JSON" {
    run "$SCRIPT_PATH" --format json

    assert_success

    # Validate JSON
    run printf '%s\n' "$output" | jq empty
    assert_success
}

@test "JSON contains expected fields" {
    local json_file="$TEST_DIR/output.json"

    run "$SCRIPT_PATH" --output "$json_file"

    assert_success

    # Check for required fields
    run jq -r '.name' "$json_file"
    assert_success
    refute_output "null"

    run jq -r '.id' "$json_file"
    assert_success
    refute_output "null"
}
```

### YAML Structure

```bash
@test "script creates valid YAML config" {
    local config_file="$TEST_DIR/config.yaml"

    run "$SCRIPT_PATH" --generate-config "$config_file"

    assert_success

    # Validate YAML
    run yq eval . "$config_file"
    assert_success
}

@test "YAML contains expected values" {
    local config_file="$TEST_DIR/config.yaml"

    run "$SCRIPT_PATH" --generate-config "$config_file"

    assert_success

    # Check YAML values
    run yq eval '.settings.enabled' "$config_file"
    assert_output "true"

    run yq eval '.settings.timeout' "$config_file"
    assert_output "30"
}
```

## Error Message Assertions

### Helpful Error Messages

```bash
@test "script provides helpful error for missing file" {
    run "$SCRIPT_PATH" --input /nonexistent/file.txt

    assert_failure

    # Assert error message is helpful
    assert_output --partial "file not found"
    assert_output --partial "/nonexistent/file.txt"

    # Assert usage help is shown
    assert_output --partial "Usage:"
}
```

### Error Format

```bash
@test "script outputs errors to stderr" {
    run "$SCRIPT_PATH" --invalid-option

    assert_failure

    # BATS combines stdout and stderr in $output
    # To test separately, capture them differently
    local stderr
    stderr=$("$SCRIPT_PATH" --invalid-option 2>&1 >/dev/null)

    [[ "$stderr" == *"ERROR:"* ]]
}
```

## Complex Assertion Patterns

### Chained Assertions

```bash
@test "script performs complete workflow" {
    # Multiple assertions for complex behavior
    run "$SCRIPT_PATH" --initialize "$TEST_DIR"

    # 1. Command succeeds
    assert_success

    # 2. Output confirms initialization
    assert_output --partial "Initialized"

    # 3. Config file created
    assert [ -f "$TEST_DIR/config.yaml" ]

    # 4. Data directory created
    assert [ -d "$TEST_DIR/data" ]

    # 5. Config has expected content
    run yq eval '.version' "$TEST_DIR/config.yaml"
    assert_output "1.0"
}
```

### Conditional Assertions

```bash
@test "script behavior depends on environment" {
    if [ -n "${CI:-}" ]; then
        # In CI environment
        run "$SCRIPT_PATH" --deploy

        assert_success
        assert_output --partial "Deploying to production"
    else
        # In local environment
        run "$SCRIPT_PATH" --deploy

        assert_success
        assert_output --partial "Deploying to development"
    fi
}
```

## Key Patterns

1. **Use bats-assert**: Provides clear, descriptive assertions
2. **Load helpers**: Always load bats-support and bats-assert
3. **Descriptive failures**: Assertion failures should be self-explanatory
4. **Test behavior**: Assert user-visible results, not implementation
5. **Multiple assertions**: Complex tests can have multiple assertions
6. **Custom helpers**: Create custom assertions for domain-specific validation

## Common Pitfalls

- **Using `echo` instead of `printf`**: Not POSIX compliant, behaves differently across platforms
- **Not loading bats-assert**: Missing helpful assertion functions
- **Testing too much**: Each test should validate one behavior
- **Vague assertions**: Use `--partial` or `--regexp` for specific checks
- **Ignoring stderr**: Remember to test error messages
- **Not testing edge cases**: Empty files, special characters, etc.
- **Forgetting `run`**: Must use `run` to capture output
- **Assuming order**: Don't assume output order unless specified
- **Using grep -P**: Not portable (BSD doesn't support), use `-E` instead
- **Terse one-liners**: Prefer readable code with extracted variables

## Complete Example

```bash
#!/usr/bin/env bats

load 'test_helper/bats-support/load'
load 'test_helper/bats-assert/load'

SCRIPT_PATH="${BATS_TEST_DIRNAME}/../../scripts/backup-memory-mcp.sh"

setup() {
    export TEST_DIR="${BATS_TEST_TMPDIR}/test-$$"
    export BACKUP_DIR="$TEST_DIR/backups"
    export VOLUME_NAME="test-volume-$$"

    mkdir -p "$BACKUP_DIR"
    docker volume create "$VOLUME_NAME"

    printf '%s\n' '{"test": "data"}' | \
        docker run --rm -i -v "${VOLUME_NAME}:/data" \
        alpine sh -c 'cat > /data/memory.json'
}

teardown() {
    docker volume rm "$VOLUME_NAME" 2>/dev/null || true
    rm -rf "$TEST_DIR"
}

@test "backup creates valid timestamped JSON file" {
    # Act
    run "$SCRIPT_PATH"

    # Assert: Command succeeds
    assert_success

    # Assert: Success message displayed
    assert_output --partial "Backup created"

    # Assert: Backup file exists with timestamp format
    local backup_files
    backup_files=$(find "$BACKUP_DIR" -name "memory-*.json")
    assert [ -n "$backup_files" ]

    # Assert: Filename matches timestamp pattern
    assert_line --regexp 'memory-[0-9]{8}-[0-9]{6}\.json'

    # Assert: Backup is valid JSON
    local backup_file
    backup_file=$(find "$BACKUP_DIR" -name "*.json" | head -1)
    run jq empty "$backup_file"
    assert_success

    # Assert: Backup contains expected data
    run jq -r '.test' "$backup_file"
    assert_output "data"
}

@test "backup fails gracefully with invalid JSON" {
    # Arrange: Put invalid JSON in volume
    printf '%s\n' 'invalid json' | \
        docker run --rm -i -v "${VOLUME_NAME}:/data" \
        alpine sh -c 'cat > /data/memory.json'

    # Act
    run "$SCRIPT_PATH"

    # Assert: Command fails
    assert_failure

    # Assert: Clear error message
    assert_output --partial "not valid JSON"

    # Assert: No backup file created
    local backup_count
    backup_count=$(find "$BACKUP_DIR" -name "*.json" | wc -l)
    assert_equal "$backup_count" "0"
}
```

## Assertion Reference

| Assertion                          | Purpose                     |
| ---------------------------------- | --------------------------- |
| `assert_success`                   | Exit code is 0              |
| `assert_failure`                   | Exit code is non-zero       |
| `assert_failure N`                 | Exit code is N              |
| `assert_output "text"`             | Output exactly matches      |
| `assert_output --partial "text"`   | Output contains substring   |
| `assert_output --regexp "pattern"` | Output matches regex        |
| `assert_line "text"`               | Any line exactly matches    |
| `assert_line --index N "text"`     | Line N exactly matches      |
| `assert_line --partial "text"`     | Any line contains substring |
| `assert_line --regexp "pattern"`   | Any line matches regex      |
| `refute_output`                    | No output produced          |
| `refute_line`                      | No line matches             |
| `assert_equal "$a" "$b"`           | Values are equal            |
| `refute_equal "$a" "$b"`           | Values are not equal        |
| `assert [ condition ]`             | Shell test condition true   |
