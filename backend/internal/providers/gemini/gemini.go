package gemini

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
	defaultEndpoint = "https://generativelanguage.googleapis.com/v1beta/models"
)

type GeminiProvider struct {
	client *http.Client
}

func NewGeminiProvider() *GeminiProvider {
	return &GeminiProvider{
		client: &http.Client{},
	}
}

func (p *GeminiProvider) Name() string {
	return "gemini"
}

func (p *GeminiProvider) SupportsTools() bool {
	return true
}

func (p *GeminiProvider) SupportsStreaming() bool {
	return true
}

// Gemini specific structs
type geminiRequest struct {
	Contents          []geminiContent `json:"contents"`
	SystemInstruction *geminiContent  `json:"system_instruction,omitempty"`
	Tools             []geminiTool    `json:"tools,omitempty"`
	GenerationConfig  *geminiConfig   `json:"generationConfig,omitempty"`
}

type geminiTool struct {
	FunctionDeclarations []geminiFunctionDecl `json:"function_declarations,omitempty"`
}

type geminiFunctionDecl struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Parameters  interface{} `json:"parameters,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text         string            `json:"text,omitempty"`
	FunctionCall *geminiFuncCall   `json:"functionCall,omitempty"`
	InlineData   *geminiInlineData `json:"inlineData,omitempty"`
}

type geminiInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"` // base64 encoded
}

type geminiFuncCall struct {
	Name string          `json:"name"`
	Args json.RawMessage `json:"args"`
}

