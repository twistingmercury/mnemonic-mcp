package routingrule_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twistingmercury/mnemonic/internal/repository"
	"github.com/twistingmercury/mnemonic/internal/repository/routingrule"
)

// testKeywordRule returns a sample keyword rule for testing.
func testKeywordRule() *routingrule.Rule {
	return &routingrule.Rule{
		ID:        uuid.New(),
		Name:      "go-keyword-rule",
		Priority:  100,
		AgentName: "go-software-agent",
		MatchType: "keyword",
		MatchConfig: routingrule.KeywordMatchConfig{
			Keywords:  []string{"go", "golang"},
			MatchMode: "any",
		},
		Enabled: true,
	}
}

// testRegexRule returns a sample regex rule for testing.
func testRegexRule() *routingrule.Rule {
	return &routingrule.Rule{
		ID:        uuid.New(),
		Name:      "python-regex-rule",
		Priority:  90,
		AgentName: "python-agent",
		MatchType: "regex",
		MatchConfig: routingrule.RegexMatchConfig{
			Pattern: `\b(python|py)\b`,
			Flags:   "i",
		},
		Enabled: true,
	}
}

// testDefaultRule returns a sample default rule for testing.
func testDefaultRule() *routingrule.Rule {
	return &routingrule.Rule{
		ID:          uuid.New(),
		Name:        "default-fallback",
		Priority:    0,
		AgentName:   "general-agent",
		MatchType:   "default",
		MatchConfig: routingrule.DefaultMatchConfig{},
		Enabled:     true,
	}
}

