# Story 3.2: Wire Two-Pass Tool Calling Path (NeuralGateway/Ollama)

Status: ready-for-dev

## Story

As an end user using the default NeuralGateway provider,
I want tools to work through prompt-engineered two-pass orchestration,
so that I get the same tool functionality as BYOK users.

## Acceptance Criteria

1. **Given** `SupportsTools()=false` and adapter tools, **When** chat sent, **Then** `BuildSystemPrompt()` used and tools NOT passed to provider.
2. **Given** model response contains tool_call JSON, **When** full response accumulated, **Then** handler parses, executes, and makes second model call with result.
3. **Given** no tool_call in response, **When** `ParseToolCall()` returns nil, **Then** response streams as-is.

## Tasks / Subtasks

- [ ] Task 1: Branch handleChat() on `SupportsTools()` (AC: #1)
  - [ ] Before `SendMessageStream()`, check `provider.SupportsTools()`
  - [ ] If false AND adapter has tools: use `orchestrator.BuildSystemPrompt()`, keep tools=nil
  - [ ] If true: use adapter raw prompt, pass tools (Story 3.1)
- [ ] Task 2: Implement two-pass response handling (AC: #2, #3)
  - [ ] For non-tool providers, accumulate full response text
  - [ ] After stream completes: `orchestrator.ParseToolCall(fullResponse)`
  - [ ] If tool call found:
    - Emit `tool_call` SSE event
    - Execute via `executor.ExecuteTool()`
    - Emit `tool_result` SSE event
    - Build second-pass messages with tool result
    - Call `SendMessageStream()` again, stream to client
  - [ ] If no tool call: response already streamed normally
- [ ] Task 3: Handle first-pass UX
  - [ ] Stream first pass normally; if tool call detected, emit SSE event to frontend to replace last message
  - [ ] Simple approach: stream everything, use `tool_call` event as signal to frontend

## Dev Notes

- Primary: `backend/internal/api/handler.go`
- For V1, stream first pass and let frontend handle replacement when tool_call is detected
- The second pass uses NeuralGateway's conversation_id to maintain context
- If model returns mixed text + JSON, ParseToolCall returns remaining text for display

### References

- [Source: backend/internal/domain/orchestrator.go#BuildSystemPrompt — from Story 2.1]
- [Source: backend/internal/domain/orchestrator.go#ParseToolCall — from Story 2.2]

## Dev Agent Record

### Agent Model Used
### Debug Log References
### Completion Notes List
### File List
