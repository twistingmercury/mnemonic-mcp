package pattern

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pgvector/pgvector-go"
	"github.com/twistingmercury/mnemonic/internal/repository"
)

// Repository defines data access operations for patterns.
type Repository interface {
	// Create stores a new pattern. Returns ErrNameExists if name already exists.
	Create(ctx context.Context, pattern *Pattern) error

	// Get retrieves a pattern by ID. Returns ErrNotFound if not found.
	Get(ctx context.Context, id uuid.UUID) (*Pattern, error)

	// GetByName retrieves a pattern by name. Returns ErrNotFound if not found.
	GetByName(ctx context.Context, name string) (*Pattern, error)

	// Update modifies an existing pattern. Returns ErrNotFound if not found.
	Update(ctx context.Context, pattern *Pattern) error

	// Delete removes a pattern by ID. Returns ErrNotFound if not found.
	Delete(ctx context.Context, id uuid.UUID) error

	// List retrieves patterns with filtering and pagination.
	// Returns the patterns, total count, and any error.
	List(ctx context.Context, filter Filter, opts repository.ListOptions) ([]*Pattern, int64, error)

	// UpdateEmbedding stores the embedding vector for a pattern.
	UpdateEmbedding(ctx context.Context, id uuid.UUID, embedding []float32) error

	// UpdateEnrichmentStatus updates the enrichment state of a pattern.
	UpdateEnrichmentStatus(ctx context.Context, id uuid.UUID, status string, errMsg *string) error

	// FindSimilar finds patterns similar to the given embedding vector.
	FindSimilar(ctx context.Context, embedding []float32, opts SimilarityOptions) ([]*Match, error)

	// SetAgentAssociations replaces all agent associations for a pattern.
	SetAgentAssociations(ctx context.Context, patternID uuid.UUID, associations []AgentAssociation) error

	// GetAgentAssociations retrieves all agent associations for a pattern.
	GetAgentAssociations(ctx context.Context, patternID uuid.UUID) ([]AgentAssociation, error)

	// GetPatternIDsByAgent returns all pattern IDs associated with the given agent.
	// Used by SearchService for agent-scoped similarity search pre-filtering.
	GetPatternIDsByAgent(ctx context.Context, agentID uuid.UUID) ([]uuid.UUID, error)

	// Exists checks if a pattern with the given ID exists.
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
}

// pgxRepository is a PostgreSQL implementation of Repository using pgx.
type pgxRepository struct {
	db repository.DBTX
}

// NewRepository creates a new PostgreSQL-backed Repository.
func NewRepository(db repository.DBTX) Repository {
	return &pgxRepository{db: db}
}

