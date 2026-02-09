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
// default decision if no rules match.
type Engine struct {
	cache        *RuleCache
	registry     *MatcherRegistry
	defaultAgent string
	metrics      *metrics.Routing
	logger       zerolog.Logger
	tracer       trace.Tracer
}

// NewEngine creates a new routing Engine.
// The metrics parameter may be nil if metric recording is not needed.
func NewEngine(
	cache *RuleCache,
	registry *MatcherRegistry,
	defaultAgent string,
	routingMetrics *metrics.Routing,
	logger zerolog.Logger,
) *Engine {
	return &Engine{
		cache:        cache,
		registry:     registry,
		defaultAgent: defaultAgent,
		metrics:      routingMetrics,
		logger:       logger,
		tracer:       otel.Tracer(tracerName),
	}
}

// Route evaluates the prompt against all enabled routing rules in priority order.
// It returns the first match or a default decision if no rules match.
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
				AgentName:       rule.AgentName,
				Confidence:      NormalizeConfidence(result.Confidence),
				MatchType:       matchType,
				MatchedKeywords: result.MatchedKeywords,
				Reasoning:       buildReasoning(matchType, result),
			}

			span.SetAttributes(
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

	// No rules matched; return default decision.
	decision := Decision{
		AgentName:  e.defaultAgent,
		Confidence: 0.5,
		MatchType:  MatchTypeDefault,
		Reasoning:  "No specific rules matched; using default agent",
	}

	span.SetAttributes(
		attribute.String("routing.agent", decision.AgentName),
		attribute.String("routing.match_type", string(MatchTypeDefault)),
		attribute.Float64("routing.confidence", decision.Confidence),
	)

	e.recordMetrics(ctx, decision)

	e.logger.Debug().
		Str("agent", decision.AgentName).
		Msg("no rules matched, using default agent")

	return decision, nil
}

// recordMetrics records routing metrics if a metrics recorder is available.
func (e *Engine) recordMetrics(ctx context.Context, decision Decision) {
	if e.metrics == nil {
		return
	}
	e.metrics.RecordRoutingDecision(ctx, decision.AgentName)
	e.metrics.RecordRuleMatch(ctx, string(decision.MatchType))
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
	case MatchTypeDefault:
		return "No specific rules matched; using default agent"
	default:
		return fmt.Sprintf("Matched rule (type: %s)", matchType)
	}
}

// joinKeywords joins keywords with ", " for human-readable output.
func joinKeywords(keywords []string) string {
	return strings.Join(keywords, ", ")
}
