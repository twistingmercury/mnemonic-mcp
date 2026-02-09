package routing_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twistingmercury/mnemonic/internal/repository/routingrule"
	"github.com/twistingmercury/mnemonic/internal/routing"
)

// newTestEngine creates an Engine for testing with the given rules and registry.
func newTestEngine(t *testing.T, rules []*routingrule.Rule, registry *routing.MatcherRegistry, defaultAgent string) *routing.Engine {
	t.Helper()

	loader := &mockRuleLoader{
		loadFn: func(_ context.Context) ([]*routingrule.Rule, error) {
			return rules, nil
		},
	}

	cache, err := routing.NewRuleCache(context.Background(), loader)
	require.NoError(t, err)

	logger := zerolog.Nop()

	return routing.NewEngine(cache, registry, defaultAgent, nil, logger)
}

func TestEngine_Route(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		rules             []*routingrule.Rule
		matchers          []routing.RuleMatcher
		defaultAgent      string
		prompt            string
		wantAgent         string
		wantMatchType     routing.MatchType
		wantConfidence    float64
		wantErr           bool
		wantKeywordsLen   int
		wantReasonContain string
	}{
		{
			name: "first rule matches (short circuit)",
			rules: []*routingrule.Rule{
				{
					ID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Name:        "high-priority",
					Priority:    100,
					AgentName:   "go-agent",
					MatchType:   "keyword",
					MatchConfig: routingrule.KeywordMatchConfig{Keywords: []string{"go"}, MatchMode: "any"},
					Enabled:     true,
				},
				{
					ID:          uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Name:        "low-priority",
					Priority:    50,
					AgentName:   "python-agent",
					MatchType:   "keyword",
					MatchConfig: routingrule.KeywordMatchConfig{Keywords: []string{"python"}, MatchMode: "any"},
					Enabled:     true,
				},
			},
			matchers: []routing.RuleMatcher{
				&mockRuleMatcher{
					matchType: routing.MatchTypeKeyword,
					matchFn: func(_ context.Context, _ string, _ routingrule.MatchConfig) (routing.MatchResult, error) {
						return routing.MatchResult{
							Matched:         true,
							Confidence:      1.0,
							MatchedKeywords: []string{"go"},
							Details:         "keyword match",
						}, nil
					},
				},
			},
			defaultAgent:      "general-agent",
			prompt:            "write Go code",
			wantAgent:         "go-agent",
			wantMatchType:     routing.MatchTypeKeyword,
			wantConfidence:    1.0,
			wantKeywordsLen:   1,
			wantReasonContain: "Matched keywords: go",
		},
		{
			name: "second rule matches when first does not",
			rules: []*routingrule.Rule{
				{
					ID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Name:        "high-priority",
					Priority:    100,
					AgentName:   "go-agent",
					MatchType:   "keyword",
					MatchConfig: routingrule.KeywordMatchConfig{Keywords: []string{"go"}, MatchMode: "any"},
					Enabled:     true,
				},
				{
					ID:          uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Name:        "mid-priority",
					Priority:    50,
					AgentName:   "python-agent",
					MatchType:   "regex",
					MatchConfig: routingrule.RegexMatchConfig{Pattern: `python`, Flags: "i"},
					Enabled:     true,
				},
			},
			matchers: []routing.RuleMatcher{
				&mockRuleMatcher{
					matchType: routing.MatchTypeKeyword,
					matchFn: func(_ context.Context, _ string, _ routingrule.MatchConfig) (routing.MatchResult, error) {
						return routing.MatchResult{Matched: false}, nil
					},
				},
				&mockRuleMatcher{
					matchType: routing.MatchTypeRegex,
					matchFn: func(_ context.Context, _ string, _ routingrule.MatchConfig) (routing.MatchResult, error) {
						return routing.MatchResult{
							Matched:    true,
							Confidence: 1.0,
							Details:    `python`,
						}, nil
					},
				},
			},
			defaultAgent:      "general-agent",
			prompt:            "write Python code",
			wantAgent:         "python-agent",
			wantMatchType:     routing.MatchTypeRegex,
			wantConfidence:    1.0,
			wantReasonContain: "Matched regex pattern: python",
		},
		{
			name: "falls through to default when no rules match",
			rules: []*routingrule.Rule{
				{
					ID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Name:        "keyword-rule",
					Priority:    100,
					AgentName:   "go-agent",
					MatchType:   "keyword",
					MatchConfig: routingrule.KeywordMatchConfig{Keywords: []string{"go"}, MatchMode: "any"},
					Enabled:     true,
				},
			},
			matchers: []routing.RuleMatcher{
				&mockRuleMatcher{
					matchType: routing.MatchTypeKeyword,
					matchFn: func(_ context.Context, _ string, _ routingrule.MatchConfig) (routing.MatchResult, error) {
						return routing.MatchResult{Matched: false}, nil
					},
				},
			},
			defaultAgent:      "general-agent",
			prompt:            "help me with something",
			wantAgent:         "general-agent",
			wantMatchType:     routing.MatchTypeDefault,
			wantConfidence:    0.5,
			wantReasonContain: "No specific rules matched",
		},
		{
			name:              "no rules at all returns default decision",
			rules:             []*routingrule.Rule{},
			matchers:          []routing.RuleMatcher{},
			defaultAgent:      "general-agent",
			prompt:            "anything",
			wantAgent:         "general-agent",
			wantMatchType:     routing.MatchTypeDefault,
			wantConfidence:    0.5,
			wantReasonContain: "No specific rules matched",
		},
		{
			name: "matcher error skips rule and continues",
			rules: []*routingrule.Rule{
				{
					ID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Name:        "broken-rule",
					Priority:    100,
					AgentName:   "broken-agent",
					MatchType:   "regex",
					MatchConfig: routingrule.RegexMatchConfig{Pattern: `(invalid`, Flags: ""},
					Enabled:     true,
				},
				{
					ID:          uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Name:        "fallback-rule",
					Priority:    50,
					AgentName:   "fallback-agent",
					MatchType:   "keyword",
					MatchConfig: routingrule.KeywordMatchConfig{Keywords: []string{"help"}, MatchMode: "any"},
					Enabled:     true,
				},
			},
			matchers: []routing.RuleMatcher{
				&mockRuleMatcher{
					matchType: routing.MatchTypeRegex,
					matchFn: func(_ context.Context, _ string, _ routingrule.MatchConfig) (routing.MatchResult, error) {
						return routing.MatchResult{}, errors.New("invalid regex pattern")
					},
				},
				&mockRuleMatcher{
					matchType: routing.MatchTypeKeyword,
					matchFn: func(_ context.Context, _ string, _ routingrule.MatchConfig) (routing.MatchResult, error) {
						return routing.MatchResult{
							Matched:         true,
							Confidence:      1.0,
							MatchedKeywords: []string{"help"},
						}, nil
					},
				},
			},
			defaultAgent:      "general-agent",
			prompt:            "help me",
			wantAgent:         "fallback-agent",
			wantMatchType:     routing.MatchTypeKeyword,
			wantConfidence:    1.0,
			wantKeywordsLen:   1,
			wantReasonContain: "Matched keywords: help",
		},
		{
			name: "unknown match type skips rule and continues",
			rules: []*routingrule.Rule{
				{
					ID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Name:        "unknown-type-rule",
					Priority:    100,
					AgentName:   "unknown-agent",
					MatchType:   "nonexistent",
					MatchConfig: routingrule.DefaultMatchConfig{},
					Enabled:     true,
				},
				{
					ID:          uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Name:        "known-rule",
					Priority:    50,
					AgentName:   "known-agent",
					MatchType:   "keyword",
					MatchConfig: routingrule.KeywordMatchConfig{Keywords: []string{"test"}, MatchMode: "any"},
					Enabled:     true,
				},
			},
			matchers: []routing.RuleMatcher{
				&mockRuleMatcher{
					matchType: routing.MatchTypeKeyword,
					matchFn: func(_ context.Context, _ string, _ routingrule.MatchConfig) (routing.MatchResult, error) {
						return routing.MatchResult{
							Matched:         true,
							Confidence:      1.0,
							MatchedKeywords: []string{"test"},
						}, nil
					},
				},
			},
			defaultAgent:      "general-agent",
			prompt:            "test something",
			wantAgent:         "known-agent",
			wantMatchType:     routing.MatchTypeKeyword,
			wantConfidence:    1.0,
			wantKeywordsLen:   1,
			wantReasonContain: "Matched keywords: test",
		},
		{
			name: "prompt normalization - uppercase and whitespace",
			rules: []*routingrule.Rule{
				{
					ID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Name:        "keyword-rule",
					Priority:    100,
					AgentName:   "go-agent",
					MatchType:   "keyword",
					MatchConfig: routingrule.KeywordMatchConfig{Keywords: []string{"go"}, MatchMode: "any"},
					Enabled:     true,
				},
			},
			matchers: []routing.RuleMatcher{
				&mockRuleMatcher{
					matchType: routing.MatchTypeKeyword,
					matchFn: func(_ context.Context, prompt string, _ routingrule.MatchConfig) (routing.MatchResult, error) {
						// Verify the prompt is normalized (lowercase + trimmed).
						if prompt != "write go code" {
							return routing.MatchResult{Matched: false}, nil
						}
						return routing.MatchResult{
							Matched:         true,
							Confidence:      1.0,
							MatchedKeywords: []string{"go"},
						}, nil
					},
				},
			},
			defaultAgent:      "general-agent",
			prompt:            "  Write GO Code  ",
			wantAgent:         "go-agent",
			wantMatchType:     routing.MatchTypeKeyword,
			wantConfidence:    1.0,
			wantKeywordsLen:   1,
			wantReasonContain: "Matched keywords: go",
		},
		{
			name: "pattern match with fractional confidence",
			rules: []*routingrule.Rule{
				{
					ID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Name:        "pattern-rule",
					Priority:    50,
					AgentName:   "pattern-agent",
					MatchType:   "pattern",
					MatchConfig: routingrule.PatternMatchConfig{PatternIDs: []uuid.UUID{uuid.New()}},
					Enabled:     true,
				},
			},
			matchers: []routing.RuleMatcher{
				&mockRuleMatcher{
					matchType: routing.MatchTypePattern,
					matchFn: func(_ context.Context, _ string, _ routingrule.MatchConfig) (routing.MatchResult, error) {
						return routing.MatchResult{
							Matched:    true,
							Confidence: 0.87,
							Details:    "semantic similarity",
						}, nil
					},
				},
			},
			defaultAgent:      "general-agent",
			prompt:            "error handling patterns",
			wantAgent:         "pattern-agent",
			wantMatchType:     routing.MatchTypePattern,
			wantConfidence:    0.87,
			wantReasonContain: "Semantic match with confidence 87%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			registry := routing.NewMatcherRegistry()
			for _, m := range tt.matchers {
				registry.Register(m)
			}

			engine := newTestEngine(t, tt.rules, registry, tt.defaultAgent)

			decision, err := engine.Route(context.Background(), routing.Request{
				Prompt: tt.prompt,
			})

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantAgent, decision.AgentName)
			assert.Equal(t, tt.wantMatchType, decision.MatchType)
			assert.InDelta(t, tt.wantConfidence, decision.Confidence, 0.001)
			assert.Contains(t, decision.Reasoning, tt.wantReasonContain)

			if tt.wantKeywordsLen > 0 {
				assert.Len(t, decision.MatchedKeywords, tt.wantKeywordsLen)
			}
		})
	}
}

