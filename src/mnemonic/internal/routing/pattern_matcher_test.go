package routing_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twistingmercury/mnemonic/internal/repository/routingrule"
	"github.com/twistingmercury/mnemonic/internal/routing"
)

func TestPatternMatcher_Type(t *testing.T) {
	t.Parallel()

	matcher := routing.NewPatternMatcher(
		&mockEmbedder{embedFn: func(_ context.Context, _ string) ([]float32, error) { return nil, nil }},
		&mockPatternStore{findSimilarByIDsFn: func(_ context.Context, _ []float32, _ []uuid.UUID, _ float64) ([]routing.PatternMatch, error) {
			return nil, nil
		}},
		0.7,
	)

	assert.Equal(t, routing.MatchTypePattern, matcher.Type())
}

func TestPatternMatcher_Match(t *testing.T) {
	t.Parallel()

	patternID1 := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	patternID2 := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	patternID3 := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	testEmbedding := []float32{0.1, 0.2, 0.3}

	tests := []struct {
		name            string
		prompt          string
		config          routingrule.MatchConfig
		embedder        *mockEmbedder
		store           *mockPatternStore
		wantMatched     bool
		wantConfidence  float64
		wantDetailsHas  string
		wantErr         bool
		wantErrContains string
	}{
		{
			name:   "successful match - single pattern",
			prompt: "write a web server",
			config: routingrule.PatternMatchConfig{
				PatternIDs: []uuid.UUID{patternID1},
			},
			embedder: &mockEmbedder{
				embedFn: func(_ context.Context, _ string) ([]float32, error) {
					return testEmbedding, nil
				},
			},
			store: &mockPatternStore{
				findSimilarByIDsFn: func(_ context.Context, _ []float32, _ []uuid.UUID, _ float64) ([]routing.PatternMatch, error) {
					return []routing.PatternMatch{
						{PatternID: patternID1, Similarity: 0.85},
					}, nil
				},
			},
			wantMatched:    true,
			wantConfidence: 0.85,
			wantDetailsHas: patternID1.String(),
		},
		{
			name:   "multiple matches - highest similarity wins",
			prompt: "build a microservice",
			config: routingrule.PatternMatchConfig{
				PatternIDs: []uuid.UUID{patternID1, patternID2, patternID3},
			},
			embedder: &mockEmbedder{
				embedFn: func(_ context.Context, _ string) ([]float32, error) {
					return testEmbedding, nil
				},
			},
			store: &mockPatternStore{
				findSimilarByIDsFn: func(_ context.Context, _ []float32, _ []uuid.UUID, _ float64) ([]routing.PatternMatch, error) {
					return []routing.PatternMatch{
						{PatternID: patternID1, Similarity: 0.72},
						{PatternID: patternID2, Similarity: 0.95},
						{PatternID: patternID3, Similarity: 0.81},
					}, nil
				},
			},
			wantMatched:    true,
			wantConfidence: 0.95,
			wantDetailsHas: patternID2.String(),
		},
		{
			name:   "no match - store returns empty",
			prompt: "unrelated prompt",
			config: routingrule.PatternMatchConfig{
				PatternIDs: []uuid.UUID{patternID1},
			},
			embedder: &mockEmbedder{
				embedFn: func(_ context.Context, _ string) ([]float32, error) {
					return testEmbedding, nil
				},
			},
			store: &mockPatternStore{
				findSimilarByIDsFn: func(_ context.Context, _ []float32, _ []uuid.UUID, _ float64) ([]routing.PatternMatch, error) {
					return nil, nil
				},
			},
			wantMatched: false,
		},
		{
			name:   "empty pattern IDs - early return without calling embedder",
			prompt: "any prompt",
			config: routingrule.PatternMatchConfig{
				PatternIDs: []uuid.UUID{},
			},
			embedder: &mockEmbedder{
				embedFn: func(_ context.Context, _ string) ([]float32, error) {
					panic("embedder should not be called when PatternIDs is empty")
				},
			},
			store: &mockPatternStore{
				findSimilarByIDsFn: func(_ context.Context, _ []float32, _ []uuid.UUID, _ float64) ([]routing.PatternMatch, error) {
					panic("store should not be called when PatternIDs is empty")
				},
			},
			wantMatched: false,
		},
		{
			name:   "nil pattern IDs - early return without calling embedder",
			prompt: "any prompt",
			config: routingrule.PatternMatchConfig{
				PatternIDs: nil,
			},
			embedder: &mockEmbedder{
				embedFn: func(_ context.Context, _ string) ([]float32, error) {
					panic("embedder should not be called when PatternIDs is nil")
				},
			},
			store: &mockPatternStore{
				findSimilarByIDsFn: func(_ context.Context, _ []float32, _ []uuid.UUID, _ float64) ([]routing.PatternMatch, error) {
					panic("store should not be called when PatternIDs is nil")
				},
			},
			wantMatched: false,
		},
		{
			name:   "empty prompt - delegates to embedder",
			prompt: "",
			config: routingrule.PatternMatchConfig{
				PatternIDs: []uuid.UUID{patternID1},
			},
			embedder: &mockEmbedder{
				embedFn: func(_ context.Context, _ string) ([]float32, error) {
					return testEmbedding, nil
				},
			},
			store: &mockPatternStore{
				findSimilarByIDsFn: func(_ context.Context, _ []float32, _ []uuid.UUID, _ float64) ([]routing.PatternMatch, error) {
					return nil, nil
				},
			},
			wantMatched: false,
		},
		{
			name:   "embedder error propagated",
			prompt: "some prompt",
			config: routingrule.PatternMatchConfig{
				PatternIDs: []uuid.UUID{patternID1},
			},
			embedder: &mockEmbedder{
				embedFn: func(_ context.Context, _ string) ([]float32, error) {
					return nil, errors.New("embedding service unavailable")
				},
			},
			store: &mockPatternStore{
				findSimilarByIDsFn: func(_ context.Context, _ []float32, _ []uuid.UUID, _ float64) ([]routing.PatternMatch, error) {
					return nil, nil
				},
			},
			wantErr:         true,
			wantErrContains: "embedding prompt",
		},
		{
			name:   "store error propagated",
			prompt: "some prompt",
			config: routingrule.PatternMatchConfig{
				PatternIDs: []uuid.UUID{patternID1},
			},
			embedder: &mockEmbedder{
				embedFn: func(_ context.Context, _ string) ([]float32, error) {
					return testEmbedding, nil
				},
			},
			store: &mockPatternStore{
				findSimilarByIDsFn: func(_ context.Context, _ []float32, _ []uuid.UUID, _ float64) ([]routing.PatternMatch, error) {
					return nil, errors.New("database connection failed")
				},
			},
			wantErr:         true,
			wantErrContains: "finding similar patterns",
		},
		{
			name:   "wrong config type returns error",
			prompt: "some prompt",
			config: routingrule.KeywordMatchConfig{
				Keywords: []string{"go"},
			},
			embedder: &mockEmbedder{
				embedFn: func(_ context.Context, _ string) ([]float32, error) {
					return nil, nil
				},
			},
			store: &mockPatternStore{
				findSimilarByIDsFn: func(_ context.Context, _ []float32, _ []uuid.UUID, _ float64) ([]routing.PatternMatch, error) {
					return nil, nil
				},
			},
			wantErr:         true,
			wantErrContains: "expected PatternMatchConfig",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			matcher := routing.NewPatternMatcher(tt.embedder, tt.store, 0.7)

			result, err := matcher.Match(context.Background(), tt.prompt, tt.config)

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrContains != "" {
					assert.Contains(t, err.Error(), tt.wantErrContains)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantMatched, result.Matched)

			if tt.wantMatched {
				assert.InDelta(t, tt.wantConfidence, result.Confidence, 0.001)
				if tt.wantDetailsHas != "" {
					assert.Contains(t, result.Details, tt.wantDetailsHas)
				}
				assert.Empty(t, result.MatchedKeywords, "PatternMatcher should never populate MatchedKeywords")
			}
		})
	}
}

