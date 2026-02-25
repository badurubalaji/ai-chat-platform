package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// MessageRole represents the role of a message sender
type MessageRole string

const (
	RoleUser       MessageRole = "user"
	RoleAssistant  MessageRole = "assistant"
	RoleSystem     MessageRole = "system"
	RoleToolUse    MessageRole = "tool_use"
	RoleToolResult MessageRole = "tool_result"
)

// FileAttachment represents a file attached to a chat message
type FileAttachment struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Base64Data  string `json:"base64"`
	Size        int64  `json:"size"`
}

// Message represents a single chat message
type Message struct {
	ID             uuid.UUID              `json:"id"`
	ConversationID uuid.UUID              `json:"conversation_id"`
	Role           MessageRole            `json:"role"`
	Content        string                 `json:"content"`
	Files          []FileAttachment       `json:"files,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
}

// Conversation represents a chat conversation
type Conversation struct {
	ID        uuid.UUID `json:"id"`
	TenantID  string    `json:"tenant_id"`
	UserID    uuid.UUID `json:"user_id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ProviderConfig represents the configuration for an AI provider
type ProviderConfig struct {
	ID              uuid.UUID              `json:"id"`
	TenantID        string                 `json:"tenant_id"`
	Provider        string                 `json:"provider"`
	Model           string                 `json:"model"`
	APIKeyEncrypted []byte                 `json:"-"`
	EndpointURL     string                 `json:"endpoint_url,omitempty"`
	Settings        map[string]interface{} `json:"settings,omitempty"`
	Enabled         bool                   `json:"enabled"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// Tool represents a tool that can be used by the AI
type Tool struct {
	Name         string      `json:"name"`
	Description  string      `json:"description"`
	Parameters   interface{} `json:"parameters"`    // JSON Schema
	RequiredRole string      `json:"required_role"` // admin, operator, viewer
}

// ToolCall represents a tool call from the AI
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// StreamChunk represents a chunk of data from a streaming response
type StreamChunk struct {
	Content  string    `json:"content"`
	Done     bool      `json:"done"`
	Usage    *Usage    `json:"usage,omitempty"`
	ToolCall *ToolCall `json:"tool_call,omitempty"`
	Error    error     `json:"-"`
}

// Usage represents token usage statistics
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// RegisteredTool represents a dynamically registered tool in the registry
type RegisteredTool struct {
	ID                   uuid.UUID       `json:"id"`
	TenantID             string          `json:"tenant_id"`
	AppName              string          `json:"app_name"`
	ToolName             string          `json:"tool_name"`
	Description          string          `json:"description"`
	Parameters           json.RawMessage `json:"parameters"`
	ExecutionConfig      json.RawMessage `json:"execution_config"`
	RequiresConfirmation bool            `json:"requires_confirmation"`
	Enabled              bool            `json:"enabled"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
}

// ToolRegistrationRequest is the request body for registering a tool
type ToolRegistrationRequest struct {
	AppName              string          `json:"app_name"`
	ToolName             string          `json:"tool_name"`
	Description          string          `json:"description"`
	Parameters           json.RawMessage `json:"parameters,omitempty"`
	Execution            json.RawMessage `json:"execution"`
	RequiresConfirmation bool            `json:"requires_confirmation"`
}

// ToolExecution represents an audit log entry for a tool execution
type ToolExecution struct {
	ID                uuid.UUID              `json:"id"`
	TenantID          string                 `json:"tenant_id"`
	UserID            uuid.UUID              `json:"user_id"`
	ConversationID    uuid.UUID              `json:"conversation_id"`
	ToolName          string                 `json:"tool_name"`
	Arguments         map[string]interface{} `json:"arguments"`
	Result            map[string]interface{} `json:"result,omitempty"`
	Status            string                 `json:"status"` // pending, success, error, cancelled
	ConfirmedByUser   bool                   `json:"confirmed_by_user"`
	ExecutionDuration int                    `json:"execution_duration_ms,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
}
