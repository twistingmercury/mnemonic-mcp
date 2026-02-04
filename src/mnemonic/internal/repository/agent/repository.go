package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/twistingmercury/mnemonic/internal/repository"
)

// Repository defines data access operations for agents.
type Repository interface {
	// Create stores a new agent. Returns ErrAgentExists if name already exists.
	Create(ctx context.Context, agent *Agent) error

	// Get retrieves an agent by name. Returns ErrAgentNotFound if not found.
	Get(ctx context.Context, name string) (*Agent, error)

	// Update modifies an existing agent. Returns ErrAgentNotFound if not found.
	Update(ctx context.Context, agent *Agent) error

	// Delete removes an agent by name. Returns ErrAgentInUse if referenced by routing rules.
	Delete(ctx context.Context, name string) error

	// List retrieves all agents with optional pagination.
	// Returns the agents, total count, and any error.
	List(ctx context.Context, opts repository.ListOptions) ([]*Agent, int64, error)

	// Exists checks if an agent with the given name exists.
	Exists(ctx context.Context, name string) (bool, error)
}

// pgxRepository is a PostgreSQL implementation of Repository using pgx.
type pgxRepository struct {
	db repository.DBTX
}

// NewRepository creates a new PostgreSQL-backed Repository.
func NewRepository(db repository.DBTX) Repository {
	return &pgxRepository{db: db}
}

// Create stores a new agent in the database.
func (r *pgxRepository) Create(ctx context.Context, agent *Agent) error {
	allowedToolsJSON, err := json.Marshal(agent.AllowedTools)
	if err != nil {
		return fmt.Errorf("marshaling allowed_tools: %w", err)
	}

	routingKeywordsJSON, err := json.Marshal(agent.RoutingKeywords)
	if err != nil {
		return fmt.Errorf("marshaling routing_keywords: %w", err)
	}

	now := time.Now()
	query := `
		INSERT INTO agents (
			name, description, system_prompt, model,
			allowed_tools, routing_keywords, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err = r.db.Exec(ctx, query,
		agent.Name,
		agent.Description,
		agent.SystemPrompt,
		agent.Model,
		allowedToolsJSON,
		routingKeywordsJSON,
		now,
		now,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case repository.PgErrCodeUniqueViolation:
				return ErrAgentExists
			case repository.PgErrCodeCheckViolation:
				// Return the constraint violation message for better debugging
				return err
			}
		}
		return err
	}

	agent.CreatedAt = now
	agent.UpdatedAt = now
	return nil
}

// Get retrieves an agent by name from the database.
func (r *pgxRepository) Get(ctx context.Context, name string) (*Agent, error) {
	query := `
		SELECT name, description, system_prompt, model,
			   allowed_tools, routing_keywords, created_at, updated_at
		FROM agents
		WHERE name = $1
	`

	var agent Agent
	var allowedToolsJSON, routingKeywordsJSON []byte

	err := r.db.QueryRow(ctx, query, name).Scan(
		&agent.Name,
		&agent.Description,
		&agent.SystemPrompt,
		&agent.Model,
		&allowedToolsJSON,
		&routingKeywordsJSON,
		&agent.CreatedAt,
		&agent.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAgentNotFound
		}
		return nil, err
	}

	if err := json.Unmarshal(allowedToolsJSON, &agent.AllowedTools); err != nil {
		return nil, fmt.Errorf("unmarshaling allowed_tools: %w", err)
	}
	if err := json.Unmarshal(routingKeywordsJSON, &agent.RoutingKeywords); err != nil {
		return nil, fmt.Errorf("unmarshaling routing_keywords: %w", err)
	}

	return &agent, nil
}

// Update modifies an existing agent in the database.
func (r *pgxRepository) Update(ctx context.Context, agent *Agent) error {
	allowedToolsJSON, err := json.Marshal(agent.AllowedTools)
	if err != nil {
		return fmt.Errorf("marshaling allowed_tools: %w", err)
	}

	routingKeywordsJSON, err := json.Marshal(agent.RoutingKeywords)
	if err != nil {
		return fmt.Errorf("marshaling routing_keywords: %w", err)
	}

	now := time.Now()
	query := `
		UPDATE agents SET
			description = $2,
			system_prompt = $3,
			model = $4,
			allowed_tools = $5,
			routing_keywords = $6,
			updated_at = $7
		WHERE name = $1
	`

	result, err := r.db.Exec(ctx, query,
		agent.Name,
		agent.Description,
		agent.SystemPrompt,
		agent.Model,
		allowedToolsJSON,
		routingKeywordsJSON,
		now,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == repository.PgErrCodeCheckViolation {
			return err
		}
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrAgentNotFound
	}

	agent.UpdatedAt = now
	return nil
}

// Delete removes an agent by name from the database.
func (r *pgxRepository) Delete(ctx context.Context, name string) error {
	query := `DELETE FROM agents WHERE name = $1`

	result, err := r.db.Exec(ctx, query, name)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == repository.PgErrCodeForeignKeyViolation {
			return ErrAgentInUse
		}
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrAgentNotFound
	}

	return nil
}

// List retrieves all agents with optional pagination.
func (r *pgxRepository) List(ctx context.Context, opts repository.ListOptions) ([]*Agent, int64, error) {
	// Build query with window function for total count in a single query
	query := `
		SELECT name, description, system_prompt, model,
			   allowed_tools, routing_keywords, created_at, updated_at,
			   COUNT(*) OVER() as total_count
		FROM agents
		ORDER BY name ASC
	`

	args := make([]any, 0, 2)
	if opts.Limit > 0 {
		query += " LIMIT $1"
		args = append(args, opts.Limit)
		if opts.Offset > 0 {
			query += " OFFSET $2"
			args = append(args, opts.Offset)
		}
	} else if opts.Offset > 0 {
		// If only offset is specified without limit, we still need to handle it
		query += " OFFSET $1"
		args = append(args, opts.Offset)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	agents := make([]*Agent, 0)
	var totalCount int64

	for rows.Next() {
		var agent Agent
		var allowedToolsJSON, routingKeywordsJSON []byte

		err := rows.Scan(
			&agent.Name,
			&agent.Description,
			&agent.SystemPrompt,
			&agent.Model,
			&allowedToolsJSON,
			&routingKeywordsJSON,
			&agent.CreatedAt,
			&agent.UpdatedAt,
			&totalCount,
		)
		if err != nil {
			return nil, 0, err
		}

		if err := json.Unmarshal(allowedToolsJSON, &agent.AllowedTools); err != nil {
			return nil, 0, fmt.Errorf("unmarshaling allowed_tools: %w", err)
		}
		if err := json.Unmarshal(routingKeywordsJSON, &agent.RoutingKeywords); err != nil {
			return nil, 0, fmt.Errorf("unmarshaling routing_keywords: %w", err)
		}

		agents = append(agents, &agent)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return agents, totalCount, nil
}

// Exists checks if an agent with the given name exists.
func (r *pgxRepository) Exists(ctx context.Context, name string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM agents WHERE name = $1)`

	var exists bool
	err := r.db.QueryRow(ctx, query, name).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}
