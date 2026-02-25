package enrichment_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/twistingmercury/mnemonic/internal/config"
	"github.com/twistingmercury/mnemonic/internal/repository"
	agentrepo "github.com/twistingmercury/mnemonic/internal/repository/agent"
	enrichmentjob "github.com/twistingmercury/mnemonic/internal/repository/enrichmentjob"
	graphrepo "github.com/twistingmercury/mnemonic/internal/repository/graph"
	patternrepo "github.com/twistingmercury/mnemonic/internal/repository/pattern"
	"github.com/twistingmercury/mnemonic/internal/service/enrichment"
	openaisvc "github.com/twistingmercury/mnemonic/internal/service/openai"
)

// --- Mock: enrichmentjob.Repository ---

type mockJobRepo struct {
	mock.Mock
}

func (m *mockJobRepo) Create(ctx context.Context, job *enrichmentjob.Job) error {
	return m.Called(ctx, job).Error(0)
}

func (m *mockJobRepo) Get(ctx context.Context, id uuid.UUID) (*enrichmentjob.Job, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*enrichmentjob.Job), args.Error(1)
}

func (m *mockJobRepo) GetByPatternID(ctx context.Context, patternID uuid.UUID) (*enrichmentjob.Job, error) {
	args := m.Called(ctx, patternID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*enrichmentjob.Job), args.Error(1)
}

func (m *mockJobRepo) ClaimPending(ctx context.Context) (*enrichmentjob.Job, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*enrichmentjob.Job), args.Error(1)
}

func (m *mockJobRepo) MarkProcessing(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *mockJobRepo) MarkCompleted(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *mockJobRepo) MarkFailed(ctx context.Context, id uuid.UUID, err error, retryDelay time.Duration) error {
	return m.Called(ctx, id, err, retryDelay).Error(0)
}

func (m *mockJobRepo) ReclaimStale(ctx context.Context, timeout time.Duration) (int64, error) {
	args := m.Called(ctx, timeout)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockJobRepo) List(ctx context.Context, filter enrichmentjob.Filter, opts repository.ListOptions) ([]*enrichmentjob.Job, int64, error) {
	args := m.Called(ctx, filter, opts)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*enrichmentjob.Job), args.Get(1).(int64), args.Error(2)
}

func (m *mockJobRepo) DeleteCompleted(ctx context.Context, retention time.Duration) (int64, error) {
	args := m.Called(ctx, retention)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockJobRepo) DeleteFailed(ctx context.Context, retention time.Duration) (int64, error) {
	args := m.Called(ctx, retention)
	return args.Get(0).(int64), args.Error(1)
}

// --- Mock: patternrepo.Repository ---

type mockPatternRepo struct {
	mock.Mock
}

func (m *mockPatternRepo) Create(ctx context.Context, p *patternrepo.Pattern) error {
	return m.Called(ctx, p).Error(0)
}

func (m *mockPatternRepo) Get(ctx context.Context, id uuid.UUID) (*patternrepo.Pattern, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*patternrepo.Pattern), args.Error(1)
}

func (m *mockPatternRepo) GetByName(ctx context.Context, name string) (*patternrepo.Pattern, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*patternrepo.Pattern), args.Error(1)
}

func (m *mockPatternRepo) Update(ctx context.Context, p *patternrepo.Pattern) error {
	return m.Called(ctx, p).Error(0)
}

func (m *mockPatternRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *mockPatternRepo) List(ctx context.Context, filter patternrepo.Filter, opts repository.ListOptions) ([]*patternrepo.Pattern, int64, error) {
	args := m.Called(ctx, filter, opts)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*patternrepo.Pattern), args.Get(1).(int64), args.Error(2)
}

func (m *mockPatternRepo) UpdateEmbedding(ctx context.Context, id uuid.UUID, embedding []float32) error {
	return m.Called(ctx, id, embedding).Error(0)
}

func (m *mockPatternRepo) UpdateEnrichmentStatus(ctx context.Context, id uuid.UUID, status string, errMsg *string) error {
	return m.Called(ctx, id, status, errMsg).Error(0)
}

func (m *mockPatternRepo) FindSimilar(ctx context.Context, embedding []float32, opts patternrepo.SimilarityOptions) ([]*patternrepo.Match, error) {
	args := m.Called(ctx, embedding, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*patternrepo.Match), args.Error(1)
}

