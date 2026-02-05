package routingrule

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/twistingmercury/mnemonic/internal/repository"
)

// Repository defines data access operations for routing rules.
type Repository interface {
	// Create stores a new routing rule. Returns ErrRuleNameExists if name already exists.
	Create(ctx context.Context, rule *RoutingRule) error

	// Get retrieves a routing rule by ID. Returns ErrRuleNotFound if not found.
	Get(ctx context.Context, id uuid.UUID) (*RoutingRule, error)

	// GetByName retrieves a routing rule by name. Returns ErrRuleNotFound if not found.
	GetByName(ctx context.Context, name string) (*RoutingRule, error)

	// Update modifies an existing routing rule. Returns ErrRuleNotFound if not found.
	Update(ctx context.Context, rule *RoutingRule) error

	// Delete removes a routing rule by ID. Returns ErrRuleNotFound if not found.
	Delete(ctx context.Context, id uuid.UUID) error

	// List retrieves routing rules with filtering and pagination.
	// Returns the rules, total count, and any error.
	List(ctx context.Context, filter RuleFilter, opts repository.ListOptions) ([]*RoutingRule, int64, error)

	// ListEnabled retrieves all enabled rules ordered by priority (descending).
	// This is the primary method used by the routing engine.
	ListEnabled(ctx context.Context) ([]*RoutingRule, error)

	// SetEnabled updates the enabled state of a rule.
	SetEnabled(ctx context.Context, id uuid.UUID, enabled bool) error

	// Exists checks if a routing rule with the given ID exists.
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

// Create stores a new routing rule in the database.
// Validates that MatchConfig type matches MatchType.
// Uses SQL now() for timestamps to ensure consistency with database time.
func (r *pgxRepository) Create(ctx context.Context, rule *RoutingRule) error {
	// Validate that MatchConfig type matches MatchType
	if rule.MatchConfig != nil && rule.MatchConfig.Type() != rule.MatchType {
		return ErrInvalidMatchConfig
	}

	matchConfigJSON, err := MarshalMatchConfig(rule.MatchConfig)
	if err != nil {
		return fmt.Errorf("marshaling match_config: %w", err)
	}

	// Generate UUID if not provided
	if rule.ID == uuid.Nil {
		rule.ID = uuid.New()
	}

	query := `
		INSERT INTO routing_rules (
			id, name, priority, agent_name, match_type,
			match_config, enabled, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, now(), now())
		RETURNING created_at, updated_at
	`

	err = r.db.QueryRow(ctx, query,
		rule.ID,
		rule.Name,
		rule.Priority,
		rule.AgentName,
		rule.MatchType,
		matchConfigJSON,
		rule.Enabled,
	).Scan(&rule.CreatedAt, &rule.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case repository.PgErrCodeUniqueViolation:
				return ErrRuleNameExists
			case repository.PgErrCodeForeignKeyViolation:
				return ErrAgentNotFound
			case repository.PgErrCodeCheckViolation:
				// Return the constraint violation for debugging
				return err
			}
		}
		return err
	}

	return nil
}

// Get retrieves a routing rule by ID from the database.
func (r *pgxRepository) Get(ctx context.Context, id uuid.UUID) (*RoutingRule, error) {
	query := `
		SELECT id, name, priority, agent_name, match_type,
			   match_config, enabled, created_at, updated_at
		FROM routing_rules
		WHERE id = $1
	`

	return r.scanRule(ctx, query, id)
}

// GetByName retrieves a routing rule by name from the database.
func (r *pgxRepository) GetByName(ctx context.Context, name string) (*RoutingRule, error) {
	query := `
		SELECT id, name, priority, agent_name, match_type,
			   match_config, enabled, created_at, updated_at
		FROM routing_rules
		WHERE name = $1
	`

	return r.scanRule(ctx, query, name)
}

// scanRule is a helper that executes a query and scans the result into a RoutingRule.
func (r *pgxRepository) scanRule(ctx context.Context, query string, args ...any) (*RoutingRule, error) {
	var rule RoutingRule
	var matchConfigJSON []byte

	err := r.db.QueryRow(ctx, query, args...).Scan(
		&rule.ID,
		&rule.Name,
		&rule.Priority,
		&rule.AgentName,
		&rule.MatchType,
		&matchConfigJSON,
		&rule.Enabled,
		&rule.CreatedAt,
		&rule.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRuleNotFound
		}
		return nil, err
	}

	matchConfig, err := UnmarshalMatchConfig(rule.MatchType, matchConfigJSON)
	if err != nil {
		return nil, fmt.Errorf("unmarshaling match_config: %w", err)
	}
	rule.MatchConfig = matchConfig

	return &rule, nil
}

