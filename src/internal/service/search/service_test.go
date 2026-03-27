package search_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	chunkrepo "github.com/twistingmercury/mnemonic/internal/repository/chunk"
	graphrepo "github.com/twistingmercury/mnemonic/internal/repository/graph"
	"github.com/twistingmercury/mnemonic/internal/repository/pattern"
	"github.com/twistingmercury/mnemonic/internal/service"
	openaisvc "github.com/twistingmercury/mnemonic/internal/service/openai"
	"github.com/twistingmercury/mnemonic/internal/service/search"

	repository "github.com/twistingmercury/mnemonic/internal/repository"
)

// --- Mock: openaisvc.EmbeddingService ---

type mockEmbeddingService struct {
	mock.Mock
}

func (m *mockEmbeddingService) Embed(ctx context.Context, text string) ([]float32, error) {
	args := m.Called(ctx, text)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]float32), args.Error(1)
}

// --- Mock: pattern.Repository ---

type mockPatternRepo struct {
	mock.Mock
}

func (m *mockPatternRepo) Create(ctx context.Context, p *pattern.Pattern) error {
	args := m.Called(ctx, p)
	return args.Error(0)
}

func (m *mockPatternRepo) Get(ctx context.Context, id uuid.UUID) (*pattern.Pattern, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pattern.Pattern), args.Error(1)
}

func (m *mockPatternRepo) GetByName(ctx context.Context, name string) (*pattern.Pattern, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pattern.Pattern), args.Error(1)
}

func (m *mockPatternRepo) Update(ctx context.Context, p *pattern.Pattern) error {
	args := m.Called(ctx, p)
	return args.Error(0)
}

func (m *mockPatternRepo) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockPatternRepo) List(ctx context.Context, filter pattern.Filter, opts repository.ListOptions) ([]*pattern.Pattern, int64, error) {
	args := m.Called(ctx, filter, opts)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*pattern.Pattern), args.Get(1).(int64), args.Error(2)
}

func (m *mockPatternRepo) UpdateEmbedding(ctx context.Context, id uuid.UUID, embedding []float32) error {
	args := m.Called(ctx, id, embedding)
	return args.Error(0)
}

func (m *mockPatternRepo) UpdateEnrichmentStatus(ctx context.Context, id uuid.UUID, status string, errMsg *string) error {
	args := m.Called(ctx, id, status, errMsg)
	return args.Error(0)
}

// FindSimilar is required by pattern.Repository but is not called by the search
// service (which uses chunkRepo.FindSimilar instead).
func (m *mockPatternRepo) FindSimilar(ctx context.Context, embedding []float32, opts pattern.SimilarityOptions) ([]*pattern.Match, error) {
	args := m.Called(ctx, embedding, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*pattern.Match), args.Error(1)
}

func (m *mockPatternRepo) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}

func (m *mockPatternRepo) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*pattern.Pattern, error) {
	args := m.Called(ctx, ids)
	if v := args.Get(0); v != nil {
		return v.([]*pattern.Pattern), args.Error(1)
	}
	return nil, args.Error(1)
}

// --- Mock: chunkrepo.Repository ---

type mockChunkRepo struct {
	mock.Mock
}

func (m *mockChunkRepo) Create(ctx context.Context, c *chunkrepo.Chunk) error {
	return m.Called(ctx, c).Error(0)
}

func (m *mockChunkRepo) CreateBatch(ctx context.Context, chunks []*chunkrepo.Chunk) error {
	return m.Called(ctx, chunks).Error(0)
}

func (m *mockChunkRepo) Get(ctx context.Context, id uuid.UUID) (*chunkrepo.Chunk, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*chunkrepo.Chunk), args.Error(1)
}

func (m *mockChunkRepo) ListByPatternID(ctx context.Context, patternID uuid.UUID) ([]*chunkrepo.Chunk, error) {
	args := m.Called(ctx, patternID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*chunkrepo.Chunk), args.Error(1)
}

func (m *mockChunkRepo) DeleteByPatternID(ctx context.Context, patternID uuid.UUID) error {
	return m.Called(ctx, patternID).Error(0)
}

func (m *mockChunkRepo) UpdateEmbedding(ctx context.Context, id uuid.UUID, embedding []float32) error {
	return m.Called(ctx, id, embedding).Error(0)
}

func (m *mockChunkRepo) UpdateEnrichmentStatus(ctx context.Context, id uuid.UUID, status string, errMsg *string) error {
	return m.Called(ctx, id, status, errMsg).Error(0)
}

func (m *mockChunkRepo) FindSimilar(ctx context.Context, embedding []float32, opts chunkrepo.SimilarityOptions) ([]*chunkrepo.Match, error) {
	args := m.Called(ctx, embedding, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*chunkrepo.Match), args.Error(1)
}

