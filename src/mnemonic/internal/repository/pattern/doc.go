// Package pattern provides the PostgreSQL repository implementation for pattern
// persistence. It supports CRUD operations, embedding storage, enrichment status
// management, agent associations, and vector similarity search via pgvector for
// cosine-distance-based semantic matching.
//
// Documentation:
//   - Architecture: docs/architecture/08-data-architecture.md (Data Model Design > Patterns, PGVector Configuration, Neo4j Graph Model)
//   - Design: docs/design/data-storage.md (Repository Interfaces > PatternRepository, PGVector Configuration)
//   - Design: docs/design/routing-engine.md (Pattern Matcher (Semantic))
package pattern
