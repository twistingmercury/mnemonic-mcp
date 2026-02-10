package routing

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/twistingmercury/mnemonic/internal/repository/routingrule"
)

const (
	// patternTTL is the sliding time-to-live for cached compiled regex patterns.
	// A pattern's TTL resets on every cache hit.
	patternTTL = 30 * time.Minute

	// patternCleanupInterval is how often the background goroutine scans for expired entries.
	patternCleanupInterval = 5 * time.Minute
)

// cachedPattern holds a compiled regex along with its last access timestamp
// for sliding TTL eviction.
type cachedPattern struct {
	re       *regexp.Regexp
	lastUsed time.Time
}

// KeywordMatcher implements RuleMatcher for keyword-based routing rules.
// It supports single-word matching with word boundary awareness and multi-word
// phrase matching via substring search. All matching is case-insensitive.
//
// Compiled regex patterns are cached with a 30-minute sliding TTL. Call Close
// to stop the background cleanup goroutine when the matcher is no longer needed.
type KeywordMatcher struct {
	mu       sync.RWMutex
	patterns map[string]*cachedPattern
	ttl      time.Duration
	done     chan struct{}
	nowFunc  func() time.Time // injectable clock for testing
}

// NewKeywordMatcher creates a new KeywordMatcher with a background goroutine
// that periodically evicts expired cache entries. Call Close to stop it.
func NewKeywordMatcher() *KeywordMatcher {
	m := &KeywordMatcher{
		patterns: make(map[string]*cachedPattern),
		ttl:      patternTTL,
		done:     make(chan struct{}),
		nowFunc:  time.Now,
	}
	go m.cleanupLoop(patternCleanupInterval)
	return m
}

// newKeywordMatcherForTest creates a KeywordMatcher with a custom TTL and clock
// function for deterministic testing. The cleanup loop is NOT started; callers
// must invoke cleanExpiredPatterns directly.
func newKeywordMatcherForTest(ttl time.Duration, nowFunc func() time.Time) *KeywordMatcher {
	return &KeywordMatcher{
		patterns: make(map[string]*cachedPattern),
		ttl:      ttl,
		done:     make(chan struct{}),
		nowFunc:  nowFunc,
	}
}

// Close stops the background cleanup goroutine. It is safe to call multiple
// times; subsequent calls are no-ops.
func (m *KeywordMatcher) Close() {
	select {
	case <-m.done:
		// Already closed.
	default:
		close(m.done)
	}
}

// Type returns the MatchType this matcher handles.
func (m *KeywordMatcher) Type() MatchType {
	return MatchTypeKeyword
}

// Match evaluates the normalized prompt against the keyword match configuration.
// The prompt is expected to be already normalized (lowercased and trimmed) by the engine.
//
// Matching behavior:
//   - Single-word keywords use word boundary regex (\bkeyword\b) to prevent
//     partial matches (e.g., "go" does not match "mango").
//   - Multi-word keywords (containing spaces) use substring matching.
//   - Match mode "any": returns true if any keyword matches.
//   - Match mode "all": returns true only if all keywords match.
//   - Confidence is always 1.0 for matches (deterministic).
func (m *KeywordMatcher) Match(ctx context.Context, prompt string, config routingrule.MatchConfig) (MatchResult, error) {
	if err := ctx.Err(); err != nil {
		return MatchResult{}, fmt.Errorf("keyword matcher: %w", err)
	}

	kwConfig, ok := config.(routingrule.KeywordMatchConfig)
	if !ok {
		return MatchResult{}, fmt.Errorf("keyword matcher: expected KeywordMatchConfig, got %T", config)
	}

	if len(kwConfig.Keywords) == 0 || prompt == "" {
		return MatchResult{Matched: false}, nil
	}

	var matchedKeywords []string

	for _, keyword := range kwConfig.Keywords {
		if err := ctx.Err(); err != nil {
			return MatchResult{}, fmt.Errorf("keyword matcher: %w", err)
		}

		lowerKeyword := strings.ToLower(keyword)
		matched, err := m.containsKeyword(prompt, lowerKeyword)
		if err != nil {
			return MatchResult{}, err
		}
		if matched {
			matchedKeywords = append(matchedKeywords, keyword)
		}
	}

	matched := false
	switch kwConfig.MatchMode {
	case routingrule.MatchModeAll:
		matched = len(matchedKeywords) == len(kwConfig.Keywords)
	default:
		// MatchModeAny is the default mode.
		matched = len(matchedKeywords) > 0
	}

	if !matched {
		return MatchResult{Matched: false}, nil
	}

	return MatchResult{
		Matched:         true,
		Confidence:      1.0,
		MatchedKeywords: matchedKeywords,
	}, nil
}

