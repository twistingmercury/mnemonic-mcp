package routing_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twistingmercury/mnemonic/internal/repository/routingrule"
	"github.com/twistingmercury/mnemonic/internal/routing"
)

func TestMatcherRegistry_Register_And_GetMatcher(t *testing.T) {
	t.Parallel()

	registry := routing.NewMatcherRegistry()

	matcher := &mockRuleMatcher{
		matchType: routing.MatchTypeKeyword,
		matchFn: func(_ context.Context, _ string, _ routingrule.MatchConfig) (routing.MatchResult, error) {
			return routing.MatchResult{Matched: true}, nil
		},
	}

	registry.Register(matcher)

	got := registry.GetMatcher(routing.MatchTypeKeyword)
	require.NotNil(t, got)
	assert.Equal(t, routing.MatchTypeKeyword, got.Type())
}

func TestMatcherRegistry_GetMatcher_Unregistered(t *testing.T) {
	t.Parallel()

	registry := routing.NewMatcherRegistry()

	got := registry.GetMatcher(routing.MatchTypeKeyword)
	assert.Nil(t, got, "should return nil for unregistered matcher")
}

func TestMatcherRegistry_Register_Replace(t *testing.T) {
	t.Parallel()

	registry := routing.NewMatcherRegistry()

	// Register first matcher.
	first := &mockRuleMatcher{
		matchType: routing.MatchTypeKeyword,
		matchFn: func(_ context.Context, _ string, _ routingrule.MatchConfig) (routing.MatchResult, error) {
			return routing.MatchResult{Matched: false, Details: "first"}, nil
		},
	}
	registry.Register(first)

	// Register second matcher with the same type (replace).
	second := &mockRuleMatcher{
		matchType: routing.MatchTypeKeyword,
		matchFn: func(_ context.Context, _ string, _ routingrule.MatchConfig) (routing.MatchResult, error) {
			return routing.MatchResult{Matched: true, Details: "second"}, nil
		},
	}
	registry.Register(second)

	// GetMatcher should return the second (replaced) matcher.
	got := registry.GetMatcher(routing.MatchTypeKeyword)
	require.NotNil(t, got)

	result, err := got.Match(context.Background(), "test", nil)
	require.NoError(t, err)
	assert.True(t, result.Matched)
	assert.Equal(t, "second", result.Details)
}

func TestMatcherRegistry_MultipleTypes(t *testing.T) {
	t.Parallel()

	registry := routing.NewMatcherRegistry()

	keywordMatcher := &mockRuleMatcher{matchType: routing.MatchTypeKeyword}
	regexMatcher := &mockRuleMatcher{matchType: routing.MatchTypeRegex}
	patternMatcher := &mockRuleMatcher{matchType: routing.MatchTypePattern}

	registry.Register(keywordMatcher)
	registry.Register(regexMatcher)
	registry.Register(patternMatcher)

	assert.NotNil(t, registry.GetMatcher(routing.MatchTypeKeyword))
	assert.NotNil(t, registry.GetMatcher(routing.MatchTypeRegex))
	assert.NotNil(t, registry.GetMatcher(routing.MatchTypePattern))
	assert.Nil(t, registry.GetMatcher(routing.MatchTypeDefault))
}

func TestMatcherRegistry_CloseAll(t *testing.T) {
	t.Parallel()

	t.Run("closes all registered matchers without panic", func(t *testing.T) {
		t.Parallel()

		registry := routing.NewMatcherRegistry()
		registry.Register(&mockRuleMatcher{matchType: routing.MatchTypeKeyword})
		registry.Register(&mockRuleMatcher{matchType: routing.MatchTypeRegex})
		registry.Register(&mockRuleMatcher{matchType: routing.MatchTypePattern})

		assert.NotPanics(t, func() {
			registry.CloseAll()
		})
	})

	t.Run("idempotent double close does not panic", func(t *testing.T) {
		t.Parallel()

		registry := routing.NewMatcherRegistry()
		registry.Register(&mockRuleMatcher{matchType: routing.MatchTypeKeyword})
		registry.Register(&mockRuleMatcher{matchType: routing.MatchTypeRegex})

		registry.CloseAll()

		assert.NotPanics(t, func() {
			registry.CloseAll()
		})
	})

	t.Run("close all on empty registry does not panic", func(t *testing.T) {
		t.Parallel()

		registry := routing.NewMatcherRegistry()

		assert.NotPanics(t, func() {
			registry.CloseAll()
		})
	})
}
