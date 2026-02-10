package routing_test

import (
	"context"

	"github.com/google/uuid"
	"github.com/twistingmercury/mnemonic/internal/repository/routingrule"
	"github.com/twistingmercury/mnemonic/internal/routing"
)

// mockRuleMatcher is a hand-rolled mock for the RuleMatcher interface.
type mockRuleMatcher struct {
	matchType routing.MatchType
	matchFn   func(ctx context.Context, prompt string, config routingrule.MatchConfig) (routing.MatchResult, error)
}

func (m *mockRuleMatcher) Match(ctx context.Context, prompt string, config routingrule.MatchConfig) (routing.MatchResult, error) {
	return m.matchFn(ctx, prompt, config)
}

func (m *mockRuleMatcher) Type() routing.MatchType {
	return m.matchType
}

func (m *mockRuleMatcher) Close() {}

// mockRuleLoader is a hand-rolled mock for the RuleLoader interface.
type mockRuleLoader struct {
	loadFn func(ctx context.Context) ([]*routingrule.Rule, error)
}

func (m *mockRuleLoader) LoadRules(ctx context.Context) ([]*routingrule.Rule, error) {
	return m.loadFn(ctx)
}

// mockEmbedder is a hand-rolled mock for the Embedder interface.
type mockEmbedder struct {
	embedFn func(ctx context.Context, text string) ([]float32, error)
}

func (m *mockEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	return m.embedFn(ctx, text)
}

// mockPatternStore is a hand-rolled mock for the PatternStore interface.
type mockPatternStore struct {
	findSimilarByIDsFn func(ctx context.Context, embedding []float32, patternIDs []uuid.UUID, threshold float64) ([]routing.PatternMatch, error)
}

func (m *mockPatternStore) FindSimilarByIDs(ctx context.Context, embedding []float32, patternIDs []uuid.UUID, threshold float64) ([]routing.PatternMatch, error) {
	return m.findSimilarByIDsFn(ctx, embedding, patternIDs, threshold)
}
