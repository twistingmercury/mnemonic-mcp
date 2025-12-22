---
entity_name: SOLID Principles for Shell Scripts Pattern
entity_type: shell-script-pattern
language: bash
domain: shell-scripting
description: Apply SOLID principles to shell script functions for better design, testability, and maintainability
tags:
  - shell
  - bash
  - solid
  - design-principles
  - architecture
  - best-practices
---

# SOLID Principles for Shell Scripts Pattern

## Overview

SOLID principles from object-oriented programming can be applied to shell script functions to create more maintainable, testable, and flexible code. This pattern shows how to implement each principle in shell scripts.

## Single Responsibility Principle

Each function should do one thing well. Use orchestrator functions to coordinate focused functions.

**Bad - function does too much:**

```bash
deploy() {
    backup_database
    stop_service
    update_code
    start_service
    verify_deployment
}
```

**Good - orchestrator calls focused functions:**

```bash
backup_database() {
    local backup_file="${1}"

    if [ -z "${backup_file}" ]; then
        printf "ERROR: Backup file path required\n" >&2
        return 1
    fi

    # Only handle database backup
    printf "Backing up database to %s\n" "${backup_file}"
    # Backup logic here...
    return 0
}

stop_service() {
    local service_name="${1}"

    if [ -z "${service_name}" ]; then
        printf "ERROR: Service name required\n" >&2
        return 1
    fi

    # Only handle service stopping
    printf "Stopping service: %s\n" "${service_name}"
    # Stop logic here...
    return 0
}

deploy() {
    local backup_file="${BACKUP_FILE}"
    local service_name="${SERVICE_NAME}"

    # Orchestrate, don't implement
    if ! backup_database "${backup_file}"; then
        printf "ERROR: Backup failed\n" >&2
        return 1
    fi

    if ! stop_service "${service_name}"; then
        printf "ERROR: Failed to stop service\n" >&2
        return 1
    fi

    # Continue orchestration...
    return 0
}
```

## Open/Closed Principle

Functions should be extensible without modification. Use environment variables to allow behavior extension.

**Good - behavior can be extended via environment variables:**

```bash
process_file() {
    local file_path="${1}"
    local processor="${FILE_PROCESSOR:-cat}"

    if [ ! -f "${file_path}" ]; then
        printf "ERROR: File not found: %s\n" "${file_path}" >&2
        return 1
    fi

    # Processor can be changed without modifying function
    "${processor}" "${file_path}"
    return 0
}

# Usage allows different processors without changing function
export FILE_PROCESSOR="jq ."
process_file "data.json"

# Or use default processor
unset FILE_PROCESSOR
process_file "data.txt"
```

**Another example - extensible notification:**

```bash
send_notification() {
    local message="${1}"
    local notifier="${NOTIFIER:-printf}"

    # Can inject different notification implementations
    "${notifier}" "Notification: %s\n" "${message}"
}

# Use custom notifier
custom_slack_notifier() {
    local message="${1}"
    # Send to Slack API
    curl -X POST "${SLACK_WEBHOOK}" -d "{\"text\": \"${message}\"}"
}

export NOTIFIER="custom_slack_notifier"
send_notification "Deployment complete"
```

## Liskov Substitution Principle

Functions with similar purposes should be interchangeable - same signature and behavior contract.

**Good - interchangeable backup implementations:**

```bash
backup_to_local() {
    local source="${1}"
    local dest="${2}"

    if [ -z "${source}" ] || [ -z "${dest}" ]; then
        printf "ERROR: Source and destination required\n" >&2
        return 1
    fi

    printf "Backing up %s to %s\n" "${source}" "${dest}"
    cp -r "${source}" "${dest}"
    return 0
}

backup_to_s3() {
    local source="${1}"
    local dest="${2}"

    if [ -z "${source}" ] || [ -z "${dest}" ]; then
        printf "ERROR: Source and destination required\n" >&2
        return 1
    fi

    printf "Backing up %s to S3: %s\n" "${source}" "${dest}"
    aws s3 cp "${source}" "${dest}" --recursive
    return 0
}

# Can swap implementations via environment variable
BACKUP_FUNCTION="${BACKUP_FUNCTION:-backup_to_local}"

# Both have same signature and behavior contract
"${BACKUP_FUNCTION}" "${SOURCE_DIR}" "${BACKUP_DEST}"
```

## Interface Segregation Principle

Don't force functions to depend on parameters they don't use. Only accept what you need.

**Bad - function doesn't use all parameters:**

```bash
process() {
    local file="${1}"
    local unused="${2}"
    local also_unused="${3}"

    cat "${file}"
}
```

**Good - only accept what you need:**

