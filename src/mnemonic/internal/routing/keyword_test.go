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

func TestKeywordMatcher_Type(t *testing.T) {
	t.Parallel()

	matcher := routing.NewKeywordMatcher()
	defer matcher.Close()

	assert.Equal(t, routing.MatchTypeKeyword, matcher.Type())
}

func TestKeywordMatcher_Match(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		prompt          string
		config          routingrule.MatchConfig
		wantMatched     bool
		wantConfidence  float64
		wantKeywords    []string
		wantErr         bool
		wantErrContains string
	}{
		// --- Single keyword matching ---
		{
			name:   "single keyword match - exact word boundary",
			prompt: "write go code",
			config: routingrule.KeywordMatchConfig{
				Keywords:  []string{"go"},
				MatchMode: routingrule.MatchModeAny,
			},
			wantMatched:    true,
			wantConfidence: 1.0,
			wantKeywords:   []string{"go"},
		},
		{
			name:   "single keyword no match",
			prompt: "write python code",
			config: routingrule.KeywordMatchConfig{
				Keywords:  []string{"go"},
				MatchMode: routingrule.MatchModeAny,
			},
			wantMatched: false,
		},
		{
			name:   "single keyword at start of prompt",
			prompt: "go is a great language",
			config: routingrule.KeywordMatchConfig{
				Keywords:  []string{"go"},
				MatchMode: routingrule.MatchModeAny,
			},
			wantMatched:    true,
			wantConfidence: 1.0,
			wantKeywords:   []string{"go"},
		},
		{
			name:   "single keyword at end of prompt",
			prompt: "i want to learn go",
			config: routingrule.KeywordMatchConfig{
				Keywords:  []string{"go"},
				MatchMode: routingrule.MatchModeAny,
			},
			wantMatched:    true,
			wantConfidence: 1.0,
			wantKeywords:   []string{"go"},
		},

		// --- Multi-word phrase matching ---
		{
			name:   "multi-word phrase match - substring",
			prompt: "help me write go code for a web server",
			config: routingrule.KeywordMatchConfig{
				Keywords:  []string{"go code"},
				MatchMode: routingrule.MatchModeAny,
			},
			wantMatched:    true,
			wantConfidence: 1.0,
			wantKeywords:   []string{"go code"},
		},
		{
			name:   "multi-word phrase no match",
			prompt: "help me write python code for a web server",
			config: routingrule.KeywordMatchConfig{
				Keywords:  []string{"go code"},
				MatchMode: routingrule.MatchModeAny,
			},
			wantMatched: false,
		},

		// --- Mode "any" (OR) ---
		{
			name:   "mode any - one of several keywords matches",
			prompt: "build a golang service",
			config: routingrule.KeywordMatchConfig{
				Keywords:  []string{"python", "golang", "rust"},
				MatchMode: routingrule.MatchModeAny,
			},
			wantMatched:    true,
			wantConfidence: 1.0,
			wantKeywords:   []string{"golang"},
		},
		{
			name:   "mode any - none match",
			prompt: "build a java service",
			config: routingrule.KeywordMatchConfig{
				Keywords:  []string{"python", "golang", "rust"},
				MatchMode: routingrule.MatchModeAny,
			},
			wantMatched: false,
		},
		{
			name:   "mode any - multiple keywords match",
			prompt: "compare go and rust performance",
			config: routingrule.KeywordMatchConfig{
				Keywords:  []string{"go", "rust", "python"},
				MatchMode: routingrule.MatchModeAny,
			},
			wantMatched:    true,
			wantConfidence: 1.0,
			wantKeywords:   []string{"go", "rust"},
		},

		// --- Mode "all" (AND) ---
		{
			name:   "mode all - all keywords match",
			prompt: "write go code with error handling",
			config: routingrule.KeywordMatchConfig{
				Keywords:  []string{"go", "error handling"},
				MatchMode: routingrule.MatchModeAll,
			},
			wantMatched:    true,
			wantConfidence: 1.0,
			wantKeywords:   []string{"go", "error handling"},
		},
		{
			name:   "mode all - partial match some but not all",
			prompt: "write go code for a service",
			config: routingrule.KeywordMatchConfig{
				Keywords:  []string{"go", "error handling"},
				MatchMode: routingrule.MatchModeAll,
			},
			wantMatched: false,
		},
		{
			name:   "mode all - none match",
			prompt: "write python code",
			config: routingrule.KeywordMatchConfig{
				Keywords:  []string{"go", "rust"},
				MatchMode: routingrule.MatchModeAll,
			},
			wantMatched: false,
		},

		// --- Case insensitivity ---
		{
			name:   "case insensitivity - uppercase keyword matches lowercase prompt",
			prompt: "write go code",
			config: routingrule.KeywordMatchConfig{
				Keywords:  []string{"Go"},
				MatchMode: routingrule.MatchModeAny,
			},
			wantMatched:    true,
			wantConfidence: 1.0,
			wantKeywords:   []string{"Go"},
		},
		{
			name:   "case insensitivity - mixed case keyword",
			prompt: "write golang code",
			config: routingrule.KeywordMatchConfig{
				Keywords:  []string{"GoLang"},
				MatchMode: routingrule.MatchModeAny,
			},
			wantMatched:    true,
			wantConfidence: 1.0,
			wantKeywords:   []string{"GoLang"},
		},
		{
			name:   "case insensitivity - multi-word phrase",
			prompt: "help with go code review",
			config: routingrule.KeywordMatchConfig{
				Keywords:  []string{"Go Code"},
				MatchMode: routingrule.MatchModeAny,
			},
			wantMatched:    true,
			wantConfidence: 1.0,
			wantKeywords:   []string{"Go Code"},
		},

		// --- Word boundary awareness ---
		{
			name:   "word boundary - go should NOT match mango",
			prompt: "i love eating mango",
			config: routingrule.KeywordMatchConfig{
				Keywords:  []string{"go"},
				MatchMode: routingrule.MatchModeAny,
			},
			wantMatched: false,
		},
		{
			name:   "word boundary - go should NOT match going",
			prompt: "i am going to the store",
			config: routingrule.KeywordMatchConfig{
				Keywords:  []string{"go"},
				MatchMode: routingrule.MatchModeAny,
			},
			wantMatched: false,
		},
		{
			name:   "word boundary - go should match write go code",
			prompt: "write go code",
			config: routingrule.KeywordMatchConfig{
				Keywords:  []string{"go"},
				MatchMode: routingrule.MatchModeAny,
			},
			wantMatched:    true,
			wantConfidence: 1.0,
			wantKeywords:   []string{"go"},
		},
		{
			name:   "word boundary - go should match go function",
			prompt: "go function implementation",
			config: routingrule.KeywordMatchConfig{
				Keywords:  []string{"go"},
				MatchMode: routingrule.MatchModeAny,
			},
			wantMatched:    true,
			wantConfidence: 1.0,
			wantKeywords:   []string{"go"},
		},
		{
			name:   "word boundary - go should NOT match ergo",
			prompt: "ergo it is correct",
			config: routingrule.KeywordMatchConfig{
				Keywords:  []string{"go"},
				MatchMode: routingrule.MatchModeAny,
			},
			wantMatched: false,
		},

		// --- Empty inputs ---
		{
			name:   "empty keywords list - no match",
			prompt: "write go code",
			config: routingrule.KeywordMatchConfig{
				Keywords:  []string{},
				MatchMode: routingrule.MatchModeAny,
			},
			wantMatched: false,
		},
		{
			name:   "nil keywords list - no match",
			prompt: "write go code",
			config: routingrule.KeywordMatchConfig{
				Keywords:  nil,
				MatchMode: routingrule.MatchModeAny,
			},
			wantMatched: false,
		},
		{
			name:   "empty prompt - no match",
			prompt: "",
			config: routingrule.KeywordMatchConfig{
				Keywords:  []string{"go"},
				MatchMode: routingrule.MatchModeAny,
			},
			wantMatched: false,
		},

		// --- Invalid config type ---
		{
			name:   "invalid config type - not KeywordMatchConfig",
			prompt: "write go code",
			config: routingrule.RegexMatchConfig{
				Pattern: "go",
			},
			wantErr:         true,
			wantErrContains: "expected KeywordMatchConfig",
		},

		// --- Special regex characters in keywords ---
		{
			name:   "keyword with special regex characters - dot",
			prompt: "use c++ for performance",
			config: routingrule.KeywordMatchConfig{
				Keywords:  []string{"c++"},
				MatchMode: routingrule.MatchModeAny,
			},
			wantMatched:    true,
			wantConfidence: 1.0,
			wantKeywords:   []string{"c++"},
		},
		{
			name:   "keyword with special regex characters - parentheses",
			prompt: "learn about func() in go",
			config: routingrule.KeywordMatchConfig{
				Keywords:  []string{"func()"},
				MatchMode: routingrule.MatchModeAny,
			},
			wantMatched:    true,
			wantConfidence: 1.0,
			wantKeywords:   []string{"func()"},
		},
		{
			name:   "keyword with special regex characters - brackets",
			prompt: "use map[string] for lookups",
			config: routingrule.KeywordMatchConfig{
				Keywords:  []string{"map[string]"},
				MatchMode: routingrule.MatchModeAny,
			},
			wantMatched:    true,
			wantConfidence: 1.0,
			wantKeywords:   []string{"map[string]"},
		},
		{
			name:   "keyword with dot should not match arbitrary character",
			prompt: "use cab for transportation",
			config: routingrule.KeywordMatchConfig{
				Keywords:  []string{"c.b"},
				MatchMode: routingrule.MatchModeAny,
			},
			wantMatched: false,
		},

		// --- Default match mode ---
		{
			name:   "empty match mode defaults to any behavior",
			prompt: "write go code",
			config: routingrule.KeywordMatchConfig{
				Keywords:  []string{"go"},
				MatchMode: routingrule.MatchMode(""),
			},
			wantMatched:    true,
			wantConfidence: 1.0,
			wantKeywords:   []string{"go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			matcher := routing.NewKeywordMatcher()
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
				assert.Equal(t, tt.wantKeywords, result.MatchedKeywords)
			}
		})
	}
}

