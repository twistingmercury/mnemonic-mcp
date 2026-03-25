// Package search provides semantic similarity search over pattern chunks.
// Both the REST search endpoint and the MCP search_patterns tool use this service.
// It coordinates between the embedding service (for query vectorization), the
// chunk repository (for pgvector similarity search), and the agent repository
// (for optional agent-scoped pre-filtering).
package search

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	agentrepo "github.com/twistingmercury/mnemonic/internal/repository/agent"
	chunkrepo "github.com/twistingmercury/mnemonic/internal/repository/chunk"
	graphrepo "github.com/twistingmercury/mnemonic/internal/repository/graph"
	patternrepo "github.com/twistingmercury/mnemonic/internal/repository/pattern"
	"github.com/twistingmercury/mnemonic/internal/service"
	openaisvc "github.com/twistingmercury/mnemonic/internal/service/openai"
)

// Service handles semantic search over patterns.
type Service interface {
	// SearchPatterns generates a query embedding and performs vector similarity
	// search. If AgentName is non-empty in opts, pre-filters to patterns
	// associated with that agent.
	SearchPatterns(ctx context.Context, opts SearchOptions) (*SearchResult, error)
}

// SearchOptions defines the parameters for a semantic search.
type SearchOptions struct {
	Query     string   // Natural language query text
	Limit     int      // Max results (default 10, max 50)
	Threshold float64  // Min similarity (default 0.5)
	Tags      []string // Conjunctive tag filter
	AgentName string   // Optional agent name filter
	Language  string   // Optional: filter by pattern language
	Domain    string   // Optional: filter by pattern domain
}

// ChunkMatch is a single semantic search hit from a pattern chunk.
type ChunkMatch struct {
	PatternID    uuid.UUID
	PatternName  string
	EntityType   string
	Language     string
	Domain       string
	Tags         []string
	SectionTitle string
	ChunkIndex   int
	Content      string
	Similarity   float64
}

// GraphMatch is a pattern discovered via Neo4j graph traversal from a vector search seed.
type GraphMatch struct {
	PatternID       uuid.UUID
	PatternName     string
	Similarity      float64  // concept-overlap similarity from RELATED_TO edge
	ConceptNames    []string // shared concept names from graph relationship
	SeedPatternID   uuid.UUID
	SeedPatternName string
}

// SearchResult wraps similarity search matches with metadata required by
// the OpenAPI PatternSearchResponse schema.
type SearchResult struct {
	Matches          []*ChunkMatch
	GraphMatches     []*GraphMatch // graph-expanded results (nil if unavailable)
	Query            string        // Echo of the original query text
	TotalCandidates  int           // Total chunk matches returned (after threshold filtering)
	SearchDurationMs int64         // Wall-clock search time in milliseconds
}

const (
	// DefaultGraphSeedCount is the number of top vector-match patterns used as graph expansion seeds.
	DefaultGraphSeedCount = 3

	// DefaultGraphResultsPerSeed is the max related patterns fetched per seed from Neo4j.
	DefaultGraphResultsPerSeed = 5

	// DefaultGraphMatchCap is the maximum graph matches returned in a single search response.
	DefaultGraphMatchCap = 5
)

// searchService implements the Service interface.
type searchService struct {
	embeddingSvc        openaisvc.EmbeddingService
	patternRepo         patternrepo.Repository
	agentRepo           agentrepo.Repository
	chunkRepo           chunkrepo.Repository
	graphRepo           graphrepo.Repository
	logger              zerolog.Logger
	graphSeedCount      int
	graphResultsPerSeed int
	graphMatchCap       int
}

// New creates a new search Service backed by the given dependencies.
// graphRepo may be nil; graph-enhanced search will be skipped when absent.
func New(
	embeddingSvc openaisvc.EmbeddingService,
	patternRepo patternrepo.Repository,
	agentRepo agentrepo.Repository,
	chunkRepo chunkrepo.Repository,
	graphRepo graphrepo.Repository,
	logger zerolog.Logger,
) Service {
	return &searchService{
		embeddingSvc:        embeddingSvc,
		patternRepo:         patternRepo,
		agentRepo:           agentRepo,
		chunkRepo:           chunkRepo,
		graphRepo:           graphRepo,
		logger:              logger,
		graphSeedCount:      DefaultGraphSeedCount,
		graphResultsPerSeed: DefaultGraphResultsPerSeed,
		graphMatchCap:       DefaultGraphMatchCap,
	}
}

