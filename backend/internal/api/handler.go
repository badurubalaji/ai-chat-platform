package api

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mdp/ai-chat-platform/backend/internal/config"
	"github.com/mdp/ai-chat-platform/backend/internal/domain"
	"github.com/mdp/ai-chat-platform/backend/internal/models"
	"github.com/mdp/ai-chat-platform/backend/internal/providers"
	"github.com/mdp/ai-chat-platform/backend/internal/providers/claude"
	"github.com/mdp/ai-chat-platform/backend/internal/providers/gemini"
	"github.com/mdp/ai-chat-platform/backend/internal/providers/generic"
	"github.com/mdp/ai-chat-platform/backend/internal/providers/neuralgate"
	"github.com/mdp/ai-chat-platform/backend/internal/providers/ollama"
	"github.com/mdp/ai-chat-platform/backend/internal/providers/openai"
	"github.com/mdp/ai-chat-platform/backend/internal/store"
	"github.com/mdp/ai-chat-platform/backend/internal/utils"
)

// PendingConfirmation holds a tool call awaiting user approval.
type PendingConfirmation struct {
	ID             string
	ConversationID uuid.UUID
	ToolName       string
	Arguments      json.RawMessage
	CreatedAt      time.Time
	ResultChan     chan bool // buffered, cap 1
}

type Handler struct {
	store                store.Store
	config               *config.Config
	providers            map[string]providers.ChatProvider
	adapter              *domain.Adapter
	orchestrator         *domain.Orchestrator
	executor             *domain.Executor
	registry             *domain.ToolRegistry
	pendingConfirmations sync.Map // map[string]*PendingConfirmation
}

func NewHandler(s store.Store, cfg *config.Config, adapter *domain.Adapter) *Handler {
	h := &Handler{
		store:     s,
		config:    cfg,
		providers: make(map[string]providers.ChatProvider),
		adapter:   adapter,
	}

	// Create tool registry (merges adapter + DB tools)
	h.registry = domain.NewToolRegistry(adapter, s)

	// Always create orchestrator (for two-pass fallback with non-native-tool providers)
	h.orchestrator = domain.NewOrchestrator(adapter)
	h.executor = domain.NewExecutorWithRegistry(adapter, h.registry)

	// Register providers
	cp := claude.NewClaudeProvider()
	h.providers[cp.Name()] = cp

	op := openai.NewOpenAIProvider()
	h.providers[op.Name()] = op

	gp := gemini.NewGeminiProvider()
	h.providers[gp.Name()] = gp

	olp := ollama.NewOllamaProvider()
	h.providers[olp.Name()] = olp

	gnp := generic.NewGenericOpenAIProvider()
	h.providers[gnp.Name()] = gnp

	ngp := neuralgate.NewNeuralGateProvider()
	h.providers[ngp.Name()] = ngp

	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	path := r.URL.Path
	switch {
	case path == "/api/v1/ai/chat/confirm":
		h.handleConfirm(w, r)
	case strings.HasPrefix(path, "/api/v1/ai/chat"):
		h.handleChat(w, r)
	case strings.HasPrefix(path, "/api/v1/ai/conversations"):
		h.handleConversations(w, r)
	case strings.HasPrefix(path, "/api/v1/ai/config/test"):
		h.handleConfigTest(w, r)
	case strings.HasPrefix(path, "/api/v1/ai/config/models"):
		h.handleConfigModels(w, r)
	case strings.HasPrefix(path, "/api/v1/ai/config"):
		h.handleConfig(w, r)
	case strings.HasPrefix(path, "/api/v1/ai/usage"):
		h.handleUsage(w, r)
	case strings.HasPrefix(path, "/api/v1/ai/providers"):
		h.handleProviders(w, r)
	case strings.HasPrefix(path, "/api/v1/ai/registry/tools"):
		h.handleRegistry(w, r)
	default:
		http.NotFound(w, r)
	}
}

