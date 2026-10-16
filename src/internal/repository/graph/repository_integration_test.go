//go:build integration

package graph_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twistingmercury/mnemonic/internal/metrics"
	"github.com/twistingmercury/mnemonic/internal/repository/graph"
	"go.opentelemetry.io/otel/metric/noop"
)

const (
	// testIntegrationPatternPrefix is used to identify test-created patterns for cleanup.
	testIntegrationPatternPrefix = "test-integration-pattern-"

	// testIntegrationConceptPrefix is used to identify test-created concepts for cleanup.
	testIntegrationConceptPrefix = "test-integration-concept-"
)

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// setupNeo4j connects to Neo4j, skipping the test if it is unavailable, and
// removes test data before and after the test. The graph is seeded with
// direct Cypher because writes belong to mnemonic-api and the enricher.
func setupNeo4j(t *testing.T) (*seeder, graph.Repository) {
	t.Helper()

	database := envOr("NEO4J_DATABASE", "neo4j")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	driver, err := neo4j.NewDriverWithContext(
		envOr("NEO4J_URI", "bolt://localhost:7688"),
		neo4j.BasicAuth(envOr("NEO4J_USER", "neo4j"), envOr("NEO4J_PASSWORD", "mnemonic_dev"), ""),
	)
	if err != nil {
		t.Skipf("skipping integration test: unable to create Neo4j driver: %v", err)
	}
	if err := driver.VerifyConnectivity(ctx); err != nil {
		_ = driver.Close(ctx)
		t.Skipf("skipping integration test: Neo4j connectivity check failed: %v", err)
	}

	s := &seeder{t: t, driver: driver, database: database}
	s.cleanup()
	t.Cleanup(func() {
		s.cleanup()
		_ = driver.Close(context.Background())
	})

	dbMetrics, err := metrics.NewDatabase(noop.NewMeterProvider().Meter("test"))
	require.NoError(t, err)
	return s, graph.NewRepository(driver, database, dbMetrics)
}

type seeder struct {
	t        *testing.T
	driver   neo4j.DriverWithContext
	database string
}

func (s *seeder) run(cypher string, params map[string]any) {
	s.t.Helper()
	ctx := context.Background()
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: s.database})
	defer func() { _ = session.Close(ctx) }()
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx, cypher, params)
		return nil, err
	})
	require.NoError(s.t, err)
}

func (s *seeder) cleanup() {
	s.run(`MATCH (p:Pattern) WHERE p.name STARTS WITH $prefix DETACH DELETE p`,
		map[string]any{"prefix": testIntegrationPatternPrefix})
	s.run(`MATCH (c:Concept) WHERE c.name STARTS WITH $prefix DETACH DELETE c`,
		map[string]any{"prefix": testIntegrationConceptPrefix})
}

// pattern creates a Pattern node mentioned by the given concepts.
func (s *seeder) pattern(suffix string, concepts ...string) uuid.UUID {
	s.t.Helper()
	id := uuid.New()
	names := make([]string, len(concepts))
	for i, c := range concepts {
		names[i] = testIntegrationConceptPrefix + c
	}
	s.run(`MERGE (p:Pattern {id: $id}) SET p.name = $name
	       WITH p UNWIND $concepts AS conceptName
	       MERGE (c:Concept {name: conceptName}) ON CREATE SET c.type = 'technology'
	       MERGE (c)-[:MENTIONED_IN]->(p)`,
		map[string]any{"id": id.String(), "name": testIntegrationPatternPrefix + suffix, "concepts": names})
	return id
}

// relate creates a RELATED_TO edge between two patterns.
func (s *seeder) relate(a, b uuid.UUID, similarity float64) {
	s.t.Helper()
	s.run(`MATCH (a:Pattern {id: $a}), (b:Pattern {id: $b})
	       MERGE (a)-[r:RELATED_TO]-(b) SET r.similarity = $similarity`,
		map[string]any{"a": a.String(), "b": b.String(), "similarity": similarity})
}

