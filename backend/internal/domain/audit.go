package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/mdp/ai-chat-platform/backend/internal/models"
)

// NewToolExecution creates a ToolExecution record for audit logging.
func NewToolExecution(tenantID string, userID, conversationID uuid.UUID, toolName string, arguments json.RawMessage) *models.ToolExecution {
	var args map[string]interface{}
	if len(arguments) > 0 {
		json.Unmarshal(arguments, &args)
	}

	return &models.ToolExecution{
		ID:             uuid.New(),
		TenantID:       tenantID,
		UserID:         userID,
		ConversationID: conversationID,
		ToolName:       toolName,
		Arguments:      args,
		Status:         "pending",
		CreatedAt:      time.Now(),
	}
}

// MarkSuccess updates the execution record with success status and result.
func MarkSuccess(exec *models.ToolExecution, result json.RawMessage, durationMs int) {
	exec.Status = "success"
	exec.ExecutionDuration = durationMs
	if len(result) > 0 {
		json.Unmarshal(result, &exec.Result)
	}
}

// MarkError updates the execution record with error status.
func MarkError(exec *models.ToolExecution, errMsg string, durationMs int) {
	exec.Status = "error"
	exec.ExecutionDuration = durationMs
	exec.Result = map[string]interface{}{"error": errMsg}
}

// MarkCancelled updates the execution record with cancelled status.
func MarkCancelled(exec *models.ToolExecution) {
	exec.Status = "cancelled"
}