// SearchPatterns generates a query embedding and performs vector similarity search.
func (s *searchService) SearchPatterns(ctx context.Context, opts SearchOptions) (*SearchResult, error) {
	start := time.Now()

	// 1. Generate query embedding.
	embedding, err := s.embeddingSvc.Embed(ctx, opts.Query)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", service.ErrServiceUnavailable, err)
	}

	// 2. If agent name provided, resolve to pattern IDs for pre-filtering.
	var patternIDs []uuid.UUID
	if opts.AgentName != "" {
		agent, agentErr := s.agentRepo.Get(ctx, opts.AgentName)
		if agentErr != nil {
			if errors.Is(agentErr, agentrepo.ErrNotFound) {
				// Unknown agent: return empty results (not an error).
				return &SearchResult{
					Matches:          []*ChunkMatch{},
					Query:            opts.Query,
					TotalCandidates:  0,
					SearchDurationMs: time.Since(start).Milliseconds(),
				}, nil
			}
			return nil, fmt.Errorf("resolve agent: %w", agentErr)
		}

		var idsErr error
		patternIDs, idsErr = s.patternRepo.GetPatternIDsByAgent(ctx, agent.ID)
		if idsErr != nil {
			return nil, fmt.Errorf("get agent patterns: %w", idsErr)
		}

		// No associated patterns: return empty results.
		if len(patternIDs) == 0 {
			return &SearchResult{
				Matches:          []*ChunkMatch{},
				Query:            opts.Query,
				TotalCandidates:  0,
				SearchDurationMs: time.Since(start).Milliseconds(),
			}, nil
		}
	}

	// 3. Perform chunk-based similarity search.

	// Guard: chunkRepo must be configured before attempting vector search.
	if s.chunkRepo == nil {
		return nil, fmt.Errorf("%w: chunk repository not configured", service.ErrServiceUnavailable)
	}

	simOpts := chunkrepo.SimilarityOptions{
		MinSimilarity: opts.Threshold,
		MaxResults:    opts.Limit,
		Language:      opts.Language,
		Domain:        opts.Domain,
		PatternIDs:    patternIDs,
		Tags:          opts.Tags,
	}

	rawMatches, err := s.chunkRepo.FindSimilar(ctx, embedding, simOpts)
	if err != nil {
		return nil, fmt.Errorf("find similar chunks: %w", err)
	}

	matches := make([]*ChunkMatch, len(rawMatches))
	for i, m := range rawMatches {
		matches[i] = &ChunkMatch{
			PatternID:    m.PatternID,
			PatternName:  m.PatternName,
			EntityType:   m.EntityType,
			Language:     m.Language,
			Domain:       m.Domain,
			Tags:         m.Tags,
			SectionTitle: m.SectionTitle,
			ChunkIndex:   m.ChunkIndex,
			Content:      m.Content,
			Similarity:   m.Similarity,
		}
	}

	graphMatches := s.expandViaGraph(ctx, matches, opts)

	return &SearchResult{
		Matches:          matches,
		GraphMatches:     graphMatches,
		Query:            opts.Query,
		TotalCandidates:  len(matches),
		SearchDurationMs: time.Since(start).Milliseconds(),
	}, nil
}

// seedPattern holds the identifying fields for a top vector-result pattern.
type seedPattern struct {
	id         uuid.UUID
	name       string
	similarity float64
}

