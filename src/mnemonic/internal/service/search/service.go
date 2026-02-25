// Package search provides semantic similarity search over patterns.
// Both the REST search endpoint and the MCP search_patterns tool use this service.
// It coordinates between the embedding service (for query vectorization), the
// pattern repository (for pgvector similarity search), and the agent repository
// (for optional agent-scoped pre-filtering).
package search

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	agentrepo "github.com/twistingmercury/mnemonic/internal/repository/agent"
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
	Threshold float64  // Min similarity (default 0.7)
	Tags      []string // Conjunctive tag filter
	AgentName string   // Optional agent name filter
}

// SearchResult wraps similarity search matches with metadata required by
// the OpenAPI PatternSearchResponse schema.
type SearchResult struct {
	Matches          []*patternrepo.Match
	Query            string // Echo of the original query text
	TotalCandidates  int    // Total patterns evaluated (before threshold filtering)
	SearchDurationMs int64  // Wall-clock search time in milliseconds
}

// searchService implements the Service interface.
type searchService struct {
	embeddingSvc openaisvc.EmbeddingService
	patternRepo  patternrepo.Repository
	agentRepo    agentrepo.Repository
	logger       zerolog.Logger
}

// New creates a new search Service backed by the given dependencies.
func New(
	embeddingSvc openaisvc.EmbeddingService,
	patternRepo patternrepo.Repository,
	agentRepo agentrepo.Repository,
	logger zerolog.Logger,
) Service {
	return &searchService{
		embeddingSvc: embeddingSvc,
		patternRepo:  patternRepo,
		agentRepo:    agentRepo,
		logger:       logger,
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

	// 2. Build similarity options from search options.
	simOpts := patternrepo.SimilarityOptions{
		MinSimilarity: opts.Threshold,
		MaxResults:    opts.Limit,
		Tags:          opts.Tags,
	}

	// 3. If agent name provided, resolve to pattern IDs for pre-filtering.
	if opts.AgentName != "" {
		agent, agentErr := s.agentRepo.Get(ctx, opts.AgentName)
		if agentErr != nil {
			if errors.Is(agentErr, agentrepo.ErrNotFound) {
				// Unknown agent: return empty results (not an error).
				return &SearchResult{
					Matches:          []*patternrepo.Match{},
					Query:            opts.Query,
					TotalCandidates:  0,
					SearchDurationMs: time.Since(start).Milliseconds(),
				}, nil
			}
			return nil, fmt.Errorf("resolve agent: %w", agentErr)
		}

		patternIDs, idsErr := s.patternRepo.GetPatternIDsByAgent(ctx, agent.ID)
		if idsErr != nil {
			return nil, fmt.Errorf("get agent patterns: %w", idsErr)
		}

		// No associated patterns: return empty results.
		if len(patternIDs) == 0 {
			return &SearchResult{
				Matches:          []*patternrepo.Match{},
				Query:            opts.Query,
				TotalCandidates:  0,
				SearchDurationMs: time.Since(start).Milliseconds(),
			}, nil
		}

		simOpts.PatternIDs = patternIDs
	}

	// 4. Perform pgvector cosine similarity search.
	matches, err := s.patternRepo.FindSimilar(ctx, embedding, simOpts)
	if err != nil {
		return nil, fmt.Errorf("find similar patterns: %w", err)
	}

	return &SearchResult{
		Matches:          matches,
		Query:            opts.Query,
		TotalCandidates:  len(matches),
		SearchDurationMs: time.Since(start).Milliseconds(),
	}, nil
}
