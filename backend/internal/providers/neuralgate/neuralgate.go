package neuralgate

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/mdp/ai-chat-platform/backend/internal/models"
)

type cachedToken struct {
	accessToken string
	expiresAt   time.Time
}

// NeuralGateProvider implements the ChatProvider interface for the NeuralGate AI Gateway.
// It uses OAuth2 client_credentials grant for authentication and caches tokens for reuse.
type NeuralGateProvider struct {
	client     *http.Client
	tokenCache sync.Map // key: "client_id:client_secret" -> value: *cachedToken
}

func NewNeuralGateProvider() *NeuralGateProvider {
	return &NeuralGateProvider{
		client: &http.Client{
			// No global timeout — streaming responses can be long-lived.
			// ResponseHeaderTimeout handles the case where the server never responds.
			Transport: &http.Transport{
				ResponseHeaderTimeout: 120 * time.Second,
			},
		},
	}
}

func (p *NeuralGateProvider) Name() string {
	return "neuralgate"
}

func (p *NeuralGateProvider) SupportsTools() bool {
	return false
}

func (p *NeuralGateProvider) SupportsStreaming() bool {
	return true
}

// Request/response types for NeuralGate API

type neuralGateMessage struct {
	Role    string                     `json:"role"`
	Content string                     `json:"content"`
	Files   []neuralGateFileAttachment `json:"files,omitempty"`
}

type neuralGateFileAttachment struct {
	Base64      string `json:"base64"`
	ContentType string `json:"content_type"`
}

type neuralGateOptions struct {
	Temperature   float64  `json:"temperature,omitempty"`
	TopP          float64  `json:"top_p,omitempty"`
	TopK          int      `json:"top_k,omitempty"`
	NumPredict    int      `json:"num_predict,omitempty"`
	NumCtx        int      `json:"num_ctx,omitempty"`
	RepeatPenalty float64  `json:"repeat_penalty,omitempty"`
	Seed          int      `json:"seed,omitempty"`
	Stop          []string `json:"stop,omitempty"`
}

type neuralGateRequest struct {
	Model    string              `json:"model,omitempty"`
	Messages []neuralGateMessage `json:"messages"`
	Stream   bool                `json:"stream"`
	System   string              `json:"system,omitempty"`
	Options  *neuralGateOptions  `json:"options,omitempty"`
}

// parseCredentials splits "client_id:client_secret" into its parts.
// Uses SplitN with limit 2, so colons in the secret are preserved.
func parseCredentials(apiKey string) (clientID, clientSecret string, err error) {
	parts := strings.SplitN(apiKey, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid NeuralGate credentials format: expected 'client_id:client_secret'")
	}
	return parts[0], parts[1], nil
}

// oauthTokenFields holds the token fields we need from the OAuth response.
type oauthTokenFields struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

// getAccessToken obtains a Bearer token via OAuth2 client_credentials grant.
// Tokens are cached and reused until 30 seconds before expiry.
// Handles both flat responses {"access_token": "..."} and
// data-wrapped responses {"data": {"access_token": "..."}}.
func (p *NeuralGateProvider) getAccessToken(ctx context.Context, apiKey, endpoint string) (string, error) {
	// Check cache first
	if cached, ok := p.tokenCache.Load(apiKey); ok {
		ct := cached.(*cachedToken)
		if time.Now().Before(ct.expiresAt.Add(-30 * time.Second)) {
			return ct.accessToken, nil
		}
	}

	clientID, clientSecret, err := parseCredentials(apiKey)
	if err != nil {
		return "", err
	}

	tokenURL := strings.TrimRight(endpoint, "/") + "/api/v1/oauth/token"

	reqBody, _ := json.Marshal(map[string]string{
		"client_id":     clientID,
		"client_secret": clientSecret,
		"grant_type":    "client_credentials",
	})

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to connect to NeuralGate OAuth endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", p.parseError(resp, "OAuth token exchange")
	}

	// Read the full response body for flexible parsing
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read NeuralGate OAuth response: %w", err)
	}
	log.Printf("[NEURALGATE] OAuth response body: %s", string(body))

	// Try flat format first: {"access_token": "...", "expires_in": 900}
	var tokenResp oauthTokenFields
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse NeuralGate OAuth response: %w", err)
	}

	// If flat format didn't yield an access_token, try data-wrapped format:
	// {"data": {"access_token": "...", "expires_in": 900}}
	if tokenResp.AccessToken == "" {
		var wrappedResp struct {
			Data oauthTokenFields `json:"data"`
		}
		if err := json.Unmarshal(body, &wrappedResp); err == nil && wrappedResp.Data.AccessToken != "" {
			tokenResp = wrappedResp.Data
			log.Printf("[NEURALGATE] OAuth token found in data-wrapped response")
		}
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("NeuralGate OAuth returned empty access token (response: %s)", string(body))
	}

	// Cache the token
	p.tokenCache.Store(apiKey, &cachedToken{
		accessToken: tokenResp.AccessToken,
		expiresAt:   time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	})

	return tokenResp.AccessToken, nil
}

