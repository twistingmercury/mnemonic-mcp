package enrichmentjob

import (
	"slices"
	"time"

	"github.com/google/uuid"
)

// EnrichmentJob represents a background enrichment task stored in the database.
type EnrichmentJob struct {
	// ID is the unique identifier for the job.
	ID uuid.UUID `db:"id"`

	// PatternID is the ID of the pattern being enriched.
	PatternID uuid.UUID `db:"pattern_id"`

	// Status is the job processing state.
	// Valid values: pending, processing, completed, failed
	Status string `db:"status"`

	// Attempts is the number of times processing has been attempted.
	Attempts int `db:"attempts"`

	// MaxAttempts is the maximum number of retries before giving up.
	MaxAttempts int `db:"max_attempts"`

	// LastError is the error message from the most recent failed attempt.
	LastError *string `db:"last_error"`

	// ScheduledFor is when the job should be processed (supports delayed retry).
	ScheduledFor time.Time `db:"scheduled_for"`

	// StartedAt is the timestamp when processing began.
	StartedAt *time.Time `db:"started_at"`

	// CompletedAt is the timestamp when processing finished successfully.
	CompletedAt *time.Time `db:"completed_at"`

	// CreatedAt is the timestamp when the job was created.
	CreatedAt time.Time `db:"created_at"`

	// UpdatedAt is the timestamp when the job was last modified.
	UpdatedAt time.Time `db:"updated_at"`
}

// JobStatus represents the valid job status values.
type JobStatus string

const (
	// StatusPending indicates the job is awaiting processing.
	StatusPending JobStatus = "pending"

	// StatusProcessing indicates the job is currently being processed by a worker.
	StatusProcessing JobStatus = "processing"

	// StatusCompleted indicates the job finished successfully.
	StatusCompleted JobStatus = "completed"

	// StatusFailed indicates processing failed (see LastError).
	StatusFailed JobStatus = "failed"
)

// ValidStatuses defines the valid values for the Status field.
var ValidStatuses = []string{
	string(StatusPending),
	string(StatusProcessing),
	string(StatusCompleted),
	string(StatusFailed),
}

// IsValidStatus checks if the given status string is valid.
func IsValidStatus(status string) bool {
	return slices.Contains(ValidStatuses, status)
}

// IsPending returns true if the job is in pending state.
func (j *EnrichmentJob) IsPending() bool {
	return j.Status == string(StatusPending)
}

// IsProcessing returns true if the job is currently being processed.
func (j *EnrichmentJob) IsProcessing() bool {
	return j.Status == string(StatusProcessing)
}

// IsCompleted returns true if the job completed successfully.
func (j *EnrichmentJob) IsCompleted() bool {
	return j.Status == string(StatusCompleted)
}

// IsFailed returns true if the job failed.
func (j *EnrichmentJob) IsFailed() bool {
	return j.Status == string(StatusFailed)
}

// CanRetry returns true if the job can be retried.
func (j *EnrichmentJob) CanRetry() bool {
	return j.Attempts < j.MaxAttempts
}

// DefaultMaxAttempts is the default maximum number of attempts for a job.
const DefaultMaxAttempts = 3

// JobFilter defines filtering options for job queries.
type JobFilter struct {
	// Status filters by job status (pending, processing, completed, failed).
	Status *string

	// PatternID filters by the associated pattern ID.
	PatternID *uuid.UUID
}
