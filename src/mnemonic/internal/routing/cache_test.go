package routing_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twistingmercury/mnemonic/internal/repository/routingrule"
	"github.com/twistingmercury/mnemonic/internal/routing"
)

func TestNewRuleCache(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		loader         routing.RuleLoader
		wantErr        bool
		wantErrContain string
		wantCount      int
		wantOrder      []string // expected rule names in sorted order
	}{
		{
			name: "successful load with sorting by priority descending",
			loader: &mockRuleLoader{
				loadFn: func(_ context.Context) ([]*routingrule.RoutingRule, error) {
					return []*routingrule.RoutingRule{
						{ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"), Name: "low", Priority: 0, Enabled: true},
						{ID: uuid.MustParse("00000000-0000-0000-0000-000000000002"), Name: "high", Priority: 100, Enabled: true},
						{ID: uuid.MustParse("00000000-0000-0000-0000-000000000003"), Name: "mid", Priority: 50, Enabled: true},
					}, nil
				},
			},
			wantCount: 3,
			wantOrder: []string{"high", "mid", "low"},
		},
		{
			name: "tie-breaking by ID ascending when priorities are equal",
			loader: &mockRuleLoader{
				loadFn: func(_ context.Context) ([]*routingrule.RoutingRule, error) {
					return []*routingrule.RoutingRule{
						{ID: uuid.MustParse("cccccccc-0000-0000-0000-000000000000"), Name: "rule-c", Priority: 50, Enabled: true},
						{ID: uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000000"), Name: "rule-a", Priority: 50, Enabled: true},
						{ID: uuid.MustParse("bbbbbbbb-0000-0000-0000-000000000000"), Name: "rule-b", Priority: 50, Enabled: true},
					}, nil
				},
			},
			wantCount: 3,
			wantOrder: []string{"rule-a", "rule-b", "rule-c"},
		},
		{
			name: "load error returns error (fail-fast)",
			loader: &mockRuleLoader{
				loadFn: func(_ context.Context) ([]*routingrule.RoutingRule, error) {
					return nil, errors.New("database connection refused")
				},
			},
			wantErr:        true,
			wantErrContain: "failed to load rules at startup",
		},
		{
			name: "empty rules returns empty cache",
			loader: &mockRuleLoader{
				loadFn: func(_ context.Context) ([]*routingrule.RoutingRule, error) {
					return []*routingrule.RoutingRule{}, nil
				},
			},
			wantCount: 0,
			wantOrder: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cache, err := routing.NewRuleCache(context.Background(), tt.loader)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrContain)
				assert.Nil(t, cache)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, cache)

			assert.Equal(t, tt.wantCount, cache.RuleCount())

			rules := cache.GetRules()
			assert.Len(t, rules, tt.wantCount)

			for i, expectedName := range tt.wantOrder {
				assert.Equal(t, expectedName, rules[i].Name, "rule at index %d", i)
			}
		})
	}
}

func TestRuleCache_GetRules_ReturnsCopy(t *testing.T) {
	t.Parallel()

	loader := &mockRuleLoader{
		loadFn: func(_ context.Context) ([]*routingrule.RoutingRule, error) {
			return []*routingrule.RoutingRule{
				{ID: uuid.New(), Name: "rule-1", Priority: 100, Enabled: true},
			}, nil
		},
	}

	cache, err := routing.NewRuleCache(context.Background(), loader)
	require.NoError(t, err)

	// Get rules and replace an entry in the returned slice.
	rules := cache.GetRules()
	require.Len(t, rules, 1)
	rules[0] = &routingrule.RoutingRule{Name: "replaced"}

	// Verify the cache is not affected by the slice replacement.
	original := cache.GetRules()
	assert.Equal(t, "rule-1", original[0].Name, "cache should not be mutated by external changes")
}

func TestRuleCache_RuleCount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		ruleCount int
	}{
		{name: "zero rules", ruleCount: 0},
		{name: "one rule", ruleCount: 1},
		{name: "many rules", ruleCount: 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rules := make([]*routingrule.RoutingRule, tt.ruleCount)
			for i := range tt.ruleCount {
				rules[i] = &routingrule.RoutingRule{
					ID:      uuid.New(),
					Name:    "rule",
					Enabled: true,
				}
			}

			loader := &mockRuleLoader{
				loadFn: func(_ context.Context) ([]*routingrule.RoutingRule, error) {
					return rules, nil
				},
			}

			cache, err := routing.NewRuleCache(context.Background(), loader)
			require.NoError(t, err)

			assert.Equal(t, tt.ruleCount, cache.RuleCount())
		})
	}
}
