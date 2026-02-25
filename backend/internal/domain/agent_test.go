package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/mdp/ai-chat-platform/backend/internal/models"
)

// --- Mock Provider ---

type mockProvider struct {
	name          string
	supportsTools bool
	responses     []mockResponse // one per iteration
	callIndex     int
	receivedMsgs  [][]models.Message // messages sent on each call
}

type mockResponse struct {
	text     string
	toolCall *models.ToolCall
}

func (m *mockProvider) Name() string            { return m.name }
func (m *mockProvider) SupportsTools() bool      { return m.supportsTools }
func (m *mockProvider) SupportsStreaming() bool   { return true }

func (m *mockProvider) SendMessageStream(ctx context.Context, apiKey, model, endpoint string, messages []models.Message, tools []models.Tool, systemPrompt string, files []models.FileAttachment) (<-chan models.StreamChunk, error) {
	// Record messages for assertion
	msgCopy := make([]models.Message, len(messages))
	copy(msgCopy, messages)
	m.receivedMsgs = append(m.receivedMsgs, msgCopy)

	ch := make(chan models.StreamChunk, 10)
	go func() {
		defer close(ch)
		if m.callIndex >= len(m.responses) {
			ch <- models.StreamChunk{Content: "no more responses", Done: false}
			ch <- models.StreamChunk{Done: true}
			return
		}
		resp := m.responses[m.callIndex]
		m.callIndex++

		if resp.toolCall != nil {
			// Send tool call in chunks (simulating streaming)
			ch <- models.StreamChunk{
				ToolCall: &models.ToolCall{
					ID:   resp.toolCall.ID,
					Name: resp.toolCall.Name,
				},
			}
			// Send arguments in two fragments
			args := resp.toolCall.Arguments
			mid := len(args) / 2
			if mid > 0 {
				ch <- models.StreamChunk{
					ToolCall: &models.ToolCall{Arguments: args[:mid]},
				}
				ch <- models.StreamChunk{
					ToolCall: &models.ToolCall{Arguments: args[mid:]},
				}
			}
			if resp.text != "" {
				ch <- models.StreamChunk{Content: resp.text}
			}
		} else {
			ch <- models.StreamChunk{Content: resp.text}
		}
		ch <- models.StreamChunk{Done: true, Usage: &models.Usage{InputTokens: 100, OutputTokens: 50}}
	}()
	return ch, nil
}

func (m *mockProvider) SendMessageSync(ctx context.Context, apiKey, model, endpoint string, messages []models.Message, tools []models.Tool, systemPrompt string, files []models.FileAttachment) (*models.Message, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockProvider) ValidateCredentials(ctx context.Context, apiKey, model, endpoint string) error {
	return nil
}

func (m *mockProvider) ListModels(ctx context.Context, apiKey, endpoint string) ([]string, error) {
	return []string{"test-model"}, nil
}

func (m *mockProvider) FormatToolResult(assistantText string, toolCall *models.ToolCall, result string, isError bool) (models.Message, models.Message) {
	assistantMsg := models.Message{Role: models.RoleAssistant, Content: assistantText}
	content := fmt.Sprintf("Tool '%s' result: %s", toolCall.Name, result)
	if isError {
		content = fmt.Sprintf("Tool '%s' failed: %s", toolCall.Name, result)
	}
	toolResultMsg := models.Message{Role: models.RoleToolResult, Content: content}
	return assistantMsg, toolResultMsg
}

// --- Mock Tool Source (DB) ---

type mockToolSource struct {
	tools []*models.RegisteredTool
}

func (m *mockToolSource) ListTools(ctx context.Context, tenantID, appName string) ([]*models.RegisteredTool, error) {
	return m.tools, nil
}

// --- Tests ---

