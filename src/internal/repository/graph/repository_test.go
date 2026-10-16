package graph_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twistingmercury/mnemonic/internal/repository/graph"
)

// --- Mock types ---

// mockResultCollector implements graph.ResultCollector for unit testing.
type mockResultCollector struct {
	records    []*db.Record
	collectErr error
}

func (m *mockResultCollector) Collect(_ context.Context) ([]*db.Record, error) {
	if m.collectErr != nil {
		return nil, m.collectErr
	}
	return m.records, nil
}

// mockCypherRunner implements graph.CypherRunner for unit testing.
// It records the Cypher queries and parameters that were executed.
type mockCypherRunner struct {
	runFn func(ctx context.Context, cypher string, params map[string]any) (graph.ResultCollector, error)
	calls []cypherCall
}

type cypherCall struct {
	cypher string
	params map[string]any
}

func (m *mockCypherRunner) Run(ctx context.Context, cypher string, params map[string]any) (graph.ResultCollector, error) {
	m.calls = append(m.calls, cypherCall{cypher: cypher, params: params})
	if m.runFn != nil {
		return m.runFn(ctx, cypher, params)
	}
	return &mockResultCollector{}, nil
}

// mockSessionExecutor implements graph.SessionExecutor for unit testing.
type mockSessionExecutor struct {
	executeReadFn func(ctx context.Context, work func(runner graph.CypherRunner) (any, error)) (any, error)
	closeFn       func(ctx context.Context) error
}

func (m *mockSessionExecutor) ExecuteRead(ctx context.Context, work func(runner graph.CypherRunner) (any, error)) (any, error) {
	if m.executeReadFn != nil {
		return m.executeReadFn(ctx, work)
	}
	return nil, nil
}

func (m *mockSessionExecutor) Close(_ context.Context) error {
	if m.closeFn != nil {
		return m.closeFn(context.Background())
	}
	return nil
}

// --- Helper functions ---

// newTestRepo creates a Repository with a mock session factory for testing.
func newTestRepo(session *mockSessionExecutor) graph.Repository {
	return graph.ExportNewRepositoryWithFactory(func(_ context.Context) graph.SessionExecutor {
		return session
	})
}

// makeRecord creates a db.Record with the given keys and values.
func makeRecord(keys []string, values []any) *db.Record {
	return &db.Record{
		Keys:   keys,
		Values: values,
	}
}

func TestRepository_GetPatternConcepts(t *testing.T) {
	t.Parallel()

	patternID := uuid.New()

	tests := []struct {
		name      string
		setupMock func() *mockSessionExecutor
		want      []graph.Concept
		wantErr   bool
		errMsg    string
	}{
		{
			name: "successful find with results",
			setupMock: func() *mockSessionExecutor {
				return &mockSessionExecutor{
					executeReadFn: func(ctx context.Context, work func(runner graph.CypherRunner) (any, error)) (any, error) {
						runner := &mockCypherRunner{
							runFn: func(_ context.Context, _ string, _ map[string]any) (graph.ResultCollector, error) {
								return &mockResultCollector{
									records: []*db.Record{
										makeRecord(
											[]string{"name", "type"},
											[]any{"golang", "technology"},
										),
										makeRecord(
											[]string{"name", "type"},
											[]any{"microservices", "practice"},
										),
									},
								}, nil
							},
						}
						return work(runner)
					},
				}
			},
			want: []graph.Concept{
				{Name: "golang", Type: "technology"},
				{Name: "microservices", Type: "practice"},
			},
			wantErr: false,
		},
		{
			name: "no results returns empty slice",
			setupMock: func() *mockSessionExecutor {
				return &mockSessionExecutor{
					executeReadFn: func(ctx context.Context, work func(runner graph.CypherRunner) (any, error)) (any, error) {
						runner := &mockCypherRunner{
							runFn: func(_ context.Context, _ string, _ map[string]any) (graph.ResultCollector, error) {
								return &mockResultCollector{records: []*db.Record{}}, nil
							},
						}
						return work(runner)
					},
				}
			},
			want:    []graph.Concept{},
			wantErr: false,
		},
		{
			name: "database error wraps with context",
			setupMock: func() *mockSessionExecutor {
				return &mockSessionExecutor{
					executeReadFn: func(_ context.Context, _ func(runner graph.CypherRunner) (any, error)) (any, error) {
						return nil, errors.New("read failed")
					},
				}
			},
			want:    nil,
			wantErr: true,
			errMsg:  "getting concepts for pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			session := tt.setupMock()
			repo := newTestRepo(session)

			result, err := repo.GetPatternConcepts(context.Background(), patternID)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, result)
			}
		})
	}
}