func (m *mockPatternRepo) SetAgentAssociations(ctx context.Context, patternID uuid.UUID, associations []patternrepo.AgentAssociation) error {
	return m.Called(ctx, patternID, associations).Error(0)
}

func (m *mockPatternRepo) GetAgentAssociations(ctx context.Context, patternID uuid.UUID) ([]patternrepo.AgentAssociation, error) {
	args := m.Called(ctx, patternID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]patternrepo.AgentAssociation), args.Error(1)
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
	return m.Called(ctx, agent).Error(0)
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
	return m.Called(ctx, agent).Error(0)
}

func (m *mockAgentRepo) Delete(ctx context.Context, name string) error {
	return m.Called(ctx, name).Error(0)
}

func (m *mockAgentRepo) DeleteByID(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
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

// --- Mock: graphrepo.Repository ---

type mockGraphRepo struct {
	mock.Mock
}

func (m *mockGraphRepo) SyncAgent(ctx context.Context, agentName string) error {
	return m.Called(ctx, agentName).Error(0)
}

func (m *mockGraphRepo) DeleteAgent(ctx context.Context, agentName string) error {
	return m.Called(ctx, agentName).Error(0)
}

func (m *mockGraphRepo) SyncPattern(ctx context.Context, pattern *graphrepo.Pattern) error {
	return m.Called(ctx, pattern).Error(0)
}

func (m *mockGraphRepo) DeletePattern(ctx context.Context, patternID uuid.UUID) error {
	return m.Called(ctx, patternID).Error(0)
}

func (m *mockGraphRepo) SyncConcepts(ctx context.Context, patternID uuid.UUID, concepts []graphrepo.Concept) error {
	return m.Called(ctx, patternID, concepts).Error(0)
}

func (m *mockGraphRepo) SetPatternAgentRelevance(ctx context.Context, patternID uuid.UUID, associations []graphrepo.AgentAssociation) error {
	return m.Called(ctx, patternID, associations).Error(0)
}

func (m *mockGraphRepo) ComputeRelatedToEdges(ctx context.Context, patternID uuid.UUID, minSimilarity float64) error {
	return m.Called(ctx, patternID, minSimilarity).Error(0)
}

func (m *mockGraphRepo) GetPatternConcepts(ctx context.Context, patternID uuid.UUID) ([]graphrepo.Concept, error) {
	args := m.Called(ctx, patternID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]graphrepo.Concept), args.Error(1)
}

func (m *mockGraphRepo) FindRelatedPatterns(ctx context.Context, patternID uuid.UUID, limit int) ([]graphrepo.RelatedPattern, error) {
	args := m.Called(ctx, patternID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]graphrepo.RelatedPattern), args.Error(1)
}

func (m *mockGraphRepo) FindPatternsByAgent(ctx context.Context, agentName string, limit int) ([]graphrepo.PatternRelevance, error) {
	args := m.Called(ctx, agentName, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]graphrepo.PatternRelevance), args.Error(1)
}

func (m *mockGraphRepo) CleanupOrphanedConcepts(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockGraphRepo) HealthCheck(ctx context.Context) error {
	return m.Called(ctx).Error(0)
}

// --- Mock: openaisvc.EmbeddingService ---

type mockEmbeddingSvc struct {
	mock.Mock
}

func (m *mockEmbeddingSvc) Embed(ctx context.Context, text string) ([]float32, error) {
	args := m.Called(ctx, text)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]float32), args.Error(1)
}

// --- Mock: openaisvc.ExtractionService ---

type mockExtractionSvc struct {
	mock.Mock
}

func (m *mockExtractionSvc) Extract(ctx context.Context, text string) ([]openaisvc.Concept, error) {
	args := m.Called(ctx, text)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]openaisvc.Concept), args.Error(1)
}

// --- Test Fixtures ---

func testConfig() config.EnrichmentConfig {
	return config.EnrichmentConfig{
		WorkerCount:            2,
		PollInterval:           5 * time.Second,
		MaxAttempts:            3,
		RetryDelay:             30 * time.Second,
		JobTimeout:             5 * time.Minute,
		CompletedRetention:     168 * time.Hour,
		FailedRetention:        720 * time.Hour,
		RelatedToMinSimilarity: 0.3,
	}
}

var (
	testPatternID = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	testJobID     = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	testAgentID   = uuid.MustParse("33333333-3333-3333-3333-333333333333")
)

func testJob() *enrichmentjob.Job {
	return &enrichmentjob.Job{
		ID:        testJobID,
		PatternID: testPatternID,
		Status:    "processing",
		Attempts:  0,
	}
}

