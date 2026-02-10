package routing_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twistingmercury/mnemonic/internal/repository/routingrule"
	"github.com/twistingmercury/mnemonic/internal/routing"
)

func TestRegexMatcher_Type(t *testing.T) {
	t.Parallel()

	matcher := routing.NewRegexMatcher()
	defer matcher.Close()

	assert.Equal(t, routing.MatchTypeRegex, matcher.Type())
}

func TestRegexMatcher_Match(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		prompt          string
		config          routingrule.MatchConfig
		wantMatched     bool
		wantConfidence  float64
		wantDetails     string
		wantErr         bool
		wantErrContains string
	}{
		// --- Basic pattern matching ---
		{
			name:   "basic pattern match",
			prompt: "write go code for a web server",
			config: routingrule.RegexMatchConfig{
				Pattern: "go.*code",
			},
			wantMatched:    true,
			wantConfidence: 1.0,
			wantDetails:    "go.*code",
		},
		{
			name:   "basic pattern no match",
			prompt: "write python code",
			config: routingrule.RegexMatchConfig{
				Pattern: "rust.*code",
			},
			wantMatched: false,
		},

		// --- Case-insensitive flag ---
		{
			name:   "case-insensitive flag matches uppercase prompt",
			prompt: "Write GO Code",
			config: routingrule.RegexMatchConfig{
				Pattern: "go.*code",
				Flags:   "i",
			},
			wantMatched:    true,
			wantConfidence: 1.0,
			wantDetails:    "go.*code",
		},
		{
			name:   "case-insensitive flag matches mixed case",
			prompt: "GoLang Function",
			config: routingrule.RegexMatchConfig{
				Pattern: "golang function",
				Flags:   "i",
			},
			wantMatched:    true,
			wantConfidence: 1.0,
			wantDetails:    "golang function",
		},

		// --- No flags (case-sensitive) ---
		{
			name:   "no flags - case-sensitive match succeeds",
			prompt: "write go code",
			config: routingrule.RegexMatchConfig{
				Pattern: "go code",
			},
			wantMatched:    true,
			wantConfidence: 1.0,
			wantDetails:    "go code",
		},
		{
			name:   "no flags - case-sensitive match fails on wrong case",
			prompt: "write GO code",
			config: routingrule.RegexMatchConfig{
				Pattern: "go code",
			},
			wantMatched: false,
		},

		// --- Invalid regex pattern ---
		{
			name:   "invalid regex pattern returns error",
			prompt: "any prompt",
			config: routingrule.RegexMatchConfig{
				Pattern: "[invalid",
			},
			wantErr:         true,
			wantErrContains: "compiling regex pattern",
		},

		// --- Wrong config type ---
		{
			name:   "wrong config type returns error",
			prompt: "write go code",
			config: routingrule.KeywordMatchConfig{
				Keywords: []string{"go"},
			},
			wantErr:         true,
			wantErrContains: "expected RegexMatchConfig",
		},

		// --- Empty pattern ---
		{
			name:   "empty pattern returns no match",
			prompt: "write go code",
			config: routingrule.RegexMatchConfig{
				Pattern: "",
			},
			wantMatched: false,
		},

		// --- Empty prompt with valid pattern ---
		{
			name:   "empty prompt with non-matching pattern",
			prompt: "",
			config: routingrule.RegexMatchConfig{
				Pattern: "go",
			},
			wantMatched: false,
		},
		{
			name:   "empty prompt with pattern matching empty string",
			prompt: "",
			config: routingrule.RegexMatchConfig{
				Pattern: ".*",
			},
			wantMatched:    true,
			wantConfidence: 1.0,
			wantDetails:    ".*",
		},

		// --- Complex regex patterns ---
		{
			name:   "character class",
			prompt: "use go1.21 for the build",
			config: routingrule.RegexMatchConfig{
				Pattern: `go[0-9]+\.[0-9]+`,
			},
			wantMatched:    true,
			wantConfidence: 1.0,
			wantDetails:    `go[0-9]+\.[0-9]+`,
		},
		{
			name:   "alternation",
			prompt: "implement a struct for the data model",
			config: routingrule.RegexMatchConfig{
				Pattern: `\b(function|method|struct)\b`,
				Flags:   "i",
			},
			wantMatched:    true,
			wantConfidence: 1.0,
			wantDetails:    `\b(function|method|struct)\b`,
		},
		{
			name:   "quantifiers - one or more",
			prompt: "aaaaab",
			config: routingrule.RegexMatchConfig{
				Pattern: "a+b",
			},
			wantMatched:    true,
			wantConfidence: 1.0,
			wantDetails:    "a+b",
		},
		{
			name:   "quantifiers - optional",
			prompt: "color",
			config: routingrule.RegexMatchConfig{
				Pattern: "colou?r",
			},
			wantMatched:    true,
			wantConfidence: 1.0,
			wantDetails:    "colou?r",
		},

		// --- Unanchored matching ---
		{
			name:   "unanchored - pattern matches in middle of prompt",
			prompt: "i want to learn golang for web development",
			config: routingrule.RegexMatchConfig{
				Pattern: "golang",
			},
			wantMatched:    true,
			wantConfidence: 1.0,
			wantDetails:    "golang",
		},
		{
			name:   "unanchored - pattern at start",
			prompt: "golang is fast",
			config: routingrule.RegexMatchConfig{
				Pattern: "golang",
			},
			wantMatched:    true,
			wantConfidence: 1.0,
			wantDetails:    "golang",
		},
		{
			name:   "unanchored - pattern at end",
			prompt: "i like golang",
			config: routingrule.RegexMatchConfig{
				Pattern: "golang",
			},
			wantMatched:    true,
			wantConfidence: 1.0,
			wantDetails:    "golang",
		},

		// --- Design doc example rule ---
		{
			name:   "design doc example - go function regex",
			prompt: "write a golang function to handle http requests",
			config: routingrule.RegexMatchConfig{
				Pattern: `\b(go|golang)\b.*\b(function|method|struct)\b`,
				Flags:   "i",
			},
			wantMatched:    true,
			wantConfidence: 1.0,
			wantDetails:    `\b(go|golang)\b.*\b(function|method|struct)\b`,
		},
		{
			name:   "design doc example - no match for unrelated prompt",
			prompt: "write a python script to parse csv",
			config: routingrule.RegexMatchConfig{
				Pattern: `\b(go|golang)\b.*\b(function|method|struct)\b`,
				Flags:   "i",
			},
			wantMatched: false,
		},

		// --- Case-sensitive matching (no flags) preserves prompt case ---
		{
			name:   "case-sensitive match preserves case",
			prompt: "write Go code",
			config: routingrule.RegexMatchConfig{
				Pattern: `\bGo\b`,
				Flags:   "",
			},
			wantMatched:    true,
			wantConfidence: 1.0,
			wantDetails:    `\bGo\b`,
		},
		{
			name:   "case-sensitive no match when case differs",
			prompt: "write go code",
			config: routingrule.RegexMatchConfig{
				Pattern: `\bGo\b`,
				Flags:   "",
			},
			wantMatched: false,
		},

		// --- Unsupported regex flags ---
		{
			name:   "unsupported flag returns error",
			prompt: "any prompt",
			config: routingrule.RegexMatchConfig{
				Pattern: "go",
				Flags:   "x",
			},
			wantErr:         true,
			wantErrContains: "unsupported regex flag",
		},
		{
			name:   "multiple flags with one unsupported returns error",
			prompt: "any prompt",
			config: routingrule.RegexMatchConfig{
				Pattern: "go",
				Flags:   "ix",
			},
			wantErr:         true,
			wantErrContains: "unsupported regex flag",
		},
		{
			name:   "unsupported flag before valid flag returns error",
			prompt: "any prompt",
			config: routingrule.RegexMatchConfig{
				Pattern: "go",
				Flags:   "xi",
			},
			wantErr:         true,
			wantErrContains: "unsupported regex flag",
		},
		{
			name:   "empty flags is valid",
			prompt: "write go code",
			config: routingrule.RegexMatchConfig{
				Pattern: "go",
				Flags:   "",
			},
			wantMatched:    true,
			wantConfidence: 1.0,
			wantDetails:    "go",
		},

		// --- Word boundary patterns ---
		{
			name:   "word boundary prevents partial match",
			prompt: "i love eating mango",
			config: routingrule.RegexMatchConfig{
				Pattern: `\bgo\b`,
			},
			wantMatched: false,
		},
		{
			name:   "word boundary matches whole word",
			prompt: "write go code",
			config: routingrule.RegexMatchConfig{
				Pattern: `\bgo\b`,
			},
			wantMatched:    true,
			wantConfidence: 1.0,
			wantDetails:    `\bgo\b`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			matcher := routing.NewRegexMatcher()
			defer matcher.Close()

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
				assert.Equal(t, tt.wantDetails, result.Details)
			}
		})
	}
}

