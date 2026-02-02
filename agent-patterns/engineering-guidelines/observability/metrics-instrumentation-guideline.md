---
entity_name: Metrics Instrumentation Guideline
entity_type: engineering-guideline
language: agnostic
domain: observability
description: Prometheus metrics standards using RED and USE methods
tags:
  - metrics
  - prometheus
  - observability
  - monitoring
---

# Metrics Instrumentation Guideline

## Scope: When to Expose Metrics

### MUST Expose Metrics For

**Hosted services** - APIs, web applications, microservices running 24/7:

- Request rates and response times
- Error rates and types
- Resource usage (CPU, memory, connections)

**Background workers and processors**:

- Job processing rates
- Queue depths and lag
- Processing durations
- Failure rates

**Rule of thumb**: If it runs continuously and serves production traffic, it needs metrics.

### Do NOT Expose Metrics For

- CLI tools (they're not long-running)
- One-off scripts (they're not long-running processes)
- Development/test utilities (they're not production workloads)

## What to Measure: RED and USE Methods

### RED Method (Request-Driven Services)

For services that handle requests (APIs, web apps), measure:

- **Rate** - Requests per second (how busy are we?)
- **Errors** - Failed requests per second (what's breaking?)
- **Duration** - Response time distribution (how fast are we?)

These three metrics give you a complete picture of service health.

### USE Method (Resource Monitoring)

For system resources (CPU, memory, disk, network), measure:

- **Utilization** - Percentage of resource capacity used
- **Saturation** - How much work is queued waiting for the resource
- **Errors** - Resource-related errors (OOM kills, disk errors, etc.)

### Business Metrics

Don't forget the metrics that matter to the business:

- Active users or sessions
- Transactions processed
- Revenue per hour
- Conversion rates
- Feature usage

## Prometheus Implementation Standards

### Endpoint Requirements

**MUST** expose metrics at `/metrics` in Prometheus exposition format:

```text
# HELP http_requests_total Total number of HTTP requests
# TYPE http_requests_total counter
http_requests_total{method="GET",status="200"} 1234
http_requests_total{method="POST",status="201"} 567

# HELP http_request_duration_seconds HTTP request duration
# TYPE http_request_duration_seconds histogram
http_request_duration_seconds_bucket{le="0.1"} 100
http_request_duration_seconds_bucket{le="0.5"} 450
http_request_duration_seconds_bucket{le="1.0"} 980
http_request_duration_seconds_sum 523.4
http_request_duration_seconds_count 1000
```

### Metric Types

**Counter** - Monotonically increasing value (resets on restart):

- Request counts
- Error counts
- Bytes processed
- **MUST** end with `_total` suffix

**Gauge** - Value that can go up or down:

- Current memory usage
- Active connections
- Queue depth
- Temperature

**Histogram** - Distribution of values in configurable buckets:

- Request duration
- Response size
- Batch sizes
- **MUST** end with appropriate unit suffix (`_seconds`, `_bytes`)

**Summary** - Similar to histogram but calculates quantiles on client:

- Use sparingly - histograms are usually better
- More expensive computationally

## Naming Conventions

### MUST Follow These Rules

**Use snake_case** for all metric names:

- ✅ `http_requests_total`
- ❌ `HttpRequests` or `http-requests`

**Start with application or domain name**:

- ✅ `api_http_requests_total`
- ✅ `worker_job_duration_seconds`
- ❌ `requests` (too generic)

**End counters with `_total`**:

- ✅ `http_requests_total`
- ✅ `errors_total`
- ❌ `http_requests` (missing suffix)

**End duration metrics with `_seconds`**:

- ✅ `request_duration_seconds`
- ✅ `job_processing_seconds`
- ❌ `duration` (missing unit)
- ❌ `request_duration_ms` (wrong unit - always use seconds)

**End size metrics with `_bytes`**:

- ✅ `response_size_bytes`
- ✅ `message_payload_bytes`
- ❌ `response_size` (missing unit)
- ❌ `response_size_kb` (wrong unit - always use bytes)

### Good vs Bad Examples

**Good names**:

- `api_http_requests_total`
- `worker_job_duration_seconds`
- `cache_hits_total`
- `response_size_bytes`

**Bad names**:

- `RequestCount` (not snake_case)
- `duration` (too vague, missing unit)
- `http_requests` (counters need `_total` suffix)
- `response_time_ms` (wrong unit - use seconds)

## Labels and Cardinality

### MUST Follow Cardinality Rules

**Labels should have bounded, finite values**:

- ✅ `method="GET"` (finite HTTP methods)
- ✅ `status="200"` (finite status codes)
- ✅ `endpoint="/api/users"` (limited set of endpoints)
- ✅ `service="api-gateway"` (bounded service names)
- ❌ `user_id="12345"` (unbounded - millions of users)
- ❌ `email="user@example.com"` (unbounded)
- ❌ `session_id="abc123"` (unbounded)
- ❌ `request_id="xyz789"` (unbounded)

### Cardinality Explosion Prevention

**The Problem**: Every unique label combination creates a new time series.

If you have:

- 10 methods × 20 status codes × 50 endpoints = 10,000 time series ✅ Acceptable
- 1 million user IDs as labels = 1,000,000 time series ❌ Will kill Prometheus

**The Solution**:

- Keep label cardinality bounded (status codes: ~20 values, not user IDs: 1,000,000 values)
- Never use IDs or unbounded strings as labels
- Use sampling or aggregation for high-cardinality data
- Monitor your metrics system's resource usage

**Safe label types**:

- HTTP methods (GET, POST, PUT, DELETE, etc.)
- Status codes (200, 404, 500, etc.)
- Endpoint paths (finite set)
- Service names (bounded)
- Environment (dev, staging, prod)
- Region/zone (bounded)

**Dangerous label types**:

- User IDs
- Email addresses
- Session IDs
- Request IDs
- Transaction IDs
- Any user-generated content

## Performance Thresholds for Tracing

When implementing distributed tracing alongside metrics, use these thresholds to determine what to trace:

| Operation Type             | Threshold | Why                                      |
| -------------------------- | --------- | ---------------------------------------- |
| User-facing API response   | >200ms    | Impacts user experience                  |
| Database query             | >100ms    | Indicates slow query or missing index    |
| Cache operation (Redis)    | >10ms     | Cache should be fast - this is too slow  |
| External API call          | >500ms    | Third-party dependency performance       |
| Memory-intensive operation | >50MB     | Indicates potential memory issues        |
| All errors                 | Always    | Errors always need investigation context |

These thresholds help correlate metrics alerts with trace data for investigation.

## Critical Metrics Checklist

### MUST Implement These for Request-Driven Services

- ✅ Request rate (`http_requests_total` counter)
- ✅ Error rate (`http_errors_total` counter or status code breakdown)
- ✅ Response time distribution (`http_request_duration_seconds` histogram with p50, p95, p99)
- ✅ Resource usage (CPU, memory gauges)
- ✅ Active connections (gauge)

### MUST Implement These for Background Workers

- ✅ Job processing rate (`jobs_processed_total` counter)
- ✅ Job processing duration (`job_duration_seconds` histogram)
- ✅ Queue depth (`queue_depth` gauge)
- ✅ Job failures (`job_failures_total` counter)
- ✅ Resource usage (CPU, memory gauges)

### SHOULD Implement Business Metrics

- Active users/sessions (gauge)
- Transactions processed (counter)
- Feature usage (counter)
- Conversion rates (gauge)

## Common Pitfalls to Avoid

### ❌ Cardinality Explosion

**Don't**: Add labels with unbounded values

```text
http_requests_total{user_id="12345"}  # Creates millions of time series
```

**Do**: Use bounded labels and aggregate high-cardinality data

```text
http_requests_total{method="GET", status="200"}  # Bounded values
```

### ❌ Over-Instrumenting

**Don't**: Measure everything just because you can

Every metric has a cost:

- Memory in your application
- Network bandwidth to send metrics
- Storage in Prometheus
- Query time on dashboards

**Do**: Start with RED/USE methods, then add business metrics as needed

### ❌ Under-Instrumenting

**Don't**: Skip critical metrics because "we'll add them later"

**Do**: If you're on-call and can't answer "Is the service healthy?" from your dashboards, you're under-instrumented

### ❌ Wrong Units

**Don't**: Use milliseconds, kilobytes, or other non-standard units

```text
request_duration_ms        # Wrong unit
response_size_kb           # Wrong unit
```

**Do**: Always use base units (seconds, bytes)

```text
request_duration_seconds   # Correct
response_size_bytes        # Correct
```

### ❌ Missing Suffixes

**Don't**: Omit required suffixes

```text
http_requests              # Missing _total
request_duration           # Missing _seconds
```

**Do**: Use standard suffixes

```text
http_requests_total        # Correct
request_duration_seconds   # Correct
```

## Storage Cost Awareness

With default Prometheus settings:

- 1,000 active time series = ~1-2 MB/hour of storage
- 1,000,000 active time series = ~1-2 GB/hour of storage

High cardinality labels can quickly make metrics expensive. Monitor your time series count and set retention policies appropriately.

## Integration with Observability Stack

Metrics work together with logs and traces for complete observability:

**Metrics answer**: "What's happening now?"

- Current request rate, error rate, latency
- Resource utilization trends
- System health at a glance
- Trigger alerts when thresholds are crossed

**Logs answer**: "What happened and why?"

- Detailed error messages and stack traces
- Contextual information about specific events
- Audit trails and security events

**Traces answer**: "Where did time go?"

- Full request journey across services
- Where did the error originate?
- Dependencies and timing breakdown

## Example Investigation Workflow

1. **Alert fires** (from metrics): "Error rate > 5%"
2. **Check metrics dashboard**: Which endpoint? What status codes?
3. **Search logs**: What errors are being logged? Any patterns?
4. **Pull traces**: For failed requests, what's the full story?
5. **Root cause found**: Database connection timeout on specific query

Each layer provides different insight. Metrics trigger alerts, logs provide details, traces show the full picture.

## References

- [Prometheus Best Practices](https://prometheus.io/docs/practices/naming/)
- [The RED Method](https://grafana.com/blog/2018/08/02/the-red-method-how-to-instrument-your-services/)
- [The USE Method](https://www.brendangregg.com/usemethod.html)
- Engineering Handbook: Observability: Metrics
- Engineering Handbook: Observability: Distributed Tracing
- Engineering Handbook: Observability: Logging
