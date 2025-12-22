---
entity_name: BATS Docker Testing Pattern
entity_type: bats-testing-pattern
language: bash
domain: testing
description: Docker container testing patterns with BATS including setup, teardown, container lifecycle management, and integration testing
tags:
  - BATS
  - Docker
  - containers
  - integration-testing
  - bash
---

# BATS Docker Testing Pattern

## Philosophy

When testing shell scripts that interact with Docker (volumes, containers, images), tests must properly manage Docker resources, ensure cleanup, and provide complete isolation between tests. Always use unique resource names to prevent conflicts.

## Quality Standards

All BATS Docker tests MUST:
- **Pass shellcheck** with the project's `.shellcheckrc` configuration
- **Be POSIX compliant** - use `printf` not `echo`, avoid bash-isms
- **Be readable** - extract variables, avoid terse one-liners
- **Assume docker available** - no need to check for Docker, it's assumed installed

## Core Approach

1. **Unique Resource Names**:
   - Use `$$` (process ID) suffix for uniqueness
   - Prevents conflicts between parallel test runs
   - Example: `test-volume-$$`, `test-container-$$`

2. **Resource Lifecycle**:
   - Create resources in `setup()` or test body
   - Always clean up in `teardown()`
   - Verify cleanup completed successfully

3. **Real Docker Resources**:
   - Use actual Docker volumes, containers, images
   - Don't mock Docker - test real interactions
   - Verify state before and after operations

## Example: Testing Script with Docker Volumes

```bash
#!/usr/bin/env bats

load 'test_helper/bats-support/load'
load 'test_helper/bats-assert/load'

# Script that interacts with Docker volumes
SCRIPT_PATH="${BATS_TEST_DIRNAME}/../../scripts/backup-volume.sh"

setup() {
    # Use unique volume name with process ID
    export VOLUME_NAME="test-volume-$$"
    export TEST_DIR="${BATS_TEST_TMPDIR}/test-$$"
    export BACKUP_DIR="$TEST_DIR/backups"

    mkdir -p "$BACKUP_DIR"

    # Create test volume
    docker volume create "$VOLUME_NAME"

    # Populate volume with test data
    printf '%s\n' '{"test": "data"}' | \
        docker run --rm -i -v "${VOLUME_NAME}:/data" \
        alpine sh -c 'cat > /data/test.json'
}

teardown() {
    # Clean up Docker volume (suppress errors if already removed)
    docker volume rm "$VOLUME_NAME" 2>/dev/null || true

    # Clean up test directory
    rm -rf "$TEST_DIR"
}

@test "backup script creates backup from volume" {
    # Act: Run backup script
    run "$SCRIPT_PATH" --volume "$VOLUME_NAME" --output "$BACKUP_DIR"

    # Assert: Backup should succeed
    assert_success
    assert_output --partial "Backup created"

    # Verify: Backup file exists
    local backup_file
    backup_file=$(find "$BACKUP_DIR" -name "*.json" | head -1)
    assert [ -f "$backup_file" ]

    # Verify: Backup contains expected data
    assert grep -q '"test"' "$backup_file"
}

@test "backup script fails with non-existent volume" {
    # Act: Try to backup non-existent volume
    run "$SCRIPT_PATH" --volume "nonexistent-volume" --output "$BACKUP_DIR"

    # Assert: Should fail with appropriate error
    assert_failure
    assert_output --partial "volume not found"
}
```

## Example: Testing Script with Docker Containers

```bash
#!/usr/bin/env bats

load 'test_helper/bats-support/load'
load 'test_helper/bats-assert/load'

SCRIPT_PATH="${BATS_TEST_DIRNAME}/../../scripts/deploy-service.sh"

setup() {
    export CONTAINER_NAME="test-service-$$"
    export TEST_DIR="${BATS_TEST_TMPDIR}/test-$$"
    mkdir -p "$TEST_DIR"
}

teardown() {
    # Stop and remove container
    docker stop "$CONTAINER_NAME" 2>/dev/null || true
    docker rm "$CONTAINER_NAME" 2>/dev/null || true

    # Clean up test directory
    rm -rf "$TEST_DIR"
}

@test "deploy script starts container successfully" {
    # Act: Run deployment script
    run "$SCRIPT_PATH" --name "$CONTAINER_NAME" --port 8080

    # Assert: Should succeed
    assert_success

    # Verify: Container is running
    run docker ps --filter "name=$CONTAINER_NAME" --format "{{.Names}}"
    assert_output "$CONTAINER_NAME"

    # Verify: Container is healthy
    local status
    status=$(docker inspect -f '{{.State.Status}}' "$CONTAINER_NAME")
    assert_equal "$status" "running"
}

@test "deploy script handles port already in use" {
    # Arrange: Start container on port 8080
    docker run -d --name "$CONTAINER_NAME" -p 8080:80 nginx:alpine

    # Act: Try to deploy another container on same port
    run "$SCRIPT_PATH" --name "test-service-2-$$" --port 8080

    # Assert: Should fail with port conflict error
    assert_failure
    assert_output --partial "port.*already.*use"

    # Cleanup: Remove first container
    docker stop "$CONTAINER_NAME" 2>/dev/null || true
    docker rm "$CONTAINER_NAME" 2>/dev/null || true
}
```

