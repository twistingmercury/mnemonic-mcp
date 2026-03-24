# Product Requirements Document: Graph-Enhanced Pattern Search

*Gralph processes cycles in this document from top to bottom. Checklist markers are significant: `- [ ]` (open), `- [x]` (complete), `- [~]` (abandoned). Each cycle must be small, independently verifiable, and assigned to exactly one agent.*

## Objective

Integrate Neo4j graph traversal into the `search_patterns` MCP tool so that search results include both vector similarity matches and conceptually related patterns discovered via graph relationships.

## Problem Statement

The `search_patterns` tool currently returns only pgvector cosine similarity results. Patterns that are conceptually related (sharing extracted concepts in Neo4j) but have dissimilar embeddings are invisible to the search unless the user explicitly calls `find_related_patterns` with a known pattern ID. This misses valuable results and requires extra manual steps from the consumer.

## Success Criteria

- `search_patterns` returns a "Related Patterns" section alongside vector results when graph data is available.
- Graph-expanded results are deduplicated against vector results.
- Graph-expanded results respect language filters (requested language + "agnostic").
- When `graphRepo` is nil or graph calls fail, search returns vector-only results identical to today's behavior.
- `make build` completes successfully.
- All unit tests pass, including new graph-expansion tests.
- All E2E tests pass, including new graph-enhanced search tests.

## Scope

### In scope

- `GetByIDs` batch lookup method on pattern repository
- Graph expansion logic in the search service
- `GraphMatch` type and `SearchResult.GraphMatches` field
- Updated response formatting in `formatSearchResults`
- Constructor wiring in `server.go`
- Unit tests for all new search service behavior
- E2E tests for graph-enhanced search results

### Out of scope

- Changes to the `search_patterns` MCP tool input schema
- Unified ranking of vector and graph results (future enhancement)
- Making Neo4j optional at the server level
- Changes to `find_related_patterns` or `get_pattern` tools

## Constraints and Decisions

- Design spec: `docs/superpowers/specs/2026-03-24-graph-enhanced-search-design.md`
- Go module root: `src/mnemonic/`
- Search service: `src/mnemonic/internal/service/search/`
- Graph repo interface: `src/mnemonic/internal/repository/graph/graph.go`
- Pattern repo: `src/mnemonic/internal/repository/pattern/`
- MCP formatting: `src/mnemonic/internal/mcpserver/format.go`
- Server wiring: `src/mnemonic/internal/server/server.go`
- Constructor signature: `New(embeddingSvc, patternRepo, agentRepo, chunkRepo, graphRepo, logger)` — graphRepo before logger
- Seed selection: top 3 patterns by highest cosine similarity across their chunks
- Graph expansion limit: 5 per seed, 5 total after dedup/filter
- Language filter: requested language OR "agnostic"
- Graceful degradation: nil graphRepo or failed graph calls = vector-only results

## Implementation Plan

- [ ] **Cycle 1 - Add GetByIDs to pattern repository**: Add `GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*Pattern, error)` to the pattern repository interface and implementation using `WHERE id = ANY($1)`.
  - Agent: `go software engineer`
  - Files: `src/mnemonic/internal/repository/pattern/repository.go`, `src/mnemonic/internal/repository/pattern/repository_test.go`
  - Steps:
    - Add `GetByIDs` to the `Repository` interface
    - Implement `GetByIDs` on the pgx repository struct using `WHERE id = ANY($1)` with the same column scanning as `Get`
    - Add unit tests: happy path with multiple IDs, empty slice input, partial matches (some IDs not found)
  - Verify: `cd src/mnemonic && go test ./internal/repository/pattern/... -v -run TestGetByIDs`
  - Done: All `TestGetByIDs` tests pass and `GetByIDs` is on the `Repository` interface

- [ ] **Cycle 2 - Add GraphMatch type and extend SearchResult**: Add the `GraphMatch` struct and `GraphMatches` field to `SearchResult` in the search service package.
  - Agent: `go software engineer`
  - Files: `src/mnemonic/internal/service/search/service.go`
  - Steps:
    - Add the `GraphMatch` struct with fields: `PatternID`, `PatternName`, `Similarity`, `ConceptNames`, `SeedPatternID`, `SeedPatternName`
    - Add `GraphMatches []*GraphMatch` field to `SearchResult`
  - Verify: `cd src/mnemonic && go build ./...`
  - Done: Build succeeds with new types; existing tests still pass

- [ ] **Cycle 3 - Add graph repo dependency to search service**: Update the search service constructor to accept an optional `graph.Repository` parameter and store it on the struct.
  - Agent: `go software engineer`
  - Files: `src/mnemonic/internal/service/search/service.go`, `src/mnemonic/internal/service/search/service_test.go`, `src/mnemonic/internal/server/server.go`
  - Steps:
    - Add `graphRepo graph.Repository` field to the `searchService` struct
    - Update `New` constructor signature to `New(embeddingSvc, patternRepo, agentRepo, chunkRepo, graphRepo, logger)` with graphRepo before logger
    - Update `server.go` `wireDependencies` to pass the existing `graphRepo` to the search service constructor
    - Update all existing test call sites to pass `nil` for graphRepo so existing tests continue to pass
  - Verify: `cd src/mnemonic && go test ./internal/service/search/... -v`
  - Done: All existing search tests pass with nil graphRepo; build succeeds