func TestPatternMatcher_Match_ContextCancellation(t *testing.T) {
	t.Parallel()

	matcher := routing.NewPatternMatcher(
		&mockEmbedder{embedFn: func(_ context.Context, _ string) ([]float32, error) { return nil, nil }},
		&mockPatternStore{findSimilarByIDsFn: func(_ context.Context, _ []float32, _ []uuid.UUID, _ float64) ([]routing.PatternMatch, error) {
			return nil, nil
		}},
		0.7,
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	config := routingrule.PatternMatchConfig{
		PatternIDs: []uuid.UUID{uuid.New()},
	}

	result, err := matcher.Match(ctx, "write go code", config)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.False(t, result.Matched)
}

func TestPatternMatcher_Match_ContextDeadlineExceeded(t *testing.T) {
	t.Parallel()

	matcher := routing.NewPatternMatcher(
		&mockEmbedder{embedFn: func(_ context.Context, _ string) ([]float32, error) { return nil, nil }},
		&mockPatternStore{findSimilarByIDsFn: func(_ context.Context, _ []float32, _ []uuid.UUID, _ float64) ([]routing.PatternMatch, error) {
			return nil, nil
		}},
		0.7,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()

	config := routingrule.PatternMatchConfig{
		PatternIDs: []uuid.UUID{uuid.New()},
	}

	result, err := matcher.Match(ctx, "write go code", config)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.False(t, result.Matched)
}

func TestPatternMatcher_RegistersInRegistry(t *testing.T) {
	t.Parallel()

	registry := routing.NewMatcherRegistry()
	matcher := routing.NewPatternMatcher(
		&mockEmbedder{embedFn: func(_ context.Context, _ string) ([]float32, error) { return nil, nil }},
		&mockPatternStore{findSimilarByIDsFn: func(_ context.Context, _ []float32, _ []uuid.UUID, _ float64) ([]routing.PatternMatch, error) {
			return nil, nil
		}},
		0.7,
	)

	registry.Register(matcher)

	got := registry.GetMatcher(routing.MatchTypePattern)
	require.NotNil(t, got)
	assert.Equal(t, routing.MatchTypePattern, got.Type())
}

func TestPatternMatcher_Close_Idempotent(t *testing.T) {
	t.Parallel()

	matcher := routing.NewPatternMatcher(
		&mockEmbedder{embedFn: func(_ context.Context, _ string) ([]float32, error) { return nil, nil }},
		&mockPatternStore{findSimilarByIDsFn: func(_ context.Context, _ []float32, _ []uuid.UUID, _ float64) ([]routing.PatternMatch, error) {
			return nil, nil
		}},
		0.7,
	)

	// Calling Close multiple times must not panic.
	matcher.Close()
	matcher.Close()
	matcher.Close()
}

func TestPatternMatcher_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	patternID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	matcher := routing.NewPatternMatcher(
		&mockEmbedder{
			embedFn: func(_ context.Context, _ string) ([]float32, error) {
				return []float32{0.1, 0.2, 0.3}, nil
			},
		},
		&mockPatternStore{
			findSimilarByIDsFn: func(_ context.Context, _ []float32, _ []uuid.UUID, _ float64) ([]routing.PatternMatch, error) {
				return []routing.PatternMatch{
					{PatternID: patternID, Similarity: 0.85},
				}, nil
			},
		},
		0.7,
	)

	config := routingrule.PatternMatchConfig{
		PatternIDs: []uuid.UUID{patternID},
	}

	prompts := []string{
		"write go code",
		"build a rust service",
		"python script for data",
		"java enterprise beans",
	}

	for i := range 10 {
		t.Run(fmt.Sprintf("goroutine-%d", i), func(t *testing.T) {
			t.Parallel()
			prompt := prompts[i%len(prompts)]
			_, err := matcher.Match(context.Background(), prompt, config)
			assert.NoError(t, err)
		})
	}
}