func testPattern() *patternrepo.Pattern {
	desc := "A test pattern"
	return &patternrepo.Pattern{
		ID:               testPatternID,
		Name:             "test-pattern",
		Description:      &desc,
		Content:          "This is test pattern content for enrichment.",
		Tags:             []string{"test"},
		EnrichmentStatus: "pending",
	}
}

func testEmbedding() []float32 {
	return []float32{0.1, 0.2, 0.3}
}

func testConcepts() []openaisvc.Concept {
	return []openaisvc.Concept{
		{Name: "error handling", Type: "domain"},
		{Name: "go", Type: "technology"},
	}
}

func testGraphConcepts() []graphrepo.Concept {
	return []graphrepo.Concept{
		{Name: "error handling", Type: "domain"},
		{Name: "go", Type: "technology"},
	}
}

type testDeps struct {
	jobRepo       *mockJobRepo
	patternRepo   *mockPatternRepo
	agentRepo     *mockAgentRepo
	graphRepo     *mockGraphRepo
	embeddingSvc  *mockEmbeddingSvc
	extractionSvc *mockExtractionSvc
}

func newTestService() (enrichment.Service, *testDeps) {
	deps := &testDeps{
		jobRepo:       new(mockJobRepo),
		patternRepo:   new(mockPatternRepo),
		agentRepo:     new(mockAgentRepo),
		graphRepo:     new(mockGraphRepo),
		embeddingSvc:  new(mockEmbeddingSvc),
		extractionSvc: new(mockExtractionSvc),
	}

	svc := enrichment.New(
		deps.jobRepo,
		deps.patternRepo,
		deps.agentRepo,
		deps.graphRepo,
		deps.embeddingSvc,
		deps.extractionSvc,
		testConfig(),
		zerolog.Nop(),
	)

	return svc, deps
}

// setupHappyPathMocks configures all mocks for a successful ProcessJob call.
func setupHappyPathMocks(deps *testDeps) {
	pattern := testPattern()
	embedding := testEmbedding()
	concepts := testConcepts()

	deps.patternRepo.On("Get", mock.Anything, testPatternID).Return(pattern, nil)
	deps.embeddingSvc.On("Embed", mock.Anything, pattern.Content).Return(embedding, nil)
	deps.patternRepo.On("UpdateEmbedding", mock.Anything, testPatternID, embedding).Return(nil)
	deps.extractionSvc.On("Extract", mock.Anything, pattern.Content).Return(concepts, nil)
	deps.graphRepo.On("SyncPattern", mock.Anything, mock.MatchedBy(func(p *graphrepo.Pattern) bool {
		return p.ID == testPatternID && p.Name == "test-pattern"
	})).Return(nil)
	deps.graphRepo.On("SyncConcepts", mock.Anything, testPatternID, testGraphConcepts()).Return(nil)

	// Agent associations: one valid agent.
	deps.patternRepo.On("GetAgentAssociations", mock.Anything, testPatternID).Return(
		[]patternrepo.AgentAssociation{
			{AgentID: testAgentID, Relevance: 0.9},
		}, nil,
	)
	deps.agentRepo.On("GetByID", mock.Anything, testAgentID).Return(
		&agentrepo.Agent{ID: testAgentID, Name: "test-agent"}, nil,
	)
	deps.graphRepo.On("SetPatternAgentRelevance", mock.Anything, testPatternID,
		[]graphrepo.AgentAssociation{{AgentName: "test-agent", Relevance: 0.9}},
	).Return(nil)

	deps.graphRepo.On("ComputeRelatedToEdges", mock.Anything, testPatternID, 0.3).Return(nil)
	deps.jobRepo.On("MarkCompleted", mock.Anything, testJobID).Return(nil)
	deps.patternRepo.On("UpdateEnrichmentStatus", mock.Anything, testPatternID, "enriched", (*string)(nil)).Return(nil)
}

// setupFailJobMocks configures mocks for a successful failJob call.
func setupFailJobMocks(deps *testDeps) {
	deps.jobRepo.On("MarkFailed", mock.Anything, testJobID, mock.Anything, 30*time.Second).Return(nil)
	deps.patternRepo.On("UpdateEnrichmentStatus", mock.Anything, testPatternID, "failed", mock.AnythingOfType("*string")).Return(nil)
}

