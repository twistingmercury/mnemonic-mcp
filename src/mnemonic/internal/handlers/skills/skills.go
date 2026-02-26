package skills

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/twistingmercury/mnemonic/internal/handlers"
	skillrepo "github.com/twistingmercury/mnemonic/internal/repository/skill"
	skillsvc "github.com/twistingmercury/mnemonic/internal/service/skill"
)

// Handler provides HTTP handlers for skill CRUD operations.
type Handler struct {
	svc skillsvc.Service
}

// New creates a new skill Handler backed by the given service.
func New(svc skillsvc.Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes binds skill endpoints to the given router group.
// The group should be mounted at /v1/api.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/skills", h.Create)
	rg.GET("/skills", h.List)
	rg.GET("/skills/:name", h.Get)
	rg.PUT("/skills/:name", h.Update)
	rg.DELETE("/skills/:name", h.Delete)
}

// --- Request/Response Types ---

type skillCreateRequest struct {
	Name          string            `json:"name" binding:"required"`
	Description   string            `json:"description" binding:"required"`
	Content       string            `json:"content" binding:"required"`
	Tags          []string          `json:"tags"`
	License       *string           `json:"license"`
	Compatibility *string           `json:"compatibility"`
	Metadata      map[string]string `json:"metadata"`
	AllowedTools  []string          `json:"allowed_tools"`
	Version       string            `json:"version" binding:"required"`
}

type skillUpdateRequest struct {
	Name          string            `json:"name"`
	Description   string            `json:"description" binding:"required"`
	Content       string            `json:"content" binding:"required"`
	Tags          []string          `json:"tags"`
	License       *string           `json:"license"`
	Compatibility *string           `json:"compatibility"`
	Metadata      map[string]string `json:"metadata"`
	AllowedTools  []string          `json:"allowed_tools"`
	Version       string            `json:"version" binding:"required"`
}

type fileCounts struct {
	ScriptsCount    int `json:"scripts_count"`
	ReferencesCount int `json:"references_count"`
	AssetsCount     int `json:"assets_count"`
}

type skillResponse struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	Content       string            `json:"content"`
	Tags          []string          `json:"tags"`
	License       *string           `json:"license"`
	Compatibility *string           `json:"compatibility"`
	Metadata      map[string]string `json:"metadata"`
	AllowedTools  []string          `json:"allowed_tools"`
	Version       string            `json:"version"`
	Files         fileCounts        `json:"files"`
	CRC64         string            `json:"crc64"`
	CreatedAt     string            `json:"created_at"`
	UpdatedAt     string            `json:"updated_at"`
}

type skillListResponse struct {
	Data       []skillResponse     `json:"data"`
	Pagination handlers.Pagination `json:"pagination"`
}

