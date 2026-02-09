package routingrule

import "errors"

// Common repository errors for routing rule operations.
var (
	// ErrNotFound is returned when a routing rule with the specified ID or name cannot be found.
	ErrNotFound = errors.New("routing rule not found")

	// ErrNameExists is returned when attempting to create a rule with a name that already exists.
	ErrNameExists = errors.New("routing rule name already exists")

	// ErrInvalidMatchType is returned when an invalid match type is provided.
	ErrInvalidMatchType = errors.New("invalid match type")

	// ErrInvalidMatchConfig is returned when match_config does not match the match_type requirements.
	ErrInvalidMatchConfig = errors.New("invalid match config for match type")

	// ErrAgentNotFound is returned when the target agent does not exist.
	ErrAgentNotFound = errors.New("target agent not found")
)
