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

// RegexMatcher implements RuleMatcher for regex-based routing rules.
// It compiles and caches regular expression patterns for performance.
// Compiled patterns are stored with a sliding TTL and evicted by a
// background goroutine. Call Close to stop the cleanup goroutine when
// the matcher is no longer needed.
//
// Matching behavior:
//   - Patterns use standard Go regex syntax (RE2).
//   - Supported flags: "i" (case-insensitive) applied as (?i) prefix.
//   - Matching is unanchored: the pattern may match anywhere in the prompt.
//   - Confidence is always 1.0 (deterministic binary match).
type RegexMatcher struct {
	mu       sync.RWMutex
	patterns map[string]*cachedRegex
	ttl      time.Duration
	done     chan struct{}
	nowFunc  func() time.Time // injectable clock for testing
}

// cachedRegex holds a compiled regex along with its last access timestamp
// for sliding TTL eviction.
type cachedRegex struct {
	re       *regexp.Regexp
	lastUsed time.Time
}

// NewRegexMatcher creates a new RegexMatcher with a background goroutine
// that periodically evicts expired cache entries. Call Close to stop it.
func NewRegexMatcher() *RegexMatcher {
	m := &RegexMatcher{
		patterns: make(map[string]*cachedRegex),
		ttl:      patternTTL,
		done:     make(chan struct{}),
		nowFunc:  time.Now,
	}
	go m.cleanupLoop(patternCleanupInterval)
	return m
}

// newRegexMatcherForTest creates a RegexMatcher with a custom TTL and clock
// function for deterministic testing. The cleanup loop is NOT started; callers
// must invoke cleanExpiredRegexPatterns directly.
//
//nolint:unusedfunc // referenced via export_test.go (ExportNewRegexMatcherForTest)
func newRegexMatcherForTest(ttl time.Duration, nowFunc func() time.Time) *RegexMatcher {
	return &RegexMatcher{
		patterns: make(map[string]*cachedRegex),
		ttl:      ttl,
		done:     make(chan struct{}),
		nowFunc:  nowFunc,
	}
}

// Close stops the background cleanup goroutine. It is safe to call multiple
// times; subsequent calls are no-ops.
func (m *RegexMatcher) Close() {
	select {
	case <-m.done:
		// Already closed.
	default:
		close(m.done)
	}
}

// Type returns the MatchType this matcher handles.
func (m *RegexMatcher) Type() MatchType {
	return MatchTypeRegex
}

// Match evaluates the prompt against the regex match configuration.
// The prompt is expected to be trimmed by the engine but NOT lowercased;
// case-sensitivity is controlled by the "i" flag in the match configuration.
//
// Matching behavior:
//   - The pattern is compiled once and cached for subsequent calls.
//   - If the "i" flag is set, the pattern is prefixed with (?i) for case-insensitive matching.
//   - An empty pattern returns no match.
//   - An invalid pattern returns an error (the engine will skip the rule and log a warning).
//   - Confidence is always 1.0 for matches (deterministic binary match).
func (m *RegexMatcher) Match(ctx context.Context, prompt string, config routingrule.MatchConfig) (MatchResult, error) {
	if err := ctx.Err(); err != nil {
		return MatchResult{}, fmt.Errorf("regex matcher: %w", err)
	}

	rxConfig, ok := config.(routingrule.RegexMatchConfig)
	if !ok {
		return MatchResult{}, fmt.Errorf("regex matcher: expected RegexMatchConfig, got %T", config)
	}

	if rxConfig.Pattern == "" {
		return MatchResult{Matched: false}, nil
	}

	re, err := m.getOrCompile(rxConfig.Pattern, rxConfig.Flags)
	if err != nil {
		return MatchResult{}, fmt.Errorf("regex matcher: %w", err)
	}

	if re.MatchString(prompt) {
		return MatchResult{
			Matched:    true,
			Confidence: 1.0,
			Details:    rxConfig.Pattern,
		}, nil
	}

	return MatchResult{Matched: false}, nil
}

// getOrCompile returns a cached compiled regex for the given pattern and flags,
// compiling and caching it on first access. The cache key format is "flags:pattern".
// On cache hit the entry's lastUsed timestamp is refreshed (sliding TTL).
func (m *RegexMatcher) getOrCompile(pattern, flags string) (*regexp.Regexp, error) {
	key := flags + ":" + pattern

	m.mu.RLock()
	entry, exists := m.patterns[key]
	m.mu.RUnlock()

	if exists {
		// Refresh the sliding TTL under a write lock.
		m.mu.Lock()
		entry.lastUsed = m.nowFunc()
		m.mu.Unlock()
		return entry.re, nil
	}

	// Apply flag prefix before compilation.
	fullPattern, err := applyRegexFlags(pattern, flags)
	if err != nil {
		return nil, fmt.Errorf("applying regex flags: %w", err)
	}

	re, err := regexp.Compile(fullPattern)
	if err != nil {
		return nil, fmt.Errorf("compiling regex pattern %q (flags %q): %w", pattern, flags, err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.patterns[key] = &cachedRegex{
		re:       re,
		lastUsed: m.nowFunc(),
	}

	return re, nil
}

// applyRegexFlags prepends inline flag syntax to the pattern based on the
// flags string. Currently supported flags:
//   - "i": case-insensitive matching via (?i) prefix
//
// An error is returned if flags contains any unsupported character.
func applyRegexFlags(pattern, flags string) (string, error) {
	var b strings.Builder
	for _, f := range flags {
		switch f {
		case 'i':
			b.WriteString("(?i)")
		default:
			return "", fmt.Errorf("unsupported regex flag %q", string(f))
		}
	}
	b.WriteString(pattern)
	return b.String(), nil
}

// cleanupLoop runs cleanExpiredRegexPatterns on a regular interval until Close is called.
func (m *RegexMatcher) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-m.done:
			return
		case <-ticker.C:
			m.cleanExpiredRegexPatterns()
		}
	}
}

// cleanExpiredRegexPatterns removes cache entries whose lastUsed time is older
// than the configured TTL relative to now.
func (m *RegexMatcher) cleanExpiredRegexPatterns() {
	now := m.nowFunc()
	m.mu.Lock()
	defer m.mu.Unlock()

	for key, entry := range m.patterns {
		if now.Sub(entry.lastUsed) > m.ttl {
			delete(m.patterns, key)
		}
	}
}
