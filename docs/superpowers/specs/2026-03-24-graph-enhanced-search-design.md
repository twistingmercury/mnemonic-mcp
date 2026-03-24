# Graph-Enhanced Pattern Search

**Date:** 2026-03-24
**Status:** Approved
**Component:** mnemonic-mcp search service

## Summary

Enhance the `search_patterns` MCP tool to include graph-expanded results alongside existing vector similarity results. The vector search executes as today, then the top-matching patterns are used as seeds for Neo4j graph traversal to surface conceptually related patterns that vector similarity alone would miss.

The MCP tool input schema is unchanged. The response markdown gains an optional "Related Patterns" section when graph data is available.

## Background

The search service currently performs vector-only search via pgvector cosine similarity on pattern chunk embeddings. Graph relationships (concept overlap stored as RELATED_TO edges in Neo4j) exist but are only exposed through the separate `find_related_patterns` MCP tool. The original intent was to integrate graph search into the primary search flow to improve result quality.

## Design

### Search Flow

The `searchService.SearchPatterns` method gains a graph expansion phase after the existing vector search:

1. **Vector search** (unchanged): Embed query via OpenAI, find similar chunks in pgvector, apply filters (language, domain, agent, tags, threshold)
2. **Seed selection**: Extract unique pattern IDs from vector results. For each pattern, take the highest cosine similarity across all its chunks as the pattern's score. Sort by this score descending and take the top 3 as seed patterns.
3. **Graph expansion**: For each seed pattern, call `graphRepo.FindRelatedPatterns(ctx, seedID, limit)` with limit of 5 per seed.
4. **Deduplication**: Collect all graph results, deduplicate by pattern ID (keep highest similarity if a pattern appears from multiple seeds).
5. **Exclude vector matches**: Remove any pattern ID already present in vector results.
6. **Post-filter**: Graph results need language/domain metadata for filtering. Since `graph.RelatedPattern` only contains ID and name, look up each graph result's language and domain from the pattern repository (batch query by IDs). Then keep only results matching the requested language OR `"agnostic"`. Apply domain and tag filters if provided.
7. **Cap**: Limit to 5 total graph-expanded results.
8. **Attach**: Set `GraphMatches` on the `SearchResult`.

### Graceful Degradation

- If `graphRepo` is nil (Neo4j not configured), graph expansion is skipped entirely. Behavior is identical to today.
- If any graph call fails, log a warning and return vector-only results. The vector results are never affected by graph failures.

### Data Types

New type on the search service:

```go
type GraphMatch struct {
    PatternID       uuid.UUID
    PatternName     string
    Similarity      float64   // concept-overlap similarity from RELATED_TO edge
    ConceptNames    []string  // maps from graph.RelatedPattern.ConceptNames
    SeedPatternID   uuid.UUID // which vector-match pattern led here
    SeedPatternName string
}
```

Extended `SearchResult`:

```go
type SearchResult struct {
    Matches          []*ChunkMatch   // existing vector results
    GraphMatches     []*GraphMatch   // new: graph-expanded results (nil if unavailable)
    Query            string
    TotalCandidates  int
    SearchDurationMs int64
}
```

### Dependency Changes

- `searchService` struct gains an optional `graphRepo graph.Repository` field.
- Constructor signature becomes: `New(embeddingSvc, patternRepo, agentRepo, chunkRepo, graphRepo, logger)` — `graphRepo` is added before `logger`. Pass `nil` to disable graph expansion.
- **Wiring site**: `internal/server/server.go` in `wireDependencies` must be updated to pass the existing `graphRepo` (already constructed at line ~185) into the search service constructor. Neo4j is currently mandatory at the server level (`openDatabases` fails if Neo4j is unavailable), so `graphRepo` will always be non-nil in production. The nil path is exercised in unit tests.
- No changes to `ToolDependencies` interface or MCP handler signatures.

### Pattern Repository Addition

The post-filter step (step 6) requires looking up language, domain, and tags for graph-expanded patterns. `patternrepo.Repository` currently only has `Get(ctx, id)` for single lookups. A new `GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*Pattern, error)` method must be added to the interface to support batch lookup. This returns full `Pattern` structs, providing language, domain, and tags in a single query. The implementation uses a `WHERE id = ANY($1)` clause.

### Response Formatting

`formatSearchResults` renders the graph section only when `GraphMatches` is non-nil and non-empty. Vector results appear first, then a separated "Related Patterns" section.

Vector results display percentage similarity (existing behavior). Graph results display decimal similarity with the seed pattern name and shared concepts for context. A new formatting helper (analogous to `writeRelatedEntry` in `format.go`) is needed for `search.GraphMatch`, since the existing helper targets `patternsvc.RelatedPatternResult`.

**Note:** Graph expansion requires vector seeds, so `GraphMatches` is always empty/nil when `Matches` is empty. The existing early return in `formatSearchResults` (when no vector matches) is safe and does not need a special case for graph-only results.

### MCP Tool Interface

No changes. `search_patterns` accepts the same input parameters. The output markdown is richer when graph data is available.

## Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Integration point | Search service (Approach A) | Natural home for search logic; graph is a planned evolution |
| Result presentation | Separated sections | Simpler than unified ranking; avoids incomparable score normalization |
| Seed selection | Top 3 patterns by cosine similarity | Focused expansion from strongest matches avoids noise |
| Deduplication | Exclude graph results already in vector results | No value in repeating patterns |
| Language filtering | Requested language + "agnostic" | Agnostic patterns are always relevant regardless of language filter |
| Graph result limit | Fixed default of 5 | Keeps graph section supplementary; avoidable schema change |
| Graceful degradation | Nil graph repo or failed calls = vector-only | Vector search must never be degraded by graph issues |

## Testing

### Unit Tests (search service)

- Happy path: vector results + graph expansion returns combined result
- Graph repo nil: returns vector-only results (backward compatible)
- Graph call fails: logs warning, returns vector-only results
- Deduplication: graph result already in vector results gets excluded
- Cross-seed deduplication: same pattern returned by two seed expansions, only highest-similarity instance kept
- Language filtering: graph results filtered to requested language + agnostic
- No seed patterns: vector results empty, no graph expansion attempted

### E2E Tests (MCP tool consumer perspective)

- `search_patterns` returns both vector matches and a "Related Patterns" section when graph data exists
- Graph section contains pattern names, similarity scores, and shared concepts
- Graph results don't duplicate patterns already in vector results
- When patterns aren't enriched (no graph data), response contains only vector results

## Definition of Done

- All unit tests pass
- All E2E tests pass (including new graph-enhanced search tests)
- `make build` completes successfully
- Existing `search_patterns` behavior is preserved when graph repo is nil