// assertExpectations checks all mock expectations.
func assertExpectations(t *testing.T, deps *testDeps) {
	t.Helper()
	deps.jobRepo.AssertExpectations(t)
	deps.patternRepo.AssertExpectations(t)
	deps.agentRepo.AssertExpectations(t)
	deps.graphRepo.AssertExpectations(t)
	deps.embeddingSvc.AssertExpectations(t)
	deps.extractionSvc.AssertExpectations(t)
}

// ---------- ClaimNextJob ----------

func TestClaimNextJob(t *testing.T) {
	t.Parallel()

	t.Run("happy path returns claimed job", func(t *testing.T) {
		t.Parallel()

		svc, deps := newTestService()
		job := testJob()
		deps.jobRepo.On("ClaimPending", mock.Anything).Return(job, nil)

		result, err := svc.ClaimNextJob(context.Background())

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, testJobID, result.ID)
		assert.Equal(t, testPatternID, result.PatternID)
		assertExpectations(t, deps)
	})

	t.Run("no jobs available returns nil nil", func(t *testing.T) {
		t.Parallel()

		svc, deps := newTestService()
		deps.jobRepo.On("ClaimPending", mock.Anything).Return(nil, nil)

		result, err := svc.ClaimNextJob(context.Background())

		require.NoError(t, err)
		assert.Nil(t, result)
		assertExpectations(t, deps)
	})
}

// ---------- ProcessJob ----------

func TestProcessJob_HappyPath(t *testing.T) {
	t.Parallel()

	svc, deps := newTestService()
	setupHappyPathMocks(deps)

	err := svc.ProcessJob(context.Background(), testJob())

	require.NoError(t, err)
	assertExpectations(t, deps)
}

func TestProcessJob_Step1_LoadPatternFails(t *testing.T) {
	t.Parallel()

	svc, deps := newTestService()
	deps.patternRepo.On("Get", mock.Anything, testPatternID).Return(nil, patternrepo.ErrNotFound)
	setupFailJobMocks(deps)

	err := svc.ProcessJob(context.Background(), testJob())

	require.NoError(t, err, "pipeline failure should return nil when failJob succeeds")
	assertExpectations(t, deps)
}

func TestProcessJob_Step2_EmbeddingFails(t *testing.T) {
	t.Parallel()

	svc, deps := newTestService()
	pattern := testPattern()
	deps.patternRepo.On("Get", mock.Anything, testPatternID).Return(pattern, nil)
	deps.embeddingSvc.On("Embed", mock.Anything, pattern.Content).Return(nil, errors.New("openai unavailable"))
	setupFailJobMocks(deps)

	err := svc.ProcessJob(context.Background(), testJob())

	require.NoError(t, err, "pipeline failure should return nil when failJob succeeds")
	assertExpectations(t, deps)
}

func TestProcessJob_Step3_StoreEmbeddingFails(t *testing.T) {
	t.Parallel()

	svc, deps := newTestService()
	pattern := testPattern()
	embedding := testEmbedding()
	deps.patternRepo.On("Get", mock.Anything, testPatternID).Return(pattern, nil)
	deps.embeddingSvc.On("Embed", mock.Anything, pattern.Content).Return(embedding, nil)
	deps.patternRepo.On("UpdateEmbedding", mock.Anything, testPatternID, embedding).
		Return(errors.New("db error"))
	setupFailJobMocks(deps)

	err := svc.ProcessJob(context.Background(), testJob())

	require.NoError(t, err, "pipeline failure should return nil when failJob succeeds")
	assertExpectations(t, deps)
}

func TestProcessJob_Step4_ExtractionFails(t *testing.T) {
	t.Parallel()

	svc, deps := newTestService()
	pattern := testPattern()
	embedding := testEmbedding()
	deps.patternRepo.On("Get", mock.Anything, testPatternID).Return(pattern, nil)
	deps.embeddingSvc.On("Embed", mock.Anything, pattern.Content).Return(embedding, nil)
	deps.patternRepo.On("UpdateEmbedding", mock.Anything, testPatternID, embedding).Return(nil)
	deps.extractionSvc.On("Extract", mock.Anything, pattern.Content).
		Return(nil, errors.New("extraction failed"))
	setupFailJobMocks(deps)

	err := svc.ProcessJob(context.Background(), testJob())

	require.NoError(t, err, "pipeline failure should return nil when failJob succeeds")
	assertExpectations(t, deps)
}

