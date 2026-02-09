package routing_test

import (
	"context"

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

// mockRuleLoader is a hand-rolled mock for the RuleLoader interface.
type mockRuleLoader struct {
	loadFn func(ctx context.Context) ([]*routingrule.RoutingRule, error)
}

func (m *mockRuleLoader) LoadRules(ctx context.Context) ([]*routingrule.RoutingRule, error) {
	return m.loadFn(ctx)
}
