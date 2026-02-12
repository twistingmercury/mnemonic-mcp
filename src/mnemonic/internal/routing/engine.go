package routing

import (
	"context"
	"fmt"
	"strings"

	"github.com/rs/zerolog"
	"github.com/twistingmercury/mnemonic/internal/metrics"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "mnemonic/routing"

// Engine implements the Evaluator interface. It evaluates routing rules in priority
// order using registered matchers and returns the first matching decision, or a
// Decision with Matched: false if no rules match.
type Engine struct {
	cache    *RuleCache
	registry *MatcherRegistry
	metrics  *metrics.Routing
	logger   zerolog.Logger
	tracer   trace.Tracer
}

// NewEngine creates a new routing Engine.
// The metrics parameter may be nil if metric recording is not needed.
func NewEngine(
	cache *RuleCache,
	registry *MatcherRegistry,
	routingMetrics *metrics.Routing,
	logger zerolog.Logger,
) *Engine {
	return &Engine{
		cache:    cache,
		registry: registry,
		metrics:  routingMetrics,
		logger:   logger,
		tracer:   otel.Tracer(tracerName),
	}
}

// Route evaluates the prompt against all enabled routing rules in priority order.
// It returns the first match or Decision{Matched: false} if no rules match.
func (e *Engine) Route(ctx context.Context, req Request) (Decision, error) {
	ctx, span := e.tracer.Start(ctx, "Engine.Route",
		trace.WithAttributes(
			attribute.Int("routing.rule_count", e.cache.RuleCount()),
		),
	)
	defer span.End()

	normalized := NormalizePrompt(req.Prompt)

	rules := e.cache.GetRules()

	for _, rule := range rules {
		// Check for context cancellation before evaluating each rule to avoid
		// wasting resources on a request that has already been cancelled.
		if err := ctx.Err(); err != nil {
			return Decision{}, err
		}

		// Skip disabled rules (defensive; cache should only contain enabled rules).
		if !rule.Enabled {
			continue
		}

		matchType := MatchType(rule.MatchType)

		matcher := e.registry.GetMatcher(matchType)
		if matcher == nil {
			e.logger.Warn().
				Str("rule_name", rule.Name).
				Str("match_type", rule.MatchType).
				Msg("no matcher registered for match type, skipping rule")
			continue
		}

		result, err := matcher.Match(ctx, normalized, rule.MatchConfig)
		if err != nil {
			e.logger.Warn().Err(err).
				Str("rule_name", rule.Name).
				Str("match_type", rule.MatchType).
				Msg("matcher returned error, skipping rule")
			continue
		}

		if result.Matched {
			decision := Decision{
				Matched:         true,
				AgentName:       rule.AgentName,
				Confidence:      NormalizeConfidence(result.Confidence),
				MatchType:       matchType,
				MatchedKeywords: result.MatchedKeywords,
				Reasoning:       buildReasoning(matchType, result),
			}

			span.SetAttributes(
				attribute.Bool("routing.matched", true),
				attribute.String("routing.agent", decision.AgentName),
				attribute.String("routing.match_type", string(decision.MatchType)),
				attribute.Float64("routing.confidence", decision.Confidence),
			)

			e.recordMetrics(ctx, decision)

			e.logger.Debug().
				Str("agent", decision.AgentName).
				Str("match_type", string(decision.MatchType)).
				Float64("confidence", decision.Confidence).
				Str("rule_name", rule.Name).
				Msg("routing decision made")

			return decision, nil
		}
	}

	// No rules matched.
	span.SetAttributes(
		attribute.Bool("routing.matched", false),
	)

	e.recordNoMatch(ctx)

	e.logger.Debug().
		Msg("no rules matched")

	return Decision{}, nil
}

// recordMetrics records routing metrics if a metrics recorder is available.
func (e *Engine) recordMetrics(ctx context.Context, decision Decision) {
	if e.metrics == nil {
		return
	}
	e.metrics.RecordRoutingDecision(ctx, decision.AgentName)
	e.metrics.RecordRuleMatch(ctx, string(decision.MatchType))
}

// recordNoMatch records a no-match metric if a metrics recorder is available.
func (e *Engine) recordNoMatch(ctx context.Context) {
	if e.metrics == nil {
		return
	}
	e.metrics.RecordRuleMatch(ctx, "no_match")
}

// buildReasoning generates a human-readable explanation for a routing decision.
func buildReasoning(matchType MatchType, result MatchResult) string {
	switch matchType {
	case MatchTypeKeyword:
		if len(result.MatchedKeywords) > 0 {
			return fmt.Sprintf("Matched keywords: %s", joinKeywords(result.MatchedKeywords))
		}
		return "Matched keyword rule"
	case MatchTypeRegex:
		if result.Details != "" {
			return fmt.Sprintf("Matched regex pattern: %s", result.Details)
		}
		return "Matched regex rule"
	case MatchTypePattern:
		return fmt.Sprintf("Semantic match with confidence %.0f%%", result.Confidence*100)
	default:
		return fmt.Sprintf("Matched rule (type: %s)", matchType)
	}
}

// joinKeywords joins keywords with ", " for human-readable output.
func joinKeywords(keywords []string) string {
	return strings.Join(keywords, ", ")
}
