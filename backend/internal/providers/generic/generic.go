package generic

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

// GenericOpenAIProvider works with any OpenAI-compatible API
// (LM Studio, vLLM, Together AI, Groq, etc.)
type GenericOpenAIProvider struct {
	client *http.Client
}

func NewGenericOpenAIProvider() *GenericOpenAIProvider {
	return &GenericOpenAIProvider{
		client: &http.Client{},
	}
}

func (p *GenericOpenAIProvider) Name() string {
	return "generic"
}

func (p *GenericOpenAIProvider) SupportsTools() bool {
	return true
}

func (p *GenericOpenAIProvider) SupportsStreaming() bool {
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
	Role       string           `json:"role"`
	Content    interface{}      `json:"content"`
	ToolCalls  *json.RawMessage `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type openAITool struct {
	Type     string         `json:"type"`
	Function openAIFunction `json:"function"`
}

type openAIFunction struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Parameters  interface{} `json:"parameters,omitempty"`
}

func (p *GenericOpenAIProvider) FormatToolResult(assistantText string, toolCall *models.ToolCall, result string, isError bool) (models.Message, models.Message) {
	toolCallObj := map[string]interface{}{
		"id":   toolCall.ID,
		"type": "function",
		"function": map[string]interface{}{
			"name":      toolCall.Name,
			"arguments": toolCall.Arguments,
		},
	}
	assistantMeta := map[string]interface{}{
		"_openai_tool_calls": []interface{}{toolCallObj},
	}
	assistantMsg := models.Message{
		Role:     models.RoleAssistant,
		Content:  assistantText,
		Metadata: assistantMeta,
	}

	content := result
	if isError {
		content = "Error: " + result
	}
	toolResultMsg := models.Message{
		Role:    models.RoleToolResult,
		Content: content,
		Metadata: map[string]interface{}{
			"_openai_tool_call_id": toolCall.ID,
			"_openai_tool_name":    toolCall.Name,
		},
	}
	return assistantMsg, toolResultMsg
}

func (p *GenericOpenAIProvider) SendMessageStream(ctx context.Context, apiKey, model, endpoint string, messages []models.Message, tools []models.Tool, systemPrompt string, files []models.FileAttachment) (<-chan models.StreamChunk, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("endpoint URL is required for generic OpenAI provider")
	}

	// Ensure endpoint ends with /chat/completions
	url := strings.TrimRight(endpoint, "/")
	if !strings.HasSuffix(url, "/chat/completions") {
		url += "/chat/completions"
	}

	var openAIMsgs []openAIMessage
	if systemPrompt != "" {
		openAIMsgs = append(openAIMsgs, openAIMessage{Role: "system", Content: systemPrompt})
	}
	for _, m := range messages {
		// Handle OpenAI-native tool call messages (from FormatToolResult)
		if m.Metadata != nil {
			if toolCalls, ok := m.Metadata["_openai_tool_calls"]; ok {
				tcJSON, _ := json.Marshal(toolCalls)
				raw := json.RawMessage(tcJSON)
				openAIMsgs = append(openAIMsgs, openAIMessage{
					Role:      "assistant",
					Content:   m.Content,
					ToolCalls: &raw,
				})
				continue
			}
			if toolCallID, ok := m.Metadata["_openai_tool_call_id"]; ok {
				openAIMsgs = append(openAIMsgs, openAIMessage{
					Role:       "tool",
					Content:    m.Content,
					ToolCallID: fmt.Sprintf("%v", toolCallID),
				})
				continue
			}
		}
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

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body[:n]))
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

					if delta.Content != "" {
						ch <- models.StreamChunk{Content: delta.Content, Done: false}
					}

					for _, tc := range delta.ToolCalls {
						toolCall := &models.ToolCall{}
						if tc.ID != "" {
							toolCall.ID = tc.ID
							toolCall.Name = tc.Function.Name
						}
						if tc.Function.Arguments != "" {
							toolCall.Arguments = tc.Function.Arguments
						}
						if toolCall.ID != "" || toolCall.Name != "" || toolCall.Arguments != "" {
							ch <- models.StreamChunk{ToolCall: toolCall, Done: false}
						}
					}
				}
			}
		}
	}()

	return ch, nil
}

func (p *GenericOpenAIProvider) SendMessageSync(ctx context.Context, apiKey, model, endpoint string, messages []models.Message, tools []models.Tool, systemPrompt string, files []models.FileAttachment) (*models.Message, error) {
	return nil, fmt.Errorf("not implemented")
}

func (p *GenericOpenAIProvider) ValidateCredentials(ctx context.Context, apiKey, model, endpoint string) error {
	if endpoint == "" {
		return fmt.Errorf("endpoint URL is required for generic provider")
	}
	url := strings.TrimRight(endpoint, "/") + "/models"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("endpoint returned status %d", resp.StatusCode)
	}
	return nil
}

func (p *GenericOpenAIProvider) ListModels(ctx context.Context, apiKey, endpoint string) ([]string, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("endpoint URL is required")
	}
	url := strings.TrimRight(endpoint, "/") + "/models"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var modelNames []string
	for _, m := range result.Data {
		modelNames = append(modelNames, m.ID)
	}
	return modelNames, nil
}