```bash
process_file() {
    local file="${1}"

    if [ ! -f "${file}" ]; then
        printf "ERROR: File not found: %s\n" "${file}" >&2
        return 1
    fi

    cat "${file}"
}

process_file_with_options() {
    local file="${1}"
    local options="${2}"

    if [ ! -f "${file}" ]; then
        printf "ERROR: File not found: %s\n" "${file}" >&2
        return 1
    fi

    # This function actually uses options
    cat "${options}" "${file}"
}
```

## Dependency Inversion Principle

Depend on abstractions (environment variables, function references) not concrete implementations.

**Bad - depends on concrete implementation:**

```bash
process_data() {
    local data="${1}"

    # Hardcoded dependency on specific logger
    echo "[LOG] Processing ${data}"

    # Processing logic...
}
```

**Good - depends on abstraction:**

```bash
log_message() {
    local message="${1}"
    local logger="${LOGGER:-printf}"

    # Depends on abstraction (LOGGER variable)
    "${logger}" "[LOG] %s\n" "${message}"
}

process_data() {
    local data="${1}"

    log_message "Processing ${data}"

    # Processing logic...
}

# Can inject different logger implementations
custom_logger() {
    local message="${1}"
    printf "%s - %s\n" "$(date +%Y-%m-%d\ %H:%M:%S)" "${message}"
}

export LOGGER="custom_logger"
process_data "important-file.txt"
```

## Complete Example: All Principles Together

```bash
#!/usr/bin/env bash

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Open/Closed: Extensible via environment variables
BACKUP_IMPL="${BACKUP_IMPL:-backup_local}"
NOTIFIER="${NOTIFIER:-notify_console}"
LOGGER="${LOGGER:-log_simple}"

# Dependency Inversion: Depend on abstraction
log() {
    local message="${1}"
    "${LOGGER}" "${message}"
}

log_simple() {
    local message="${1}"
    printf "[LOG] %s\n" "${message}"
}

# Liskov Substitution: Interchangeable implementations
backup_local() {
    local source="${1}"
    local dest="${2}"

    cp -r "${source}" "${dest}"
    return 0
}

backup_s3() {
    local source="${1}"
    local dest="${2}"

    aws s3 cp "${source}" "${dest}" --recursive
    return 0
}

# Interface Segregation: Only required parameters
validate_path() {
    local path="${1}"

    [ -e "${path}" ]
}

# Liskov Substitution: Interchangeable notifiers
notify_console() {
    local message="${1}"
    printf "NOTIFICATION: %s\n" "${message}"
}

notify_email() {
    local message="${1}"
    local email="${ADMIN_EMAIL}"
    echo "${message}" | mail -s "Backup Status" "${email}"
}

# Single Responsibility: Each function has one job
create_backup() {
    local source="${1}"
    local dest="${2}"

    # Only creates backup
    "${BACKUP_IMPL}" "${source}" "${dest}"
}

send_notification() {
    local message="${1}"

    # Only sends notification
    "${NOTIFIER}" "${message}"
}

# Single Responsibility: Orchestrator coordinates
main() {
    local source_dir="${SOURCE_DIR}"
    local backup_dir="${BACKUP_DIR}"

    if ! validate_path "${source_dir}"; then
        log "ERROR: Source directory not found"
        return 1
    fi

    log "Starting backup process"

    if create_backup "${source_dir}" "${backup_dir}"; then
        log "Backup successful"
        send_notification "Backup completed successfully"
        return 0
    fi

    log "Backup failed"
    send_notification "Backup failed"
    return 1
}

main "$@"
```

## Best Practices

1. **Single Responsibility** - One function, one purpose; use orchestrators for coordination
2. **Open/Closed** - Use environment variables for extensibility without modification
3. **Liskov Substitution** - Maintain consistent signatures and behavior contracts for interchangeable functions
4. **Interface Segregation** - Functions should only accept parameters they actually use
5. **Dependency Inversion** - Depend on environment variables and function references, not hardcoded implementations

## Benefits

- **Testability** - Small, focused functions are easier to test
- **Flexibility** - Can swap implementations via environment variables
- **Maintainability** - Changes are localized to specific functions
- **Reusability** - Well-defined functions can be used in multiple contexts
- **Composability** - Functions can be combined in different ways

## Common Pitfalls

- **God functions** - Functions that do everything violate Single Responsibility
- **Hardcoded dependencies** - Makes functions inflexible and hard to test
- **Inconsistent signatures** - Breaks Liskov Substitution, can't swap implementations
- **Unused parameters** - Violates Interface Segregation, confuses callers
- **No abstraction layer** - Direct coupling makes code rigid

## Related Patterns

- [Never-Nester Pattern](never-nester-pattern.md) - Single Responsibility promotes flat structure
- [Readability Pattern](readability-pattern.md) - SOLID principles improve clarity
- [Shell Script Pattern](shell-script-pattern.md) - Structure supports SOLID design
