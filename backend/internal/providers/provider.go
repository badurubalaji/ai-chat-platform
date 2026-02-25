package providers

import (
	"context"

	"github.com/mdp/ai-chat-platform/backend/internal/models"
)

// ChatProvider defines the interface for interacting with different AI providers
type ChatProvider interface {
	// SendMessageStream sends a message to the AI provider and returns a stream of chunks
	SendMessageStream(ctx context.Context, apiKey, model, endpoint string, messages []models.Message, tools []models.Tool, systemPrompt string, files []models.FileAttachment) (<-chan models.StreamChunk, error)

	// SendMessageSync sends a message to the AI provider and returns the complete response
	SendMessageSync(ctx context.Context, apiKey, model, endpoint string, messages []models.Message, tools []models.Tool, systemPrompt string, files []models.FileAttachment) (*models.Message, error)

	// ValidateCredentials checks if the provided credentials are valid
	ValidateCredentials(ctx context.Context, apiKey string, model string, endpoint string) error

	// ListModels returns a list of available models for the provider
	ListModels(ctx context.Context, apiKey string, endpoint string) ([]string, error)

	// Name returns the name of the provider (e.g., "claude", "openai")
	Name() string

	// SupportsTools returns true if the provider supports tool use
	SupportsTools() bool

	// SupportsStreaming returns true if the provider supports streaming
	SupportsStreaming() bool

	// FormatToolResult returns the (assistant message, tool result message) pair
	// in the provider's native format for multi-turn tool calling.
	// assistantText is the text the assistant generated before the tool call.
	// toolCall is the tool call that was made.
	// result is the tool execution result (JSON string).
	// isError indicates whether the result is an error.
	FormatToolResult(assistantText string, toolCall *models.ToolCall, result string, isError bool) (assistantMsg models.Message, toolResultMsg models.Message)
}