func TestRepository_Create(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name      string
		rule      *routingrule.Rule
		setupMock func(mock pgxmock.PgxPoolIface)
		wantErr   error
	}{
		{
			name: "successful creation with keyword config",
			rule: testKeywordRule(),
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"created_at", "updated_at"}).
					AddRow(now, now)
				mock.ExpectQuery("INSERT INTO routing_rules").
					WithArgs(
						pgxmock.AnyArg(), // id
						"go-keyword-rule",
						100,
						"go-software-agent",
						"keyword",
						pgxmock.AnyArg(), // match_config JSON
						true,
					).
					WillReturnRows(rows)
			},
			wantErr: nil,
		},
		{
			name: "successful creation with regex config",
			rule: testRegexRule(),
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"created_at", "updated_at"}).
					AddRow(now, now)
				mock.ExpectQuery("INSERT INTO routing_rules").
					WithArgs(
						pgxmock.AnyArg(),
						"python-regex-rule",
						90,
						"python-agent",
						"regex",
						pgxmock.AnyArg(),
						true,
					).
					WillReturnRows(rows)
			},
			wantErr: nil,
		},
		{
			name: "successful creation with default config",
			rule: testDefaultRule(),
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"created_at", "updated_at"}).
					AddRow(now, now)
				mock.ExpectQuery("INSERT INTO routing_rules").
					WithArgs(
						pgxmock.AnyArg(),
						"default-fallback",
						0,
						"general-agent",
						"default",
						pgxmock.AnyArg(),
						true,
					).
					WillReturnRows(rows)
			},
			wantErr: nil,
		},
		{
			name: "duplicate name returns ErrNameExists",
			rule: testKeywordRule(),
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("INSERT INTO routing_rules").
					WithArgs(
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
					).
					WillReturnError(&pgconn.PgError{Code: "23505"})
			},
			wantErr: routingrule.ErrNameExists,
		},
		{
			name: "agent not found returns ErrAgentNotFound",
			rule: testKeywordRule(),
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("INSERT INTO routing_rules").
					WithArgs(
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
					).
					WillReturnError(&pgconn.PgError{Code: "23503"})
			},
			wantErr: routingrule.ErrAgentNotFound,
		},
		{
			name: "mismatched match type returns ErrInvalidMatchConfig",
			rule: &routingrule.Rule{
				ID:        uuid.New(),
				Name:      "mismatched-rule",
				Priority:  100,
				AgentName: "agent",
				MatchType: "regex", // Mismatch: MatchType is regex
				MatchConfig: routingrule.KeywordMatchConfig{ // but config is keyword
					Keywords:  []string{"go"},
					MatchMode: "any",
				},
				Enabled: true,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				// No DB call expected - validation fails before query
			},
			wantErr: routingrule.ErrInvalidMatchConfig,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()

			tt.setupMock(mock)

			repo := routingrule.NewRepository(mock)
			err = repo.Create(context.Background(), tt.rule)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
				assert.False(t, tt.rule.CreatedAt.IsZero())
				assert.False(t, tt.rule.UpdatedAt.IsZero())
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepository_Create_GeneratesUUID(t *testing.T) {
	t.Parallel()

	now := time.Now()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	rule := &routingrule.Rule{
		// ID is not set - should be generated
		Name:        "test-rule",
		Priority:    50,
		AgentName:   "test-agent",
		MatchType:   "default",
		MatchConfig: routingrule.DefaultMatchConfig{},
		Enabled:     true,
	}

	rows := pgxmock.NewRows([]string{"created_at", "updated_at"}).
		AddRow(now, now)
	mock.ExpectQuery("INSERT INTO routing_rules").
		WithArgs(
			pgxmock.AnyArg(), // generated id
			"test-rule",
			50,
			"test-agent",
			"default",
			pgxmock.AnyArg(),
			true,
		).
		WillReturnRows(rows)

	repo := routingrule.NewRepository(mock)
	err = repo.Create(context.Background(), rule)

	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, rule.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_Get(t *testing.T) {
	t.Parallel()

	now := time.Now()
	ruleID := uuid.New()

	tests := []struct {
		name      string
		ruleID    uuid.UUID
		setupMock func(mock pgxmock.PgxPoolIface)
		wantRule  *routingrule.Rule
		wantErr   error
	}{
		{
			name:   "successful retrieval of keyword rule",
			ruleID: ruleID,
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "name", "priority", "agent_name", "match_type",
					"match_config", "enabled", "created_at", "updated_at",
				}).AddRow(
					ruleID,
					"go-keyword-rule",
					100,
					"go-software-agent",
					"keyword",
					[]byte(`{"keywords":["go","golang"],"match_mode":"any"}`),
					true,
					now,
					now,
				)
				mock.ExpectQuery("SELECT .* FROM routing_rules").
					WithArgs(ruleID).
					WillReturnRows(rows)
			},
			wantRule: &routingrule.Rule{
				ID:        ruleID,
				Name:      "go-keyword-rule",
				Priority:  100,
				AgentName: "go-software-agent",
				MatchType: "keyword",
				MatchConfig: routingrule.KeywordMatchConfig{
					Keywords:  []string{"go", "golang"},
					MatchMode: "any",
				},
				Enabled:   true,
				CreatedAt: now,
				UpdatedAt: now,
			},
			wantErr: nil,
		},
		{
			name:   "successful retrieval of default rule",
			ruleID: ruleID,
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "name", "priority", "agent_name", "match_type",
					"match_config", "enabled", "created_at", "updated_at",
				}).AddRow(
					ruleID,
					"default-fallback",
					0,
					"general-agent",
					"default",
					[]byte(`{}`),
					true,
					now,
					now,
				)
				mock.ExpectQuery("SELECT .* FROM routing_rules").
					WithArgs(ruleID).
					WillReturnRows(rows)
			},
			wantRule: &routingrule.Rule{
				ID:          ruleID,
				Name:        "default-fallback",
				Priority:    0,
				AgentName:   "general-agent",
				MatchType:   "default",
				MatchConfig: routingrule.DefaultMatchConfig{},
				Enabled:     true,
				CreatedAt:   now,
				UpdatedAt:   now,
			},
			wantErr: nil,
		},
		{
			name:   "not found returns ErrNotFound",
			ruleID: ruleID,
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT .* FROM routing_rules").
					WithArgs(ruleID).
					WillReturnError(pgx.ErrNoRows)
			},
			wantRule: nil,
			wantErr:  routingrule.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()

			tt.setupMock(mock)

			repo := routingrule.NewRepository(mock)
			rule, err := repo.Get(context.Background(), tt.ruleID)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, rule)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantRule.ID, rule.ID)
				assert.Equal(t, tt.wantRule.Name, rule.Name)
				assert.Equal(t, tt.wantRule.Priority, rule.Priority)
				assert.Equal(t, tt.wantRule.AgentName, rule.AgentName)
				assert.Equal(t, tt.wantRule.MatchType, rule.MatchType)
				assert.Equal(t, tt.wantRule.Enabled, rule.Enabled)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepository_GetByName(t *testing.T) {
	t.Parallel()

	now := time.Now()
	ruleID := uuid.New()

	tests := []struct {
		name      string
		ruleName  string
		setupMock func(mock pgxmock.PgxPoolIface)
		wantErr   error
	}{
		{
			name:     "successful retrieval",
			ruleName: "go-keyword-rule",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "name", "priority", "agent_name", "match_type",
					"match_config", "enabled", "created_at", "updated_at",
				}).AddRow(
					ruleID,
					"go-keyword-rule",
					100,
					"go-software-agent",
					"keyword",
					[]byte(`{"keywords":["go"],"match_mode":"any"}`),
					true,
					now,
					now,
				)
				mock.ExpectQuery("SELECT .* FROM routing_rules").
					WithArgs("go-keyword-rule").
					WillReturnRows(rows)
			},
			wantErr: nil,
		},
		{
			name:     "not found returns ErrNotFound",
			ruleName: "nonexistent",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT .* FROM routing_rules").
					WithArgs("nonexistent").
					WillReturnError(pgx.ErrNoRows)
			},
			wantErr: routingrule.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()

			tt.setupMock(mock)

			repo := routingrule.NewRepository(mock)
			rule, err := repo.GetByName(context.Background(), tt.ruleName)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, rule)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.ruleName, rule.Name)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepository_Update(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name      string
		rule      *routingrule.Rule
		setupMock func(mock pgxmock.PgxPoolIface)
		wantErr   error
	}{
		{
			name: "successful update",
			rule: testKeywordRule(),
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"updated_at"}).
					AddRow(now)
				mock.ExpectQuery("UPDATE routing_rules SET").
					WithArgs(
						pgxmock.AnyArg(), // id
						"go-keyword-rule",
						100,
						"go-software-agent",
						"keyword",
						pgxmock.AnyArg(), // match_config
						true,
					).
					WillReturnRows(rows)
			},
			wantErr: nil,
		},
		{
			name: "not found returns ErrNotFound",
			rule: testKeywordRule(),
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("UPDATE routing_rules SET").
					WithArgs(
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
					).
					WillReturnError(pgx.ErrNoRows)
			},
			wantErr: routingrule.ErrNotFound,
		},
		{
			name: "duplicate name returns ErrNameExists",
			rule: testKeywordRule(),
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("UPDATE routing_rules SET").
					WithArgs(
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
					).
					WillReturnError(&pgconn.PgError{Code: "23505"})
			},
			wantErr: routingrule.ErrNameExists,
		},
		{
			name: "agent not found returns ErrAgentNotFound",
			rule: testKeywordRule(),
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("UPDATE routing_rules SET").
					WithArgs(
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
					).
					WillReturnError(&pgconn.PgError{Code: "23503"})
			},
			wantErr: routingrule.ErrAgentNotFound,
		},
		{
			name: "mismatched match type returns ErrInvalidMatchConfig",
			rule: &routingrule.Rule{
				ID:        uuid.New(),
				Name:      "mismatched-rule",
				Priority:  100,
				AgentName: "agent",
				MatchType: "keyword", // Mismatch: MatchType is keyword
				MatchConfig: routingrule.RegexMatchConfig{ // but config is regex
					Pattern: `\bgo\b`,
					Flags:   "i",
				},
				Enabled: true,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				// No DB call expected - validation fails before query
			},
			wantErr: routingrule.ErrInvalidMatchConfig,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()

			tt.setupMock(mock)

			repo := routingrule.NewRepository(mock)
			err = repo.Update(context.Background(), tt.rule)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
				assert.False(t, tt.rule.UpdatedAt.IsZero())
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepository_Delete(t *testing.T) {
	t.Parallel()

	ruleID := uuid.New()

	tests := []struct {
		name      string
		ruleID    uuid.UUID
		setupMock func(mock pgxmock.PgxPoolIface)
		wantErr   error
	}{
		{
			name:   "successful deletion",
			ruleID: ruleID,
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("DELETE FROM routing_rules").
					WithArgs(ruleID).
					WillReturnResult(pgxmock.NewResult("DELETE", 1))
			},
			wantErr: nil,
		},
		{
			name:   "not found returns ErrNotFound",
			ruleID: ruleID,
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("DELETE FROM routing_rules").
					WithArgs(ruleID).
					WillReturnResult(pgxmock.NewResult("DELETE", 0))
			},
			wantErr: routingrule.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()

			tt.setupMock(mock)

			repo := routingrule.NewRepository(mock)
			err = repo.Delete(context.Background(), tt.ruleID)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepository_List(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name      string
		filter    routingrule.Filter
		opts      repository.ListOptions
		setupMock func(mock pgxmock.PgxPoolIface)
		wantCount int
		wantTotal int64
		wantNames []string
		wantErr   error
	}{
		{
			name:   "list all rules without filter",
			filter: routingrule.Filter{},
			opts:   repository.ListOptions{},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "name", "priority", "agent_name", "match_type",
					"match_config", "enabled", "created_at", "updated_at", "total_count",
				}).
					AddRow(uuid.New(), "rule-a", 100, "agent-a", "keyword",
						[]byte(`{"keywords":["go"],"match_mode":"any"}`), true, now, now, int64(2)).
					AddRow(uuid.New(), "rule-b", 50, "agent-b", "default",
						[]byte(`{}`), true, now, now, int64(2))

				mock.ExpectQuery("SELECT .* FROM routing_rules ORDER BY priority DESC").
					WillReturnRows(rows)
			},
			wantCount: 2,
			wantTotal: 2,
			wantNames: []string{"rule-a", "rule-b"},
		},
		{
			name: "list with agent filter",
			filter: routingrule.Filter{
				AgentName: ptr("go-agent"),
			},
			opts: repository.ListOptions{},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "name", "priority", "agent_name", "match_type",
					"match_config", "enabled", "created_at", "updated_at", "total_count",
				}).
					AddRow(uuid.New(), "go-rule", 100, "go-agent", "keyword",
						[]byte(`{"keywords":["go"],"match_mode":"any"}`), true, now, now, int64(1))

				mock.ExpectQuery("SELECT .* FROM routing_rules WHERE agent_name").
					WithArgs("go-agent").
					WillReturnRows(rows)
			},
			wantCount: 1,
			wantTotal: 1,
			wantNames: []string{"go-rule"},
		},
		{
			name: "list with enabled filter",
			filter: routingrule.Filter{
				Enabled: ptr(true),
			},
			opts: repository.ListOptions{},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "name", "priority", "agent_name", "match_type",
					"match_config", "enabled", "created_at", "updated_at", "total_count",
				}).
					AddRow(uuid.New(), "enabled-rule", 100, "agent", "default",
						[]byte(`{}`), true, now, now, int64(1))

				mock.ExpectQuery("SELECT .* FROM routing_rules WHERE enabled").
					WithArgs(true).
					WillReturnRows(rows)
			},
			wantCount: 1,
			wantTotal: 1,
			wantNames: []string{"enabled-rule"},
		},
		{
			name:   "list with pagination",
			filter: routingrule.Filter{},
			opts:   repository.ListOptions{Limit: 1, Offset: 1},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "name", "priority", "agent_name", "match_type",
					"match_config", "enabled", "created_at", "updated_at", "total_count",
				}).
					AddRow(uuid.New(), "rule-b", 50, "agent-b", "default",
						[]byte(`{}`), true, now, now, int64(2))

				mock.ExpectQuery("SELECT .* FROM routing_rules ORDER BY priority DESC, id ASC LIMIT").
					WithArgs(1, 1).
					WillReturnRows(rows)
			},
			wantCount: 1,
			wantTotal: 2,
			wantNames: []string{"rule-b"},
		},
		{
			name:   "empty list returns empty slice",
			filter: routingrule.Filter{},
			opts:   repository.ListOptions{},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "name", "priority", "agent_name", "match_type",
					"match_config", "enabled", "created_at", "updated_at", "total_count",
				})

				mock.ExpectQuery("SELECT .* FROM routing_rules ORDER BY priority DESC").
					WillReturnRows(rows)
			},
			wantCount: 0,
			wantTotal: 0,
			wantNames: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()

			tt.setupMock(mock)

			repo := routingrule.NewRepository(mock)
			rules, total, err := repo.List(context.Background(), tt.filter, tt.opts)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantTotal, total)
				assert.Len(t, rules, tt.wantCount)

				for i, expectedName := range tt.wantNames {
					assert.Equal(t, expectedName, rules[i].Name)
				}
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepository_ListEnabled(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name      string
		setupMock func(mock pgxmock.PgxPoolIface)
		wantCount int
		wantErr   error
	}{
		{
			name: "returns enabled rules in priority order",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "name", "priority", "agent_name", "match_type",
					"match_config", "enabled", "created_at", "updated_at",
				}).
					AddRow(uuid.New(), "high-priority", 100, "agent-a", "keyword",
						[]byte(`{"keywords":["go"],"match_mode":"any"}`), true, now, now).
					AddRow(uuid.New(), "low-priority", 10, "agent-b", "default",
						[]byte(`{}`), true, now, now)

				mock.ExpectQuery("SELECT .* FROM routing_rules WHERE enabled = true ORDER BY priority DESC").
					WillReturnRows(rows)
			},
			wantCount: 2,
		},
		{
			name: "empty result returns empty slice",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "name", "priority", "agent_name", "match_type",
					"match_config", "enabled", "created_at", "updated_at",
				})

				mock.ExpectQuery("SELECT .* FROM routing_rules WHERE enabled = true ORDER BY priority DESC").
					WillReturnRows(rows)
			},
			wantCount: 0,
		},
		{
			name: "database error",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT .* FROM routing_rules WHERE enabled = true ORDER BY priority DESC").
					WillReturnError(errors.New("connection failed"))
			},
			wantErr: errors.New("connection failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()

			tt.setupMock(mock)

			repo := routingrule.NewRepository(mock)
			rules, err := repo.ListEnabled(context.Background())

			if tt.wantErr != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, rules, tt.wantCount)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepository_SetEnabled(t *testing.T) {
	t.Parallel()

	ruleID := uuid.New()

	tests := []struct {
		name      string
		ruleID    uuid.UUID
		enabled   bool
		setupMock func(mock pgxmock.PgxPoolIface)
		wantErr   error
	}{
		{
			name:    "successfully enable rule",
			ruleID:  ruleID,
			enabled: true,
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("UPDATE routing_rules SET").
					WithArgs(ruleID, true).
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))
			},
			wantErr: nil,
		},
		{
			name:    "successfully disable rule",
			ruleID:  ruleID,
			enabled: false,
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("UPDATE routing_rules SET").
					WithArgs(ruleID, false).
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))
			},
			wantErr: nil,
		},
		{
			name:    "not found returns ErrNotFound",
			ruleID:  ruleID,
			enabled: true,
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("UPDATE routing_rules SET").
					WithArgs(ruleID, true).
					WillReturnResult(pgxmock.NewResult("UPDATE", 0))
			},
			wantErr: routingrule.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()

			tt.setupMock(mock)

			repo := routingrule.NewRepository(mock)
			err = repo.SetEnabled(context.Background(), tt.ruleID, tt.enabled)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepository_Exists(t *testing.T) {
	t.Parallel()

	ruleID := uuid.New()

	tests := []struct {
		name      string
		ruleID    uuid.UUID
		setupMock func(mock pgxmock.PgxPoolIface)
		want      bool
		wantErr   error
	}{
		{
			name:   "rule exists",
			ruleID: ruleID,
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"exists"}).AddRow(true)
				mock.ExpectQuery("SELECT EXISTS").
					WithArgs(ruleID).
					WillReturnRows(rows)
			},
			want:    true,
			wantErr: nil,
		},
		{
			name:   "rule does not exist",
			ruleID: ruleID,
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"exists"}).AddRow(false)
				mock.ExpectQuery("SELECT EXISTS").
					WithArgs(ruleID).
					WillReturnRows(rows)
			},
			want:    false,
			wantErr: nil,
		},
		{
			name:   "database error",
			ruleID: ruleID,
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT EXISTS").
					WithArgs(ruleID).
					WillReturnError(errors.New("connection failed"))
			},
			want:    false,
			wantErr: errors.New("connection failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()

			tt.setupMock(mock)

			repo := routingrule.NewRepository(mock)
			exists, err := repo.Exists(context.Background(), tt.ruleID)

			if tt.wantErr != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, exists)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestMatchConfig_UnmarshalMarshal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		matchType string
		jsonData  []byte
		wantType  string
	}{
		{
			name:      "keyword config",
			matchType: "keyword",
			jsonData:  []byte(`{"keywords":["go","golang"],"match_mode":"any"}`),
			wantType:  "keyword",
		},
		{
			name:      "regex config",
			matchType: "regex",
			jsonData:  []byte(`{"pattern":"\\bgo\\b","flags":"i"}`),
			wantType:  "regex",
		},
		{
			name:      "pattern config",
			matchType: "pattern",
			jsonData:  []byte(`{"pattern_ids":["550e8400-e29b-41d4-a716-446655440000"]}`),
			wantType:  "pattern",
		},
		{
			name:      "default config",
			matchType: "default",
			jsonData:  []byte(`{}`),
			wantType:  "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Test unmarshal
			cfg, err := routingrule.UnmarshalMatchConfig(tt.matchType, tt.jsonData)
			require.NoError(t, err)
			assert.Equal(t, tt.wantType, cfg.Type())

			// Test marshal
			data, err := routingrule.MarshalMatchConfig(cfg)
			require.NoError(t, err)
			assert.NotEmpty(t, data)
		})
	}
}

func TestMatchConfig_UnmarshalUnknownType(t *testing.T) {
	t.Parallel()

	_, err := routingrule.UnmarshalMatchConfig("unknown", []byte(`{}`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown match type")
}

func TestMatchConfig_MarshalNil(t *testing.T) {
	t.Parallel()

	data, err := routingrule.MarshalMatchConfig(nil)
	assert.NoError(t, err)
	assert.Equal(t, []byte("{}"), data)
}

func TestIsValidMatchType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		matchType string
		want      bool
	}{
		{"keyword", true},
		{"regex", true},
		{"pattern", true},
		{"default", true},
		{"invalid", false},
		{"KEYWORD", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.matchType, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, routingrule.IsValidMatchType(tt.matchType))
		})
	}
}

func TestIsValidMatchMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		mode string
		want bool
	}{
		{"any", true},
		{"all", true},
		{"none", false},
		{"ANY", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, routingrule.IsValidMatchMode(tt.mode))
		})
	}
}

// ptr is a helper function to create a pointer to a value.
func ptr[T any](v T) *T {
	return &v
}
