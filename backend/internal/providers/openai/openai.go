package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mdp/ai-chat-platform/backend/internal/models"
)

const (
	defaultEndpoint = "https://api.openai.com/v1/chat/completions"
)

type OpenAIProvider struct {
	client *http.Client
}

func NewOpenAIProvider() *OpenAIProvider {
	return &OpenAIProvider{
		client: &http.Client{},
	}
}

func (p *OpenAIProvider) Name() string {
	return "openai"
}

func (p *OpenAIProvider) SupportsTools() bool {
	return true
}

func (p *OpenAIProvider) SupportsStreaming() bool {
	return true
}

type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Stream      bool            `json:"stream,omitempty"`
	Tools       []openAITool    `json:"tools,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAITool struct {
	Type     string         `json:"type"` // "function"
	Function openAIFunction `json:"function"`
}

type openAIFunction struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Parameters  interface{} `json:"parameters,omitempty"` // JSON Schema
}

func (p *OpenAIProvider) SendMessageStream(ctx context.Context, apiKey, model, endpoint string, messages []models.Message, tools []models.Tool, systemPrompt string) (<-chan models.StreamChunk, error) {
	if endpoint == "" {
		endpoint = defaultEndpoint
	}

	// Prepare messages (prepend system prompt if present)
	var openAIMsgs []openAIMessage
	if systemPrompt != "" {
		openAIMsgs = append(openAIMsgs, openAIMessage{Role: "system", Content: systemPrompt})
	}
	for _, m := range messages {
		openAIMsgs = append(openAIMsgs, openAIMessage{Role: string(m.Role), Content: m.Content})
	}

	// Map tools
	var openAITools []openAITool
	if len(tools) > 0 {
		for _, t := range tools {
			openAITools = append(openAITools, openAITool{
				Type: "function",
				Function: openAIFunction{
					Name:        t.Name,
					Description: t.Description,
					Parameters:  t.Parameters,
				},
			})
		}
	}

	reqBody, err := json.Marshal(openAIRequest{
		Model:    model,
		Messages: openAIMsgs,
		Tools:    openAITools,
		Stream:   true,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		return nil, fmt.Errorf("OpenAI API error (status %d): %s", resp.StatusCode, string(body[:n]))
	}

	ch := make(chan models.StreamChunk)

	go func() {
		defer close(ch)
		defer resp.Body.Close()

		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}

			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				ch <- models.StreamChunk{Done: true}
				return
			}

			var chunk struct {
				Choices []struct {
					Delta struct {
						Content   string `json:"content"`
						ToolCalls []struct {
							Index    int    `json:"index"`
							ID       string `json:"id"`
							Type     string `json:"type"`
							Function struct {
								Name      string `json:"name"`
								Arguments string `json:"arguments"`
							} `json:"function"`
						} `json:"tool_calls"`
					} `json:"delta"`
				} `json:"choices"`
			}
			if err := json.Unmarshal([]byte(data), &chunk); err == nil {
				if len(chunk.Choices) > 0 {
					delta := chunk.Choices[0].Delta

					// Handle Content
					if delta.Content != "" {
						ch <- models.StreamChunk{Content: delta.Content, Done: false}
					}

					// Handle Tool Calls
					for _, tc := range delta.ToolCalls {
						toolCall := &models.ToolCall{}

						if tc.ID != "" {
							toolCall.ID = tc.ID
							toolCall.Name = tc.Function.Name
						}

						if tc.Function.Arguments != "" {
							toolCall.Arguments = tc.Function.Arguments
						}

						// Only send if we have something
						if toolCall.ID != "" || toolCall.Name != "" || toolCall.Arguments != "" {
							ch <- models.StreamChunk{
								ToolCall: toolCall,
								Done:     false,
							}
						}
					}
				}
			}
		}
	}()

	return ch, nil
}

func (p *OpenAIProvider) SendMessageSync(ctx context.Context, apiKey, model, endpoint string, messages []models.Message, tools []models.Tool, systemPrompt string) (*models.Message, error) {
	return nil, fmt.Errorf("not implemented")
}

func (p *OpenAIProvider) ValidateCredentials(ctx context.Context, apiKey, model, endpoint string) error {
	if endpoint == "" {
		endpoint = "https://api.openai.com/v1"
	}
	url := strings.TrimRight(endpoint, "/")
	if !strings.HasSuffix(url, "/models") {
		url += "/models"
	}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to OpenAI: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("invalid API key")
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}
	return nil
}

func (p *OpenAIProvider) ListModels(ctx context.Context, apiKey, endpoint string) ([]string, error) {
	return []string{"gpt-4o", "gpt-4o-mini", "o1", "o3-mini"}, nil
}
