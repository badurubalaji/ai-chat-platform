package claude

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestParseError(t *testing.T) {
	provider := &ClaudeProvider{}

	tests := []struct {
		name          string
		statusCode    int
		body          string
		expectedError string
	}{
		{
			name:          "Anthropic Low Balance Error",
			statusCode:    400,
			body:          `{"type":"error","error":{"type":"invalid_request_error","message":"Your credit balance is too low to access the Anthropic API. Please go to Plans & Billing to upgrade or purchase credits."},"request_id":"req_011CY6QfxHQ4H5MPkJ8d2KCF"}`,
			expectedError: "Claude API error (status 400): Your credit balance is too low to access the Anthropic API. Please go to Plans & Billing to upgrade or purchase credits.",
		},
		// ... potentially other cases
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				StatusCode: tt.statusCode,
				Body:       io.NopCloser(bytes.NewBufferString(tt.body)),
			}

			err := provider.parseError(resp)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}

			if err.Error() != tt.expectedError {
				t.Errorf("expected error %q, got %q", tt.expectedError, err.Error())
			}
		})
	}
}

func TestToolUseStreaming(t *testing.T) {
	mockBody := `event: content_block_start
data: {"type": "content_block_start", "index": 0, "content_block": {"type": "tool_use", "id": "toolu_01", "name": "get_weather", "input": {}}}

event: content_block_delta
data: {"type": "content_block_delta", "index": 0, "delta": {"type": "input_json_delta", "partial_json": "{\"location\": \"London\","}}

event: content_block_delta
data: {"type": "content_block_delta", "index": 0, "delta": {"type": "input_json_delta", "partial_json": "\"unit\": \"celsius\"}"}}

event: content_block_stop
data: {"type": "content_block_stop", "index": 0}

event: message_stop
data: {"type": "message_stop"}
`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Connection", "close") // Ensure connection closes
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockBody))
	}))
	defer server.Close()

	provider := NewClaudeProvider()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ch, err := provider.SendMessageStream(ctx, "fake-key", "claude-3", server.URL, nil, nil, "", nil)
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

	if collectedID != "toolu_01" {
		t.Errorf("Expected tool ID 'toolu_01', got %q", collectedID)
	}
	if collectedName != "get_weather" {
		t.Errorf("Expected tool name 'get_weather', got %q", collectedName)
	}
	// Note: The mock data sends partial JSON strings.
	// The first chunk has "{\"location\": \"London\","
	// The second chunk has "\"unit\": \"celsius\"}"
	// Combined: "{\"location\": \"London\",\"unit\": \"celsius\"}"
	expectedArgs := `{"location": "London","unit": "celsius"}`
	if collectedArgs != expectedArgs {
		t.Errorf("Expected args %q, got %q", expectedArgs, collectedArgs)
	}
}
