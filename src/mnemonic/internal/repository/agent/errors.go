package agent

import "errors"

// Common repository errors for agent operations.
var (
	// ErrAgentExists is returned when attempting to create an agent with a name that already exists.
	ErrAgentExists = errors.New("agent already exists")

	// ErrAgentNotFound is returned when an agent with the specified name cannot be found.
	ErrAgentNotFound = errors.New("agent not found")

	// ErrAgentInUse is returned when attempting to delete an agent that is referenced by routing rules.
	ErrAgentInUse = errors.New("agent is referenced by routing rules")
)