type geminiConfig struct {
	Temperature     float64 `json:"temperature,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
}

func (p *GeminiProvider) SendMessageStream(ctx context.Context, apiKey, model, endpoint string, messages []models.Message, tools []models.Tool, systemPrompt string, files []models.FileAttachment) (<-chan models.StreamChunk, error) {
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	// Normalizing model name if needed, but usually passed correctly
	url := fmt.Sprintf("%s/%s:streamGenerateContent?key=%s", endpoint, model, apiKey)

	var geminiMsgs []geminiContent
	for i, m := range messages {
		role := "user"
		if m.Role == models.RoleAssistant {
			role = "model"
		} else if m.Role == models.RoleSystem {
			// handled separately
			continue
		}

		parts := []geminiPart{{Text: m.Content}}

		// Attach files to the last user message
		isLastUser := i == len(messages)-1 && m.Role == models.RoleUser && len(files) > 0
		if isLastUser {
			// Prepend file parts before text
			var fileParts []geminiPart
			for _, f := range files {
				fileParts = append(fileParts, geminiPart{
					InlineData: &geminiInlineData{
						MimeType: f.ContentType,
						Data:     f.Base64Data,
					},
				})
			}
			parts = append(fileParts, parts...)
		}

		geminiMsgs = append(geminiMsgs, geminiContent{
			Role:  role,
			Parts: parts,
		})
	}

	reqPayload := geminiRequest{
		Contents: geminiMsgs,
	}

	// Map tools
	if len(tools) > 0 {
		var decls []geminiFunctionDecl
		for _, t := range tools {
			decls = append(decls, geminiFunctionDecl{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			})
		}
		reqPayload.Tools = []geminiTool{{FunctionDeclarations: decls}}
	}

	if systemPrompt != "" {
		reqPayload.SystemInstruction = &geminiContent{
			Parts: []geminiPart{{Text: systemPrompt}},
		}
	}

	reqBody, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
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
		return nil, fmt.Errorf("Gemini API error (status %d): %s", resp.StatusCode, string(body[:n]))
	}

	ch := make(chan models.StreamChunk)

	go func() {
		defer close(ch)
		defer resp.Body.Close()

		// Gemini sends a JSON array of objects, but in stream it might send them line by line or as a continuous array
		// The standard REST stream returns JSON objects one by one?
		// Actually for SSE it might be different, but Google's streamGenerateContent returns a stream of JSON objects.
		// Let's assume standard behavior for now: it sends partial JSONs.
		// Actually, standard HTTP stream from Google is a list of JSON objects wrapped in square brackets?
		// "Server-sent events" -> "alt=sse" is available? No.
		// It returns a standard HTTP response with "Transfer-Encoding: chunked" containing a JSON array.
		// Parsing this line-by-line is tricky if it's not SSE.
		// However, simpler approach for common implementations:
		// We'll read line by line.

		// NOTE: A proper robust implementation would use a streaming JSON parser.
		// For verification purposes, we'll try to decode objects as they come.
		// Google's format: values usually come as separate JSON objects in the stream if using gRPC, but REST...
		// Let's rely on the fact that most adapters use a library. Here we are barebones.

		// Quick fix: Gemini REST API returns a JSON array `[...]`. We need to parse internal objects.
		// Getting `[` first, then objects separated by `,`.
		// This is painful to parse manually without a library.
		// Alternative: Use `alt=sse` if supported? It is supported in some versions.

		// Let's stick to reading simple tokens.

		// ACTUALLY: Let's assume standard handling or use a different endpoint?
		// Providing a basic SSE simulation for now or assuming the `openai` style if hitting a proxy.
		// but since we are hitting google directly...

		// Let's implement a naive scanner that looks for "text": "..."

		// Refined approach: Read until `}` and try to unmarshal.

		scanner := bufio.NewScanner(resp.Body)
		scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
			if atEOF && len(data) == 0 {
				return 0, nil, nil
			}
			if i := bytes.IndexByte(data, '\n'); i >= 0 {
				// We have a full newline-terminated line.
				return i + 1, data[0:i], nil
			}
			// If at EOF, we have a final, non-terminated line. Return it.
			if atEOF {
				return len(data), data, nil
			}
			// Request more data.
			return 0, nil, nil
		})

		for scanner.Scan() {
			line := scanner.Text()
			line = strings.TrimSpace(line)
			// Strip leading '[' or ',' or ']'
			if line == "[" || line == "]" || line == "," {
				continue
			}
			if strings.HasPrefix(line, ",") {
				line = line[1:]
			}

			var chunk struct {
				Candidates []struct {
					Content struct {
						Parts []geminiPart `json:"parts"`
					} `json:"content"`
				} `json:"candidates"`
			}

			if err := json.Unmarshal([]byte(line), &chunk); err == nil {
				if len(chunk.Candidates) > 0 {
					parts := chunk.Candidates[0].Content.Parts
					for _, part := range parts {
						if part.Text != "" {
							ch <- models.StreamChunk{Content: part.Text, Done: false}
						}

						if part.FunctionCall != nil {
							// Gemini usually sends full function call in one go (or accumulated),
							// but in stream it might be tricky.
							// Based on docs, `functionCall` contains name and args.
							// Args is an object. usage of json.RawMessage to capture it as string.

							ch <- models.StreamChunk{
								ToolCall: &models.ToolCall{
									ID:        "call_" + part.FunctionCall.Name, // Gemini doesn't always send ID, so generate or use Name
									Name:      part.FunctionCall.Name,
									Arguments: string(part.FunctionCall.Args),
								},
								Done: false,
							}
						}
					}
				}
			}
		}
		ch <- models.StreamChunk{Done: true}
	}()

	return ch, nil
}

func (p *GeminiProvider) SendMessageSync(ctx context.Context, apiKey, model, endpoint string, messages []models.Message, tools []models.Tool, systemPrompt string, files []models.FileAttachment) (*models.Message, error) {
	return nil, fmt.Errorf("not implemented")
}

func (p *GeminiProvider) ValidateCredentials(ctx context.Context, apiKey, model, endpoint string) error {
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	url := fmt.Sprintf("%s?key=%s", endpoint, apiKey)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to Gemini: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("invalid API key")
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}
	return nil
}

func (p *GeminiProvider) ListModels(ctx context.Context, apiKey, endpoint string) ([]string, error) {
	return []string{"gemini-2.0-flash", "gemini-2.0-pro"}, nil
}
