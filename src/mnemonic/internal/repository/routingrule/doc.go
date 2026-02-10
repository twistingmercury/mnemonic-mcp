// Package routingrule provides the PostgreSQL repository implementation for
// routing rule persistence. It defines the authoritative MatchType constants
// and match configuration types (keyword, regex, pattern, default), handles
// JSONB serialization of match configs, and supports priority-ordered queries
// for the routing engine.
//
// Documentation:
//   - Architecture: docs/architecture/08-data-architecture.md (Data Model Design > Routing Rules, Data Flow Patterns)
//   - Design: docs/design/data-storage.md (Repository Interfaces > RoutingRule Repository)
//   - Design: docs/design/routing-engine.md (Match Type Implementations, Rule Loading and Caching)
package routingrule