// chatRequest holds the parsed chat request from either JSON or multipart.
type chatRequest struct {
	ConversationID *uuid.UUID
	Message        string
	Context        map[string]interface{}
	Files          []models.FileAttachment
}

// parseMultipartChat parses a multipart/form-data chat request with file uploads.
func (h *Handler) parseMultipartChat(r *http.Request) (*chatRequest, error) {
	const maxMemory = 50 << 20 // 50MB total
	if err := r.ParseMultipartForm(maxMemory); err != nil {
		return nil, fmt.Errorf("failed to parse multipart form: %w", err)
	}

	req := &chatRequest{
		Message: r.FormValue("message"),
	}

	if convID := r.FormValue("conversation_id"); convID != "" {
		parsed, err := uuid.Parse(convID)
		if err != nil {
			return nil, fmt.Errorf("invalid conversation_id: %w", err)
		}
		req.ConversationID = &parsed
	}

	if ctxStr := r.FormValue("context"); ctxStr != "" {
		if err := json.Unmarshal([]byte(ctxStr), &req.Context); err != nil {
			log.Printf("[CHAT] Warning: failed to parse context JSON: %v", err)
		}
	}

	// Parse uploaded files
	if r.MultipartForm != nil && r.MultipartForm.File != nil {
		fileHeaders := r.MultipartForm.File["files"]
		if len(fileHeaders) > 5 {
			return nil, fmt.Errorf("too many files: maximum 5 files allowed")
		}
		for _, fh := range fileHeaders {
			if fh.Size > 20<<20 { // 20MB per file
				return nil, fmt.Errorf("file %q exceeds 20MB limit", fh.Filename)
			}
			f, err := fh.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open file %q: %w", fh.Filename, err)
			}
			data, err := io.ReadAll(f)
			f.Close()
			if err != nil {
				return nil, fmt.Errorf("failed to read file %q: %w", fh.Filename, err)
			}
			req.Files = append(req.Files, models.FileAttachment{
				Filename:    fh.Filename,
				ContentType: fh.Header.Get("Content-Type"),
				Base64Data:  base64.StdEncoding.EncodeToString(data),
				Size:        fh.Size,
			})
		}
	}

	return req, nil
}

