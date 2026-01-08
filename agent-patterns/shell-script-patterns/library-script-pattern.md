---
entity_name: Shell Script Library Pattern
entity_type: shell-script-pattern
language: bash
domain: shell-scripting
description: Reusable function libraries that are sourced by other scripts without direct execution
tags:
  - shell
  - bash
  - library
  - functions
  - reusable
  - sourcing
---

# Shell Script Library Pattern

Library scripts contain reusable functions that are sourced by other scripts. They do not execute directly and therefore do not have the standard script header (no shebang, no `set -e`, no `main()` function).

## Structure

```bash
# Library: my-functions.sh
# Description: Common utility functions for project
# Usage: source ./my-functions.sh

# Global constants (if needed)
# (SCREAMING_SNAKE_CASE)

# Namespace: my_functions
# All functions in this library use the my_functions:: prefix

my_functions::validate_required_var() {
    local var_name="${1}"
    local var_value="${2}"

    if [ -z "${var_value}" ]; then
        printf "ERROR: %s is required\n" "${var_name}" >&2
        return 1
    fi

    return 0
}

my_functions::print_success() {
    local message="${1}"
    printf "SUCCESS: %s\n" "${message}"
}

my_functions::print_error() {
    local message="${1}"
    printf "ERROR: %s\n" "${message}" >&2
}

my_functions::file_exists() {
    local file_path="${1}"

    if [ ! -f "${file_path}" ]; then
        my_functions::print_error "File not found: ${file_path}"
        return 1
    fi

    return 0
}
```

## Key Differences from Executable Scripts

**Library scripts DO NOT have**:

- Shebang (`#!/usr/bin/env bash`)
- `set -e` or other set options
- `SCRIPT_DIR` or `PROJ_ROOT` variables
- `main()` function
- `main "$@"` invocation

**Library scripts DO have**:

- Descriptive header comment block
- Namespace prefix on all functions
- Same naming conventions (snake_case locals, SCREAMING_SNAKE_CASE globals)
- Same quoting style (`"${var}"` always)

## Namespace Convention

The namespace is derived from the filename:

- Filename: `my-functions.sh` → Namespace: `my_functions::`
- Filename: `string_utils.sh` → Namespace: `string_utils::`
- Filename: `file-helpers.sh` → Namespace: `file_helpers::` (hyphens become underscores)

## Usage Example

```bash
#!/usr/bin/env bash

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJ_ROOT="${PROJ_ROOT:-$(cd "${SCRIPT_DIR}/.." && pwd)}"

# sources
source "${SCRIPT_DIR}/lib/my-functions.sh"

# global var declarations
CONFIG_FILE="${CONFIG_FILE:-config.json}"

# internal functions

main() {
    if ! my_functions::file_exists "${CONFIG_FILE}"; then
        return 1
    fi

    my_functions::print_success "Configuration loaded"

    # Do other work...
}

main "$@"
```

## Best Practices

1. **One library per domain** - Group related functions together
2. **Clear namespace** - Always use the namespace prefix for clarity
3. **Document functions** - Add comments explaining parameters and return values
4. **No side effects** - Library loading should not execute code or modify state
5. **Pure functions** - Prefer functions that don't depend on global state
6. **Error handling** - Return non-zero on errors, use stderr for error messages