// wordBoundarySafePattern matches strings that consist entirely of regex "word"
// characters (letters, digits, underscore). Only these keywords can safely use
// \b word boundary matching; keywords containing non-word characters (e.g., "c++",
// "func()") must fall back to substring matching.
var wordBoundarySafePattern = regexp.MustCompile(`^\w+$`)

// containsKeyword checks whether the keyword appears in the prompt.
// It returns true if the keyword is found and an error if the underlying
// regex pattern fails to compile.
//   - Multi-word keywords (containing spaces) use substring matching.
//   - Single-word keywords composed entirely of word characters (\w) use
//     word boundary regex to prevent partial matches.
//   - Single-word keywords containing non-word characters (e.g., "c++",
//     "func()") fall back to substring matching because \b boundaries do
//     not work correctly at non-word character edges.
func (m *KeywordMatcher) containsKeyword(prompt, keyword string) (bool, error) {
	if strings.Contains(keyword, " ") {
		return strings.Contains(prompt, keyword), nil
	}
	if !wordBoundarySafePattern.MatchString(keyword) {
		return strings.Contains(prompt, keyword), nil
	}
	return m.matchWordBoundary(prompt, keyword)
}

// matchWordBoundary checks if a single word appears in the prompt with word boundaries.
// It uses a compiled regex cache to avoid recompiling patterns on every call.
// It returns an error if the regex pattern fails to compile.
func (m *KeywordMatcher) matchWordBoundary(prompt, keyword string) (bool, error) {
	re, err := m.getOrCompilePattern(keyword)
	if err != nil {
		return false, fmt.Errorf("keyword matcher: word boundary match: %w", err)
	}
	return re.MatchString(prompt), nil
}

// getOrCompilePattern returns a cached compiled regex for the given keyword,
// compiling and caching it on first access. The pattern uses \b word boundaries.
// On cache hit the entry's lastUsed timestamp is refreshed (sliding TTL).
func (m *KeywordMatcher) getOrCompilePattern(keyword string) (*regexp.Regexp, error) {
	m.mu.RLock()
	entry, exists := m.patterns[keyword]
	m.mu.RUnlock()

	if exists {
		// Refresh the sliding TTL under a write lock.
		m.mu.Lock()
		entry.lastUsed = m.nowFunc()
		m.mu.Unlock()
		return entry.re, nil
	}

	// Escape any regex metacharacters in the keyword before wrapping with \b.
	escaped := regexp.QuoteMeta(keyword)
	pattern := `\b` + escaped + `\b`

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("compiling keyword pattern %q: %w", keyword, err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.patterns[keyword] = &cachedPattern{
		re:       re,
		lastUsed: m.nowFunc(),
	}

	return re, nil
}

// cleanupLoop runs cleanExpiredPatterns on a regular interval until Close is called.
func (m *KeywordMatcher) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-m.done:
			return
		case <-ticker.C:
			m.cleanExpiredPatterns()
		}
	}
}

// cleanExpiredPatterns removes cache entries whose lastUsed time is older than
// the configured TTL relative to now.
func (m *KeywordMatcher) cleanExpiredPatterns() {
	now := m.nowFunc()
	m.mu.Lock()
	defer m.mu.Unlock()

	for key, entry := range m.patterns {
		if now.Sub(entry.lastUsed) > m.ttl {
			delete(m.patterns, key)
		}
	}
}
