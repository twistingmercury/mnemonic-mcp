---
entity_name: Startup Validation Guideline
entity_type: engineering-guideline
language: agnostic
domain: configuration
description: Application startup configuration validation and fail-fast behavior
tags:
  - configuration
  - validation
  - fail-fast
  - startup
---

# Startup Validation Guideline

## Purpose

Applications must validate their configuration at startup and fail immediately if configuration is invalid. This fail-fast approach catches configuration mistakes early rather than causing subtle production issues during runtime.

## Core Requirements

### 1. Validate ALL Configuration at Startup

**Rule**: Perform comprehensive validation of all configuration values before the application begins processing requests.

**What to validate**:

- Required values are present
- Formats are correct (URLs, ports, timeouts, etc.)
- Values are within valid ranges
- Connections are testable (database, external APIs) where possible

**Example checks**:

- `DATABASE_URL` is present and well-formed
- `PORT` is between 1-65535
- `TIMEOUT` values are positive integers
- API keys are present when required

### 2. Exit with Code 1 on Validation Failure

**Rule**: When configuration validation fails, exit the process with exit code 1.

**Rationale**: Exit code 1 signals a fatal error to the container runtime or orchestrator, enabling automatic restart policies and alerting mechanisms.

**Implementation**:

```text
if validation_fails:
    log_critical("Configuration validation failed")
    exit(1)
```

### 3. Log at CRITICAL Level

**Rule**: All configuration validation failures must be logged at CRITICAL level with specific field names identifying what's wrong.

**Good error messages**:

- "DATABASE_URL is missing"
- "PORT must be between 1-65535, got: 99999"
- "TIMEOUT_SECONDS must be positive, got: -5"

**Bad error messages**:

- "Configuration error"
- "Invalid config"
- "Missing required field"

### 4. Report All Errors at Once

**Rule**: Collect and report ALL configuration validation failures in one operation, not just the first error encountered.

**Rationale**: Developers need to see all configuration problems to fix them in one iteration, rather than discovering them one at a time through repeated restart cycles.

**Implementation pattern**:

```text
validation_errors = []

if not database_url:
    validation_errors.append("DATABASE_URL is missing")

if port < 1 or port > 65535:
    validation_errors.append(f"PORT must be between 1-65535, got: {port}")

if timeout <= 0:
    validation_errors.append(f"TIMEOUT_SECONDS must be positive, got: {timeout}")

if validation_errors:
    for error in validation_errors:
        log_critical(error)
    exit(1)
```

### 5. No Retry Loops for Configuration Issues

**Rule**: Do not implement retry loops for configuration validation. If configuration is wrong, it won't fix itself.

**Rationale**: Configuration errors require human intervention to fix. Retry loops waste resources and delay problem detection.

**Let the orchestrator handle restarts**: Kubernetes and other orchestrators will restart your container based on their policies while you investigate and fix the configuration.

## What NOT to Do

### ❌ Fail Silently

```text
# BAD: Using defaults for missing required config
if not database_url:
    database_url = "localhost:5432"  # Silent fallback
```

### ❌ Continue with Invalid Config

```text
# BAD: Logging warning but continuing
if port > 65535:
    log_warning(f"Invalid port {port}, using 8080")
    port = 8080
```

### ❌ Discover Errors One at a Time

```text
# BAD: Stopping at first error
if not database_url:
    log_critical("DATABASE_URL is missing")
    exit(1)

# Never reaches this check if DATABASE_URL is missing
if port < 1 or port > 65535:
    log_critical(f"Invalid PORT: {port}")
    exit(1)
```

### ❌ Retry Configuration Validation

```text
# BAD: Retrying config validation
max_retries = 3
for attempt in range(max_retries):
    if validate_config():
        break
    time.sleep(5)
```

## Benefits

1. **Early detection**: Configuration problems are caught immediately at startup
2. **Clear feedback**: Developers see exactly what's wrong and can fix all issues at once
3. **No mysterious failures**: Prevents runtime failures caused by invalid configuration
4. **Orchestrator integration**: Exit code 1 enables proper restart policies and alerting
5. **Faster debugging**: All errors reported together reduce iteration cycles

## Integration with Defaults

This guideline works together with the [Configuration Defaults Guideline](./defaults-guideline.md):

- **Defaults**: Provide sensible values for optional configuration (fail loudly in production if accidentally used)
- **Validation**: Ensure required configuration is present and all values (including defaults) are valid

Configuration validation should happen AFTER defaults are applied, ensuring that both explicit and default values are validated.
