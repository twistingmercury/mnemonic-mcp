package search_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	agentrepo "github.com/twistingmercury/mnemonic/internal/repository/agent"
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

func (m *mockPatternRepo) FindSimilar(ctx context.Context, embedding []float32, opts pattern.SimilarityOptions) ([]*pattern.Match, error) {
	args := m.Called(ctx, embedding, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*pattern.Match), args.Error(1)
}

func (m *mockPatternRepo) SetAgentAssociations(ctx context.Context, patternID uuid.UUID, associations []pattern.AgentAssociation) error {
	args := m.Called(ctx, patternID, associations)
	return args.Error(0)
}

func (m *mockPatternRepo) GetAgentAssociations(ctx context.Context, patternID uuid.UUID) ([]pattern.AgentAssociation, error) {
	args := m.Called(ctx, patternID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]pattern.AgentAssociation), args.Error(1)
}

func (m *mockPatternRepo) GetPatternIDsByAgent(ctx context.Context, agentID uuid.UUID) ([]uuid.UUID, error) {
	args := m.Called(ctx, agentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]uuid.UUID), args.Error(1)
}

func (m *mockPatternRepo) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}

// --- Mock: agentrepo.Repository ---

type mockAgentRepo struct {
	mock.Mock
}

func (m *mockAgentRepo) Create(ctx context.Context, agent *agentrepo.Agent) error {
	args := m.Called(ctx, agent)
	return args.Error(0)
}

func (m *mockAgentRepo) Get(ctx context.Context, name string) (*agentrepo.Agent, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*agentrepo.Agent), args.Error(1)
}

func (m *mockAgentRepo) GetByID(ctx context.Context, id uuid.UUID) (*agentrepo.Agent, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*agentrepo.Agent), args.Error(1)
}

func (m *mockAgentRepo) Update(ctx context.Context, agent *agentrepo.Agent) error {
	args := m.Called(ctx, agent)
	return args.Error(0)
}

func (m *mockAgentRepo) Delete(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *mockAgentRepo) DeleteByID(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockAgentRepo) List(ctx context.Context, opts repository.ListOptions) ([]*agentrepo.Agent, int64, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*agentrepo.Agent), args.Get(1).(int64), args.Error(2)
}

func (m *mockAgentRepo) Exists(ctx context.Context, name string) (bool, error) {
	args := m.Called(ctx, name)
	return args.Bool(0), args.Error(1)
}

func (m *mockAgentRepo) GetManifest(ctx context.Context) ([]agentrepo.ManifestEntry, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]agentrepo.ManifestEntry), args.Error(1)
}

// --- Helpers ---

var testEmbedding = []float32{0.1, 0.2, 0.3}

func newTestService(embSvc *mockEmbeddingService, patternRepo *mockPatternRepo, agentRepo *mockAgentRepo) search.Service {
	logger := zerolog.Nop()
	return search.New(embSvc, patternRepo, agentRepo, logger)
}

func testPatternMatch(id uuid.UUID, name string, similarity float64) *pattern.Match {
	return &pattern.Match{
		Pattern: &pattern.Pattern{
			ID:      id,
			Name:    name,
			Content: "test content for " + name,
			Tags:    []string{"go", "testing"},
		},
		Similarity: similarity,
	}
}

// ---------- SearchPatterns ----------

