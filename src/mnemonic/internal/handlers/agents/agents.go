// Package agents provides HTTP handlers for agent CRUD operations.
// It registers Gin routes for creating, reading, updating, and deleting
// agent definitions via the REST API.
//
// Documentation:
//   - API: docs/api/openapi/mnemonic-v1.yaml (Agent Endpoints)
//   - Design: docs/design/service-layer.md (AgentService)
package agents

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/twistingmercury/mnemonic/internal/handlers"
	agentrepo "github.com/twistingmercury/mnemonic/internal/repository/agent"
	agentsvc "github.com/twistingmercury/mnemonic/internal/service/agent"
)

// Handler provides HTTP handlers for agent CRUD operations.
type Handler struct {
	svc agentsvc.Service
}

// New creates a new agent Handler backed by the given service.
func New(svc agentsvc.Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes binds agent endpoints to the given router group.
// The group should be mounted at /v1/api.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/agents", h.Create)
	rg.GET("/agents", h.List)
	rg.GET("/agents/:name", h.Get)
	rg.PUT("/agents/:name", h.Update)
	rg.DELETE("/agents/:name", h.Delete)
}

// agentCreateRequest maps the OpenAPI AgentCreate schema.
type agentCreateRequest struct {
	Name         string   `json:"name" binding:"required"`
	Description  string   `json:"description" binding:"required"`
	SystemPrompt string   `json:"system_prompt" binding:"required"`
	Model        string   `json:"model" binding:"required"`
	AllowedTools []string `json:"allowed_tools"`
	Version      string   `json:"version" binding:"required"`
}

// agentUpdateRequest maps the OpenAPI AgentUpdate schema.
type agentUpdateRequest struct {
	Name         string   `json:"name"`
	Description  string   `json:"description" binding:"required"`
	SystemPrompt string   `json:"system_prompt" binding:"required"`
	Model        string   `json:"model" binding:"required"`
	AllowedTools []string `json:"allowed_tools"`
	Version      string   `json:"version" binding:"required"`
}

// agentResponse maps an agent repository object to the OpenAPI Agent schema.
type agentResponse struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	SystemPrompt string   `json:"system_prompt"`
	Model        string   `json:"model"`
	AllowedTools []string `json:"allowed_tools"`
	Version      string   `json:"version"`
	CRC64        string   `json:"crc64"`
	CreatedAt    string   `json:"created_at"`
	UpdatedAt    string   `json:"updated_at"`
}

// agentListResponse maps the OpenAPI AgentList schema.
type agentListResponse struct {
	Data       []agentResponse     `json:"data"`
	Pagination handlers.Pagination `json:"pagination"`
}

// toAgentResponse converts a repository Agent to the wire-format response.
// It returns an error if the stored JSONB definition cannot be unmarshaled.
func toAgentResponse(a *agentrepo.Agent) (agentResponse, error) {
	// Unmarshal the JSONB definition to extract flat fields.
	var def struct {
		Description  string   `json:"description"`
		SystemPrompt string   `json:"system_prompt"`
		Model        string   `json:"model"`
		AllowedTools []string `json:"allowed_tools"`
		Version      string   `json:"version"`
	}
	if err := json.Unmarshal(a.Definition, &def); err != nil {
		return agentResponse{}, fmt.Errorf("corrupt agent definition for %s: %w", a.ID, err)
	}

	if def.AllowedTools == nil {
		def.AllowedTools = []string{}
	}

	return agentResponse{
		ID:           a.ID.String(),
		Name:         a.Name,
		Description:  def.Description,
		SystemPrompt: def.SystemPrompt,
		Model:        def.Model,
		AllowedTools: def.AllowedTools,
		Version:      def.Version,
		CRC64:        a.CRC64,
		CreatedAt:    a.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:    a.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}, nil
}

// Create handles POST /v1/api/agents.
func (h *Handler) Create(c *gin.Context) {
	var req agentCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handlers.RespondValidationError(c, "The request body contains invalid fields", nil)
		return
	}

	if req.AllowedTools == nil {
		req.AllowedTools = []string{}
	}

	agent, err := h.svc.Create(c.Request.Context(), agentsvc.CreateInput{
		Name:         req.Name,
		Description:  req.Description,
		SystemPrompt: req.SystemPrompt,
		Model:        req.Model,
		AllowedTools: req.AllowedTools,
		Version:      req.Version,
	})
	if err != nil {
		handlers.RespondError(c, err)
		return
	}

	resp, err := toAgentResponse(agent)
	if err != nil {
		handlers.RespondError(c, err)
		return
	}
	c.Header("Location", fmt.Sprintf("/v1/api/agents/%s", agent.Name))
	c.JSON(http.StatusCreated, resp)
}

// List handles GET /v1/api/agents.
func (h *Handler) List(c *gin.Context) {
	limit := handlers.ParseIntQuery(c, "limit", 100, 1, 200)
	offset := handlers.DecodeCursor(c.Query("cursor"))

	agents, _, err := h.svc.List(c.Request.Context(), agentsvc.ListOptions{
		Offset: offset,
		Limit:  limit + 1, // Fetch one extra to detect has_more.
	})
	if err != nil {
		handlers.RespondError(c, err)
		return
	}

	hasMore := len(agents) > limit
	if hasMore {
		agents = agents[:limit]
	}

	data := make([]agentResponse, len(agents))
	for i, a := range agents {
		resp, err := toAgentResponse(a)
		if err != nil {
			handlers.RespondError(c, err)
			return
		}
		data[i] = resp
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

	c.JSON(http.StatusOK, agentListResponse{
		Data: data,
		Pagination: handlers.Pagination{
			Limit:      limit,
			Cursor:     cursorPtr,
			NextCursor: nextCursor,
			HasMore:    hasMore,
		},
	})
}

// Get handles GET /v1/api/agents/:name.
func (h *Handler) Get(c *gin.Context) {
	name := c.Param("name")

	agent, err := h.svc.Get(c.Request.Context(), name)
	if err != nil {
		handlers.RespondError(c, err)
		return
	}

	resp, err := toAgentResponse(agent)
	if err != nil {
		handlers.RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// Update handles PUT /v1/api/agents/:name.
func (h *Handler) Update(c *gin.Context) {
	name := c.Param("name")

	var req agentUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handlers.RespondValidationError(c, "The request body contains invalid fields", nil)
		return
	}

	// If name is provided in body, it must match the path parameter.
	if req.Name != "" && req.Name != name {
		handlers.RespondValidationError(c, "Name in body must match path parameter", []handlers.FieldError{
			{Field: "name", Code: "INVALID_VALUE", Message: "name in body must match path parameter or be omitted"},
		})
		return
	}

	if req.AllowedTools == nil {
		req.AllowedTools = []string{}
	}

	agent, err := h.svc.Update(c.Request.Context(), name, agentsvc.UpdateInput{
		Description:  req.Description,
		SystemPrompt: req.SystemPrompt,
		Model:        req.Model,
		AllowedTools: req.AllowedTools,
		Version:      req.Version,
	})
	if err != nil {
		handlers.RespondError(c, err)
		return
	}

	resp, err := toAgentResponse(agent)
	if err != nil {
		handlers.RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// Delete handles DELETE /v1/api/agents/:name.
func (h *Handler) Delete(c *gin.Context) {
	name := c.Param("name")

	if err := h.svc.Delete(c.Request.Context(), name); err != nil {
		handlers.RespondError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
