package agent

import (
	"slices"
	"time"
)

// Agent represents an agent definition stored in the database.
type Agent struct {
	// Name is the unique identifier for the agent (lowercase-with-hyphens format).
	// Examples: go-software-agent, data-architect, api-gateway
	Name string `db:"name"`

	// Description is a human-readable description of the agent's purpose.
	Description string `db:"description"`

	// SystemPrompt is the full system prompt provided to the LLM (up to 50KB).
	SystemPrompt string `db:"system_prompt"`

	// Model is the Claude model preference: sonnet, opus, haiku, or inherit from caller.
	Model string `db:"model"`

	// AllowedTools is a list of MCP tool names this agent can use.
	// Stored as JSONB in the database and unmarshaled during retrieval.
	AllowedTools []string `db:"-"`

	// RoutingKeywords are denormalized keywords for fast routing lookups.
	// Stored as JSONB in the database and unmarshaled during retrieval.
	RoutingKeywords []string `db:"-"`

	// CreatedAt is the timestamp when the agent was created.
	CreatedAt time.Time `db:"created_at"`

	// UpdatedAt is the timestamp when the agent was last modified.
	UpdatedAt time.Time `db:"updated_at"`
}

// ValidModels defines the valid values for the Model field.
var ValidModels = []string{"sonnet", "opus", "haiku", "inherit"}

// IsValidModel checks if the given model string is a valid model option.
func IsValidModel(model string) bool {
	return slices.Contains(ValidModels, model)
}