func TestRegexMatcher_Match_ContextCancellation(t *testing.T) {
	t.Parallel()

	matcher := routing.NewRegexMatcher()
	defer matcher.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	config := routingrule.RegexMatchConfig{
		Pattern: "go",
	}

	result, err := matcher.Match(ctx, "write go code", config)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.False(t, result.Matched)
}

func TestRegexMatcher_Match_ContextDeadlineExceeded(t *testing.T) {
	t.Parallel()

	matcher := routing.NewRegexMatcher()
	defer matcher.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()

	config := routingrule.RegexMatchConfig{
		Pattern: "go",
	}

	result, err := matcher.Match(ctx, "write go code", config)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.False(t, result.Matched)
}

func TestRegexMatcher_PatternCaching(t *testing.T) {
	t.Parallel()

	matcher := routing.NewRegexMatcher()
	defer matcher.Close()

	config := routingrule.RegexMatchConfig{
		Pattern: `\bgo\b`,
		Flags:   "i",
	}

	// First call compiles and caches the pattern.
	result1, err := matcher.Match(context.Background(), "write Go code", config)
	require.NoError(t, err)
	assert.True(t, result1.Matched)

	// Second call should reuse the cached pattern and produce the same result.
	result2, err := matcher.Match(context.Background(), "Go function implementation", config)
	require.NoError(t, err)
	assert.True(t, result2.Matched)

	// Verify only one cache entry exists (same pattern+flags = same key).
	assert.Equal(t, 1, routing.ExportRegexCacheLen(matcher))
}