func TestEngine_Route_ShortCircuit(t *testing.T) {
	t.Parallel()

	// Verify the second matcher is NOT called when the first matches.
	var secondMatcherCalled atomic.Bool

	rules := []*routingrule.Rule{
		{
			ID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			Name:        "first-rule",
			Priority:    100,
			AgentName:   "first-agent",
			MatchType:   "keyword",
			MatchConfig: routingrule.KeywordMatchConfig{Keywords: []string{"go"}, MatchMode: "any"},
			Enabled:     true,
		},
		{
			ID:          uuid.MustParse("00000000-0000-0000-0000-000000000002"),
			Name:        "second-rule",
			Priority:    50,
			AgentName:   "second-agent",
			MatchType:   "regex",
			MatchConfig: routingrule.RegexMatchConfig{Pattern: `go`},
			Enabled:     true,
		},
	}

	registry := routing.NewMatcherRegistry()
	registry.Register(&mockRuleMatcher{
		matchType: routing.MatchTypeKeyword,
		matchFn: func(_ context.Context, _ string, _ routingrule.MatchConfig) (routing.MatchResult, error) {
			return routing.MatchResult{
				Matched:         true,
				Confidence:      1.0,
				MatchedKeywords: []string{"go"},
			}, nil
		},
	})
	registry.Register(&mockRuleMatcher{
		matchType: routing.MatchTypeRegex,
		matchFn: func(_ context.Context, _ string, _ routingrule.MatchConfig) (routing.MatchResult, error) {
			secondMatcherCalled.Store(true)
			return routing.MatchResult{Matched: true, Confidence: 1.0}, nil
		},
	})

	engine := newTestEngine(t, rules, registry, "general-agent")

	decision, err := engine.Route(context.Background(), routing.Request{
		Prompt: "write Go code",
	})

	require.NoError(t, err)
	assert.Equal(t, "first-agent", decision.AgentName)
	assert.False(t, secondMatcherCalled.Load(), "second matcher should NOT be called due to short-circuit")
}

