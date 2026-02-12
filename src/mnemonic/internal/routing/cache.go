package routing

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/twistingmercury/mnemonic/internal/repository/routingrule"
)

// RuleLoader defines the interface for loading routing rules from a data source.
// The return type uses pointers to match the routingrule.Repository.ListEnabled()
// signature, allowing a repository to be used directly without conversion.
type RuleLoader interface {
	// LoadRules retrieves all enabled routing rules.
	LoadRules(ctx context.Context) ([]*routingrule.Rule, error)
}

// RuleCache provides an in-memory cache of routing rules, pre-sorted by
// priority (descending) then ID (ascending). It is safe for concurrent access.
type RuleCache struct {
	rules []*routingrule.Rule
	mu    sync.RWMutex
}

// NewRuleCache creates a new RuleCache by loading rules from the provided loader.
// Rules are sorted by priority DESC, then by ID ASC (lexicographic) for deterministic
// tie-breaking. Returns an error if loading fails (fail-fast on startup).
func NewRuleCache(ctx context.Context, loader RuleLoader) (*RuleCache, error) {
	rules, err := loader.LoadRules(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load rules at startup: %w", err)
	}

	// Sort rules: priority descending, then ID ascending for tie-breaking.
	sort.Slice(rules, func(i, j int) bool {
		if rules[i].Priority != rules[j].Priority {
			return rules[i].Priority > rules[j].Priority
		}
		return rules[i].ID.String() < rules[j].ID.String()
	})

	return &RuleCache{rules: rules}, nil
}

// GetRules returns the cached rules. The returned slice is a shallow copy to
// prevent external mutation of the cache. Rules are pre-sorted by priority
// DESC, ID ASC.
//
// Safety invariant: the slice is copied so callers cannot reorder or replace
// entries. The pointers themselves share the underlying Rule structs
// with the cache, but this is safe because all MatchConfig implementations
// (KeywordMatchConfig, RegexMatchConfig, PatternMatchConfig)
// are value types -- structs whose fields are either primitive types or slices
// that are never mutated after construction. If a new MatchConfig type is
// introduced with pointer fields or mutable shared state, this method must be
// updated to perform a deep copy.
func (c *RuleCache) GetRules() []*routingrule.Rule {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Return a shallow copy to prevent external mutation of the slice.
	result := make([]*routingrule.Rule, len(c.rules))
	copy(result, c.rules)
	return result
}

// RuleCount returns the number of cached rules.
func (c *RuleCache) RuleCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.rules)
}
