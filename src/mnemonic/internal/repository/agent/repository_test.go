package agent_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twistingmercury/mnemonic/internal/repository"
	"github.com/twistingmercury/mnemonic/internal/repository/agent"
)

// testAgent returns a sample agent for testing.
func testAgent() *agent.Agent {
	return &agent.Agent{
		Name:            "test-agent",
		Description:     "A test agent for unit testing",
		SystemPrompt:    "You are a helpful assistant.",
		Model:           "sonnet",
		AllowedTools:    []string{"read_file", "write_file"},
		RoutingKeywords: []string{"test", "example"},
	}
}

func TestRepository_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		agent     *agent.Agent
		setupMock func(mock pgxmock.PgxPoolIface)
		wantErr   error
	}{
		{
			name:  "successful creation",
			agent: testAgent(),
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("INSERT INTO agents").
					WithArgs(
						"test-agent",
						"A test agent for unit testing",
						"You are a helpful assistant.",
						"sonnet",
						pgxmock.AnyArg(), // allowed_tools JSON
						pgxmock.AnyArg(), // routing_keywords JSON
						pgxmock.AnyArg(), // created_at
						pgxmock.AnyArg(), // updated_at
					).
					WillReturnResult(pgxmock.NewResult("INSERT", 1))
			},
			wantErr: nil,
		},
		{
			name:  "duplicate name returns ErrExists",
			agent: testAgent(),
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("INSERT INTO agents").
					WithArgs(
						pgxmock.AnyArg(),
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
			wantErr: agent.ErrExists,
		},
		{
			name: "empty allowed_tools creates valid JSON",
			agent: &agent.Agent{
				Name:            "empty-tools-agent",
				Description:     "Agent with no tools",
				SystemPrompt:    "Hello",
				Model:           "haiku",
				AllowedTools:    []string{},
				RoutingKeywords: []string{},
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("INSERT INTO agents").
					WithArgs(
						"empty-tools-agent",
						"Agent with no tools",
						"Hello",
						"haiku",
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
					).
					WillReturnResult(pgxmock.NewResult("INSERT", 1))
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()

			tt.setupMock(mock)

			repo := agent.NewRepository(mock)
			err = repo.Create(context.Background(), tt.agent)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
				assert.False(t, tt.agent.CreatedAt.IsZero())
				assert.False(t, tt.agent.UpdatedAt.IsZero())
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepository_Create_CheckConstraintViolation(t *testing.T) {
	t.Parallel()

	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	// Setup mock to return a PgError with code "23514" (check violation)
	checkViolationErr := &pgconn.PgError{
		Code:           "23514",
		ConstraintName: "agents_model_check",
		Message:        "new row violates check constraint",
	}

	mock.ExpectExec("INSERT INTO agents").
		WithArgs(
			pgxmock.AnyArg(),
			pgxmock.AnyArg(),
			pgxmock.AnyArg(),
			pgxmock.AnyArg(),
			pgxmock.AnyArg(),
			pgxmock.AnyArg(),
			pgxmock.AnyArg(),
			pgxmock.AnyArg(),
		).
		WillReturnError(checkViolationErr)

	repo := agent.NewRepository(mock)
	a := &agent.Agent{
		Name:            "invalid-model-agent",
		Description:     "Agent with invalid model value",
		SystemPrompt:    "Hello",
		Model:           "invalid_model",
		AllowedTools:    []string{},
		RoutingKeywords: []string{},
	}

	err = repo.Create(context.Background(), a)

	// Verify the error is returned (not wrapped as a domain error like ErrExists)
	assert.Error(t, err)
	assert.NotErrorIs(t, err, agent.ErrExists)
	assert.NotErrorIs(t, err, agent.ErrNotFound)
	assert.NotErrorIs(t, err, agent.ErrInUse)

	// Verify the original PgError is returned
	var pgErr *pgconn.PgError
	assert.True(t, errors.As(err, &pgErr))
	assert.Equal(t, "23514", pgErr.Code)
	assert.Equal(t, "agents_model_check", pgErr.ConstraintName)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_Update_CheckConstraintViolation(t *testing.T) {
	t.Parallel()

	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	// Setup mock to return a PgError with code "23514" (check violation)
	checkViolationErr := &pgconn.PgError{
		Code:           "23514",
		ConstraintName: "agents_model_check",
		Message:        "new row violates check constraint",
	}

	mock.ExpectExec("UPDATE agents SET").
		WithArgs(
			pgxmock.AnyArg(),
			pgxmock.AnyArg(),
			pgxmock.AnyArg(),
			pgxmock.AnyArg(),
			pgxmock.AnyArg(),
			pgxmock.AnyArg(),
			pgxmock.AnyArg(),
		).
		WillReturnError(checkViolationErr)

	repo := agent.NewRepository(mock)
	a := &agent.Agent{
		Name:            "existing-agent",
		Description:     "Updating to invalid model",
		SystemPrompt:    "Hello",
		Model:           "invalid_model",
		AllowedTools:    []string{},
		RoutingKeywords: []string{},
	}

	err = repo.Update(context.Background(), a)

	// Verify the error is returned (not wrapped as a domain error)
	assert.Error(t, err)
	assert.NotErrorIs(t, err, agent.ErrExists)
	assert.NotErrorIs(t, err, agent.ErrNotFound)
	assert.NotErrorIs(t, err, agent.ErrInUse)

	// Verify the original PgError is returned
	var pgErr *pgconn.PgError
	assert.True(t, errors.As(err, &pgErr))
	assert.Equal(t, "23514", pgErr.Code)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_Get(t *testing.T) {
	t.Parallel()

	now := time.Now()
	allowedToolsJSON, _ := json.Marshal([]string{"read_file", "write_file"})
	routingKeywordsJSON, _ := json.Marshal([]string{"test", "example"})

	tests := []struct {
		name        string
		agentName   string
		setupMock   func(mock pgxmock.PgxPoolIface)
		wantAgent   *agent.Agent
		wantErr     error
		wantErrText string // partial error message to check
	}{
		{
			name:      "successful retrieval",
			agentName: "test-agent",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"name", "description", "system_prompt", "model",
					"allowed_tools", "routing_keywords", "created_at", "updated_at",
				}).AddRow(
					"test-agent",
					"A test agent",
					"You are helpful.",
					"sonnet",
					allowedToolsJSON,
					routingKeywordsJSON,
					now,
					now,
				)
				mock.ExpectQuery("SELECT .* FROM agents").
					WithArgs("test-agent").
					WillReturnRows(rows)
			},
			wantAgent: &agent.Agent{
				Name:            "test-agent",
				Description:     "A test agent",
				SystemPrompt:    "You are helpful.",
				Model:           "sonnet",
				AllowedTools:    []string{"read_file", "write_file"},
				RoutingKeywords: []string{"test", "example"},
				CreatedAt:       now,
				UpdatedAt:       now,
			},
			wantErr: nil,
		},
		{
			name:      "not found returns ErrNotFound",
			agentName: "nonexistent",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT .* FROM agents").
					WithArgs("nonexistent").
					WillReturnError(pgx.ErrNoRows)
			},
			wantAgent: nil,
			wantErr:   agent.ErrNotFound,
		},
		{
			name:      "corrupted allowed_tools JSON returns error with context",
			agentName: "corrupted-agent",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"name", "description", "system_prompt", "model",
					"allowed_tools", "routing_keywords", "created_at", "updated_at",
				}).AddRow(
					"corrupted-agent",
					"Agent with corrupted JSON",
					"Hello",
					"sonnet",
					[]byte(`{invalid json`), // corrupted JSON
					routingKeywordsJSON,
					now,
					now,
				)
				mock.ExpectQuery("SELECT .* FROM agents").
					WithArgs("corrupted-agent").
					WillReturnRows(rows)
			},
			wantAgent:   nil,
			wantErrText: "unmarshaling allowed_tools",
		},
		{
			name:      "corrupted routing_keywords JSON returns error with context",
			agentName: "corrupted-keywords-agent",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"name", "description", "system_prompt", "model",
					"allowed_tools", "routing_keywords", "created_at", "updated_at",
				}).AddRow(
					"corrupted-keywords-agent",
					"Agent with corrupted keywords",
					"Hello",
					"sonnet",
					allowedToolsJSON,
					[]byte(`not valid json`), // corrupted JSON
					now,
					now,
				)
				mock.ExpectQuery("SELECT .* FROM agents").
					WithArgs("corrupted-keywords-agent").
					WillReturnRows(rows)
			},
			wantAgent:   nil,
			wantErrText: "unmarshaling routing_keywords",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()

			tt.setupMock(mock)

			repo := agent.NewRepository(mock)
			a, err := repo.Get(context.Background(), tt.agentName)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, a)
			} else if tt.wantErrText != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrText)
				assert.Nil(t, a)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantAgent.Name, a.Name)
				assert.Equal(t, tt.wantAgent.Description, a.Description)
				assert.Equal(t, tt.wantAgent.SystemPrompt, a.SystemPrompt)
				assert.Equal(t, tt.wantAgent.Model, a.Model)
				assert.Equal(t, tt.wantAgent.AllowedTools, a.AllowedTools)
				assert.Equal(t, tt.wantAgent.RoutingKeywords, a.RoutingKeywords)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepository_Get_ContextCancellation(t *testing.T) {
	t.Parallel()

	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancel

	// Setup mock to return context.Canceled error
	mock.ExpectQuery("SELECT .* FROM agents").
		WithArgs("test-agent").
		WillReturnError(context.Canceled)

	repo := agent.NewRepository(mock)
	a, err := repo.Get(ctx, "test-agent")

	assert.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, a)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_Update(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		agent     *agent.Agent
		setupMock func(mock pgxmock.PgxPoolIface)
		wantErr   error
	}{
		{
			name:  "successful update",
			agent: testAgent(),
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("UPDATE agents SET").
					WithArgs(
						"test-agent",
						pgxmock.AnyArg(), // description
						pgxmock.AnyArg(), // system_prompt
						pgxmock.AnyArg(), // model
						pgxmock.AnyArg(), // allowed_tools
						pgxmock.AnyArg(), // routing_keywords
						pgxmock.AnyArg(), // updated_at
					).
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))
			},
			wantErr: nil,
		},
		{
			name: "not found returns ErrNotFound",
			agent: &agent.Agent{
				Name:            "nonexistent",
				Description:     "Does not exist",
				SystemPrompt:    "Hello",
				Model:           "sonnet",
				AllowedTools:    []string{},
				RoutingKeywords: []string{},
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("UPDATE agents SET").
					WithArgs(
						"nonexistent",
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
					).
					WillReturnResult(pgxmock.NewResult("UPDATE", 0))
			},
			wantErr: agent.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()

			tt.setupMock(mock)

			repo := agent.NewRepository(mock)
			err = repo.Update(context.Background(), tt.agent)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
				assert.False(t, tt.agent.UpdatedAt.IsZero())
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepository_Delete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		agentName string
		setupMock func(mock pgxmock.PgxPoolIface)
		wantErr   error
	}{
		{
			name:      "successful deletion",
			agentName: "test-agent",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("DELETE FROM agents").
					WithArgs("test-agent").
					WillReturnResult(pgxmock.NewResult("DELETE", 1))
			},
			wantErr: nil,
		},
		{
			name:      "not found returns ErrNotFound",
			agentName: "nonexistent",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("DELETE FROM agents").
					WithArgs("nonexistent").
					WillReturnResult(pgxmock.NewResult("DELETE", 0))
			},
			wantErr: agent.ErrNotFound,
		},
		{
			name:      "foreign key violation returns ErrInUse",
			agentName: "in-use-agent",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("DELETE FROM agents").
					WithArgs("in-use-agent").
					WillReturnError(&pgconn.PgError{Code: "23503"})
			},
			wantErr: agent.ErrInUse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()

			tt.setupMock(mock)

			repo := agent.NewRepository(mock)
			err = repo.Delete(context.Background(), tt.agentName)

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
	toolsJSON, _ := json.Marshal([]string{"tool1"})
	keywordsJSON, _ := json.Marshal([]string{"keyword1"})

	tests := []struct {
		name       string
		opts       repository.ListOptions
		setupMock  func(mock pgxmock.PgxPoolIface)
		wantCount  int
		wantTotal  int64
		wantErr    error
		wantAgents []string // agent names in order
	}{
		{
			name: "list all agents without pagination",
			opts: repository.ListOptions{},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				// Single query with window function for total count
				rows := pgxmock.NewRows([]string{
					"name", "description", "system_prompt", "model",
					"allowed_tools", "routing_keywords", "created_at", "updated_at", "total_count",
				}).
					AddRow("agent-a", "First agent", "Prompt A", "sonnet", toolsJSON, keywordsJSON, now, now, int64(2)).
					AddRow("agent-b", "Second agent", "Prompt B", "opus", toolsJSON, keywordsJSON, now, now, int64(2))

				mock.ExpectQuery("SELECT .* FROM agents ORDER BY name").
					WillReturnRows(rows)
			},
			wantCount:  2,
			wantTotal:  2,
			wantAgents: []string{"agent-a", "agent-b"},
		},
		{
			name: "list with limit and offset",
			opts: repository.ListOptions{Limit: 1, Offset: 1},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				// Single query with window function - returns one row with total_count of 2
				rows := pgxmock.NewRows([]string{
					"name", "description", "system_prompt", "model",
					"allowed_tools", "routing_keywords", "created_at", "updated_at", "total_count",
				}).
					AddRow("agent-b", "Second agent", "Prompt B", "opus", toolsJSON, keywordsJSON, now, now, int64(2))

				mock.ExpectQuery("SELECT .* FROM agents ORDER BY name ASC LIMIT").
					WithArgs(1, 1).
					WillReturnRows(rows)
			},
			wantCount:  1,
			wantTotal:  2,
			wantAgents: []string{"agent-b"},
		},
		{
			name: "empty list returns empty slice",
			opts: repository.ListOptions{},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				// Empty result set - no rows returned
				rows := pgxmock.NewRows([]string{
					"name", "description", "system_prompt", "model",
					"allowed_tools", "routing_keywords", "created_at", "updated_at", "total_count",
				})
				mock.ExpectQuery("SELECT .* FROM agents ORDER BY name").
					WillReturnRows(rows)
			},
			wantCount:  0,
			wantTotal:  0,
			wantAgents: []string{},
		},
		{
			name: "list with offset only (no limit)",
			opts: repository.ListOptions{Offset: 5, Limit: 0},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				// Query with OFFSET but no LIMIT - returns remaining agents after offset
				rows := pgxmock.NewRows([]string{
					"name", "description", "system_prompt", "model",
					"allowed_tools", "routing_keywords", "created_at", "updated_at", "total_count",
				}).
					AddRow("agent-f", "Sixth agent", "Prompt F", "sonnet", toolsJSON, keywordsJSON, now, now, int64(10)).
					AddRow("agent-g", "Seventh agent", "Prompt G", "opus", toolsJSON, keywordsJSON, now, now, int64(10))

				mock.ExpectQuery("SELECT .* FROM agents ORDER BY name ASC OFFSET").
					WithArgs(5).
					WillReturnRows(rows)
			},
			wantCount:  2,
			wantTotal:  10,
			wantAgents: []string{"agent-f", "agent-g"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()

			tt.setupMock(mock)

			repo := agent.NewRepository(mock)
			agents, total, err := repo.List(context.Background(), tt.opts)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantTotal, total)
				assert.Len(t, agents, tt.wantCount)

				for i, expectedName := range tt.wantAgents {
					assert.Equal(t, expectedName, agents[i].Name)
				}
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepository_Exists(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		agentName  string
		setupMock  func(mock pgxmock.PgxPoolIface)
		wantExists bool
		wantErr    error
	}{
		{
			name:      "agent exists",
			agentName: "existing-agent",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT EXISTS").
					WithArgs("existing-agent").
					WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))
			},
			wantExists: true,
		},
		{
			name:      "agent does not exist",
			agentName: "nonexistent",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT EXISTS").
					WithArgs("nonexistent").
					WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))
			},
			wantExists: false,
		},
		{
			name:      "database error",
			agentName: "any-agent",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT EXISTS").
					WithArgs("any-agent").
					WillReturnError(errors.New("connection failed"))
			},
			wantExists: false,
			wantErr:    errors.New("connection failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()

			tt.setupMock(mock)

			repo := agent.NewRepository(mock)
			exists, err := repo.Exists(context.Background(), tt.agentName)

			if tt.wantErr != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantExists, exists)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepository_JSONBMarshaling(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		allowedTools    []string
		routingKeywords []string
		setupMock       func(mock pgxmock.PgxPoolIface, expectedToolsJSON, expectedKeywordsJSON []byte)
	}{
		{
			name:            "nil slices marshal to empty arrays",
			allowedTools:    nil,
			routingKeywords: nil,
			setupMock: func(mock pgxmock.PgxPoolIface, expectedToolsJSON, expectedKeywordsJSON []byte) {
				mock.ExpectExec("INSERT INTO agents").
					WithArgs(
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						expectedToolsJSON,
						expectedKeywordsJSON,
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
					).
					WillReturnResult(pgxmock.NewResult("INSERT", 1))
			},
		},
		{
			name:            "multiple tools marshal correctly",
			allowedTools:    []string{"read_file", "write_file", "execute_command"},
			routingKeywords: []string{"go", "golang", "backend"},
			setupMock: func(mock pgxmock.PgxPoolIface, expectedToolsJSON, expectedKeywordsJSON []byte) {
				mock.ExpectExec("INSERT INTO agents").
					WithArgs(
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						expectedToolsJSON,
						expectedKeywordsJSON,
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
					).
					WillReturnResult(pgxmock.NewResult("INSERT", 1))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()

			// Marshal the expected JSON
			toolsJSON, _ := json.Marshal(tt.allowedTools)
			keywordsJSON, _ := json.Marshal(tt.routingKeywords)

			tt.setupMock(mock, toolsJSON, keywordsJSON)

			repo := agent.NewRepository(mock)
			a := &agent.Agent{
				Name:            "json-test-agent",
				Description:     "Testing JSON marshaling",
				SystemPrompt:    "Hello",
				Model:           "inherit",
				AllowedTools:    tt.allowedTools,
				RoutingKeywords: tt.routingKeywords,
			}

			err = repo.Create(context.Background(), a)
			assert.NoError(t, err)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestAgentModel_IsValidModel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		model string
		want  bool
	}{
		{"sonnet", true},
		{"opus", true},
		{"haiku", true},
		{"inherit", true},
		{"invalid", false},
		{"SONNET", false}, // case-sensitive
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, agent.IsValidModel(tt.model))
		})
	}
}

func TestListOptions_Default(t *testing.T) {
	t.Parallel()

	opts := repository.DefaultListOptions()
	assert.Equal(t, 100, opts.Limit)
	assert.Equal(t, 0, opts.Offset)
}
