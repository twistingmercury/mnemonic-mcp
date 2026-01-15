---
entity_name: POSIX Compliance Pattern
entity_type: shell-script-pattern
language: bash
domain: shell-scripting
description: POSIX-compliant constructs for portable shell scripts that work across different platforms and shells
tags:
  - shell
  - bash
  - posix
  - portability
  - compliance
---

# POSIX Compliance Pattern

## Overview

POSIX compliance ensures shell scripts work consistently across different platforms (macOS, Linux) and shell implementations (bash, sh, dash). Using POSIX-compliant constructs prevents subtle bugs and platform-specific failures.

## Pattern

### Use printf Instead of echo

**Bad (non-POSIX):**

```bash
echo "Processing $file"
```

**Good (POSIX-compliant):**

```bash
printf "Processing %s\n" "${file}"
```

### Use $() Instead of Backticks

**Bad (non-POSIX):**

```bash
result=`ls -la`
```

**Good (POSIX-compliant):**

```bash
result=$(ls -la)
```

### Use [ ] Instead of [[ ]]

**Bad (bash-specific):**

```bash
[[ $status == "success" ]]
```

**Good (POSIX-compliant):**

```bash
[ "${status}" = "success" ]
```

Note: Use single `=` for string comparison in `[ ]`, not `==`.

### Use Portable Flag Syntax

**Bad (GNU-specific):**

```bash
grep -P 'pattern' file.txt  # -P not available on BSD/macOS
```

**Good (portable):**

```bash
grep -E 'pattern' file.txt  # -E works on both BSD and GNU
```

## Complete Example

```bash
#!/usr/bin/env bash

set -e

# POSIX-compliant variable assignment
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
STATUS="${STATUS:-pending}"

# POSIX-compliant function
check_status() {
    local current_status="${1}"

    # Use [ ] for tests, = for string comparison
    if [ "${current_status}" = "success" ]; then
        printf "Status is success\n"
        return 0
    fi

    printf "Status is %s\n" "${current_status}"
    return 1
}

main() {
    local result

    # Use $() for command substitution
    result=$(check_status "${STATUS}")

    # Use printf for output
    printf "Result: %s\n" "${result}"

    return 0
}

main "$@"
```

## Best Practices

1. **Always use printf** - More predictable output formatting across platforms
2. **Prefer $() over backticks** - Easier to nest, more readable
3. **Use [ ] for portability** - Works in all POSIX shells
4. **Test with different shells** - Run scripts with `sh`, `bash`, and `dash` to verify portability
5. **Use grep -E** - Extended regex works on both BSD and GNU grep
6. **Avoid bash-isms** - Unless you explicitly require bash-specific features (arrays, associative arrays)

## Common Pitfalls

- **Using echo for variable output** - Behavior varies across platforms, especially with escape sequences
- **Using [[ ]] in POSIX mode** - Not available in all shells, bash-specific
- **Using grep -P** - Perl regex not available on BSD/macOS systems
- **Using == in [ ]** - Not POSIX compliant, use single = for string comparison
- **Backticks for command substitution** - Harder to read and nest, deprecated in favor of $()

## Related Patterns

- [Variable Naming and Quoting Pattern](variable-naming-quoting-pattern.md) - Proper variable referencing complements POSIX compliance
- [Cross-Platform Pattern](cross-platform-pattern.md) - Additional platform compatibility considerations
- [Shell Script Pattern](shell-script-pattern.md) - Complete script structure using POSIX constructs