func TestProcessJob_Step5_SyncPatternFails(t *testing.T) {
	t.Parallel()

	svc, deps := newTestService()
	pattern := testPattern()
	embedding := testEmbedding()
	concepts := testConcepts()
	deps.patternRepo.On("Get", mock.Anything, testPatternID).Return(pattern, nil)
	deps.embeddingSvc.On("Embed", mock.Anything, pattern.Content).Return(embedding, nil)
	deps.patternRepo.On("UpdateEmbedding", mock.Anything, testPatternID, embedding).Return(nil)
	deps.extractionSvc.On("Extract", mock.Anything, pattern.Content).Return(concepts, nil)
	deps.graphRepo.On("SyncPattern", mock.Anything, mock.Anything).
		Return(errors.New("neo4j unavailable"))
	setupFailJobMocks(deps)

	err := svc.ProcessJob(context.Background(), testJob())

	require.NoError(t, err, "pipeline failure should return nil when failJob succeeds")
	assertExpectations(t, deps)
}

func TestProcessJob_Step6_SyncConceptsFails(t *testing.T) {
	t.Parallel()

	svc, deps := newTestService()
	pattern := testPattern()
	embedding := testEmbedding()
	concepts := testConcepts()
	deps.patternRepo.On("Get", mock.Anything, testPatternID).Return(pattern, nil)
	deps.embeddingSvc.On("Embed", mock.Anything, pattern.Content).Return(embedding, nil)
	deps.patternRepo.On("UpdateEmbedding", mock.Anything, testPatternID, embedding).Return(nil)
	deps.extractionSvc.On("Extract", mock.Anything, pattern.Content).Return(concepts, nil)
	deps.graphRepo.On("SyncPattern", mock.Anything, mock.Anything).Return(nil)
	deps.graphRepo.On("SyncConcepts", mock.Anything, testPatternID, testGraphConcepts()).
		Return(errors.New("neo4j unavailable"))
	setupFailJobMocks(deps)

	err := svc.ProcessJob(context.Background(), testJob())

	require.NoError(t, err, "pipeline failure should return nil when failJob succeeds")
	assertExpectations(t, deps)
}

func TestProcessJob_Step7_GetAgentAssociationsFails(t *testing.T) {
	t.Parallel()

	svc, deps := newTestService()
	pattern := testPattern()
	embedding := testEmbedding()
	concepts := testConcepts()
	deps.patternRepo.On("Get", mock.Anything, testPatternID).Return(pattern, nil)
	deps.embeddingSvc.On("Embed", mock.Anything, pattern.Content).Return(embedding, nil)
	deps.patternRepo.On("UpdateEmbedding", mock.Anything, testPatternID, embedding).Return(nil)
	deps.extractionSvc.On("Extract", mock.Anything, pattern.Content).Return(concepts, nil)
	deps.graphRepo.On("SyncPattern", mock.Anything, mock.Anything).Return(nil)
	deps.graphRepo.On("SyncConcepts", mock.Anything, testPatternID, testGraphConcepts()).Return(nil)
	deps.patternRepo.On("GetAgentAssociations", mock.Anything, testPatternID).
		Return(nil, errors.New("db error"))
	setupFailJobMocks(deps)

	err := svc.ProcessJob(context.Background(), testJob())

	require.NoError(t, err, "pipeline failure should return nil when failJob succeeds")
	assertExpectations(t, deps)
}

func TestProcessJob_Step7_SetPatternAgentRelevanceFails(t *testing.T) {
	t.Parallel()

	svc, deps := newTestService()
	pattern := testPattern()
	embedding := testEmbedding()
	concepts := testConcepts()
	deps.patternRepo.On("Get", mock.Anything, testPatternID).Return(pattern, nil)
	deps.embeddingSvc.On("Embed", mock.Anything, pattern.Content).Return(embedding, nil)
	deps.patternRepo.On("UpdateEmbedding", mock.Anything, testPatternID, embedding).Return(nil)
	deps.extractionSvc.On("Extract", mock.Anything, pattern.Content).Return(concepts, nil)
	deps.graphRepo.On("SyncPattern", mock.Anything, mock.Anything).Return(nil)
	deps.graphRepo.On("SyncConcepts", mock.Anything, testPatternID, testGraphConcepts()).Return(nil)
	deps.patternRepo.On("GetAgentAssociations", mock.Anything, testPatternID).Return(
		[]patternrepo.AgentAssociation{{AgentID: testAgentID, Relevance: 0.9}}, nil,
	)
	deps.agentRepo.On("GetByID", mock.Anything, testAgentID).Return(
		&agentrepo.Agent{ID: testAgentID, Name: "test-agent"}, nil,
	)
	deps.graphRepo.On("SetPatternAgentRelevance", mock.Anything, testPatternID, mock.Anything).
		Return(errors.New("neo4j unavailable"))
	setupFailJobMocks(deps)

	err := svc.ProcessJob(context.Background(), testJob())

	require.NoError(t, err, "pipeline failure should return nil when failJob succeeds")
	assertExpectations(t, deps)
}

