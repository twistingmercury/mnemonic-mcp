package pattern

import (
	"slices"
	"time"

	"github.com/google/uuid"
)

// Pattern represents a context pattern stored in the database.
type Pattern struct {
	// ID is the unique UUID identifier for the pattern.
	ID uuid.UUID `db:"id"`

	// Name is the unique human-readable name for the pattern (e.g., "go-error-handling").
	Name string `db:"name"`

	// Description is an optional description of when/how to use this pattern.
	Description *string `db:"description"`

	// Content is the actual context text that will be injected into prompts (up to 10KB).
	Content string `db:"content"`

	// Tags is a list of categorization tags (e.g., ["golang", "best-practices"]).
	// Stored as JSONB in the database and unmarshaled during retrieval.
	Tags []string `db:"-"`

	// Embedding is the vector embedding (1536 dimensions) for semantic similarity search.
	// Nil until enrichment completes.
	Embedding []float32 `db:"-"`

	// EnrichmentStatus is the processing state: "pending", "enriched", or "failed".
	EnrichmentStatus string `db:"enrichment_status"`

	// EnrichmentError contains the error message if enrichment failed (nil otherwise).
	EnrichmentError *string `db:"enrichment_error"`

	// EnrichedAt is the timestamp when enrichment completed successfully (nil if pending or failed).
	EnrichedAt *time.Time `db:"enriched_at"`

	// CreatedAt is the timestamp when the pattern was created.
	CreatedAt time.Time `db:"created_at"`

	// UpdatedAt is the timestamp when the pattern was last modified.
	UpdatedAt time.Time `db:"updated_at"`
}

// ValidEnrichmentStatuses defines the valid values for the EnrichmentStatus field.
var ValidEnrichmentStatuses = []string{"pending", "enriched", "failed"}

// IsValidEnrichmentStatus checks if the given status string is a valid enrichment status.
func IsValidEnrichmentStatus(status string) bool {
	return slices.Contains(ValidEnrichmentStatuses, status)
}

// Filter defines filtering options for pattern queries.
type Filter struct {
	// Tags filters patterns that have any of these tags.
	Tags []string

	// EnrichmentStatus filters patterns by enrichment status.
	EnrichmentStatus string

	// SearchQuery performs full-text search in name/description.
	SearchQuery string
}

// SimilarityOptions defines options for similarity search.
type SimilarityOptions struct {
	// MinSimilarity is the minimum similarity threshold (0.0-1.0).
	// Only patterns with similarity >= MinSimilarity are returned.
	MinSimilarity float64

	// MaxResults is the maximum number of results to return.
	MaxResults int

	// Tags optionally filters results by tag.
	Tags []string
}

// Match represents a similarity search result.
type Match struct {
	// Pattern is the matched pattern.
	Pattern *Pattern

	// Similarity is the cosine similarity score (0.0-1.0, where 1.0 is identical).
	Similarity float64
}

// AgentAssociation represents a pattern-agent relationship with relevance score.
type AgentAssociation struct {
	// AgentName is the name of the associated agent.
	AgentName string `db:"agent_name"`

	// Relevance is the relevance score from 0.0 (minimally relevant) to 1.0 (highly relevant).
	Relevance float64 `db:"relevance"`
}
