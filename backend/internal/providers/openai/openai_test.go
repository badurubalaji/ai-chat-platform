package openai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestToolUseStreaming(t *testing.T) {
	mockBody := `data: {"choices": [{"delta": {"tool_calls": [{"index": 0, "id": "call_123", "type": "function", "function": {"name": "get_weather", "arguments": ""}}]}}]}

data: {"choices": [{"delta": {"tool_calls": [{"index": 0, "function": {"arguments": "{\"location\": \"Paris\"}"}}]}}]}

data: [DONE]
`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Connection", "close")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockBody))
	}))
	defer server.Close()

	provider := NewOpenAIProvider()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ch, err := provider.SendMessageStream(ctx, "fake-key", "gpt-4", server.URL, nil, nil, "")
	if err != nil {
		t.Fatalf("SendMessageStream failed: %v", err)
	}

	var collectedID, collectedName, collectedArgs string

	for chunk := range ch {
		if chunk.ToolCall != nil {
			if chunk.ToolCall.ID != "" {
				collectedID = chunk.ToolCall.ID
			}
			if chunk.ToolCall.Name != "" {
				collectedName = chunk.ToolCall.Name
			}
			collectedArgs += chunk.ToolCall.Arguments
		}
	}

	if collectedID != "call_123" {
		t.Errorf("Expected tool ID 'call_123', got %q", collectedID)
	}
	if collectedName != "get_weather" {
		t.Errorf("Expected tool name 'get_weather', got %q", collectedName)
	}
	expectedArgs := `{"location": "Paris"}`
	if collectedArgs != expectedArgs {
		t.Errorf("Expected args %q, got %q", expectedArgs, collectedArgs)
	}
}
