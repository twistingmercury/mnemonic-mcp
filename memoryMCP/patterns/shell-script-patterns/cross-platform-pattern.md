---
entity_name: Cross-Platform Pattern
entity_type: shell-script-pattern
language: bash
domain: shell-scripting
description: Handle platform differences between macOS (BSD) and Linux (GNU) to write portable shell scripts
tags:
  - shell
  - bash
  - cross-platform
  - portability
  - bsd
  - gnu
  - macos
  - linux
---

# Cross-Platform Pattern

## Overview

Shell scripts often need to work across different platforms, primarily macOS (BSD-based) and Linux (GNU-based). This pattern shows how to handle platform-specific differences in common commands to ensure portability.

## Pattern

### stat Command (File Information)

The `stat` command has different syntax on BSD (macOS) and GNU (Linux).

**Portable file size:**

```bash
get_file_size() {
    local file="${1}"

    if [ ! -f "${file}" ]; then
        printf "ERROR: File not found: %s\n" "${file}" >&2
        return 1
    fi

    # Try BSD stat first, fall back to GNU stat
    stat -f%z "${file}" 2>/dev/null || stat -c%s "${file}" 2>/dev/null
}
```

**Portable file modification time:**

```bash
get_file_mtime() {
    local file="${1}"

    if [ ! -f "${file}" ]; then
        printf "ERROR: File not found: %s\n" "${file}" >&2
        return 1
    fi

    # BSD: -f%m, GNU: -c%Y
    stat -f%m "${file}" 2>/dev/null || stat -c%Y "${file}" 2>/dev/null
}
```

### grep Command (Pattern Matching)

**Bad (GNU-specific Perl regex):**

```bash
# -P flag only works on GNU grep
grep -P '\d{3}-\d{3}-\d{4}' file.txt
```

**Good (portable extended regex):**

```bash
# -E works on both BSD and GNU grep
grep -E '[0-9]{3}-[0-9]{3}-[0-9]{4}' file.txt
```

**Portable grep pattern matching:**

```bash
find_pattern() {
    local pattern="${1}"
    local file="${2}"

    if [ ! -f "${file}" ]; then
        printf "ERROR: File not found: %s\n" "${file}" >&2
        return 1
    fi

    # Use -E for extended regex (portable)
    grep -E "${pattern}" "${file}"
}
```

### find Command (File Search)

**Always specify -type explicitly:**

```bash
# Good - explicit type specification
find "${dir}" -type f -name "*.sh"

# Works consistently across platforms
find "${dir}" -type d -name "config"
```

**Portable find with execution:**

```bash
# Use -exec with explicit {} and \; (portable)
find "${dir}" -type f -name "*.log" -exec rm {} \;

# Or use + for better performance (also portable)
find "${dir}" -type f -name "*.log" -exec rm {} +
```

### mktemp Command (Temporary Files)

The `mktemp` command behaves differently on BSD and GNU systems.

**Portable temporary directory:**

```bash
create_temp_dir() {
    local temp_dir

    # BSD mktemp requires template, GNU mktemp doesn't
    temp_dir=$(mktemp -d 2>/dev/null || mktemp -d -t 'prefix')

    if [ -z "${temp_dir}" ] || [ ! -d "${temp_dir}" ]; then
        printf "ERROR: Failed to create temporary directory\n" >&2
        return 1
    fi

    printf "%s" "${temp_dir}"
}
```

**Portable temporary file:**

```bash
create_temp_file() {
    local temp_file

    # Try GNU syntax first, fall back to BSD
    temp_file=$(mktemp 2>/dev/null || mktemp -t 'prefix')

    if [ -z "${temp_file}" ] || [ ! -f "${temp_file}" ]; then
        printf "ERROR: Failed to create temporary file\n" >&2
        return 1
    fi

    printf "%s" "${temp_file}"
}
```

## Complete Cross-Platform Example

