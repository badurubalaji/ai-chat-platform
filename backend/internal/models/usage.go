package models

import (
	"time"

	"github.com/google/uuid"
)

// ... existing models ...

type UsageLog struct {
	ID             uuid.UUID `json:"id"`
	TenantID       string    `json:"tenant_id"`
	UserID         uuid.UUID `json:"user_id"`
	ConversationID uuid.UUID `json:"conversation_id"`
	Provider       string    `json:"provider"`
	Model          string    `json:"model"`
	InputTokens    int       `json:"input_tokens"`
	OutputTokens   int       `json:"output_tokens"`
	CreatedAt      time.Time `json:"created_at"`
}

type UsageStats struct {
	Daily             []UsagePeriod `json:"daily"`
	TotalInputTokens  int           `json:"total_input_tokens"`
	TotalOutputTokens int           `json:"total_output_tokens"`
	ConversationCount int           `json:"conversation_count"`
}

type UsagePeriod struct {
	Date         string `json:"date"`
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
	RequestCount int    `json:"request_count"`
}