- [ ] **Cycle 4 - Implement graph expansion logic**: Add the graph expansion phase to `SearchPatterns` — seed selection, graph traversal, deduplication, filtering, and capping.
  - Agent: `go software engineer`
  - Files: `src/mnemonic/internal/service/search/service.go`, `src/mnemonic/internal/service/search/service_test.go`
  - Steps:
    - After vector search, extract unique pattern IDs sorted by highest cosine similarity across their chunks
    - Take top 3 as seeds, call `graphRepo.FindRelatedPatterns` for each with limit 5
    - Deduplicate graph results by pattern ID, keeping highest similarity
    - Exclude pattern IDs already in vector results
    - Batch-fetch pattern metadata via `patternRepo.GetByIDs` for remaining graph results
    - Post-filter: keep results matching requested language OR "agnostic"; apply domain and tag filters if provided
    - Cap at 5 total graph results
    - Set `GraphMatches` on `SearchResult`
    - If graphRepo is nil, skip entirely; if graph calls fail, log warning and return vector-only results
  - Verify: `cd src/mnemonic && go test ./internal/service/search/... -v -count=1`
  - Done: All tests pass including new tests for: happy path, nil graphRepo, graph failure, dedup against vector, cross-seed dedup, language filtering, empty vector results

- [ ] **Cycle 5 - Update response formatting**: Add graph results section to `formatSearchResults` output.
  - Agent: `go software engineer`
  - Files: `src/mnemonic/internal/mcpserver/format.go`, `src/mnemonic/internal/mcpserver/format_test.go`
  - Steps:
    - Add a new formatting helper for `search.GraphMatch` entries (analogous to `writeRelatedEntry`)
    - Update `formatSearchResults` to append a "Related Patterns" section after vector results when `GraphMatches` is non-nil and non-empty
    - Graph results display decimal similarity, seed pattern name, and shared concept names
    - Add unit tests for: format with graph matches, format without graph matches (nil), format with empty graph matches slice
  - Verify: `cd src/mnemonic && go test ./internal/mcpserver/... -v -run TestFormat`
  - Done: All format tests pass; graph section renders correctly in output

- [ ] **Cycle 6 - Unit test completeness and make build**: Run full test suite and Docker build to verify everything integrates correctly.
  - Agent: `go software engineer`
  - Files: `src/mnemonic/internal/service/search/service_test.go`
  - Steps:
    - Run full unit test suite and fix any failures
    - Run `make build` from repo root and fix any issues
    - Verify no regressions in existing behavior
  - Verify: `cd src/mnemonic && go test ./... && cd ../.. && make build`
  - Done: All unit tests pass and `make build` exits 0

- [ ] **Cycle 7 - E2E tests for graph-enhanced search**: Add E2E tests that verify the MCP tool returns graph-expanded results from the consumer perspective.
  - Agent: `go e2e test engineer`
  - Files: `src/mnemonic/tests/e2e/mcp/mcp_test.go`
  - Steps:
    - Add test: `search_patterns` returns both vector matches and "Related Patterns" section when graph data exists
    - Add test: graph section contains pattern names, similarity scores, and shared concepts
    - Add test: graph results don't duplicate patterns already in vector results
    - Add test: when patterns aren't enriched, response contains only vector results (no graph section)
  - Verify: `cd src/mnemonic/tests/e2e && go test ./... -v -run TestGraphEnhanced`
  - Done: All graph-enhanced E2E tests pass

## Risks and Mitigations

- Risk: Graph expansion adds latency to every search call due to Neo4j round-trips.
  - Mitigation: Graph calls are bounded (3 seeds × 5 limit = 15 max results before dedup). Monitor with existing telemetry. Can add a timeout or disable flag if needed.

- Risk: `GetByIDs` batch query returns partial results when some IDs don't exist in Postgres (e.g., graph has stale pattern references).
  - Mitigation: `GetByIDs` returns only found patterns. The post-filter step works with whatever is returned; missing patterns are silently excluded from graph results.

- Risk: E2E tests require enriched patterns with graph data to exist in the test database.
  - Mitigation: E2E test setup must seed patterns, run enrichment, and verify graph edges exist before testing graph-enhanced search.

## Definition of Done

- All unit tests pass: `cd src/mnemonic && go test ./...`
- All E2E tests pass: `cd src/mnemonic/tests/e2e && go test ./...`
- Docker build succeeds: `make build`
- Existing `search_patterns` behavior is preserved when graphRepo is nil
