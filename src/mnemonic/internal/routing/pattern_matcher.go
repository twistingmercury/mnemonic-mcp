package routing

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/twistingmercury/mnemonic/internal/repository/routingrule"
)

// PatternMatch represents a single pattern match result with its similarity score.
type PatternMatch struct {
	// PatternID is the unique identifier of the matched pattern.
	PatternID uuid.UUID

	// Similarity is the cosine similarity score between the prompt embedding
	// and the pattern embedding, ranging from 0.0 to 1.0.
	Similarity float64
}

// Embedder generates vector embeddings from text. Concrete implementations
// are provided by the embedding service; this interface is defined here
// (consumer side) to avoid coupling the routing package to a specific provider.
type Embedder interface {
	// Embed returns a vector embedding for the given text.
	Embed(ctx context.Context, text string) ([]float32, error)
}

// PatternStore retrieves patterns that are similar to a given embedding.
// It wraps the underlying pattern repository, abstracting away storage details.
type PatternStore interface {
	// FindSimilarByIDs searches for patterns matching the given IDs whose
	// embeddings exceed the similarity threshold relative to the provided embedding.
	FindSimilarByIDs(ctx context.Context, embedding []float32, patternIDs []uuid.UUID, threshold float64) ([]PatternMatch, error)
}

// PatternMatcher implements RuleMatcher for semantic/vector-similarity routing rules.
// It embeds the user prompt, then searches for similar patterns within a
// configured set of pattern IDs. The highest-similarity result determines
// the match confidence.
type PatternMatcher struct {
	embedder Embedder
	store    PatternStore
	// threshold is the default similarity threshold for pattern matching.
	// Per-request override via Options.PatternRelevanceThreshold will be
	// wired through the engine in a future phase.
	threshold float64
}

// NewPatternMatcher creates a new PatternMatcher with the given embedder,
// pattern store, and default similarity threshold.
func NewPatternMatcher(embedder Embedder, store PatternStore, defaultThreshold float64) *PatternMatcher {
	return &PatternMatcher{
		embedder:  embedder,
		store:     store,
		threshold: defaultThreshold,
	}
}

// Type returns the MatchType this matcher handles.
func (m *PatternMatcher) Type() MatchType {
	return MatchTypePattern
}

// Close is a no-op for PatternMatcher since it holds no background resources.
// It is safe to call multiple times.
func (m *PatternMatcher) Close() {}

// Match evaluates the prompt against the pattern match configuration using
// vector similarity search.
//
// Algorithm:
//  1. Check context for early cancellation.
//  2. Type-assert config to PatternMatchConfig.
//  3. If no pattern IDs are configured, return no match.
//  4. Embed the prompt text into a vector.
//  5. Search for similar patterns among the configured IDs.
//  6. Return the highest-similarity match as the result.
func (m *PatternMatcher) Match(ctx context.Context, prompt string, config routingrule.MatchConfig) (MatchResult, error) {
	if err := ctx.Err(); err != nil {
		return MatchResult{}, fmt.Errorf("pattern matcher: %w", err)
	}

	patConfig, ok := config.(routingrule.PatternMatchConfig)
	if !ok {
		return MatchResult{}, fmt.Errorf("pattern matcher: expected PatternMatchConfig, got %T", config)
	}

	if len(patConfig.PatternIDs) == 0 {
		return MatchResult{Matched: false}, nil
	}

	embedding, err := m.embedder.Embed(ctx, prompt)
	if err != nil {
		return MatchResult{}, fmt.Errorf("pattern matcher: embedding prompt: %w", err)
	}

	results, err := m.store.FindSimilarByIDs(ctx, embedding, patConfig.PatternIDs, m.threshold)
	if err != nil {
		return MatchResult{}, fmt.Errorf("pattern matcher: finding similar patterns: %w", err)
	}

	if len(results) == 0 {
		return MatchResult{Matched: false}, nil
	}

	// Find the result with the highest similarity score.
	best := results[0]
	for _, r := range results[1:] {
		if r.Similarity > best.Similarity {
			best = r
		}
	}

	return MatchResult{
		Matched:    true,
		Confidence: NormalizeConfidence(best.Similarity),
		Details:    fmt.Sprintf("matched pattern %s with similarity %.4f", best.PatternID, best.Similarity),
	}, nil
}