func (m *mockChunkRepo) AllEnrichedForPattern(ctx context.Context, patternID uuid.UUID) (bool, error) {
	args := m.Called(ctx, patternID)
	return args.Bool(0), args.Error(1)
}

func (m *mockChunkRepo) AnyFailedForPattern(ctx context.Context, patternID uuid.UUID) (bool, error) {
	args := m.Called(ctx, patternID)
	return args.Bool(0), args.Error(1)
}

// --- Mock: graphrepo.Repository ---

type mockGraphRepo struct {
	mock.Mock
}

func (m *mockGraphRepo) SyncPattern(ctx context.Context, p *graphrepo.Pattern) error {
	return m.Called(ctx, p).Error(0)
}

func (m *mockGraphRepo) DeletePattern(ctx context.Context, patternID uuid.UUID) error {
	return m.Called(ctx, patternID).Error(0)
}

func (m *mockGraphRepo) SyncConcepts(ctx context.Context, patternID uuid.UUID, concepts []graphrepo.Concept) error {
	return m.Called(ctx, patternID, concepts).Error(0)
}

func (m *mockGraphRepo) ComputeRelatedToEdges(ctx context.Context, patternID uuid.UUID, minSimilarity float64) error {
	return m.Called(ctx, patternID, minSimilarity).Error(0)
}

