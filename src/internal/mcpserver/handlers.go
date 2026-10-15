package mcpserver

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog"

	"github.com/twistingmercury/mnemonic/internal/service"
	searchsvc "github.com/twistingmercury/mnemonic/internal/service/search"
)

const (
	defaultSearchLimit = 10
	maxSearchLimit     = 50

	defaultRelatedLimit = 5
	maxRelatedLimit     = 20
)

// handleSearchPatterns returns a handler for the search_patterns tool.
// It validates input constraints, delegates to the search service, and
// formats results as markdown.
func handleSearchPatterns(deps ToolDependencies, logger zerolog.Logger, defaultThreshold float64) mcp.ToolHandlerFor[SearchPatternsInput, any] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input SearchPatternsInput) (*mcp.CallToolResult, any, error) {
		// Apply defaults.
		limit := defaultSearchLimit
		if input.Limit != nil {
			limit = *input.Limit
		}

		threshold := defaultThreshold
		if input.Threshold != nil {
			threshold = *input.Threshold
		}

		// Validate constraints.
		if limit < 1 || limit > maxSearchLimit {
			return nil, nil, fmt.Errorf("%w: limit must be between 1 and %d, got %d", ErrInvalidInput, maxSearchLimit, limit)
		}

		if threshold < 0 || threshold > 1 {
			return nil, nil, fmt.Errorf("%w: threshold must be between 0.0 and 1.0, got %f", ErrInvalidInput, threshold)
		}

		withTrace(ctx, logger).Info().
			Str("tool", "search_patterns").
			Str("query_preview", truncate(input.Query, 100)).
			Msg("mcp tool invoked")

		result, err := deps.SearchPatterns(ctx, searchsvc.SearchOptions{
			Query:     input.Query,
			Limit:     limit,
			Threshold: threshold,
			Tags:      input.Tags,
			Language:  input.Language,
			Domain:    input.Domain,
		})
		if err != nil {
			return nil, nil, mapServiceError(err)
		}

		markdown := formatSearchResults(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: markdown}},
		}, nil, nil
	}
}

// handleFindRelatedPatterns returns a handler for the find_related_patterns tool.
// It validates the pattern UUID and limit, delegates to the pattern service,
// and formats results as markdown.
func handleFindRelatedPatterns(deps ToolDependencies, logger zerolog.Logger) mcp.ToolHandlerFor[FindRelatedPatternsInput, any] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input FindRelatedPatternsInput) (*mcp.CallToolResult, any, error) {
		patternID, err := uuid.Parse(input.PatternID)
		if err != nil {
			return nil, nil, fmt.Errorf("%w: invalid UUID %q", ErrInvalidInput, input.PatternID)
		}

		limit := defaultRelatedLimit
		if input.Limit != nil {
			limit = *input.Limit
		}

		if limit < 1 || limit > maxRelatedLimit {
			return nil, nil, fmt.Errorf("%w: limit must be between 1 and %d, got %d", ErrInvalidInput, maxRelatedLimit, limit)
		}

		withTrace(ctx, logger).Info().
			Str("tool", "find_related_patterns").
			Str("pattern_id", patternID.String()).
			Msg("mcp tool invoked")

		sourceName, results, err := deps.FindRelatedPatterns(ctx, patternID, limit)
		if err != nil {
			return nil, nil, mapServiceError(err)
		}

		markdown := formatRelatedPatterns(sourceName, results)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: markdown}},
		}, nil, nil
	}
}

// handleGetPattern returns a handler for the get_pattern tool.
// It validates the pattern UUID, retrieves the pattern with graph context,
// and formats the result as markdown.
func handleGetPattern(deps ToolDependencies, logger zerolog.Logger) mcp.ToolHandlerFor[GetPatternInput, any] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input GetPatternInput) (*mcp.CallToolResult, any, error) {
		id, err := uuid.Parse(input.ID)
		if err != nil {
			return nil, nil, fmt.Errorf("%w: invalid UUID %q", ErrInvalidInput, input.ID)
		}

		withTrace(ctx, logger).Info().
			Str("tool", "get_pattern").
			Str("pattern_id", id.String()).
			Msg("mcp tool invoked")

		pattern, graphCtx, err := deps.GetPatternWithGraph(ctx, id)
		if err != nil {
			return nil, nil, mapServiceError(err)
		}

		markdown := formatPattern(pattern, graphCtx)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: markdown}},
		}, nil, nil
	}
}

// mapServiceError translates service-layer errors to MCP-layer sentinel
// errors: unknown patterns are not found, everything else is unavailable.
func mapServiceError(err error) error {
	if errors.Is(err, service.ErrNotFound) {
		return fmt.Errorf("%w: %v", ErrPatternNotFound, err)
	}
	return fmt.Errorf("%w: %v", ErrServiceUnavailable, err)
}

// truncate shortens a string to maxLen characters, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
