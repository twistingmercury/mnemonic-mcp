// Package patterns provides HTTP handlers for pattern CRUD, association,
// and search operations. It registers Gin routes for creating, reading,
// updating, deleting, and searching context patterns via the REST API.
//
// Documentation:
//   - API: docs/api/openapi/mnemonic-v1.yaml (Pattern Endpoints)
//   - Design: docs/design/service-layer.md (PatternService, SearchService)
package patterns

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/twistingmercury/mnemonic/internal/handlers"
	patternrepo "github.com/twistingmercury/mnemonic/internal/repository/pattern"
	patternsvc "github.com/twistingmercury/mnemonic/internal/service/pattern"
	searchsvc "github.com/twistingmercury/mnemonic/internal/service/search"
)

// Handler provides HTTP handlers for pattern CRUD and search operations.
type Handler struct {
	patternSvc patternsvc.Service
	searchSvc  searchsvc.Service
}

// New creates a new pattern Handler backed by the given services.
func New(patternSvc patternsvc.Service, searchSvc searchsvc.Service) *Handler {
	return &Handler{
		patternSvc: patternSvc,
		searchSvc:  searchSvc,
	}
}

// RegisterRoutes binds pattern endpoints to the given router group.
// The group should be mounted at /v1/api.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/patterns", h.Create)
	rg.GET("/patterns", h.List)
	rg.GET("/patterns/search", h.Search)
	rg.GET("/patterns/:id", h.Get)
	rg.PUT("/patterns/:id", h.Update)
	rg.DELETE("/patterns/:id", h.Delete)
	rg.GET("/patterns/:id/agents", h.GetAgentAssociations)
	rg.PUT("/patterns/:id/agents", h.SetAgentAssociations)
}

// --- Request/Response Types ---

type associationRequest struct {
	AgentName string  `json:"agent_name" binding:"required"`
	Relevance float64 `json:"relevance"`
}

type patternCreateRequest struct {
	Name              string               `json:"name" binding:"required"`
	Description       *string              `json:"description"`
	Content           string               `json:"content" binding:"required"`
	Tags              []string             `json:"tags"`
	AgentAssociations []associationRequest `json:"agent_associations"`
}

type patternUpdateRequest struct {
	Name              string               `json:"name" binding:"required"`
	Description       *string              `json:"description"`
	Content           string               `json:"content" binding:"required"`
	Tags              []string             `json:"tags"`
	AgentAssociations []associationRequest `json:"agent_associations"`
}

type associationResponse struct {
	AgentName string  `json:"agent_name"`
	Relevance float64 `json:"relevance"`
}

type relatedPatternResponse struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Relationship string  `json:"relationship"`
	Strength     float64 `json:"strength"`
}

type conceptResponse struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type graphContextResponse struct {
	RelatedPatterns []relatedPatternResponse `json:"related_patterns"`
	Concepts        []conceptResponse        `json:"concepts"`
}

type patternResponse struct {
	ID               string                `json:"id"`
	Name             string                `json:"name"`
	Description      *string               `json:"description"`
	Content          string                `json:"content"`
	Tags             []string              `json:"tags"`
	AgentAssociation []associationResponse `json:"agent_associations,omitempty"`
	EnrichmentStatus string                `json:"enrichment_status"`
	EnrichmentError  *string               `json:"enrichment_error"`
	EnrichedAt       *string               `json:"enriched_at"`
	Graph            *graphContextResponse `json:"graph"`
	CreatedAt        string                `json:"created_at"`
	UpdatedAt        string                `json:"updated_at"`
}

type patternSummaryResponse struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Description      *string  `json:"description"`
	Tags             []string `json:"tags"`
	EnrichmentStatus string   `json:"enrichment_status"`
	CreatedAt        string   `json:"created_at"`
	UpdatedAt        string   `json:"updated_at"`
}

type patternListResponse struct {
	Data       []patternSummaryResponse `json:"data"`
	Pagination handlers.Pagination      `json:"pagination"`
}

type associationsRequest struct {
	Associations []associationRequest `json:"associations" binding:"required"`
}

