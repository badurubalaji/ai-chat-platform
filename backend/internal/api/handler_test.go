package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mdp/ai-chat-platform/backend/internal/config"
	"github.com/mdp/ai-chat-platform/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockStore is a mock implementation of store.Store
type MockStore struct {
	mock.Mock
}

func (m *MockStore) GetProviderConfig(ctx context.Context, tenantID string) (*models.ProviderConfig, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ProviderConfig), args.Error(1)
}

func (m *MockStore) SaveProviderConfig(ctx context.Context, config *models.ProviderConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

func (m *MockStore) CreateConversation(ctx context.Context, convo *models.Conversation) error {
	args := m.Called(ctx, convo)
	return args.Error(0)
}

func (m *MockStore) GetConversation(ctx context.Context, id uuid.UUID) (*models.Conversation, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.Conversation), args.Error(1)
}

func (m *MockStore) ListConversations(ctx context.Context, tenantID string, userID uuid.UUID) ([]*models.Conversation, error) {
	args := m.Called(ctx, tenantID, userID)
	return args.Get(0).([]*models.Conversation), args.Error(1)
}

func (m *MockStore) CreateMessage(ctx context.Context, msg *models.Message) error {
	args := m.Called(ctx, msg)
	return args.Error(0)
}

func (m *MockStore) ListMessages(ctx context.Context, conversationID uuid.UUID) ([]*models.Message, error) {
	args := m.Called(ctx, conversationID)
	return args.Get(0).([]*models.Message), args.Error(1)
}

func (m *MockStore) LogUsage(ctx context.Context, usage *models.UsageLog) error {
	args := m.Called(ctx, usage)
	return args.Error(0)
}

func (m *MockStore) GetUsageStats(ctx context.Context, tenantID string, days int) (*models.UsageStats, error) {
	args := m.Called(ctx, tenantID, days)
	return args.Get(0).(*models.UsageStats), args.Error(1)
}

func (m *MockStore) DeleteProviderConfig(ctx context.Context, tenantID string) error {
	args := m.Called(ctx, tenantID)
	return args.Error(0)
}

func (m *MockStore) DeleteConversation(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestHandleConfig_Get(t *testing.T) {
	mockStore := new(MockStore)
	cfg := &config.Config{
		AIEncryptionKey: "00000000000000000000000000000000",
	}
	handler := NewHandler(mockStore, cfg)

	mockStore.On("GetProviderConfig", mock.Anything, "default-tenant").Return(&models.ProviderConfig{
		Provider:        "openai",
		Model:           "gpt-4",
		APIKeyEncrypted: []byte("secret"),
		Enabled:         true,
	}, nil)

	req := httptest.NewRequest("GET", "/api/v1/ai/config", nil)
	w := httptest.NewRecorder()

	handler.handleConfig(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp models.ProviderConfig
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "openai", resp.Provider)
	assert.Empty(t, resp.APIKeyEncrypted, "API key should be masked")
}

func TestHandleConfig_Post(t *testing.T) {
	mockStore := new(MockStore)
	cfg := &config.Config{
		AIEncryptionKey: "00000000000000000000000000000000",
	}
	handler := NewHandler(mockStore, cfg)

	payload := map[string]interface{}{
		"provider": "claude",
		"model":    "claude-3-opus",
		"apiKey":   "sk-test-key",
		"enabled":  true,
	}
	body, _ := json.Marshal(payload)

	mockStore.On("SaveProviderConfig", mock.Anything, mock.MatchedBy(func(c *models.ProviderConfig) bool {
		return c.Provider == "claude" && c.Model == "claude-3-opus" && len(c.APIKeyEncrypted) > 0
	})).Return(nil)

	req := httptest.NewRequest("POST", "/api/v1/ai/config", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler.handleConfig(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockStore.AssertExpectations(t)
}

func TestHandleConversations_List(t *testing.T) {
	mockStore := new(MockStore)
	cfg := &config.Config{}
	handler := NewHandler(mockStore, cfg)

	expectedConvos := []*models.Conversation{
		{ID: uuid.New(), Title: "Test Chat 1", CreatedAt: time.Now()},
		{ID: uuid.New(), Title: "Test Chat 2", CreatedAt: time.Now()},
	}

	mockStore.On("ListConversations", mock.Anything, "default-tenant", mock.Anything).Return(expectedConvos, nil)

	req := httptest.NewRequest("GET", "/api/v1/ai/conversations", nil)
	w := httptest.NewRecorder()

	handler.handleConversations(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp []*models.Conversation
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Len(t, resp, 2)
	assert.Equal(t, "Test Chat 1", resp[0].Title)
}
