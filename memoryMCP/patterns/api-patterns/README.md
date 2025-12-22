# API Architecture Patterns

This directory has comprehensive patterns for designing RESTful, GraphQL, and gRPC APIs in Go projects.

## Overview

These patterns give you complete, production-ready templates that Claude Code agents can pull from Cognee when they're designing APIs.

## Pattern Categories

### OpenAPI Patterns (`openapi/`)

REST API specs using OpenAPI 3.1:

1. **REST API Specification Pattern** (`rest-api-specification-pattern.md`)

   - Complete CRUD operations
   - Pagination and filtering
   - API versioning
   - Comprehensive error handling
   - Standard response codes
   - Request/response schemas

2. **Authentication Patterns** (`authentication-patterns.md`)
   - JWT Bearer tokens
   - API key authentication
   - OAuth 2.0 flows (Authorization Code, Client Credentials, Password)
   - OpenID Connect
   - Basic authentication
   - Multiple authentication schemes

**When to use these:**

- RESTful microservices
- Public APIs
- Integration APIs
- Mobile/web backends

### GraphQL Patterns (`graphql/`)

GraphQL schemas using gqlgen:

1. **Schema Pattern** (`schema-pattern.md`)

   - Complete schema with queries, mutations, subscriptions
   - Relay cursor connections for pagination
   - Interfaces and union types
   - Custom directives (auth, rate limiting)
   - DataLoader pattern for N+1 prevention
   - Error handling strategies

2. **Federation Pattern** (`federation-pattern.md`)
   - Apollo Federation v2 setup
   - Multiple subgraph schemas
   - Entity resolution across services
   - Gateway configuration
   - Rover CLI usage
   - Reference resolvers

**When to use these:**

- Unified API gateway
- Microservices federation
- Real-time applications
- Complex data relationships

### gRPC Patterns (`grpc/`)

Protocol Buffer service definitions:

1. **Service Definition Pattern** (`service-definition-pattern.md`)

   - Complete service with all RPC types
   - Message definitions
   - Enums and nested types
   - Field masks for partial updates
   - Error handling with status codes
   - Pagination patterns

2. **Streaming Patterns** (`streaming-patterns.md`)
   - Server streaming RPCs
   - Client streaming RPCs
   - Bidirectional streaming RPCs
   - Practical examples (file transfer, chat, metrics)
   - Flow control and backpressure
   - Connection lifecycle

**When to use these:**

- Service-to-service communication
- Real-time data streaming
- High-performance APIs
- Binary protocol requirements

## Cognee Integration

All patterns get loaded into Cognee's knowledge graph so architecture agents can grab them easily:

```bash
# # OpenAPI patterns
# search(search_query="REST API specification pattern", search_type="GRAPH_COMPLETION")
# search(search_query="OpenAPI authentication patterns", search_type="GRAPH_COMPLETION")

# # GraphQL patterns
# search(search_query="GraphQL schema pattern", search_type="GRAPH_COMPLETION")
# search(search_query="GraphQL federation pattern", search_type="GRAPH_COMPLETION")

# # gRPC patterns
# search(search_query="gRPC service definition pattern", search_type="GRAPH_COMPLETION")
# search(search_query="gRPC streaming patterns", search_type="GRAPH_COMPLETION")
```

## Related Agents

These Claude Code agents use these patterns:

- [go-openapi-architect](../../subagents/go/go-openapi-architect.md) - REST API/OpenAPI design
- [go-graphql-architect](../../subagents/go/go-graphql-architect.md) - GraphQL schema design
- [go-grpc-architect](../../subagents/go/go-grpc-architect.md) - gRPC service design

## Pattern Structure

Each pattern file contains:

1. **Frontmatter**

   - `entity_name`: Pattern name for Cognee
   - `entity_type`: Pattern category
   - `description`: Pattern purpose

2. **Complete Specification**

   - Full OpenAPI YAML, GraphQL schema, or Protocol Buffer definition
   - Production-ready examples
   - All major features demonstrated

3. **Implementation Examples**

   - Go server code
   - Client code
   - Middleware/interceptors
   - Helper functions

4. **Configuration**

   - Tool-specific configuration (gqlgen.yml, buf.yaml, etc.)
   - Code generation commands
   - Build and validation steps

5. **Best Practices**
   - Pattern-specific recommendations
   - Performance considerations
   - Security guidelines
   - Testing strategies

## Code Generation Tools

### OpenAPI (oapi-codegen)

```bash
# Install
go install github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen@latest

# Generate server
oapi-codegen -package api -generate types,server,spec openapi.yaml > api/generated.go

# Generate client
oapi-codegen -package client -generate types,client openapi.yaml > client/generated.go
```

