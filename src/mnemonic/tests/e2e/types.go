// Package e2e provides end-to-end tests for the Mnemonic API.
package e2e

// ErrorResponse represents RFC 7807 Problem Details error response.
// OpenAPI: api/openapi/mnemonic-v1.yaml:152 (ErrorResponse)
type ErrorResponse struct {
	Type     string       `json:"type"`
	Title    string       `json:"title"`
	Status   int          `json:"status"`
	Detail   string       `json:"detail,omitempty"`
	Instance string       `json:"instance,omitempty"`
	TraceID  string       `json:"traceId"`
	Errors   []FieldError `json:"errors,omitempty"`
}

// FieldError represents individual field validation error.
// OpenAPI: api/openapi/mnemonic-v1.yaml:229 (FieldError)
type FieldError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Pagination represents cursor-based pagination metadata.
// OpenAPI: api/openapi/mnemonic-v1.yaml:265 (Pagination)
type Pagination struct {
	Limit      int    `json:"limit"`
	Cursor     string `json:"cursor"`
	NextCursor string `json:"next_cursor"`
	HasMore    bool   `json:"has_more"`
}

// Agent represents the full agent definition.
// OpenAPI: api/openapi/mnemonic-v1.yaml:315 (Agent)
type Agent struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	SystemPrompt string   `json:"system_prompt"`
	Model        string   `json:"model"`
	AllowedTools []string `json:"allowed_tools,omitempty"`
	CreatedAt    string   `json:"created_at,omitempty"`
	UpdatedAt    string   `json:"updated_at,omitempty"`
}

// AgentCreate represents request body for creating a new agent.
// OpenAPI: api/openapi/mnemonic-v1.yaml:390 (AgentCreate)
type AgentCreate struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	SystemPrompt string   `json:"system_prompt"`
	Model        string   `json:"model"`
	AllowedTools []string `json:"allowed_tools,omitempty"`
}

// AgentUpdate represents request body for updating an agent.
// OpenAPI: api/openapi/mnemonic-v1.yaml:447 (AgentUpdate)
type AgentUpdate struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	SystemPrompt string   `json:"system_prompt"`
	Model        string   `json:"model"`
	AllowedTools []string `json:"allowed_tools,omitempty"`
}

// AgentSummary represents agent summary without system_prompt.
// OpenAPI: api/openapi/mnemonic-v1.yaml:483 (AgentSummary)
type AgentSummary struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Model       string `json:"model"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// AgentList represents paginated list of agents.
// OpenAPI: api/openapi/mnemonic-v1.yaml:514 (AgentList)
type AgentList struct {
	Data       []AgentSummary `json:"data"`
	Pagination Pagination     `json:"pagination"`
}

// AgentAssociation represents association between pattern and agent.
// OpenAPI: api/openapi/mnemonic-v1.yaml:529 (AgentAssociation)
type AgentAssociation struct {
	AgentID   string  `json:"agent_id"`
	Relevance float64 `json:"relevance"`
}

// Pattern represents the full pattern definition.
// OpenAPI: api/openapi/mnemonic-v1.yaml:550 (Pattern)
type Pattern struct {
	ID                string             `json:"id,omitempty"`
	Name              string             `json:"name"`
	Description       string             `json:"description,omitempty"`
	Content           string             `json:"content"`
	Tags              []string           `json:"tags,omitempty"`
	AgentAssociations []AgentAssociation `json:"agent_associations,omitempty"`
	CreatedAt         string             `json:"created_at,omitempty"`
	UpdatedAt         string             `json:"updated_at,omitempty"`
	EnrichmentStatus  string             `json:"enrichment_status,omitempty"`
	EnrichmentError   string             `json:"enrichment_error,omitempty"`
	EnrichedAt        string             `json:"enriched_at,omitempty"`
}

// PatternCreate represents request body for creating a new pattern.
// OpenAPI: api/openapi/mnemonic-v1.yaml:644 (PatternCreate)
type PatternCreate struct {
	Name              string             `json:"name"`
	Description       string             `json:"description,omitempty"`
	Content           string             `json:"content"`
	Tags              []string           `json:"tags,omitempty"`
	AgentAssociations []AgentAssociation `json:"agent_associations,omitempty"`
}

// PatternUpdate represents request body for updating a pattern.
// OpenAPI: api/openapi/mnemonic-v1.yaml:683 (PatternUpdate)
type PatternUpdate struct {
	Name              string             `json:"name"`
	Description       string             `json:"description,omitempty"`
	Content           string             `json:"content"`
	Tags              []string           `json:"tags,omitempty"`
	AgentAssociations []AgentAssociation `json:"agent_associations,omitempty"`
}

// PatternSummary represents pattern summary without content.
// OpenAPI: api/openapi/mnemonic-v1.yaml:711 (PatternSummary)
type PatternSummary struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

// PatternList represents paginated list of patterns.
// OpenAPI: api/openapi/mnemonic-v1.yaml:752 (PatternList)
type PatternList struct {
	Data       []PatternSummary `json:"data"`
	Pagination Pagination       `json:"pagination"`
}

// HealthResponse represents health check response.
// OpenAPI: api/openapi/mnemonic-v1.yaml:1185 (HealthResponse)
type HealthResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
	Reason string            `json:"reason,omitempty"`
}

// VersionResponse represents version information response.
// OpenAPI: api/openapi/mnemonic-v1.yaml:1209 (VersionResponse)
type VersionResponse struct {
	Version     string `json:"version"`
	APIVersion  string `json:"api_version"`
	BuildCommit string `json:"build_commit,omitempty"`
	BuildTime   string `json:"build_time,omitempty"`
	GoVersion   string `json:"go_version,omitempty"`
}
