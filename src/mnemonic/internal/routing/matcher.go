package routing

import (
	"context"
	"sync"

	"github.com/twistingmercury/mnemonic/internal/repository/routingrule"
)

// RuleMatcher defines the interface for match type implementations.
// Each concrete matcher (keyword, regex, pattern) implements this interface.
type RuleMatcher interface {
	// Match evaluates the normalized prompt against the rule's match configuration.
	Match(ctx context.Context, prompt string, config routingrule.MatchConfig) (MatchResult, error)

	// Type returns the MatchType this matcher handles.
	Type() MatchType

	// Close releases resources held by the matcher (e.g., background goroutines).
	// Implementations must be safe to call multiple times.
	Close()
}

// MatcherRegistry manages the mapping from MatchType to RuleMatcher implementations.
// It is safe for concurrent registration and lookup via sync.RWMutex.
type MatcherRegistry struct {
	mu       sync.RWMutex
	matchers map[MatchType]RuleMatcher
}

// NewMatcherRegistry creates a new empty MatcherRegistry.
func NewMatcherRegistry() *MatcherRegistry {
	return &MatcherRegistry{
		matchers: make(map[MatchType]RuleMatcher),
	}
}

// Register adds a RuleMatcher to the registry. If a matcher for the same
// MatchType already exists, it is replaced.
func (r *MatcherRegistry) Register(matcher RuleMatcher) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.matchers[matcher.Type()] = matcher
}

// GetMatcher returns the RuleMatcher for the given MatchType.
// Returns nil if no matcher is registered for the type.
func (r *MatcherRegistry) GetMatcher(t MatchType) RuleMatcher {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.matchers[t]
}

// CloseAll closes all registered matchers, releasing their resources.
// It is safe to call multiple times because each matcher's Close method
// must be idempotent.
func (r *MatcherRegistry) CloseAll() {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, matcher := range r.matchers {
		matcher.Close()
	}
}
