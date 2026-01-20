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
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	SystemPrompt    string   `json:"system_prompt"`
	Model           string   `json:"model"`
	AllowedTools    []string `json:"allowed_tools,omitempty"`
	RoutingKeywords []string `json:"routing_keywords,omitempty"`
	CreatedAt       string   `json:"created_at,omitempty"`
	UpdatedAt       string   `json:"updated_at,omitempty"`
}

// AgentCreate represents request body for creating a new agent.
// OpenAPI: api/openapi/mnemonic-v1.yaml:390 (AgentCreate)
type AgentCreate struct {
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	SystemPrompt    string   `json:"system_prompt"`
	Model           string   `json:"model"`
	AllowedTools    []string `json:"allowed_tools,omitempty"`
	RoutingKeywords []string `json:"routing_keywords,omitempty"`
}

// AgentUpdate represents request body for updating an agent.
// OpenAPI: api/openapi/mnemonic-v1.yaml:447 (AgentUpdate)
type AgentUpdate struct {
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	SystemPrompt    string   `json:"system_prompt"`
	Model           string   `json:"model"`
	AllowedTools    []string `json:"allowed_tools,omitempty"`
	RoutingKeywords []string `json:"routing_keywords,omitempty"`
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
	AgentName string  `json:"agent_name"`
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

// KeywordMatchConfig represents configuration for keyword-based matching.
// OpenAPI: api/openapi/mnemonic-v1.yaml:964 (KeywordMatchConfig)
type KeywordMatchConfig struct {
	Keywords  []string `json:"keywords"`
	MatchMode string   `json:"match_mode"`
}

// RegexMatchConfig represents configuration for regex-based matching.
// OpenAPI: api/openapi/mnemonic-v1.yaml:993 (RegexMatchConfig)
type RegexMatchConfig struct {
	Pattern string `json:"pattern"`
	Flags   string `json:"flags,omitempty"`
}

// PatternMatchConfig represents configuration for semantic pattern matching.
// OpenAPI: api/openapi/mnemonic-v1.yaml:1010 (PatternMatchConfig)
type PatternMatchConfig struct {
	PatternIDs []string `json:"pattern_ids"`
}

// RoutingRule represents routing rule definition.
// OpenAPI: api/openapi/mnemonic-v1.yaml:1028 (RoutingRule)
type RoutingRule struct {
	ID          string      `json:"id,omitempty"`
	Name        string      `json:"name"`
	Priority    int         `json:"priority"`
	AgentName   string      `json:"agent_name"`
	MatchType   string      `json:"match_type"`
	MatchConfig interface{} `json:"match_config"`
	Enabled     bool        `json:"enabled"`
	CreatedAt   string      `json:"created_at,omitempty"`
	UpdatedAt   string      `json:"updated_at,omitempty"`
}

// RoutingRuleCreate represents request body for creating a routing rule.
// OpenAPI: api/openapi/mnemonic-v1.yaml:1094 (RoutingRuleCreate)
type RoutingRuleCreate struct {
	Name        string      `json:"name"`
	Priority    int         `json:"priority"`
	AgentName   string      `json:"agent_name"`
	MatchType   string      `json:"match_type"`
	MatchConfig interface{} `json:"match_config"`
	Enabled     bool        `json:"enabled"`
}

// RoutingRuleUpdate represents request body for updating a routing rule.
// OpenAPI: api/openapi/mnemonic-v1.yaml:1132 (RoutingRuleUpdate)
type RoutingRuleUpdate struct {
	Name        string      `json:"name"`
	Priority    int         `json:"priority"`
	AgentName   string      `json:"agent_name"`
	MatchType   string      `json:"match_type"`
	MatchConfig interface{} `json:"match_config"`
	Enabled     bool        `json:"enabled"`
}

// RoutingRuleList represents paginated list of routing rules.
// OpenAPI: api/openapi/mnemonic-v1.yaml:1163 (RoutingRuleList)
type RoutingRuleList struct {
	Data       []RoutingRule `json:"data"`
	Pagination Pagination    `json:"pagination"`
}

// RouteContext represents optional context for routing decisions.
// OpenAPI: api/openapi/mnemonic-v1.yaml:767 (RouteContext)
type RouteContext struct {
	WorkingDirectory string   `json:"working_directory,omitempty"`
	FileTypes        []string `json:"file_types,omitempty"`
	RecentAgents     []string `json:"recent_agents,omitempty"`
}

// RouteOptions represents options for routing behavior.
// OpenAPI: api/openapi/mnemonic-v1.yaml:792 (RouteOptions)
type RouteOptions struct {
	IncludePatterns           bool    `json:"include_patterns,omitempty"`
	MaxPatterns               int     `json:"max_patterns,omitempty"`
	PatternRelevanceThreshold float64 `json:"pattern_relevance_threshold,omitempty"`
}

// RouteRequest represents request body for routing a prompt.
// OpenAPI: api/openapi/mnemonic-v1.yaml:814 (RouteRequest)
type RouteRequest struct {
	Prompt  string        `json:"prompt"`
	Context *RouteContext `json:"context,omitempty"`
	Options *RouteOptions `json:"options,omitempty"`
}

// RoutingDecision represents routing decision details.
// OpenAPI: api/openapi/mnemonic-v1.yaml:832 (RoutingDecision)
type RoutingDecision struct {
	AgentName       string   `json:"agent_name"`
	Confidence      float64  `json:"confidence"`
	Method          string   `json:"method"`
	MatchedKeywords []string `json:"matched_keywords,omitempty"`
	Reasoning       string   `json:"reasoning"`
}

// RoutePatternResult represents pattern included in routing response.
// OpenAPI: api/openapi/mnemonic-v1.yaml:878 (RoutePatternResult)
type RoutePatternResult struct {
	Name           string   `json:"name"`
	Content        string   `json:"content"`
	RelevanceScore float64  `json:"relevance_score"`
	Tags           []string `json:"tags,omitempty"`
}

// RouteMetadata represents timing and debugging information.
// OpenAPI: api/openapi/mnemonic-v1.yaml:910 (RouteMetadata)
type RouteMetadata struct {
	RoutingDurationMs          int `json:"routing_duration_ms,omitempty"`
	PatternRetrievalDurationMs int `json:"pattern_retrieval_duration_ms,omitempty"`
	TotalPatternsConsidered    int `json:"total_patterns_considered,omitempty"`
}

// RouteResponse represents complete routing response.
// OpenAPI: api/openapi/mnemonic-v1.yaml:930 (RouteResponse)
type RouteResponse struct {
	Routing  RoutingDecision      `json:"routing"`
	Agent    Agent                `json:"agent"`
	Patterns []RoutePatternResult `json:"patterns,omitempty"`
	Metadata *RouteMetadata       `json:"metadata,omitempty"`
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