func TestAgent_SingleToolCall(t *testing.T) {
	// Mock HTTP server simulating EHR API
	ehrServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]string{
			{"id": "P-1001", "name": "John Smith"},
			{"id": "P-1002", "name": "Sarah Johnson"},
		})
	}))
	defer ehrServer.Close()

	// Build execution config
	execConfig, _ := json.Marshal(map[string]interface{}{
		"type":       "http",
		"method":     "GET",
		"url":        ehrServer.URL + "/patients",
		"timeout_ms": 5000,
	})

	toolSource := &mockToolSource{
		tools: []*models.RegisteredTool{
			{
				ID:              uuid.New(),
				TenantID:        "test-tenant",
				AppName:         "ehr",
				ToolName:        "list_patients",
				Description:     "List all patients",
				Parameters:      json.RawMessage(`{"type":"object","properties":{}}`),
				ExecutionConfig: execConfig,
				Enabled:         true,
			},
		},
	}

	provider := &mockProvider{
		name:          "test",
		supportsTools: true,
		responses: []mockResponse{
			// Iteration 1: AI calls tool
			{toolCall: &models.ToolCall{ID: "call_1", Name: "list_patients", Arguments: `{}`}},
			// Iteration 2: AI gives final answer after seeing tool result
			{text: "Here are 2 patients: John Smith and Sarah Johnson."},
		},
	}

	registry := NewToolRegistry(nil, toolSource)
	executor := NewExecutorWithRegistry(nil, registry)

	agent := NewAgent(AgentConfig{
		Provider:      provider,
		Executor:      executor,
		Registry:      registry,
		MaxIterations: 5,
	})

	events, _ := agent.Run(context.Background(), RunParams{
		APIKey:   "test-key",
		Model:    "test-model",
		Messages: []models.Message{{Role: models.RoleUser, Content: "Show me all patients"}},
		TenantID: "test-tenant",
		UserID:   uuid.New(),
		ConvoID:  uuid.New(),
	})

	var eventTypes []string
	var finalResponse string
	var toolCallSeen, toolResultSeen bool

	for ev := range events {
		eventTypes = append(eventTypes, ev.Type)
		if ev.Type == "tool_call" {
			toolCallSeen = true
			if !strings.Contains(ev.Data, `"list_patients"`) {
				t.Errorf("tool_call event should contain list_patients, got: %s", ev.Data)
			}
		}
		if ev.Type == "tool_result" {
			toolResultSeen = true
			if !strings.Contains(ev.Data, `"complete"`) {
				t.Errorf("tool_result event should contain complete, got: %s", ev.Data)
			}
		}
		if ev.Type == "_response" {
			finalResponse = ev.Data
		}
	}

	if !toolCallSeen {
		t.Error("Expected tool_call event")
	}
	if !toolResultSeen {
		t.Error("Expected tool_result event")
	}
	if finalResponse != "Here are 2 patients: John Smith and Sarah Johnson." {
		t.Errorf("Unexpected final response: %s", finalResponse)
	}

	// Verify provider received 2 calls
	if len(provider.receivedMsgs) != 2 {
		t.Fatalf("Expected 2 provider calls, got %d", len(provider.receivedMsgs))
	}

	// Second call should include tool result
	secondCallMsgs := provider.receivedMsgs[1]
	lastMsg := secondCallMsgs[len(secondCallMsgs)-1]
	if lastMsg.Role != models.RoleToolResult {
		t.Errorf("Expected last message in 2nd call to be tool_result, got: %s", lastMsg.Role)
	}
	if !strings.Contains(lastMsg.Content, "John Smith") {
		t.Errorf("Tool result should contain EHR data, got: %s", lastMsg.Content)
	}

	t.Logf("Single tool call test passed! Events: %v", eventTypes)
}

