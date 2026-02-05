package enrichmentjob

import "errors"

// Common repository errors for enrichment job operations.
var (
	// ErrJobNotFound is returned when an enrichment job with the specified ID cannot be found.
	ErrJobNotFound = errors.New("enrichment job not found")

	// ErrPatternNotFound is returned when the referenced pattern does not exist.
	ErrPatternNotFound = errors.New("pattern not found")

	// ErrInvalidStatus is returned when an invalid status transition is attempted.
	ErrInvalidStatus = errors.New("invalid job status")

	// ErrNoPendingJobs is returned when ClaimPending finds no available jobs.
	ErrNoPendingJobs = errors.New("no pending jobs available")
)