func TestRepository_GetPatternConcepts_CypherParams(t *testing.T) {
	t.Parallel()

	patternID := uuid.New()
	var capturedRunner *mockCypherRunner

	session := &mockSessionExecutor{
		executeReadFn: func(ctx context.Context, work func(runner graph.CypherRunner) (any, error)) (any, error) {
			capturedRunner = &mockCypherRunner{
				runFn: func(_ context.Context, _ string, _ map[string]any) (graph.ResultCollector, error) {
					return &mockResultCollector{records: []*db.Record{}}, nil
				},
			}
			return work(capturedRunner)
		},
	}

	repo := newTestRepo(session)
	_, err := repo.GetPatternConcepts(context.Background(), patternID)

	require.NoError(t, err)
	require.Len(t, capturedRunner.calls, 1)
	assert.Contains(t, capturedRunner.calls[0].cypher, "MENTIONED_IN")
	assert.Contains(t, capturedRunner.calls[0].cypher, "c.name AS name")
	assert.Contains(t, capturedRunner.calls[0].cypher, "c.type AS type")
	assert.Equal(t, patternID.String(), capturedRunner.calls[0].params["patternId"])
}

func TestRepository_GetPatternConcepts_CollectError(t *testing.T) {
	t.Parallel()

	patternID := uuid.New()

	session := &mockSessionExecutor{
		executeReadFn: func(ctx context.Context, work func(runner graph.CypherRunner) (any, error)) (any, error) {
			runner := &mockCypherRunner{
				runFn: func(_ context.Context, _ string, _ map[string]any) (graph.ResultCollector, error) {
					return &mockResultCollector{collectErr: errors.New("collect failed")}, nil
				},
			}
			return work(runner)
		},
	}

	repo := newTestRepo(session)
	_, err := repo.GetPatternConcepts(context.Background(), patternID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "collect failed")
}

// --- FindRelatedPatterns tests ---

func TestRepository_FindRelatedPatterns(t *testing.T) {
	t.Parallel()

	patternID := uuid.New()
	relatedID1 := uuid.New()
	relatedID2 := uuid.New()

	tests := []struct {
		name      string
		limit     int
		setupMock func() *mockSessionExecutor
		want      []graph.RelatedPattern
		wantErr   bool
		errMsg    string
	}{
		{
			name:  "successful find with results",
			limit: 10,
			setupMock: func() *mockSessionExecutor {
				return &mockSessionExecutor{
					executeReadFn: func(ctx context.Context, work func(runner graph.CypherRunner) (any, error)) (any, error) {
						runner := &mockCypherRunner{
							runFn: func(_ context.Context, _ string, _ map[string]any) (graph.ResultCollector, error) {
								return &mockResultCollector{
									records: []*db.Record{
										makeRecord(
											[]string{"id", "name", "sharedConcepts", "similarity", "conceptNames"},
											[]any{relatedID1.String(), "pattern-b", int64(5), float64(0.83), []any{"go", "concurrency", "channels", "testing", "errors"}},
										),
										makeRecord(
											[]string{"id", "name", "sharedConcepts", "similarity", "conceptNames"},
											[]any{relatedID2.String(), "pattern-c", int64(2), float64(0.5), []any{"go", "testing"}},
										),
									},
								}, nil
							},
						}
						return work(runner)
					},
				}
			},
			want: []graph.RelatedPattern{
				{ID: relatedID1, Name: "pattern-b", SharedConcepts: 5, Similarity: 0.83, ConceptNames: []string{"go", "concurrency", "channels", "testing", "errors"}},
				{ID: relatedID2, Name: "pattern-c", SharedConcepts: 2, Similarity: 0.5, ConceptNames: []string{"go", "testing"}},
			},
			wantErr: false,
		},
		{
			name:  "no results returns empty slice",
			limit: 10,
			setupMock: func() *mockSessionExecutor {
				return &mockSessionExecutor{
					executeReadFn: func(ctx context.Context, work func(runner graph.CypherRunner) (any, error)) (any, error) {
						runner := &mockCypherRunner{
							runFn: func(_ context.Context, _ string, _ map[string]any) (graph.ResultCollector, error) {
								return &mockResultCollector{records: []*db.Record{}}, nil
							},
						}
						return work(runner)
					},
				}
			},
			want:    []graph.RelatedPattern{},
			wantErr: false,
		},
		{
			name:  "database error wraps with context",
			limit: 10,
			setupMock: func() *mockSessionExecutor {
				return &mockSessionExecutor{
					executeReadFn: func(_ context.Context, _ func(runner graph.CypherRunner) (any, error)) (any, error) {
						return nil, errors.New("read failed")
					},
				}
			},
			want:    nil,
			wantErr: true,
			errMsg:  "finding related patterns",
		},
		{
			name:  "invalid UUID in result returns error",
			limit: 10,
			setupMock: func() *mockSessionExecutor {
				return &mockSessionExecutor{
					executeReadFn: func(ctx context.Context, work func(runner graph.CypherRunner) (any, error)) (any, error) {
						runner := &mockCypherRunner{
							runFn: func(_ context.Context, _ string, _ map[string]any) (graph.ResultCollector, error) {
								return &mockResultCollector{
									records: []*db.Record{
										makeRecord(
											[]string{"id", "name", "sharedConcepts", "similarity", "conceptNames"},
											[]any{"not-a-uuid", "bad-pattern", int64(1), float64(0.5), []any{"go"}},
										),
									},
								}, nil
							},
						}
						return work(runner)
					},
				}
			},
			want:    nil,
			wantErr: true,
			errMsg:  "parsing pattern ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			session := tt.setupMock()
			repo := newTestRepo(session)

			result, err := repo.FindRelatedPatterns(context.Background(), patternID, tt.limit)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, result)
			}
		})
	}
}

