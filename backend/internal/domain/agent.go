package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mdp/ai-chat-platform/backend/internal/models"
	"github.com/mdp/ai-chat-platform/backend/internal/providers"
)

const DefaultMaxIterations = 5

// AgentEvent represents an event emitted by the agent during execution.
type AgentEvent struct {
	Type string // "chunk", "tool_call", "tool_confirm", "tool_result", "done", "error"
	Data string // JSON-encoded event data
}

// ConfirmFunc is called when a tool requires user confirmation.
// It should block until the user responds or times out.
// Returns true if approved, false if denied.
type ConfirmFunc func(confirmID, toolName, description string, params json.RawMessage) (approved bool, err error)

// Agent orchestrates multi-step tool-use conversations.
// It runs a loop: send to LLM -> detect tool call -> execute -> repeat.
type Agent struct {
	provider      providers.ChatProvider
	executor      *Executor
	registry      *ToolRegistry
	orchestrator  *Orchestrator
	adapter       *Adapter
	maxIterations int
}

// AgentConfig holds parameters for creating an agent.
type AgentConfig struct {
	Provider      providers.ChatProvider
	Executor      *Executor
	Registry      *ToolRegistry
	Orchestrator  *Orchestrator
	Adapter       *Adapter
	MaxIterations int
}

// NewAgent creates a new agent with the given configuration.
func NewAgent(cfg AgentConfig) *Agent {
	maxIter := cfg.MaxIterations
	if maxIter <= 0 {
		maxIter = DefaultMaxIterations
	}
	return &Agent{
		provider:      cfg.Provider,
		executor:      cfg.Executor,
		registry:      cfg.Registry,
		orchestrator:  cfg.Orchestrator,
		adapter:       cfg.Adapter,
		maxIterations: maxIter,
	}
}

// RunParams holds all parameters needed for an agent run.
type RunParams struct {
	APIKey       string
	Model        string
	Endpoint     string
	Messages     []models.Message
	SystemPrompt string
	Files        []models.FileAttachment
	TenantID     string
	UserID       uuid.UUID
	ConvoID      uuid.UUID
	ConfirmFn    ConfirmFunc // handles tool confirmation (blocks until user responds)
}

// Run executes the agent loop, streaming events to the returned channel.
// The agent will:
// 1. Send messages to the LLM
// 2. If LLM returns a tool call, execute it and loop
// 3. If LLM returns text (no tool call), emit it and stop
// 4. Repeat up to maxIterations
func (a *Agent) Run(ctx context.Context, params RunParams) (<-chan AgentEvent, string) {
	events := make(chan AgentEvent, 100)

	go func() {
		defer close(events)

		messages := make([]models.Message, len(params.Messages))
		copy(messages, params.Messages)

		// Resolve tools
		tools, toolConfigs := a.resolveTools(ctx, params.TenantID)
		useNativeTools := len(tools) > 0 && a.provider.SupportsTools()
		useTwoPass := len(tools) > 0 && !a.provider.SupportsTools()

		systemPrompt := params.SystemPrompt
		if useTwoPass && a.orchestrator != nil {
			systemPrompt = a.orchestrator.BuildSystemPromptWithTools(toolConfigs)
		}

		var nativeTools []models.Tool
		if useNativeTools {
			nativeTools = tools
		}

		fullResponse := ""
		var totalInputTokens, totalOutputTokens int

		for iteration := 0; iteration < a.maxIterations; iteration++ {
			log.Printf("[AGENT] Iteration %d/%d, messages: %d, tools: %d", iteration+1, a.maxIterations, len(messages), len(tools))

			// Send to LLM
			var files []models.FileAttachment
			if iteration == 0 {
				files = params.Files // only send files on first iteration
			}

			stream, err := a.provider.SendMessageStream(ctx, params.APIKey, params.Model, params.Endpoint, messages, nativeTools, systemPrompt, files)
			if err != nil {
				events <- AgentEvent{Type: "error", Data: err.Error()}
				return
			}

			// Consume stream
			iterResponse, inputTok, outputTok, detectedToolCall := a.consumeStream(ctx, events, stream, useTwoPass)
			totalInputTokens += inputTok
			totalOutputTokens += outputTok

			// Two-pass: parse tool call from text
			if useTwoPass && detectedToolCall == nil && iterResponse != "" && a.orchestrator != nil {
				if tc, _ := a.orchestrator.ParseToolCall(iterResponse); tc != nil {
					detectedToolCall = tc
					log.Printf("[AGENT] Two-pass detected tool call: %s", tc.Name)
				}
			}

			// No tool call -> final response, we're done
			if detectedToolCall == nil {
				fullResponse = iterResponse
				break
			}

			// Tool call detected -> execute it
			toolName := detectedToolCall.Name
			arguments := json.RawMessage(detectedToolCall.Arguments)

			// Look up tool config
			toolConfig := a.lookupToolConfig(toolName, toolConfigs)
			if toolConfig == nil {
				log.Printf("[AGENT] Unknown tool: %s", toolName)
				aMsg, tMsg := a.provider.FormatToolResult(iterResponse, detectedToolCall, fmt.Sprintf("tool '%s' not found", toolName), true)
				messages = append(messages, aMsg, tMsg)
				continue
			}

			// Handle confirmation if required
			if toolConfig.RequiresConfirmation && params.ConfirmFn != nil {
				confirmID := uuid.New().String()[:12]
				approved, err := params.ConfirmFn(confirmID, toolName, toolConfig.Description, arguments)
				if err != nil || !approved {
					log.Printf("[AGENT] Tool %s cancelled by user", toolName)
					// Tell model it was cancelled
					messages = append(messages,
						models.Message{Role: models.RoleAssistant, Content: iterResponse},
						models.Message{Role: models.RoleSystem, Content: fmt.Sprintf("The user cancelled the '%s' action. Acknowledge this gracefully.", toolName)},
					)
					continue
				}
			}

			// Execute the tool
			events <- AgentEvent{
				Type: "tool_call",
				Data: fmt.Sprintf(`{"tool":"%s","status":"executing"}`, toolName),
			}

			start := time.Now()
			result, execErr := a.executor.ExecuteToolConfig(ctx, toolConfig, arguments, params.TenantID, params.UserID.String())
			durationMs := int(time.Since(start).Milliseconds())

			if execErr != nil {
				log.Printf("[AGENT] Tool %s failed in %dms: %v", toolName, durationMs, execErr)
				events <- AgentEvent{
					Type: "tool_result",
					Data: fmt.Sprintf(`{"tool":"%s","status":"error","error":"%s"}`, toolName, execErr.Error()),
				}
				aMsg, tMsg := a.provider.FormatToolResult(iterResponse, detectedToolCall, execErr.Error(), true)
				messages = append(messages, aMsg, tMsg)
			} else {
				log.Printf("[AGENT] Tool %s executed in %dms", toolName, durationMs)
				events <- AgentEvent{
					Type: "tool_result",
					Data: fmt.Sprintf(`{"tool":"%s","status":"complete"}`, toolName),
				}
				aMsg, tMsg := a.provider.FormatToolResult(iterResponse, detectedToolCall, string(result), false)
				messages = append(messages, aMsg, tMsg)
			}

			// Loop continues — model will process tool result
		}

		// Emit done event
		doneData, _ := json.Marshal(struct {
			Content string       `json:"content"`
			Done    bool         `json:"done"`
			ConvoID string       `json:"conversation_id"`
			Usage   *models.Usage `json:"usage,omitempty"`
		}{
			Done:    true,
			ConvoID: params.ConvoID.String(),
			Usage: &models.Usage{
				InputTokens:  totalInputTokens,
				OutputTokens: totalOutputTokens,
			},
		})
		events <- AgentEvent{Type: "done", Data: string(doneData)}

		// Store full response in the last event for caller to use
		if fullResponse != "" {
			events <- AgentEvent{Type: "_response", Data: fullResponse}
		}
	}()

	return events, ""
}

