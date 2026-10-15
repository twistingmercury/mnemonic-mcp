// Package mcpserver provides the MCP (Model Context Protocol) server implementation.
package mcpserver

import (
	"context"

	"github.com/google/uuid"
	patternrepo "github.com/twistingmercury/mnemonic/internal/repository/pattern"
	patternsvc "github.com/twistingmercury/mnemonic/internal/service/pattern"
	searchsvc "github.com/twistingmercury/mnemonic/internal/service/search"
)

// ToolDependencies defines what the MCP tool handlers require, so they can be
// tested without databases. Each method maps 1:1 to an MCP tool:
//   - SearchPatterns      -> search_patterns tool
//   - FindRelatedPatterns -> find_related_patterns tool (returns the source pattern's name too)
//   - GetPatternWithGraph -> get_pattern tool
type ToolDependencies interface {
	SearchPatterns(ctx context.Context, opts searchsvc.SearchOptions) (*searchsvc.SearchResult, error)
	FindRelatedPatterns(ctx context.Context, patternID uuid.UUID, limit int) (string, []patternsvc.RelatedPatternResult, error)
	GetPatternWithGraph(ctx context.Context, id uuid.UUID) (*patternrepo.Pattern, *patternsvc.GraphContext, error)
}

// toolDeps delegates each call to the underlying service.
type toolDeps struct {
	search  *searchsvc.Service
	pattern *patternsvc.Service
}

// NewToolDependencies creates ToolDependencies backed by the given search and
// pattern services.
func NewToolDependencies(search *searchsvc.Service, pattern *patternsvc.Service) ToolDependencies {
	return toolDeps{search: search, pattern: pattern}
}

func (t toolDeps) SearchPatterns(ctx context.Context, opts searchsvc.SearchOptions) (*searchsvc.SearchResult, error) {
	return t.search.SearchPatterns(ctx, opts)
}

func (t toolDeps) FindRelatedPatterns(ctx context.Context, patternID uuid.UUID, limit int) (string, []patternsvc.RelatedPatternResult, error) {
	return t.pattern.FindRelatedPatterns(ctx, patternID, limit)
}

func (t toolDeps) GetPatternWithGraph(ctx context.Context, id uuid.UUID) (*patternrepo.Pattern, *patternsvc.GraphContext, error) {
	return t.pattern.GetPatternWithGraph(ctx, id)
}
