---
entity_name: Configuration Precedence Guideline
entity_type: engineering-guideline
language: agnostic
domain: configuration
description: 12-factor app configuration hierarchy and precedence rules
tags:
  - configuration
  - 12-factor
  - environment-variables
---

# Configuration Precedence Guideline

## Overview

Applications must support multiple configuration sources to work effectively across development, testing, and production environments. This guideline defines the standardized precedence hierarchy for configuration values, ensuring consistent behavior while maintaining flexibility for different deployment contexts.

This approach follows the [12-factor app configuration methodology](https://12factor.net/config), emphasizing environment-based configuration that separates code from config.

## Configuration Precedence Order

Configuration sources are evaluated in this order, with **later sources overriding earlier ones**:

1. **Built-in defaults** – Safe values for development and testing
2. **Configuration files** – Structured config for complex settings
3. **Environment variables** – Container-friendly overrides for runtime
4. **Command line flags** – Explicit overrides for debugging and testing

### Precedence Rules

- Each configuration source completely overrides the previous source for any values it provides
- Unspecified values in higher-precedence sources inherit from lower-precedence sources
- Missing required configuration should cause startup failure (fail fast principle)

### 12-Factor App Alignment

In production containerized environments, environment variables should contain all necessary configuration rather than relying on built-in defaults or configuration files. The precedence order supports local development workflows where config files provide convenience, while maintaining production best practices.

## Built-in Defaults

### Requirements

Built-in defaults **must never** be production-ready values. They should be development values that fail safely if accidentally used in production.

### Safe Default Patterns

**Database URLs:**

- Use: `localhost:5432`, `127.0.0.1:3306`
- Result: Connection refused errors in production
- Prevents: Silent data corruption or wrong database access

**External APIs:**

- Use: Mock endpoints, stub services, or dedicated sandbox environments
- Result: Obvious failures or sandbox-scoped operations
- Prevents: Accidental production API calls during development

**Timeouts:**

- Use: Conservative values like `30s` instead of aggressive `5s`
- Result: More breathing room for local debugging
- Prevents: Premature timeouts in development environments

**Feature Flags:**

- Use: All experimental features OFF by default
- Result: Only tested features run in production
- Prevents: Untested code paths executing unexpectedly

### Anti-patterns

Never use these as built-in defaults:

- Production database connection strings
- Production API keys or endpoints
- Aggressive timeout values optimized for production
- Feature flags that enable untested functionality

## Configuration Files

### Supported Formats

Applications should support at least one structured configuration file format. Common choices:

- **YAML** – Human-readable, good for complex nested structures
- **JSON** – Ubiquitous, strict syntax, machine-friendly
- **TOML** – Simple, clear syntax, good for application config

### File Usage Patterns

**Include in version control:**

- Non-sensitive settings (port numbers, log levels, timeouts)
- Feature flags and default behaviors
- Retry counts and backoff strategies

**Do NOT include in version control:**

- Environment-specific settings (service URLs, database names)
- Credentials and secrets (always use secrets management)
- Developer-specific overrides (add to `.gitignore`)

### Configuration File Locations

Applications should check for configuration files in standard locations:

1. Working directory: `./config.yaml`, `./config.json`
2. User config directory: `~/.config/appname/config.yaml`
3. System config directory: `/etc/appname/config.yaml`

Allow users to override with a `--config` flag or `CONFIG_FILE` environment variable.

## Environment Variables

### When to Use

Environment variables are the preferred configuration method for:

- **Docker/Kubernetes deployments** – Native runtime injection
- **Secrets and credentials** – Populated from secret stores
- **Environment-specific overrides** – Different values per deployment
- **Cloud-native platforms** – Universal orchestration support

### Environment Variable Naming

While specific naming conventions may vary by language ecosystem, follow these general principles:

- Use UPPERCASE with underscores: `DATABASE_URL`, `API_KEY`
- Prefix with application or service name: `MYAPP_DATABASE_URL`
- Group related settings logically: `DB_HOST`, `DB_PORT`, `DB_NAME`
- Be descriptive and unambiguous: `MAX_RETRY_ATTEMPTS` not `MAX_RETRIES`

### Secrets Handling

Environment variables containing secrets must:

- Be populated at runtime from secure sources (never hardcoded)
- Use secrets management systems (Azure Key Vault, AWS Secrets Manager, Kubernetes Secrets)
- Never be logged or included in error messages
- Be redacted in stack traces and diagnostic output

### Anti-patterns

Never:

- Commit `.env` files containing secrets to version control (add to `.gitignore`)
- Log environment variable values during startup
- Include credentials in container image environment settings
- Use environment variables for large configuration payloads (use config files)

## Command Line Flags

### Purpose

Command line flags provide the highest precedence for:

- **Debugging overrides** – Temporary configuration changes during troubleshooting
- **Explicit testing** – Running with known configuration states
- **Quick changes** – Modifying behavior without editing files or env vars
- **Transparency** – Making configuration differences visible in process lists

### Flag Design Principles

- Use clear, descriptive names: `--database-url` not `-d`
- Provide both short and long forms for common flags: `-p` / `--port`
- Include help text with examples: `--port PORT  Server port (default: 8080)`
- Support reading from stdin for sensitive values: `--api-key-file=/dev/stdin`

### Examples

```bash
# Override database URL for testing
./myapp --database-url=localhost:5432

# Temporary debug logging
./myapp --log-level=debug

# Explicit port binding
./myapp --port=8080

# Multiple overrides
./myapp --port=8080 --log-level=debug --read-timeout=60s
```

## Configuration Validation

### Fail Fast on Startup

Applications must validate configuration at startup and fail immediately if something is wrong.

### Validation Behaviors

When configuration validation fails:

- **Exit with code 1** – Signal fatal error to container runtime
- **Log at CRITICAL level** – Use highest severity logging level
- **Report specific failures** – Include field names and invalid values
- **Validate all configuration** – Report all errors at once, not just the first
- **No retry loops** – Don't attempt to fix invalid configuration automatically
- **Let orchestrator handle restarts** – Kubernetes/Docker will manage retry policies

### What to Validate

**Required values:**

- Check presence of mandatory configuration parameters
- Fail if required values are missing or empty

**Format validation:**

- URLs are well-formed and parseable
- Ports are in valid range (1-65535)
- Timeouts are positive durations
- File paths exist and are accessible

**Connection testing (optional):**

- Test database connectivity if connection strings provided
- Verify external API endpoints are reachable
- Validate credentials if feasible without side effects

### Error Message Requirements

Configuration validation errors must be:

- **Specific**: "DATABASE_URL is missing" not "configuration error"
- **Actionable**: "PORT must be between 1-65535, got: 99999"
- **Complete**: List all validation failures, not just the first
- **Clear**: Avoid technical jargon in user-facing messages

### Example Validation Output

```text
CRITICAL: Configuration validation failed:
  - DATABASE_URL is missing or empty
  - PORT must be between 1-65535, got: 99999
  - LOG_LEVEL must be one of [debug, info, warn, error], got: trace
  - API_TIMEOUT must be positive duration, got: -5s
Exiting with code 1
```

## Docker and Kubernetes Patterns

### Docker Containers

**Best practices:**

- Use environment variables for runtime configuration
- Bake default config files into container image
- Mount custom config files as volumes when needed
- Never hardcode values that differ between environments

**Example Dockerfile:**

```dockerfile
FROM alpine:3.19
COPY config.yaml /etc/myapp/config.yaml
COPY myapp /usr/local/bin/myapp
ENV LOG_LEVEL=info
CMD ["myapp"]
```

### Kubernetes Deployments

**Configuration sources:**

- **ConfigMaps** – Non-sensitive configuration data
- **Secrets** – Sensitive data (passwords, API keys, tokens)
- **Environment variables** – Expose ConfigMap and Secret values
- **Volume mounts** – Complex configuration files

**Example Deployment:**

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp
spec:
  template:
    spec:
      containers:
        - name: myapp
          image: myapp:1.0.0
          env:
            - name: DATABASE_URL
              valueFrom:
                secretKeyRef:
                  name: myapp-secrets
                  key: database-url
            - name: LOG_LEVEL
              valueFrom:
                configMapKeyRef:
                  name: myapp-config
                  key: log-level
```

## Configuration Storage by Type

Different configuration data types have different security and lifecycle requirements:

| Configuration Type                | Storage Method                       | Examples                                                          | Version Controlled?          |
| --------------------------------- | ------------------------------------ | ----------------------------------------------------------------- | ---------------------------- |
| **Non-sensitive settings**        | Config files in repo                 | Port numbers, log levels, timeouts, feature flags, retry counts   | Yes                          |
| **Environment-specific settings** | ConfigMaps / Environment variables   | Service URLs, database names, queue names, external API endpoints | No (managed per environment) |
| **Credentials & secrets**         | Secrets Manager / Kubernetes Secrets | Passwords, API keys, certificates, encryption keys, access tokens | Never                        |

## Common Pitfalls

### Hardcoded Production Values

**Problem:** Built-in defaults use production endpoints or credentials.

**Solution:** Use localhost and sandbox endpoints that fail safely in production.

### Missing Configuration Validation

**Problem:** Invalid configuration causes runtime failures instead of startup failures.

**Solution:** Validate all configuration at startup and fail fast with clear error messages.

### Secrets in Version Control

**Problem:** `.env` files or config files containing credentials committed to git.

**Solution:** Add `.env` to `.gitignore` immediately. Use secrets management systems. Rotate any exposed credentials.

### Configuration Sprawl

**Problem:** Configuration spread across too many files and sources, hard to track.

**Solution:** Standardize on 2-3 configuration methods maximum. Document the precedence clearly.

### Environment-Specific Code

**Problem:** Code checks environment names and branches behavior.

**Solution:** Use configuration values instead of environment detection. Make behavior configurable, not hardcoded.

## Related Guidelines

- [Security in Development](../../security/security-in-development.md)
- [Secrets Management](../../security/secrets-management.md)
- [12-Factor App Methodology](https://12factor.net/config)
- [Docker Best Practices](../../docker/docker-best-practices.md)
- [Kubernetes Configuration Patterns](../../kubernetes/config-patterns.md)
