package routing_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/twistingmercury/mnemonic/internal/repository/routingrule"
	"github.com/twistingmercury/mnemonic/internal/routing"
)

func BenchmarkKeywordMatcher_SingleKeyword_CacheHit(b *testing.B) {
	b.ReportAllocs()

	matcher := routing.NewKeywordMatcher()
	defer matcher.Close()

	ctx := context.Background()
	config := routingrule.KeywordMatchConfig{
		Keywords:  []string{"go"},
		MatchMode: routingrule.MatchModeAny,
	}

	// Warm the cache with one match before timing.
	_, _ = matcher.Match(ctx, "write go code", config)
	b.ResetTimer()

	for range b.N {
		_, _ = matcher.Match(ctx, "write go code", config)
	}
}

func BenchmarkKeywordMatcher_SingleKeyword_CacheMiss(b *testing.B) {
	b.ReportAllocs()

	matcher := routing.NewKeywordMatcher()
	defer matcher.Close()

	ctx := context.Background()

	b.ResetTimer()

	for i := range b.N {
		kw := fmt.Sprintf("keyword%d", i)
		config := routingrule.KeywordMatchConfig{
			Keywords:  []string{kw},
			MatchMode: routingrule.MatchModeAny,
		}
		_, _ = matcher.Match(ctx, "this prompt contains keyword"+fmt.Sprintf("%d", i)+" somewhere", config)
	}
}

func BenchmarkKeywordMatcher_MultiWordPhrase(b *testing.B) {
	b.ReportAllocs()

	matcher := routing.NewKeywordMatcher()
	defer matcher.Close()

	ctx := context.Background()
	config := routingrule.KeywordMatchConfig{
		Keywords:  []string{"go code"},
		MatchMode: routingrule.MatchModeAny,
	}

	// Warm the cache (multi-word takes substring path, but warm for consistency).
	_, _ = matcher.Match(ctx, "help me write go code for a web server", config)
	b.ResetTimer()

	for range b.N {
		_, _ = matcher.Match(ctx, "help me write go code for a web server", config)
	}
}

func BenchmarkKeywordMatcher_ManyKeywords_AnyMode(b *testing.B) {
	b.ReportAllocs()

	matcher := routing.NewKeywordMatcher()
	defer matcher.Close()

	ctx := context.Background()

	keywords := make([]string, 100)
	for i := range 100 {
		keywords[i] = fmt.Sprintf("kw%d", i)
	}
	// The prompt matches only the last keyword, forcing a full scan.
	prompt := "the answer is kw99 in this text"

	config := routingrule.KeywordMatchConfig{
		Keywords:  keywords,
		MatchMode: routingrule.MatchModeAny,
	}

	// Warm the cache so all 100 patterns are compiled.
	_, _ = matcher.Match(ctx, prompt, config)
	b.ResetTimer()

	for range b.N {
		_, _ = matcher.Match(ctx, prompt, config)
	}
}

func BenchmarkKeywordMatcher_ManyKeywords_AllMode(b *testing.B) {
	b.ReportAllocs()

	matcher := routing.NewKeywordMatcher()
	defer matcher.Close()

	ctx := context.Background()

	keywords := make([]string, 10)
	promptParts := make([]string, 10)
	for i := range 10 {
		keywords[i] = fmt.Sprintf("term%d", i)
		promptParts[i] = fmt.Sprintf("term%d", i)
	}
	prompt := "prompt containing " + strings.Join(promptParts, " and ") + " here"

	config := routingrule.KeywordMatchConfig{
		Keywords:  keywords,
		MatchMode: routingrule.MatchModeAll,
	}

	// Warm the cache so all 10 patterns are compiled.
	_, _ = matcher.Match(ctx, prompt, config)
	b.ResetTimer()

	for range b.N {
		_, _ = matcher.Match(ctx, prompt, config)
	}
}

func BenchmarkKeywordMatcher_NonWordCharKeyword(b *testing.B) {
	b.ReportAllocs()

	matcher := routing.NewKeywordMatcher()
	defer matcher.Close()

	ctx := context.Background()
	config := routingrule.KeywordMatchConfig{
		Keywords:  []string{"c++"},
		MatchMode: routingrule.MatchModeAny,
	}

	// Warm (substring fallback path, no regex cache involved).
	_, _ = matcher.Match(ctx, "use c++ for performance", config)
	b.ResetTimer()

	for range b.N {
		_, _ = matcher.Match(ctx, "use c++ for performance", config)
	}
}
