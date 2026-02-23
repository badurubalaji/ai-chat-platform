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
	"github.com/mdp/ai-chat-platform/backend/internal/utils"
	"github.com/stretchr/testify/assert"
)

// InMemoryStore is a simple in-memory implementation of store.Store for integration testing
type InMemoryStore struct {
	configs       map[string]*models.ProviderConfig
	conversations map[uuid.UUID]*models.Conversation
	messages      map[uuid.UUID][]*models.Message
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		configs:       make(map[string]*models.ProviderConfig),
		conversations: make(map[uuid.UUID]*models.Conversation),
		messages:      make(map[uuid.UUID][]*models.Message),
	}
}

func (s *InMemoryStore) GetProviderConfig(ctx context.Context, tenantID string) (*models.ProviderConfig, error) {
	if cfg, ok := s.configs[tenantID]; ok {
		// Return a copy to avoid modification by caller affecting store (simulating DB)
		copy := *cfg
		return &copy, nil
	}
	return nil, assert.AnError
}

func (s *InMemoryStore) SaveProviderConfig(ctx context.Context, config *models.ProviderConfig) error {
	if config.ID == uuid.Nil {
		config.ID = uuid.New()
	}
	// Store a deep copy to avoid caller mutations (e.g., nil-ing APIKeyEncrypted)
	cp := *config
	cp.APIKeyEncrypted = make([]byte, len(config.APIKeyEncrypted))
	copy(cp.APIKeyEncrypted, config.APIKeyEncrypted)
	s.configs[config.TenantID] = &cp
	return nil
}

func (s *InMemoryStore) CreateConversation(ctx context.Context, convo *models.Conversation) error {
	s.conversations[convo.ID] = convo
	return nil
}

func (s *InMemoryStore) GetConversation(ctx context.Context, id uuid.UUID) (*models.Conversation, error) {
	if convo, ok := s.conversations[id]; ok {
		return convo, nil
	}
	return nil, assert.AnError
}

func (s *InMemoryStore) ListConversations(ctx context.Context, tenantID string, userID uuid.UUID) ([]*models.Conversation, error) {
	var convos []*models.Conversation
	for _, c := range s.conversations {
		if c.TenantID == tenantID && c.UserID == userID {
			convos = append(convos, c)
		}
	}
	return convos, nil
}

func (s *InMemoryStore) CreateMessage(ctx context.Context, msg *models.Message) error {
	s.messages[msg.ConversationID] = append(s.messages[msg.ConversationID], msg)
	return nil
}

func (s *InMemoryStore) ListMessages(ctx context.Context, conversationID uuid.UUID) ([]*models.Message, error) {
	return s.messages[conversationID], nil
}

func (s *InMemoryStore) LogUsage(ctx context.Context, usage *models.UsageLog) error {
	return nil
}

func (s *InMemoryStore) GetUsageStats(ctx context.Context, tenantID string, days int) (*models.UsageStats, error) {
	return &models.UsageStats{}, nil
}

func (s *InMemoryStore) DeleteProviderConfig(ctx context.Context, tenantID string) error {
	delete(s.configs, tenantID)
	return nil
}

func (s *InMemoryStore) DeleteConversation(ctx context.Context, id uuid.UUID) error {
	delete(s.conversations, id)
	delete(s.messages, id)
	return nil
}

func (s *InMemoryStore) LogToolExecution(ctx context.Context, exec *models.ToolExecution) error {
	return nil
}

func TestIntegration_ConfigFlow(t *testing.T) {
	// Setup
	memStore := NewInMemoryStore()
	cfg := &config.Config{
		AIEncryptionKey: "00000000000000000000000000000000",
	}
	handler := NewHandler(memStore, cfg, nil)
	server := httptest.NewServer(handler)
	defer server.Close()

	client := server.Client()

	// 1. Get Config (should be empty initially)
	resp, err := client.Get(server.URL + "/api/v1/ai/config")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var initialConfig map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&initialConfig)
	assert.Equal(t, false, initialConfig["enabled"])

	// 2. Save Config
	newConfig := map[string]interface{}{
		"provider": "openai",
		"model":    "gpt-4",
		"apiKey":   "sk-test-key",
		"enabled":  true,
		"settings": map[string]interface{}{"temperature": 0.7},
	}
	body, _ := json.Marshal(newConfig)
	resp, err = client.Post(server.URL+"/api/v1/ai/config", "application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// 3. Get Config Again (should verify saved data)
	resp, err = client.Get(server.URL + "/api/v1/ai/config")
	assert.NoError(t, err)

	var savedConfig models.ProviderConfig
	json.NewDecoder(resp.Body).Decode(&savedConfig)
	assert.Equal(t, "openai", savedConfig.Provider)
	assert.Equal(t, "gpt-4", savedConfig.Model)
	assert.True(t, savedConfig.Enabled)
	// Check store has it
	stored, _ := memStore.GetProviderConfig(context.Background(), "default-tenant")
	assert.NotNil(t, stored)
	// Verify encryption
	assert.NotEqual(t, "sk-test-key", string(stored.APIKeyEncrypted))

	// decrypt to verify
	decrypted, err := utils.Decrypt(stored.APIKeyEncrypted, cfg.AIEncryptionKey)
	assert.NoError(t, err)
	assert.Equal(t, "sk-test-key", string(decrypted))
}

func TestIntegration_ConversationFlow(t *testing.T) {
	memStore := NewInMemoryStore()
	handler := NewHandler(memStore, &config.Config{
		AIEncryptionKey: "00000000000000000000000000000000",
	}, nil)
	server := httptest.NewServer(handler)
	defer server.Close()

	// Pre-seed some data
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	convoID := uuid.New()
	memStore.CreateConversation(context.Background(), &models.Conversation{
		ID:        convoID,
		TenantID:  "default-tenant",
		UserID:    userID,
		Title:     "Existing Chat",
		CreatedAt: time.Now(),
	})

	client := server.Client()

	// List Conversations
	resp, err := client.Get(server.URL + "/api/v1/ai/conversations")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var convos []*models.Conversation
	json.NewDecoder(resp.Body).Decode(&convos)
	assert.Len(t, convos, 1)
	assert.Equal(t, "Existing Chat", convos[0].Title)
}