// Update modifies an existing routing rule in the database.
// Validates that MatchConfig type matches MatchType.
// Uses SQL now() for timestamps to ensure consistency with database time.
func (r *pgxRepository) Update(ctx context.Context, rule *RoutingRule) error {
	// Validate that MatchConfig type matches MatchType
	if rule.MatchConfig != nil && rule.MatchConfig.Type() != rule.MatchType {
		return ErrInvalidMatchConfig
	}

	matchConfigJSON, err := MarshalMatchConfig(rule.MatchConfig)
	if err != nil {
		return fmt.Errorf("marshaling match_config: %w", err)
	}

	query := `
		UPDATE routing_rules SET
			name = $2,
			priority = $3,
			agent_name = $4,
			match_type = $5,
			match_config = $6,
			enabled = $7,
			updated_at = now()
		WHERE id = $1
		RETURNING updated_at
	`

	err = r.db.QueryRow(ctx, query,
		rule.ID,
		rule.Name,
		rule.Priority,
		rule.AgentName,
		rule.MatchType,
		matchConfigJSON,
		rule.Enabled,
	).Scan(&rule.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrRuleNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case repository.PgErrCodeUniqueViolation:
				return ErrRuleNameExists
			case repository.PgErrCodeForeignKeyViolation:
				return ErrAgentNotFound
			case repository.PgErrCodeCheckViolation:
				return err
			}
		}
		return err
	}

	return nil
}

// Delete removes a routing rule by ID from the database.
func (r *pgxRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM routing_rules WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrRuleNotFound
	}

	return nil
}

// List retrieves routing rules with filtering and pagination.
func (r *pgxRepository) List(ctx context.Context, filter RuleFilter, opts repository.ListOptions) ([]*RoutingRule, int64, error) {
	// Build the WHERE clause dynamically
	var conditions []string
	var args []any
	argIndex := 1

	if filter.AgentName != nil {
		conditions = append(conditions, fmt.Sprintf("agent_name = $%d", argIndex))
		args = append(args, *filter.AgentName)
		argIndex++
	}

	if filter.MatchType != nil {
		conditions = append(conditions, fmt.Sprintf("match_type = $%d", argIndex))
		args = append(args, *filter.MatchType)
		argIndex++
	}

	if filter.Enabled != nil {
		conditions = append(conditions, fmt.Sprintf("enabled = $%d", argIndex))
		args = append(args, *filter.Enabled)
		argIndex++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Build query with window function for total count
	query := fmt.Sprintf(`
		SELECT id, name, priority, agent_name, match_type,
			   match_config, enabled, created_at, updated_at,
			   COUNT(*) OVER() as total_count
		FROM routing_rules
		%s
		ORDER BY priority DESC, id ASC
	`, whereClause)

	// Add pagination
	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, opts.Limit)
		argIndex++
		if opts.Offset > 0 {
			query += fmt.Sprintf(" OFFSET $%d", argIndex)
			args = append(args, opts.Offset)
		}
	} else if opts.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, opts.Offset)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	rules := make([]*RoutingRule, 0)
	var totalCount int64

	for rows.Next() {
		var rule RoutingRule
		var matchConfigJSON []byte

		err := rows.Scan(
			&rule.ID,
			&rule.Name,
			&rule.Priority,
			&rule.AgentName,
			&rule.MatchType,
			&matchConfigJSON,
			&rule.Enabled,
			&rule.CreatedAt,
			&rule.UpdatedAt,
			&totalCount,
		)
		if err != nil {
			return nil, 0, err
		}

		matchConfig, err := UnmarshalMatchConfig(rule.MatchType, matchConfigJSON)
		if err != nil {
			return nil, 0, fmt.Errorf("unmarshaling match_config: %w", err)
		}
		rule.MatchConfig = matchConfig

		rules = append(rules, &rule)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return rules, totalCount, nil
}

// ListEnabled retrieves all enabled rules ordered by priority (descending).
func (r *pgxRepository) ListEnabled(ctx context.Context) ([]*RoutingRule, error) {
	query := `
		SELECT id, name, priority, agent_name, match_type,
			   match_config, enabled, created_at, updated_at
		FROM routing_rules
		WHERE enabled = true
		ORDER BY priority DESC, id ASC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rules := make([]*RoutingRule, 0)

	for rows.Next() {
		var rule RoutingRule
		var matchConfigJSON []byte

		err := rows.Scan(
			&rule.ID,
			&rule.Name,
			&rule.Priority,
			&rule.AgentName,
			&rule.MatchType,
			&matchConfigJSON,
			&rule.Enabled,
			&rule.CreatedAt,
			&rule.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		matchConfig, err := UnmarshalMatchConfig(rule.MatchType, matchConfigJSON)
		if err != nil {
			return nil, fmt.Errorf("unmarshaling match_config: %w", err)
		}
		rule.MatchConfig = matchConfig

		rules = append(rules, &rule)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return rules, nil
}

// SetEnabled updates the enabled state of a rule.
// Uses SQL now() for timestamps to ensure consistency with database time.
func (r *pgxRepository) SetEnabled(ctx context.Context, id uuid.UUID, enabled bool) error {
	query := `
		UPDATE routing_rules SET
			enabled = $2,
			updated_at = now()
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query, id, enabled)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrRuleNotFound
	}

	return nil
}

// Exists checks if a routing rule with the given ID exists.
func (r *pgxRepository) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM routing_rules WHERE id = $1)`

	var exists bool
	err := r.db.QueryRow(ctx, query, id).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}
