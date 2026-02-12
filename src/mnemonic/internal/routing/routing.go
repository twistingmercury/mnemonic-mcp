package routing

import (
	"context"
	"strings"

	"github.com/twistingmercury/mnemonic/internal/repository/routingrule"
)

// MatchType is an alias for routingrule.MatchType, the authoritative definition.
// This alias allows the routing package to use MatchType without a package qualifier
// while maintaining a single source of truth in the routingrule package.
type MatchType = routingrule.MatchType

// Match type constants re-exported from the routingrule package for convenience.
const (
	MatchTypeKeyword = routingrule.MatchTypeKeyword
	MatchTypeRegex   = routingrule.MatchTypeRegex
	MatchTypePattern = routingrule.MatchTypePattern
)

// MatchResult contains the outcome of a single rule matcher evaluation.
type MatchResult struct {
	// Matched indicates whether the rule matched the prompt.
	Matched bool

	// Confidence is a score from 0.0 to 1.0 indicating match strength.
	Confidence float64

	// MatchedKeywords contains keywords that triggered a keyword match.
	// Empty for other match types.
	MatchedKeywords []string

	// Details contains additional match information for logging.
	Details string
}

// Request encapsulates the input for a routing decision.
type Request struct {
	// Prompt is the user's input text to be routed.
	Prompt string

	// Context provides additional routing context.
	Context RequestContext

	// Options allows per-request overrides of routing behavior.
	Options Options
}

// RequestContext provides contextual information for routing decisions.
type RequestContext struct {
	// WorkingDirectory is the current working directory of the user.
	WorkingDirectory string

	// FileTypes contains the file types present in the working directory.
	FileTypes []string

	// RecentAgents contains the names of recently used agents.
	RecentAgents []string
}

// Options allows per-request overrides of default routing behavior.
type Options struct {
	// IncludePatterns enables pattern matching for this request.
	IncludePatterns bool

	// MaxPatterns limits the number of patterns to consider.
	MaxPatterns int

	// PatternRelevanceThreshold sets the minimum similarity score for pattern matches.
	PatternRelevanceThreshold float64
}

// Decision is the result of routing evaluation.
// It identifies the selected agent and the reasoning behind the decision.
// When Matched is false, all other fields are zero-valued and should not be used.
type Decision struct {
	// Matched indicates whether a routing rule matched the prompt.
	// When false, all other fields are zero-valued and should not be used.
	Matched bool

	// AgentName is the identifier of the selected agent.
	AgentName string

	// Confidence is the routing confidence from 0.0 to 1.0.
	Confidence float64

	// MatchType indicates which type of matching triggered the route.
	MatchType MatchType

	// MatchedKeywords contains keywords that triggered the route.
	// Only populated for MatchTypeKeyword.
	MatchedKeywords []string

	// Reasoning is a human-readable explanation of why this agent was selected.
	Reasoning string
}

// Evaluator defines the primary routing contract.
// It evaluates the prompt against all enabled routing rules in priority order
// and returns the first match, or a Decision with Matched: false if no rules match.
type Evaluator interface {
	// Route evaluates the prompt against routing rules and returns a decision.
	Route(ctx context.Context, req Request) (Decision, error)
}

// NormalizePrompt normalizes a prompt for matching by trimming whitespace.
// Case folding is NOT performed here; individual matchers are responsible for
// applying their own case-sensitivity rules (e.g., the keyword matcher always
// lowercases, while the regex matcher honours the "i" flag).
func NormalizePrompt(prompt string) string {
	return strings.TrimSpace(prompt)
}

// NormalizeConfidence clamps a confidence score to the range [0.0, 1.0].
func NormalizeConfidence(score float64) float64 {
	if score < 0.0 {
		return 0.0
	}
	if score > 1.0 {
		return 1.0
	}
	return score
}