// skillDefinition mirrors the JSONB definition structure for unmarshaling.
type skillDefinition struct {
	Description   string            `json:"description"`
	Content       string            `json:"content"`
	Tags          []string          `json:"tags,omitempty"`
	License       *string           `json:"license,omitempty"`
	Compatibility *string           `json:"compatibility,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	AllowedTools  []string          `json:"allowed_tools,omitempty"`
	Version       string            `json:"version"`
}

func toSkillResponse(s *skillrepo.Skill) skillResponse {
	var def skillDefinition
	_ = json.Unmarshal(s.Definition, &def)

	tags := def.Tags
	if tags == nil {
		tags = []string{}
	}
	allowedTools := def.AllowedTools
	if allowedTools == nil {
		allowedTools = []string{}
	}

	return skillResponse{
		ID:            s.ID.String(),
		Name:          s.Name,
		Description:   def.Description,
		Content:       def.Content,
		Tags:          tags,
		License:       def.License,
		Compatibility: def.Compatibility,
		Metadata:      def.Metadata,
		AllowedTools:  allowedTools,
		Version:       def.Version,
		Files: fileCounts{
			ScriptsCount:    0,
			ReferencesCount: 0,
			AssetsCount:     0,
		},
		CRC64:     s.CRC64,
		CreatedAt: s.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt: s.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

// --- Handlers ---

// Create handles POST /v1/api/skills.
func (h *Handler) Create(c *gin.Context) {
	var req skillCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handlers.RespondValidationError(c, "The request body contains invalid fields", nil)
		return
	}

	if req.Tags == nil {
		req.Tags = []string{}
	}
	if req.AllowedTools == nil {
		req.AllowedTools = []string{}
	}

	skill, err := h.svc.Create(c.Request.Context(), skillsvc.CreateInput{
		Name:          req.Name,
		Description:   req.Description,
		Content:       req.Content,
		Tags:          req.Tags,
		License:       req.License,
		Compatibility: req.Compatibility,
		Metadata:      req.Metadata,
		AllowedTools:  req.AllowedTools,
		Version:       req.Version,
	})
	if err != nil {
		handlers.RespondError(c, err)
		return
	}

	resp := toSkillResponse(skill)
	c.Header("Location", fmt.Sprintf("/v1/api/skills/%s", skill.Name))
	c.JSON(http.StatusCreated, resp)
}

// List handles GET /v1/api/skills.
func (h *Handler) List(c *gin.Context) {
	limit := handlers.ParseIntQuery(c, "limit", 100, 1, 200)
	offset := handlers.DecodeCursor(c.Query("cursor"))

	// The tags filter is documented in the OpenAPI spec but the skill service
	// List method does not currently support tag filtering. For MVP we fetch
	// all and let the caller filter. The tags query param is accepted but
	// not applied to the query.
	_ = c.Query("tags")

	skills, _, err := h.svc.List(c.Request.Context(), skillsvc.ListOptions{
		Offset: offset,
		Limit:  limit + 1,
	})
	if err != nil {
		handlers.RespondError(c, err)
		return
	}

	hasMore := len(skills) > limit
	if hasMore {
		skills = skills[:limit]
	}

	// Post-filter by tags if provided.
	if rawTags := c.Query("tags"); rawTags != "" {
		requestedTags := strings.Split(rawTags, ",")
		skills = filterSkillsByTags(skills, requestedTags)
	}

	data := make([]skillResponse, len(skills))
	for i, s := range skills {
		data[i] = toSkillResponse(s)
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

	c.JSON(http.StatusOK, skillListResponse{
		Data: data,
		Pagination: handlers.Pagination{
			Limit:      limit,
			Cursor:     cursorPtr,
			NextCursor: nextCursor,
			HasMore:    hasMore,
		},
	})
}

// Get handles GET /v1/api/skills/:name.
func (h *Handler) Get(c *gin.Context) {
	name := c.Param("name")

	skill, err := h.svc.GetByName(c.Request.Context(), name)
	if err != nil {
		handlers.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, toSkillResponse(skill))
}

// Update handles PUT /v1/api/skills/:name.
func (h *Handler) Update(c *gin.Context) {
	name := c.Param("name")

	var req skillUpdateRequest
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

	if req.Tags == nil {
		req.Tags = []string{}
	}
	if req.AllowedTools == nil {
		req.AllowedTools = []string{}
	}

	skill, err := h.svc.Update(c.Request.Context(), name, skillsvc.UpdateInput{
		Description:   req.Description,
		Content:       req.Content,
		Tags:          req.Tags,
		License:       req.License,
		Compatibility: req.Compatibility,
		Metadata:      req.Metadata,
		AllowedTools:  req.AllowedTools,
		Version:       req.Version,
	})
	if err != nil {
		handlers.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, toSkillResponse(skill))
}

// Delete handles DELETE /v1/api/skills/:name.
func (h *Handler) Delete(c *gin.Context) {
	name := c.Param("name")

	if err := h.svc.Delete(c.Request.Context(), name); err != nil {
		handlers.RespondError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// filterSkillsByTags filters skills in-memory by checking if their definition
// tags contain any of the requested tags.
func filterSkillsByTags(skills []*skillrepo.Skill, requestedTags []string) []*skillrepo.Skill {
	if len(requestedTags) == 0 {
		return skills
	}
	tagSet := make(map[string]bool, len(requestedTags))
	for _, t := range requestedTags {
		tagSet[strings.TrimSpace(t)] = true
	}

	var filtered []*skillrepo.Skill
	for _, s := range skills {
		var def skillDefinition
		if err := json.Unmarshal(s.Definition, &def); err != nil {
			continue
		}
		for _, t := range def.Tags {
			if tagSet[t] {
				filtered = append(filtered, s)
				break
			}
		}
	}
	return filtered
}
