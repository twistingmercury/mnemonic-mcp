---
entity_name: Structured Logging Guideline
entity_type: engineering-guideline
language: agnostic
domain: observability
description: Standard structured logging format with trace correlation for all services
tags:
  - logging
  - observability
  - tracing
  - opentelemetry
  - correlation
---

# Structured Logging Guideline

## Philosophy

Logs are for anomalies, not normal operations. Distributed tracing captures successful request flows. Logs surface deviations from expected behavior that need human attention. Trace context ties logs and traces together for complete investigation.

## Production Log Level

**Default to WARN level in production.**

Traces capture normal operational flow with timing, dependencies, and context. Logs at WARN and above focus on actionable anomalies and failures. This approach provides complete observability without storing logs for millions of successful requests.

### Log Level Usage

- **Debug**: Low-level diagnostic information for development and troubleshooting
- **Info**: Normal business events (user logged in, order processed, etc.)
- **Warn**: System working correctly, but something unusual happened that needs attention
- **Error**: Something failed, but attempting recovery
- **Critical**: Fatal errors preventing application start or function

### Resource Limit Thresholds

Use graduated severity thresholds when logging resource utilization warnings:

| Resource Utilization | Severity Level | Urgency                             |
| -------------------- | -------------- | ----------------------------------- |
| < 75%                | (No log)       | Normal operation                    |
| 75% - 80%            | **Warn**       | Approaching limit, investigate soon |
| 80% - 90%            | **Error**      | Critical threshold, action needed   |
| > 90%                | **Critical**   | Imminent failure risk               |

**Apply these thresholds to:**

- Memory usage
- CPU utilization
- Disk space
- Database connection pools
- File descriptor counts
- Cache hit ratios
- Queue depths
- Thread pool sizes

### Warning Usage Discipline

Do NOT overuse the `warn` level. Reserve it for conditions requiring investigation to maintain system health:

- Invalid request content
- Approaching resource limits (per thresholds above)
- Degraded performance patterns
- Retryable errors
- Business rule violations

If it does not require investigation, log it at `info` level. Excessive warnings create noise and alert fatigue.

## Required Log Fields

Every log entry MUST include these standard fields:

### Always Required

- **timestamp**: UTC in ISO 8601 format (e.g., `2025-11-11T14:23:45Z`)
- **severity**: `debug`, `info`, `warn`, `error`, `critical`
- **action**: What operation was happening
- **component**: Which service or module
- **method**: Which function or handler

### Required When Trace Context Active

- **trace_id**: 32 lowercase hexadecimal characters (e.g., `4bf92f3577b34da6a3ce929d0e0e4736`)
- **span_id**: 16 lowercase hexadecimal characters (e.g., `00f067aa0ba902b7`)

These fields enable correlation with distributed traces per OpenTelemetry/W3C Trace Context standard. Most OpenTelemetry SDKs automatically inject these into logging context when using structured logging.

### Include When Relevant

- **error**: Error type and message
- **stack_trace**: Sanitized stack trace (see sanitization rules below)
- **duration**: For operations that take time (in milliseconds or seconds with unit)
- **input_values**: Sanitized request parameters
- **output_values**: Sanitized response data
- **database_operation**: Connection info, query type, result counts (not full queries)
- **message_queue_operation**: Topics, queues, producer/consumer IDs, message counts
- **user_id**: Internal UUID or database ID (never PII like username, email)
- **session_id**: Internal session identifier

## Stack Trace Sanitization

"Sanitized" means removing information that exposes system internals or sensitive data while preserving diagnostic value.

### Remove from Stack Traces

- **Absolute file paths**: Use relative paths from project root
  - Replace `/home/jsmith/company-app/src/handlers/auth.go:42`
  - With `src/handlers/auth.go:42`
- **Usernames and home directories**: Replace with generic markers
  - Replace `/home/jsmith/`
  - With `$HOME/`
- **Environment variable values**: Remove any values appearing in stack context
- **Memory addresses**: Provide minimal debugging value, expose internal state

### Keep in Stack Traces

- Function and method names
- Line numbers
- Error messages
- Relative file paths from project root
- Call hierarchy showing execution flow

### Additional Sanitization Rules

- Truncate stack traces to **50 frames maximum** to prevent log bloat
- Use structured logging libraries that handle sanitization automatically when available
- For Go, libraries like `github.com/pkg/errors` provide easier programmatic sanitization
- Test sanitization logic to ensure critical debugging information is preserved

## Never Log These

**Non-negotiable restrictions:**

- Passwords or password hashes
- API keys, tokens, or credentials
- Credit card numbers or financial account data
- Social Security Numbers or tax identifiers
- Personal Health Information (PHI)
- Personally Identifiable Information (PII):
  - Full names
  - Email addresses
  - Phone numbers
  - Physical addresses
  - IP addresses (log anonymized versions if needed)
- Full database queries (log query type and parameters only)
- Full request/response bodies (log sanitized versions)
- Session tokens or cookies