func TestIntegration_FindRelatedPatterns(t *testing.T) {
	s, repo := setupNeo4j(t)
	ctx := context.Background()

	// pA shares 3 concepts with pB and 1 with pC; pD shares none.
	pA := s.pattern("related-a", "go", "concurrency", "channels", "error-handling")
	pB := s.pattern("related-b", "go", "concurrency", "channels")
	pC := s.pattern("related-c", "go", "testing")
	pD := s.pattern("related-unique", "python", "django")
	s.relate(pA, pB, 0.75)
	s.relate(pA, pC, 0.25)
	s.relate(pB, pC, 1.0/3)

	t.Run("finds patterns ordered by similarity with shared concepts", func(t *testing.T) {
		results, err := repo.FindRelatedPatterns(ctx, pA, 10)
		require.NoError(t, err)
		require.Len(t, results, 2)

		assert.Equal(t, pB, results[0].ID)
		assert.Equal(t, testIntegrationPatternPrefix+"related-b", results[0].Name)
		assert.Equal(t, 3, results[0].SharedConcepts)
		assert.InDelta(t, 0.75, results[0].Similarity, 1e-9)
		assert.ElementsMatch(t, []string{
			testIntegrationConceptPrefix + "go",
			testIntegrationConceptPrefix + "concurrency",
			testIntegrationConceptPrefix + "channels",
		}, results[0].ConceptNames)

		assert.Equal(t, pC, results[1].ID)
		assert.Equal(t, 1, results[1].SharedConcepts)
	})

	t.Run("does not include the source pattern itself", func(t *testing.T) {
		results, err := repo.FindRelatedPatterns(ctx, pA, 10)
		require.NoError(t, err)
		for _, r := range results {
			assert.NotEqual(t, pA, r.ID)
		}
	})

	t.Run("returns no results for pattern without edges", func(t *testing.T) {
		results, err := repo.FindRelatedPatterns(ctx, pD, 10)
		require.NoError(t, err)
		assert.Empty(t, results)
	})

	t.Run("respects limit parameter", func(t *testing.T) {
		results, err := repo.FindRelatedPatterns(ctx, pA, 1)
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, pB, results[0].ID)
	})

	t.Run("returns empty for nonexistent pattern", func(t *testing.T) {
		results, err := repo.FindRelatedPatterns(ctx, uuid.New(), 10)
		require.NoError(t, err)
		assert.Empty(t, results)
	})
}

func TestIntegration_GetPatternConcepts(t *testing.T) {
	s, repo := setupNeo4j(t)
	ctx := context.Background()

	t.Run("returns concepts ordered by name", func(t *testing.T) {
		id := s.pattern("get-concepts", "gamma", "alpha", "beta")

		concepts, err := repo.GetPatternConcepts(ctx, id)
		require.NoError(t, err)
		require.Len(t, concepts, 3)
		assert.Equal(t, testIntegrationConceptPrefix+"alpha", concepts[0].Name)
		assert.Equal(t, testIntegrationConceptPrefix+"beta", concepts[1].Name)
		assert.Equal(t, testIntegrationConceptPrefix+"gamma", concepts[2].Name)
		for _, c := range concepts {
			assert.Equal(t, "technology", c.Type)
		}
	})

	t.Run("returns empty for pattern with no concepts", func(t *testing.T) {
		id := s.pattern("get-concepts-empty")

		concepts, err := repo.GetPatternConcepts(ctx, id)
		require.NoError(t, err)
		assert.Empty(t, concepts)
	})

	t.Run("returns empty for nonexistent pattern", func(t *testing.T) {
		concepts, err := repo.GetPatternConcepts(ctx, uuid.New())
		require.NoError(t, err)
		assert.Empty(t, concepts)
	})
}

func TestIntegration_ContextCancellation(t *testing.T) {
	_, repo := setupNeo4j(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := repo.FindRelatedPatterns(ctx, uuid.New(), 10)
	assert.Error(t, err)
	_, err = repo.GetPatternConcepts(ctx, uuid.New())
	assert.Error(t, err)
}