```bash
#!/usr/bin/env bash

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Portable file size retrieval
get_file_size() {
    local file="${1}"

    if [ ! -f "${file}" ]; then
        printf "ERROR: File not found: %s\n" "${file}" >&2
        return 1
    fi

    stat -f%z "${file}" 2>/dev/null || stat -c%s "${file}" 2>/dev/null
}

# Portable temp directory creation
create_temp_workspace() {
    local temp_dir

    temp_dir=$(mktemp -d 2>/dev/null || mktemp -d -t 'workspace')

    if [ -z "${temp_dir}" ] || [ ! -d "${temp_dir}" ]; then
        printf "ERROR: Failed to create temp workspace\n" >&2
        return 1
    fi

    printf "%s" "${temp_dir}"
}

# Portable pattern search
find_log_errors() {
    local log_file="${1}"

    if [ ! -f "${log_file}" ]; then
        printf "ERROR: Log file not found: %s\n" "${log_file}" >&2
        return 1
    fi

    # Use -E for extended regex (works on BSD and GNU)
    grep -E '(ERROR|FATAL|CRITICAL)' "${log_file}"
}

# Portable file finding
find_shell_scripts() {
    local search_dir="${1}"

    if [ ! -d "${search_dir}" ]; then
        printf "ERROR: Directory not found: %s\n" "${search_dir}" >&2
        return 1
    fi

    # Explicitly specify -type for portability
    find "${search_dir}" -type f -name "*.sh"
}

cleanup_temp_workspace() {
    local temp_dir="${1}"

    if [ -d "${temp_dir}" ]; then
        rm -rf "${temp_dir}"
    fi
}

main() {
    local input_file="${INPUT_FILE}"
    local temp_workspace
    local file_size

    # Create portable temp directory
    temp_workspace=$(create_temp_workspace)

    if [ -z "${temp_workspace}" ]; then
        return 1
    fi

    # Get file size portably
    file_size=$(get_file_size "${input_file}")
    printf "File size: %s bytes\n" "${file_size}"

    # Find errors using portable grep
    if find_log_errors "${input_file}" > "${temp_workspace}/errors.txt"; then
        printf "Found errors in log file\n"
    fi

    # Clean up
    cleanup_temp_workspace "${temp_workspace}"

    return 0
}

main "$@"
```

## Platform Detection (When Needed)

Sometimes you need to detect the platform explicitly:

```bash
detect_platform() {
    local os_name

    os_name=$(uname -s)

    case "${os_name}" in
        Darwin*)
            printf "macos"
            ;;
        Linux*)
            printf "linux"
            ;;
        *)
            printf "unknown"
            ;;
    esac
}

# Use platform-specific behavior
main() {
    local platform
    platform=$(detect_platform)

    case "${platform}" in
        macos)
            # macOS-specific logic
            ;;
        linux)
            # Linux-specific logic
            ;;
        *)
            printf "ERROR: Unsupported platform\n" >&2
            return 1
            ;;
    esac
}
```

## Best Practices

1. **Use portable command flags** - Prefer options that work on both BSD and GNU
2. **Test on multiple platforms** - Run scripts on both macOS and Linux
3. **Prefer fallback patterns** - Try one syntax, fall back to another with `||`
4. **Use grep -E** - Extended regex works everywhere, -P doesn't
5. **Specify find -type** - Explicit type makes behavior consistent
6. **Handle mktemp differences** - Use fallback syntax for temp files/directories
7. **Document platform requirements** - If script is platform-specific, document it

## Common Pitfalls

- **Using grep -P** - Perl regex not available on BSD/macOS
- **Assuming GNU stat syntax** - BSD stat uses different flags
- **Not handling mktemp variations** - Different template requirements
- **Platform-specific sed** - sed behavior varies, use simple patterns or awk
- **Using readlink -f** - Not available on macOS, use alternative methods
- **Assuming GNU date** - Date command syntax differs significantly

## Testing Cross-Platform Scripts

```bash
# Test on macOS
./script.sh

# Test on Linux (using Docker)
docker run --rm -v "$PWD:/workspace" ubuntu:latest /workspace/script.sh

# Test on Alpine (uses busybox, very minimal)
docker run --rm -v "$PWD:/workspace" alpine:latest /workspace/script.sh
```

## Related Patterns

- [POSIX Compliance Pattern](posix-compliance-pattern.md) - POSIX compliance helps with portability
- [Temp Directory Pattern](common-patterns/temp-directory-pattern.md) - Portable temp directory handling
- [Shell Script Pattern](shell-script-pattern.md) - Standard structure works across platforms
