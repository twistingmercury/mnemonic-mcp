---
entity_name: File Locking Pattern
entity_type: shell-script-pattern
language: bash
domain: shell-scripting
description: Implement file-based locking mechanism to prevent concurrent script execution
tags:
  - shell
  - bash
  - locking
  - concurrency
  - synchronization
  - mutual-exclusion
---

# File Locking Pattern

## Overview

File locking prevents multiple instances of a script from running simultaneously, which is critical for scripts that modify shared resources. This pattern uses lock files with timeout mechanisms to ensure safe concurrent access.

## Pattern

### Basic Lock Acquisition with Timeout

```bash
acquire_lock() {
    local lock_file="${1}"
    local max_wait="${2:-30}"
    local waited=0

    # Wait for lock to be released
    while [ -f "${lock_file}" ] && [ "${waited}" -lt "${max_wait}" ]; do
        sleep 1
        waited=$((waited + 1))
    done

    # Check if we timed out
    if [ -f "${lock_file}" ]; then
        printf "ERROR: Could not acquire lock after %s seconds\n" "${max_wait}" >&2
        return 1
    fi

    # Acquire lock
    touch "${lock_file}"
    return 0
}
```

### Lock Release

```bash
release_lock() {
    local lock_file="${1}"

    if [ -f "${lock_file}" ]; then
        rm -f "${lock_file}"
    fi
}
```

### Complete Example with Lock Management

```bash
#!/usr/bin/env bash

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
LOCK_FILE="${LOCK_FILE:-/tmp/my-script.lock}"
MAX_WAIT="${MAX_WAIT:-30}"

acquire_lock() {
    local lock_file="${1}"
    local max_wait="${2}"
    local waited=0

    # Wait for existing lock to be released
    while [ -f "${lock_file}" ] && [ "${waited}" -lt "${max_wait}" ]; do
        printf "Waiting for lock... (%s/%s seconds)\n" "${waited}" "${max_wait}"
        sleep 1
        waited=$((waited + 1))
    done

    # Check if we exceeded timeout
    if [ -f "${lock_file}" ]; then
        printf "ERROR: Could not acquire lock after %s seconds\n" "${max_wait}" >&2
        printf "ERROR: Another instance may be running or lock is stale\n" >&2
        return 1
    fi

    # Create lock file
    touch "${lock_file}"

    printf "Lock acquired: %s\n" "${lock_file}"
    return 0
}

release_lock() {
    local lock_file="${1}"

    if [ -f "${lock_file}" ]; then
        rm -f "${lock_file}"
        printf "Lock released: %s\n" "${lock_file}"
    fi
}

cleanup() {
    release_lock "${LOCK_FILE}"
}

# Ensure lock is released on exit
trap cleanup EXIT

do_critical_work() {
    printf "Performing critical work...\n"
    sleep 2
    printf "Critical work complete\n"
}

main() {
    # Acquire lock before proceeding
    if ! acquire_lock "${LOCK_FILE}" "${MAX_WAIT}"; then
        printf "ERROR: Failed to acquire lock\n" >&2
        return 1
    fi

    # Do work that requires exclusive access
    do_critical_work

    # Lock will be released automatically via trap
    return 0
}

main "$@"
```

## Advanced Pattern: Lock with Process ID

Store the process ID in the lock file to identify which process holds the lock:

```bash
acquire_lock_with_pid() {
    local lock_file="${1}"
    local max_wait="${2:-30}"
    local waited=0

    while [ -f "${lock_file}" ] && [ "${waited}" -lt "${max_wait}" ]; do
        local lock_pid
        lock_pid=$(cat "${lock_file}" 2>/dev/null || printf "")

        if [ -n "${lock_pid}" ]; then
            printf "Waiting for lock held by PID %s... (%s/%s seconds)\n" \
                   "${lock_pid}" "${waited}" "${max_wait}"
        fi

        sleep 1
        waited=$((waited + 1))
    done

    if [ -f "${lock_file}" ]; then
        printf "ERROR: Could not acquire lock\n" >&2
        return 1
    fi

    # Store current process ID in lock file
    printf "%s" "$$" > "${lock_file}"

    return 0
}

release_lock_with_pid() {
    local lock_file="${1}"
    local current_pid="$$"

    if [ ! -f "${lock_file}" ]; then
        return 0
    fi

    local lock_pid
    lock_pid=$(cat "${lock_file}" 2>/dev/null || printf "")

    # Only release if we own the lock
    if [ "${lock_pid}" = "${current_pid}" ]; then
        rm -f "${lock_file}"
        printf "Lock released by PID %s\n" "${current_pid}"
    else
        printf "WARNING: Lock held by different PID: %s\n" "${lock_pid}" >&2
    fi
}
```

## Pattern: Stale Lock Detection

Check if the process holding the lock is still running:

```bash
is_process_running() {
    local pid="${1}"

    if [ -z "${pid}" ]; then
        return 1
    fi

    # Check if process exists
    kill -0 "${pid}" 2>/dev/null
}

acquire_lock_safe() {
    local lock_file="${1}"
    local max_wait="${2:-30}"
    local waited=0

    while [ -f "${lock_file}" ] && [ "${waited}" -lt "${max_wait}" ]; do
        local lock_pid
        lock_pid=$(cat "${lock_file}" 2>/dev/null || printf "")

        # Check if lock is stale (process no longer running)
        if [ -n "${lock_pid}" ] && ! is_process_running "${lock_pid}"; then
            printf "Removing stale lock from dead process %s\n" "${lock_pid}"
            rm -f "${lock_file}"
            break
        fi

        printf "Waiting for lock held by PID %s...\n" "${lock_pid}"
        sleep 1
        waited=$((waited + 1))
    done

    if [ -f "${lock_file}" ]; then
        printf "ERROR: Could not acquire lock\n" >&2
        return 1
    fi

    printf "%s" "$$" > "${lock_file}"
    return 0
}
```

## Complete Example with Stale Lock Handling

```bash
#!/usr/bin/env bash

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
LOCK_FILE="${LOCK_FILE:-/tmp/backup.lock}"
MAX_WAIT="${MAX_WAIT:-60}"

is_process_running() {
    local pid="${1}"

    if [ -z "${pid}" ]; then
        return 1
    fi

    kill -0 "${pid}" 2>/dev/null
}

acquire_lock() {
    local lock_file="${1}"
    local max_wait="${2}"
    local waited=0

    while [ "${waited}" -lt "${max_wait}" ]; do
        # Check if lock exists
        if [ ! -f "${lock_file}" ]; then
            # No lock, acquire it
            printf "%s" "$$" > "${lock_file}"
            printf "Lock acquired by PID %s\n" "$$"
            return 0
        fi

        # Lock exists, check if stale
        local lock_pid
        lock_pid=$(cat "${lock_file}" 2>/dev/null || printf "")

        if [ -n "${lock_pid}" ] && ! is_process_running "${lock_pid}"; then
            printf "Removing stale lock from PID %s\n" "${lock_pid}"
            rm -f "${lock_file}"
            continue
        fi

        # Lock is valid, wait
        printf "Waiting for lock held by PID %s... (%s/%s)\n" \
               "${lock_pid}" "${waited}" "${max_wait}"
        sleep 1
        waited=$((waited + 1))
    done

    printf "ERROR: Timeout waiting for lock\n" >&2
    return 1
}

release_lock() {
    local lock_file="${1}"
    local current_pid="$$"

    if [ ! -f "${lock_file}" ]; then
        return 0
    fi

    local lock_pid
    lock_pid=$(cat "${lock_file}" 2>/dev/null || printf "")

    if [ "${lock_pid}" = "${current_pid}" ]; then
        rm -f "${lock_file}"
        printf "Lock released by PID %s\n" "${current_pid}"
    fi
}

cleanup() {
    release_lock "${LOCK_FILE}"
}

trap cleanup EXIT

backup_database() {
    printf "Backing up database...\n"
    sleep 3
    printf "Backup complete\n"
}

main() {
    printf "Starting backup process (PID: %s)\n" "$$"

    if ! acquire_lock "${LOCK_FILE}" "${MAX_WAIT}"; then
        printf "ERROR: Could not acquire lock\n" >&2
        return 1
    fi

    backup_database

    return 0
}

main "$@"
```

## Best Practices

1. **Always use timeout** - Prevent indefinite waiting
2. **Use trap for cleanup** - Ensure lock is released on exit
3. **Store PID in lock file** - Helps identify lock owner
4. **Handle stale locks** - Check if process is still running
5. **Unique lock file path** - Use script-specific lock file name
6. **Proper return codes** - Return non-zero when lock acquisition fails
7. **Log lock operations** - Help debug concurrent execution issues

## Common Pitfalls

- **No timeout** - Script hangs indefinitely waiting for lock
- **Forgetting to release** - Lock persists after script exits
- **No stale lock handling** - Crashed scripts leave permanent locks
- **Not checking lock ownership** - Releasing another process's lock
- **Using same lock file** - Multiple scripts interfere with each other
- **No trap handler** - Lock not released on errors or signals

## Testing Concurrent Execution

```bash
# Terminal 1 - acquire lock for 30 seconds
LOCK_FILE=/tmp/test.lock ./script.sh &

# Terminal 2 - try to acquire same lock (should wait/fail)
LOCK_FILE=/tmp/test.lock ./script.sh

# Test timeout behavior
MAX_WAIT=5 LOCK_FILE=/tmp/test.lock ./script.sh
```

## Related Patterns

- [Retry Logic Pattern](retry-logic-pattern.md) - Can be combined with lock acquisition retries
- [Temp Directory Pattern](temp-directory-pattern.md) - Lock files often stored in temp directories
- [Never-Nester Pattern](../never-nester-pattern.md) - Guard clauses for lock validation