func TestProcessJob_Step8_ComputeRelatedToEdgesFails(t *testing.T) {
	t.Parallel()

	svc, deps := newTestService()
	pattern := testPattern()
	embedding := testEmbedding()
	concepts := testConcepts()
	deps.patternRepo.On("Get", mock.Anything, testPatternID).Return(pattern, nil)
	deps.embeddingSvc.On("Embed", mock.Anything, pattern.Content).Return(embedding, nil)
	deps.patternRepo.On("UpdateEmbedding", mock.Anything, testPatternID, embedding).Return(nil)
	deps.extractionSvc.On("Extract", mock.Anything, pattern.Content).Return(concepts, nil)
	deps.graphRepo.On("SyncPattern", mock.Anything, mock.Anything).Return(nil)
	deps.graphRepo.On("SyncConcepts", mock.Anything, testPatternID, testGraphConcepts()).Return(nil)
	deps.patternRepo.On("GetAgentAssociations", mock.Anything, testPatternID).Return(
		[]patternrepo.AgentAssociation{{AgentID: testAgentID, Relevance: 0.9}}, nil,
	)
	deps.agentRepo.On("GetByID", mock.Anything, testAgentID).Return(
		&agentrepo.Agent{ID: testAgentID, Name: "test-agent"}, nil,
	)
	deps.graphRepo.On("SetPatternAgentRelevance", mock.Anything, testPatternID, mock.Anything).Return(nil)
	deps.graphRepo.On("ComputeRelatedToEdges", mock.Anything, testPatternID, 0.3).
		Return(errors.New("neo4j unavailable"))
	setupFailJobMocks(deps)

	err := svc.ProcessJob(context.Background(), testJob())

	require.NoError(t, err, "pipeline failure should return nil when failJob succeeds")
	assertExpectations(t, deps)
}

func TestProcessJob_MarkCompletedFails(t *testing.T) {
	t.Parallel()

	svc, deps := newTestService()

	// Set up the full happy path through step 8, but fail on MarkCompleted.
	pattern := testPattern()
	embedding := testEmbedding()
	concepts := testConcepts()
	deps.patternRepo.On("Get", mock.Anything, testPatternID).Return(pattern, nil)
	deps.embeddingSvc.On("Embed", mock.Anything, pattern.Content).Return(embedding, nil)
	deps.patternRepo.On("UpdateEmbedding", mock.Anything, testPatternID, embedding).Return(nil)
	deps.extractionSvc.On("Extract", mock.Anything, pattern.Content).Return(concepts, nil)
	deps.graphRepo.On("SyncPattern", mock.Anything, mock.Anything).Return(nil)
	deps.graphRepo.On("SyncConcepts", mock.Anything, testPatternID, testGraphConcepts()).Return(nil)
	deps.patternRepo.On("GetAgentAssociations", mock.Anything, testPatternID).Return(
		[]patternrepo.AgentAssociation{{AgentID: testAgentID, Relevance: 0.9}}, nil,
	)
	deps.agentRepo.On("GetByID", mock.Anything, testAgentID).Return(
		&agentrepo.Agent{ID: testAgentID, Name: "test-agent"}, nil,
	)
	deps.graphRepo.On("SetPatternAgentRelevance", mock.Anything, testPatternID, mock.Anything).Return(nil)
	deps.graphRepo.On("ComputeRelatedToEdges", mock.Anything, testPatternID, 0.3).Return(nil)
	deps.jobRepo.On("MarkCompleted", mock.Anything, testJobID).Return(errors.New("db error"))

	err := svc.ProcessJob(context.Background(), testJob())

	require.Error(t, err, "unrecoverable failure should return non-nil error")
	assert.Contains(t, err.Error(), "mark job completed")
	assertExpectations(t, deps)
}

