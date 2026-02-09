//go:build integration

package agent_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twistingmercury/mnemonic/internal/repository"
	"github.com/twistingmercury/mnemonic/internal/repository/agent"
)

const (
	// testAgentPrefix is used to identify test-created agents for cleanup.
	testAgentPrefix = "test-integration-"
)

// setupTestDB creates a connection pool to the test database.
// It skips the test if the database is unavailable.
func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	connStr := os.Getenv("TEST_DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://mnemonic:mnemonic_dev@localhost:5433/mnemonic?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Skipf("skipping integration test: unable to connect to database: %v", err)
	}

	// Verify connection is working
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("skipping integration test: database ping failed: %v", err)
	}

	// Register cleanup to close pool when test completes
	t.Cleanup(func() {
		pool.Close()
	})

	return pool
}

// cleanupTestAgents removes all agents with the test prefix from the database.
func cleanupTestAgents(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := fmt.Sprintf("DELETE FROM agents WHERE name LIKE '%s%%'", testAgentPrefix)
	_, err := pool.Exec(ctx, query)
	if err != nil {
		t.Logf("warning: failed to cleanup test agents: %v", err)
	}
}

// testIntegrationAgent creates a sample agent for integration testing with a unique name.
func testIntegrationAgent(suffix string) *agent.Agent {
	return &agent.Agent{
		Name:            testAgentPrefix + suffix,
		Description:     "Integration test agent: " + suffix,
		SystemPrompt:    "You are a test assistant for integration testing.",
		Model:           "sonnet",
		AllowedTools:    []string{"read_file", "write_file"},
		RoutingKeywords: []string{"test", "integration"},
	}
}