func TestKeywordMatcher_Match_ContextCancellation(t *testing.T) {
	t.Parallel()

	matcher := routing.NewKeywordMatcher()
	defer matcher.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	config := routingrule.KeywordMatchConfig{
		Keywords:  []string{"go", "rust", "python"},
		MatchMode: routingrule.MatchModeAny,
	}

	result, err := matcher.Match(ctx, "write go code", config)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.False(t, result.Matched)
}

func TestKeywordMatcher_Match_ContextDeadlineExceeded(t *testing.T) {
	t.Parallel()

	matcher := routing.NewKeywordMatcher()
	defer matcher.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()

	config := routingrule.KeywordMatchConfig{
		Keywords:  []string{"go"},
		MatchMode: routingrule.MatchModeAny,
	}

	result, err := matcher.Match(ctx, "write go code", config)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.False(t, result.Matched)
}

func TestKeywordMatcher_PatternCaching(t *testing.T) {
	t.Parallel()

	// Verify that the matcher caches compiled patterns by calling Match twice
	// with the same keyword. The second call should reuse the cached pattern.
	matcher := routing.NewKeywordMatcher()
	defer matcher.Close()

	config := routingrule.KeywordMatchConfig{
		Keywords:  []string{"go"},
		MatchMode: routingrule.MatchModeAny,
	}

	result1, err := matcher.Match(context.Background(), "write go code", config)
	require.NoError(t, err)
	assert.True(t, result1.Matched)

	result2, err := matcher.Match(context.Background(), "go function", config)
	require.NoError(t, err)
	assert.True(t, result2.Matched)
}