func TestProcessJob_UpdateEnrichmentStatusAfterCompletionFails(t *testing.T) {
	t.Parallel()

	svc, deps := newTestService()

	// Set up the full happy path through MarkCompleted, but fail on UpdateEnrichmentStatus.
	pattern := testPattern()
	embedding := testEmbedding()
	concepts := testConcepts()
	deps.patternRepo.On("Get", mock.Anything, testPatternID).Return(pattern, nil)
	deps.embeddingSvc.On("Embed", mock.Anything, pattern.Content).Return(embedding, nil)
	deps.patternRepo.On("UpdateEmbedding", mock.Anything, testPatternID, embedding).Return(nil)
	deps.extractionSvc.On("Extract", mock.Anything, pattern.Content).Return(concepts, nil)
	deps.graphRepo.On("SyncPattern", mock.Anything, mock.Anything).Return(nil)
	deps.graphRepo.On("SyncConcepts", mock.Anything, testPatternID, testGraphConcepts()).Return(nil)
	deps.patternRepo.On("GetAgentAssociations", mock.Anything, testPatternID).Return(
		[]patternrepo.AgentAssociation{{AgentID: testAgentID, Relevance: 0.9}}, nil,
	)
	deps.agentRepo.On("GetByID", mock.Anything, testAgentID).Return(
		&agentrepo.Agent{ID: testAgentID, Name: "test-agent"}, nil,
	)
	deps.graphRepo.On("SetPatternAgentRelevance", mock.Anything, testPatternID, mock.Anything).Return(nil)
	deps.graphRepo.On("ComputeRelatedToEdges", mock.Anything, testPatternID, 0.3).Return(nil)
	deps.jobRepo.On("MarkCompleted", mock.Anything, testJobID).Return(nil)
	deps.patternRepo.On("UpdateEnrichmentStatus", mock.Anything, testPatternID, "enriched", (*string)(nil)).
		Return(errors.New("db error"))

	err := svc.ProcessJob(context.Background(), testJob())

	require.Error(t, err, "unrecoverable failure should return non-nil error")
	assert.Contains(t, err.Error(), "update enrichment status")
	assertExpectations(t, deps)
}

// ---------- failJob ----------

func TestFailJob_MarkFailedFails(t *testing.T) {
	t.Parallel()

	svc, deps := newTestService()

	// Step 1 fails (pattern not found), triggering failJob.
	// failJob's MarkFailed also fails => returns error.
	deps.patternRepo.On("Get", mock.Anything, testPatternID).Return(nil, patternrepo.ErrNotFound)
	deps.jobRepo.On("MarkFailed", mock.Anything, testJobID, mock.Anything, 30*time.Second).
		Return(errors.New("db error"))

	err := svc.ProcessJob(context.Background(), testJob())

	require.Error(t, err, "should return error when MarkFailed fails")
	assert.Contains(t, err.Error(), "mark job failed")
	assertExpectations(t, deps)
}

func TestFailJob_UpdateEnrichmentStatusFails(t *testing.T) {
	t.Parallel()

	svc, deps := newTestService()

	// Step 1 fails, triggering failJob. MarkFailed succeeds,
	// but UpdateEnrichmentStatus fails => returns error.
	deps.patternRepo.On("Get", mock.Anything, testPatternID).Return(nil, patternrepo.ErrNotFound)
	deps.jobRepo.On("MarkFailed", mock.Anything, testJobID, mock.Anything, 30*time.Second).Return(nil)
	deps.patternRepo.On("UpdateEnrichmentStatus", mock.Anything, testPatternID, "failed", mock.AnythingOfType("*string")).
		Return(errors.New("db error"))

	err := svc.ProcessJob(context.Background(), testJob())

	require.Error(t, err, "should return error when UpdateEnrichmentStatus fails in failJob")
	assert.Contains(t, err.Error(), "update enrichment status")
	assertExpectations(t, deps)
}

// ---------- ProcessJob: agent not found is skipped ----------

