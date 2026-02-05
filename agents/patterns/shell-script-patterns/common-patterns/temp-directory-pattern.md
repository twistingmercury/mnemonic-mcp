---
entity_name: Temporary Directory Management Pattern
entity_type: shell-script-pattern
language: bash
domain: shell-scripting
description: Cross-platform temporary directory creation and cleanup pattern for shell scripts
tags:
  - shell
  - bash
  - temporary
  - cleanup
  - cross-platform
  - resource-management
---

# Temporary Directory Management Pattern

## Overview

Creating and managing temporary directories safely across platforms (macOS BSD and Linux GNU) requires handling different `mktemp` behaviors. This pattern ensures temporary resources are created reliably and cleaned up properly.

## Pattern

### Cross-Platform Temporary Directory Creation

```bash
create_temp_dir() {
    local temp_dir

    # BSD mktemp requires template with -t, GNU doesn't
    # Try GNU syntax first, fall back to BSD
    temp_dir=$(mktemp -d 2>/dev/null || mktemp -d -t 'prefix')

    if [ -z "${temp_dir}" ] || [ ! -d "${temp_dir}" ]; then
        printf "ERROR: Failed to create temporary directory\n" >&2
        return 1
    fi

    printf "%s" "${temp_dir}"
}
```

### Cleanup Helper Function

```bash
cleanup_temp_dir() {
    local temp_dir="${1}"

    if [ -z "${temp_dir}" ]; then
        return 0
    fi

    if [ -d "${temp_dir}" ]; then
        rm -rf "${temp_dir}"
    fi
}
```

### Using with Trap for Automatic Cleanup

```bash
#!/usr/bin/env bash

set -e

TEMP_DIR=""

cleanup() {
    if [ -n "${TEMP_DIR}" ] && [ -d "${TEMP_DIR}" ]; then
        rm -rf "${TEMP_DIR}"
    fi
}

# Register cleanup to run on exit
trap cleanup EXIT

main() {
    # Create temp directory
    TEMP_DIR=$(mktemp -d 2>/dev/null || mktemp -d -t 'script')

    if [ -z "${TEMP_DIR}" ]; then
        printf "ERROR: Failed to create temp directory\n" >&2
        return 1
    fi

    printf "Using temp directory: %s\n" "${TEMP_DIR}"

    # Use temp directory
    printf "test data\n" > "${TEMP_DIR}/data.txt"

    # Cleanup happens automatically via trap on exit
    return 0
}

main "$@"
```

## Complete Example with Error Handling

```bash
#!/usr/bin/env bash

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
TEMP_WORKSPACE=""

create_temp_workspace() {
    local temp_dir

    # Cross-platform mktemp
    temp_dir=$(mktemp -d 2>/dev/null || mktemp -d -t 'workspace')

    if [ -z "${temp_dir}" ] || [ ! -d "${temp_dir}" ]; then
        printf "ERROR: Failed to create temporary workspace\n" >&2
        return 1
    fi

    # Create subdirectories
    mkdir -p "${temp_dir}/input"
    mkdir -p "${temp_dir}/output"
    mkdir -p "${temp_dir}/logs"

    printf "%s" "${temp_dir}"
}

cleanup_workspace() {
    if [ -z "${TEMP_WORKSPACE}" ]; then
        return 0
    fi

    if [ -d "${TEMP_WORKSPACE}" ]; then
        printf "Cleaning up workspace: %s\n" "${TEMP_WORKSPACE}"
        rm -rf "${TEMP_WORKSPACE}"
    fi
}

# Cleanup on exit (success or failure)
trap cleanup_workspace EXIT

process_files() {
    local workspace="${1}"
    local input_file="${2}"

    # Copy input to workspace
    cp "${input_file}" "${workspace}/input/"

    # Process in isolated workspace
    cat "${workspace}/input/$(basename "${input_file}")" > \
        "${workspace}/output/result.txt"

    return 0
}

main() {
    local input_file="${INPUT_FILE}"

    if [ -z "${input_file}" ]; then
        printf "ERROR: INPUT_FILE required\n" >&2
        return 1
    fi

    if [ ! -f "${input_file}" ]; then
        printf "ERROR: Input file not found: %s\n" "${input_file}" >&2
        return 1
    fi

    # Create isolated workspace
    TEMP_WORKSPACE=$(create_temp_workspace)

    if [ -z "${TEMP_WORKSPACE}" ]; then
        return 1
    fi

    printf "Workspace created: %s\n" "${TEMP_WORKSPACE}"

    # Process files in workspace
    if ! process_files "${TEMP_WORKSPACE}" "${input_file}"; then
        printf "ERROR: Processing failed\n" >&2
        return 1
    fi

    printf "Processing complete\n"
    return 0
}

main "$@"
```

