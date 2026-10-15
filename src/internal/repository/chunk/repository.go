package chunk

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pgvector/pgvector-go"
	"github.com/twistingmercury/mnemonic/internal/repository"
)

// Repository defines the chunk reads the MCP tools need.
type Repository interface {
	// FindSimilar finds chunks similar to the given embedding vector.
	// Joins to patterns to return parent pattern metadata.
	FindSimilar(ctx context.Context, embedding []float32, opts SimilarityOptions) ([]*Match, error)
}

// pgxRepository is a PostgreSQL implementation of Repository using pgx.
type pgxRepository struct {
	db repository.DBTX
}

// NewRepository creates a new PostgreSQL-backed Repository.
func NewRepository(db repository.DBTX) Repository {
	return &pgxRepository{db: db}
}

// FindSimilar finds chunks similar to the given embedding vector.
// Joins to patterns to return parent pattern metadata.
func (r *pgxRepository) FindSimilar(ctx context.Context, embedding []float32, opts SimilarityOptions) ([]*Match, error) {
	vec := pgvector.NewVector(embedding)

	maxResults := opts.MaxResults
	if maxResults <= 0 {
		maxResults = 10
	}

	// Fixed parameters: $1 = embedding vector, $2 = min similarity, $3 = language, $4 = domain, $5 = limit.
	query := `
		SELECT pc.pattern_id, p.name, p.tags, pc.section_title, pc.content,
		       1 - (pc.embedding <=> $1) AS similarity
		FROM pattern_chunks pc
		JOIN patterns p ON p.id = pc.pattern_id
		WHERE pc.embedding IS NOT NULL
		  AND pc.enrichment_status = 'enriched'
		  AND 1 - (pc.embedding <=> $1) >= $2
		  AND ($3::text = '' OR p.language = $3)
		  AND ($4::text = '' OR p.domain = $4)
	`

	args := []any{vec, opts.MinSimilarity, opts.Language, opts.Domain}
	nextParam := 5

	if len(opts.Tags) > 0 {
		tagsJSON, err := json.Marshal(opts.Tags)
		if err != nil {
			return nil, fmt.Errorf("finding similar chunks: marshaling tags filter: %w", err)
		}
		query += fmt.Sprintf(" AND p.tags @> $%d::jsonb", nextParam)
		args = append(args, string(tagsJSON))
		nextParam++
	}

	query += fmt.Sprintf(" ORDER BY pc.embedding <=> $1 LIMIT $%d", nextParam)
	args = append(args, maxResults)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("finding similar chunks: %w", err)
	}
	defer rows.Close()

	matches := make([]*Match, 0, maxResults)

	for rows.Next() {
		var m Match
		var tagsJSON []byte

		if err := rows.Scan(&m.PatternID, &m.PatternName, &tagsJSON, &m.SectionTitle, &m.Content, &m.Similarity); err != nil {
			return nil, fmt.Errorf("finding similar chunks: scanning row: %w", err)
		}

		if err := json.Unmarshal(tagsJSON, &m.Tags); err != nil {
			return nil, fmt.Errorf("finding similar chunks: unmarshaling tags: %w", err)
		}

		matches = append(matches, &m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("finding similar chunks: iterating rows: %w", err)
	}

	return matches, nil
}
