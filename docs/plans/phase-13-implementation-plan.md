# Phase 13 Implementation Plan: Remove Default Fallback, Implement Explicit No-Match Signaling

**Date:** 2026-02-12
**Status:** Ready for implementation
**Prerequisite:** Solutions architect assessment approved (see `docs/architecture/design-change-remove-default-matcher.md`)
**Branch:** `phase/13`

## Table of Contents

- [Overview](#overview)
- [Migration Steps](#migration-steps)
  - [Step 1: Add Matched Field to Decision Struct](#step-1-add-matched-field-to-decision-struct)
  - [Step 2: Update Engine Tests for Matched Field](#step-2-update-engine-tests-for-matched-field)
  - [Step 3: Remove defaultAgent from Engine](#step-3-remove-defaultagent-from-engine)
  - [Step 4: Remove MatchTypeDefault from routingrule Package](#step-4-remove-matchtypedefault-from-routingrule-package)
  - [Step 5: Delete DefaultMatcher Files](#step-5-delete-defaultmatcher-files)
  - [Step 6: Remove routing.default_agent from Config](#step-6-remove-routingdefault_agent-from-config)
  - [Step 7: Update OpenAPI Spec](#step-7-update-openapi-spec)
  - [Step 8: Update Design and Architecture Docs](#step-8-update-design-and-architecture-docs)
  - [Step 9: Full Test Suite and Static Analysis](#step-9-full-test-suite-and-static-analysis)
- [Delegation Plan](#delegation-plan)

---

## Overview

Phase 13 is redefined from "Implement default matcher (fallback routing)" to "Remove default fallback, implement explicit no-match signaling." The engine returns `Decision{Matched: false}, nil` when no rules match instead of silently routing to a hardcoded default agent.

**Principle:** "A default match is no match."

**Migration ordering rationale:** Steps 1-2 are additive and backward-compatible (all existing tests pass). Steps 3-6 are breaking changes that remove code. This ordering minimizes the window where tests are red.

---

## Migration Steps

### Step 1: Add Matched Field to Decision Struct

**Goal:** Add `Matched bool` to `Decision` and set it `true` on the match path. The existing default fallback still works. All existing tests still pass.

**Files modified:** 2

---

#### File: `src/mnemonic/internal/routing/routing.go`

**Change 1a: Add `Matched bool` to `Decision` struct (lines 77-93)**

Replace the current `Decision` struct with:

```go
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
```

The `Matched` field is placed first for prominence. The zero-value `Decision{}` has `Matched: false`, which is the correct no-match signal.

**Change 1b: Update `Evaluator` doc comment (lines 95-97)**

Replace:

```go
// Evaluator defines the primary routing contract.
// It evaluates the prompt against all enabled routing rules in priority order
// and returns the first match.
```

With:

```go
// Evaluator defines the primary routing contract.
// It evaluates the prompt against all enabled routing rules in priority order
// and returns the first match, or a Decision with Matched: false if no rules match.
```

---

#### File: `src/mnemonic/internal/routing/engine.go`

**Change 1c: Set `Matched: true` in the match path (line 95)**

In the `Route` method, add `Matched: true` to the Decision literal at line 95. Change:

```go
		if result.Matched {
			decision := Decision{
				AgentName:       rule.AgentName,
				Confidence:      NormalizeConfidence(result.Confidence),
				MatchType:       matchType,
				MatchedKeywords: result.MatchedKeywords,
				Reasoning:       buildReasoning(matchType, result),
			}
```

To:

```go
		if result.Matched {
			decision := Decision{
				Matched:         true,
				AgentName:       rule.AgentName,
				Confidence:      NormalizeConfidence(result.Confidence),
				MatchType:       matchType,
				MatchedKeywords: result.MatchedKeywords,
				Reasoning:       buildReasoning(matchType, result),
			}
```

**Change 1d: Set `Matched: true` on the existing default fallback (line 123)**

Add `Matched: true` to the existing fallback `Decision` literal at line 123. This keeps the default fallback working during the transition. Change:

```go
	decision := Decision{
		AgentName:  e.defaultAgent,
		Confidence: 0.5,
		MatchType:  MatchTypeDefault,
		Reasoning:  "No specific rules matched; using default agent",
	}
```

To:

```go
	decision := Decision{
		Matched:    true,
		AgentName:  e.defaultAgent,
		Confidence: 0.5,
		MatchType:  MatchTypeDefault,
		Reasoning:  "No specific rules matched; using default agent",
	}
```

**Verification:** Run `go build ./...` and `go test ./internal/routing/...` in `src/mnemonic/`. All tests pass. The `Matched` field is additive; existing assertions are unaffected.

---

### Step 2: Update Engine Tests for Matched Field

**Goal:** Add `Matched` assertions to all test cases so they explicitly verify the new field. Tests still pass against the Step 1 code.

**Files modified:** 1

---

#### File: `src/mnemonic/internal/routing/engine_test.go`

**Change 2a: Add `wantMatched bool` to the table-driven test struct (line 48)**

Add the field after `wantErr`:

```go
	tests := []struct {
		name              string
		rules             []*routingrule.Rule
		matchers          []routing.RuleMatcher
		defaultAgent      string
		prompt            string
		wantAgent         string
		wantMatchType     routing.MatchType
		wantConfidence    float64
		wantErr           bool
		wantMatched       bool
		wantKeywordsLen   int
		wantReasonContain string
	}{
```

**Change 2b: Set `wantMatched: true` for all matching test cases**

Add `wantMatched: true` to these test cases:
- "first rule matches (short circuit)" (around line 51)
- "second rule matches when first does not" (around line 94)
- "matcher error skips rule and continues" (around line 180)
- "unknown match type skips rule and continues" (around line 228)
- "prompt normalization - trimmed but case preserved" (around line 270)
- "pattern match with fractional confidence" (around line 307)

**Change 2c: Set `wantMatched: true` for fallback test cases (temporarily)**

These will be changed to `false` in Step 3, but for now set them to `true` since the default fallback still produces `Matched: true`:
- "falls through to default when no rules match" (around line 141) -- set `wantMatched: true`
- "no rules at all returns default decision" (around line 169) -- set `wantMatched: true`

**Change 2d: Add `Matched` assertion in the test loop (after line 361)**

After the `require.NoError(t, err)` line and before the `assert.Equal(t, tt.wantAgent, ...)` line, add:

```go
			assert.Equal(t, tt.wantMatched, decision.Matched, "Decision.Matched")
```

**Change 2e: Add `Matched` assertions to standalone test functions**

In `TestEngine_Route_ShortCircuit` (line 427), after `require.NoError(t, err)` add:

```go
	assert.True(t, decision.Matched)
```

In `TestEngine_Route_NilMetrics` (line 467), after `require.NoError(t, err)` add:

```go
	assert.True(t, decision.Matched)
```

In `TestEngine_Route_DisabledRulesSkipped` (line 513), after `require.NoError(t, err)` add:

```go
	assert.True(t, decision.Matched)
```

In `TestEngine_Route_ContextCancellation` (line 556), the assertion `assert.Equal(t, routing.Decision{}, decision)` already covers `Matched: false` since `Decision{}` has `Matched: false`. No change needed.

**Verification:** Run `go test ./internal/routing/...`. All tests pass.

---

### Step 3: Remove defaultAgent from Engine

**Goal:** Remove the `defaultAgent` field, the `defaultAgent` parameter from `NewEngine`, the hardcoded fallback block, and the `MatchTypeDefault` case from `buildReasoning`. Add `recordNoMatch` for observability.

**Files modified:** 2

---

#### File: `src/mnemonic/internal/routing/engine.go`

**Change 3a: Remove `defaultAgent` field from Engine struct (line 23)**

Delete the line:

```go
	defaultAgent string
```

The `Engine` struct becomes:

```go
type Engine struct {
	cache    *RuleCache
	registry *MatcherRegistry
	metrics  *metrics.Routing
	logger   zerolog.Logger
	tracer   trace.Tracer
}
```

**Change 3b: Update Engine doc comment (lines 17-19)**

Replace:

```go
// Engine implements the Evaluator interface. It evaluates routing rules in priority
// order using registered matchers and returns the first matching decision, or a
// default decision if no rules match.
```

With:

```go
// Engine implements the Evaluator interface. It evaluates routing rules in priority
// order using registered matchers and returns the first matching decision, or a
// Decision with Matched: false if no rules match.
```

**Change 3c: Remove `defaultAgent` parameter from `NewEngine` (lines 31-46)**

Replace the entire `NewEngine` function:

```go
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
```

**Change 3d: Update Route doc comment (lines 48-49)**

Replace:

```go
// Route evaluates the prompt against all enabled routing rules in priority order.
// It returns the first match or a default decision if no rules match.
```

With:

```go
// Route evaluates the prompt against all enabled routing rules in priority order.
// It returns the first match or Decision{Matched: false} if no rules match.
```

**Change 3e: Replace the hardcoded fallback block (lines 122-142)**

Replace the entire block from `// No rules matched; return default decision.` through `return decision, nil` with:

```go
	// No rules matched.
	span.SetAttributes(
		attribute.Bool("routing.matched", false),
	)

	e.recordNoMatch(ctx)

	e.logger.Debug().
		Msg("no rules matched")

	return Decision{}, nil
```

Note: `Decision{}` has `Matched: false` by zero-value.

**Change 3f: Add `recordNoMatch` method (after `recordMetrics` method, after line 152)**

Add the following method:

```go
// recordNoMatch records a no-match metric if a metrics recorder is available.
func (e *Engine) recordNoMatch(ctx context.Context) {
	if e.metrics == nil {
		return
	}
	e.metrics.RecordRuleMatch(ctx, "no_match")
}
```

This reuses the existing `RecordRuleMatch` counter with `rule_type=no_match` to record when no rules matched, giving operators visibility without adding a new metric instrument. The `"no_match"` string is a bounded, non-user-provided value, so cardinality is safe.

**Change 3g: Remove `MatchTypeDefault` case from `buildReasoning` (lines 169-170)**

Delete the two lines:

```go
	case MatchTypeDefault:
		return "No specific rules matched; using default agent"
```

The `buildReasoning` function becomes:

```go
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
```

---

#### File: `src/mnemonic/internal/routing/engine_test.go`

**Change 3h: Update `newTestEngine` helper (lines 18-33)**

Remove the `defaultAgent` parameter and the corresponding argument to `NewEngine`. Replace:

```go
func newTestEngine(t *testing.T, rules []*routingrule.Rule, registry *routing.MatcherRegistry, defaultAgent string) *routing.Engine {
	t.Helper()

	loader := &mockRuleLoader{
		loadFn: func(_ context.Context) ([]*routingrule.Rule, error) {
			return rules, nil
		},
	}

	cache, err := routing.NewRuleCache(context.Background(), loader)
	require.NoError(t, err)

	logger := zerolog.Nop()

	return routing.NewEngine(cache, registry, defaultAgent, nil, logger)
}
```

With:

```go
func newTestEngine(t *testing.T, rules []*routingrule.Rule, registry *routing.MatcherRegistry) *routing.Engine {
	t.Helper()

	loader := &mockRuleLoader{
		loadFn: func(_ context.Context) ([]*routingrule.Rule, error) {
			return rules, nil
		},
	}

	cache, err := routing.NewRuleCache(context.Background(), loader)
	require.NoError(t, err)

	logger := zerolog.Nop()

	return routing.NewEngine(cache, registry, nil, logger)
}
```

**Change 3i: Remove `defaultAgent` field from the table-driven test struct**

Remove `defaultAgent string` from the struct definition and all `defaultAgent: "general-agent"` values from every test case (lines 42, 86, 134, 162, 220, 262, 299, 332).

**Change 3j: Update `newTestEngine` calls in the test loop**

At line 350, change:

```go
			engine := newTestEngine(t, tt.rules, registry, tt.defaultAgent)
```

To:

```go
			engine := newTestEngine(t, tt.rules, registry)
```

**Change 3k: Update the "falls through to default" test case (around line 141)**

Change the expected values:
- `wantAgent` from `"general-agent"` to `""`
- `wantMatchType` from `routing.MatchTypeDefault` to `routing.MatchType("")`
- `wantConfidence` from `0.5` to `0.0`
- `wantMatched` from `true` to `false`
- `wantReasonContain` from `"No specific rules matched"` to `""`

Rename the test to `"no match when no rules match prompt"`.

The updated test case:

```go
		{
			name: "no match when no rules match prompt",
			rules: []*routingrule.Rule{
				{
					ID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Name:        "keyword-rule",
					Priority:    100,
					AgentName:   "go-agent",
					MatchType:   "keyword",
					MatchConfig: routingrule.KeywordMatchConfig{Keywords: []string{"go"}, MatchMode: routingrule.MatchModeAny},
					Enabled:     true,
				},
			},
			matchers: []routing.RuleMatcher{
				&mockRuleMatcher{
					matchType: routing.MatchTypeKeyword,
					matchFn: func(_ context.Context, _ string, _ routingrule.MatchConfig) (routing.MatchResult, error) {
						return routing.MatchResult{Matched: false}, nil
					},
				},
			},
			prompt:     "help me with something",
			wantAgent:  "",
			wantMatchType: routing.MatchType(""),
			wantConfidence: 0.0,
			wantMatched: false,
		},
```

Note: `wantReasonContain` is empty string, and the `assert.Contains(t, decision.Reasoning, tt.wantReasonContain)` assertion passes because `strings.Contains("", "")` is `true`. If you want stricter validation, add a conditional: only check `wantReasonContain` when it is non-empty, or assert `decision.Reasoning == ""` for no-match cases.

**Change 3l: Update the "no rules at all" test case (around line 169)**

Apply the same changes as 3k. Rename to `"no match when no rules exist"`:

```go
		{
			name:       "no match when no rules exist",
			rules:      []*routingrule.Rule{},
			matchers:   []routing.RuleMatcher{},
			prompt:     "anything",
			wantAgent:  "",
			wantMatchType: routing.MatchType(""),
			wantConfidence: 0.0,
			wantMatched: false,
		},
```

**Change 3m: Update standalone test function `newTestEngine` calls**

Update all calls in:
- `TestEngine_Route_ShortCircuit` (line 420): change `newTestEngine(t, rules, registry, "general-agent")` to `newTestEngine(t, rules, registry)`
- `TestEngine_Route_NilMetrics` (line 460): same change
- `TestEngine_Route_DisabledRulesSkipped` (line 507): same change
- `TestEngine_Route_ContextCancellation` (line 542): same change

**Change 3n: Update the "unknown match type" test case MatchConfig (around line 237)**

The test case at line 237 uses `routingrule.DefaultMatchConfig{}` for the unknown-type rule's match config. Since `DefaultMatchConfig` is being removed in Step 4, change this to `routingrule.KeywordMatchConfig{}` (or any valid config -- the matcher is never called because the type is unregistered):

```go
				{
					ID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Name:        "unknown-type-rule",
					Priority:    100,
					AgentName:   "unknown-agent",
					MatchType:   "nonexistent",
					MatchConfig: routingrule.KeywordMatchConfig{},
					Enabled:     true,
				},
```

**Verification:** Run `go test ./internal/routing/...`. All tests pass. The no-match cases now assert `Matched: false`, empty agent name, zero confidence.

---

### Step 4: Remove MatchTypeDefault from routingrule Package

**Goal:** Remove the `MatchTypeDefault` constant, `DefaultMatchConfig` struct and method, the `"default"` entry from `ValidMatchTypes`, the `"default"` case from `UnmarshalMatchConfig`, and the corresponding re-export in the routing package.

**Files modified:** 3

---

#### File: `src/mnemonic/internal/repository/routingrule/routingrule.go`

**Change 4a: Remove `MatchTypeDefault` constant (line 21)**

Delete the line:

```go
	MatchTypeDefault MatchType = "default"
```

The constant block becomes:

```go
const (
	MatchTypeKeyword MatchType = "keyword"
	MatchTypeRegex   MatchType = "regex"
	MatchTypePattern MatchType = "pattern"
)
```

**Change 4b: Remove `"default"` from `ValidMatchTypes` (line 62)**

Delete `string(MatchTypeDefault)` from the slice. The slice becomes:

```go
var ValidMatchTypes = []string{
	string(MatchTypeKeyword),
	string(MatchTypeRegex),
	string(MatchTypePattern),
}
```

**Change 4c: Update `Rule.MatchType` doc comment (line 39)**

Change:

```go
	// MatchType determines how MatchConfig is interpreted.
	// Valid values: keyword, regex, pattern, default
```

To:

```go
	// MatchType determines how MatchConfig is interpreted.
	// Valid values: keyword, regex, pattern
```

**Change 4d: Remove `DefaultMatchConfig` struct and method (lines 131-136)**

Delete:

```go
// DefaultMatchConfig is the configuration for match_type = 'default'.
// The default match type always matches and is used as a fallback.
type DefaultMatchConfig struct{}

// Type returns the match type identifier.
func (d DefaultMatchConfig) Type() string { return "default" }
```

**Change 4e: Remove `"default"` case from `UnmarshalMatchConfig` (lines 162-163)**

Delete:

```go
	case "default":
		return DefaultMatchConfig{}, nil
```

---

#### File: `src/mnemonic/internal/routing/routing.go`

**Change 4f: Remove `MatchTypeDefault` re-export (line 20)**

Delete the line:

```go
	MatchTypeDefault = routingrule.MatchTypeDefault
```

The constant block becomes:

```go
const (
	MatchTypeKeyword = routingrule.MatchTypeKeyword
	MatchTypeRegex   = routingrule.MatchTypeRegex
	MatchTypePattern = routingrule.MatchTypePattern
)
```

---

#### File: `src/mnemonic/internal/routing/routing_test.go`

**Change 4g: Remove `MatchTypeDefault` assertion (line 67)**

Delete the line:

```go
	assert.Equal(t, routing.MatchType("default"), routing.MatchTypeDefault)
```

---

#### File: `src/mnemonic/internal/routing/matcher.go`

**Change 4h: Update `RuleMatcher` doc comment (line 11)**

Change:

```go
// RuleMatcher defines the interface for match type implementations.
// Each concrete matcher (keyword, regex, pattern, default) implements this interface.
```

To:

```go
// RuleMatcher defines the interface for match type implementations.
// Each concrete matcher (keyword, regex, pattern) implements this interface.
```

---

#### File: `src/mnemonic/internal/routing/matcher_test.go`

**Change 4i: Update `TestMatcherRegistry_MultipleTypes` (line 90)**

The line `assert.Nil(t, registry.GetMatcher(routing.MatchTypeDefault))` references the removed constant. Replace the line with a string literal test for an unregistered type:

```go
	assert.Nil(t, registry.GetMatcher(routing.MatchType("nonexistent")))
```

**Verification:** Run `go build ./...` and `go test ./internal/routing/... ./internal/repository/routingrule/...`. All compile and pass.

---

### Step 5: Delete DefaultMatcher Files

**Goal:** Delete the DefaultMatcher implementation and its tests.

**Files deleted:** 2

---

**Delete:**

1. `src/mnemonic/internal/routing/default_matcher.go`
2. `src/mnemonic/internal/routing/default_matcher_test.go`

Both files are untracked (`??` in git status), so deletion is simply removing the files from the working tree.

**Verification:** Run `go build ./...` and `go test ./internal/routing/...`. No compilation errors (nothing imports these files). All tests pass.

---

### Step 6: Remove routing.default_agent from Config

**Goal:** Remove the `DefaultAgent` field from `RoutingConfig`, its default value, its validation, and all test references.

**Files modified:** 3

---

#### File: `src/mnemonic/internal/config/defaults.go`

**Change 6a: Remove `DefaultRoutingDefaultAgent` constant (line 60)**

Delete the line:

```go
	DefaultRoutingDefaultAgent        = "general-agent"
```

The routing defaults block becomes:

```go
const (
	DefaultRoutingCacheRefreshTTL     = 5 * time.Minute
	DefaultRoutingCacheStartupTimeout = 30 * time.Second
)
```

---

#### File: `src/mnemonic/internal/config/config.go`

**Change 6b: Remove `DefaultAgent` field from `RoutingConfig` (line 100)**

Delete the line:

```go
	DefaultAgent string             `mapstructure:"default_agent" yaml:"default_agent"`
```

The struct becomes:

```go
type RoutingConfig struct {
	Cache RoutingCacheConfig `mapstructure:"cache" yaml:"cache"`
}
```

**Change 6c: Remove `routing.default_agent` default value (line 299)**

Delete the line:

```go
	v.SetDefault("routing.default_agent", DefaultRoutingDefaultAgent)
```

**Change 6d: Remove `DefaultAgent` validation (lines 634-639)**

In the `(c *RoutingConfig) validate()` method, delete:

```go
	if c.DefaultAgent == "" {
		errs = append(errs, ValidationError{
			Field:   "routing.default_agent",
			Message: "required",
		})
	}
```

The `validate()` method becomes:

```go
func (c *RoutingConfig) validate() ValidationErrors {
	var errs ValidationErrors

	if c.Cache.RefreshTTL < 0 {
		errs = append(errs, ValidationError{
			Field:   "routing.cache.refresh_ttl",
			Message: "must be non-negative",
		})
	}

	if c.Cache.StartupTimeout < 0 {
		errs = append(errs, ValidationError{
			Field:   "routing.cache.startup_timeout",
			Message: "must be non-negative",
		})
	}

	return errs
}
```

---

#### File: `src/mnemonic/internal/config/config_test.go`

**Change 6e: Remove `DefaultAgent` assertion from `TestDefaultValues` (line 73)**

Delete the line:

```go
	assert.Equal(t, config.DefaultRoutingDefaultAgent, cfg.Routing.DefaultAgent)
```

**Change 6f: Remove `default_agent` from YAML in `TestYAMLFileLoading` (line 140)**

In the `configContent` string literal, delete:

```yaml
routing:
  default_agent: custom-agent
```

And delete the corresponding assertion at line 182:

```go
	assert.Equal(t, "custom-agent", cfg.Routing.DefaultAgent)
```

Note: If `routing:` has no remaining keys in the YAML, the block can be removed entirely. However, the routing cache keys are not in this YAML, so just remove the `routing:` block.

**Change 6g: Remove `default_agent` from YAML in `TestEnvironmentVariableOverrides` (line 208)**

In the `configContent` string literal, delete the `routing:` block:

```yaml
routing:
  default_agent: file-agent
```

And delete the corresponding assertion at line 243:

```go
	assert.Equal(t, "file-agent", cfg.Routing.DefaultAgent)
```

**Change 6h: Remove the "empty default_agent" validation test case (lines 665-671)**

In `TestValidation_RoutingConfig`, delete the test case:

```go
		{
			name: "empty default_agent",
			modify: func(cfg *config.MnemonicConfig) {
				cfg.Routing.DefaultAgent = ""
			},
			expectError: "routing.default_agent",
		},
```

**Change 6i: Remove `DefaultAgent` from `TestValidation_MultipleErrors` (line 964)**

Delete the line:

```go
	cfg.Routing.DefaultAgent = ""
```

And delete the corresponding assertion at line 972:

```go
	assert.Contains(t, errStr, "routing.default_agent")
```

**Change 6j: Remove `DefaultAgent` from YAML in `TestConfigFileFlagOverride` (lines 1246, 1252)**

In the `envConfig` string literal, delete:

```yaml
routing:
  default_agent: env-agent
```

In the `flagConfig` string literal, delete:

```yaml
routing:
  default_agent: flag-agent
```

These YAML blocks can be removed entirely since no other routing keys are set in them.

**Change 6k: Remove `DefaultAgent` from `validConfig()` helper (line 1449)**

Delete the line:

```go
			DefaultAgent: "general-agent",
```

The `Routing` field becomes:

```go
		Routing: config.RoutingConfig{
			Cache: config.RoutingCacheConfig{
				RefreshTTL:     5 * time.Minute,
				StartupTimeout: 30 * time.Second,
			},
		},
```

**Verification:** Run `go test ./internal/config/...`. All tests pass.

---

### Step 7: Update OpenAPI Spec

**Goal:** Add `matched` field to `RoutingDecision`, make `agent` optional in `RouteResponse`, remove `MatchMethodDefault` from enum, remove `DefaultMatchConfig` schema, remove `default` from `MatchType` enum.

**File modified:** `api/openapi/mnemonic-v1.yaml`

---

**Change 7a: Update `RoutingDecision` schema (lines 834-878)**

Replace the entire `RoutingDecision` schema with:

```yaml
    RoutingDecision:
      type: object
      description: Routing decision details
      required:
        - matched
      properties:
        matched:
          type: boolean
          description: Whether a routing rule matched the prompt. When false, all other fields are absent.
        agent_name:
          type: string
          description: Selected agent identifier (present only when matched is true)
          examples:
            - go-software-agent
        confidence:
          type: number
          format: double
          description: Routing confidence score (0-1, present only when matched is true)
          minimum: 0
          maximum: 1
          examples:
            - 1.0
        method:
          type: string
          description: Routing method used (present only when matched is true)
          enum:
            - MatchMethodKeyword
            - MatchMethodRegex
            - MatchMethodPattern
          examples:
            - MatchMethodKeyword
        matched_keywords:
          type: array
          description: Keywords that triggered the route (for MatchMethodKeyword)
          items:
            type: string
          examples:
            - - go
              - function
        reasoning:
          type: string
          description: Human-readable routing explanation (present only when matched is true)
          examples:
            - "Matched keywords: go, function"
```

Key changes: `matched` is the only required field. `agent_name`, `confidence`, `method`, and `reasoning` move from required to optional. `MatchMethodDefault` is removed from the `method` enum.

**Change 7b: Update `RouteResponse` schema (lines 932-949)**

Make `agent` optional by removing it from `required`:

```yaml
    RouteResponse:
      type: object
      description: Complete routing response with agent and patterns
      required:
        - routing
      properties:
        routing:
          $ref: "#/components/schemas/RoutingDecision"
        agent:
          $ref: "#/components/schemas/Agent"
          description: Agent details (present only when routing.matched is true)
        patterns:
          type: array
          description: Relevant patterns (if include_patterns is true and routing.matched is true)
          items:
            $ref: "#/components/schemas/RoutePatternResult"
        metadata:
          $ref: "#/components/schemas/RouteMetadata"
```

**Change 7c: Remove `DefaultMatchConfig` schema (lines 1025-1028)**

Delete the entire schema:

```yaml
    DefaultMatchConfig:
      type: object
      description: Configuration for default fallback (no configuration needed)
      additionalProperties: false
```

**Change 7d: Remove `default` from `MatchType` enum (lines 952-964)**

Update the `MatchType` schema. Remove `- default` from the enum and update the description:

```yaml
    MatchType:
      type: string
      description: |
        Type of matching logic for the routing rule.
        - `keyword`: Match against a list of keywords
        - `regex`: Match using regular expression
        - `pattern`: Semantic pattern matching using pattern IDs
      enum:
        - keyword
        - regex
        - pattern
```

**Change 7e: Remove `DefaultMatchConfig` from `RoutingRule.match_config` oneOf (line 1077)**

Delete the line:

```yaml
            - $ref: "#/components/schemas/DefaultMatchConfig"
```

So `match_config` becomes:

```yaml
        match_config:
          oneOf:
            - $ref: "#/components/schemas/KeywordMatchConfig"
            - $ref: "#/components/schemas/RegexMatchConfig"
            - $ref: "#/components/schemas/PatternMatchConfig"
          description: Type-specific configuration
```

**Change 7f: Remove `DefaultMatchConfig` from `RoutingRuleCreate.match_config` oneOf (line 1129)**

Same deletion as 7e.

**Change 7g: Remove `DefaultMatchConfig` from `RoutingRuleUpdate.match_config` oneOf (line 1161)**

Same deletion as 7e.

**Change 7h: Update `/api/route` endpoint description (lines 1405-1409)**

Replace step 3 in the routing decision process:

```
        3. If no rules match, fall back to default agent
```

With:

```
        3. If no rules match, return matched: false (client decides next steps)
```

**Change 7i: Remove the "default-fallback" routing rule from list example (lines 2055-2063)**

Delete the example routing rule with `match_type: default`:

```yaml
                  - id: 550e8400-e29b-41d4-a716-446655440004
                    name: default-fallback
                    priority: 0
                    agent_name: general-agent
                    match_type: default
                    match_config: {}
                    enabled: true
                    created_at: "2024-01-10T08:00:00Z"
                    updated_at: "2024-01-10T08:00:00Z"
```

**Change 7j: Remove `default_rule` example from create routing rule endpoint (lines 2114-2122)**

Delete the `default_rule` example:

```yaml
              default_rule:
                summary: Default fallback rule
                value:
                  name: fallback
                  priority: 0
                  agent_name: general-agent
                  match_type: default
                  match_config: {}
                  enabled: true
```

**Change 7k: Update the 404 response description for `/api/route` (lines 1498-1512)**

Replace the current 404 description and example to reflect the new behavior. Since no-match now returns 200 with `matched: false`, the 404 response becomes applicable only if the matched agent is not found in the database. Update:

```yaml
        "404":
          description: Matched agent not found in database (data integrity issue)
          headers:
            X-Request-ID:
              $ref: "#/components/headers/X-Request-ID"
          content:
            application/problem+json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
              example:
                type: https://mnemonic.example.com/problems/not-found
                title: Not Found
                status: 404
                detail: Agent referenced by routing rule not found
                traceId: 550e8400-e29b-41d4-a716-446655440000
```

**Verification:** Validate the OpenAPI spec with `npx @redocly/cli lint api/openapi/mnemonic-v1.yaml` or similar tool.

---

### Step 8: Update Design and Architecture Docs

**Goal:** Update all design documents to remove default matcher references and reflect the new no-match behavior.

**Files modified:** 6

---

#### File: `docs/design/routing-engine.md`

**Change 8a: Remove "Default Matcher" from Table of Contents (line 27)**

Delete the "Default Matcher" entry.

**Change 8b: Delete the "Default Matcher" section (lines 896-927)**

Delete the entire section from `### Default Matcher` through the example JSON block ending with `}`.

**Change 8c: Remove `default` row from Confidence Scoring table (line 942)**

Delete the row:

```
| default    | 0.5              | Baseline for fallback   |
```

**Change 8d: Remove `D[Default<br/>0.5]` from confidence score diagram (line 956)**

Delete the `D[Default<br/>0.5]` node from the Mermaid flowchart.

**Change 8e: Remove `default` row from Reasoning generation table (line 990)**

Delete the row:

```
| default    | `"No specific rules matched; using default agent"` |
```

**Change 8f: Remove `Default` row from Latency targets table (line 1014)**

Delete the row:

```
| Default | <1ms | <5ms | <10ms | No-op, always matches |
```

**Change 8g: Remove "Default" from Operation-level targets (line 1020)**

Change `Keyword, Regex, Default` to `Keyword, Regex`.

**Change 8h: Remove `default | Fallback only` row from Optimization priority table (line 1082)**

Delete the row:

```
| 0        | default    | Fallback only                      |
```

**Change 8i: Update Error handling table (line 1108)**

Change:

```
| All rules fail          | Return default agent             |
```

To:

```
| All rules fail          | Return no-match decision         |
```

**Change 8j: Update Error handling diagram (line 1136)**

Change `ReturnDefaultDecision` to `ReturnNoMatch` in the Mermaid stateDiagram.

**Change 8k: Update Error handling key principle (line 1142)**

Replace:

```
**Key principle:** The routing engine should never fail to return a routing decision. If all rules fail or error, the default agent handles the request.
```

With:

```
**Key principle:** The routing engine returns a no-match decision when no rules match or all rules fail. The client decides how to handle unmatched prompts.
```

**Change 8l: Add `Matched bool` to Decision in Complete Type Relationships diagram (lines 358-364)**

Add `+bool Matched` as the first field in the `Decision` class.

**Change 8m: Remove `defaultAgent` from Engine class diagram (line 319)**

Delete the line `-string defaultAgent` from the Engine class.

**Change 8n: Remove `MatchType: default -> DefaultMatchConfig` from supporting types (line 299)**

Delete the line.

**Change 8o: Remove `DefaultMatchConfig` class from MatchConfig class diagram (lines 280-288)**

Delete the `DefaultMatchConfig` class definition and its `implements` relationship.

**Change 8p: Remove `"default"` from match type implementations list (around line 300)**

Update any prose that lists concrete matcher types to only include: keyword, regex, pattern.

---

#### File: `docs/design/configuration.md`

**Change 8q: Remove `routing.default_agent` row from configuration table (line 301)**

Delete the entire table row for `routing.default_agent`.

**Change 8r: Remove `default_agent` from YAML example (lines 162-163)**

Delete:

```yaml
  # Default agent when no rules match
  default_agent: general-agent
```

---

#### File: `docs/architecture/08-data-architecture.md`

**Change 8s: Remove `default` from ER diagram `match_type` enum (line 213)**

Change `"keyword|regex|pattern|default"` to `"keyword|regex|pattern"`.

Add a note: `# Note: 'default' is deprecated and will be removed from the database constraint in a future migration.`

**Change 8t: Remove `# Default match (fallback)` example (lines 331-332)**

Delete:

```yaml
# Default match (fallback)
match_config: {}
```

**Change 8u: Add deprecation comment to SQL CHECK constraint (line 408)**

The SQL block is documentation, not a live migration. Add a comment noting `default` is deprecated:

```sql
    match_type VARCHAR(20) NOT NULL
        CHECK (match_type IN ('keyword', 'regex', 'pattern', 'default')),
        -- NOTE: 'default' is deprecated. The application no longer creates default-type rules.
        -- A future migration (post-MVP) will remove 'default' from this constraint.
```

**Change 8v: Remove `(match_type = 'default')` from match_config validation constraint (line 929)**

Update the constraint documentation to remove the default condition:

```sql
CHECK (
    (match_type = 'keyword' AND match_config ? 'keywords' AND match_config ? 'match_mode') OR
    (match_type = 'regex' AND match_config ? 'pattern') OR
    (match_type = 'pattern' AND match_config ? 'pattern_ids')
);
-- NOTE: 'default' match_type condition removed. Existing default-type rules are
-- deprecated and will be skipped by the routing engine.
```

---

#### File: `docs/code-reviews/phase-13-default-matcher.md`

**Change 8w: Add superseded notice**

Add the following at the top of the file, after any frontmatter:

```
> **Superseded:** The DefaultMatcher was removed before merge. See
> `docs/architecture/design-change-remove-default-matcher.md` for the design change
> assessment that led to this decision.
```

---

#### File: `docs/architecture/02-architectural-decisions.md`

**Change 8x: Add ADR-007 for Explicit No-Match Signaling**

Add to the Table of Contents:

```
- [ADR-007: Explicit No-Match Signaling](#adr-007-explicit-no-match-signaling)
```

Add the ADR at the end of the document, before the Decision Summary:

```markdown
## ADR-007: Explicit No-Match Signaling

**Date:** 2026-02-12
**Status:** Accepted
**Supersedes:** Phase 13 original design (implement DefaultMatcher)

### Context

Phase 13 was originally scoped as "Implement default matcher (fallback routing)." During code review, finding H1 identified a dual-path default fallback: (1) an engine-level hardcoded fallback using `defaultAgent` and (2) a `DefaultMatcher` that always returns `Matched: true` with confidence 0.5.

Both paths silently route unmatched prompts to a default agent, preventing the client from knowing that no rules actually matched. This violates separation of concerns: the routing engine's job is to evaluate rules, not to decide policy for unmatched prompts.

### Decision

"A default match is no match."

When no routing rules match a prompt, the engine returns `Decision{Matched: false}, nil`. The `Matched bool` field on `Decision` provides an explicit, unambiguous signal. The client (ACE CLI) decides how to handle unmatched prompts: route to a general agent, ask the user to rephrase, or take any other action.

### Consequences

- The `DefaultMatcher`, `DefaultMatchConfig`, `MatchTypeDefault` constant, and `routing.default_agent` configuration are all removed.
- The `Decision` struct gains a `Matched bool` field (zero-value `false` is the safe default).
- The HTTP handler for `/api/route` returns 200 OK with `matched: false` for unmatched prompts.
- The database `match_type CHECK` constraint still allows `'default'` for backward compatibility; a future migration will tighten it.
- Downstream phases (14-16) are updated to work with the new signaling.
```

Update the Decision Summary table to include ADR-007.

---

#### File: `docs/plans/mvp-implementation-plan.md`

**Change 8y: Update Phase 13 row (line 42)**

Replace:

```
| 13    |      | Implement default matcher (fallback routing).                                                                                     | go software agent                             | [Routing Engine - Default Matcher](../design/routing-engine.md#default-matcher)                                                                                                                                                                                                                           |          |
```

With:

```
| 13    |      | Remove default fallback, implement explicit no-match signaling (see [Phase 13 Plan](phase-13-implementation-plan.md)).            | go software agent                             | [Design Change Assessment](../architecture/design-change-remove-default-matcher.md)                                                                                                                                                                                                                       |          |
```

---

#### File: `docs/function-documentation-map.md`

**Change 8z: Remove DefaultMatcher entries**

If `docs/function-documentation-map.md` contains entries for `default_matcher.go` exported functions (`NewDefaultMatcher`, `DefaultMatcher.Type`, `DefaultMatcher.Close`, `DefaultMatcher.Match`), remove those entries.

Add the new `Engine.recordNoMatch` method if the map tracks unexported methods (unlikely, but verify).

---

### Step 9: Full Test Suite and Static Analysis

**Goal:** Verify everything compiles, all tests pass, and static analysis is clean.

**Commands to run (from `src/mnemonic/`):**

```bash
# Compile check
go build ./...

# All unit tests
go test ./...

# Run goimports on all changed .go files
goimports -w internal/routing/routing.go
goimports -w internal/routing/engine.go
goimports -w internal/routing/matcher.go
goimports -w internal/repository/routingrule/routingrule.go
goimports -w internal/config/config.go
goimports -w internal/config/defaults.go

# Static analysis
golangci-lint run ./...
govulncheck ./...
gosec ./...
```

**Expected results:**

- `go build ./...` -- no errors
- `go test ./...` -- all tests pass
- `golangci-lint` -- no new findings
- No `MatchTypeDefault` or `DefaultMatcher` or `defaultAgent` or `default_agent` references remain in Go source files (excluding the database migration SQL files which are immutable)

---

## Delegation Plan

Main Claude should coordinate the following delegations:

| Order | Agent                    | Task                                                                 |
| ----- | ------------------------ | -------------------------------------------------------------------- |
| 1     | `go-software-engineer`   | Execute Steps 1-6 and Step 9 (all Go code changes)                   |
| 2     | `api-architect`          | Execute Step 7 (OpenAPI spec changes)                                |
| 3     | `technical-writer`       | Execute Step 8 (design and architecture doc updates)                 |
| 4     | `/code-review` skill     | Review all changes before merge                                      |

Steps 1-6 should be executed sequentially by the Go engineer in a single session, running tests after each step to catch issues incrementally. Steps 7 and 8 can be done in parallel with or after the Go changes. Step 9 is the final verification gate.

### E2E Test Updates

The E2E test file `src/mnemonic/tests/e2e/routing_test.go` contains stub tests (all `t.Skip`). The following stubs need comment/description updates but no code changes since they are not yet implemented:

- `TestRoute_DefaultMatch` (line 201): Update comments to describe the new expected behavior (200 OK with `matched: false` instead of `MatchMethodDefault`)
- `TestRoute_DisabledRulesSkipped` (line 259): Update comment "routes to default instead" to "returns no-match"
- `TestRoute_ReasoningExplains` (line 491): Remove "For default: indicates fallback" comment
- `TestRoute_ConfidenceValues` (line 508): Remove "Default match: confidence may be lower" comment
- Header comment block (line 31): Remove `MatchMethodDefault` from the routing methods list

These are documentation-only changes to stub tests and should be included in the `technical-writer` delegation.