## Example: Testing Script with Docker Images

```bash
#!/usr/bin/env bats

load 'test_helper/bats-support/load'
load 'test_helper/bats-assert/load'

SCRIPT_PATH="${BATS_TEST_DIRNAME}/../../scripts/build-image.sh"

setup() {
    export IMAGE_NAME="test-image-$$"
    export IMAGE_TAG="test"
    export TEST_DIR="${BATS_TEST_TMPDIR}/test-$$"
    mkdir -p "$TEST_DIR"

    # Create test Dockerfile
    cat > "$TEST_DIR/Dockerfile" <<'EOF'
FROM alpine:latest
RUN echo "test image"
EOF
}

teardown() {
    # Remove test image
    docker rmi "${IMAGE_NAME}:${IMAGE_TAG}" 2>/dev/null || true

    # Clean up test directory
    rm -rf "$TEST_DIR"
}

@test "build script creates Docker image" {
    # Act: Build image
    run "$SCRIPT_PATH" \
        --name "$IMAGE_NAME" \
        --tag "$IMAGE_TAG" \
        --dockerfile "$TEST_DIR/Dockerfile" \
        --context "$TEST_DIR"

    # Assert: Should succeed
    assert_success

    # Verify: Image exists
    run docker images "$IMAGE_NAME" --format "{{.Repository}}:{{.Tag}}"
    assert_output --partial "${IMAGE_NAME}:${IMAGE_TAG}"
}
```

## Helper Functions for Docker Testing

```bash
# Helper: Check if Docker volume exists
volume_exists() {
    local volume_name="$1"
    docker volume inspect "$volume_name" >/dev/null 2>&1
}

# Helper: Get Docker volume size
get_volume_size() {
    local volume_name="$1"
    docker run --rm -v "${volume_name}:/data" \
        alpine du -sh /data | cut -f1
}

# Helper: Check if container is running
container_running() {
    local container_name="$1"
    local status
    status=$(docker inspect -f '{{.State.Running}}' "$container_name" 2>/dev/null)
    [ "$status" = "true" ]
}

# Helper: Wait for container to be healthy
wait_for_container_healthy() {
    local container_name="$1"
    local timeout="${2:-30}"
    local elapsed=0

    while [ $elapsed -lt $timeout ]; do
        local health
        health=$(docker inspect -f '{{.State.Health.Status}}' "$container_name" 2>/dev/null)

        if [ "$health" = "healthy" ]; then
            return 0
        fi

        sleep 1
        elapsed=$((elapsed + 1))
    done

    return 1
}

# Helper: Get container logs
get_container_logs() {
    local container_name="$1"
    docker logs "$container_name" 2>&1
}

# Helper: Clean up all test containers
cleanup_test_containers() {
    local pattern="${1:-test-.*-$$}"
    docker ps -aq --filter "name=$pattern" | xargs -r docker rm -f 2>/dev/null || true
}

# Helper: Clean up all test volumes
cleanup_test_volumes() {
    local pattern="${1:-test-.*-$$}"
    docker volume ls -q --filter "name=$pattern" | xargs -r docker volume rm 2>/dev/null || true
}

# Example using helpers
@test "deploy and wait for healthy container" {
    # Act: Deploy container
    run "$SCRIPT_PATH" --name "$CONTAINER_NAME"
    assert_success

    # Wait for container to be healthy (with 30s timeout)
    run wait_for_container_healthy "$CONTAINER_NAME" 30
    assert_success

    # Verify container is running
    assert container_running "$CONTAINER_NAME"
}
```

## Docker Compose Testing

```bash
#!/usr/bin/env bats

load 'test_helper/bats-support/load'
load 'test_helper/bats-assert/load'

SCRIPT_PATH="${BATS_TEST_DIRNAME}/../../scripts/deploy-stack.sh"
COMPOSE_FILE="${BATS_TEST_DIRNAME}/fixtures/docker-compose.test.yaml"

setup() {
    export COMPOSE_PROJECT_NAME="test-project-$$"
    export TEST_DIR="${BATS_TEST_TMPDIR}/test-$$"
    mkdir -p "$TEST_DIR"
}

teardown() {
    # Bring down Docker Compose stack
    docker-compose -f "$COMPOSE_FILE" -p "$COMPOSE_PROJECT_NAME" down -v 2>/dev/null || true

    # Clean up test directory
    rm -rf "$TEST_DIR"
}

@test "deploy script starts Docker Compose stack" {
    # Act: Deploy stack
    run "$SCRIPT_PATH" --compose-file "$COMPOSE_FILE" --project "$COMPOSE_PROJECT_NAME"

    # Assert: Should succeed
    assert_success

    # Verify: Services are running
    run docker-compose -f "$COMPOSE_FILE" -p "$COMPOSE_PROJECT_NAME" ps --services --filter "status=running"
    assert_output --partial "web"
    assert_output --partial "db"
}

@test "deploy script handles compose file errors" {
    # Create invalid compose file
    local invalid_compose="$TEST_DIR/invalid-compose.yaml"
    echo "invalid: yaml: content" > "$invalid_compose"

    # Act: Try to deploy with invalid file
    run "$SCRIPT_PATH" --compose-file "$invalid_compose" --project "$COMPOSE_PROJECT_NAME"

    # Assert: Should fail
    assert_failure
}
```