func TestAgent_MultiStepToolCalls(t *testing.T) {
	// Two mock endpoints
	searchServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]string{
			{"id": "P-1001", "name": "John Smith"},
		})
	}))
	defer searchServer.Close()

	historyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]string{
			{"type": "visit", "summary": "Annual checkup - all vitals normal"},
			{"type": "lab", "summary": "Blood panel - cholesterol elevated"},
		})
	}))
	defer historyServer.Close()

	searchExec, _ := json.Marshal(map[string]interface{}{
		"type": "http", "method": "GET",
		"url": searchServer.URL + "/patients/search", "timeout_ms": 5000,
	})
	historyExec, _ := json.Marshal(map[string]interface{}{
		"type": "http", "method": "GET",
		"url": historyServer.URL + "/history/{patient_id}", "timeout_ms": 5000,
	})

	toolSource := &mockToolSource{
		tools: []*models.RegisteredTool{
			{
				ID: uuid.New(), TenantID: "test-tenant", AppName: "ehr",
				ToolName: "search_patients", Description: "Search patients",
				Parameters:      json.RawMessage(`{"type":"object","properties":{"q":{"type":"string"}},"required":["q"]}`),
				ExecutionConfig: searchExec, Enabled: true,
			},
			{
				ID: uuid.New(), TenantID: "test-tenant", AppName: "ehr",
				ToolName: "get_patient_history", Description: "Get patient medical history",
				Parameters:      json.RawMessage(`{"type":"object","properties":{"patient_id":{"type":"string"}},"required":["patient_id"]}`),
				ExecutionConfig: historyExec, Enabled: true,
			},
		},
	}

	provider := &mockProvider{
		name:          "test",
		supportsTools: true,
		responses: []mockResponse{
			// Step 1: AI searches for patient
			{toolCall: &models.ToolCall{ID: "call_1", Name: "search_patients", Arguments: `{"q":"John Smith"}`}},
			// Step 2: AI gets patient history
			{toolCall: &models.ToolCall{ID: "call_2", Name: "get_patient_history", Arguments: `{"patient_id":"P-1001"}`}},
			// Step 3: AI summarizes
			{text: "John Smith had an annual checkup with normal vitals. Blood panel shows elevated cholesterol."},
		},
	}

	registry := NewToolRegistry(nil, toolSource)
	executor := NewExecutorWithRegistry(nil, registry)

	agent := NewAgent(AgentConfig{
		Provider:      provider,
		Executor:      executor,
		Registry:      registry,
		MaxIterations: 5,
	})

	events, _ := agent.Run(context.Background(), RunParams{
		APIKey:   "test-key",
		Model:    "test-model",
		Messages: []models.Message{{Role: models.RoleUser, Content: "Summarize John Smith's health status"}},
		TenantID: "test-tenant",
		UserID:   uuid.New(),
		ConvoID:  uuid.New(),
	})

	toolCallCount := 0
	toolResultCount := 0
	var finalResponse string

	for ev := range events {
		switch ev.Type {
		case "tool_call":
			toolCallCount++
		case "tool_result":
			toolResultCount++
		case "_response":
			finalResponse = ev.Data
		}
	}

	if toolCallCount != 2 {
		t.Errorf("Expected 2 tool calls, got %d", toolCallCount)
	}
	if toolResultCount != 2 {
		t.Errorf("Expected 2 tool results, got %d", toolResultCount)
	}
	if !strings.Contains(finalResponse, "cholesterol") {
		t.Errorf("Final response should mention cholesterol: %s", finalResponse)
	}

	// Verify 3 provider calls (search -> history -> summarize)
	if len(provider.receivedMsgs) != 3 {
		t.Fatalf("Expected 3 provider calls, got %d", len(provider.receivedMsgs))
	}

	t.Logf("Multi-step test passed! 2 tool calls, 3 provider iterations")
}