type associationsResponse struct {
	Associations []associationResponse `json:"associations"`
}

type searchResultResponse struct {
	ID               string                `json:"id"`
	Name             string                `json:"name"`
	Description      *string               `json:"description"`
	Content          string                `json:"content"`
	Tags             []string              `json:"tags"`
	Similarity       float64               `json:"similarity"`
	AgentAssociation []associationResponse `json:"agent_associations,omitempty"`
}

type searchMetadata struct {
	Query            string `json:"query"`
	TotalCandidates  int    `json:"total_candidates"`
	SearchDurationMs int64  `json:"search_duration_ms"`
}

type searchResponse struct {
	Results  []searchResultResponse `json:"results"`
	Metadata searchMetadata         `json:"metadata"`
}

// --- Converters ---

func toPatternResponse(p *patternrepo.Pattern, graph *patternsvc.GraphContext, assocs []associationResponse) patternResponse {
	tags := p.Tags
	if tags == nil {
		tags = []string{}
	}

	resp := patternResponse{
		ID:               p.ID.String(),
		Name:             p.Name,
		Description:      p.Description,
		Content:          p.Content,
		Tags:             tags,
		AgentAssociation: assocs,
		EnrichmentStatus: p.EnrichmentStatus,
		EnrichmentError:  p.EnrichmentError,
		CreatedAt:        p.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:        p.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}

	if p.EnrichedAt != nil {
		s := p.EnrichedAt.UTC().Format("2006-01-02T15:04:05Z")
		resp.EnrichedAt = &s
	}

	if graph != nil {
		gc := graphContextResponse{
			RelatedPatterns: make([]relatedPatternResponse, len(graph.RelatedPatterns)),
			Concepts:        make([]conceptResponse, len(graph.Concepts)),
		}
		for i, r := range graph.RelatedPatterns {
			gc.RelatedPatterns[i] = relatedPatternResponse{
				ID:           r.ID.String(),
				Name:         r.Name,
				Relationship: r.Relationship,
				Strength:     r.Similarity,
			}
		}
		for i, c := range graph.Concepts {
			gc.Concepts[i] = conceptResponse{
				Name: c.Name,
			}
		}
		resp.Graph = &gc
	}

	return resp
}