## Testing Docker Volume Data Integrity

```bash
@test "restore script preserves data integrity" {
    # Arrange: Create volume with known data
    local source_volume="test-source-$$"
    docker volume create "$source_volume"

    local test_data='{"id": 123, "name": "test", "checksum": "abc123"}'
    echo "$test_data" | \
        docker run --rm -i -v "${source_volume}:/data" \
        alpine sh -c 'cat > /data/data.json'

    # Create backup
    local backup_file="$BACKUP_DIR/backup.json"
    docker run --rm -v "${source_volume}:/data" -v "${BACKUP_DIR}:/backup" \
        alpine cp /data/data.json /backup/backup.json

    # Act: Restore to new volume
    run "$SCRIPT_PATH" --backup "$backup_file" --volume "$VOLUME_NAME"
    assert_success

    # Assert: Data matches original
    local restored_data
    restored_data=$(docker run --rm -v "${VOLUME_NAME}:/data" alpine cat /data/data.json)
    assert_equal "$restored_data" "$test_data"

    # Cleanup
    docker volume rm "$source_volume" 2>/dev/null || true
}
```

## Prerequisites

**Docker is assumed to be available.** Tests should not check for Docker availability or skip if it's missing. Docker is considered a required part of the development environment.

## Key Patterns

1. **Unique Names**: Always use `$$` suffix for Docker resources
2. **Guaranteed Cleanup**: Use `teardown()` with `|| true` for cleanup commands
3. **Real Resources**: Test with actual Docker resources, not mocks
4. **State Verification**: Check Docker state before and after operations
5. **Helper Functions**: Create reusable helpers for common Docker operations
6. **Timeout Handling**: Use timeouts when waiting for containers/health checks

## Common Pitfalls

- **Using `echo` instead of `printf`**: Not POSIX compliant, behaves differently across platforms
- **Checking Docker availability**: Docker is assumed available, don't check or skip
- **Not using unique names**: Causes conflicts between parallel tests
- **Forgetting to clean up**: Leaves Docker resources consuming system resources
- **Not suppressing errors in cleanup**: Cleanup failures cause test failures
- **Not waiting for containers**: Tests pass/fail inconsistently due to timing
- **Hardcoding image tags**: Use variables for flexibility
- **Not verifying cleanup**: Resources leak over time
- **Terse one-liners**: Extract variables for readability

## Complete Example

```bash
#!/usr/bin/env bats

load 'test_helper/bats-support/load'
load 'test_helper/bats-assert/load'

SCRIPT_PATH="${BATS_TEST_DIRNAME}/../../scripts/backup-memory-mcp.sh"

setup() {
    # Check Docker prerequisites
    if ! command -v docker >/dev/null 2>&1; then
        skip "Docker not installed"
    fi

    # Set up test environment
    export VOLUME_NAME="test-memory-$$"
    export TEST_DIR="${BATS_TEST_TMPDIR}/test-$$"
    export BACKUP_DIR="$TEST_DIR/backups"

    mkdir -p "$BACKUP_DIR"

    # Create and populate test volume
    docker volume create "$VOLUME_NAME"
    printf '%s\n' '{"type":"entity","name":"test"}' | \
        docker run --rm -i -v "${VOLUME_NAME}:/data" \
        alpine sh -c 'cat > /data/memory.json'
}

teardown() {
    # Clean up Docker resources
    docker volume rm "$VOLUME_NAME" 2>/dev/null || true

    # Clean up test directory
    rm -rf "$TEST_DIR"
}

@test "backup script creates timestamped backup" {
    run "$SCRIPT_PATH"

    assert_success
    assert_output --partial "Backup created"

    # Verify backup file exists
    local backup_count
    backup_count=$(find "$BACKUP_DIR" -name "memory-*.json" | wc -l)
    assert [ "$backup_count" -eq 1 ]
}

@test "backup script validates JSON" {
    # Arrange: Put invalid JSON in volume
    printf '%s\n' 'invalid json' | \
        docker run --rm -i -v "${VOLUME_NAME}:/data" \
        alpine sh -c 'cat > /data/memory.json'

    # Act
    run "$SCRIPT_PATH"

    # Assert
    assert_failure
    assert_output --partial "not valid JSON"
}
```