## Advanced Pattern: Named Temp Directories

```bash
create_named_temp_dir() {
    local prefix="${1:-temp}"
    local temp_base
    local temp_dir

    # Use $TMPDIR if set, otherwise use /tmp
    temp_base="${TMPDIR:-/tmp}"

    # Create unique directory with prefix
    temp_dir=$(mktemp -d "${temp_base}/${prefix}.XXXXXX" 2>/dev/null || \
               mktemp -d -t "${prefix}")

    if [ -z "${temp_dir}" ] || [ ! -d "${temp_dir}" ]; then
        printf "ERROR: Failed to create temp directory\n" >&2
        return 1
    fi

    printf "%s" "${temp_dir}"
}

# Usage
BACKUP_WORKSPACE=$(create_named_temp_dir "backup")
BUILD_WORKSPACE=$(create_named_temp_dir "build")
```

## Pattern: Temp Directory with Validation

```bash
create_validated_temp_dir() {
    local temp_dir
    local required_space="${1:-1048576}"  # Default 1MB

    temp_dir=$(mktemp -d 2>/dev/null || mktemp -d -t 'validated')

    if [ -z "${temp_dir}" ] || [ ! -d "${temp_dir}" ]; then
        printf "ERROR: Failed to create temp directory\n" >&2
        return 1
    fi

    # Verify we can write to it
    if [ ! -w "${temp_dir}" ]; then
        printf "ERROR: Temp directory not writable: %s\n" "${temp_dir}" >&2
        rm -rf "${temp_dir}"
        return 1
    fi

    # Check available space
    local available_space
    available_space=$(df "${temp_dir}" | tail -1 | awk '{print $4}')

    if [ "${available_space}" -lt "${required_space}" ]; then
        printf "ERROR: Insufficient space in temp directory\n" >&2
        rm -rf "${temp_dir}"
        return 1
    fi

    printf "%s" "${temp_dir}"
}
```

## Best Practices

1. **Always use mktemp** - Don't create temp directories manually
2. **Handle both BSD and GNU** - Use fallback pattern with `||`
3. **Validate creation** - Check that directory exists and is writable
4. **Use trap for cleanup** - Ensures cleanup happens even on errors
5. **Store in global variable** - Makes cleanup in trap handler possible
6. **Create subdirectories** - Organize temp workspace with clear structure
7. **Clean up on exit** - Use `trap cleanup EXIT` pattern

## Error Handling

Different `mktemp` behaviors to handle:

- **BSD (macOS)**: Requires `-t` flag with template
- **GNU (Linux)**: Doesn't require template
- **Busybox (Alpine)**: Limited options support

The fallback pattern handles all three:

```bash
mktemp -d 2>/dev/null || mktemp -d -t 'prefix'
```

## Common Pitfalls

- **Hardcoding /tmp** - Use mktemp to respect $TMPDIR
- **Not validating creation** - Check return value and directory existence
- **Forgetting cleanup** - Always use trap or explicit cleanup
- **Not making directory unique** - mktemp ensures uniqueness
- **Leaving cleanup to manual call** - Use trap for automatic cleanup

## Testing Across Platforms

```bash
# Test on macOS
./script.sh

# Test on Linux
docker run --rm -v "$PWD:/workspace" ubuntu:latest bash /workspace/script.sh

# Test on Alpine (busybox)
docker run --rm -v "$PWD:/workspace" alpine:latest sh /workspace/script.sh
```

## Related Patterns

- [Cross-Platform Pattern](../cross-platform-pattern.md) - General platform compatibility
- [Shell Script Pattern](../shell-script-pattern.md) - Standard script structure with cleanup
- [Never-Nester Pattern](../never-nester-pattern.md) - Guard clauses for temp dir validation