// consumeStream reads the provider stream, emitting chunk events and returning the full response.
// It accumulates streaming tool call fragments (ID, Name, Arguments) into a single ToolCall.
func (a *Agent) consumeStream(ctx context.Context, events chan<- AgentEvent, stream <-chan models.StreamChunk, isTwoPass bool) (string, int, int, *models.ToolCall) {
	fullResponse := ""
	var inputTokens, outputTokens int
	var detectedToolCall *models.ToolCall
	var toolCallArgs strings.Builder

	for chunk := range stream {
		if chunk.Error != nil {
			events <- AgentEvent{Type: "error", Data: chunk.Error.Error()}
			return fullResponse, inputTokens, outputTokens, nil
		}

		if chunk.Done {
			if chunk.Usage != nil {
				inputTokens = chunk.Usage.InputTokens
				outputTokens = chunk.Usage.OutputTokens
			}
			continue
		}

		fullResponse += chunk.Content

		if chunk.ToolCall != nil {
			if detectedToolCall == nil {
				detectedToolCall = &models.ToolCall{
					ID:   chunk.ToolCall.ID,
					Name: chunk.ToolCall.Name,
				}
			}
			if chunk.ToolCall.ID != "" {
				detectedToolCall.ID = chunk.ToolCall.ID
			}
			if chunk.ToolCall.Name != "" {
				detectedToolCall.Name = chunk.ToolCall.Name
			}
			if chunk.ToolCall.Arguments != "" {
				toolCallArgs.WriteString(chunk.ToolCall.Arguments)
			}
		}

		// Stream to client
		data, _ := json.Marshal(struct {
			Content  string           `json:"content"`
			ToolCall *models.ToolCall `json:"tool_call,omitempty"`
		}{
			Content:  chunk.Content,
			ToolCall: chunk.ToolCall,
		})
		events <- AgentEvent{Type: "chunk", Data: string(data)}
	}

	// Finalize accumulated arguments
	if detectedToolCall != nil && toolCallArgs.Len() > 0 {
		detectedToolCall.Arguments = toolCallArgs.String()
	}

	return fullResponse, inputTokens, outputTokens, detectedToolCall
}

// resolveTools gets all available tools in both model and config format.
func (a *Agent) resolveTools(ctx context.Context, tenantID string) ([]models.Tool, []ToolConfig) {
	if a.registry != nil {
		modelTools, err := a.registry.GetAllModelTools(ctx, tenantID)
		if err != nil {
			log.Printf("[AGENT] Failed to load registry tools: %v", err)
		}
		configs, err := a.registry.GetAllToolConfigs(ctx, tenantID)
		if err != nil {
			log.Printf("[AGENT] Failed to load registry tool configs: %v", err)
		}
		if len(modelTools) > 0 {
			return modelTools, configs
		}
	}

	// Fallback to adapter-only
	if a.adapter != nil && a.adapter.HasTools() {
		return a.adapter.ToolsForProvider(), a.adapter.Tools()
	}

	return nil, nil
}

// lookupToolConfig finds a tool config by name from the pre-resolved list.
func (a *Agent) lookupToolConfig(name string, configs []ToolConfig) *ToolConfig {
	for i := range configs {
		if configs[i].Name == name {
			return &configs[i]
		}
	}
	return nil
}