func TestSearchPatterns_HappyPath(t *testing.T) {
	t.Parallel()

	embSvc := new(mockEmbeddingService)
	patternRepo := new(mockPatternRepo)
	agentRepo := new(mockAgentRepo)
	svc := newTestService(embSvc, patternRepo, agentRepo)

	id1 := uuid.New()
	id2 := uuid.New()

	embSvc.On("Embed", mock.Anything, "error handling in Go").Return(testEmbedding, nil)
	patternRepo.On("FindSimilar", mock.Anything, testEmbedding, pattern.SimilarityOptions{
		MinSimilarity: 0.7,
		MaxResults:    10,
	}).Return([]*pattern.Match{
		testPatternMatch(id1, "go-error-handling", 0.92),
		testPatternMatch(id2, "go-error-wrapping", 0.85),
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
	assert.Equal(t, id1, result.Matches[0].Pattern.ID)
	assert.InDelta(t, 0.92, result.Matches[0].Similarity, 0.001)

	embSvc.AssertExpectations(t)
	patternRepo.AssertExpectations(t)
	agentRepo.AssertNotCalled(t, "Get")
}

func TestSearchPatterns_WithAgentFilter(t *testing.T) {
	t.Parallel()

	embSvc := new(mockEmbeddingService)
	patternRepo := new(mockPatternRepo)
	agentRepo := new(mockAgentRepo)
	svc := newTestService(embSvc, patternRepo, agentRepo)

	agentID := uuid.New()
	patternID1 := uuid.New()
	patternID2 := uuid.New()

	embSvc.On("Embed", mock.Anything, "testing patterns").Return(testEmbedding, nil)
	agentRepo.On("Get", mock.Anything, "go-engineer").Return(&agentrepo.Agent{
		ID:         agentID,
		Name:       "go-engineer",
		Definition: json.RawMessage(`{}`),
		CRC64:      "123",
	}, nil)
	patternRepo.On("GetPatternIDsByAgent", mock.Anything, agentID).Return([]uuid.UUID{patternID1, patternID2}, nil)
	patternRepo.On("FindSimilar", mock.Anything, testEmbedding, pattern.SimilarityOptions{
		MinSimilarity: 0.7,
		MaxResults:    10,
		PatternIDs:    []uuid.UUID{patternID1, patternID2},
	}).Return([]*pattern.Match{
		testPatternMatch(patternID1, "go-testing", 0.88),
	}, nil)

	result, err := svc.SearchPatterns(context.Background(), search.SearchOptions{
		Query:     "testing patterns",
		Limit:     10,
		Threshold: 0.7,
		AgentName: "go-engineer",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Matches, 1)
	assert.Equal(t, patternID1, result.Matches[0].Pattern.ID)

	embSvc.AssertExpectations(t)
	patternRepo.AssertExpectations(t)
	agentRepo.AssertExpectations(t)
}

func TestSearchPatterns_AgentFilterUnknownAgent(t *testing.T) {
	t.Parallel()

	embSvc := new(mockEmbeddingService)
	patternRepo := new(mockPatternRepo)
	agentRepo := new(mockAgentRepo)
	svc := newTestService(embSvc, patternRepo, agentRepo)

	embSvc.On("Embed", mock.Anything, "some query").Return(testEmbedding, nil)
	agentRepo.On("Get", mock.Anything, "unknown-agent").Return(nil, agentrepo.ErrNotFound)

	result, err := svc.SearchPatterns(context.Background(), search.SearchOptions{
		Query:     "some query",
		Limit:     10,
		Threshold: 0.7,
		AgentName: "unknown-agent",
	})

	require.NoError(t, err, "unknown agent should not return an error")
	require.NotNil(t, result)
	assert.Empty(t, result.Matches)
	assert.Equal(t, "some query", result.Query)
	assert.Equal(t, 0, result.TotalCandidates)
	assert.Greater(t, result.SearchDurationMs, int64(-1))

	patternRepo.AssertNotCalled(t, "FindSimilar")
	patternRepo.AssertNotCalled(t, "GetPatternIDsByAgent")
}

func TestSearchPatterns_AgentFilterNoAssociatedPatterns(t *testing.T) {
	t.Parallel()

	embSvc := new(mockEmbeddingService)
	patternRepo := new(mockPatternRepo)
	agentRepo := new(mockAgentRepo)
	svc := newTestService(embSvc, patternRepo, agentRepo)

	agentID := uuid.New()

	embSvc.On("Embed", mock.Anything, "some query").Return(testEmbedding, nil)
	agentRepo.On("Get", mock.Anything, "lonely-agent").Return(&agentrepo.Agent{
		ID:         agentID,
		Name:       "lonely-agent",
		Definition: json.RawMessage(`{}`),
		CRC64:      "456",
	}, nil)
	patternRepo.On("GetPatternIDsByAgent", mock.Anything, agentID).Return([]uuid.UUID{}, nil)

	result, err := svc.SearchPatterns(context.Background(), search.SearchOptions{
		Query:     "some query",
		Limit:     10,
		Threshold: 0.7,
		AgentName: "lonely-agent",
	})

	require.NoError(t, err, "agent with no patterns should not return an error")
	require.NotNil(t, result)
	assert.Empty(t, result.Matches)
	assert.Equal(t, 0, result.TotalCandidates)

	patternRepo.AssertNotCalled(t, "FindSimilar")
}

func TestSearchPatterns_EmbeddingFailure(t *testing.T) {
	t.Parallel()

	embSvc := new(mockEmbeddingService)
	patternRepo := new(mockPatternRepo)
	agentRepo := new(mockAgentRepo)
	svc := newTestService(embSvc, patternRepo, agentRepo)

	embSvc.On("Embed", mock.Anything, "some query").Return(nil, openaisvc.ErrEmbeddingFailed)

	result, err := svc.SearchPatterns(context.Background(), search.SearchOptions{
		Query:     "some query",
		Limit:     10,
		Threshold: 0.7,
	})

	assert.Nil(t, result)
	require.Error(t, err)
	assert.True(t, errors.Is(err, service.ErrServiceUnavailable), "expected service.ErrServiceUnavailable, got: %v", err)

	patternRepo.AssertNotCalled(t, "FindSimilar")
}

func TestSearchPatterns_NoMatchingPatterns(t *testing.T) {
	t.Parallel()

	embSvc := new(mockEmbeddingService)
	patternRepo := new(mockPatternRepo)
	agentRepo := new(mockAgentRepo)
	svc := newTestService(embSvc, patternRepo, agentRepo)

	embSvc.On("Embed", mock.Anything, "obscure topic").Return(testEmbedding, nil)
	patternRepo.On("FindSimilar", mock.Anything, testEmbedding, pattern.SimilarityOptions{
		MinSimilarity: 0.9,
		MaxResults:    5,
	}).Return([]*pattern.Match{}, nil)

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
	agentRepo := new(mockAgentRepo)
	svc := newTestService(embSvc, patternRepo, agentRepo)

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
	agentRepo := new(mockAgentRepo)
	svc := newTestService(embSvc, patternRepo, agentRepo)

	id1 := uuid.New()

	embSvc.On("Embed", mock.Anything, "go patterns").Return(testEmbedding, nil)
	patternRepo.On("FindSimilar", mock.Anything, testEmbedding, pattern.SimilarityOptions{
		MinSimilarity: 0.7,
		MaxResults:    10,
		Tags:          []string{"go", "best-practices"},
	}).Return([]*pattern.Match{
		testPatternMatch(id1, "go-best-practices", 0.91),
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
	patternRepo.AssertExpectations(t)
}

func TestSearchPatterns_FindSimilarError(t *testing.T) {
	t.Parallel()

	embSvc := new(mockEmbeddingService)
	patternRepo := new(mockPatternRepo)
	agentRepo := new(mockAgentRepo)
	svc := newTestService(embSvc, patternRepo, agentRepo)

	embSvc.On("Embed", mock.Anything, "some query").Return(testEmbedding, nil)
	patternRepo.On("FindSimilar", mock.Anything, testEmbedding, mock.Anything).
		Return(nil, errors.New("database connection lost"))

	result, err := svc.SearchPatterns(context.Background(), search.SearchOptions{
		Query:     "some query",
		Limit:     10,
		Threshold: 0.7,
	})

	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "find similar patterns")
}

func TestSearchPatterns_AgentRepoError(t *testing.T) {
	t.Parallel()

	embSvc := new(mockEmbeddingService)
	patternRepo := new(mockPatternRepo)
	agentRepo := new(mockAgentRepo)
	svc := newTestService(embSvc, patternRepo, agentRepo)

	embSvc.On("Embed", mock.Anything, "some query").Return(testEmbedding, nil)
	agentRepo.On("Get", mock.Anything, "broken-agent").Return(nil, errors.New("connection refused"))

	result, err := svc.SearchPatterns(context.Background(), search.SearchOptions{
		Query:     "some query",
		Limit:     10,
		Threshold: 0.7,
		AgentName: "broken-agent",
	})

	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolve agent")

	patternRepo.AssertNotCalled(t, "FindSimilar")
}

func TestSearchPatterns_GetPatternIDsByAgentError(t *testing.T) {
	t.Parallel()

	embSvc := new(mockEmbeddingService)
	patternRepo := new(mockPatternRepo)
	agentRepo := new(mockAgentRepo)
	svc := newTestService(embSvc, patternRepo, agentRepo)

	agentID := uuid.New()

	embSvc.On("Embed", mock.Anything, "some query").Return(testEmbedding, nil)
	agentRepo.On("Get", mock.Anything, "my-agent").Return(&agentrepo.Agent{
		ID:         agentID,
		Name:       "my-agent",
		Definition: json.RawMessage(`{}`),
		CRC64:      "789",
	}, nil)
	patternRepo.On("GetPatternIDsByAgent", mock.Anything, agentID).
		Return(nil, errors.New("query timeout"))

	result, err := svc.SearchPatterns(context.Background(), search.SearchOptions{
		Query:     "some query",
		Limit:     10,
		Threshold: 0.7,
		AgentName: "my-agent",
	})

	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get agent patterns")

	patternRepo.AssertNotCalled(t, "FindSimilar")
}