// parseError reads NeuralGate's error response format: {"error": {"code": ..., "message": ...}}
func (p *NeuralGateProvider) parseError(resp *http.Response, context string) error {
	var errResp struct {
		Error struct {
			Code    json.RawMessage `json:"code"`
			Message string          `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Error.Message != "" {
		return fmt.Errorf("NeuralGate %s error (status %d): %s", context, resp.StatusCode, errResp.Error.Message)
	}
	return fmt.Errorf("NeuralGate %s error (status %d)", context, resp.StatusCode)
}

func (p *NeuralGateProvider) SendMessageStream(ctx context.Context, apiKey, model, endpoint string, messages []models.Message, tools []models.Tool, systemPrompt string, files []models.FileAttachment) (<-chan models.StreamChunk, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("endpoint URL is required for NeuralGate provider")
	}

	// Get OAuth2 Bearer token
	token, err := p.getAccessToken(ctx, apiKey, endpoint)
	if err != nil {
		return nil, err
	}

	chatURL := strings.TrimRight(endpoint, "/") + "/api/v1/ai/chat"

	// Map messages to NeuralGate format; system prompt goes in the top-level "system" field
	var ngMsgs []neuralGateMessage
	for i, m := range messages {
		ngMsg := neuralGateMessage{Role: string(m.Role), Content: m.Content}
		// Attach files to the last user message
		if i == len(messages)-1 && m.Role == models.RoleUser && len(files) > 0 {
			for _, f := range files {
				ngMsg.Files = append(ngMsg.Files, neuralGateFileAttachment{
					Base64:      f.Base64Data,
					ContentType: f.ContentType,
				})
			}
		}
		ngMsgs = append(ngMsgs, ngMsg)
	}

	reqBody, err := json.Marshal(neuralGateRequest{
		Model:    model,
		Messages: ngMsgs,
		Stream:   true,
		System:   systemPrompt,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", chatURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusTooManyRequests {
			retryAfter := resp.Header.Get("Retry-After")
			return nil, fmt.Errorf("NeuralGate rate limited, retry after %s seconds", retryAfter)
		}
		return nil, p.parseError(resp, "chat")
	}

	ch := make(chan models.StreamChunk)

	go func() {
		defer close(ch)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		var currentEvent string

		for scanner.Scan() {
			line := scanner.Text()

			// Parse SSE event type
			if strings.HasPrefix(line, "event: ") {
				currentEvent = strings.TrimPrefix(line, "event: ")
				continue
			}

			// Parse SSE data payload
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")

			switch currentEvent {
			case "message":
				var msg struct {
					Model   string `json:"model"`
					Message struct {
						Role    string `json:"role"`
						Content string `json:"content"`
					} `json:"message"`
					Done            bool `json:"done"`
					PromptEvalCount int  `json:"prompt_eval_count"`
					EvalCount       int  `json:"eval_count"`
				}
				if err := json.Unmarshal([]byte(data), &msg); err != nil {
					continue
				}

				if msg.Done {
					ch <- models.StreamChunk{
						Done: true,
						Usage: &models.Usage{
							InputTokens:  msg.PromptEvalCount,
							OutputTokens: msg.EvalCount,
						},
					}
					return
				}

				if msg.Message.Content != "" {
					ch <- models.StreamChunk{Content: msg.Message.Content, Done: false}
				}

			case "conversation":
				// NeuralGate sends conversation_id and message_id here.
				// We ignore this because the platform manages its own conversations.
				continue
			}
		}
	}()

	return ch, nil
}

func (p *NeuralGateProvider) SendMessageSync(ctx context.Context, apiKey, model, endpoint string, messages []models.Message, tools []models.Tool, systemPrompt string, files []models.FileAttachment) (*models.Message, error) {
	return nil, fmt.Errorf("not implemented")
}

func (p *NeuralGateProvider) ValidateCredentials(ctx context.Context, apiKey, model, endpoint string) error {
	if endpoint == "" {
		return fmt.Errorf("endpoint URL is required for NeuralGate provider")
	}
	// Validate by attempting OAuth2 token exchange
	_, err := p.getAccessToken(ctx, apiKey, endpoint)
	if err != nil {
		return fmt.Errorf("NeuralGate credential validation failed: %w", err)
	}
	return nil
}

func (p *NeuralGateProvider) ListModels(ctx context.Context, apiKey, endpoint string) ([]string, error) {
	// NeuralGate auto-resolves models from project configuration
	return []string{"auto"}, nil
}
