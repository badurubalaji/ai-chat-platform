package neuralgate

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mdp/ai-chat-platform/backend/internal/models"
)

func TestParseCredentials_Valid(t *testing.T) {
	clientID, clientSecret, err := parseCredentials("myid:mysecret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clientID != "myid" {
		t.Errorf("expected clientID 'myid', got %q", clientID)
	}
	if clientSecret != "mysecret" {
		t.Errorf("expected clientSecret 'mysecret', got %q", clientSecret)
	}
}

func TestParseCredentials_SecretWithColons(t *testing.T) {
	clientID, clientSecret, err := parseCredentials("myid:secret:with:colons")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clientID != "myid" {
		t.Errorf("expected clientID 'myid', got %q", clientID)
	}
	if clientSecret != "secret:with:colons" {
		t.Errorf("expected clientSecret 'secret:with:colons', got %q", clientSecret)
	}
}

func TestParseCredentials_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"no colon", "nocolon"},
		{"missing secret", "id:"},
		{"missing id", ":secret"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := parseCredentials(tt.input)
			if err == nil {
				t.Errorf("expected error for input %q, got nil", tt.input)
			}
		})
	}
}

func mockOAuthServer(t *testing.T, requestCount *int32) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requestCount != nil {
			atomic.AddInt32(requestCount, 1)
		}

		if r.URL.Path == "/api/v1/oauth/token" {
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)

			if body["client_id"] == "valid_id" && body["client_secret"] == "valid_secret" {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"access_token": "test_token_abc123",
					"token_type":   "Bearer",
					"expires_in":   900,
				})
				return
			}

			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]interface{}{
					"code":    401,
					"message": "Invalid client credentials",
				},
			})
			return
		}

		http.NotFound(w, r)
	}))
}

func TestOAuthTokenExchange(t *testing.T) {
	server := mockOAuthServer(t, nil)
	defer server.Close()

	p := NewNeuralGateProvider()
	token, err := p.getAccessToken(context.Background(), "valid_id:valid_secret", server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "test_token_abc123" {
		t.Errorf("expected token 'test_token_abc123', got %q", token)
	}
}

func TestOAuthTokenCaching(t *testing.T) {
	var requestCount int32
	server := mockOAuthServer(t, &requestCount)
	defer server.Close()

	p := NewNeuralGateProvider()

	// First call — should make HTTP request
	_, err := p.getAccessToken(context.Background(), "valid_id:valid_secret", server.URL)
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}

	// Second call — should use cache
	_, err = p.getAccessToken(context.Background(), "valid_id:valid_secret", server.URL)
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}

	count := atomic.LoadInt32(&requestCount)
	if count != 1 {
		t.Errorf("expected 1 HTTP request (cached), got %d", count)
	}
}

func TestOAuthTokenExpiry(t *testing.T) {
	var requestCount int32
	server := mockOAuthServer(t, &requestCount)
	defer server.Close()

	p := NewNeuralGateProvider()

	// Manually cache an expired token
	p.tokenCache.Store("valid_id:valid_secret", &cachedToken{
		accessToken: "old_token",
		expiresAt:   time.Now().Add(-1 * time.Second), // already expired
	})

	// Should re-fetch because token is expired
	token, err := p.getAccessToken(context.Background(), "valid_id:valid_secret", server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "test_token_abc123" {
		t.Errorf("expected fresh token, got %q", token)
	}

	count := atomic.LoadInt32(&requestCount)
	if count != 1 {
		t.Errorf("expected 1 HTTP request for refresh, got %d", count)
	}
}

func TestValidateCredentials_Success(t *testing.T) {
	server := mockOAuthServer(t, nil)
	defer server.Close()

	p := NewNeuralGateProvider()
	err := p.ValidateCredentials(context.Background(), "valid_id:valid_secret", "auto", server.URL)
	if err != nil {
		t.Errorf("expected success, got error: %v", err)
	}
}

func TestValidateCredentials_Failure(t *testing.T) {
	server := mockOAuthServer(t, nil)
	defer server.Close()

	p := NewNeuralGateProvider()
	err := p.ValidateCredentials(context.Background(), "bad_id:bad_secret", "auto", server.URL)
	if err == nil {
		t.Error("expected error for invalid credentials, got nil")
	}
}

func TestValidateCredentials_EmptyEndpoint(t *testing.T) {
	p := NewNeuralGateProvider()
	err := p.ValidateCredentials(context.Background(), "id:secret", "auto", "")
	if err == nil {
		t.Error("expected error for empty endpoint, got nil")
	}
}

func TestStreamingParsing(t *testing.T) {
	// Mock server that returns NeuralGate SSE format
	sseBody := `event: message
data: {"model":"test-model","message":{"role":"assistant","content":"Hello"},"done":false}

event: message
data: {"model":"test-model","message":{"role":"assistant","content":" world"},"done":false}

event: conversation
data: {"conversation_id":"abc-123","message_id":"msg-789"}

event: message
data: {"model":"test-model","message":{"role":"assistant","content":""},"done":true,"prompt_eval_count":42,"eval_count":10}

`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/oauth/token" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token": "tok",
				"expires_in":   900,
			})
			return
		}
		if r.URL.Path == "/api/v1/ai/chat" {
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(sseBody))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	p := NewNeuralGateProvider()
	ch, err := p.SendMessageStream(
		context.Background(),
		"valid_id:valid_secret",
		"test-model",
		server.URL,
		[]models.Message{{Role: models.RoleUser, Content: "Hi"}},
		nil,
		"",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var chunks []models.StreamChunk
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}

	// Expect: "Hello" chunk, " world" chunk, done chunk (conversation event is ignored)
	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunks, got %d: %+v", len(chunks), chunks)
	}

	if chunks[0].Content != "Hello" || chunks[0].Done {
		t.Errorf("chunk 0: expected Content='Hello' Done=false, got %+v", chunks[0])
	}
	if chunks[1].Content != " world" || chunks[1].Done {
		t.Errorf("chunk 1: expected Content=' world' Done=false, got %+v", chunks[1])
	}
	if !chunks[2].Done {
		t.Errorf("chunk 2: expected Done=true, got %+v", chunks[2])
	}
	if chunks[2].Usage == nil {
		t.Fatal("chunk 2: expected Usage to be set")
	}
	if chunks[2].Usage.InputTokens != 42 {
		t.Errorf("expected InputTokens=42, got %d", chunks[2].Usage.InputTokens)
	}
	if chunks[2].Usage.OutputTokens != 10 {
		t.Errorf("expected OutputTokens=10, got %d", chunks[2].Usage.OutputTokens)
	}
}