func TestKeywordMatcher_RegistersInRegistry(t *testing.T) {
	t.Parallel()

	registry := routing.NewMatcherRegistry()
	matcher := routing.NewKeywordMatcher()
	defer matcher.Close()

	registry.Register(matcher)

	got := registry.GetMatcher(routing.MatchTypeKeyword)
	require.NotNil(t, got)
	assert.Equal(t, routing.MatchTypeKeyword, got.Type())
}

func TestKeywordMatcher_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	matcher := routing.NewKeywordMatcher()
	defer matcher.Close()

	config := routingrule.KeywordMatchConfig{
		Keywords:  []string{"go", "rust", "python"},
		MatchMode: routingrule.MatchModeAny,
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

func TestKeywordMatcher_Close_Idempotent(t *testing.T) {
	t.Parallel()

	matcher := routing.NewKeywordMatcher()

	// Calling Close multiple times must not panic.
	matcher.Close()
	matcher.Close()
	matcher.Close()
}

func TestKeywordMatcher_CleanExpiredPatterns_EvictsStaleEntries(t *testing.T) {
	t.Parallel()

	// Use a fake clock so we can control time without sleeping.
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	fakeClock := func() time.Time { return now }

	ttl := 10 * time.Minute
	matcher := routing.ExportNewKeywordMatcherForTest(ttl, fakeClock)
	defer matcher.Close()

	config := routingrule.KeywordMatchConfig{
		Keywords:  []string{"go", "rust"},
		MatchMode: routingrule.MatchModeAny,
	}

	// Populate the cache with two patterns.
	_, err := matcher.Match(context.Background(), "write go and rust code", config)
	require.NoError(t, err)

	assert.Equal(t, 2, routing.ExportPatternCacheLen(matcher), "expected 2 cached patterns after match")

	// Advance time past the TTL.
	now = now.Add(11 * time.Minute)

	// Run cleanup explicitly (no background goroutine in test matcher).
	routing.ExportCleanExpiredPatterns(matcher)

	assert.Equal(t, 0, routing.ExportPatternCacheLen(matcher), "expected 0 cached patterns after cleanup")
}

func TestKeywordMatcher_CleanExpiredPatterns_PreservesRecentEntries(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	fakeClock := func() time.Time { return now }

	ttl := 10 * time.Minute
	matcher := routing.ExportNewKeywordMatcherForTest(ttl, fakeClock)
	defer matcher.Close()

	config := routingrule.KeywordMatchConfig{
		Keywords:  []string{"go", "rust"},
		MatchMode: routingrule.MatchModeAny,
	}

	// Populate the cache.
	_, err := matcher.Match(context.Background(), "write go and rust code", config)
	require.NoError(t, err)

	// Advance time to within TTL.
	now = now.Add(5 * time.Minute)

	routing.ExportCleanExpiredPatterns(matcher)

	assert.Equal(t, 2, routing.ExportPatternCacheLen(matcher), "entries within TTL should be preserved")
}

func TestKeywordMatcher_SlidingTTL_RefreshesOnAccess(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	fakeClock := func() time.Time { return now }

	ttl := 10 * time.Minute
	matcher := routing.ExportNewKeywordMatcherForTest(ttl, fakeClock)
	defer matcher.Close()

	config := routingrule.KeywordMatchConfig{
		Keywords:  []string{"go"},
		MatchMode: routingrule.MatchModeAny,
	}

	// Populate: "go" cached at T+0.
	_, err := matcher.Match(context.Background(), "write go code", config)
	require.NoError(t, err)

	initialLastUsed := routing.ExportPatternLastUsed(matcher, "go")
	assert.Equal(t, now, initialLastUsed)

	// Advance 8 minutes (within TTL).
	now = now.Add(8 * time.Minute)

	// Access "go" again -- this should refresh lastUsed to T+8m.
	_, err = matcher.Match(context.Background(), "write go code", config)
	require.NoError(t, err)

	refreshedLastUsed := routing.ExportPatternLastUsed(matcher, "go")
	assert.Equal(t, now, refreshedLastUsed, "lastUsed should be refreshed on cache hit")

	// Advance another 8 minutes (T+16m total, but only 8m since last access).
	now = now.Add(8 * time.Minute)

	routing.ExportCleanExpiredPatterns(matcher)

	// Pattern was accessed 8 minutes ago, TTL is 10 minutes -- should still be cached.
	assert.Equal(t, 1, routing.ExportPatternCacheLen(matcher),
		"pattern accessed within TTL should survive cleanup")

	// Advance 3 more minutes (T+19m total, 11m since last access).
	now = now.Add(3 * time.Minute)

	routing.ExportCleanExpiredPatterns(matcher)

	// Now 11 minutes since last access -- should be evicted.
	assert.Equal(t, 0, routing.ExportPatternCacheLen(matcher),
		"pattern not accessed within TTL should be evicted")
}

func TestKeywordMatcher_CleanExpiredPatterns_PartialEviction(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	fakeClock := func() time.Time { return now }

	ttl := 10 * time.Minute
	matcher := routing.ExportNewKeywordMatcherForTest(ttl, fakeClock)
	defer matcher.Close()

	// Cache "go" at T+0.
	configGo := routingrule.KeywordMatchConfig{
		Keywords:  []string{"go"},
		MatchMode: routingrule.MatchModeAny,
	}
	_, err := matcher.Match(context.Background(), "write go code", configGo)
	require.NoError(t, err)

	// Advance 6 minutes, then cache "rust" at T+6m.
	now = now.Add(6 * time.Minute)

	configRust := routingrule.KeywordMatchConfig{
		Keywords:  []string{"rust"},
		MatchMode: routingrule.MatchModeAny,
	}
	_, err = matcher.Match(context.Background(), "build a rust service", configRust)
	require.NoError(t, err)

	assert.Equal(t, 2, routing.ExportPatternCacheLen(matcher))

	// Advance to T+11m: "go" is 11m old (expired), "rust" is 5m old (fresh).
	now = now.Add(5 * time.Minute)

	routing.ExportCleanExpiredPatterns(matcher)

	assert.Equal(t, 1, routing.ExportPatternCacheLen(matcher),
		"only the expired entry should be evicted")

	// Verify "rust" is the one that survived.
	rustLastUsed := routing.ExportPatternLastUsed(matcher, "rust")
	assert.False(t, rustLastUsed.IsZero(), "rust pattern should still be cached")

	goLastUsed := routing.ExportPatternLastUsed(matcher, "go")
	assert.True(t, goLastUsed.IsZero(), "go pattern should have been evicted")
}