// parseJSONChat parses a JSON chat request (supports inline base64 file attachments).
func (h *Handler) parseJSONChat(r *http.Request) (*chatRequest, error) {
	var body struct {
		ConversationID *uuid.UUID              `json:"conversation_id"`
		Message        string                  `json:"message"`
		Context        map[string]interface{}  `json:"context"`
		Files          []models.FileAttachment `json:"files"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return nil, err
	}
	// Validate files
	if len(body.Files) > 5 {
		return nil, fmt.Errorf("too many files: maximum 5 files allowed")
	}
	for _, f := range body.Files {
		if f.Size > 20<<20 {
			return nil, fmt.Errorf("file %q exceeds 20MB limit", f.Filename)
		}
	}
	return &chatRequest{
		ConversationID: body.ConversationID,
		Message:        body.Message,
		Context:        body.Context,
		Files:          body.Files,
	}, nil
}

func (h *Handler) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tenantID := "default-tenant"
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	// Parse request: multipart (with files) or JSON (text-only or base64 files)
	var req *chatRequest
	var parseErr error
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		req, parseErr = h.parseMultipartChat(r)
	} else {
		req, parseErr = h.parseJSONChat(r)
	}
	if parseErr != nil {
		log.Printf("[CHAT] Failed to decode request: %v", parseErr)
		http.Error(w, parseErr.Error(), http.StatusBadRequest)
		return
	}
	log.Printf("[CHAT] Received message: %q (conversation_id: %v, files: %d)", req.Message, req.ConversationID, len(req.Files))

	// Resolve provider: BYOK config > adapter default > error
	provider, apiKey, modelName, endpoint, providerName, err := h.resolveProvider(r.Context(), tenantID)
	if err != nil {
		log.Printf("[CHAT] Provider not configured: %v", err)
		http.Error(w, "Provider not configured: "+err.Error(), http.StatusNotFound)
		return
	}
	log.Printf("[CHAT] Provider: %s, Model: %s", providerName, modelName)

	// Conversation setup
	var convoID uuid.UUID
	var history []*models.Message

	if req.ConversationID != nil {
		convoID = *req.ConversationID
		history, _ = h.store.ListMessages(r.Context(), convoID)
		log.Printf("[CHAT] Loaded %d history messages for conversation %s", len(history), convoID)
	} else {
		convoID = uuid.New()
		title := req.Message
		if len(title) > 50 {
			title = title[:50]
		}
		newConvo := &models.Conversation{
			ID:       convoID,
			TenantID: tenantID,
			UserID:   userID,
			Title:    title,
		}
		if err := h.store.CreateConversation(r.Context(), newConvo); err != nil {
			log.Printf("[CHAT] Failed to create conversation: %v", err)
			http.Error(w, "Failed to create conversation", http.StatusInternalServerError)
			return
		}
		log.Printf("[CHAT] Created conversation: %s", convoID)
	}

	// Save user message
	userMsg := &models.Message{
		ID:             uuid.New(),
		ConversationID: convoID,
		Role:           models.RoleUser,
		Content:        req.Message,
		Files:          req.Files,
		CreatedAt:      time.Now(),
	}
	if err := h.store.CreateMessage(r.Context(), userMsg); err != nil {
		log.Printf("[CHAT] Failed to save user message: %v", err)
		http.Error(w, "Failed to save message", http.StatusInternalServerError)
		return
	}

	// Build message history for provider
	var providerMessages []models.Message
	for _, m := range history {
		providerMessages = append(providerMessages, *m)
	}
	providerMessages = append(providerMessages, *userMsg)

	// SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	systemPrompt := h.resolveSystemPrompt()

	// Create and run the agent
	agent := domain.NewAgent(domain.AgentConfig{
		Provider:     provider,
		Executor:     h.executor,
		Registry:     h.registry,
		Orchestrator: h.orchestrator,
		Adapter:      h.adapter,
	})

	// Confirmation callback — bridges agent to SSE confirmation flow
	confirmFn := func(confirmID, toolName, description string, params json.RawMessage) (bool, error) {
		pending := &PendingConfirmation{
			ID:             confirmID,
			ConversationID: convoID,
			ToolName:       toolName,
			Arguments:      params,
			CreatedAt:      time.Now(),
			ResultChan:     make(chan bool, 1),
		}
		h.pendingConfirmations.Store(confirmID, pending)

		confirmData, _ := json.Marshal(struct {
			ConfirmationID string          `json:"confirmation_id"`
			Tool           string          `json:"tool"`
			Description    string          `json:"description"`
			Params         json.RawMessage `json:"params"`
		}{confirmID, toolName, description, params})
		h.sendSSE(w, "tool_confirm", string(confirmData))

		select {
		case approved := <-pending.ResultChan:
			h.pendingConfirmations.Delete(confirmID)
			return approved, nil
		case <-time.After(5 * time.Minute):
			h.pendingConfirmations.Delete(confirmID)
			return false, fmt.Errorf("confirmation timed out")
		case <-r.Context().Done():
			h.pendingConfirmations.Delete(confirmID)
			return false, r.Context().Err()
		}
	}

	events, _ := agent.Run(r.Context(), domain.RunParams{
		APIKey:       apiKey,
		Model:        modelName,
		Endpoint:     endpoint,
		Messages:     providerMessages,
		SystemPrompt: systemPrompt,
		Files:        req.Files,
		TenantID:     tenantID,
		UserID:       userID,
		ConvoID:      convoID,
		ConfirmFn:    confirmFn,
	})

	// Stream agent events to client
	fullResponse := ""
	var totalInputTokens, totalOutputTokens int
	for event := range events {
		switch event.Type {
		case "_response":
			// Internal event: final response text for saving
			fullResponse = event.Data
		case "done":
			var doneData struct {
				Usage *models.Usage `json:"usage"`
			}
			json.Unmarshal([]byte(event.Data), &doneData)
			if doneData.Usage != nil {
				totalInputTokens = doneData.Usage.InputTokens
				totalOutputTokens = doneData.Usage.OutputTokens
			}
			h.sendSSE(w, event.Type, event.Data)
		default:
			h.sendSSE(w, event.Type, event.Data)
		}
	}

	// Save assistant message
	if fullResponse != "" {
		asstMsg := &models.Message{
			ID:             uuid.New(),
			ConversationID: convoID,
			Role:           models.RoleAssistant,
			Content:        fullResponse,
			CreatedAt:      time.Now(),
		}
		h.store.CreateMessage(r.Context(), asstMsg)
	}

	// Estimate tokens if not provided
	if totalInputTokens == 0 {
		totalInputTokens = len(strings.Fields(req.Message)) * 2
		totalOutputTokens = len(strings.Fields(fullResponse)) * 2
	}

	h.store.LogUsage(r.Context(), &models.UsageLog{
		ID:             uuid.New(),
		TenantID:       tenantID,
		UserID:         userID,
		ConversationID: convoID,
		Provider:       providerName,
		Model:          modelName,
		InputTokens:    totalInputTokens,
		OutputTokens:   totalOutputTokens,
	})
}

// resolveProvider resolves the AI provider: BYOK user config first, then adapter default.
func (h *Handler) resolveProvider(ctx context.Context, tenantID string) (providers.ChatProvider, string, string, string, string, error) {
	// Try user's BYOK config first
	providerConfig, err := h.store.GetProviderConfig(ctx, tenantID)
	if err == nil {
		// User has a BYOK config
		decryptedKey, err := utils.Decrypt(providerConfig.APIKeyEncrypted, h.config.AIEncryptionKey)
		if err != nil {
			return nil, "", "", "", "", fmt.Errorf("failed to decrypt API key: %w", err)
		}
		provider, ok := h.providers[providerConfig.Provider]
		if !ok {
			return nil, "", "", "", "", fmt.Errorf("provider %q not supported", providerConfig.Provider)
		}
		return provider, string(decryptedKey), providerConfig.Model, providerConfig.EndpointURL, providerConfig.Provider, nil
	}

	// No user config — try adapter default
	if err == sql.ErrNoRows && h.adapter != nil && h.adapter.HasDefaultProvider() {
		dp := h.adapter.DefaultProvider()
		provider, ok := h.providers[dp.Provider]
		if !ok {
			return nil, "", "", "", "", fmt.Errorf("adapter default provider %q not supported", dp.Provider)
		}
		// NeuralGate expects "client_id:client_secret" format
		apiKey := dp.ClientID + ":" + dp.ClientSecret
		return provider, apiKey, dp.Model, dp.EndpointURL, dp.Provider, nil
	}

	return nil, "", "", "", "", fmt.Errorf("no provider configured")
}

// handleConfirm handles POST /api/v1/ai/chat/confirm
func (h *Handler) handleConfirm(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ConfirmationID string `json:"confirmation_id"`
		Approved       bool   `json:"approved"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	val, ok := h.pendingConfirmations.Load(req.ConfirmationID)
	if !ok {
		http.Error(w, "Confirmation not found or expired", http.StatusNotFound)
		return
	}

	pending := val.(*PendingConfirmation)
	pending.ResultChan <- req.Approved

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// sendSSE writes a single SSE event to the response writer and flushes.
func (h *Handler) sendSSE(w http.ResponseWriter, event, data string) {
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

// sendDoneEvent sends the final SSE done event.
func (h *Handler) sendDoneEvent(w http.ResponseWriter, convoID uuid.UUID, inputTokens, outputTokens int) {
	doneData, _ := json.Marshal(struct {
		Content        string `json:"content"`
		Done           bool   `json:"done"`
		ConversationID string `json:"conversation_id"`
		Usage          *struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage,omitempty"`
	}{
		Content:        "",
		Done:           true,
		ConversationID: convoID.String(),
		Usage: &struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		}{
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
		},
	})
	h.sendSSE(w, "done", string(doneData))
}

func (h *Handler) resolveSystemPrompt() string {
	if h.adapter != nil {
		return h.adapter.SystemPrompt()
	}
	return "You are a helpful assistant."
}

func (h *Handler) handleConversations(w http.ResponseWriter, r *http.Request) {
	tenantID := "default-tenant"
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	// Extract conversation ID from path: /api/v1/ai/conversations/:id
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/ai/conversations")
	path = strings.TrimPrefix(path, "/")

	if path != "" {
		// We have a conversation ID
		convoID, err := uuid.Parse(path)
		if err != nil {
			http.Error(w, "Invalid conversation ID", http.StatusBadRequest)
			return
		}

		switch r.Method {
		case "GET":
			// Get conversation with messages
			convo, err := h.store.GetConversation(r.Context(), convoID)
			if err != nil {
				http.Error(w, "Conversation not found", http.StatusNotFound)
				return
			}
			messages, err := h.store.ListMessages(r.Context(), convoID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"conversation": convo,
				"messages":     messages,
			})

		case "DELETE":
			if err := h.store.DeleteConversation(r.Context(), convoID); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)

		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// No conversation ID — list or create
	switch r.Method {
	case "GET":
		conversations, err := h.store.ListConversations(r.Context(), tenantID, userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(conversations)

	case "POST":
		var req struct {
			Title string `json:"title"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		title := req.Title
		if title == "" {
			title = "New Chat"
		}

		convo := &models.Conversation{
			ID:       uuid.New(),
			TenantID: tenantID,
			UserID:   userID,
			Title:    title,
		}
		if err := h.store.CreateConversation(r.Context(), convo); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(convo)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleConfig(w http.ResponseWriter, r *http.Request) {
	tenantID := "default-tenant"

	switch r.Method {
	case "GET":
		config, err := h.store.GetProviderConfig(r.Context(), tenantID)
		if err != nil {
			// Return empty config if not found
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"enabled": false})
			return
		}
		// Return config with api_key_set indicator instead of actual key
		hasKey := len(config.APIKeyEncrypted) > 0
		config.APIKeyEncrypted = nil
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":           config.ID,
			"tenant_id":    config.TenantID,
			"provider":     config.Provider,
			"model":        config.Model,
			"endpoint_url": config.EndpointURL,
			"settings":     config.Settings,
			"enabled":      config.Enabled,
			"created_at":   config.CreatedAt,
			"api_key_set":  hasKey,
		})

	case "POST":
		var req struct {
			Provider    string          `json:"provider"`
			Model       string          `json:"model"`
			APIKey      string          `json:"apiKey"` // Plain text from frontend
			EndpointURL string          `json:"endpoint_url"`
			Settings    json.RawMessage `json:"settings"`
			Enabled     bool            `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var settingsMap map[string]interface{}
		if len(req.Settings) > 0 {
			if err := json.Unmarshal(req.Settings, &settingsMap); err != nil {
				http.Error(w, "Invalid settings format", http.StatusBadRequest)
				return
			}
		}

		// Encrypt API Key — only if user sent a new one
		var encryptedKey []byte
		if req.APIKey != "" {
			var err2 error
			encryptedKey, err2 = utils.Encrypt([]byte(req.APIKey), h.config.AIEncryptionKey)
			if err2 != nil {
				http.Error(w, "Failed to encrypt API key", http.StatusInternalServerError)
				return
			}
		} else {
			// Preserve existing key
			existingConfig, err := h.store.GetProviderConfig(r.Context(), tenantID)
			if err == nil && existingConfig != nil {
				encryptedKey = existingConfig.APIKeyEncrypted
			}
		}

		if len(encryptedKey) == 0 {
			http.Error(w, "API key is required", http.StatusBadRequest)
			return
		}

		// Save config
		cfg := &models.ProviderConfig{
			TenantID:        tenantID,
			Provider:        req.Provider,
			Model:           req.Model,
			APIKeyEncrypted: encryptedKey,
			EndpointURL:     req.EndpointURL,
			Settings:        settingsMap,
			Enabled:         req.Enabled,
		}

		if err := h.store.SaveProviderConfig(r.Context(), cfg); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		cfg.APIKeyEncrypted = nil // Don't send back encrypted key
		json.NewEncoder(w).Encode(cfg)

	case "DELETE":
		if err := h.store.DeleteProviderConfig(r.Context(), tenantID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleConfigTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Provider    string `json:"provider"`
		APIKey      string `json:"apiKey"`
		Model       string `json:"model"`
		EndpointURL string `json:"endpoint_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[TEST] ❌ Failed to decode request: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("[TEST] 🔍 Testing connection: provider=%s, model=%s, endpoint=%s, apiKey_len=%d", req.Provider, req.Model, req.EndpointURL, len(req.APIKey))

	provider, ok := h.providers[req.Provider]
	if !ok {
		log.Printf("[TEST] ❌ Unknown provider: %s", req.Provider)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "error",
			"message": "Unknown provider: " + req.Provider,
		})
		return
	}

	if err := provider.ValidateCredentials(r.Context(), req.APIKey, req.Model, req.EndpointURL); err != nil {
		log.Printf("[TEST] ❌ Validation failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	log.Printf("[TEST] ✅ Connection successful")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (h *Handler) handleConfigModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Provider    string `json:"provider"`
		APIKey      string `json:"apiKey"`
		EndpointURL string `json:"endpoint_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	provider, ok := h.providers[req.Provider]
	if !ok {
		http.Error(w, "Unknown provider: "+req.Provider, http.StatusBadRequest)
		return
	}

	modelList, err := provider.ListModels(r.Context(), req.APIKey, req.EndpointURL)
	if err != nil {
		// Fallback to static list on error
		modelList = h.getStaticModels(req.Provider)
	}

	json.NewEncoder(w).Encode(modelList)
}

func (h *Handler) getStaticModels(provider string) []string {
	switch provider {
	case "openai":
		return []string{"gpt-4o", "gpt-4o-mini", "o1", "o3-mini"}
	case "claude":
		return []string{"claude-sonnet-4-5-20250929", "claude-opus-4-6", "claude-haiku-4-5-20251001"}
	case "gemini":
		return []string{"gemini-2.0-flash", "gemini-2.0-pro"}
	case "ollama":
		return []string{"llama3", "mistral"}
	case "neuralgate":
		return []string{"auto"}
	default:
		return []string{"default-model"}
	}
}

func (h *Handler) handleUsage(w http.ResponseWriter, r *http.Request) {
	tenantID := "default-tenant"

	days := 30
	if d := r.URL.Query().Get("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 {
			days = parsed
		}
	}

	stats, err := h.store.GetUsageStats(r.Context(), tenantID, days)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (h *Handler) handleProviders(w http.ResponseWriter, r *http.Request) {
	providers := []map[string]interface{}{
		{"id": "openai", "name": "OpenAI", "icon": "smart_toy", "requires_endpoint": false},
		{"id": "claude", "name": "Anthropic Claude", "icon": "smart_toy", "requires_endpoint": false},
		{"id": "gemini", "name": "Google Gemini", "icon": "smart_toy", "requires_endpoint": false},
		{"id": "ollama", "name": "Ollama (Local)", "icon": "dns", "requires_endpoint": true, "default_endpoint": "http://localhost:11434"},
		{"id": "generic", "name": "Generic OpenAI", "icon": "settings_ethernet", "requires_endpoint": true},
		{"id": "neuralgate", "name": "NeuralGate AI Gateway", "icon": "hub", "requires_endpoint": true},
	}
	json.NewEncoder(w).Encode(providers)
}

// -- Tool Registry Handlers --

func (h *Handler) handleRegistry(w http.ResponseWriter, r *http.Request) {
	// Extract tool ID from path: /api/v1/ai/registry/tools/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/ai/registry/tools")
	path = strings.TrimPrefix(path, "/")

	switch r.Method {
	case "GET":
		h.handleListTools(w, r)
	case "POST":
		h.handleRegisterTool(w, r)
	case "PUT":
		if path == "" {
			http.Error(w, "tool ID required", http.StatusBadRequest)
			return
		}
		h.handleUpdateTool(w, r, path)
	case "DELETE":
		if path == "" {
			http.Error(w, "tool ID required", http.StatusBadRequest)
			return
		}
		h.handleDeleteTool(w, r, path)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleRegisterTool(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}

	var req models.ToolRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.AppName == "" || req.ToolName == "" || req.Description == "" {
		http.Error(w, "app_name, tool_name, and description are required", http.StatusBadRequest)
		return
	}
	if len(req.Execution) == 0 {
		http.Error(w, "execution config is required", http.StatusBadRequest)
		return
	}

	tool := &models.RegisteredTool{
		ID:                   uuid.New(),
		TenantID:             tenantID,
		AppName:              req.AppName,
		ToolName:             req.ToolName,
		Description:          req.Description,
		Parameters:           req.Parameters,
		ExecutionConfig:      req.Execution,
		RequiresConfirmation: req.RequiresConfirmation,
		Enabled:              true,
	}

	// Default empty parameters to {}
	if len(tool.Parameters) == 0 {
		tool.Parameters = json.RawMessage(`{}`)
	}

	if err := h.store.RegisterTool(r.Context(), tool); err != nil {
		http.Error(w, "failed to register tool: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(tool)
}

func (h *Handler) handleListTools(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}
	appName := r.URL.Query().Get("app_name")

	tools, err := h.store.ListTools(r.Context(), tenantID, appName)
	if err != nil {
		http.Error(w, "failed to list tools: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if tools == nil {
		tools = []*models.RegisteredTool{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tools)
}

func (h *Handler) handleUpdateTool(w http.ResponseWriter, r *http.Request, toolID string) {
	id, err := uuid.Parse(toolID)
	if err != nil {
		http.Error(w, "invalid tool ID", http.StatusBadRequest)
		return
	}

	existing, err := h.store.GetTool(r.Context(), id)
	if err != nil {
		http.Error(w, "tool not found", http.StatusNotFound)
		return
	}

	var req models.ToolRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Update fields that were provided
	if req.Description != "" {
		existing.Description = req.Description
	}
	if len(req.Parameters) > 0 {
		existing.Parameters = req.Parameters
	}
	if len(req.Execution) > 0 {
		existing.ExecutionConfig = req.Execution
	}
	existing.RequiresConfirmation = req.RequiresConfirmation

	if err := h.store.UpdateTool(r.Context(), existing); err != nil {
		http.Error(w, "failed to update tool: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(existing)
}

func (h *Handler) handleDeleteTool(w http.ResponseWriter, r *http.Request, toolID string) {
	id, err := uuid.Parse(toolID)
	if err != nil {
		http.Error(w, "invalid tool ID", http.StatusBadRequest)
		return
	}

	if err := h.store.DeleteTool(r.Context(), id); err != nil {
		http.Error(w, "failed to delete tool: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
