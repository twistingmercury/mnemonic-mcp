package routing_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/twistingmercury/mnemonic/internal/repository/routingrule"
	"github.com/twistingmercury/mnemonic/internal/routing"
)

func BenchmarkRegexMatcher_CacheHit(b *testing.B) {
	b.ReportAllocs()

	matcher := routing.NewRegexMatcher()
	defer matcher.Close()

	ctx := context.Background()
	config := routingrule.RegexMatchConfig{
		Pattern: `\b(go|golang)\b.*\b(function|method|struct)\b`,
		Flags:   "i",
	}

	// Warm the cache with one match before timing.
	_, _ = matcher.Match(ctx, "write a golang function for http", config)
	b.ResetTimer()

	for b.Loop() {
		_, _ = matcher.Match(ctx, "write a golang function for http", config)
	}
}

func BenchmarkRegexMatcher_CacheMiss(b *testing.B) {
	b.ReportAllocs()

	matcher := routing.NewRegexMatcher()
	defer matcher.Close()

	ctx := context.Background()

	b.ResetTimer()

	for i := range b.N {
		pattern := fmt.Sprintf("keyword%d", i)
		config := routingrule.RegexMatchConfig{
			Pattern: pattern,
			Flags:   "i",
		}
		_, _ = matcher.Match(ctx, "this prompt contains keyword"+fmt.Sprintf("%d", i)+" somewhere", config)
	}
}

func BenchmarkRegexMatcher_SimplePattern_CacheHit(b *testing.B) {
	b.ReportAllocs()

	matcher := routing.NewRegexMatcher()
	defer matcher.Close()

	ctx := context.Background()
	config := routingrule.RegexMatchConfig{
		Pattern: "golang",
	}

	// Warm the cache.
	_, _ = matcher.Match(ctx, "learn golang today", config)
	b.ResetTimer()

	for b.Loop() {
		_, _ = matcher.Match(ctx, "learn golang today", config)
	}
}

func BenchmarkRegexMatcher_ComplexPattern_CacheHit(b *testing.B) {
	b.ReportAllocs()

	matcher := routing.NewRegexMatcher()
	defer matcher.Close()

	ctx := context.Background()
	config := routingrule.RegexMatchConfig{
		Pattern: `\b(go|golang|rust|python)\b.*\b(function|method|struct|class|interface)\b.*\b(implement|create|write|build)\b`,
		Flags:   "i",
	}

	prompt := "please implement a golang function to handle the build process"

	// Warm the cache.
	_, _ = matcher.Match(ctx, prompt, config)
	b.ResetTimer()

	for b.Loop() {
		_, _ = matcher.Match(ctx, prompt, config)
	}
}

func BenchmarkRegexMatcher_MultiplePatterns(b *testing.B) {
	b.ReportAllocs()

	matcher := routing.NewRegexMatcher()
	defer matcher.Close()

	ctx := context.Background()

	configs := []routingrule.RegexMatchConfig{
		{Pattern: `\bgo\b`, Flags: "i"},
		{Pattern: `\brust\b`, Flags: "i"},
		{Pattern: `\bpython\b`, Flags: "i"},
		{Pattern: `\b(function|method)\b`, Flags: "i"},
		{Pattern: `\b(docker|kubernetes)\b`, Flags: "i"},
	}

	prompt := "write a go function for kubernetes deployment"

	// Warm the cache with all patterns.
	for _, cfg := range configs {
		_, _ = matcher.Match(ctx, prompt, cfg)
	}
	b.ResetTimer()

	for i := range b.N {
		cfg := configs[i%len(configs)]
		_, _ = matcher.Match(ctx, prompt, cfg)
	}
}

func BenchmarkRegexMatcher_NoMatch_CacheHit(b *testing.B) {
	b.ReportAllocs()

	matcher := routing.NewRegexMatcher()
	defer matcher.Close()

	ctx := context.Background()
	config := routingrule.RegexMatchConfig{
		Pattern: `\b(ruby|elixir|haskell)\b`,
		Flags:   "i",
	}

	prompt := "write a golang function for http"

	// Warm the cache.
	_, _ = matcher.Match(ctx, prompt, config)
	b.ResetTimer()

	for b.Loop() {
		_, _ = matcher.Match(ctx, prompt, config)
	}
}

func BenchmarkRegexMatcher_CaseInsensitiveVsSensitive(b *testing.B) {
	matcher := routing.NewRegexMatcher()
	defer matcher.Close()

	ctx := context.Background()
	prompt := "Write a Golang Function for HTTP handling"

	b.Run("case-insensitive", func(b *testing.B) {
		b.ReportAllocs()
		config := routingrule.RegexMatchConfig{
			Pattern: `golang.*function`,
			Flags:   "i",
		}
		_, _ = matcher.Match(ctx, prompt, config)
		b.ResetTimer()

		for b.Loop() {
			_, _ = matcher.Match(ctx, prompt, config)
		}
	})

	b.Run("case-sensitive", func(b *testing.B) {
		b.ReportAllocs()
		config := routingrule.RegexMatchConfig{
			Pattern: `Golang.*Function`,
		}
		_, _ = matcher.Match(ctx, prompt, config)
		b.ResetTimer()

		for b.Loop() {
			_, _ = matcher.Match(ctx, prompt, config)
		}
	})
}
