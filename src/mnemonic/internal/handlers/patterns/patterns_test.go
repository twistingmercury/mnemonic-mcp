package patterns_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/twistingmercury/mnemonic/internal/handlers/patterns"
	patternrepo "github.com/twistingmercury/mnemonic/internal/repository/pattern"
	"github.com/twistingmercury/mnemonic/internal/service"
	patternsvc "github.com/twistingmercury/mnemonic/internal/service/pattern"
	searchsvc "github.com/twistingmercury/mnemonic/internal/service/search"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// --- Mock PatternService ---

type mockPatternService struct {
	mock.Mock
}

func (m *mockPatternService) Create(ctx context.Context, input patternsvc.CreateInput) (*patternrepo.Pattern, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*patternrepo.Pattern), args.Error(1)
}

func (m *mockPatternService) Get(ctx context.Context, id uuid.UUID) (*patternrepo.Pattern, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*patternrepo.Pattern), args.Error(1)
}

func (m *mockPatternService) GetWithGraph(ctx context.Context, id uuid.UUID) (*patternrepo.Pattern, *patternsvc.GraphContext, error) {
	args := m.Called(ctx, id)
	var p *patternrepo.Pattern
	var g *patternsvc.GraphContext
	if args.Get(0) != nil {
		p = args.Get(0).(*patternrepo.Pattern)
	}
	if args.Get(1) != nil {
		g = args.Get(1).(*patternsvc.GraphContext)
	}
	return p, g, args.Error(2)
}

func (m *mockPatternService) Update(ctx context.Context, id uuid.UUID, input patternsvc.UpdateInput) (*patternrepo.Pattern, error) {
	args := m.Called(ctx, id, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*patternrepo.Pattern), args.Error(1)
}

func (m *mockPatternService) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockPatternService) List(ctx context.Context, filter patternrepo.Filter, opts patternsvc.ListOptions) ([]*patternrepo.Pattern, int64, error) {
	args := m.Called(ctx, filter, opts)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*patternrepo.Pattern), args.Get(1).(int64), args.Error(2)
}

func (m *mockPatternService) SetAgentAssociations(ctx context.Context, patternID uuid.UUID, associations []patternsvc.AssociationInput) error {
	args := m.Called(ctx, patternID, associations)
	return args.Error(0)
}

func (m *mockPatternService) GetAgentAssociations(ctx context.Context, patternID uuid.UUID) ([]patternrepo.AgentAssociation, error) {
	args := m.Called(ctx, patternID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]patternrepo.AgentAssociation), args.Error(1)
}

func (m *mockPatternService) FindRelated(ctx context.Context, patternID uuid.UUID, limit int) ([]patternsvc.RelatedPatternResult, error) {
	args := m.Called(ctx, patternID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]patternsvc.RelatedPatternResult), args.Error(1)
}

// --- Mock SearchService ---

type mockSearchService struct {
	mock.Mock
}

func (m *mockSearchService) SearchPatterns(ctx context.Context, opts searchsvc.SearchOptions) (*searchsvc.SearchResult, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*searchsvc.SearchResult), args.Error(1)
}

// --- Helpers ---

func newTestRouter(psvc patternsvc.Service, ssvc searchsvc.Service) *gin.Engine {
	router := gin.New()
	h := patterns.New(psvc, ssvc)
	v1 := router.Group("/v1/api")
	h.RegisterRoutes(v1)
	return router
}

func makePattern(name string) *patternrepo.Pattern {
	desc := "Test pattern description"
	return &patternrepo.Pattern{
		ID:               uuid.New(),
		Name:             name,
		Description:      &desc,
		Content:          "# Test Pattern\n\nContent here",
		Tags:             []string{"go", "test"},
		EnrichmentStatus: "pending",
		CreatedAt:        time.Date(2026, 1, 10, 8, 0, 0, 0, time.UTC),
		UpdatedAt:        time.Date(2026, 1, 10, 8, 0, 0, 0, time.UTC),
	}
}

// --- Tests ---

func TestPatternCreate_Success(t *testing.T) {
	t.Parallel()
	psvc := new(mockPatternService)
	ssvc := new(mockSearchService)
	router := newTestRouter(psvc, ssvc)

	pattern := makePattern("go-error-handling")
	psvc.On("Create", mock.Anything, mock.AnythingOfType("pattern.CreateInput")).Return(pattern, nil)

	body := `{
		"name": "go-error-handling",
		"description": "Test pattern description",
		"content": "# Test Pattern\n\nContent here",
		"tags": ["go", "test"]
	}`

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/api/patterns", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "/v1/api/patterns/")

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "go-error-handling", resp["name"])
	assert.Equal(t, "pending", resp["enrichment_status"])
}