## Trace Context Correlation

OpenTelemetry uses two core identifiers to correlate distributed requests:

- **trace_id**: Unique identifier for entire request journey (32 lowercase hex chars)
- **span_id**: Unique identifier for each operation within trace (16 lowercase hex chars)

These ARE your correlation IDs. They tie logs, traces, and metrics together across all services.

### Trace Context Propagation

Services pass trace context via `traceparent` HTTP header following W3C Trace Context standard:

```text
traceparent: {version}-{trace_id}-{span_id}-{trace_flags}
```

Example:

```text
traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01
```

When a service receives this header, it continues the same trace by creating child spans with the same `trace_id` but new `span_id` values.

### Using Trace Context for Investigation

1. Find the relevant log entry (the anomaly needing attention)
2. Pull up the associated trace using the `trace_id` (full context of what happened)
3. Navigate span hierarchy using `span_id` values
4. Understand both what went wrong and why

## Log Output Format

### Containerized Applications

Always log to `stdout`. Let container orchestration platform handle log collection and routing. Do NOT manage log files inside containers.

### Structured Format

Use JSON or another structured format for operational data, not just debugging output.

**Example structured log entry:**

```json
{
  "timestamp": "2025-11-11T14:23:45Z",
  "severity": "warn",
  "action": "login",
  "component": "auth-service",
  "method": "HandleLogin",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "span_id": "00f067aa0ba902b7",
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "error": "invalid_password",
  "duration_ms": 127
}
```

**Benefits of structured logging:**

- Filter by specific fields
- Automated alerting on patterns
- Performance analysis
- Trend detection
- Machine-readable for aggregation tools

## Daily Log Review Requirements

On-call engineers MUST review logs daily as part of standard operational duties.

### Daily Review Focus Areas

1. **Error and Critical logs**: Investigate ALL errors and critical events, even without alerts
2. **Warning patterns**: Look for repeated warnings indicating emerging problems
3. **Anomalies**: Unusual patterns in log volume, timing, or content
4. **Resource warnings**: Memory, CPU, disk, or connection pool warnings indicating capacity issues

### Efficient Review Process

- Start with highest severity (Critical → Error → Warn)
- Look for patterns across services, not individual entries
- Use log aggregation tool filtering and grouping to identify trends
- Document findings in team runbook or incident tracking system
- **Time investment**: 15-30 minutes daily

### Benefits

Daily log review catches problems before they become incidents. Builds operational awareness of normal vs. abnormal behavior. Teams practicing this consistently see far fewer midnight incidents - problems addressed during business hours before escalation.

## Implementation Checklist

Use this checklist when implementing structured logging:

### Setup

- [ ] Configure logging library to output JSON format
- [ ] Set production log level to WARN
- [ ] Configure log output to stdout for containerized apps
- [ ] Integrate OpenTelemetry SDK for automatic trace context injection

### Required Fields

- [ ] All logs include timestamp (UTC ISO 8601)
- [ ] All logs include severity level
- [ ] All logs include action, component, method
- [ ] Trace-aware logs include trace_id and span_id
- [ ] Stack traces are sanitized (no absolute paths, no PII)

### Sanitization

- [ ] Sensitive data never logged (passwords, keys, PII, PHI)
- [ ] Stack traces truncated to 50 frames maximum
- [ ] Database queries logged as types/parameters, not full text
- [ ] Request/response bodies sanitized before logging

### Resource Monitoring

- [ ] Resource utilization warnings use graduated thresholds (75%/80%/90%)
- [ ] Memory, CPU, disk, connection pools monitored
- [ ] Warning level reserved for actionable anomalies

### Operations

- [ ] Daily log review process established
- [ ] On-call engineers review Error/Critical/Warn logs daily
- [ ] Log aggregation tool configured for pattern detection
- [ ] Runbook documents common log patterns and responses

## Validation Rules for Architect Agents

When reviewing logging implementations, enforce these rules:

1. **Log level**: Production default is WARN, not INFO or DEBUG
2. **Trace correlation**: trace_id (32 hex) and span_id (16 hex) present when trace context active
3. **Timestamp format**: UTC ISO 8601 only
4. **Structured format**: JSON or equivalent machine-readable format
5. **Required fields**: timestamp, severity, action, component, method always present
6. **Sanitization**: No PII, PHI, credentials, absolute paths in logs
7. **Stack traces**: Truncated to 50 frames, sanitized paths
8. **Resource thresholds**: 75% warn, 80% error, 90% critical
9. **Output destination**: stdout for containers, not file writes
10. **Daily review**: Process documented and followed

## References

- [OpenTelemetry Traces Documentation](https://opentelemetry.io/docs/concepts/signals/traces/)
- [W3C Trace Context Specification](https://www.w3.org/TR/trace-context/)
- Engineering Handbook: Observability: Logging
- Engineering Handbook: Observability: Distributed Tracing