func TestErrorParsing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/oauth/token" {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]interface{}{
					"code":    403,
					"message": "Access denied for this project",
				},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	p := NewNeuralGateProvider()
	_, err := p.getAccessToken(context.Background(), "id:secret", server.URL)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	expected := "NeuralGate OAuth token exchange error (status 403): Access denied for this project"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestRateLimiting(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/oauth/token" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token": "tok",
				"expires_in":   900,
			})
			return
		}
		if r.URL.Path == "/api/v1/ai/chat" {
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]interface{}{
					"code":    429,
					"message": "rate limit exceeded",
				},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	p := NewNeuralGateProvider()
	_, err := p.SendMessageStream(
		context.Background(),
		"id:secret",
		"model",
		server.URL,
		[]models.Message{{Role: models.RoleUser, Content: "Hi"}},
		nil,
		"",
	)
	if err == nil {
		t.Fatal("expected rate limit error, got nil")
	}
	if !contains(err.Error(), "rate limited") {
		t.Errorf("expected rate limit error, got: %v", err)
	}
	if !contains(err.Error(), "60") {
		t.Errorf("expected retry-after value in error, got: %v", err)
	}
}

func TestEmptyEndpoint(t *testing.T) {
	p := NewNeuralGateProvider()
	_, err := p.SendMessageStream(
		context.Background(),
		"id:secret",
		"model",
		"",
		[]models.Message{{Role: models.RoleUser, Content: "Hi"}},
		nil,
		"",
	)
	if err == nil {
		t.Fatal("expected error for empty endpoint, got nil")
	}
	if !contains(err.Error(), "endpoint URL is required") {
		t.Errorf("expected endpoint required error, got: %v", err)
	}
}

func TestMalformedSSEData(t *testing.T) {
	sseBody := `event: message
data: {"model":"test","message":{"role":"assistant","content":"Hello"},"done":false}

event: message
data: {INVALID JSON HERE}

event: message
data: {"model":"test","message":{"role":"assistant","content":""},"done":true,"prompt_eval_count":5,"eval_count":1}

`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/oauth/token" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token": "tok",
				"expires_in":   900,
			})
			return
		}
		if r.URL.Path == "/api/v1/ai/chat" {
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(sseBody))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	p := NewNeuralGateProvider()
	ch, err := p.SendMessageStream(
		context.Background(),
		"id:secret",
		"model",
		server.URL,
		[]models.Message{{Role: models.RoleUser, Content: "Hi"}},
		nil,
		"",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var chunks []models.StreamChunk
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}

	// Should get "Hello" chunk and done chunk; malformed line is skipped
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks (skipping malformed), got %d: %+v", len(chunks), chunks)
	}
	if chunks[0].Content != "Hello" {
		t.Errorf("expected first chunk content 'Hello', got %q", chunks[0].Content)
	}
	if !chunks[1].Done {
		t.Errorf("expected second chunk to be Done")
	}
}

func TestStreamInterruption(t *testing.T) {
	// Server that closes connection mid-stream
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/oauth/token" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token": "tok",
				"expires_in":   900,
			})
			return
		}
		if r.URL.Path == "/api/v1/ai/chat" {
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			flusher, _ := w.(http.Flusher)
			fmt.Fprintf(w, "event: message\ndata: {\"model\":\"test\",\"message\":{\"role\":\"assistant\",\"content\":\"Hello\"},\"done\":false}\n\n")
			flusher.Flush()
			// Close without sending done — simulates interrupted stream
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	p := NewNeuralGateProvider()
	ch, err := p.SendMessageStream(
		context.Background(),
		"id:secret",
		"model",
		server.URL,
		[]models.Message{{Role: models.RoleUser, Content: "Hi"}},
		nil,
		"",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var chunks []models.StreamChunk
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}

	// Should get at least the "Hello" chunk; channel should close cleanly
	if len(chunks) == 0 {
		t.Fatal("expected at least 1 chunk before stream interruption")
	}
	if chunks[0].Content != "Hello" {
		t.Errorf("expected content 'Hello', got %q", chunks[0].Content)
	}
}

func TestProviderMetadata(t *testing.T) {
	p := NewNeuralGateProvider()

	if p.Name() != "neuralgate" {
		t.Errorf("expected Name() = 'neuralgate', got %q", p.Name())
	}
	if p.SupportsTools() {
		t.Error("expected SupportsTools() = false")
	}
	if !p.SupportsStreaming() {
		t.Error("expected SupportsStreaming() = true")
	}
}

func TestListModels(t *testing.T) {
	p := NewNeuralGateProvider()
	models, err := p.ListModels(context.Background(), "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) != 1 || models[0] != "auto" {
		t.Errorf("expected [\"auto\"], got %v", models)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
