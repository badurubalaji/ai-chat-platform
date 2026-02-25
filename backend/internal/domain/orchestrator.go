package domain

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/mdp/ai-chat-platform/backend/internal/models"
)

// Orchestrator handles two-pass tool calling for models without native function calling.
type Orchestrator struct {
	adapter *Adapter
	tools   []ToolConfig // override tools (from registry, etc.)
}

// NewOrchestrator creates a new orchestrator for the given adapter.
func NewOrchestrator(adapter *Adapter) *Orchestrator {
	return &Orchestrator{adapter: adapter}
}

// NewOrchestratorWithTools creates an orchestrator with explicit tools (no adapter needed).
func NewOrchestratorWithTools(tools []ToolConfig) *Orchestrator {
	return &Orchestrator{tools: tools}
}

// Regex patterns for extracting tool calls from model responses.
var (
	// Match JSON inside markdown code fences: ```json ... ``` or ``` ... ```
	codeFencePattern = regexp.MustCompile("(?s)```(?:json)?\\s*\\n?(\\{.*?\\})\\s*\\n?```")
	// Match raw JSON object containing tool_call
	rawJSONPattern = regexp.MustCompile(`(?s)\{"tool_call"\s*:\s*\{.*?\}\s*\}`)
)

// BuildSystemPrompt creates the full system prompt with tool schemas injected.
// For providers that don't support native tool calling.
func (o *Orchestrator) BuildSystemPrompt() string {
	var base string
	if o.adapter != nil {
		base = o.adapter.SystemPrompt()
	}

	tools := o.getTools()
	if len(tools) == 0 {
		return base
	}

	var sb strings.Builder
	sb.WriteString(base)
	sb.WriteString("\n\n## Available Tools\n\nYou have access to the following tools. Use them when the user's request requires performing an action.\n\n")

	for _, t := range tools {
		sb.WriteString(fmt.Sprintf("### %s\n", t.Name))
		sb.WriteString(fmt.Sprintf("**Description:** %s\n", t.Description))

		if len(t.Parameters) > 0 {
			var prettyParams interface{}
			if err := json.Unmarshal(t.Parameters, &prettyParams); err == nil {
				paramBytes, _ := json.MarshalIndent(prettyParams, "", "  ")
				sb.WriteString(fmt.Sprintf("**Parameters:**\n```json\n%s\n```\n", string(paramBytes)))
			}
		}

		if t.RequiresConfirmation {
			sb.WriteString("**Note:** This tool requires user confirmation before execution.\n")
		}
		sb.WriteString("\n")
	}

	sb.WriteString(`## Tool Calling Instructions

When you need to use a tool, respond with ONLY a JSON object in this exact format (no other text):

` + "```json" + `
{"tool_call": {"name": "tool_name", "arguments": {"param": "value"}}}
` + "```" + `

Important rules:
- Respond with ONLY the JSON block when making a tool call — no other text before or after.
- Do NOT wrap additional explanation around a tool call.
- If the user's message does not require a tool, respond naturally with conversational text.
- Never fabricate tool results — wait for the actual execution result.
`)

	return sb.String()
}

// BuildSystemPromptWithTools creates a system prompt using explicit tools instead of adapter tools.
func (o *Orchestrator) BuildSystemPromptWithTools(tools []ToolConfig) string {
	saved := o.tools
	o.tools = tools
	defer func() { o.tools = saved }()
	return o.BuildSystemPrompt()
}

// getTools returns tools from override list or adapter.
func (o *Orchestrator) getTools() []ToolConfig {
	if len(o.tools) > 0 {
		return o.tools
	}
	if o.adapter != nil {
		return o.adapter.Tools()
	}
	return nil
}

// ParseToolCall attempts to extract a tool call from a model's text response.
// Returns the parsed ToolCall and any remaining conversational text.
// Returns nil ToolCall if the response is plain conversation.
func (o *Orchestrator) ParseToolCall(response string) (*models.ToolCall, string) {
	trimmed := strings.TrimSpace(response)

	// Strategy 1: Look for JSON inside markdown code fences
	if matches := codeFencePattern.FindStringSubmatch(trimmed); len(matches) > 1 {
		if tc := parseToolCallJSON(matches[1]); tc != nil {
			remaining := codeFencePattern.ReplaceAllString(trimmed, "")
			return tc, strings.TrimSpace(remaining)
		}
	}

	// Strategy 2: Look for raw {"tool_call": ...} JSON
	if match := rawJSONPattern.FindString(trimmed); match != "" {
		if tc := parseToolCallJSON(match); tc != nil {
			remaining := strings.Replace(trimmed, match, "", 1)
			return tc, strings.TrimSpace(remaining)
		}
	}

	// Strategy 3: Try parsing the entire response as JSON
	if tc := parseToolCallJSON(trimmed); tc != nil {
		return tc, ""
	}

	// No tool call found — this is plain conversational text
	return nil, response
}

// parseToolCallJSON attempts to parse a JSON string as a tool call.
func parseToolCallJSON(s string) *models.ToolCall {
	var wrapper struct {
		ToolCall struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		} `json:"tool_call"`
	}

	if err := json.Unmarshal([]byte(s), &wrapper); err != nil {
		return nil
	}

	if wrapper.ToolCall.Name == "" {
		return nil
	}

	return &models.ToolCall{
		ID:        "tc_" + uuid.New().String()[:8],
		Name:      wrapper.ToolCall.Name,
		Arguments: string(wrapper.ToolCall.Arguments),
	}
}
