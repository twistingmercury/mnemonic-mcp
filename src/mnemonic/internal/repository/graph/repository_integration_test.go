//go:build integration

package graph_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twistingmercury/mnemonic/internal/repository/graph"
)

const (
	// testAgentPrefix is used to identify test-created agents for cleanup.
	testIntegrationAgentPrefix = "test-integration-"
)

// setupNeo4j creates a Neo4j driver and returns a graph.Repository.
// It skips the test if the Neo4j instance is unavailable.
func setupNeo4j(t *testing.T) graph.Repository {
	t.Helper()

	uri := os.Getenv("NEO4J_URI")
	if uri == "" {
		uri = "bolt://localhost:7688"
	}

	user := os.Getenv("NEO4J_USER")
	if user == "" {
		user = "neo4j"
	}

	password := os.Getenv("NEO4J_PASSWORD")
	if password == "" {
		password = "mnemonic_dev"
	}

	database := os.Getenv("NEO4J_DATABASE")
	if database == "" {
		database = "neo4j"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(user, password, ""))
	if err != nil {
		t.Skipf("skipping integration test: unable to create Neo4j driver: %v", err)
	}

	if err := driver.VerifyConnectivity(ctx); err != nil {
		_ = driver.Close(ctx)
		t.Skipf("skipping integration test: Neo4j connectivity check failed: %v", err)
	}

	t.Cleanup(func() {
		_ = driver.Close(context.Background())
	})

	return graph.NewRepository(driver, database)
}

// cleanupTestAgents removes all agents with the test prefix from Neo4j.
func cleanupNeo4jTestAgents(t *testing.T, repo graph.Repository) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Best-effort cleanup: delete any agents we created.
	_ = repo.DeleteAgent(ctx, testIntegrationAgentPrefix+"healthcheck")
	_ = repo.DeleteAgent(ctx, testIntegrationAgentPrefix+"crud")
}

func TestIntegration_HealthCheck(t *testing.T) {
	repo := setupNeo4j(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := repo.HealthCheck(ctx)
	assert.NoError(t, err)
}

func TestIntegration_SyncAgent_DeleteAgent(t *testing.T) {
	repo := setupNeo4j(t)
	cleanupNeo4jTestAgents(t, repo)
	t.Cleanup(func() { cleanupNeo4jTestAgents(t, repo) })

	ctx := context.Background()
	agentName := testIntegrationAgentPrefix + "crud"

	t.Run("sync creates agent", func(t *testing.T) {
		err := repo.SyncAgent(ctx, agentName)
		require.NoError(t, err)
	})

	t.Run("sync is idempotent", func(t *testing.T) {
		err := repo.SyncAgent(ctx, agentName)
		require.NoError(t, err)
	})

	t.Run("delete removes agent", func(t *testing.T) {
		err := repo.DeleteAgent(ctx, agentName)
		require.NoError(t, err)
	})

	t.Run("delete nonexistent is not an error", func(t *testing.T) {
		// Neo4j MATCH + DETACH DELETE is a no-op for nonexistent nodes.
		err := repo.DeleteAgent(ctx, agentName)
		assert.NoError(t, err)
	})
}