func (m *mockGraphRepo) GetPatternConcepts(ctx context.Context, patternID uuid.UUID) ([]graphrepo.Concept, error) {
	args := m.Called(ctx, patternID)
	if v := args.Get(0); v != nil {
		return v.([]graphrepo.Concept), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockGraphRepo) FindRelatedPatterns(ctx context.Context, patternID uuid.UUID, limit int) ([]graphrepo.RelatedPattern, error) {
	args := m.Called(ctx, patternID, limit)
	if v := args.Get(0); v != nil {
		return v.([]graphrepo.RelatedPattern), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockGraphRepo) CleanupOrphanedConcepts(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockGraphRepo) HealthCheck(ctx context.Context) error {
	return m.Called(ctx).Error(0)
}

// --- Helpers ---

var testEmbedding = []float32{0.1, 0.2, 0.3}

func newTestService(embSvc *mockEmbeddingService, patternRepo *mockPatternRepo, chunkRepo *mockChunkRepo) search.Service {
	logger := zerolog.Nop()
	return search.New(embSvc, patternRepo, chunkRepo, nil, logger)
}

func newTestServiceWithGraph(embSvc *mockEmbeddingService, patternRepo *mockPatternRepo, chunkRepo *mockChunkRepo, graphRepo *mockGraphRepo) search.Service {
	logger := zerolog.Nop()
	return search.New(embSvc, patternRepo, chunkRepo, graphRepo, logger)
}

func testChunkMatch(patternID uuid.UUID, patternName string, similarity float64) *chunkrepo.Match {
	return &chunkrepo.Match{
		PatternID:    patternID,
		PatternName:  patternName,
		EntityType:   "go-pattern",
		Language:     "go",
		Domain:       "backend",
		Tags:         []string{"go", "testing"},
		SectionTitle: "Overview",
		ChunkIndex:   0,
		Content:      "test content for " + patternName,
		Similarity:   similarity,
	}
}

// ---------- SearchPatterns ----------

func TestSearchPatterns_HappyPath(t *testing.T) {
	t.Parallel()

	embSvc := new(mockEmbeddingService)
	patternRepo := new(mockPatternRepo)
	chunkRepo := new(mockChunkRepo)
	svc := newTestService(embSvc, patternRepo, chunkRepo)

	id1 := uuid.New()
	id2 := uuid.New()

	embSvc.On("Embed", mock.Anything, "error handling in Go").Return(testEmbedding, nil)
	chunkRepo.On("FindSimilar", mock.Anything, testEmbedding, chunkrepo.SimilarityOptions{
		MinSimilarity: 0.7,
		MaxResults:    10,
	}).Return([]*chunkrepo.Match{
		testChunkMatch(id1, "go-error-handling", 0.92),
		testChunkMatch(id2, "go-error-wrapping", 0.85),
	}, nil)

	result, err := svc.SearchPatterns(context.Background(), search.SearchOptions{
		Query:     "error handling in Go",
		Limit:     10,
		Threshold: 0.7,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "error handling in Go", result.Query)
	assert.Len(t, result.Matches, 2)
	assert.Equal(t, 2, result.TotalCandidates)
	assert.Greater(t, result.SearchDurationMs, int64(-1))
	assert.Equal(t, id1, result.Matches[0].PatternID)
	assert.Equal(t, "go-error-handling", result.Matches[0].PatternName)
	assert.InDelta(t, 0.92, result.Matches[0].Similarity, 0.001)

	embSvc.AssertExpectations(t)
	chunkRepo.AssertExpectations(t)
}

func TestSearchPatterns_PassesLanguageAndDomainFilters(t *testing.T) {
	t.Parallel()

	embSvc := new(mockEmbeddingService)
	patternRepo := new(mockPatternRepo)
	chunkRepo := new(mockChunkRepo)
	svc := newTestService(embSvc, patternRepo, chunkRepo)

	patternID := uuid.New()

	embSvc.On("Embed", mock.Anything, "testing patterns").Return(testEmbedding, nil)
	chunkRepo.On("FindSimilar", mock.Anything, testEmbedding, chunkrepo.SimilarityOptions{
		MinSimilarity: 0.7,
		MaxResults:    10,
		Language:      "go",
		Domain:        "backend",
	}).Return([]*chunkrepo.Match{testChunkMatch(patternID, "go-testing", 0.88)}, nil)

	result, err := svc.SearchPatterns(context.Background(), search.SearchOptions{
		Query:     "testing patterns",
		Limit:     10,
		Threshold: 0.7,
		Language:  "go",
		Domain:    "backend",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Matches, 1)
	assert.Equal(t, patternID, result.Matches[0].PatternID)

	embSvc.AssertExpectations(t)
	chunkRepo.AssertExpectations(t)
	patternRepo.AssertNotCalled(t, "GetPatternIDsByAgent")
}

func TestSearchPatterns_PassesTagsLanguageDomainTogether(t *testing.T) {
	t.Parallel()

	embSvc := new(mockEmbeddingService)
	patternRepo := new(mockPatternRepo)
	chunkRepo := new(mockChunkRepo)
	svc := newTestService(embSvc, patternRepo, chunkRepo)

	embSvc.On("Embed", mock.Anything, "platform patterns").Return(testEmbedding, nil)
	chunkRepo.On("FindSimilar", mock.Anything, testEmbedding, chunkrepo.SimilarityOptions{
		MinSimilarity: 0.65,
		MaxResults:    25,
		Tags:          []string{"observability", "go"},
		Language:      "go",
		Domain:        "platform",
	}).Return([]*chunkrepo.Match{}, nil)

	result, err := svc.SearchPatterns(context.Background(), search.SearchOptions{
		Query:     "platform patterns",
		Limit:     25,
		Threshold: 0.65,
		Tags:      []string{"observability", "go"},
		Language:  "go",
		Domain:    "platform",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.Matches)
	assert.Equal(t, "platform patterns", result.Query)
	assert.Equal(t, 0, result.TotalCandidates)

	chunkRepo.AssertExpectations(t)
	patternRepo.AssertNotCalled(t, "GetPatternIDsByAgent")
}

func TestSearchPatterns_DoesNotUseLegacyPatternIDPrefilter(t *testing.T) {
	t.Parallel()

	embSvc := new(mockEmbeddingService)
	patternRepo := new(mockPatternRepo)
	chunkRepo := new(mockChunkRepo)
	svc := newTestService(embSvc, patternRepo, chunkRepo)

	patternID := uuid.New()
	embSvc.On("Embed", mock.Anything, "legacy prefilter check").Return(testEmbedding, nil)
	chunkRepo.On("FindSimilar", mock.Anything, testEmbedding, chunkrepo.SimilarityOptions{
		MinSimilarity: 0.7,
		MaxResults:    10,
		// PatternIDs should remain empty for MCP search.
		PatternIDs: nil,
	}).Return([]*chunkrepo.Match{testChunkMatch(patternID, "no-prefilter", 0.8)}, nil)

	result, err := svc.SearchPatterns(context.Background(), search.SearchOptions{
		Query:     "legacy prefilter check",
		Limit:     10,
		Threshold: 0.7,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Matches, 1)
	assert.Equal(t, "no-prefilter", result.Matches[0].PatternName)

	chunkRepo.AssertExpectations(t)
	patternRepo.AssertNotCalled(t, "GetPatternIDsByAgent")
}

func TestSearchPatterns_EmbeddingFailure(t *testing.T) {
	t.Parallel()

	embSvc := new(mockEmbeddingService)
	patternRepo := new(mockPatternRepo)
	chunkRepo := new(mockChunkRepo)
	svc := newTestService(embSvc, patternRepo, chunkRepo)

	embSvc.On("Embed", mock.Anything, "some query").Return(nil, openaisvc.ErrEmbeddingFailed)

	result, err := svc.SearchPatterns(context.Background(), search.SearchOptions{
		Query:     "some query",
		Limit:     10,
		Threshold: 0.7,
	})

	assert.Nil(t, result)
	require.Error(t, err)
	assert.True(t, errors.Is(err, service.ErrServiceUnavailable), "expected service.ErrServiceUnavailable, got: %v", err)

	chunkRepo.AssertNotCalled(t, "FindSimilar")
}

func TestSearchPatterns_NoMatchingPatterns(t *testing.T) {
	t.Parallel()

	embSvc := new(mockEmbeddingService)
	patternRepo := new(mockPatternRepo)
	chunkRepo := new(mockChunkRepo)
	svc := newTestService(embSvc, patternRepo, chunkRepo)

	embSvc.On("Embed", mock.Anything, "obscure topic").Return(testEmbedding, nil)
	chunkRepo.On("FindSimilar", mock.Anything, testEmbedding, chunkrepo.SimilarityOptions{
		MinSimilarity: 0.9,
		MaxResults:    5,
	}).Return([]*chunkrepo.Match{}, nil)

	result, err := svc.SearchPatterns(context.Background(), search.SearchOptions{
		Query:     "obscure topic",
		Limit:     5,
		Threshold: 0.9,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.Matches)
	assert.Equal(t, "obscure topic", result.Query)
	assert.Equal(t, 0, result.TotalCandidates)
}

func TestSearchPatterns_ContextCancellation(t *testing.T) {
	t.Parallel()

	embSvc := new(mockEmbeddingService)
	patternRepo := new(mockPatternRepo)
	chunkRepo := new(mockChunkRepo)
	svc := newTestService(embSvc, patternRepo, chunkRepo)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	embSvc.On("Embed", mock.Anything, "cancelled query").Return(nil, context.Canceled)

	result, err := svc.SearchPatterns(ctx, search.SearchOptions{
		Query:     "cancelled query",
		Limit:     10,
		Threshold: 0.7,
	})

	assert.Nil(t, result)
	require.Error(t, err)
	assert.True(t, errors.Is(err, service.ErrServiceUnavailable), "expected service.ErrServiceUnavailable, got: %v", err)
}

func TestSearchPatterns_WithTags(t *testing.T) {
	t.Parallel()

	embSvc := new(mockEmbeddingService)
	patternRepo := new(mockPatternRepo)
	chunkRepo := new(mockChunkRepo)
	svc := newTestService(embSvc, patternRepo, chunkRepo)

	id1 := uuid.New()

	embSvc.On("Embed", mock.Anything, "go patterns").Return(testEmbedding, nil)
	chunkRepo.On("FindSimilar", mock.Anything, testEmbedding, chunkrepo.SimilarityOptions{
		MinSimilarity: 0.7,
		MaxResults:    10,
		Tags:          []string{"go", "best-practices"},
	}).Return([]*chunkrepo.Match{
		testChunkMatch(id1, "go-best-practices", 0.91),
	}, nil)

	result, err := svc.SearchPatterns(context.Background(), search.SearchOptions{
		Query:     "go patterns",
		Limit:     10,
		Threshold: 0.7,
		Tags:      []string{"go", "best-practices"},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Matches, 1)

	embSvc.AssertExpectations(t)
	chunkRepo.AssertExpectations(t)
}

func TestSearchPatterns_FindSimilarError(t *testing.T) {
	t.Parallel()

	embSvc := new(mockEmbeddingService)
	patternRepo := new(mockPatternRepo)
	chunkRepo := new(mockChunkRepo)
	svc := newTestService(embSvc, patternRepo, chunkRepo)

	embSvc.On("Embed", mock.Anything, "some query").Return(testEmbedding, nil)
	chunkRepo.On("FindSimilar", mock.Anything, testEmbedding, mock.Anything).
		Return(nil, errors.New("database connection lost"))

	result, err := svc.SearchPatterns(context.Background(), search.SearchOptions{
		Query:     "some query",
		Limit:     10,
		Threshold: 0.7,
	})

	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "find similar chunks")
}

func TestSearchPatterns_NoAgentRepoCallsOnSearch(t *testing.T) {
	t.Parallel()

	embSvc := new(mockEmbeddingService)
	patternRepo := new(mockPatternRepo)
	chunkRepo := new(mockChunkRepo)
	svc := newTestService(embSvc, patternRepo, chunkRepo)

	id := uuid.New()
	embSvc.On("Embed", mock.Anything, "no agent lookups").Return(testEmbedding, nil)
	chunkRepo.On("FindSimilar", mock.Anything, testEmbedding, chunkrepo.SimilarityOptions{
		MinSimilarity: 0.7,
		MaxResults:    10,
	}).Return([]*chunkrepo.Match{testChunkMatch(id, "no-agent-lookup", 0.83)}, nil)

	result, err := svc.SearchPatterns(context.Background(), search.SearchOptions{
		Query:     "no agent lookups",
		Limit:     10,
		Threshold: 0.7,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Matches, 1)
	patternRepo.AssertNotCalled(t, "GetPatternIDsByAgent")
}

func TestSearchPatterns_NoPatternIDLookupOnSearch(t *testing.T) {
	t.Parallel()

	embSvc := new(mockEmbeddingService)
	patternRepo := new(mockPatternRepo)
	chunkRepo := new(mockChunkRepo)
	svc := newTestService(embSvc, patternRepo, chunkRepo)

	id := uuid.New()
	embSvc.On("Embed", mock.Anything, "pattern id lookup removed").Return(testEmbedding, nil)
	chunkRepo.On("FindSimilar", mock.Anything, testEmbedding, chunkrepo.SimilarityOptions{
		MinSimilarity: 0.7,
		MaxResults:    10,
	}).Return([]*chunkrepo.Match{testChunkMatch(id, "lookup-removed", 0.79)}, nil)

	result, err := svc.SearchPatterns(context.Background(), search.SearchOptions{
		Query:     "pattern id lookup removed",
		Limit:     10,
		Threshold: 0.7,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Matches, 1)
	patternRepo.AssertNotCalled(t, "GetPatternIDsByAgent")
}

func TestSearchPatterns_ChunkRepoNotConfigured(t *testing.T) {
	t.Parallel()

	embSvc := new(mockEmbeddingService)
	patternRepo := new(mockPatternRepo)
	// Pass nil chunkRepo explicitly.
	svc := search.New(embSvc, patternRepo, nil, nil, zerolog.Nop())

	embSvc.On("Embed", mock.Anything, "some query").Return(testEmbedding, nil)

	result, err := svc.SearchPatterns(context.Background(), search.SearchOptions{
		Query:     "some query",
		Limit:     10,
		Threshold: 0.7,
	})

	assert.Nil(t, result)
	require.Error(t, err)
	assert.True(t, errors.Is(err, service.ErrServiceUnavailable))
}

// ---------- Graph Expansion Tests ----------

func TestSearchPatterns_GraphExpansion_HappyPath(t *testing.T) {
	t.Parallel()

	embSvc := new(mockEmbeddingService)
	patternRepo := new(mockPatternRepo)
	chunkRepo := new(mockChunkRepo)
	graphRepo := new(mockGraphRepo)
	svc := newTestServiceWithGraph(embSvc, patternRepo, chunkRepo, graphRepo)

	id1 := uuid.New()
	id2 := uuid.New()
	id3 := uuid.New()
	graphID1 := uuid.New()
	graphID2 := uuid.New()

	embSvc.On("Embed", mock.Anything, "error handling").Return(testEmbedding, nil)
	chunkRepo.On("FindSimilar", mock.Anything, testEmbedding, mock.Anything).Return([]*chunkrepo.Match{
		testChunkMatch(id1, "pattern-a", 0.95),
		testChunkMatch(id2, "pattern-b", 0.88),
		testChunkMatch(id3, "pattern-c", 0.80),
	}, nil)

	// Top 3 seeds: id1 (0.95), id2 (0.88), id3 (0.80)
	graphRepo.On("FindRelatedPatterns", mock.Anything, id1, 5).Return([]graphrepo.RelatedPattern{
		{ID: graphID1, Name: "graph-pattern-1", Similarity: 0.75, ConceptNames: []string{"error", "context"}},
	}, nil)
	graphRepo.On("FindRelatedPatterns", mock.Anything, id2, 5).Return([]graphrepo.RelatedPattern{
		{ID: graphID2, Name: "graph-pattern-2", Similarity: 0.65, ConceptNames: []string{"retry"}},
	}, nil)
	graphRepo.On("FindRelatedPatterns", mock.Anything, id3, 5).Return([]graphrepo.RelatedPattern{}, nil)

	patternRepo.On("GetByIDs", mock.Anything, mock.Anything).Return([]*pattern.Pattern{
		{ID: graphID1, Name: "graph-pattern-1", Language: "go", Domain: "backend", Tags: []string{"go"}},
		{ID: graphID2, Name: "graph-pattern-2", Language: "go", Domain: "backend", Tags: []string{"go"}},
	}, nil)

	result, err := svc.SearchPatterns(context.Background(), search.SearchOptions{
		Query:     "error handling",
		Limit:     10,
		Threshold: 0.7,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Matches, 3)
	require.NotNil(t, result.GraphMatches)
	assert.Len(t, result.GraphMatches, 2)

	// Verify GraphMatch fields are populated correctly.
	gm1 := result.GraphMatches[0]
	assert.Equal(t, graphID1, gm1.PatternID)
	assert.Equal(t, "graph-pattern-1", gm1.PatternName)
	assert.InDelta(t, 0.75, gm1.Similarity, 0.001)
	assert.Equal(t, []string{"error", "context"}, gm1.ConceptNames)
	assert.Equal(t, id1, gm1.SeedPatternID)
	assert.Equal(t, "pattern-a", gm1.SeedPatternName)
}

func TestSearchPatterns_GraphExpansion_NilGraphRepo(t *testing.T) {
	t.Parallel()

	embSvc := new(mockEmbeddingService)
	patternRepo := new(mockPatternRepo)
	chunkRepo := new(mockChunkRepo)
	svc := newTestService(embSvc, patternRepo, chunkRepo) // graphRepo = nil

	id1 := uuid.New()

	embSvc.On("Embed", mock.Anything, "some query").Return(testEmbedding, nil)
	chunkRepo.On("FindSimilar", mock.Anything, testEmbedding, mock.Anything).Return([]*chunkrepo.Match{
		testChunkMatch(id1, "pattern-a", 0.92),
	}, nil)

	result, err := svc.SearchPatterns(context.Background(), search.SearchOptions{
		Query:     "some query",
		Limit:     10,
		Threshold: 0.7,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Matches, 1)
	assert.Nil(t, result.GraphMatches, "GraphMatches should be nil when graphRepo is nil")
}

func TestSearchPatterns_GraphExpansion_GraphFailure(t *testing.T) {
	t.Parallel()

	embSvc := new(mockEmbeddingService)
	patternRepo := new(mockPatternRepo)
	chunkRepo := new(mockChunkRepo)
	graphRepo := new(mockGraphRepo)
	svc := newTestServiceWithGraph(embSvc, patternRepo, chunkRepo, graphRepo)

	id1 := uuid.New()

	embSvc.On("Embed", mock.Anything, "some query").Return(testEmbedding, nil)
	chunkRepo.On("FindSimilar", mock.Anything, testEmbedding, mock.Anything).Return([]*chunkrepo.Match{
		testChunkMatch(id1, "pattern-a", 0.92),
	}, nil)
	graphRepo.On("FindRelatedPatterns", mock.Anything, id1, 5).Return(nil, errors.New("neo4j unavailable"))

	result, err := svc.SearchPatterns(context.Background(), search.SearchOptions{
		Query:     "some query",
		Limit:     10,
		Threshold: 0.7,
	})

	require.NoError(t, err, "graph failure should not propagate as an error")
	require.NotNil(t, result)
	assert.Len(t, result.Matches, 1, "vector results should still be returned")
	assert.Nil(t, result.GraphMatches, "GraphMatches should be nil on graph failure")
}

func TestSearchPatterns_GraphExpansion_DeduplicateAgainstVector(t *testing.T) {
	t.Parallel()

	embSvc := new(mockEmbeddingService)
	patternRepo := new(mockPatternRepo)
	chunkRepo := new(mockChunkRepo)
	graphRepo := new(mockGraphRepo)
	svc := newTestServiceWithGraph(embSvc, patternRepo, chunkRepo, graphRepo)

	id1 := uuid.New()
	id2 := uuid.New() // Also returned by graph

	embSvc.On("Embed", mock.Anything, "some query").Return(testEmbedding, nil)
	chunkRepo.On("FindSimilar", mock.Anything, testEmbedding, mock.Anything).Return([]*chunkrepo.Match{
		testChunkMatch(id1, "pattern-a", 0.95),
		testChunkMatch(id2, "pattern-b", 0.85),
	}, nil)
	// Graph returns id2 which is already in vector results.
	graphRepo.On("FindRelatedPatterns", mock.Anything, id1, 5).Return([]graphrepo.RelatedPattern{
		{ID: id2, Name: "pattern-b", Similarity: 0.70, ConceptNames: []string{"shared"}},
	}, nil)
	graphRepo.On("FindRelatedPatterns", mock.Anything, id2, 5).Return([]graphrepo.RelatedPattern{}, nil)

	result, err := svc.SearchPatterns(context.Background(), search.SearchOptions{
		Query:     "some query",
		Limit:     10,
		Threshold: 0.7,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Matches, 2)
	assert.Nil(t, result.GraphMatches, "pattern already in vector results should be excluded from GraphMatches")

	patternRepo.AssertNotCalled(t, "GetByIDs")
}

func TestSearchPatterns_GraphExpansion_CrossSeedDedup(t *testing.T) {
	t.Parallel()

	embSvc := new(mockEmbeddingService)
	patternRepo := new(mockPatternRepo)
	chunkRepo := new(mockChunkRepo)
	graphRepo := new(mockGraphRepo)
	svc := newTestServiceWithGraph(embSvc, patternRepo, chunkRepo, graphRepo)

	id1 := uuid.New()
	id2 := uuid.New()
	sharedGraphID := uuid.New()

	embSvc.On("Embed", mock.Anything, "some query").Return(testEmbedding, nil)
	chunkRepo.On("FindSimilar", mock.Anything, testEmbedding, mock.Anything).Return([]*chunkrepo.Match{
		testChunkMatch(id1, "pattern-a", 0.95),
		testChunkMatch(id2, "pattern-b", 0.85),
	}, nil)
	// Both seeds return the same graph pattern with different similarities.
	graphRepo.On("FindRelatedPatterns", mock.Anything, id1, 5).Return([]graphrepo.RelatedPattern{
		{ID: sharedGraphID, Name: "shared-graph-pattern", Similarity: 0.60, ConceptNames: []string{"concept-a"}},
	}, nil)
	graphRepo.On("FindRelatedPatterns", mock.Anything, id2, 5).Return([]graphrepo.RelatedPattern{
		{ID: sharedGraphID, Name: "shared-graph-pattern", Similarity: 0.80, ConceptNames: []string{"concept-a", "concept-b"}},
	}, nil)

	patternRepo.On("GetByIDs", mock.Anything, mock.Anything).Return([]*pattern.Pattern{
		{ID: sharedGraphID, Name: "shared-graph-pattern", Language: "go", Domain: "backend", Tags: []string{"go"}},
	}, nil)

	result, err := svc.SearchPatterns(context.Background(), search.SearchOptions{
		Query:     "some query",
		Limit:     10,
		Threshold: 0.7,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.GraphMatches)
	assert.Len(t, result.GraphMatches, 1, "duplicate cross-seed graph result should appear only once")
	// Should keep the higher similarity (0.80 from id2).
	assert.InDelta(t, 0.80, result.GraphMatches[0].Similarity, 0.001)
}

func TestSearchPatterns_GraphExpansion_LanguageFiltering(t *testing.T) {
	t.Parallel()

	embSvc := new(mockEmbeddingService)
	patternRepo := new(mockPatternRepo)
	chunkRepo := new(mockChunkRepo)
	graphRepo := new(mockGraphRepo)
	svc := newTestServiceWithGraph(embSvc, patternRepo, chunkRepo, graphRepo)

	id1 := uuid.New()
	goID := uuid.New()
	agnosticID := uuid.New()
	pythonID := uuid.New()

	embSvc.On("Embed", mock.Anything, "some query").Return(testEmbedding, nil)
	chunkRepo.On("FindSimilar", mock.Anything, testEmbedding, mock.Anything).Return([]*chunkrepo.Match{
		testChunkMatch(id1, "pattern-a", 0.95),
	}, nil)
	graphRepo.On("FindRelatedPatterns", mock.Anything, id1, 5).Return([]graphrepo.RelatedPattern{
		{ID: goID, Name: "go-pattern", Similarity: 0.80, ConceptNames: []string{"concurrency"}},
		{ID: agnosticID, Name: "agnostic-pattern", Similarity: 0.75, ConceptNames: []string{"logging"}},
		{ID: pythonID, Name: "python-pattern", Similarity: 0.70, ConceptNames: []string{"async"}},
	}, nil)
	patternRepo.On("GetByIDs", mock.Anything, mock.Anything).Return([]*pattern.Pattern{
		{ID: goID, Name: "go-pattern", Language: "go", Domain: "backend"},
		{ID: agnosticID, Name: "agnostic-pattern", Language: "agnostic", Domain: "backend"},
		{ID: pythonID, Name: "python-pattern", Language: "python", Domain: "backend"},
	}, nil)

	result, err := svc.SearchPatterns(context.Background(), search.SearchOptions{
		Query:     "some query",
		Limit:     10,
		Threshold: 0.7,
		Language:  "go",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.GraphMatches)
	assert.Len(t, result.GraphMatches, 2, "only go and agnostic patterns should pass language filter")

	ids := make([]uuid.UUID, len(result.GraphMatches))
	for i, gm := range result.GraphMatches {
		ids[i] = gm.PatternID
	}
	assert.Contains(t, ids, goID)
	assert.Contains(t, ids, agnosticID)
	assert.NotContains(t, ids, pythonID)
}

func TestSearchPatterns_GraphExpansion_LanguageFilter_Empty(t *testing.T) {
	t.Parallel()

	embSvc := new(mockEmbeddingService)
	patternRepo := new(mockPatternRepo)
	chunkRepo := new(mockChunkRepo)
	graphRepo := new(mockGraphRepo)
	svc := newTestServiceWithGraph(embSvc, patternRepo, chunkRepo, graphRepo)

	id1 := uuid.New()
	goID := uuid.New()
	pythonID := uuid.New()

	embSvc.On("Embed", mock.Anything, "some query").Return(testEmbedding, nil)
	chunkRepo.On("FindSimilar", mock.Anything, testEmbedding, mock.Anything).Return([]*chunkrepo.Match{
		testChunkMatch(id1, "pattern-a", 0.95),
	}, nil)
	graphRepo.On("FindRelatedPatterns", mock.Anything, id1, 5).Return([]graphrepo.RelatedPattern{
		{ID: goID, Name: "go-pattern", Similarity: 0.80, ConceptNames: []string{"concurrency"}},
		{ID: pythonID, Name: "python-pattern", Similarity: 0.70, ConceptNames: []string{"async"}},
	}, nil)
	patternRepo.On("GetByIDs", mock.Anything, mock.Anything).Return([]*pattern.Pattern{
		{ID: goID, Name: "go-pattern", Language: "go", Domain: "backend"},
		{ID: pythonID, Name: "python-pattern", Language: "python", Domain: "backend"},
	}, nil)

	result, err := svc.SearchPatterns(context.Background(), search.SearchOptions{
		Query:     "some query",
		Limit:     10,
		Threshold: 0.7,
		Language:  "", // empty — all languages pass
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.GraphMatches)
	assert.Len(t, result.GraphMatches, 2, "all languages should pass when Language is empty")
}

func TestSearchPatterns_GraphExpansion_EmptyVectorResults(t *testing.T) {
	t.Parallel()

	embSvc := new(mockEmbeddingService)
	patternRepo := new(mockPatternRepo)
	chunkRepo := new(mockChunkRepo)
	graphRepo := new(mockGraphRepo)
	svc := newTestServiceWithGraph(embSvc, patternRepo, chunkRepo, graphRepo)

	embSvc.On("Embed", mock.Anything, "obscure query").Return(testEmbedding, nil)
	chunkRepo.On("FindSimilar", mock.Anything, testEmbedding, mock.Anything).Return([]*chunkrepo.Match{}, nil)

	result, err := svc.SearchPatterns(context.Background(), search.SearchOptions{
		Query:     "obscure query",
		Limit:     10,
		Threshold: 0.9,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.Matches)
	assert.Nil(t, result.GraphMatches, "graph expansion should be skipped when vector returns no results")

	graphRepo.AssertNotCalled(t, "FindRelatedPatterns")
}

func TestSearchPatterns_GraphExpansion_Cap(t *testing.T) {
	t.Parallel()

	embSvc := new(mockEmbeddingService)
	patternRepo := new(mockPatternRepo)
	chunkRepo := new(mockChunkRepo)
	graphRepo := new(mockGraphRepo)
	svc := newTestServiceWithGraph(embSvc, patternRepo, chunkRepo, graphRepo)

	id1 := uuid.New()

	embSvc.On("Embed", mock.Anything, "some query").Return(testEmbedding, nil)
	chunkRepo.On("FindSimilar", mock.Anything, testEmbedding, mock.Anything).Return([]*chunkrepo.Match{
		testChunkMatch(id1, "pattern-a", 0.95),
	}, nil)

	// Graph returns 7 unique related patterns.
	graphIDs := make([]uuid.UUID, 7)
	related := make([]graphrepo.RelatedPattern, 7)
	patterns := make([]*pattern.Pattern, 7)
	for i := range graphIDs {
		graphIDs[i] = uuid.New()
		related[i] = graphrepo.RelatedPattern{
			ID:           graphIDs[i],
			Name:         "graph-pattern",
			Similarity:   0.70,
			ConceptNames: []string{"shared"},
		}
		patterns[i] = &pattern.Pattern{
			ID:       graphIDs[i],
			Name:     "graph-pattern",
			Language: "go",
			Domain:   "backend",
		}
	}

	graphRepo.On("FindRelatedPatterns", mock.Anything, id1, 5).Return(related, nil)
	patternRepo.On("GetByIDs", mock.Anything, mock.Anything).Return(patterns, nil)

	result, err := svc.SearchPatterns(context.Background(), search.SearchOptions{
		Query:     "some query",
		Limit:     10,
		Threshold: 0.7,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.GraphMatches)
	assert.Len(t, result.GraphMatches, 5, "graph matches should be capped at 5")
}

func TestSearchPatterns_GraphExpansion_GetByIDsFailure(t *testing.T) {
	t.Parallel()

	embSvc := new(mockEmbeddingService)
	patternRepo := new(mockPatternRepo)
	chunkRepo := new(mockChunkRepo)
	graphRepo := new(mockGraphRepo)
	svc := newTestServiceWithGraph(embSvc, patternRepo, chunkRepo, graphRepo)

	id1 := uuid.New()
	graphID1 := uuid.New()

	embSvc.On("Embed", mock.Anything, "some query").Return(testEmbedding, nil)
	chunkRepo.On("FindSimilar", mock.Anything, testEmbedding, mock.Anything).Return([]*chunkrepo.Match{
		testChunkMatch(id1, "pattern-a", 0.95),
	}, nil)
	graphRepo.On("FindRelatedPatterns", mock.Anything, id1, 5).Return([]graphrepo.RelatedPattern{
		{ID: graphID1, Name: "graph-pattern-1", Similarity: 0.75, ConceptNames: []string{"error"}},
	}, nil)
	patternRepo.On("GetByIDs", mock.Anything, mock.Anything).Return(nil, errors.New("postgres timeout"))

	result, err := svc.SearchPatterns(context.Background(), search.SearchOptions{
		Query:     "some query",
		Limit:     10,
		Threshold: 0.7,
	})

	require.NoError(t, err, "GetByIDs failure should not propagate as an error")
	require.NotNil(t, result)
	assert.Len(t, result.Matches, 1, "vector results should still be returned")
	assert.Nil(t, result.GraphMatches, "GraphMatches should be nil when GetByIDs fails")
}
