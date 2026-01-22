# Mnemonic REST API Design

[Back to Architecture Overview](../../architecture/00-overview.md) | [Back to Project README](../../../README.md)

## Table of Contents

- [API Specification](#api-specification)
- [Design Decisions](#design-decisions)
  - [POST vs GET for Routing](#post-vs-get-for-routing)
  - [Single Routing Endpoint](#single-routing-endpoint)
  - [Cursor-Based Pagination](#cursor-based-pagination)
  - [Large Payload Handling](#large-payload-handling)
- [Error Handling Philosophy](#error-handling-philosophy)
- [References](#references)

## API Specification

[Table of Contents](#table-of-contents)

The complete API specification is defined in OpenAPI 3.1 format:

**[api/openapi/mnemonic-v1.yaml](../../../api/openapi/mnemonic-v1.yaml)** (Authoritative Source)

This document describes the design rationale behind API decisions. For complete endpoint definitions, request/response schemas, and authentication requirements, consult the OpenAPI specification.

## Design Decisions

[Table of Contents](#table-of-contents)

> **Architecture Reference:** [Architectural Decisions](../../architecture/02-architectural-decisions.md) | [Communication Patterns - Request Flow](../../architecture/04-communication-patterns.md#request-flow)

### POST vs GET for Routing

**Decision**: Use POST for `/v1/api/route` instead of GET.

**Rationale**:

1. Prompts can be large (up to 10KB) - exceeds practical URL length limits
2. Request body allows structured context and options
3. Semantically, routing performs processing rather than simple retrieval
4. Allows consistent JSON request/response pattern

### Single Routing Endpoint

**Decision**: Combine routing + agent + patterns in single response.

**Rationale** (per architecture requirement):

1. Minimizes round trips for primary flow
2. CLI can make single call instead of 3 sequential calls
3. Server can optimize internal queries
4. Reduces latency for the critical path

The `/v1/api/route` endpoint returns:

- Routing decision (which agent was selected and why)
- Full agent definition (including system prompt)
- Relevant patterns (with relevance scores)
- Performance metadata (timing information)

### Cursor-Based Pagination

**Decision**: Use cursor-based pagination instead of offset.

**Rationale**:

1. More efficient for large datasets
2. Consistent results when data changes between pages
3. No risk of skipping/duplicating items with concurrent modifications

Cursors are opaque base64-encoded strings that expire after 24 hours.

### Large Payload Handling

**Considerations for large system_prompt and pattern content**:

1. List endpoints exclude large fields (`system_prompt`, `content`)
2. Detail endpoints include full content
3. Response compression (gzip) should be enabled at Envoy

## Error Handling Philosophy

[Table of Contents](#table-of-contents)

> **Architecture Reference:** [Communication Patterns - Error Handling](../../architecture/04-communication-patterns.md#error-handling)

Errors follow [RFC 7807 Problem Details](https://tools.ietf.org/html/rfc7807) format with content type `application/problem+json`.

**Why RFC 7807**:

1. Standard format across all endpoints
2. Machine-readable error codes for client handling
3. Human-readable messages for debugging
4. Extensible for field-level validation errors
5. `traceId` field enables log correlation

**Key principles**:

- Every error includes a correlation `traceId` for debugging
- Validation errors include field-level details in `errors` array
- Error `type` URIs are stable and can be used for client logic
- HTTP status codes follow standard semantics

**Post-MVP Features**:

- Rate limiting (429 responses): Server-side rate limiting will be available in a later phase. The OpenAPI spec defines the response format for forward compatibility, but rate limiting is not enforced in MVP.

See the OpenAPI spec for complete error schemas and example responses.

## References

[Table of Contents](#table-of-contents)

- [OpenAPI Specification](../../../api/openapi/mnemonic-v1.yaml) - Source of truth for API details
- [Pattern Processing](pattern-processing.md) - Enrichment pipeline details
- [Architectural Decisions](../../architecture/02-architectural-decisions.md)
