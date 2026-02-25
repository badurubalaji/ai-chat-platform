package claude

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
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
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // string for text-only, []claudeContentBlock for multimodal
}

type claudeContentBlock struct {
	Type   string             `json:"type"` // "text", "image", "document"
	Text   string             `json:"text,omitempty"`
	Source *claudeMediaSource `json:"source,omitempty"`
}

type claudeMediaSource struct {
	Type      string `json:"type"`       // "base64"
	MediaType string `json:"media_type"` // e.g. "image/png", "application/pdf"
	Data      string `json:"data"`       // base64 encoded
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

func (p *ClaudeProvider) SendMessageStream(ctx context.Context, apiKey, model, endpoint string, messages []models.Message, tools []models.Tool, systemPrompt string, files []models.FileAttachment) (<-chan models.StreamChunk, error) {
	if endpoint == "" {
		endpoint = defaultEndpoint
	}

	reqBody, err := p.prepareRequest(model, messages, tools, systemPrompt, true, files)
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

func (p *ClaudeProvider) SendMessageSync(ctx context.Context, apiKey, model, endpoint string, messages []models.Message, tools []models.Tool, systemPrompt string, files []models.FileAttachment) (*models.Message, error) {
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

func (p *ClaudeProvider) FormatToolResult(assistantText string, toolCall *models.ToolCall, result string, isError bool) (models.Message, models.Message) {
	// Claude expects:
	// 1. Assistant message with tool_use content block
	// 2. User message with tool_result content block
	toolUseBlock := map[string]interface{}{
		"type":  "tool_use",
		"id":    toolCall.ID,
		"name":  toolCall.Name,
		"input": json.RawMessage(toolCall.Arguments),
	}

	assistantContent := []interface{}{}
	if assistantText != "" {
		assistantContent = append(assistantContent, map[string]interface{}{
			"type": "text",
			"text": assistantText,
		})
	}
	assistantContent = append(assistantContent, toolUseBlock)

	assistantJSON, _ := json.Marshal(assistantContent)
	assistantMsg := models.Message{
		Role:    models.RoleAssistant,
		Content: string(assistantJSON),
		Metadata: map[string]interface{}{
			"_raw_content": true, // signal to prepareRequest to use raw JSON
		},
	}

	toolResultBlock := map[string]interface{}{
		"type":       "tool_result",
		"tool_use_id": toolCall.ID,
		"content":    result,
	}
	if isError {
		toolResultBlock["is_error"] = true
	}

	resultContent := []interface{}{toolResultBlock}
	resultJSON, _ := json.Marshal(resultContent)
	toolResultMsg := models.Message{
		Role:    models.RoleUser,
		Content: string(resultJSON),
		Metadata: map[string]interface{}{
			"_raw_content": true,
		},
	}

	return assistantMsg, toolResultMsg
}

func (p *ClaudeProvider) prepareRequest(model string, messages []models.Message, tools []models.Tool, systemPrompt string, stream bool, files []models.FileAttachment) ([]byte, error) {
	// Adapt models.Message to claudeMessage
	var claudeMsgs []claudeMessage
	for i, m := range messages {
		// Handle raw content blocks (from FormatToolResult)
		if m.Metadata != nil {
			if _, ok := m.Metadata["_raw_content"]; ok {
				var rawContent interface{}
				if err := json.Unmarshal([]byte(m.Content), &rawContent); err == nil {
					role := string(m.Role)
					claudeMsgs = append(claudeMsgs, claudeMessage{Role: role, Content: rawContent})
					continue
				}
			}
		}

		// Check if this is the last user message and has file attachments
		isLastUser := i == len(messages)-1 && m.Role == models.RoleUser && len(files) > 0

		if isLastUser {
			// Build multimodal content blocks
			var blocks []claudeContentBlock
			for _, f := range files {
				blockType := "image" // default for images
				if f.ContentType == "application/pdf" {
					blockType = "document"
				} else if !strings.HasPrefix(f.ContentType, "image/") {
					log.Printf("[CLAUDE] Skipping unsupported file type: %s", f.ContentType)
					continue
				}
				blocks = append(blocks, claudeContentBlock{
					Type: blockType,
					Source: &claudeMediaSource{
						Type:      "base64",
						MediaType: f.ContentType,
						Data:      f.Base64Data,
					},
				})
			}
			// Add the text content
			if m.Content != "" {
				blocks = append(blocks, claudeContentBlock{
					Type: "text",
					Text: m.Content,
				})
			}
			claudeMsgs = append(claudeMsgs, claudeMessage{
				Role:    string(m.Role),
				Content: blocks,
			})
		} else {
			claudeMsgs = append(claudeMsgs, claudeMessage{
				Role:    string(m.Role),
				Content: m.Content,
			})
		}
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
