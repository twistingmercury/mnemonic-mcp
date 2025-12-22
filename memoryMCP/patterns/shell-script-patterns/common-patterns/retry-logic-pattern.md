---
entity_name: Retry Logic Pattern
entity_type: shell-script-pattern
language: bash
domain: shell-scripting
description: Implement retry mechanisms with configurable attempts and backoff strategies for unreliable operations
tags:
  - shell
  - bash
  - retry
  - resilience
  - error-handling
  - backoff
---

# Retry Logic Pattern

## Overview

Retry logic adds resilience to operations that may fail transiently (network requests, remote services, temporary resource unavailability). This pattern implements configurable retry attempts with backoff strategies to avoid overwhelming failing systems.

## Pattern

### Basic Retry with Attempts

```bash
retry_command() {
    local max_attempts="${1}"
    shift
    local attempt=1

    while [ "${attempt}" -le "${max_attempts}" ]; do
        # Try to execute the command
        if "$@"; then
            return 0
        fi

        printf "Attempt %s/%s failed, retrying...\n" "${attempt}" "${max_attempts}"
        attempt=$((attempt + 1))
        sleep 2
    done

    printf "ERROR: Command failed after %s attempts\n" "${max_attempts}" >&2
    return 1
}
```

### Usage

```bash
# Retry a command up to 3 times
if retry_command 3 curl -f "https://api.example.com/data"; then
    printf "Command succeeded\n"
else
    printf "Command failed after all retries\n"
fi
```

## Pattern with Exponential Backoff

```bash
retry_with_backoff() {
    local max_attempts="${1}"
    local initial_delay="${2:-1}"
    shift 2
    local attempt=1
    local delay="${initial_delay}"

    while [ "${attempt}" -le "${max_attempts}" ]; do
        # Try to execute the command
        if "$@"; then
            return 0
        fi

        printf "Attempt %s/%s failed\n" "${attempt}" "${max_attempts}"

        if [ "${attempt}" -lt "${max_attempts}" ]; then
            printf "Waiting %s seconds before retry...\n" "${delay}"
            sleep "${delay}"

            # Exponential backoff: double delay each time
            delay=$((delay * 2))
        fi

        attempt=$((attempt + 1))
    done

    printf "ERROR: Command failed after %s attempts\n" "${max_attempts}" >&2
    return 1
}
```

## Complete Example with Retry Logic

```bash
#!/usr/bin/env bash

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Configuration
MAX_RETRIES="${MAX_RETRIES:-3}"
RETRY_DELAY="${RETRY_DELAY:-2}"
API_ENDPOINT="${API_ENDPOINT}"

retry_command() {
    local max_attempts="${1}"
    local delay="${2}"
    shift 2
    local attempt=1

    while [ "${attempt}" -le "${max_attempts}" ]; do
        printf "Attempt %s/%s: " "${attempt}" "${max_attempts}"

        if "$@"; then
            printf "SUCCESS\n"
            return 0
        fi

        printf "FAILED\n"

        if [ "${attempt}" -lt "${max_attempts}" ]; then
            printf "Waiting %s seconds before retry...\n" "${delay}"
            sleep "${delay}"
        fi

        attempt=$((attempt + 1))
    done

    printf "ERROR: All %s attempts failed\n" "${max_attempts}" >&2
    return 1
}

fetch_data() {
    local endpoint="${1}"

    if [ -z "${endpoint}" ]; then
        printf "ERROR: Endpoint required\n" >&2
        return 1
    fi

    # Use retry logic for network operation
    if retry_command "${MAX_RETRIES}" "${RETRY_DELAY}" \
        curl -f -s -o /tmp/data.json "${endpoint}"; then
        printf "Data fetched successfully\n"
        return 0
    fi

    printf "ERROR: Failed to fetch data\n" >&2
    return 1
}

main() {
    if [ -z "${API_ENDPOINT}" ]; then
        printf "ERROR: API_ENDPOINT is required\n" >&2
        return 1
    fi

    if ! fetch_data "${API_ENDPOINT}"; then
        return 1
    fi

    printf "Processing complete\n"
    return 0
}

main "$@"
```

## Advanced Pattern: Retry with Jitter

Add randomness to backoff delay to prevent thundering herd:

```bash
retry_with_jitter() {
    local max_attempts="${1}"
    local base_delay="${2:-2}"
    shift 2
    local attempt=1

    while [ "${attempt}" -le "${max_attempts}" ]; do
        if "$@"; then
            return 0
        fi

        if [ "${attempt}" -lt "${max_attempts}" ]; then
            # Calculate delay with exponential backoff
            local backoff_delay=$((base_delay * (2 ** (attempt - 1))))

            # Add jitter (random 0-50% of delay)
            local jitter=$((RANDOM % (backoff_delay / 2)))
            local total_delay=$((backoff_delay + jitter))

            printf "Attempt %s/%s failed. Waiting %s seconds...\n" \
                   "${attempt}" "${max_attempts}" "${total_delay}"
            sleep "${total_delay}"
        fi

        attempt=$((attempt + 1))
    done

    printf "ERROR: Command failed after %s attempts\n" "${max_attempts}" >&2
    return 1
}
```

## Pattern: Retry with Conditional Retry

Only retry on specific error conditions:

```bash
should_retry() {
    local exit_code="${1}"

    # Retry on network errors (curl exit codes)
    case "${exit_code}" in
        6|7|28|52|56)  # Connection errors, timeouts
            return 0
            ;;
        *)
            return 1
            ;;
    esac
}

retry_smart() {
    local max_attempts="${1}"
    local delay="${2}"
    shift 2
    local attempt=1

    while [ "${attempt}" -le "${max_attempts}" ]; do
        # Capture exit code
        local exit_code=0
        "$@" || exit_code=$?

        if [ "${exit_code}" -eq 0 ]; then
            return 0
        fi

        # Check if we should retry this error
        if ! should_retry "${exit_code}"; then
            printf "ERROR: Non-retryable error (exit code: %s)\n" "${exit_code}" >&2
            return "${exit_code}"
        fi

        printf "Retryable error (exit code: %s). Attempt %s/%s\n" \
               "${exit_code}" "${attempt}" "${max_attempts}"

        if [ "${attempt}" -lt "${max_attempts}" ]; then
            sleep "${delay}"
        fi

        attempt=$((attempt + 1))
    done

    printf "ERROR: Command failed after %s retries\n" "${max_attempts}" >&2
    return 1
}
```

## Pattern: Retry with Progress Callback

```bash
retry_with_progress() {
    local max_attempts="${1}"
    local delay="${2}"
    local progress_fn="${3:-printf}"
    shift 3
    local attempt=1

    while [ "${attempt}" -le "${max_attempts}" ]; do
        # Notify progress
        "${progress_fn}" "Attempting operation (${attempt}/${max_attempts})..."

        if "$@"; then
            "${progress_fn}" "Operation succeeded on attempt ${attempt}"
            return 0
        fi

        "${progress_fn}" "Attempt ${attempt} failed"

        if [ "${attempt}" -lt "${max_attempts}" ]; then
            "${progress_fn}" "Waiting ${delay} seconds before retry..."
            sleep "${delay}"
        fi

        attempt=$((attempt + 1))
    done

    "${progress_fn}" "Operation failed after ${max_attempts} attempts"
    return 1
}

# Custom progress function
log_progress() {
    local message="${1}"
    printf "[%s] %s\n" "$(date +%H:%M:%S)" "${message}"
}

# Usage
retry_with_progress 3 2 log_progress curl -f "https://api.example.com/data"
```

## Complete Production Example

```bash
#!/usr/bin/env bash

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
MAX_RETRIES="${MAX_RETRIES:-5}"
INITIAL_DELAY="${INITIAL_DELAY:-1}"

retry_with_backoff() {
    local max_attempts="${1}"
    local initial_delay="${2}"
    shift 2
    local attempt=1
    local delay="${initial_delay}"

    while [ "${attempt}" -le "${max_attempts}" ]; do
        local exit_code=0

        printf "[Attempt %s/%s] Running: %s\n" "${attempt}" "${max_attempts}" "$*"

        # Execute command and capture exit code
        "$@" || exit_code=$?

        if [ "${exit_code}" -eq 0 ]; then
            printf "[SUCCESS] Command succeeded on attempt %s\n" "${attempt}"
            return 0
        fi

        printf "[FAILED] Command failed with exit code %s\n" "${exit_code}"

        if [ "${attempt}" -lt "${max_attempts}" ]; then
            printf "[RETRY] Waiting %s seconds before retry...\n" "${delay}"
            sleep "${delay}"

            # Exponential backoff with max cap
            delay=$((delay * 2))
            if [ "${delay}" -gt 60 ]; then
                delay=60
            fi
        fi

        attempt=$((attempt + 1))
    done

    printf "[ERROR] Command failed after %s attempts\n" "${max_attempts}" >&2
    return 1
}

fetch_remote_file() {
    local url="${1}"
    local output="${2}"

    curl -f -s -S -o "${output}" "${url}"
}

main() {
    local remote_url="${REMOTE_URL}"
    local output_file="${OUTPUT_FILE:-/tmp/download.txt}"

    if [ -z "${remote_url}" ]; then
        printf "ERROR: REMOTE_URL is required\n" >&2
        return 1
    fi

    printf "Downloading file with retry logic...\n"

    if retry_with_backoff "${MAX_RETRIES}" "${INITIAL_DELAY}" \
        fetch_remote_file "${remote_url}" "${output_file}"; then
        printf "File downloaded successfully to: %s\n" "${output_file}"
        return 0
    fi

    printf "ERROR: Failed to download file\n" >&2
    return 1
}

main "$@"
```

## Best Practices

1. **Configurable attempts** - Use environment variables for max retries
2. **Implement backoff** - Don't retry immediately, add delay between attempts
3. **Exponential backoff** - Increase delay exponentially to avoid overwhelming failing systems
4. **Add jitter** - Random delays prevent thundering herd
5. **Limit max delay** - Cap exponential backoff at reasonable maximum
6. **Conditional retry** - Only retry on transient errors, not permanent failures
7. **Log attempts** - Clear visibility into retry behavior

## When to Use Retry Logic

**Good candidates:**

- Network requests (API calls, downloads)
- Remote service calls
- Database connections
- File system operations (NFS, distributed FS)
- Resource acquisition (locks, semaphores)

**Poor candidates:**

- Validation errors (will never succeed)
- Permission errors (need manual fix)
- Syntax errors (code bug)
- Missing required files (need human intervention)

## Common Pitfalls

- **Retrying non-transient errors** - Permanent failures won't be fixed by retrying
- **No backoff strategy** - Overwhelming failing systems with rapid retries
- **Too many retries** - Excessive retry count delays failure detection
- **No timeout** - Individual operations should have timeouts
- **Not logging attempts** - Hard to debug retry behavior

## Related Patterns

- [File Locking Pattern](file-locking-pattern.md) - Can use retry logic for lock acquisition
- [Never-Nester Pattern](../never-nester-pattern.md) - Guard clauses for retry validation
- [Cross-Platform Pattern](../cross-platform-pattern.md) - Retry logic works across platforms
