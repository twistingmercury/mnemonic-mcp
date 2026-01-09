---
entity_name: Variable Naming and Quoting Pattern
entity_type: shell-script-pattern
language: bash
domain: shell-scripting
description: Strict conventions for naming and referencing shell variables to ensure consistency and prevent common errors
tags:
  - shell
  - bash
  - variables
  - naming-conventions
  - quoting
---

# Variable Naming and Quoting Pattern

## Overview

Consistent variable naming and proper quoting prevents common shell scripting errors and makes scripts more maintainable. This pattern enforces strict conventions for global variables, local variables, and how to reference them safely.

## Pattern

### Naming Convention

**Global Variables: SCREAMING_SNAKE_CASE**

```bash
# Global variables - visible throughout script
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJ_ROOT="${PROJ_ROOT:-$(cd "${SCRIPT_DIR}/.." && pwd)}"
MAX_RETRIES="${MAX_RETRIES:-3}"
BACKUP_DIR="${PROJ_ROOT}/backups"
```

**Local Variables: snake_case**

```bash
# Function with local variables
process_file() {
    local file_path="${1}"
    local output_dir="${2}"
    local result_code
    local temp_file

    # All lowercase for local scope
    if [ -f "${file_path}" ]; then
        result_code=$(process "${file_path}" "${output_dir}")
        printf "Result: %s\n" "${result_code}"
    fi

    return 0
}
```

**Constants: SCREAMING_SNAKE_CASE with readonly**

```bash
# Define constants that should not be modified
readonly DEFAULT_TIMEOUT=30
readonly MAX_CONNECTIONS=100
readonly CONFIG_VERSION="1.0.0"
```

### Referencing Style

**Always use curly brackets: "${var}"**

```bash
# Good - curly brackets always
printf "Processing %s\n" "${file_path}"
cp "${source_file}" "${destination_dir}/"

# Bad - missing curly brackets
printf "Processing %s\n" "$file_path"
cp "$source_file" "$destination_dir/"
```

**Always quote variables: "${var}" not $var or ${var}**

```bash
# Good - quoted with curly brackets
if [ -f "${file_path}" ]; then
    cat "${file_path}" > "${output_file}"
fi

# Bad - unquoted (breaks with spaces in filenames)
if [ -f $file_path ]; then
    cat $file_path > $output_file
fi

# Bad - curly brackets but not quoted (still breaks with spaces)
if [ -f ${file_path} ]; then
    cat ${file_path} > ${output_file}
fi
```

## Complete Function Example

```bash
#!/usr/bin/env bash

set -e

# Global variables - SCREAMING_SNAKE_CASE
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJ_ROOT="${PROJ_ROOT:-$(cd "${SCRIPT_DIR}/.." && pwd)}"
OUTPUT_DIR="${OUTPUT_DIR:-${PROJ_ROOT}/output}"
VERBOSE="${VERBOSE:-false}"

# Constants
readonly DEFAULT_EXTENSION=".txt"
readonly MAX_FILE_SIZE=1048576  # 1MB

validate_args() {
    # Local variables - snake_case
    local required_var="${1}"

    if [ -z "${required_var}" ]; then
        printf "ERROR: Required variable is missing\n" >&2
        return 1
    fi

    return 0
}

process_file() {
    # Local variables - all snake_case
    local file_path="${1}"
    local output_dir="${2}"
    local file_size
    local result_code

    # Guard clauses - always quote, always use curly brackets
    if [ ! -f "${file_path}" ]; then
        printf "ERROR: File not found: %s\n" "${file_path}" >&2
        return 1
    fi

    # Get file size using local variable
    file_size=$(stat -f%z "${file_path}" 2>/dev/null || stat -c%s "${file_path}" 2>/dev/null)

    # Check against global constant
    if [ "${file_size}" -gt "${MAX_FILE_SIZE}" ]; then
        printf "ERROR: File too large: %s bytes\n" "${file_size}" >&2
        return 1
    fi

    # Process with properly quoted variables
    if [ "${VERBOSE}" = "true" ]; then
        printf "Processing: %s -> %s\n" "${file_path}" "${output_dir}"
    fi

    # All variables quoted with curly brackets
    result_code=$(cat "${file_path}" > "${output_dir}/result${DEFAULT_EXTENSION}")

    return 0
}

main() {
    # Local variables in main function
    local input_file="${INPUT_FILE}"
    local status

    if ! validate_args "${input_file}"; then
        return 1
    fi

    if ! process_file "${input_file}" "${OUTPUT_DIR}"; then
        printf "ERROR: Processing failed\n" >&2
        return 1
    fi

    printf "SUCCESS: File processed\n"
    return 0
}

main "$@"
```

## Best Practices

1. **Global variables always SCREAMING_SNAKE_CASE** - Makes globals immediately visible
2. **Local variables always snake_case** - Clear distinction from globals
3. **Always use curly brackets "${var}"** - Consistent style, prevents ambiguity
4. **Always quote variables** - Prevents word splitting and glob expansion
5. **Declare locals at function start** - Makes function signature clear
6. **Use readonly for constants** - Prevents accidental modification
7. **Initialize locals from parameters** - Clear parameter mapping

## Why Always Quote?

Unquoted variables cause problems:

```bash
# Without quotes - breaks with spaces
file_name="my document.txt"
cat $file_name  # Expands to: cat my document.txt (two arguments!)

# With quotes - works correctly
cat "${file_name}"  # Expands to: cat "my document.txt" (one argument)
```

```bash
# Without quotes - glob expansion
pattern="*.txt"
files=$pattern  # Expands to list of .txt files!

# With quotes - literal value
files="${pattern}"  # Stores the literal string "*.txt"
```

## Common Pitfalls

- **Mixing naming conventions** - Using snake_case for some globals, SCREAMING_SNAKE_CASE for others
- **Unquoted variables** - Breaks with spaces, special characters, or empty values
- **Missing curly brackets** - Inconsistent style, harder to read
- **Using $var directly** - Should always be "${var}"
- **Not declaring local variables** - Variables leak into global scope
- **Modifying readonly variables** - Causes script errors

## Related Patterns

- [POSIX Compliance Pattern](posix-compliance-pattern.md) - POSIX-compliant variable usage
- [Shell Script Pattern](shell-script-pattern.md) - Complete script structure with proper variable usage
- [Never-Nester Pattern](never-nester-pattern.md) - Uses proper variable quoting in guard clauses