func TestRegexMatcher_CacheSeparatesByFlags(t *testing.T) {
	t.Parallel()

	matcher := routing.NewRegexMatcher()
	defer matcher.Close()

	configNoFlag := routingrule.RegexMatchConfig{
		Pattern: "go",
	}
	configWithFlag := routingrule.RegexMatchConfig{
		Pattern: "go",
		Flags:   "i",
	}

	_, err := matcher.Match(context.Background(), "write go code", configNoFlag)
	require.NoError(t, err)

	_, err = matcher.Match(context.Background(), "write GO code", configWithFlag)
	require.NoError(t, err)

	// Two distinct cache entries: ":go" and "i:go".
	assert.Equal(t, 2, routing.ExportRegexCacheLen(matcher))
}

func TestRegexMatcher_RegistersInRegistry(t *testing.T) {
	t.Parallel()

	registry := routing.NewMatcherRegistry()
	matcher := routing.NewRegexMatcher()
	defer matcher.Close()

	registry.Register(matcher)

	got := registry.GetMatcher(routing.MatchTypeRegex)
	require.NotNil(t, got)
	assert.Equal(t, routing.MatchTypeRegex, got.Type())
}

func TestRegexMatcher_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	matcher := routing.NewRegexMatcher()
	defer matcher.Close()

	configs := []routingrule.RegexMatchConfig{
		{Pattern: `\bgo\b`, Flags: "i"},
		{Pattern: `\brust\b`, Flags: "i"},
		{Pattern: `python`, Flags: ""},
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
			config := configs[i%len(configs)]
			_, err := matcher.Match(context.Background(), prompt, config)
			assert.NoError(t, err)
		})
	}
}

func TestRegexMatcher_Close_Idempotent(t *testing.T) {
	t.Parallel()

	matcher := routing.NewRegexMatcher()

	// Calling Close multiple times must not panic.
	matcher.Close()
	matcher.Close()
	matcher.Close()
}

func TestRegexMatcher_CleanExpiredPatterns_EvictsStaleEntries(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	fakeClock := func() time.Time { return now }

	ttl := 10 * time.Minute
	matcher := routing.ExportNewRegexMatcherForTest(ttl, fakeClock)
	defer matcher.Close()

	config := routingrule.RegexMatchConfig{
		Pattern: `\bgo\b`,
		Flags:   "i",
	}

	// Populate the cache.
	_, err := matcher.Match(context.Background(), "write go code", config)
	require.NoError(t, err)

	assert.Equal(t, 1, routing.ExportRegexCacheLen(matcher), "expected 1 cached pattern after match")

	// Advance time past the TTL.
	now = now.Add(11 * time.Minute)

	// Run cleanup explicitly (no background goroutine in test matcher).
	routing.ExportCleanExpiredRegexPatterns(matcher)

	assert.Equal(t, 0, routing.ExportRegexCacheLen(matcher), "expected 0 cached patterns after cleanup")
}

