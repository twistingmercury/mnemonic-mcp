package enrichmentjob

import "errors"

// Common repository errors for enrichment job operations.
var (
	// ErrNotFound is returned when an enrichment job with the specified ID cannot be found.
	ErrNotFound = errors.New("enrichment job not found")

	// ErrPatternNotFound is returned when the referenced pattern does not exist.
	ErrPatternNotFound = errors.New("pattern not found")

	// ErrInvalidStatus is returned when an invalid status transition is attempted.
	ErrInvalidStatus = errors.New("invalid job status")

	// ErrNoPending is returned when ClaimPending finds no available jobs.
	ErrNoPending = errors.New("no pending jobs available")

	// ErrJobAlreadyPending is returned when a pending enrichment job already exists for this pattern.
	ErrJobAlreadyPending = errors.New("enrichment job already pending for this pattern")
)
