package pattern

import "errors"

// Common repository errors for pattern operations.
var (
	// ErrPatternNotFound is returned when a pattern with the specified ID or name cannot be found.
	ErrPatternNotFound = errors.New("pattern not found")

	// ErrPatternNameExists is returned when attempting to create a pattern with a name that already exists.
	ErrPatternNameExists = errors.New("pattern name already exists")

	// ErrAgentNotFound is returned when one or more agent names do not exist in the agents table.
	ErrAgentNotFound = errors.New("agent not found")
)