func TestRegexMatcher_CleanExpiredPatterns_PreservesRecentEntries(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	fakeClock := func() time.Time { return now }

	ttl := 10 * time.Minute
	matcher := routing.ExportNewRegexMatcherForTest(ttl, fakeClock)
	defer matcher.Close()

	config := routingrule.RegexMatchConfig{
		Pattern: `\bgo\b`,
		Flags:   "i",
	}

	// Populate the cache.
	_, err := matcher.Match(context.Background(), "write go code", config)
	require.NoError(t, err)

	// Advance time to within TTL.
	now = now.Add(5 * time.Minute)

	routing.ExportCleanExpiredRegexPatterns(matcher)

	assert.Equal(t, 1, routing.ExportRegexCacheLen(matcher), "entries within TTL should be preserved")
}

func TestRegexMatcher_SlidingTTL_RefreshesOnAccess(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	fakeClock := func() time.Time { return now }

	ttl := 10 * time.Minute
	matcher := routing.ExportNewRegexMatcherForTest(ttl, fakeClock)
	defer matcher.Close()

	config := routingrule.RegexMatchConfig{
		Pattern: `\bgo\b`,
		Flags:   "i",
	}
	cacheKey := "i:" + `\bgo\b`

	// Populate: cached at T+0.
	_, err := matcher.Match(context.Background(), "write go code", config)
	require.NoError(t, err)

	initialLastUsed := routing.ExportRegexCacheLastUsed(matcher, cacheKey)
	assert.Equal(t, now, initialLastUsed)

	// Advance 8 minutes (within TTL).
	now = now.Add(8 * time.Minute)

	// Access again -- this should refresh lastUsed to T+8m.
	_, err = matcher.Match(context.Background(), "write go code", config)
	require.NoError(t, err)

	refreshedLastUsed := routing.ExportRegexCacheLastUsed(matcher, cacheKey)
	assert.Equal(t, now, refreshedLastUsed, "lastUsed should be refreshed on cache hit")

	// Advance another 8 minutes (T+16m total, but only 8m since last access).
	now = now.Add(8 * time.Minute)

	routing.ExportCleanExpiredRegexPatterns(matcher)

	// Pattern was accessed 8 minutes ago, TTL is 10 minutes -- should still be cached.
	assert.Equal(t, 1, routing.ExportRegexCacheLen(matcher),
		"pattern accessed within TTL should survive cleanup")

	// Advance 3 more minutes (T+19m total, 11m since last access).
	now = now.Add(3 * time.Minute)

	routing.ExportCleanExpiredRegexPatterns(matcher)

	// Now 11 minutes since last access -- should be evicted.
	assert.Equal(t, 0, routing.ExportRegexCacheLen(matcher),
		"pattern not accessed within TTL should be evicted")
}

func TestRegexMatcher_ConfidenceAlways1(t *testing.T) {
	t.Parallel()

	matcher := routing.NewRegexMatcher()
	defer matcher.Close()

	// Various patterns that match -- all should yield confidence 1.0.
	tests := []struct {
		name   string
		prompt string
		config routingrule.RegexMatchConfig
	}{
		{
			name:   "simple literal",
			prompt: "write go code",
			config: routingrule.RegexMatchConfig{Pattern: "go"},
		},
		{
			name:   "complex pattern",
			prompt: "golang function handler",
			config: routingrule.RegexMatchConfig{
				Pattern: `\b(go|golang)\b.*\b(function|method)\b`,
				Flags:   "i",
			},
		},
		{
			name:   "match-all pattern",
			prompt: "any prompt at all",
			config: routingrule.RegexMatchConfig{Pattern: ".*"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := matcher.Match(context.Background(), tt.prompt, tt.config)
			require.NoError(t, err)
			require.True(t, result.Matched)
			assert.InDelta(t, 1.0, result.Confidence, 0.001)
		})
	}
}

func TestRegexMatcher_DetailsPopulatedWithPattern(t *testing.T) {
	t.Parallel()

	matcher := routing.NewRegexMatcher()
	defer matcher.Close()

	pattern := `\b(go|golang)\b`
	config := routingrule.RegexMatchConfig{
		Pattern: pattern,
		Flags:   "i",
	}

	result, err := matcher.Match(context.Background(), "write go code", config)
	require.NoError(t, err)
	require.True(t, result.Matched)
	assert.Equal(t, pattern, result.Details, "Details should contain the original pattern string")
}

func TestRegexMatcher_MatchedKeywordsEmpty(t *testing.T) {
	t.Parallel()

	matcher := routing.NewRegexMatcher()
	defer matcher.Close()

	config := routingrule.RegexMatchConfig{
		Pattern: "go",
	}

	result, err := matcher.Match(context.Background(), "write go code", config)
	require.NoError(t, err)
	require.True(t, result.Matched)
	assert.Empty(t, result.MatchedKeywords, "MatchedKeywords should be empty for regex matches")
}