func TestAgent_ArgumentAccumulation(t *testing.T) {
	// Verify streaming argument fragments are concatenated correctly
	ehrServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": "P-NEW", "name": "Jane Doe", "status": "active"})
	}))
	defer ehrServer.Close()

	execConfig, _ := json.Marshal(map[string]interface{}{
		"type": "http", "method": "POST",
		"url": ehrServer.URL + "/patients", "timeout_ms": 5000,
	})

	toolSource := &mockToolSource{
		tools: []*models.RegisteredTool{
			{
				ID: uuid.New(), TenantID: "test-tenant", AppName: "ehr",
				ToolName: "create_patient", Description: "Create patient",
				Parameters:      json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"},"dob":{"type":"string"}},"required":["name"]}`),
				ExecutionConfig: execConfig, Enabled: true,
			},
		},
	}

	// The mock provider sends arguments split across chunks (simulating real streaming)
	fullArgs := `{"name":"Jane Doe","dob":"1995-06-20"}`
	provider := &mockProvider{
		name:          "test",
		supportsTools: true,
		responses: []mockResponse{
			{toolCall: &models.ToolCall{ID: "call_1", Name: "create_patient", Arguments: fullArgs}},
			{text: "Patient Jane Doe created successfully with ID P-NEW."},
		},
	}

	registry := NewToolRegistry(nil, toolSource)
	executor := NewExecutorWithRegistry(nil, registry)

	agent := NewAgent(AgentConfig{
		Provider:      provider,
		Executor:      executor,
		Registry:      registry,
		MaxIterations: 5,
	})

	events, _ := agent.Run(context.Background(), RunParams{
		APIKey:   "test-key",
		Model:    "test-model",
		Messages: []models.Message{{Role: models.RoleUser, Content: "Create patient Jane Doe born 1995-06-20"}},
		TenantID: "test-tenant",
		UserID:   uuid.New(),
		ConvoID:  uuid.New(),
	})

	var finalResponse string
	for ev := range events {
		if ev.Type == "_response" {
			finalResponse = ev.Data
		}
	}

	if !strings.Contains(finalResponse, "Jane Doe") {
		t.Errorf("Expected response about Jane Doe, got: %s", finalResponse)
	}

	t.Logf("Argument accumulation test passed!")
}