func TestIntegration_Create(t *testing.T) {
	pool := setupTestDB(t)
	cleanupTestAgents(t, pool)
	t.Cleanup(func() { cleanupTestAgents(t, pool) })

	repo := agent.NewRepository(pool)
	ctx := context.Background()

	tests := []struct {
		name    string
		agent   *agent.Agent
		wantErr bool
	}{
		{
			name:    "successful creation with all fields",
			agent:   testIntegrationAgent("create-full"),
			wantErr: false,
		},
		{
			name: "successful creation with empty tools and keywords",
			agent: &agent.Agent{
				Name:            testAgentPrefix + "create-empty",
				Description:     "Agent with empty arrays",
				SystemPrompt:    "Minimal prompt",
				Model:           "haiku",
				AllowedTools:    []string{},
				RoutingKeywords: []string{},
			},
			wantErr: false,
		},
		{
			name: "successful creation with inherit model",
			agent: &agent.Agent{
				Name:            testAgentPrefix + "create-inherit",
				Description:     "Agent that inherits model",
				SystemPrompt:    "Test prompt",
				Model:           "inherit",
				AllowedTools:    []string{"bash"},
				RoutingKeywords: []string{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Create(ctx, tt.agent)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.False(t, tt.agent.CreatedAt.IsZero(), "CreatedAt should be set")
			assert.False(t, tt.agent.UpdatedAt.IsZero(), "UpdatedAt should be set")

			// Verify by reading back
			retrieved, err := repo.Get(ctx, tt.agent.Name)
			require.NoError(t, err)
			assert.Equal(t, tt.agent.Name, retrieved.Name)
			assert.Equal(t, tt.agent.Description, retrieved.Description)
			assert.Equal(t, tt.agent.SystemPrompt, retrieved.SystemPrompt)
			assert.Equal(t, tt.agent.Model, retrieved.Model)
			assert.Equal(t, tt.agent.AllowedTools, retrieved.AllowedTools)
			assert.Equal(t, tt.agent.RoutingKeywords, retrieved.RoutingKeywords)
		})
	}
}

func TestIntegration_Get(t *testing.T) {
	pool := setupTestDB(t)
	cleanupTestAgents(t, pool)
	t.Cleanup(func() { cleanupTestAgents(t, pool) })

	repo := agent.NewRepository(pool)
	ctx := context.Background()

	// Create a test agent first
	testAgent := testIntegrationAgent("get-test")
	require.NoError(t, repo.Create(ctx, testAgent))

	t.Run("retrieves existing agent", func(t *testing.T) {
		retrieved, err := repo.Get(ctx, testAgent.Name)

		require.NoError(t, err)
		assert.Equal(t, testAgent.Name, retrieved.Name)
		assert.Equal(t, testAgent.Description, retrieved.Description)
		assert.Equal(t, testAgent.SystemPrompt, retrieved.SystemPrompt)
		assert.Equal(t, testAgent.Model, retrieved.Model)
		assert.Equal(t, testAgent.AllowedTools, retrieved.AllowedTools)
		assert.Equal(t, testAgent.RoutingKeywords, retrieved.RoutingKeywords)
		assert.False(t, retrieved.CreatedAt.IsZero())
		assert.False(t, retrieved.UpdatedAt.IsZero())
	})

	t.Run("returns ErrNotFound for nonexistent agent", func(t *testing.T) {
		retrieved, err := repo.Get(ctx, testAgentPrefix+"nonexistent")

		assert.ErrorIs(t, err, agent.ErrNotFound)
		assert.Nil(t, retrieved)
	})
}

func TestIntegration_Update(t *testing.T) {
	pool := setupTestDB(t)
	cleanupTestAgents(t, pool)
	t.Cleanup(func() { cleanupTestAgents(t, pool) })

	repo := agent.NewRepository(pool)
	ctx := context.Background()

	// Create a test agent first
	testAgent := testIntegrationAgent("update-test")
	require.NoError(t, repo.Create(ctx, testAgent))
	originalCreatedAt := testAgent.CreatedAt

	t.Run("updates existing agent", func(t *testing.T) {
		// Modify the agent
		testAgent.Description = "Updated description"
		testAgent.SystemPrompt = "Updated system prompt"
		testAgent.Model = "opus"
		testAgent.AllowedTools = []string{"new_tool"}
		testAgent.RoutingKeywords = []string{"updated", "keywords"}

		err := repo.Update(ctx, testAgent)
		require.NoError(t, err)
		assert.False(t, testAgent.UpdatedAt.IsZero())

		// Verify by reading back
		retrieved, err := repo.Get(ctx, testAgent.Name)
		require.NoError(t, err)
		assert.Equal(t, "Updated description", retrieved.Description)
		assert.Equal(t, "Updated system prompt", retrieved.SystemPrompt)
		assert.Equal(t, "opus", retrieved.Model)
		assert.Equal(t, []string{"new_tool"}, retrieved.AllowedTools)
		assert.Equal(t, []string{"updated", "keywords"}, retrieved.RoutingKeywords)
		assert.Equal(t, originalCreatedAt.UTC().Truncate(time.Microsecond),
			retrieved.CreatedAt.UTC().Truncate(time.Microsecond),
			"CreatedAt should not change")
		assert.True(t, retrieved.UpdatedAt.After(retrieved.CreatedAt) ||
			retrieved.UpdatedAt.Equal(retrieved.CreatedAt),
			"UpdatedAt should be >= CreatedAt")
	})

	t.Run("returns ErrNotFound for nonexistent agent", func(t *testing.T) {
		nonexistent := &agent.Agent{
			Name:            testAgentPrefix + "nonexistent-update",
			Description:     "Does not exist",
			SystemPrompt:    "Test",
			Model:           "sonnet",
			AllowedTools:    []string{},
			RoutingKeywords: []string{},
		}

		err := repo.Update(ctx, nonexistent)
		assert.ErrorIs(t, err, agent.ErrNotFound)
	})
}

func TestIntegration_Delete(t *testing.T) {
	pool := setupTestDB(t)
	cleanupTestAgents(t, pool)
	t.Cleanup(func() { cleanupTestAgents(t, pool) })

	repo := agent.NewRepository(pool)
	ctx := context.Background()

	t.Run("deletes existing agent", func(t *testing.T) {
		// Create a test agent
		testAgent := testIntegrationAgent("delete-test")
		require.NoError(t, repo.Create(ctx, testAgent))

		// Verify it exists
		exists, err := repo.Exists(ctx, testAgent.Name)
		require.NoError(t, err)
		require.True(t, exists)

		// Delete it
		err = repo.Delete(ctx, testAgent.Name)
		require.NoError(t, err)

		// Verify it's gone
		exists, err = repo.Exists(ctx, testAgent.Name)
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("returns ErrNotFound for nonexistent agent", func(t *testing.T) {
		err := repo.Delete(ctx, testAgentPrefix+"nonexistent-delete")
		assert.ErrorIs(t, err, agent.ErrNotFound)
	})
}

func TestIntegration_List(t *testing.T) {
	pool := setupTestDB(t)
	cleanupTestAgents(t, pool)
	t.Cleanup(func() { cleanupTestAgents(t, pool) })

	repo := agent.NewRepository(pool)
	ctx := context.Background()

	// Create multiple test agents with names that sort predictably
	agents := []*agent.Agent{
		testIntegrationAgent("list-a"),
		testIntegrationAgent("list-b"),
		testIntegrationAgent("list-c"),
		testIntegrationAgent("list-d"),
		testIntegrationAgent("list-e"),
	}

	for _, a := range agents {
		require.NoError(t, repo.Create(ctx, a))
	}

	t.Run("lists all agents without pagination", func(t *testing.T) {
		result, total, err := repo.List(ctx, repository.ListOptions{})

		require.NoError(t, err)
		// Total includes all agents in DB, not just test agents
		assert.GreaterOrEqual(t, total, int64(5))

		// Verify our test agents are in the result
		names := make(map[string]bool)
		for _, a := range result {
			names[a.Name] = true
		}
		for _, expected := range agents {
			assert.True(t, names[expected.Name], "expected agent %s in result", expected.Name)
		}
	})

	t.Run("lists with limit", func(t *testing.T) {
		result, total, err := repo.List(ctx, repository.ListOptions{Limit: 2})

		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.GreaterOrEqual(t, total, int64(5))
	})

	t.Run("lists with limit and offset", func(t *testing.T) {
		// Get first page
		page1, total1, err := repo.List(ctx, repository.ListOptions{Limit: 2, Offset: 0})
		require.NoError(t, err)
		assert.Len(t, page1, 2)

		// Get second page
		page2, total2, err := repo.List(ctx, repository.ListOptions{Limit: 2, Offset: 2})
		require.NoError(t, err)
		assert.Len(t, page2, 2)

		// Totals should be the same
		assert.Equal(t, total1, total2)

		// Pages should have different agents
		for _, a1 := range page1 {
			for _, a2 := range page2 {
				assert.NotEqual(t, a1.Name, a2.Name,
					"pages should not overlap")
			}
		}
	})

	t.Run("lists with offset only", func(t *testing.T) {
		result, total, err := repo.List(ctx, repository.ListOptions{Offset: 2})

		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, int64(5))
		// Should return all agents after offset
		assert.GreaterOrEqual(t, len(result), 3)
	})

	t.Run("returns ordered results", func(t *testing.T) {
		result, _, err := repo.List(ctx, repository.ListOptions{})

		require.NoError(t, err)
		require.GreaterOrEqual(t, len(result), 2)

		// Verify alphabetical ordering
		for i := 1; i < len(result); i++ {
			assert.LessOrEqual(t, result[i-1].Name, result[i].Name,
				"results should be ordered by name")
		}
	})
}

func TestIntegration_Exists(t *testing.T) {
	pool := setupTestDB(t)
	cleanupTestAgents(t, pool)
	t.Cleanup(func() { cleanupTestAgents(t, pool) })

	repo := agent.NewRepository(pool)
	ctx := context.Background()

	// Create a test agent
	testAgent := testIntegrationAgent("exists-test")
	require.NoError(t, repo.Create(ctx, testAgent))

	t.Run("returns true for existing agent", func(t *testing.T) {
		exists, err := repo.Exists(ctx, testAgent.Name)

		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("returns false for nonexistent agent", func(t *testing.T) {
		exists, err := repo.Exists(ctx, testAgentPrefix+"nonexistent-exists")

		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestIntegration_CreateDuplicate(t *testing.T) {
	pool := setupTestDB(t)
	cleanupTestAgents(t, pool)
	t.Cleanup(func() { cleanupTestAgents(t, pool) })

	repo := agent.NewRepository(pool)
	ctx := context.Background()

	// Create a test agent
	testAgent := testIntegrationAgent("duplicate-test")
	require.NoError(t, repo.Create(ctx, testAgent))

	t.Run("returns ErrExists for duplicate name", func(t *testing.T) {
		duplicate := &agent.Agent{
			Name:            testAgent.Name, // Same name
			Description:     "Different description",
			SystemPrompt:    "Different prompt",
			Model:           "haiku",
			AllowedTools:    []string{},
			RoutingKeywords: []string{},
		}

		err := repo.Create(ctx, duplicate)
		assert.ErrorIs(t, err, agent.ErrExists)
	})
}

func TestIntegration_Constraints(t *testing.T) {
	pool := setupTestDB(t)
	cleanupTestAgents(t, pool)
	t.Cleanup(func() { cleanupTestAgents(t, pool) })

	repo := agent.NewRepository(pool)
	ctx := context.Background()

	t.Run("rejects invalid name format - starts with number", func(t *testing.T) {
		a := &agent.Agent{
			Name:            "123-invalid-name",
			Description:     "Test",
			SystemPrompt:    "Test",
			Model:           "sonnet",
			AllowedTools:    []string{},
			RoutingKeywords: []string{},
		}

		err := repo.Create(ctx, a)
		require.Error(t, err)

		var pgErr *pgconn.PgError
		assert.True(t, errors.As(err, &pgErr))
		assert.Equal(t, "23514", pgErr.Code, "expected check constraint violation")
	})

	t.Run("rejects invalid name format - uppercase letters", func(t *testing.T) {
		a := &agent.Agent{
			Name:            "Invalid-Name",
			Description:     "Test",
			SystemPrompt:    "Test",
			Model:           "sonnet",
			AllowedTools:    []string{},
			RoutingKeywords: []string{},
		}

		err := repo.Create(ctx, a)
		require.Error(t, err)

		var pgErr *pgconn.PgError
		assert.True(t, errors.As(err, &pgErr))
		assert.Equal(t, "23514", pgErr.Code, "expected check constraint violation")
	})

	t.Run("rejects invalid name format - starts with hyphen", func(t *testing.T) {
		a := &agent.Agent{
			Name:            "-invalid-start",
			Description:     "Test",
			SystemPrompt:    "Test",
			Model:           "sonnet",
			AllowedTools:    []string{},
			RoutingKeywords: []string{},
		}

		err := repo.Create(ctx, a)
		require.Error(t, err)

		var pgErr *pgconn.PgError
		assert.True(t, errors.As(err, &pgErr))
		assert.Equal(t, "23514", pgErr.Code, "expected check constraint violation")
	})

	t.Run("rejects invalid model value", func(t *testing.T) {
		a := &agent.Agent{
			Name:            testAgentPrefix + "invalid-model",
			Description:     "Test",
			SystemPrompt:    "Test",
			Model:           "invalid_model",
			AllowedTools:    []string{},
			RoutingKeywords: []string{},
		}

		err := repo.Create(ctx, a)
		require.Error(t, err)

		var pgErr *pgconn.PgError
		assert.True(t, errors.As(err, &pgErr))
		assert.Equal(t, "23514", pgErr.Code, "expected check constraint violation")
	})

	t.Run("accepts all valid model values", func(t *testing.T) {
		validModels := []string{"sonnet", "opus", "haiku", "inherit"}

		for _, model := range validModels {
			a := &agent.Agent{
				Name:            testAgentPrefix + "model-" + model,
				Description:     "Test with " + model,
				SystemPrompt:    "Test",
				Model:           model,
				AllowedTools:    []string{},
				RoutingKeywords: []string{},
			}

			err := repo.Create(ctx, a)
			assert.NoError(t, err, "model %q should be accepted", model)
		}
	})

	t.Run("accepts valid name with numbers after first letter", func(t *testing.T) {
		a := &agent.Agent{
			Name:            testAgentPrefix + "agent-v2-test",
			Description:     "Test",
			SystemPrompt:    "Test",
			Model:           "sonnet",
			AllowedTools:    []string{},
			RoutingKeywords: []string{},
		}

		err := repo.Create(ctx, a)
		assert.NoError(t, err)
	})
}

func TestIntegration_ContextCancellation(t *testing.T) {
	pool := setupTestDB(t)
	cleanupTestAgents(t, pool)
	t.Cleanup(func() { cleanupTestAgents(t, pool) })

	repo := agent.NewRepository(pool)

	t.Run("respects cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := repo.Get(ctx, testAgentPrefix+"any")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, context.Canceled),
			"expected context.Canceled error, got: %v", err)
	})

	t.Run("respects context timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		// Give time for the context to expire
		time.Sleep(1 * time.Millisecond)

		_, err := repo.Get(ctx, testAgentPrefix+"any")
		assert.Error(t, err)
	})
}

func TestIntegration_ConcurrentOperations(t *testing.T) {
	pool := setupTestDB(t)
	cleanupTestAgents(t, pool)
	t.Cleanup(func() { cleanupTestAgents(t, pool) })

	repo := agent.NewRepository(pool)
	ctx := context.Background()

	t.Run("handles concurrent creates", func(t *testing.T) {
		const numAgents = 10
		errChan := make(chan error, numAgents)

		for i := 0; i < numAgents; i++ {
			go func(idx int) {
				a := &agent.Agent{
					Name:            fmt.Sprintf("%sconcurrent-%d", testAgentPrefix, idx),
					Description:     fmt.Sprintf("Concurrent agent %d", idx),
					SystemPrompt:    "Test",
					Model:           "sonnet",
					AllowedTools:    []string{},
					RoutingKeywords: []string{},
				}
				errChan <- repo.Create(ctx, a)
			}(i)
		}

		// Collect results
		var createErrors []error
		for i := 0; i < numAgents; i++ {
			if err := <-errChan; err != nil {
				createErrors = append(createErrors, err)
			}
		}

		assert.Empty(t, createErrors, "all concurrent creates should succeed")

		// Verify all were created
		for i := 0; i < numAgents; i++ {
			name := fmt.Sprintf("%sconcurrent-%d", testAgentPrefix, i)
			exists, err := repo.Exists(ctx, name)
			require.NoError(t, err)
			assert.True(t, exists, "agent %s should exist", name)
		}
	})
}

func TestIntegration_LargeSystemPrompt(t *testing.T) {
	pool := setupTestDB(t)
	cleanupTestAgents(t, pool)
	t.Cleanup(func() { cleanupTestAgents(t, pool) })

	repo := agent.NewRepository(pool)
	ctx := context.Background()

	t.Run("accepts system prompt up to 50KB", func(t *testing.T) {
		// Create a 50KB prompt (just under the limit)
		largePrompt := make([]byte, 50*1024)
		for i := range largePrompt {
			largePrompt[i] = 'a' + byte(i%26)
		}

		a := &agent.Agent{
			Name:            testAgentPrefix + "large-prompt",
			Description:     "Agent with large prompt",
			SystemPrompt:    string(largePrompt),
			Model:           "sonnet",
			AllowedTools:    []string{},
			RoutingKeywords: []string{},
		}

		err := repo.Create(ctx, a)
		assert.NoError(t, err)

		// Verify it was stored correctly
		retrieved, err := repo.Get(ctx, a.Name)
		require.NoError(t, err)
		assert.Equal(t, len(largePrompt), len(retrieved.SystemPrompt))
	})

	t.Run("rejects system prompt over 50KB", func(t *testing.T) {
		// Create a prompt over 50KB
		oversizedPrompt := make([]byte, 51*1024+1)
		for i := range oversizedPrompt {
			oversizedPrompt[i] = 'a' + byte(i%26)
		}

		a := &agent.Agent{
			Name:            testAgentPrefix + "oversized-prompt",
			Description:     "Agent with oversized prompt",
			SystemPrompt:    string(oversizedPrompt),
			Model:           "sonnet",
			AllowedTools:    []string{},
			RoutingKeywords: []string{},
		}

		err := repo.Create(ctx, a)
		require.Error(t, err)

		var pgErr *pgconn.PgError
		assert.True(t, errors.As(err, &pgErr))
		assert.Equal(t, "23514", pgErr.Code, "expected check constraint violation")
	})
}

func TestIntegration_JSONBFields(t *testing.T) {
	pool := setupTestDB(t)
	cleanupTestAgents(t, pool)
	t.Cleanup(func() { cleanupTestAgents(t, pool) })

	repo := agent.NewRepository(pool)
	ctx := context.Background()

	t.Run("handles complex allowed_tools array", func(t *testing.T) {
		tools := []string{
			"read_file",
			"write_file",
			"execute_command",
			"search_code",
			"tool-with-special-chars_v2",
		}

		a := &agent.Agent{
			Name:            testAgentPrefix + "complex-tools",
			Description:     "Agent with many tools",
			SystemPrompt:    "Test",
			Model:           "sonnet",
			AllowedTools:    tools,
			RoutingKeywords: []string{},
		}

		err := repo.Create(ctx, a)
		require.NoError(t, err)

		retrieved, err := repo.Get(ctx, a.Name)
		require.NoError(t, err)
		assert.Equal(t, tools, retrieved.AllowedTools)
	})

	t.Run("handles complex routing_keywords array", func(t *testing.T) {
		keywords := []string{
			"go",
			"golang",
			"backend",
			"api",
			"microservices",
			"keyword-with-hyphen",
		}

		a := &agent.Agent{
			Name:            testAgentPrefix + "complex-keywords",
			Description:     "Agent with many keywords",
			SystemPrompt:    "Test",
			Model:           "sonnet",
			AllowedTools:    []string{},
			RoutingKeywords: keywords,
		}

		err := repo.Create(ctx, a)
		require.NoError(t, err)

		retrieved, err := repo.Get(ctx, a.Name)
		require.NoError(t, err)
		assert.Equal(t, keywords, retrieved.RoutingKeywords)
	})

	t.Run("rejects nil slices due to constraint", func(t *testing.T) {
		// Note: Go's json.Marshal produces "null" for nil slices, which violates
		// the database constraint that requires JSONB arrays. This test verifies
		// that the constraint is enforced correctly. Callers should initialize
		// slices to empty ([]string{}) rather than nil.
		a := &agent.Agent{
			Name:            testAgentPrefix + "nil-arrays",
			Description:     "Agent with nil arrays",
			SystemPrompt:    "Test",
			Model:           "sonnet",
			AllowedTools:    nil,
			RoutingKeywords: nil,
		}

		err := repo.Create(ctx, a)
		require.Error(t, err)

		var pgErr *pgconn.PgError
		assert.True(t, errors.As(err, &pgErr))
		assert.Equal(t, "23514", pgErr.Code, "expected check constraint violation")
	})

	t.Run("handles empty slices correctly", func(t *testing.T) {
		// Empty slices ([]string{}) marshal to "[]" which satisfies the constraint
		a := &agent.Agent{
			Name:            testAgentPrefix + "empty-arrays",
			Description:     "Agent with empty arrays",
			SystemPrompt:    "Test",
			Model:           "sonnet",
			AllowedTools:    []string{},
			RoutingKeywords: []string{},
		}

		err := repo.Create(ctx, a)
		require.NoError(t, err)

		retrieved, err := repo.Get(ctx, a.Name)
		require.NoError(t, err)
		assert.NotNil(t, retrieved.AllowedTools)
		assert.NotNil(t, retrieved.RoutingKeywords)
		assert.Empty(t, retrieved.AllowedTools)
		assert.Empty(t, retrieved.RoutingKeywords)
	})
}