func TestRepository_FindRelatedPatterns_CypherParams(t *testing.T) {
	t.Parallel()

	patternID := uuid.New()
	var capturedRunner *mockCypherRunner

	session := &mockSessionExecutor{
		executeReadFn: func(ctx context.Context, work func(runner graph.CypherRunner) (any, error)) (any, error) {
			capturedRunner = &mockCypherRunner{
				runFn: func(_ context.Context, _ string, _ map[string]any) (graph.ResultCollector, error) {
					return &mockResultCollector{records: []*db.Record{}}, nil
				},
			}
			return work(capturedRunner)
		},
	}

	repo := newTestRepo(session)
	_, err := repo.FindRelatedPatterns(context.Background(), patternID, 5)

	require.NoError(t, err)
	require.Len(t, capturedRunner.calls, 1)
	assert.Contains(t, capturedRunner.calls[0].cypher, "RELATED_TO")
	assert.Contains(t, capturedRunner.calls[0].cypher, "similarity")
	assert.Contains(t, capturedRunner.calls[0].cypher, "conceptNames")
	assert.Equal(t, patternID.String(), capturedRunner.calls[0].params["patternId"])
	assert.Equal(t, 5, capturedRunner.calls[0].params["limit"])
}

func TestRepository_FindRelatedPatterns_CollectError(t *testing.T) {
	t.Parallel()

	patternID := uuid.New()

	session := &mockSessionExecutor{
		executeReadFn: func(ctx context.Context, work func(runner graph.CypherRunner) (any, error)) (any, error) {
			runner := &mockCypherRunner{
				runFn: func(_ context.Context, _ string, _ map[string]any) (graph.ResultCollector, error) {
					return &mockResultCollector{collectErr: errors.New("collect failed")}, nil
				},
			}
			return work(runner)
		},
	}

	repo := newTestRepo(session)
	_, err := repo.FindRelatedPatterns(context.Background(), patternID, 10)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "collect failed")
}

// --- CleanupOrphanedConcepts tests ---

func TestRepository_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	session := &mockSessionExecutor{
		executeReadFn: func(_ context.Context, _ func(runner graph.CypherRunner) (any, error)) (any, error) {
			return nil, context.Canceled
		},
	}

	_, err := newTestRepo(session).GetPatternConcepts(ctx, uuid.New())
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}
