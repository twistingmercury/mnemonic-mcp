---
entity_name: Distributed Tracing Guideline
entity_type: engineering-guideline
language: agnostic
domain: observability
description: OpenTelemetry tracing standards with sampling rates
tags:
  - tracing
  - opentelemetry
  - observability
  - distributed-systems
---

# Distributed Tracing Guideline

## Overview

This guideline defines standards for implementing distributed tracing using OpenTelemetry to track requests across microservices. It covers trace context formats, sampling strategies, and correlation requirements.

## Trace Context Standards

### Trace Identifiers

**Trace ID Format:**

- **Length**: 32 lowercase hexadecimal characters
- **Purpose**: Unique identifier for the entire request journey across all services
- **Example**: `4bf92f3577b34da6a3ce929d0e0e4736`

**Span ID Format:**

- **Length**: 16 lowercase hexadecimal characters
- **Purpose**: Unique identifier for each operation within a trace
- **Example**: `00f067aa0ba902b7`

### Context Propagation

**Standard**: Follow [W3C Trace Context](https://www.w3.org/TR/trace-context/) specification

**HTTP Header**: `traceparent`

**Format**:

```text
{version}-{trace_id}-{span_id}-{trace_flags}
```

**Example**:

```text
traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01
```

**Implementation Requirements:**

- Services MUST propagate trace context when making downstream calls
- When receiving a `traceparent` header, continue the same trace by creating child spans
- Child spans MUST use the same `trace_id` but generate new `span_id` values
- This creates parent-child relationships that build the complete trace tree

## Sampling Strategies

### Traffic-Based Sampling Rates

Adjust sampling rates based on request volume to balance observability with infrastructure costs:

| Traffic Volume (req/min) | Sampling Rate | Notes                     |
| ------------------------ | ------------- | ------------------------- |
| < 1,000                  | 10%           | Default for low traffic   |
| 1,000 - 10,000           | 1-5%          | Medium traffic            |
| > 10,000                 | 0.1-1%        | High traffic              |
| **Error traces**         | **100%**      | **Always capture errors** |

### Critical Rules

**MUST ALWAYS capture:**

- All traces containing errors (100% sampling rate)
- Traces with errors take precedence over traffic-based sampling

**Sampling Types:**

- **Head-based sampling**: Decision made at trace start (recommended for most cases)
- **Tail-based sampling**: Decision after complete trace (useful for capturing all error traces)

### Review Cadence

**Monthly**: Review sampling rates under normal operations

**Post-Release**: Review sampling effectiveness daily for 5 days after releases to ensure:

- Sufficient data capture for issue identification
- Tracing infrastructure is not overwhelmed

## What to Trace

### Always Trace

**Service Boundaries:**

- HTTP endpoints (all incoming requests)
- gRPC methods
- Message queue handlers
- Background jobs

### Trace When Meeting Thresholds

Use these as starting points; adjust based on service performance profile:

| Operation Type             | Threshold | Reference                                                                                                              |
| -------------------------- | --------- | ---------------------------------------------------------------------------------------------------------------------- |
| User-facing API response   | > 200ms   | [Google SRE Workbook](https://sre.google/workbook/implementing-slos/)                                                  |
| Database query             | > 100ms   | [PostgreSQL Wiki](https://wiki.postgresql.org/wiki/Logging_Difficult_Queries)                                          |
| Cache operation (Redis)    | > 10ms    | [Redis Latency Docs](https://redis.io/docs/latest/operate/oss_and_stack/management/optimization/latency-monitor/)      |
| External API call          | > 500ms   | [AWS Well-Architected](https://docs.aws.amazon.com/wellarchitected/latest/framework/ops_observability_dist_trace.html) |
| Memory-intensive operation | > 50MB    | Industry practice                                                                                                      |
| **All errors**             | **100%**  | [OpenTelemetry Best Practices](https://opentelemetry.io/docs/concepts/sampling/)                                       |

### Don't Over-Trace

- Tracing adds overhead
- Don't trace every function call
- Focus on operations that matter for debugging and performance monitoring

## Span Naming Standards

**DO:**

- Use descriptive, consistent names: `GET /users/{id}`
- Use path templates, not actual values

**DON'T:**

- Use actual parameter values: `GET /users/12345`

## Integration with Logging

### Correlation Fields

Most OpenTelemetry SDKs automatically inject trace context into structured logs:

**Required Fields:**

- `trace_id` (32 lowercase hex characters)
- `span_id` (16 lowercase hex characters - when trace context is active)

**Purpose:**

1. Find relevant log entry (the anomaly)
2. Pull up associated trace using `trace_id` (full context)
3. Understand what went wrong and why

This enables complete observability while keeping costs reasonable - traces capture successful request flow efficiently, logs capture exceptions and warnings.

## Implementation Requirements

### OpenTelemetry SDK

**Auto-Instrumentation**: Use OpenTelemetry auto-instrumentation libraries when available for:

- HTTP clients/servers
- Database clients
- Message queue clients
- Common frameworks

**Manual Instrumentation**: Add spans for:

- Business-critical operations
- Custom logic that needs tracking
- Operations meeting threshold criteria

### Export Configuration

- Configure exporters for your observability backend (Jaeger, Zipkin, vendor-specific)
- Ensure trace data is exported to centralized storage
- Configure appropriate retention policies

## References

- [OpenTelemetry Documentation](https://opentelemetry.io/docs/)
- [OpenTelemetry Traces Concepts](https://opentelemetry.io/docs/concepts/signals/traces/)
- [W3C Trace Context Standard](https://www.w3.org/TR/trace-context/)
- [OpenTelemetry Sampling Best Practices](https://opentelemetry.io/docs/concepts/sampling/)
- [OpenTelemetry Instrumentation Guides](https://opentelemetry.io/docs/instrumentation/)

## Architect Agent Usage

When designing observability for services:

1. Specify OpenTelemetry SDK integration in service dependencies
2. Define sampling strategy based on expected traffic volume
3. Identify service boundaries and operations requiring explicit instrumentation
4. Ensure trace context propagation for all inter-service communication
5. Configure trace-to-log correlation in structured logging setup
6. Plan for 100% error trace capture regardless of normal sampling rate
