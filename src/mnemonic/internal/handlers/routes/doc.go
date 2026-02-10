// Package routes provides the HTTP handler for prompt routing requests.
// It registers the POST /api/route endpoint that accepts a prompt and returns
// a routing decision identifying the appropriate agent.
//
// Documentation:
//   - Architecture: docs/architecture/03-system-architecture.md (Component Breakdown)
//   - Architecture: docs/architecture/04-communication-patterns.md (Request Flow)
//   - Design: docs/design/api-specification.md (POST vs GET for Routing)
package routes