// expandViaGraph performs graph-based expansion from the top vector seeds.
// Returns nil (no error surfaced to caller) when graph is unavailable or fails.
func (s *searchService) expandViaGraph(ctx context.Context, matches []*ChunkMatch, opts SearchOptions) []*GraphMatch {
	// Guard: skip if graph repo not configured or no vector results to seed from.
	if s.graphRepo == nil || len(matches) == 0 {
		return nil
	}

	// Build per-pattern best similarity from vector results.
	bestSim := make(map[uuid.UUID]seedPattern)
	for _, m := range matches {
		if existing, ok := bestSim[m.PatternID]; !ok || m.Similarity > existing.similarity {
			bestSim[m.PatternID] = seedPattern{
				id:         m.PatternID,
				name:       m.PatternName,
				similarity: m.Similarity,
			}
		}
	}

	// Sort seeds by similarity descending; take top 3.
	seeds := make([]seedPattern, 0, len(bestSim))
	for _, sp := range bestSim {
		seeds = append(seeds, sp)
	}
	sort.Slice(seeds, func(i, j int) bool {
		return seeds[i].similarity > seeds[j].similarity
	})
	if len(seeds) > s.graphSeedCount {
		seeds = seeds[:s.graphSeedCount]
	}

	// Call graph repo for each seed; collect across all seeds with cross-seed dedup.
	// Keep the entry with the highest Similarity when the same pattern appears from multiple seeds.
	type graphEntry struct {
		related graphrepo.RelatedPattern
		seed    seedPattern
	}
	best := make(map[uuid.UUID]graphEntry)

	for _, seed := range seeds {
		related, err := s.graphRepo.FindRelatedPatterns(ctx, seed.id, s.graphResultsPerSeed)
		if err != nil {
			s.logger.Warn().Err(err).Str("seed_pattern_id", seed.id.String()).Msg("graph expansion failed for seed; continuing with remaining seeds")
			continue
		}
		for _, rp := range related {
			if existing, ok := best[rp.ID]; !ok || rp.Similarity > existing.related.Similarity {
				best[rp.ID] = graphEntry{related: rp, seed: seed}
			}
		}
	}

	// Exclude patterns already present in the vector result set.
	vectorIDs := make(map[uuid.UUID]struct{}, len(bestSim))
	for id := range bestSim {
		vectorIDs[id] = struct{}{}
	}
	filtered := make([]graphEntry, 0, len(best))
	for id, entry := range best {
		if _, inVector := vectorIDs[id]; !inVector {
			filtered = append(filtered, entry)
		}
	}

	if len(filtered) == 0 {
		return nil
	}

	// Batch-fetch metadata for remaining graph candidates.
	candidateIDs := make([]uuid.UUID, len(filtered))
	for i, entry := range filtered {
		candidateIDs[i] = entry.related.ID
	}

	patterns, err := s.patternRepo.GetByIDs(ctx, candidateIDs)
	if err != nil {
		s.logger.Warn().Err(err).Msg("failed to fetch graph-expanded pattern metadata; returning vector-only results")
		return nil
	}

	// Index metadata by ID for O(1) lookup.
	meta := make(map[uuid.UUID]*patternrepo.Pattern, len(patterns))
	for _, p := range patterns {
		meta[p.ID] = p
	}

	// Post-filter by language / domain / tags and build GraphMatch slice.
	graphMatches := make([]*GraphMatch, 0, len(filtered))
	for _, entry := range filtered {
		p, ok := meta[entry.related.ID]
		if !ok {
			continue
		}
		if opts.Language != "" && p.Language != opts.Language && p.Language != "agnostic" {
			continue
		}
		if opts.Domain != "" && p.Domain != opts.Domain {
			continue
		}
		if len(opts.Tags) > 0 && !hasAllTags(p.Tags, opts.Tags) {
			continue
		}
		graphMatches = append(graphMatches, &GraphMatch{
			PatternID:       p.ID,
			PatternName:     p.Name,
			Similarity:      entry.related.Similarity,
			ConceptNames:    entry.related.ConceptNames,
			SeedPatternID:   entry.seed.id,
			SeedPatternName: entry.seed.name,
		})
	}

	if len(graphMatches) == 0 {
		return nil
	}

	// Sort by similarity descending, then cap.
	sort.Slice(graphMatches, func(i, j int) bool {
		return graphMatches[i].Similarity > graphMatches[j].Similarity
	})
	if len(graphMatches) > s.graphMatchCap {
		graphMatches = graphMatches[:s.graphMatchCap]
	}

	return graphMatches
}

// hasAllTags reports whether tags contains every element of required (conjunctive filter).
func hasAllTags(tags []string, required []string) bool {
	tagSet := make(map[string]struct{}, len(tags))
	for _, t := range tags {
		tagSet[t] = struct{}{}
	}
	for _, r := range required {
		if _, ok := tagSet[r]; !ok {
			return false
		}
	}
	return true
}