func TestProcessJob_AgentNotFoundSkippedDuringAssociationSync(t *testing.T) {
	t.Parallel()

	svc, deps := newTestService()
	pattern := testPattern()
	embedding := testEmbedding()
	concepts := testConcepts()
	missingAgentID := uuid.MustParse("44444444-4444-4444-4444-444444444444")

	deps.patternRepo.On("Get", mock.Anything, testPatternID).Return(pattern, nil)
	deps.embeddingSvc.On("Embed", mock.Anything, pattern.Content).Return(embedding, nil)
	deps.patternRepo.On("UpdateEmbedding", mock.Anything, testPatternID, embedding).Return(nil)
	deps.extractionSvc.On("Extract", mock.Anything, pattern.Content).Return(concepts, nil)
	deps.graphRepo.On("SyncPattern", mock.Anything, mock.Anything).Return(nil)
	deps.graphRepo.On("SyncConcepts", mock.Anything, testPatternID, testGraphConcepts()).Return(nil)

	// Two associations: one resolvable, one not found (should be skipped).
	deps.patternRepo.On("GetAgentAssociations", mock.Anything, testPatternID).Return(
		[]patternrepo.AgentAssociation{
			{AgentID: testAgentID, Relevance: 0.9},
			{AgentID: missingAgentID, Relevance: 0.5},
		}, nil,
	)
	deps.agentRepo.On("GetByID", mock.Anything, testAgentID).Return(
		&agentrepo.Agent{ID: testAgentID, Name: "test-agent"}, nil,
	)
	deps.agentRepo.On("GetByID", mock.Anything, missingAgentID).Return(nil, agentrepo.ErrNotFound)

	// Only the resolvable agent should be synced.
	deps.graphRepo.On("SetPatternAgentRelevance", mock.Anything, testPatternID,
		[]graphrepo.AgentAssociation{{AgentName: "test-agent", Relevance: 0.9}},
	).Return(nil)

	deps.graphRepo.On("ComputeRelatedToEdges", mock.Anything, testPatternID, 0.3).Return(nil)
	deps.jobRepo.On("MarkCompleted", mock.Anything, testJobID).Return(nil)
	deps.patternRepo.On("UpdateEnrichmentStatus", mock.Anything, testPatternID, "enriched", (*string)(nil)).Return(nil)

	err := svc.ProcessJob(context.Background(), testJob())

	require.NoError(t, err)
	assertExpectations(t, deps)
}

// ---------- ReclaimStaleJobs ----------

func TestReclaimStaleJobs(t *testing.T) {
	t.Parallel()

	svc, deps := newTestService()
	deps.jobRepo.On("ReclaimStale", mock.Anything, 5*time.Minute).Return(int64(3), nil)

	count, err := svc.ReclaimStaleJobs(context.Background())

	require.NoError(t, err)
	assert.Equal(t, int64(3), count)
	assertExpectations(t, deps)
}

func TestReclaimStaleJobs_Error(t *testing.T) {
	t.Parallel()

	svc, deps := newTestService()
	deps.jobRepo.On("ReclaimStale", mock.Anything, 5*time.Minute).Return(int64(0), errors.New("db error"))

	count, err := svc.ReclaimStaleJobs(context.Background())

	require.Error(t, err)
	assert.Equal(t, int64(0), count)
	assertExpectations(t, deps)
}

// ---------- CleanupCompletedJobs ----------

func TestCleanupCompletedJobs(t *testing.T) {
	t.Parallel()

	svc, deps := newTestService()
	deps.jobRepo.On("DeleteCompleted", mock.Anything, 168*time.Hour).Return(int64(5), nil)

	count, err := svc.CleanupCompletedJobs(context.Background())

	require.NoError(t, err)
	assert.Equal(t, int64(5), count)
	assertExpectations(t, deps)
}

func TestCleanupCompletedJobs_Error(t *testing.T) {
	t.Parallel()

	svc, deps := newTestService()
	deps.jobRepo.On("DeleteCompleted", mock.Anything, 168*time.Hour).Return(int64(0), errors.New("db error"))

	count, err := svc.CleanupCompletedJobs(context.Background())

	require.Error(t, err)
	assert.Equal(t, int64(0), count)
	assertExpectations(t, deps)
}

// ---------- CleanupFailedJobs ----------

func TestCleanupFailedJobs(t *testing.T) {
	t.Parallel()

	svc, deps := newTestService()
	deps.jobRepo.On("DeleteFailed", mock.Anything, 720*time.Hour).Return(int64(2), nil)

	count, err := svc.CleanupFailedJobs(context.Background())

	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
	assertExpectations(t, deps)
}

func TestCleanupFailedJobs_Error(t *testing.T) {
	t.Parallel()

	svc, deps := newTestService()
	deps.jobRepo.On("DeleteFailed", mock.Anything, 720*time.Hour).Return(int64(0), errors.New("db error"))

	count, err := svc.CleanupFailedJobs(context.Background())

	require.Error(t, err)
	assert.Equal(t, int64(0), count)
	assertExpectations(t, deps)
}