// Create stores a new pattern in the database.
func (r *pgxRepository) Create(ctx context.Context, pattern *Pattern) error {
	// Defensive check: ensure Tags is not nil (database requires JSON array, not null)
	if pattern.Tags == nil {
		pattern.Tags = []string{}
	}

	tagsJSON, err := json.Marshal(pattern.Tags)
	if err != nil {
		return fmt.Errorf("marshaling tags: %w", err)
	}

	// Generate UUID if not set
	if pattern.ID == uuid.Nil {
		pattern.ID = uuid.New()
	}

	now := time.Now()
	query := `
		INSERT INTO patterns (
			id, name, description, content, tags,
			enrichment_status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err = r.db.Exec(ctx, query,
		pattern.ID,
		pattern.Name,
		pattern.Description,
		pattern.Content,
		tagsJSON,
		"pending", // New patterns start as pending
		now,
		now,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == repository.PgErrCodeUniqueViolation {
			return ErrNameExists
		}
		return fmt.Errorf("creating pattern: %w", err)
	}

	pattern.EnrichmentStatus = "pending"
	pattern.CreatedAt = now
	pattern.UpdatedAt = now
	return nil
}

// Get retrieves a pattern by ID from the database.
func (r *pgxRepository) Get(ctx context.Context, id uuid.UUID) (*Pattern, error) {
	query := `
		SELECT id, name, description, content, tags, embedding,
			   enrichment_status, enrichment_error, enriched_at,
			   created_at, updated_at
		FROM patterns
		WHERE id = $1
	`

	return r.scanPattern(ctx, query, id)
}

// GetByName retrieves a pattern by name from the database.
func (r *pgxRepository) GetByName(ctx context.Context, name string) (*Pattern, error) {
	query := `
		SELECT id, name, description, content, tags, embedding,
			   enrichment_status, enrichment_error, enriched_at,
			   created_at, updated_at
		FROM patterns
		WHERE name = $1
	`

	return r.scanPattern(ctx, query, name)
}

// scanPattern is a helper that executes a query and scans the result into a Pattern.
func (r *pgxRepository) scanPattern(ctx context.Context, query string, arg any) (*Pattern, error) {
	var pattern Pattern
	var tagsJSON []byte
	var embedding *pgvector.Vector

	err := r.db.QueryRow(ctx, query, arg).Scan(
		&pattern.ID,
		&pattern.Name,
		&pattern.Description,
		&pattern.Content,
		&tagsJSON,
		&embedding,
		&pattern.EnrichmentStatus,
		&pattern.EnrichmentError,
		&pattern.EnrichedAt,
		&pattern.CreatedAt,
		&pattern.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("getting pattern: %w", err)
	}

	if err := json.Unmarshal(tagsJSON, &pattern.Tags); err != nil {
		return nil, fmt.Errorf("unmarshaling tags: %w", err)
	}

	// Convert pgvector.Vector to []float32 if embedding exists
	if embedding != nil {
		pattern.Embedding = embedding.Slice()
	}

	return &pattern, nil
}

// Update modifies an existing pattern in the database.
// Per design spec, updating a pattern resets enrichment status to "pending"
// and clears the embedding, ensuring stale embeddings don't remain active.
func (r *pgxRepository) Update(ctx context.Context, pattern *Pattern) error {
	tagsJSON, err := json.Marshal(pattern.Tags)
	if err != nil {
		return fmt.Errorf("marshaling tags: %w", err)
	}

	now := time.Now()
	query := `
		UPDATE patterns SET
			name = $2,
			description = $3,
			content = $4,
			tags = $5,
			enrichment_status = 'pending',
			embedding = NULL,
			enrichment_error = NULL,
			enriched_at = NULL,
			updated_at = $6
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query,
		pattern.ID,
		pattern.Name,
		pattern.Description,
		pattern.Content,
		tagsJSON,
		now,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == repository.PgErrCodeUniqueViolation {
			return ErrNameExists
		}
		return fmt.Errorf("updating pattern: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	// Update pattern struct to reflect the reset enrichment fields
	pattern.EnrichmentStatus = "pending"
	pattern.Embedding = nil
	pattern.EnrichmentError = nil
	pattern.EnrichedAt = nil
	pattern.UpdatedAt = now
	return nil
}

// Delete removes a pattern by ID from the database.
func (r *pgxRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM patterns WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("deleting pattern: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// List retrieves patterns with filtering and pagination.
func (r *pgxRepository) List(ctx context.Context, filter Filter, opts repository.ListOptions) ([]*Pattern, int64, error) {
	// Build query dynamically based on filter
	var whereConditions []string
	var args []any
	argNum := 1

	if filter.EnrichmentStatus != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("enrichment_status = $%d", argNum))
		args = append(args, filter.EnrichmentStatus)
		argNum++
	}

	if len(filter.Tags) > 0 {
		// Use ?| operator to check if tags contain any of the specified tags
		whereConditions = append(whereConditions, fmt.Sprintf("tags ?| $%d", argNum))
		args = append(args, filter.Tags)
		argNum++
	}

	if filter.SearchQuery != "" {
		// Full-text search on name and description
		whereConditions = append(whereConditions, fmt.Sprintf(
			"to_tsvector('english', name || ' ' || coalesce(description, '')) @@ plainto_tsquery('english', $%d)",
			argNum,
		))
		args = append(args, filter.SearchQuery)
		argNum++
	}

	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + strings.Join(whereConditions, " AND ")
	}

	// Build query with window function for total count
	query := fmt.Sprintf(`
		SELECT id, name, description, content, tags, embedding,
			   enrichment_status, enrichment_error, enriched_at,
			   created_at, updated_at,
			   COUNT(*) OVER() as total_count
		FROM patterns
		%s
		ORDER BY name ASC
	`, whereClause)

	// Add pagination
	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argNum)
		args = append(args, opts.Limit)
		argNum++
		if opts.Offset > 0 {
			query += fmt.Sprintf(" OFFSET $%d", argNum)
			args = append(args, opts.Offset)
		}
	} else if opts.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argNum)
		args = append(args, opts.Offset)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("listing patterns: %w", err)
	}
	defer rows.Close()

	patterns := make([]*Pattern, 0)
	var totalCount int64

	for rows.Next() {
		var pattern Pattern
		var tagsJSON []byte
		var embedding *pgvector.Vector

		err := rows.Scan(
			&pattern.ID,
			&pattern.Name,
			&pattern.Description,
			&pattern.Content,
			&tagsJSON,
			&embedding,
			&pattern.EnrichmentStatus,
			&pattern.EnrichmentError,
			&pattern.EnrichedAt,
			&pattern.CreatedAt,
			&pattern.UpdatedAt,
			&totalCount,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("listing patterns: scanning row: %w", err)
		}

		if err := json.Unmarshal(tagsJSON, &pattern.Tags); err != nil {
			return nil, 0, fmt.Errorf("unmarshaling tags: %w", err)
		}

		if embedding != nil {
			pattern.Embedding = embedding.Slice()
		}

		patterns = append(patterns, &pattern)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("listing patterns: iterating rows: %w", err)
	}

	return patterns, totalCount, nil
}

