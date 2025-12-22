---
entity_name: Shell Script Pattern
entity_type: shell-script-pattern
language: bash
domain: shell-scripting
description: Standard structure for executable shell scripts ensuring consistency, testability, and maintainability
tags:
  - shell
  - bash
  - structure
  - template
  - best-practices
  - executable
---

# Shell Script Pattern

Standard structure for executable shell scripts. All scripts follow this pattern to ensure consistency, testability, and maintainability.

## Structure

```bash
#!/usr/bin/env bash

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJ_ROOT="${PROJ_ROOT:-$(cd "${SCRIPT_DIR}/.." && pwd)}"
# Other directory referencing variables
# (e.g., CONFIG_DIR, DATA_DIR, etc.)

# sources
# (source library files here if needed)

# global var declarations
# (SCREAMING_SNAKE_CASE variables)

# global var validation function

validate_args() {
    if [ -z "${SOME_ARG}" ]; then
        printf "ERROR: %s is required\n" "SOME_ARG"
        return 1
    fi

    # other validations as needed

    # 0 means all validations passed
    return 0
}

# internal functions
# (helper functions, business logic)

main() {
    if ! validate_args; then
        return 1
    fi
    # Script entry point
    # Parse environment variables
    # Call internal functions
    # Return appropriate exit code
}

main "$@"
```

## Section Breakdown

### 1. Standard Header

```bash
#!/usr/bin/env bash

set -e
```

- **Shebang**: `#!/usr/bin/env bash` for portability
- **Error handling**: `set -e` exits script on first error

### 2. Directory Variables

```bash
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJ_ROOT="${PROJ_ROOT:-$(cd "${SCRIPT_DIR}/.." && pwd)}"
```

- **SCRIPT_DIR**: Absolute path to script's directory
- **PROJ_ROOT**: Project root, overridable via environment variable
- Add other directory variables as needed (CONFIG_DIR, DATA_DIR, etc.)

### 3. Sources

```bash
# sources
source "${SCRIPT_DIR}/lib/my-functions.sh"
```

Source any library files needed by the script.

### 4. Global Variable Declarations

```bash
# global var declarations
BACKUP_DIR="${BACKUP_DIR:-${PROJ_ROOT}/backups}"
MAX_RETRIES="${MAX_RETRIES:-3}"
VERBOSE="${VERBOSE:-false}"
```

- Use **SCREAMING_SNAKE_CASE** for global variables
- Allow override via environment variables using `${VAR:-default}` pattern
- Group related variables together

### 5. Validation Function

```bash
validate_args() {
    if [ -z "${REQUIRED_VAR}" ]; then
        printf "ERROR: REQUIRED_VAR is required\n" >&2
        return 1
    fi

    if [ ! -d "${SOME_DIR}" ]; then
        printf "ERROR: Directory not found: %s\n" "${SOME_DIR}" >&2
        return 1
    fi

    return 0
}
```

- Validates all required environment variables
- Uses early returns (never-nester pattern)
- Sends errors to stderr (`>&2`)
- Returns non-zero on validation failure

### 6. Internal Functions

```bash
process_file() {
    local file_path="${1}"
    local output_dir="${2}"

    # Early return on error
    if [ ! -f "${file_path}" ]; then
        printf "ERROR: File not found: %s\n" "${file_path}" >&2
        return 1
    fi

    # Do processing...
    printf "Processed: %s\n" "${file_path}"
    return 0
}
```

- Use **snake_case** for local variables
- Always quote variables: `"${var}"`
- Use early returns to avoid nesting
- One function, one responsibility

### 7. Main Function

```bash
main() {
    # Validate first
    if ! validate_args; then
        return 1
    fi

    # Then execute
    process_file "${INPUT_FILE}" "${OUTPUT_DIR}"

    # Return success
    return 0
}
```

- Entry point for script logic
- Calls validation first
- Coordinates other functions
- Returns appropriate exit code

### 8. Script Invocation

```bash
main "$@"
```

- Always at the end
- Passes all arguments to main function
- Exit code propagates from main

## Complete Example

```bash
#!/usr/bin/env bash

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJ_ROOT="${PROJ_ROOT:-$(cd "${SCRIPT_DIR}/.." && pwd)}"
BACKUP_DIR="${PROJ_ROOT}/backups"

# sources
source "${SCRIPT_DIR}/lib/file-helpers.sh"

# global var declarations
INPUT_FILE="${INPUT_FILE}"
OUTPUT_FILE="${OUTPUT_FILE:-output.txt}"
VERBOSE="${VERBOSE:-false}"

validate_args() {
    if [ -z "${INPUT_FILE}" ]; then
        printf "ERROR: INPUT_FILE is required\n" >&2
        return 1
    fi

    if [ ! -f "${INPUT_FILE}" ]; then
        printf "ERROR: File not found: %s\n" "${INPUT_FILE}" >&2
        return 1
    fi

    return 0
}

process_input() {
    local input="${1}"
    local output="${2}"

    if [ "${VERBOSE}" = "true" ]; then
        printf "Processing: %s -> %s\n" "${input}" "${output}"
    fi

    # Do actual processing
    cat "${input}" > "${output}"

    return 0
}

main() {
    if ! validate_args; then
        return 1
    fi

    if ! process_input "${INPUT_FILE}" "${OUTPUT_FILE}"; then
        printf "ERROR: Processing failed\n" >&2
        return 1
    fi

    printf "SUCCESS: Output written to %s\n" "${OUTPUT_FILE}"
    return 0
}

main "$@"
```

## Usage

```bash
# Set required environment variables
export INPUT_FILE="data.txt"
export OUTPUT_FILE="result.txt"
export VERBOSE="true"

# Run script
./process.sh
```

## Best Practices

1. **Never-nester pattern** - Use early returns instead of deep nesting
2. **Always quote variables** - Use `"${var}"` format everywhere
3. **POSIX compliance** - Use `printf` instead of `echo`
4. **Named environment variables** - Don't use flags for script-level arguments
5. **Validation first** - Check all requirements before execution
6. **Clear sections** - Follow the standard structure order
7. **Testability** - Functions can be unit tested by sourcing the script
8. **Error handling** - Return non-zero on errors, use stderr for error messages