func toPatternSummary(p *patternrepo.Pattern) patternSummaryResponse {
	tags := p.Tags
	if tags == nil {
		tags = []string{}
	}
	return patternSummaryResponse{
		ID:               p.ID.String(),
		Name:             p.Name,
		Description:      p.Description,
		Tags:             tags,
		EnrichmentStatus: p.EnrichmentStatus,
		CreatedAt:        p.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:        p.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

func toAssociationResponses(assocs []patternrepo.AgentAssociation, names map[uuid.UUID]string) []associationResponse {
	result := make([]associationResponse, len(assocs))
	for i, a := range assocs {
		result[i] = associationResponse{
			AgentName: names[a.AgentID],
			Relevance: a.Relevance,
		}
	}
	return result
}

func toAssociationInputs(reqs []associationRequest) []patternsvc.AssociationInput {
	inputs := make([]patternsvc.AssociationInput, len(reqs))
	for i, r := range reqs {
		relevance := r.Relevance
		if relevance == 0 {
			relevance = 1.0
		}
		inputs[i] = patternsvc.AssociationInput{
			AgentName: r.AgentName,
			Relevance: relevance,
		}
	}
	return inputs
}

// --- Handlers ---

// Create handles POST /v1/api/patterns.
func (h *Handler) Create(c *gin.Context) {
	var req patternCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handlers.RespondValidationError(c, "The request body contains invalid fields", nil)
		return
	}

	if req.Tags == nil {
		req.Tags = []string{}
	}
	if req.AgentAssociations == nil {
		req.AgentAssociations = []associationRequest{}
	}

	pattern, err := h.patternSvc.Create(c.Request.Context(), patternsvc.CreateInput{
		Name:              req.Name,
		Description:       req.Description,
		Content:           req.Content,
		Tags:              req.Tags,
		AgentAssociations: toAssociationInputs(req.AgentAssociations),
	})
	if err != nil {
		handlers.RespondError(c, err)
		return
	}

	resp := toPatternResponse(pattern, nil, nil)
	c.Header("Location", fmt.Sprintf("/v1/api/patterns/%s", pattern.ID))
	c.JSON(http.StatusAccepted, resp)
}

// List handles GET /v1/api/patterns.
func (h *Handler) List(c *gin.Context) {
	limit := handlers.ParseIntQuery(c, "limit", 20, 1, 100)
	offset := handlers.DecodeCursor(c.Query("cursor"))

	var tags []string
	if raw := c.Query("tags"); raw != "" {
		tags = strings.Split(raw, ",")
	}

	filter := patternrepo.Filter{
		Tags:        tags,
		SearchQuery: c.Query("search"),
	}

	patterns, _, err := h.patternSvc.List(c.Request.Context(), filter, patternsvc.ListOptions{
		Offset: offset,
		Limit:  limit + 1,
	})
	if err != nil {
		handlers.RespondError(c, err)
		return
	}

	hasMore := len(patterns) > limit
	if hasMore {
		patterns = patterns[:limit]
	}

	data := make([]patternSummaryResponse, len(patterns))
	for i, p := range patterns {
		data[i] = toPatternSummary(p)
	}

	var cursorPtr *string
	if raw := c.Query("cursor"); raw != "" {
		cursorPtr = &raw
	}

	var nextCursor *string
	if hasMore {
		nc := handlers.EncodeCursor(offset + limit)
		nextCursor = &nc
	}

	c.JSON(http.StatusOK, patternListResponse{
		Data: data,
		Pagination: handlers.Pagination{
			Limit:      limit,
			Cursor:     cursorPtr,
			NextCursor: nextCursor,
			HasMore:    hasMore,
		},
	})
}

// Get handles GET /v1/api/patterns/:id.
func (h *Handler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		handlers.RespondValidationError(c, "Invalid pattern ID format", []handlers.FieldError{
			{Field: "id", Code: "INVALID_FORMAT", Message: "id must be a valid UUID"},
		})
		return
	}

	pattern, graph, err := h.patternSvc.GetWithGraph(c.Request.Context(), id)
	if err != nil {
		handlers.RespondError(c, err)
		return
	}

	// Fetch agent associations separately (handler composition per design doc).
	pgAssocs, err := h.patternSvc.GetAgentAssociations(c.Request.Context(), id)
	if err != nil {
		handlers.RespondError(c, err)
		return
	}

	// Resolve agent UUIDs to human-readable names for the REST response.
	agentIDs := make([]uuid.UUID, len(pgAssocs))
	for i, a := range pgAssocs {
		agentIDs[i] = a.AgentID
	}
	names, err := h.patternSvc.ResolveAgentNames(c.Request.Context(), agentIDs)
	if err != nil {
		handlers.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, toPatternResponse(pattern, graph, toAssociationResponses(pgAssocs, names)))
}

// Update handles PUT /v1/api/patterns/:id.
func (h *Handler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		handlers.RespondValidationError(c, "Invalid pattern ID format", []handlers.FieldError{
			{Field: "id", Code: "INVALID_FORMAT", Message: "id must be a valid UUID"},
		})
		return
	}

	var req patternUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handlers.RespondValidationError(c, "The request body contains invalid fields", nil)
		return
	}

	if req.Tags == nil {
		req.Tags = []string{}
	}
	if req.AgentAssociations == nil {
		req.AgentAssociations = []associationRequest{}
	}

	pattern, err := h.patternSvc.Update(c.Request.Context(), id, patternsvc.UpdateInput{
		Name:              req.Name,
		Description:       req.Description,
		Content:           req.Content,
		Tags:              req.Tags,
		AgentAssociations: toAssociationInputs(req.AgentAssociations),
	})
	if err != nil {
		handlers.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, toPatternResponse(pattern, nil, nil))
}