// UpdateEmbedding stores the embedding vector for a pattern.
func (r *pgxRepository) UpdateEmbedding(ctx context.Context, id uuid.UUID, embedding []float32) error {
	now := time.Now()
	query := `
		UPDATE patterns SET
			embedding = $2,
			updated_at = $3
		WHERE id = $1
	`

	vec := pgvector.NewVector(embedding)
	result, err := r.db.Exec(ctx, query, id, vec, now)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// UpdateEnrichmentStatus updates the enrichment state of a pattern.
func (r *pgxRepository) UpdateEnrichmentStatus(ctx context.Context, id uuid.UUID, status string, errMsg *string) error {
	if !IsValidEnrichmentStatus(status) {
		return fmt.Errorf("invalid enrichment status: %q", status)
	}

	now := time.Now()

	var enrichedAt *time.Time
	if status == "enriched" {
		enrichedAt = &now
	}

	query := `
		UPDATE patterns SET
			enrichment_status = $2,
			enrichment_error = $3,
			enriched_at = $4,
			updated_at = $5
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query, id, status, errMsg, enrichedAt, now)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// FindSimilar finds patterns similar to the given embedding vector.
func (r *pgxRepository) FindSimilar(ctx context.Context, embedding []float32, opts SimilarityOptions) ([]*Match, error) {
	vec := pgvector.NewVector(embedding)

	// Build query dynamically based on options
	var whereConditions []string
	var args []any
	argNum := 1

	// Always filter by enriched status and non-null embedding
	whereConditions = append(whereConditions,
		"enrichment_status = 'enriched'",
		"embedding IS NOT NULL",
	)

	// Add the embedding parameter
	args = append(args, vec)
	embeddingArgNum := argNum
	argNum++

	// Apply minimum similarity threshold if specified
	if opts.MinSimilarity > 0 {
		// Convert similarity threshold to distance threshold
		// similarity = 1 - distance, so distance = 1 - similarity
		whereConditions = append(whereConditions, fmt.Sprintf(
			"(embedding <=> $%d) < $%d",
			embeddingArgNum, argNum,
		))
		distanceThreshold := 1 - opts.MinSimilarity
		args = append(args, distanceThreshold)
		argNum++
	}

	// Apply tag filter if specified
	if len(opts.Tags) > 0 {
		whereConditions = append(whereConditions, fmt.Sprintf("tags ?| $%d", argNum))
		args = append(args, opts.Tags)
		argNum++
	}

	// Apply pattern ID filter if specified
	if len(opts.PatternIDs) > 0 {
		whereConditions = append(whereConditions, fmt.Sprintf("id = ANY($%d)", argNum))
		args = append(args, opts.PatternIDs)
		argNum++
	}

	whereClause := strings.Join(whereConditions, " AND ")

	query := fmt.Sprintf(`
		SELECT id, name, description, content, tags, embedding,
			   enrichment_status, enrichment_error, enriched_at,
			   created_at, updated_at,
			   1 - (embedding <=> $%d) AS similarity
		FROM patterns
		WHERE %s
		ORDER BY embedding <=> $%d
	`, embeddingArgNum, whereClause, embeddingArgNum)

	// Apply max results limit
	if opts.MaxResults > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argNum)
		args = append(args, opts.MaxResults)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("finding similar patterns: %w", err)
	}
	defer rows.Close()

	matches := make([]*Match, 0)

	for rows.Next() {
		var pattern Pattern
		var tagsJSON []byte
		var embeddingVec *pgvector.Vector
		var similarity float64

		err := rows.Scan(
			&pattern.ID,
			&pattern.Name,
			&pattern.Description,
			&pattern.Content,
			&tagsJSON,
			&embeddingVec,
			&pattern.EnrichmentStatus,
			&pattern.EnrichmentError,
			&pattern.EnrichedAt,
			&pattern.CreatedAt,
			&pattern.UpdatedAt,
			&similarity,
		)
		if err != nil {
			return nil, fmt.Errorf("finding similar patterns: scanning row: %w", err)
		}

		if err := json.Unmarshal(tagsJSON, &pattern.Tags); err != nil {
			return nil, fmt.Errorf("unmarshaling tags: %w", err)
		}

		if embeddingVec != nil {
			pattern.Embedding = embeddingVec.Slice()
		}

		matches = append(matches, &Match{
			Pattern:    &pattern,
			Similarity: similarity,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("finding similar patterns: iterating rows: %w", err)
	}

	return matches, nil
}

// SetAgentAssociations replaces all agent associations for a pattern.
// This operation is atomic: either all associations are replaced or none are.
func (r *pgxRepository) SetAgentAssociations(ctx context.Context, patternID uuid.UUID, associations []AgentAssociation) error {
	// First, verify the pattern exists
	exists, err := r.Exists(ctx, patternID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}

	// Check if we can start a transaction for atomicity
	txBeginner, canBeginTx := r.db.(interface {
		Begin(ctx context.Context) (pgx.Tx, error)
	})

	if canBeginTx {
		return r.setAgentAssociationsWithTx(ctx, txBeginner, patternID, associations)
	}

	// If already in a transaction (r.db is a pgx.Tx), operations are already atomic
	return r.setAgentAssociationsNoTx(ctx, patternID, associations)
}

// setAgentAssociationsWithTx performs the operation within a new transaction.
func (r *pgxRepository) setAgentAssociationsWithTx(ctx context.Context, txBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}, patternID uuid.UUID, associations []AgentAssociation) error {
	tx, err := txBeginner.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	// Validate agent IDs exist before modifying associations
	if len(associations) > 0 {
		agentIDs := make([]uuid.UUID, len(associations))
		for i, assoc := range associations {
			agentIDs[i] = assoc.AgentID
		}
		if err = r.validateAgentIDs(ctx, tx, agentIDs); err != nil {
			return err
		}
	}

	// Delete existing associations
	deleteQuery := `DELETE FROM pattern_agent_associations WHERE pattern_id = $1`
	_, err = tx.Exec(ctx, deleteQuery, patternID)
	if err != nil {
		return fmt.Errorf("deleting existing associations: %w", err)
	}

	// Insert new associations using batch for efficiency
	if len(associations) > 0 {
		err = r.insertAssociationsBatch(ctx, tx, patternID, associations)
		if err != nil {
			return err
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}

// setAgentAssociationsNoTx performs the operation assuming we're already in a transaction.
func (r *pgxRepository) setAgentAssociationsNoTx(ctx context.Context, patternID uuid.UUID, associations []AgentAssociation) error {
	// Validate agent IDs exist before modifying associations
	if len(associations) > 0 {
		agentIDs := make([]uuid.UUID, len(associations))
		for i, assoc := range associations {
			agentIDs[i] = assoc.AgentID
		}
		if err := r.validateAgentIDs(ctx, r.db, agentIDs); err != nil {
			return err
		}
	}

	// Delete existing associations
	deleteQuery := `DELETE FROM pattern_agent_associations WHERE pattern_id = $1`
	_, err := r.db.Exec(ctx, deleteQuery, patternID)
	if err != nil {
		return fmt.Errorf("deleting existing associations: %w", err)
	}

	// Insert new associations using batch for efficiency
	if len(associations) > 0 {
		return r.insertAssociationsBatch(ctx, r.db, patternID, associations)
	}

	return nil
}

// insertAssociationsBatch inserts multiple associations in a single query for atomicity and performance.
func (r *pgxRepository) insertAssociationsBatch(ctx context.Context, db repository.DBTX, patternID uuid.UUID, associations []AgentAssociation) error {
	if len(associations) == 0 {
		return nil
	}

	// Build a single INSERT with multiple VALUES for atomicity at the database level
	var valueStrings []string
	var args []any
	argNum := 1

	for _, assoc := range associations {
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d)", argNum, argNum+1, argNum+2))
		args = append(args, patternID, assoc.AgentID, assoc.Relevance)
		argNum += 3
	}

	query := fmt.Sprintf(`
		INSERT INTO pattern_agent_associations (pattern_id, agent_id, relevance)
		VALUES %s
	`, strings.Join(valueStrings, ", "))

	_, err := db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("inserting associations: %w", err)
	}

	return nil
}

// GetAgentAssociations retrieves all agent associations for a pattern.
func (r *pgxRepository) GetAgentAssociations(ctx context.Context, patternID uuid.UUID) ([]AgentAssociation, error) {
	query := `
		SELECT agent_id, relevance
		FROM pattern_agent_associations
		WHERE pattern_id = $1
		ORDER BY relevance DESC
	`

	rows, err := r.db.Query(ctx, query, patternID)
	if err != nil {
		return nil, fmt.Errorf("getting agent associations: %w", err)
	}
	defer rows.Close()

	associations := make([]AgentAssociation, 0)

	for rows.Next() {
		var assoc AgentAssociation
		err := rows.Scan(&assoc.AgentID, &assoc.Relevance)
		if err != nil {
			return nil, fmt.Errorf("getting agent associations: scanning row: %w", err)
		}
		associations = append(associations, assoc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("getting agent associations: iterating rows: %w", err)
	}

	return associations, nil
}

// GetPatternIDsByAgent returns all pattern IDs associated with the given agent.
// Uses idx_pattern_agent_assoc_agent index for efficient lookup.
func (r *pgxRepository) GetPatternIDsByAgent(ctx context.Context, agentID uuid.UUID) ([]uuid.UUID, error) {
	query := `SELECT pattern_id FROM pattern_agent_associations WHERE agent_id = $1`

	rows, err := r.db.Query(ctx, query, agentID)
	if err != nil {
		return nil, fmt.Errorf("getting pattern IDs by agent: %w", err)
	}
	defer rows.Close()

	var patternIDs []uuid.UUID

	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("getting pattern IDs by agent: scanning row: %w", err)
		}
		patternIDs = append(patternIDs, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("getting pattern IDs by agent: iterating rows: %w", err)
	}

	return patternIDs, nil
}

// Exists checks if a pattern with the given ID exists.
func (r *pgxRepository) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM patterns WHERE id = $1)`

	var exists bool
	err := r.db.QueryRow(ctx, query, id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("checking pattern exists: %w", err)
	}

	return exists, nil
}

// validateAgentIDs checks that all agent IDs exist in the agents table.
// Returns ErrAgentNotFound with details if any IDs are invalid.
func (r *pgxRepository) validateAgentIDs(ctx context.Context, db repository.DBTX, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}

	// Build query to find which IDs exist
	var placeholders []string
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders = append(placeholders, fmt.Sprintf("$%d", i+1))
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT id FROM agents WHERE id IN (%s)
	`, strings.Join(placeholders, ", "))

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("querying agents: %w", err)
	}
	defer rows.Close()

	// Collect found IDs
	foundIDs := make(map[uuid.UUID]bool)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("scanning agent id: %w", err)
		}
		foundIDs[id] = true
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("reading agent ids: %w", err)
	}

	// Check for missing IDs
	var missingIDs []string
	for _, id := range ids {
		if !foundIDs[id] {
			missingIDs = append(missingIDs, id.String())
		}
	}

	if len(missingIDs) > 0 {
		return fmt.Errorf("%w: %s", ErrAgentNotFound, strings.Join(missingIDs, ", "))
	}

	return nil
}
