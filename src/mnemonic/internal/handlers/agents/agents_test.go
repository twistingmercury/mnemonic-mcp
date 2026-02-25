package agents_test

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
	"github.com/twistingmercury/mnemonic/internal/handlers/agents"
	agentrepo "github.com/twistingmercury/mnemonic/internal/repository/agent"
	"github.com/twistingmercury/mnemonic/internal/service"
	agentsvc "github.com/twistingmercury/mnemonic/internal/service/agent"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// mockAgentService implements agentsvc.Service for testing.
type mockAgentService struct {
	mock.Mock
}

func (m *mockAgentService) Create(ctx context.Context, input agentsvc.CreateInput) (*agentrepo.Agent, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*agentrepo.Agent), args.Error(1)
}

func (m *mockAgentService) Get(ctx context.Context, name string) (*agentrepo.Agent, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*agentrepo.Agent), args.Error(1)
}

func (m *mockAgentService) Update(ctx context.Context, name string, input agentsvc.UpdateInput) (*agentrepo.Agent, error) {
	args := m.Called(ctx, name, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*agentrepo.Agent), args.Error(1)
}

func (m *mockAgentService) Delete(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *mockAgentService) List(ctx context.Context, opts agentsvc.ListOptions) ([]*agentrepo.Agent, int64, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*agentrepo.Agent), args.Get(1).(int64), args.Error(2)
}

func (m *mockAgentService) GetManifest(ctx context.Context) ([]agentrepo.ManifestEntry, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]agentrepo.ManifestEntry), args.Error(1)
}

func newTestRouter(svc agentsvc.Service) *gin.Engine {
	router := gin.New()
	h := agents.New(svc)
	v1 := router.Group("/v1/api")
	h.RegisterRoutes(v1)
	return router
}

func makeAgent(name string) *agentrepo.Agent {
	def, _ := json.Marshal(map[string]any{
		"description":   "Test agent description",
		"system_prompt": "You are a test agent",
		"model":         "sonnet",
		"allowed_tools": []string{"Read", "Write"},
		"version":       "1.0.0",
	})
	return &agentrepo.Agent{
		ID:         uuid.New(),
		Name:       name,
		Definition: def,
		CRC64:      "123456789",
		CreatedAt:  time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
		UpdatedAt:  time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
	}
}

func TestCreate_Success(t *testing.T) {
	t.Parallel()
	svc := new(mockAgentService)
	router := newTestRouter(svc)

	agent := makeAgent("test-agent")
	svc.On("Create", mock.Anything, mock.AnythingOfType("agent.CreateInput")).Return(agent, nil)

	body := `{
		"name": "test-agent",
		"description": "Test agent description",
		"system_prompt": "You are a test agent",
		"model": "sonnet",
		"allowed_tools": ["Read", "Write"],
		"version": "1.0.0"
	}`

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/api/agents", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "/v1/api/agents/test-agent")

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "test-agent", resp["name"])
	assert.Equal(t, "sonnet", resp["model"])
}

func TestCreate_Conflict(t *testing.T) {
	t.Parallel()
	svc := new(mockAgentService)
	router := newTestRouter(svc)

	svc.On("Create", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("%w: agent %q", service.ErrConflict, "test-agent"))

	body := `{
		"name": "test-agent",
		"description": "desc",
		"system_prompt": "prompt",
		"model": "sonnet",
		"version": "1.0.0"
	}`

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/api/agents", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestCreate_BadRequest(t *testing.T) {
	t.Parallel()
	svc := new(mockAgentService)
	router := newTestRouter(svc)

	// Missing required fields.
	body := `{"name": "test"}`

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/api/agents", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGet_Success(t *testing.T) {
	t.Parallel()
	svc := new(mockAgentService)
	router := newTestRouter(svc)

	agent := makeAgent("test-agent")
	svc.On("Get", mock.Anything, "test-agent").Return(agent, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/api/agents/test-agent", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "test-agent", resp["name"])
}

func TestGet_NotFound(t *testing.T) {
	t.Parallel()
	svc := new(mockAgentService)
	router := newTestRouter(svc)

	svc.On("Get", mock.Anything, "unknown").Return(nil, fmt.Errorf("%w: agent %q", service.ErrNotFound, "unknown"))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/api/agents/unknown", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestList_Success(t *testing.T) {
	t.Parallel()
	svc := new(mockAgentService)
	router := newTestRouter(svc)

	a1 := makeAgent("agent-a")
	a2 := makeAgent("agent-b")
	svc.On("List", mock.Anything, mock.AnythingOfType("agent.ListOptions")).
		Return([]*agentrepo.Agent{a1, a2}, int64(2), nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/api/agents?limit=100", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].([]any)
	assert.Len(t, data, 2)
	pagination := resp["pagination"].(map[string]any)
	assert.False(t, pagination["has_more"].(bool))
}

func TestUpdate_Success(t *testing.T) {
	t.Parallel()
	svc := new(mockAgentService)
	router := newTestRouter(svc)

	agent := makeAgent("test-agent")
	svc.On("Update", mock.Anything, "test-agent", mock.AnythingOfType("agent.UpdateInput")).Return(agent, nil)

	body := `{
		"description": "Updated",
		"system_prompt": "Updated prompt",
		"model": "opus",
		"version": "2.0.0"
	}`

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/v1/api/agents/test-agent", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestUpdate_NameMismatch(t *testing.T) {
	t.Parallel()
	svc := new(mockAgentService)
	router := newTestRouter(svc)

	body := `{
		"name": "different-name",
		"description": "Updated",
		"system_prompt": "Updated prompt",
		"model": "opus",
		"version": "2.0.0"
	}`

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/v1/api/agents/test-agent", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdate_NotFound(t *testing.T) {
	t.Parallel()
	svc := new(mockAgentService)
	router := newTestRouter(svc)

	svc.On("Update", mock.Anything, "unknown", mock.Anything).
		Return(nil, fmt.Errorf("%w: agent %q", service.ErrNotFound, "unknown"))

	body := `{
		"description": "Updated",
		"system_prompt": "Updated prompt",
		"model": "opus",
		"version": "2.0.0"
	}`

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/v1/api/agents/unknown", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDelete_Success(t *testing.T) {
	t.Parallel()
	svc := new(mockAgentService)
	router := newTestRouter(svc)

	svc.On("Delete", mock.Anything, "test-agent").Return(nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/v1/api/agents/test-agent", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestDelete_NotFound(t *testing.T) {
	t.Parallel()
	svc := new(mockAgentService)
	router := newTestRouter(svc)

	svc.On("Delete", mock.Anything, "unknown").
		Return(fmt.Errorf("%w: agent %q", service.ErrNotFound, "unknown"))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/v1/api/agents/unknown", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
