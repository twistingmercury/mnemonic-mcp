package routing_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/twistingmercury/mnemonic/internal/routing"
)

func TestNormalizePrompt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{name: "lowercase", input: "HELLO WORLD", expect: "hello world"},
		{name: "trim whitespace", input: "  hello  ", expect: "hello"},
		{name: "both", input: "  WRITE Go CODE  ", expect: "write go code"},
		{name: "empty string", input: "", expect: ""},
		{name: "only whitespace", input: "   ", expect: ""},
		{name: "already normalized", input: "go code", expect: "go code"},
		{name: "tabs and newlines", input: "\t\nHello\t\n", expect: "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expect, routing.NormalizePrompt(tt.input))
		})
	}
}

func TestNormalizeConfidence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  float64
		expect float64
	}{
		{name: "zero", input: 0.0, expect: 0.0},
		{name: "one", input: 1.0, expect: 1.0},
		{name: "mid", input: 0.5, expect: 0.5},
		{name: "negative clamped to zero", input: -0.5, expect: 0.0},
		{name: "over one clamped to one", input: 1.5, expect: 1.0},
		{name: "very negative", input: -100.0, expect: 0.0},
		{name: "very large", input: 100.0, expect: 1.0},
		{name: "small positive", input: 0.001, expect: 0.001},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.InDelta(t, tt.expect, routing.NormalizeConfidence(tt.input), 0.0001)
		})
	}
}

func TestMatchType_Constants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, routing.MatchType("keyword"), routing.MatchTypeKeyword)
	assert.Equal(t, routing.MatchType("regex"), routing.MatchTypeRegex)
	assert.Equal(t, routing.MatchType("pattern"), routing.MatchTypePattern)
	assert.Equal(t, routing.MatchType("default"), routing.MatchTypeDefault)
}
