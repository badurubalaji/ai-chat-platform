package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mdp/ai-chat-platform/backend/internal/models"
)

type OllamaProvider struct {
	client *http.Client
}

func NewOllamaProvider() *OllamaProvider {
	return &OllamaProvider{
		client: &http.Client{},
	}
}

func (p *OllamaProvider) Name() string {
	return "ollama"
}

func (p *OllamaProvider) SupportsTools() bool {
	return false // Basic implementation
}

func (p *OllamaProvider) SupportsStreaming() bool {
	return true
}

type ollamaRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (p *OllamaProvider) SendMessageStream(ctx context.Context, apiKey, model, endpoint string, messages []models.Message, tools []models.Tool, systemPrompt string, files []models.FileAttachment) (<-chan models.StreamChunk, error) {
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}
	url := fmt.Sprintf("%s/api/chat", endpoint)

	var ollamaMsgs []ollamaMessage
	if systemPrompt != "" {
		ollamaMsgs = append(ollamaMsgs, ollamaMessage{Role: "system", Content: systemPrompt})
	}
	for _, m := range messages {
		ollamaMsgs = append(ollamaMsgs, ollamaMessage{Role: string(m.Role), Content: m.Content})
	}

	reqBody, err := json.Marshal(ollamaRequest{
		Model:    model,
		Messages: ollamaMsgs,
		Stream:   true,
	})
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
		return nil, fmt.Errorf("Ollama API error (status %d): %s", resp.StatusCode, string(body[:n]))
	}

	ch := make(chan models.StreamChunk)

	go func() {
		defer close(ch)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Bytes()

			var chunk struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
				Done bool `json:"done"`
			}

			if err := json.Unmarshal(line, &chunk); err == nil {
				if chunk.Done {
					ch <- models.StreamChunk{Done: true}
					return
				}
				ch <- models.StreamChunk{Content: chunk.Message.Content, Done: false}
			}
		}
	}()

	return ch, nil
}

func (p *OllamaProvider) SendMessageSync(ctx context.Context, apiKey, model, endpoint string, messages []models.Message, tools []models.Tool, systemPrompt string, files []models.FileAttachment) (*models.Message, error) {
	return nil, fmt.Errorf("not implemented")
}

func (p *OllamaProvider) ValidateCredentials(ctx context.Context, apiKey, model, endpoint string) error {
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint+"/api/tags", nil)
	if err != nil {
		return err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to Ollama at %s: %w", endpoint, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Ollama returned status %d", resp.StatusCode)
	}
	return nil
}

func (p *OllamaProvider) ListModels(ctx context.Context, apiKey, endpoint string) ([]string, error) {
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint+"/api/tags", nil)
	if err != nil {
		return nil, err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return []string{"llama3", "mistral"}, nil // fallback on connection error
	}
	defer resp.Body.Close()

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return []string{"llama3", "mistral"}, nil // fallback
	}

	var names []string
	for _, m := range result.Models {
		names = append(names, m.Name)
	}
	if len(names) == 0 {
		return []string{"llama3", "mistral"}, nil
	}
	return names, nil
}
