---
entity_name: Never-Nester Pattern
entity_type: shell-script-pattern
language: bash
domain: shell-scripting
description: Avoid deep nesting by using early returns and guard clauses to create flatter, more readable code
tags:
  - shell
  - bash
  - never-nester
  - guard-clauses
  - early-return
  - readability
---

# Never-Nester Pattern

## Overview

The never-nester pattern eliminates deep nesting by using guard clauses with early returns. Instead of wrapping success logic in multiple nested if statements, validate conditions first and return early on failure. This creates flatter, more readable code where the happy path is clear.

## Pattern

### Avoid Deep Nesting with Guard Clauses

**Bad (nested):**

```bash
process_file() {
    if [ -f "$1" ]; then
        if [ -r "$1" ]; then
            if [ -s "$1" ]; then
                # Do processing
                return 0
            else
                echo "File is empty"
                return 1
            fi
        else
            echo "File not readable"
            return 1
        fi
    else
        echo "File not found"
        return 1
    fi
}
```

**Good (never-nester with early returns):**

```bash
process_file() {
    local file_path="${1}"

    # Guard clauses with early returns
    if [ ! -f "${file_path}" ]; then
        printf "ERROR: File not found: %s\n" "${file_path}" >&2
        return 1
    fi

    if [ ! -r "${file_path}" ]; then
        printf "ERROR: File not readable: %s\n" "${file_path}" >&2
        return 1
    fi

    if [ ! -s "${file_path}" ]; then
        printf "ERROR: File is empty: %s\n" "${file_path}" >&2
        return 1
    fi

    # Happy path - no nesting
    printf "Processing %s\n" "${file_path}"
    # Do processing...
    return 0
}
```

## Complete Example: Refactoring Nested to Flat

**Before (deeply nested):**

```bash
deploy_application() {
    if [ -n "$APP_NAME" ]; then
        if [ -d "$DEPLOY_DIR" ]; then
            if backup_database; then
                if stop_service "$APP_NAME"; then
                    if update_code; then
                        if start_service "$APP_NAME"; then
                            printf "Deployment successful\n"
                            return 0
                        else
                            printf "Failed to start service\n"
                            return 1
                        fi
                    else
                        printf "Failed to update code\n"
                        return 1
                    fi
                else
                    printf "Failed to stop service\n"
                    return 1
                fi
            else
                printf "Failed to backup database\n"
                return 1
            fi
        else
            printf "Deploy directory not found\n"
            return 1
        fi
    else
        printf "APP_NAME not set\n"
        return 1
    fi
}
```

**After (flat with early returns):**

```bash
deploy_application() {
    local app_name="${APP_NAME}"
    local deploy_dir="${DEPLOY_DIR}"

    # Guard clauses - validate everything upfront
    if [ -z "${app_name}" ]; then
        printf "ERROR: APP_NAME not set\n" >&2
        return 1
    fi

    if [ ! -d "${deploy_dir}" ]; then
        printf "ERROR: Deploy directory not found: %s\n" "${deploy_dir}" >&2
        return 1
    fi

    # Each step checked with early return on failure
    if ! backup_database; then
        printf "ERROR: Failed to backup database\n" >&2
        return 1
    fi

    if ! stop_service "${app_name}"; then
        printf "ERROR: Failed to stop service\n" >&2
        return 1
    fi

    if ! update_code; then
        printf "ERROR: Failed to update code\n" >&2
        return 1
    fi

    if ! start_service "${app_name}"; then
        printf "ERROR: Failed to start service\n" >&2
        return 1
    fi

    # Happy path at the end - clear and unindented
    printf "Deployment successful\n"
    return 0
}
```

## Pattern Variations

### Validation Functions

```bash
validate_environment() {
    # Check all prerequisites with early returns
    if [ -z "${DATABASE_URL}" ]; then
        printf "ERROR: DATABASE_URL not set\n" >&2
        return 1
    fi

    if [ -z "${API_KEY}" ]; then
        printf "ERROR: API_KEY not set\n" >&2
        return 1
    fi

    if ! command -v jq >/dev/null 2>&1; then
        printf "ERROR: jq is required but not installed\n" >&2
        return 1
    fi

    # All validations passed
    return 0
}

main() {
    # Single validation check at start
    if ! validate_environment; then
        return 1
    fi

    # Proceed with main logic
    process_data
    return 0
}
```

### Flattening else/if Chains

**Bad (nested else/if):**

```bash
process_status() {
    if [ "$status" = "success" ]; then
        printf "Operation succeeded\n"
        return 0
    else
        if [ "$status" = "pending" ]; then
            printf "Operation pending\n"
            return 2
        else
            if [ "$status" = "failed" ]; then
                printf "Operation failed\n"
                return 1
            else
                printf "Unknown status\n"
                return 3
            fi
        fi
    fi
}
```

**Good (flat with early returns):**

```bash
process_status() {
    local status="${1}"

    if [ "${status}" = "success" ]; then
        printf "Operation succeeded\n"
        return 0
    fi

    if [ "${status}" = "pending" ]; then
        printf "Operation pending\n"
        return 2
    fi

    if [ "${status}" = "failed" ]; then
        printf "Operation failed\n"
        return 1
    fi

    printf "Unknown status: %s\n" "${status}"
    return 3
}
```

## Best Practices

1. **Validate early** - Check all prerequisites at the start of the function
2. **Return immediately on failure** - Don't continue if conditions aren't met
3. **Put happy path last** - Success case should be at the end, unindented
4. **Use guard clauses** - Invert conditions to check for failure cases first
5. **One condition per if statement** - Don't combine multiple checks unless necessary
6. **Clear error messages** - Each early return should explain what failed
7. **Send errors to stderr** - Use `>&2` for error messages

## Benefits

- **Easier to read** - Linear flow from top to bottom
- **Easier to modify** - Add new validations without increasing nesting
- **Easier to debug** - Each failure condition is isolated and clear
- **Easier to test** - Each guard clause can be tested independently
- **Less indentation** - Happy path remains at consistent indentation level
- **Clearer intent** - Preconditions are explicit and upfront

## Common Pitfalls

- **Combining guards** - Using `&&` to combine multiple checks hides individual failures
- **Nesting after guards** - Adding nested if statements after guard clauses
- **Not using early returns** - Checking conditions but not returning immediately
- **Unclear error messages** - Not explaining which guard clause failed
- **Using else unnecessarily** - After a return statement, else is redundant

## Related Patterns

- [Readability Pattern](readability-pattern.md) - Clear code complements flat structure
- [SOLID Principles Shell Pattern](solid-principles-shell-pattern.md) - Single Responsibility works well with guard clauses
- [Shell Script Pattern](shell-script-pattern.md) - validate_args() function uses guard clauses
