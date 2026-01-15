---
entity_name: Readability Over Terseness Pattern
entity_type: shell-script-pattern
language: bash
domain: shell-scripting
description: Prioritize clear, readable code over clever one-liners to improve maintainability and debugging
tags:
  - shell
  - bash
  - readability
  - maintainability
  - best-practices
---

# Readability Over Terseness Pattern

## Overview

Shell scripts should be clear and maintainable, not clever or terse. Extract complex logic into named functions with descriptive variables. This makes scripts easier to understand, debug, and modify.

## Pattern

### Prefer Clear Code Over Clever One-Liners

**Bad (terse, hard to understand):**

```bash
[ $(find "$dir" -name "*.json" | wc -l) -eq 3 ] && echo "OK" || exit 1
```

**Good (readable, clear intent):**

```bash
count_json_files() {
    local dir="${1}"
    local file_count

    file_count=$(find "${dir}" -name "*.json" -type f | wc -l | tr -d ' ')

    if [ "${file_count}" -ne 3 ]; then
        printf "ERROR: Expected 3 JSON files, found %s\n" "${file_count}" >&2
        return 1
    fi

    printf "Found 3 JSON files\n"
    return 0
}
```

### Extract Complex Logic to Named Functions

**Bad (inline complexity):**

```bash
if [ -f "$config" ] && [ $(jq -r '.enabled' "$config") = "true" ] && [ $(date +%H) -ge 9 ]; then
    process_data
fi
```

**Good (extracted to descriptive functions):**

```bash
is_config_enabled() {
    local config_file="${1}"

    if [ ! -f "${config_file}" ]; then
        return 1
    fi

    local enabled
    enabled=$(jq -r '.enabled' "${config_file}")

    [ "${enabled}" = "true" ]
}

is_business_hours() {
    local current_hour
    current_hour=$(date +%H)

    [ "${current_hour}" -ge 9 ]
}

main() {
    local config_file="${CONFIG_FILE}"

    if is_config_enabled "${config_file}" && is_business_hours; then
        process_data
    fi

    return 0
}
```

### Use Descriptive Variable Names

**Bad (cryptic abbreviations):**

```bash
f="data.txt"
d="/tmp/out"
c=$(cat "$f" | wc -l)
```

**Good (clear, descriptive names):**

```bash
input_file="data.txt"
output_dir="/tmp/out"
line_count=$(wc -l < "${input_file}" | tr -d ' ')
```

## Complete Example: Refactoring for Readability

**Before (terse, hard to maintain):**

```bash
#!/usr/bin/env bash
set -e
[ -z "$1" ] && echo "Missing arg" && exit 1
[ ! -f "$1" ] && echo "Not found" && exit 1
cnt=$(cat "$1" | grep -c "ERROR" || echo 0)
[ $cnt -gt 0 ] && echo "Has errors: $cnt" || echo "Clean"
```

**After (readable, maintainable):**

```bash
#!/usr/bin/env bash

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
LOG_FILE="${1}"

validate_args() {
    if [ -z "${LOG_FILE}" ]; then
        printf "ERROR: Log file argument is required\n" >&2
        return 1
    fi

    if [ ! -f "${LOG_FILE}" ]; then
        printf "ERROR: Log file not found: %s\n" "${LOG_FILE}" >&2
        return 1
    fi

    return 0
}

count_errors() {
    local log_file="${1}"
    local error_count

    # Count ERROR entries, default to 0 if none found
    error_count=$(grep -c "ERROR" "${log_file}" || printf "0")

    printf "%s" "${error_count}"
}

check_log_status() {
    local log_file="${1}"
    local error_count

    error_count=$(count_errors "${log_file}")

    if [ "${error_count}" -gt 0 ]; then
        printf "Log has errors: %s\n" "${error_count}"
        return 1
    fi

    printf "Log is clean\n"
    return 0
}

main() {
    if ! validate_args; then
        return 1
    fi

    check_log_status "${LOG_FILE}"
    return $?
}

main "$@"
```

## Best Practices

1. **Extract complex conditions** - Put multi-part conditions in named functions
2. **Use descriptive variable names** - Avoid abbreviations and single letters
3. **One function, one purpose** - Keep functions focused and small
4. **Prefer readability over brevity** - Code is read more often than written
5. **Add intermediate variables** - Store complex command results in named variables
6. **Use clear function names** - Function should describe what it does, not how
7. **Avoid chaining with && and ||** - Use if statements for clarity

## Why Readability Matters

Readable code:

- **Easier to debug** - When something breaks, you can quickly understand what it does
- **Easier to modify** - Future changes don't require deciphering clever tricks
- **Easier to test** - Clear functions with single responsibilities are testable
- **Easier to review** - Team members can understand and verify the logic
- **Self-documenting** - Good naming reduces need for comments

## Common Pitfalls

- **Overusing && and ||** - Chains of conditions are hard to read and debug
- **Cryptic variable names** - `f`, `d`, `c` tell you nothing about purpose
- **Inline complex logic** - Embedding jq, awk, sed in conditions
- **Too many operations in one line** - Piping multiple commands without intermediate variables
- **Using echo instead of printf** - Less consistent, platform-dependent behavior
- **No function extraction** - Putting all logic in main() or as one-liners

## Related Patterns

- [Never-Nester Pattern](never-nester-pattern.md) - Uses readability principles with early returns
- [SOLID Principles Shell Pattern](solid-principles-shell-pattern.md) - Single Responsibility Principle promotes readability
- [Variable Naming and Quoting Pattern](variable-naming-quoting-pattern.md) - Naming conventions for readability
