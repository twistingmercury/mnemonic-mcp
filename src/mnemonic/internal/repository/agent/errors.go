package agent

import "errors"

// Common repository errors for agent operations.
var (
	// ErrExists is returned when attempting to create an agent with a name that already exists.
	ErrExists = errors.New("agent already exists")

	// ErrNotFound is returned when an agent with the specified name cannot be found.
	ErrNotFound = errors.New("agent not found")

	// ErrInUse is returned when attempting to delete an agent that is referenced by routing rules.
	ErrInUse = errors.New("agent is referenced by routing rules")
)