### GraphQL (gqlgen)

```bash
# Install
go install github.com/99designs/gqlgen@latest

# Initialize
gqlgen init

# Generate
gqlgen generate
```

### gRPC (buf)

```bash
# Install
go install github.com/bufbuild/buf/cmd/buf@latest

# Initialize
buf config init

# Generate
buf generate

# Lint
buf lint

# Breaking changes
buf breaking --against '.git#branch=main'
```

## API Design Principles

### RESTful APIs

- Use nouns for resources, not verbs
- Leverage HTTP methods (GET, POST, PUT, PATCH, DELETE)
- Version in URL path (`/v1/`, `/v2/`)
- Use meaningful HTTP status codes
- Implement HATEOAS when it makes sense

### GraphQL APIs

- Design schema-first
- Use descriptive type and field names
- Implement pagination (Relay connections)
- Handle N+1 queries with DataLoader
- Use subscriptions for real-time updates

### gRPC APIs

- Define clear service boundaries
- Use streaming for large data sets
- Implement proper error codes
- Version proto files carefully
- Think about backward compatibility

## Testing

### OpenAPI

```bash
# Validate spec
docker run --rm -v "${PWD}:/local" openapitools/openapi-generator-cli validate -i /local/openapi.yaml

# Test with Postman/Insomnia
# Import OpenAPI spec and run collections
```

### GraphQL

```bash
# Test with GraphQL Playground
# Available at http://localhost:8080/graphql

# Test queries
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ user(id: \"123\") { name email } }"}'
```

### gRPC

```bash
# Test with grpcurl
grpcurl -plaintext localhost:9090 list
grpcurl -plaintext localhost:9090 user.v1.UserService/GetUser

# Test with evans (interactive)
evans -p 9090 -r repl
```

## Performance Considerations

### OpenAPI/REST

- Implement response caching (ETag, Last-Modified)
- Use pagination for large collections
- Support field selection (sparse fieldsets)
- Enable compression (gzip, brotli)
- Rate limit per client

### GraphQL

- Implement query complexity analysis
- Set maximum query depth
- Use persisted queries
- Enable automatic persisted queries (APQ)
- Cache at field level

### gRPC

- Use streaming for large payloads
- Implement connection pooling
- Enable keep-alive
- Use interceptors for logging/metrics
- Consider load balancing strategies

## Security Best Practices

### Authentication & Authorization

- Use JWT for user authentication
- Implement API key rotation
- Support OAuth 2.0 for third parties
- Use TLS/HTTPS only
- Validate all inputs

### Rate Limiting

- Implement per-user rate limits
- Use sliding window algorithm
- Return `429 Too Many Requests`
- Include rate limit headers

### Data Validation

- Validate at API boundary
- Use schema validation
- Sanitize inputs
- Validate content types
- Check request sizes

## Documentation

### OpenAPI

- Generate interactive docs with Swagger UI
- Include examples for all endpoints
- Document error responses
- Provide authentication guides

### GraphQL

- Use GraphQL Playground or GraphiQL
- Document with schema descriptions
- Include query examples
- Explain subscription usage

### gRPC

- Generate docs with protoc-gen-doc
- Include service descriptions
- Document error codes
- Provide client examples

## Version Control

### API Versioning Strategy

1. **URL Versioning** (REST): `/v1/users`, `/v2/users`
2. **Header Versioning** (REST): `Accept: application/vnd.api.v2+json`
3. **Schema Versioning** (GraphQL): Deprecate fields, add new fields
4. **Package Versioning** (gRPC): `user.v1`, `user.v2`

### Breaking Changes

- Major version bump for breaking changes
- Maintain previous version for transition period
- Document migration guides
- Provide deprecation notices

## Monitoring & Observability

### Metrics to Track

- Request rate
- Error rate
- Response times (p50, p95, p99)
- Request payload sizes
- Active connections (WebSocket, gRPC streams)

### Logging

- Log all errors
- Include request IDs
- Log slow queries
- Mask sensitive data

### Tracing

- Implement OpenTelemetry
- Trace across service boundaries
- Include spans for database queries
- Track external API calls

## Contributing

When you're adding new patterns:

1. Use the frontmatter format
2. Include complete, working examples
3. Add Go implementation code
4. Document configuration requirements
5. Provide testing instructions
6. Update this README

## Additional Resources

- [OpenAPI Specification](https://spec.openapis.org/oas/latest.html)
- [GraphQL Documentation](https://graphql.org/)
- [gRPC Documentation](https://grpc.io/docs/)
- [Protocol Buffers Guide](https://protobuf.dev/)
- [Apollo Federation](https://www.apollographql.com/docs/federation/)
- [Buf Documentation](https://buf.build/docs)