func TestEngine_Route_NilMetrics(t *testing.T) {
	t.Parallel()

	// Verify that nil metrics does not cause a panic.
	rules := []*routingrule.Rule{
		{
			ID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			Name:        "test-rule",
			Priority:    100,
			AgentName:   "test-agent",
			MatchType:   "keyword",
			MatchConfig: routingrule.KeywordMatchConfig{Keywords: []string{"test"}, MatchMode: "any"},
			Enabled:     true,
		},
	}

	registry := routing.NewMatcherRegistry()
	registry.Register(&mockRuleMatcher{
		matchType: routing.MatchTypeKeyword,
		matchFn: func(_ context.Context, _ string, _ routingrule.MatchConfig) (routing.MatchResult, error) {
			return routing.MatchResult{
				Matched:         true,
				Confidence:      1.0,
				MatchedKeywords: []string{"test"},
			}, nil
		},
	})

	// Engine created with nil metrics via newTestEngine (which passes nil).
	engine := newTestEngine(t, rules, registry, "general-agent")

	// Should not panic.
	decision, err := engine.Route(context.Background(), routing.Request{
		Prompt: "test",
	})

	require.NoError(t, err)
	assert.Equal(t, "test-agent", decision.AgentName)
}

func TestEngine_Route_DisabledRulesSkipped(t *testing.T) {
	t.Parallel()

	rules := []*routingrule.Rule{
		{
			ID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			Name:        "disabled-rule",
			Priority:    100,
			AgentName:   "disabled-agent",
			MatchType:   "keyword",
			MatchConfig: routingrule.KeywordMatchConfig{Keywords: []string{"go"}, MatchMode: "any"},
			Enabled:     false, // disabled
		},
		{
			ID:          uuid.MustParse("00000000-0000-0000-0000-000000000002"),
			Name:        "enabled-rule",
			Priority:    50,
			AgentName:   "enabled-agent",
			MatchType:   "keyword",
			MatchConfig: routingrule.KeywordMatchConfig{Keywords: []string{"go"}, MatchMode: "any"},
			Enabled:     true,
		},
	}

	registry := routing.NewMatcherRegistry()
	registry.Register(&mockRuleMatcher{
		matchType: routing.MatchTypeKeyword,
		matchFn: func(_ context.Context, _ string, _ routingrule.MatchConfig) (routing.MatchResult, error) {
			return routing.MatchResult{
				Matched:         true,
				Confidence:      1.0,
				MatchedKeywords: []string{"go"},
			}, nil
		},
	})

	engine := newTestEngine(t, rules, registry, "general-agent")

	decision, err := engine.Route(context.Background(), routing.Request{
		Prompt: "write Go code",
	})

	require.NoError(t, err)
	assert.Equal(t, "enabled-agent", decision.AgentName, "disabled rule should be skipped")
}

func TestEngine_Route_ContextCancellation(t *testing.T) {
	t.Parallel()

	rules := []*routingrule.Rule{
		{
			ID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			Name:        "test-rule",
			Priority:    100,
			AgentName:   "test-agent",
			MatchType:   "keyword",
			MatchConfig: routingrule.KeywordMatchConfig{Keywords: []string{"go"}, MatchMode: "any"},
			Enabled:     true,
		},
	}

	registry := routing.NewMatcherRegistry()
	registry.Register(&mockRuleMatcher{
		matchType: routing.MatchTypeKeyword,
		matchFn: func(_ context.Context, _ string, _ routingrule.MatchConfig) (routing.MatchResult, error) {
			// This should never be reached because the engine checks ctx.Err()
			// before evaluating each rule.
			return routing.MatchResult{Matched: true, Confidence: 1.0}, nil
		},
	})

	engine := newTestEngine(t, rules, registry, "general-agent")

	// Create a cancelled context.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// The engine detects the cancelled context at the top of the rule loop
	// and returns the context error immediately.
	decision, err := engine.Route(ctx, routing.Request{
		Prompt: "write Go code",
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, routing.Decision{}, decision)
}