func TestPatternCreate_Conflict(t *testing.T) {
	t.Parallel()
	psvc := new(mockPatternService)
	ssvc := new(mockSearchService)
	router := newTestRouter(psvc, ssvc)

	psvc.On("Create", mock.Anything, mock.Anything).
		Return(nil, fmt.Errorf("%w: pattern %q", service.ErrConflict, "go-error-handling"))

	body := `{
		"name": "go-error-handling",
		"content": "content"
	}`

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/api/patterns", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestPatternCreate_BadRequest(t *testing.T) {
	t.Parallel()
	psvc := new(mockPatternService)
	ssvc := new(mockSearchService)
	router := newTestRouter(psvc, ssvc)

	// Missing required content field.
	body := `{"name": "test"}`

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/api/patterns", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPatternGet_Success(t *testing.T) {
	t.Parallel()
	psvc := new(mockPatternService)
	ssvc := new(mockSearchService)
	router := newTestRouter(psvc, ssvc)

	pattern := makePattern("go-error-handling")
	psvc.On("GetWithGraph", mock.Anything, pattern.ID).Return(pattern, (*patternsvc.GraphContext)(nil), nil)
	psvc.On("GetAgentAssociations", mock.Anything, pattern.ID).
		Return([]patternrepo.AgentAssociation{}, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/api/patterns/"+pattern.ID.String(), nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "go-error-handling", resp["name"])
}

func TestPatternGet_InvalidUUID(t *testing.T) {
	t.Parallel()
	psvc := new(mockPatternService)
	ssvc := new(mockSearchService)
	router := newTestRouter(psvc, ssvc)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/api/patterns/not-a-uuid", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPatternGet_NotFound(t *testing.T) {
	t.Parallel()
	psvc := new(mockPatternService)
	ssvc := new(mockSearchService)
	router := newTestRouter(psvc, ssvc)

	id := uuid.New()
	psvc.On("GetWithGraph", mock.Anything, id).
		Return(nil, nil, fmt.Errorf("%w: pattern %s", service.ErrNotFound, id))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/api/patterns/"+id.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPatternList_Success(t *testing.T) {
	t.Parallel()
	psvc := new(mockPatternService)
	ssvc := new(mockSearchService)
	router := newTestRouter(psvc, ssvc)

	p1 := makePattern("pattern-a")
	p2 := makePattern("pattern-b")
	psvc.On("List", mock.Anything, mock.Anything, mock.Anything).
		Return([]*patternrepo.Pattern{p1, p2}, int64(2), nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/api/patterns?limit=20", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].([]any)
	assert.Len(t, data, 2)
}

func TestPatternUpdate_Success(t *testing.T) {
	t.Parallel()
	psvc := new(mockPatternService)
	ssvc := new(mockSearchService)
	router := newTestRouter(psvc, ssvc)

	pattern := makePattern("go-error-handling")
	psvc.On("Update", mock.Anything, pattern.ID, mock.Anything).Return(pattern, nil)

	body := `{
		"name": "go-error-handling",
		"content": "Updated content"
	}`

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/v1/api/patterns/"+pattern.ID.String(), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestPatternDelete_Success(t *testing.T) {
	t.Parallel()
	psvc := new(mockPatternService)
	ssvc := new(mockSearchService)
	router := newTestRouter(psvc, ssvc)

	id := uuid.New()
	psvc.On("Delete", mock.Anything, id).Return(nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/v1/api/patterns/"+id.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestPatternDelete_NotFound(t *testing.T) {
	t.Parallel()
	psvc := new(mockPatternService)
	ssvc := new(mockSearchService)
	router := newTestRouter(psvc, ssvc)

	id := uuid.New()
	psvc.On("Delete", mock.Anything, id).
		Return(fmt.Errorf("%w: pattern %s", service.ErrNotFound, id))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/v1/api/patterns/"+id.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSetAgentAssociations_Success(t *testing.T) {
	t.Parallel()
	psvc := new(mockPatternService)
	ssvc := new(mockSearchService)
	router := newTestRouter(psvc, ssvc)

	id := uuid.New()
	psvc.On("SetAgentAssociations", mock.Anything, id, mock.Anything).Return(nil)

	body := `{
		"associations": [
			{"agent_name": "go-software-engineer", "relevance": 0.95}
		]
	}`

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/v1/api/patterns/"+id.String()+"/agents", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assocs := resp["associations"].([]any)
	assert.Len(t, assocs, 1)
}

func TestGetAgentAssociations_Success(t *testing.T) {
	t.Parallel()
	psvc := new(mockPatternService)
	ssvc := new(mockSearchService)
	router := newTestRouter(psvc, ssvc)

	id := uuid.New()
	agentID := uuid.New()
	psvc.On("GetAgentAssociations", mock.Anything, id).
		Return([]patternrepo.AgentAssociation{
			{AgentID: agentID, Relevance: 0.95},
		}, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/api/patterns/"+id.String()+"/agents", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assocs := resp["associations"].([]any)
	assert.Len(t, assocs, 1)
}

func TestSearch_Success(t *testing.T) {
	t.Parallel()
	psvc := new(mockPatternService)
	ssvc := new(mockSearchService)
	router := newTestRouter(psvc, ssvc)

	pattern := makePattern("go-error-handling")
	ssvc.On("SearchPatterns", mock.Anything, mock.AnythingOfType("search.SearchOptions")).
		Return(&searchsvc.SearchResult{
			Matches: []*patternrepo.Match{
				{Pattern: pattern, Similarity: 0.92},
			},
			Query:            "error handling",
			TotalCandidates:  47,
			SearchDurationMs: 23,
		}, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/api/patterns/search?q=error+handling", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	results := resp["results"].([]any)
	assert.Len(t, results, 1)
	metadata := resp["metadata"].(map[string]any)
	assert.Equal(t, "error handling", metadata["query"])
}

func TestSearch_MissingQuery(t *testing.T) {
	t.Parallel()
	psvc := new(mockPatternService)
	ssvc := new(mockSearchService)
	router := newTestRouter(psvc, ssvc)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/api/patterns/search", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSearch_ServiceUnavailable(t *testing.T) {
	t.Parallel()
	psvc := new(mockPatternService)
	ssvc := new(mockSearchService)
	router := newTestRouter(psvc, ssvc)

	ssvc.On("SearchPatterns", mock.Anything, mock.Anything).
		Return(nil, fmt.Errorf("%w: embedding service down", service.ErrServiceUnavailable))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/api/patterns/search?q=test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}