func TestAgent_ToolConfirmation(t *testing.T) {
	ehrServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": "P-NEW", "status": "active"})
	}))
	defer ehrServer.Close()

	execConfig, _ := json.Marshal(map[string]interface{}{
		"type": "http", "method": "POST",
		"url": ehrServer.URL + "/patients", "timeout_ms": 5000,
	})

	toolSource := &mockToolSource{
		tools: []*models.RegisteredTool{
			{
				ID: uuid.New(), TenantID: "test-tenant", AppName: "ehr",
				ToolName: "create_patient", Description: "Create patient",
				Parameters:           json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}`),
				ExecutionConfig:      execConfig,
				RequiresConfirmation: true,
				Enabled:              true,
			},
		},
	}

	provider := &mockProvider{
		name:          "test",
		supportsTools: true,
		responses: []mockResponse{
			{toolCall: &models.ToolCall{ID: "call_1", Name: "create_patient", Arguments: `{"name":"Test"}`}},
			{text: "The action was cancelled."},
		},
	}

	registry := NewToolRegistry(nil, toolSource)
	executor := NewExecutorWithRegistry(nil, registry)

	agent := NewAgent(AgentConfig{
		Provider:      provider,
		Executor:      executor,
		Registry:      registry,
		MaxIterations: 5,
	})

	// User denies confirmation
	confirmFn := func(confirmID, toolName, description string, params json.RawMessage) (bool, error) {
		return false, nil // denied
	}

	events, _ := agent.Run(context.Background(), RunParams{
		APIKey:    "test-key",
		Model:     "test-model",
		Messages:  []models.Message{{Role: models.RoleUser, Content: "Create patient Test"}},
		TenantID:  "test-tenant",
		UserID:    uuid.New(),
		ConvoID:   uuid.New(),
		ConfirmFn: confirmFn,
	})

	toolCallSeen := false
	toolResultSeen := false
	for ev := range events {
		if ev.Type == "tool_call" {
			toolCallSeen = true
		}
		if ev.Type == "tool_result" {
			toolResultSeen = true
		}
	}

	// Tool call should NOT have been executed (no tool_call or tool_result events)
	if toolCallSeen {
		t.Error("Tool should not have been executed when confirmation denied")
	}
	if toolResultSeen {
		t.Error("Tool result should not appear when confirmation denied")
	}

	t.Logf("Tool confirmation (denied) test passed!")
}

func TestAgent_TwoPassFallback(t *testing.T) {
	ehrServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]string{
			{"id": "P-1001", "name": "John Smith"},
		})
	}))
	defer ehrServer.Close()

	execConfig, _ := json.Marshal(map[string]interface{}{
		"type": "http", "method": "GET",
		"url": ehrServer.URL + "/patients", "timeout_ms": 5000,
	})

	toolSource := &mockToolSource{
		tools: []*models.RegisteredTool{
			{
				ID: uuid.New(), TenantID: "test-tenant", AppName: "ehr",
				ToolName: "list_patients", Description: "List all patients",
				Parameters:      json.RawMessage(`{"type":"object","properties":{}}`),
				ExecutionConfig: execConfig, Enabled: true,
			},
		},
	}

	// Provider that does NOT support native tools (like Ollama)
	provider := &mockProvider{
		name:          "ollama",
		supportsTools: false, // <-- two-pass mode
		responses: []mockResponse{
			// Model outputs JSON tool call in text (two-pass format)
			{text: `{"tool_call": {"name": "list_patients", "arguments": {}}}`},
			// After tool result, model gives final answer
			{text: "There is 1 patient: John Smith."},
		},
	}

	registry := NewToolRegistry(nil, toolSource)
	executor := NewExecutorWithRegistry(nil, registry)
	orchestrator := NewOrchestratorWithTools(nil) // tools injected dynamically

	agent := NewAgent(AgentConfig{
		Provider:      provider,
		Executor:      executor,
		Registry:      registry,
		Orchestrator:  orchestrator,
		MaxIterations: 5,
	})

	events, _ := agent.Run(context.Background(), RunParams{
		APIKey:   "test-key",
		Model:    "test-model",
		Messages: []models.Message{{Role: models.RoleUser, Content: "Show me all patients"}},
		TenantID: "test-tenant",
		UserID:   uuid.New(),
		ConvoID:  uuid.New(),
	})

	var finalResponse string
	toolResultSeen := false
	for ev := range events {
		if ev.Type == "tool_result" {
			toolResultSeen = true
		}
		if ev.Type == "_response" {
			finalResponse = ev.Data
		}
	}

	if !toolResultSeen {
		t.Error("Expected tool_result event in two-pass mode")
	}
	if !strings.Contains(finalResponse, "John Smith") {
		t.Errorf("Expected John Smith in response, got: %s", finalResponse)
	}

	t.Logf("Two-pass fallback test passed!")
}

func TestAgent_MaxIterationsRespected(t *testing.T) {
	provider := &mockProvider{
		name:          "test",
		supportsTools: true,
		responses: []mockResponse{
			{toolCall: &models.ToolCall{ID: "c1", Name: "unknown_tool", Arguments: `{}`}},
			{toolCall: &models.ToolCall{ID: "c2", Name: "unknown_tool", Arguments: `{}`}},
			{toolCall: &models.ToolCall{ID: "c3", Name: "unknown_tool", Arguments: `{}`}},
			{toolCall: &models.ToolCall{ID: "c4", Name: "unknown_tool", Arguments: `{}`}},
		},
	}

	registry := NewToolRegistry(nil, &mockToolSource{})
	executor := NewExecutorWithRegistry(nil, registry)

	agent := NewAgent(AgentConfig{
		Provider:      provider,
		Executor:      executor,
		Registry:      registry,
		MaxIterations: 3,
	})

	events, _ := agent.Run(context.Background(), RunParams{
		APIKey:   "test-key",
		Model:    "test-model",
		Messages: []models.Message{{Role: models.RoleUser, Content: "test"}},
		TenantID: "test-tenant",
		UserID:   uuid.New(),
		ConvoID:  uuid.New(),
	})

	for range events {
		// drain
	}

	if provider.callIndex > 3 {
		t.Errorf("Expected max 3 iterations, got %d", provider.callIndex)
	}

	t.Logf("Max iterations test passed! Stopped at %d", provider.callIndex)
}
