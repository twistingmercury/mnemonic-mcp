// Package agent provides the PostgreSQL repository implementation for agent
// persistence. It implements CRUD operations, pagination, and existence checks
// for agent definitions, with JSONB handling for allowed tools and routing
// keywords.
//
// Documentation:
//   - Architecture: docs/architecture/08-data-architecture.md (Data Model Design > Agents, Data Flow Patterns > Write Paths)
//   - Design: docs/design/data-storage.md (Repository Interfaces > AgentRepository)
package agent