// Delete handles DELETE /v1/api/patterns/:id.
func (h *Handler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		handlers.RespondValidationError(c, "Invalid pattern ID format", []handlers.FieldError{
			{Field: "id", Code: "INVALID_FORMAT", Message: "id must be a valid UUID"},
		})
		return
	}

	if err := h.patternSvc.Delete(c.Request.Context(), id); err != nil {
		handlers.RespondError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// GetAgentAssociations handles GET /v1/api/patterns/:id/agents.
func (h *Handler) GetAgentAssociations(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		handlers.RespondValidationError(c, "Invalid pattern ID format", []handlers.FieldError{
			{Field: "id", Code: "INVALID_FORMAT", Message: "id must be a valid UUID"},
		})
		return
	}

	pgAssocs, err := h.patternSvc.GetAgentAssociations(c.Request.Context(), id)
	if err != nil {
		handlers.RespondError(c, err)
		return
	}

	// Resolve agent UUIDs to human-readable names for the REST response.
	agentIDs := make([]uuid.UUID, len(pgAssocs))
	for i, a := range pgAssocs {
		agentIDs[i] = a.AgentID
	}
	names, err := h.patternSvc.ResolveAgentNames(c.Request.Context(), agentIDs)
	if err != nil {
		handlers.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, associationsResponse{Associations: toAssociationResponses(pgAssocs, names)})
}

// SetAgentAssociations handles PUT /v1/api/patterns/:id/agents.
func (h *Handler) SetAgentAssociations(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		handlers.RespondValidationError(c, "Invalid pattern ID format", []handlers.FieldError{
			{Field: "id", Code: "INVALID_FORMAT", Message: "id must be a valid UUID"},
		})
		return
	}

	var req associationsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handlers.RespondValidationError(c, "The request body contains invalid fields", nil)
		return
	}

	inputs := toAssociationInputs(req.Associations)
	if err := h.patternSvc.SetAgentAssociations(c.Request.Context(), id, inputs); err != nil {
		handlers.RespondError(c, err)
		return
	}

	// Return the newly set associations.
	assocs := make([]associationResponse, len(req.Associations))
	for i, a := range req.Associations {
		relevance := a.Relevance
		if relevance == 0 {
			relevance = 1.0
		}
		assocs[i] = associationResponse{
			AgentName: a.AgentName,
			Relevance: relevance,
		}
	}

	c.JSON(http.StatusOK, associationsResponse{Associations: assocs})
}

// Search handles GET /v1/api/patterns/search.
func (h *Handler) Search(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		handlers.RespondValidationError(c, "Search query is required", []handlers.FieldError{
			{Field: "q", Code: "REQUIRED", Message: "q query parameter is required"},
		})
		return
	}

	limit := handlers.ParseIntQuery(c, "limit", 10, 1, 50)
	threshold := handlers.ParseFloatQuery(c, "threshold", 0.7, 0.0, 1.0)

	var tags []string
	if raw := c.Query("tags"); raw != "" {
		tags = strings.Split(raw, ",")
	}

	result, err := h.searchSvc.SearchPatterns(c.Request.Context(), searchsvc.SearchOptions{
		Query:     query,
		Limit:     limit,
		Threshold: threshold,
		Tags:      tags,
		AgentName: c.Query("agent"),
	})
	if err != nil {
		handlers.RespondError(c, err)
		return
	}

	results := make([]searchResultResponse, len(result.Matches))
	for i, m := range result.Matches {
		matchTags := m.Pattern.Tags
		if matchTags == nil {
			matchTags = []string{}
		}
		results[i] = searchResultResponse{
			ID:          m.Pattern.ID.String(),
			Name:        m.Pattern.Name,
			Description: m.Pattern.Description,
			Content:     m.Pattern.Content,
			Tags:        matchTags,
			Similarity:  m.Similarity,
		}
	}

	c.JSON(http.StatusOK, searchResponse{
		Results: results,
		Metadata: searchMetadata{
			Query:            result.Query,
			TotalCandidates:  result.TotalCandidates,
			SearchDurationMs: result.SearchDurationMs,
		},
	})
}
