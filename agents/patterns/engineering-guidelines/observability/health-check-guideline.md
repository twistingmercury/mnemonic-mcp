---
entity_name: Health Check Endpoint Guideline
entity_type: engineering-guideline
language: agnostic
domain: observability
description: Standard /ops/health endpoint format with dependency checking
tags:
  - health-check
  - observability
  - kubernetes
  - monitoring
---

# Health Check Endpoint Guideline

## Purpose

Every service must expose a standardized health check endpoint at `/ops/health` to enable:

- **Auto-healing** - Kubernetes can restart unhealthy containers
- **Auto-scaling** - Only send traffic to healthy instances
- **Troubleshooting** - Ops teams can quickly identify what's broken
- **Alerting** - Automated monitoring can catch problems before users notice

## Endpoint Specification

### Path

```text
/ops/health
```

### HTTP Method

```text
GET
```

### Response Status Codes

- **200** - Service is healthy
- **503** - Service is unhealthy (Kubernetes will take appropriate action)

## Response Schema

### Required Fields

Every health check response MUST include:

- **Status** (string enum) - Overall health status
- **URL** (string, URI format) - The actual endpoint URL that responded
- **Machine** (string) - Hostname or container name (never IP address)
- **UtcDateTime** (string, ISO 8601 format) - Timestamp when response was generated

### Optional Fields

- **Message** (string) - Optional details about current health condition
- **RequestDuration** (number, minimum 0) - How long the health check took (milliseconds)
- **Dependencies** (array) - Health status of direct dependencies

### Status Enum Values

The `Status` field MUST be one of:

- `NotSet` - Status has not been determined
- `OK` - Service is healthy
- `Warning` - Service is operating but showing concerning signs
- `Critical` - Service is unhealthy or failing

## Health Status Determination

### Resource Utilization Thresholds

Apply these thresholds when checking memory, CPU, disk, database connections, and file descriptors:

| Resource Utilization | Health Status |
| -------------------- | ------------- |
| < 75%                | OK            |
| 75% - 80%            | Warning       |
| > 80%                | Critical      |

### Overall Status Rules

The overall health status should reflect the **worst status** among all checked resources and dependencies.

**Example logic:**

- All resources at <75% and all dependencies OK → Status: `OK`
- Any resource 75-80% or any dependency Warning → Status: `Warning`
- Any resource >80% or any dependency Critical → Status: `Critical`

## Dependency Health Checking

### DependencyHealth Schema

Each dependency entry MUST include:

- **Status** (string enum) - Health status of the dependency (NotSet, OK, Warning, Critical)
- **URL** (string, URI format) - Endpoint checked for this dependency
- **UtcDateTime** (string, ISO 8601 format) - Timestamp when dependency was checked

### Optional Dependency Fields

- **RequestDuration** (number, minimum 0) - How long the dependency check took (milliseconds)

### What to Check

Include health status for:

- **Databases** - Connection pool status, query response time
- **External APIs** - Connectivity and response time
- **Message queues** - Connection status, queue depth
- **Caches** - Connection status, memory usage

## Example Response

### Healthy Service

```json
{
  "Status": "OK",
  "URL": "https://api.example.com/ops/health",
  "Machine": "pod-api-7d9f8c6b5-xk4m2",
  "UtcDateTime": "2026-02-02T15:30:45Z",
  "RequestDuration": 12.5,
  "Dependencies": [
    {
      "Status": "OK",
      "URL": "postgresql://db.example.com:5432/maindb",
      "UtcDateTime": "2026-02-02T15:30:45Z",
      "RequestDuration": 8.3
    },
    {
      "Status": "OK",
      "URL": "https://cache.example.com:6379",
      "UtcDateTime": "2026-02-02T15:30:45Z",
      "RequestDuration": 2.1
    }
  ]
}
```

### Service with Warning

```json
{
  "Status": "Warning",
  "Message": "Memory utilization at 78%",
  "URL": "https://api.example.com/ops/health",
  "Machine": "pod-api-7d9f8c6b5-xk4m2",
  "UtcDateTime": "2026-02-02T15:30:45Z",
  "RequestDuration": 15.2,
  "Dependencies": [
    {
      "Status": "OK",
      "URL": "postgresql://db.example.com:5432/maindb",
      "UtcDateTime": "2026-02-02T15:30:45Z",
      "RequestDuration": 10.5
    },
    {
      "Status": "OK",
      "URL": "https://cache.example.com:6379",
      "UtcDateTime": "2026-02-02T15:30:45Z",
      "RequestDuration": 2.3
    }
  ]
}
```

### Service with Critical Dependency

```json
{
  "Status": "Critical",
  "Message": "Database connection failed",
  "URL": "https://api.example.com/ops/health",
  "Machine": "pod-api-7d9f8c6b5-xk4m2",
  "UtcDateTime": "2026-02-02T15:30:45Z",
  "RequestDuration": 5005.7,
  "Dependencies": [
    {
      "Status": "Critical",
      "URL": "postgresql://db.example.com:5432/maindb",
      "UtcDateTime": "2026-02-02T15:30:45Z",
      "RequestDuration": 5000.0
    },
    {
      "Status": "OK",
      "URL": "https://cache.example.com:6379",
      "UtcDateTime": "2026-02-02T15:30:45Z",
      "RequestDuration": 2.1
    }
  ]
}
```

## Kubernetes Integration

### Unified Probe Configuration

While Kubernetes supports separate readiness and liveness probes, use a single `/ops/health` endpoint for both. This simplifies implementation while providing Kubernetes with the information it needs through HTTP status codes.

### Example Kubernetes Configuration

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: api-service
spec:
  containers:
    - name: api
      image: api-service:latest
      livenessProbe:
        httpGet:
          path: /ops/health
          port: 8080
        initialDelaySeconds: 30
        periodSeconds: 10
        timeoutSeconds: 5
        failureThreshold: 3
      readinessProbe:
        httpGet:
          path: /ops/health
          port: 8080
        initialDelaySeconds: 10
        periodSeconds: 5
        timeoutSeconds: 3
        failureThreshold: 2
```

## Implementation Notes

### Performance Considerations

- Health checks should complete quickly (target <100ms)
- Avoid expensive operations (complex queries, external API calls)
- Cache dependency checks if they're slow (refresh every 10-30 seconds)
- Set reasonable timeouts for dependency checks (2-5 seconds)

### Error Handling

- If a dependency check times out, mark it as `Critical`
- If a resource metric cannot be determined, mark it as `NotSet`
- Always return a valid JSON response, even during failures
- Include a descriptive `Message` field when status is Warning or Critical

### Go Package Reference

The status values align with the [twistingmercury/heartbeat](https://github.com/twistingmercury/heartbeat) Go package, which provides a standard implementation of this pattern.

## Related Guidelines

- [Observability: Logging](../logging/logging-guideline.md) - For resource limit threshold logging
- [Observability: Metrics](../metrics/metrics-guideline.md) - For exposing health metrics
- [Scale & High Availability](../scale/high-availability-guideline.md) - For auto-scaling integration
