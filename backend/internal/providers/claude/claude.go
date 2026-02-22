package claude

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
	defaultEndpoint = "https://api.anthropic.com/v1/messages"
	defaultVersion  = "2023-06-01"
)

type anthropicErrorResponse struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

type ClaudeProvider struct {
	client *http.Client
}

func NewClaudeProvider() *ClaudeProvider {
	return &ClaudeProvider{
		client: &http.Client{},
	}
}

type claudeRequest struct {
	Model     string          `json:"model"`
	Messages  []claudeMessage `json:"messages"`
	System    string          `json:"system,omitempty"`
	MaxTokens int             `json:"max_tokens,omitempty"`
	Stream    bool            `json:"stream,omitempty"`
	Tools     []claudeTool    `json:"tools,omitempty"`
}

type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type claudeTool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"input_schema"`
}

type claudeStreamEvent struct {
	Type         string `json:"type"`
	Index        int    `json:"index"`
	ContentBlock *struct {
		Type string `json:"type"`
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"content_block,omitempty"`
	Delta *struct {
		Type        string `json:"type"`
		Text        string `json:"text"`
		PartialJSON string `json:"partial_json"`
	} `json:"delta,omitempty"`
}

func (p *ClaudeProvider) Name() string {
	return "claude"
}

func (p *ClaudeProvider) SupportsTools() bool {
	return true
}

func (p *ClaudeProvider) SupportsStreaming() bool {
	return true
}

func (p *ClaudeProvider) SendMessageStream(ctx context.Context, apiKey, model, endpoint string, messages []models.Message, tools []models.Tool, systemPrompt string) (<-chan models.StreamChunk, error) {
	if endpoint == "" {
		endpoint = defaultEndpoint
	}

	reqBody, err := p.prepareRequest(model, messages, tools, systemPrompt, true)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	p.setHeaders(req, apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return nil, p.parseError(resp)
	}

	ch := make(chan models.StreamChunk)

	go func() {
		defer close(ch)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		var currentEvent string

		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "event: ") {
				currentEvent = strings.TrimPrefix(line, "event: ")
				continue
			}
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")

			switch currentEvent {
			case "content_block_start":
				var event claudeStreamEvent
				if err := json.Unmarshal([]byte(data), &event); err != nil {
					continue
				}
				if event.ContentBlock != nil && event.ContentBlock.Type == "tool_use" {
					// Send start of tool call
					ch <- models.StreamChunk{
						ToolCall: &models.ToolCall{
							ID:   event.ContentBlock.ID,
							Name: event.ContentBlock.Name,
						},
					}
				}

			case "content_block_delta":
				var event claudeStreamEvent
				if err := json.Unmarshal([]byte(data), &event); err != nil {
					continue
				}
				if event.Delta != nil {
					if event.Delta.Type == "text_delta" {
						ch <- models.StreamChunk{Content: event.Delta.Text}
					} else if event.Delta.Type == "input_json_delta" {
						ch <- models.StreamChunk{
							ToolCall: &models.ToolCall{
								Arguments: event.Delta.PartialJSON,
							},
						}
					}
				}

			case "message_stop":
				ch <- models.StreamChunk{Done: true}
				return
			}
		}
		// Handle ping config etc
	}()

	return ch, nil
}

func (p *ClaudeProvider) SendMessageSync(ctx context.Context, apiKey, model, endpoint string, messages []models.Message, tools []models.Tool, systemPrompt string) (*models.Message, error) {
	// Implementation for sync (omitted for brevity, similar to stream but unmarshal full response)
	return nil, fmt.Errorf("not implemented")
}

func (p *ClaudeProvider) ValidateCredentials(ctx context.Context, apiKey string, model string, endpoint string) error {
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	reqBody, _ := json.Marshal(claudeRequest{
		Model:     model,
		Messages:  []claudeMessage{{Role: "user", Content: "Hi"}},
		MaxTokens: 1,
	})
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}
	p.setHeaders(req, apiKey)
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to Anthropic: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return p.parseError(resp)
	}
	return nil
}

func (p *ClaudeProvider) ListModels(ctx context.Context, apiKey string, endpoint string) ([]string, error) {
	return []string{"claude-sonnet-4-5-20250929", "claude-opus-4-6", "claude-haiku-4-5-20251001"}, nil
}

func (p *ClaudeProvider) setHeaders(req *http.Request, apiKey string) {
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", defaultVersion)
	req.Header.Set("content-type", "application/json")
}

func (p *ClaudeProvider) prepareRequest(model string, messages []models.Message, tools []models.Tool, systemPrompt string, stream bool) ([]byte, error) {
	// Adapt models.Message to claudeMessage
	var claudeMsgs []claudeMessage
	for _, m := range messages {
		claudeMsgs = append(claudeMsgs, claudeMessage{
			Role:    string(m.Role),
			Content: m.Content,
		})
	}

	req := claudeRequest{
		Model:     model,
		Messages:  claudeMsgs,
		System:    systemPrompt,
		MaxTokens: 4096,
		Stream:    stream,
	}
	// Handle tools if present
	if len(tools) > 0 {
		var claudeTools []claudeTool
		for _, t := range tools {
			claudeTools = append(claudeTools, claudeTool{
				Name:        t.Name,
				Description: t.Description,
				InputSchema: t.Parameters,
			})
		}
		req.Tools = claudeTools
	}

	return json.Marshal(req)
}
