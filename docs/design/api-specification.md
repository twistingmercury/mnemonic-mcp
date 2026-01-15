# Mnemonic REST API Design

[Back to Architecture Overview](../architecture/00-overview.md) | [Back to Project README](../../README.md)

## Table of Contents

- [Overview](#overview)
- [API Specification](#api-specification)
- [Design Decisions](#design-decisions)
  - [POST vs GET for Routing](#post-vs-get-for-routing)
  - [Single Routing Endpoint](#single-routing-endpoint)
  - [Cursor-Based Pagination](#cursor-based-pagination)
  - [Large Payload Handling](#large-payload-handling)
- [Authentication Contract](#authentication-contract)
- [Error Handling Philosophy](#error-handling-philosophy)
- [References](#references)

## Overview

Mnemonic is the backend server for ACE, providing routing decisions and pattern retrieval via REST API. For MVP, Mnemonic serves only ACE (see [ADR-004](../architecture/02-architectural-decisions.md#adr-004-unified-backend-with-rest-api)).

The API follows REST principles with:

- URL path versioning (`/v1/`)
- JSON request/response bodies
- RFC 7807 Problem Details for errors
- Cursor-based pagination for list endpoints

## API Specification

The complete API specification is defined in OpenAPI 3.1 format:

**[api/openapi/mnemonic-v1.yaml](../../api/openapi/mnemonic-v1.yaml)**

This file is the source of truth for:

- All endpoint definitions and paths
- Request/response schemas
- Error formats and status codes
- Authentication requirements
- Pagination parameters
- Validation constraints

Use this OpenAPI spec to:

- Generate client SDKs
- Validate requests/responses during development
- Generate API documentation
- Configure API gateways

## Design Decisions

### POST vs GET for Routing

**Decision**: Use POST for `/ace/route` instead of GET.

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

The `/ace/route` endpoint returns:

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

## Authentication Contract

Authentication is handled by Envoy proxy (outside Mnemonic scope per architecture). Mnemonic expects pre-validated identity information in request headers.

**Headers from Envoy**:

| Header         | Required | Description                                     |
| -------------- | -------- | ----------------------------------------------- |
| `X-User-ID`    | Yes      | Authenticated user identifier (UUID)            |
| `X-Team-ID`    | Yes      | Team/organization identifier (UUID)             |
| `X-User-Roles` | No       | Comma-separated roles (e.g., `admin,developer`) |

**Authorization Rules**:

| Operation                          | Required Role          |
| ---------------------------------- | ---------------------- |
| `POST /ace/route`                  | Any authenticated user |
| `GET /ace/patterns`                | Any authenticated user |
| `GET /ace/agents`                  | Any authenticated user |
| Admin operations (PUT/DELETE)      | `admin` role           |

**Why headers instead of token validation**:

1. Security handled at edge (Envoy) per architecture decision
2. Mnemonic stays lightweight and focused on routing/patterns
3. Clear separation of concerns
4. Headers provide all needed identity information

## Error Handling Philosophy

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

See the OpenAPI spec for complete error schemas and example responses.

## References

- [OpenAPI Specification](../../api/openapi/mnemonic-v1.yaml) - Source of truth for API details
- [Architecture Overview](../architecture/00-overview.md)
- [Architectural Decisions](../architecture/02-architectural-decisions.md)
- [System Architecture](../architecture/03-system-architecture.md)
- [Communication Patterns](../architecture/04-communication-patterns.md)
- [RFC 7807 - Problem Details for HTTP APIs](https://tools.ietf.org/html/rfc7807)
- [OpenAPI Specification 3.1](https://spec.openapis.org/oas/v3.1.0)
