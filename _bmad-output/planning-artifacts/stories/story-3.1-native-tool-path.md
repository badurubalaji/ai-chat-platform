# Story 3.1: Wire Native Tool Calling Path (Claude/OpenAI/Gemini)

Status: done

## Audit (2026-04-16)

- AC#1 ✅ `backend/internal/domain/agent.go` resolves tools, sets `useNativeTools` when `provider.SupportsTools()`; `nativeTools` passed to `SendMessageStream`.
- AC#2 ✅ Confirmation branch skipped when `!RequiresConfirmation`; tool executes immediately; `tool_result` events emitted; loop iterates so next provider call runs with tool result appended.
- AC#3 ✅ Multi-iteration loop (default 5), tool result appended via `FormatToolResult`; model's follow-up streams as chunk events.
- AC#4 ✅ Error path: `FormatToolResult(..., isError=true)`, error `tool_result` SSE emitted, model sees the error in next iteration and generates an error-aware response.
- Task 3 ✅ Tool interactions are now persisted to `ai_messages`:
  - New role `models.RoleToolCall = "tool_call"` (aligned with frontend) at `models/models.go:19`.
  - Agent emits internal `_tool_msg` event after success, error, and cancel paths via `emitToolMessage()` helper (`agent.go`).
  - Handler's event loop unmarshals `_tool_msg` and calls `h.store.CreateMessage` (`handler.go`).
  - Provider history at `handler.go` skips `RoleToolCall` rows so providers receive only user/assistant/system turns on reload.
- Design divergence (positive): entire flow was factored into `domain.Agent` rather than inline in `handleChat`, and expanded to **multi-step** tool calling (not just two-pass). This is an improvement over the story's "second SendMessageStream call" design.

## Story

As an end user with a BYOK Claude/OpenAI/Gemini key,
I want the AI to use domain tools through native function calling,
so that tool invocations work seamlessly within the streaming chat experience.

## Acceptance Criteria

1. **Given** `SupportsTools()=true` and adapter tools, **When** chat sent, **Then** tools passed to `SendMessageStream()` instead of nil.
2. **Given** provider returns ToolCall chunk and tool does NOT require confirmation, **When** detected, **Then** handler executes immediately and sends result back to model.
3. **Given** tool execution succeeds, **When** result sent to model, **Then** model streams a natural response to client.
4. **Given** tool execution fails, **When** error sent to model, **Then** model generates an error-aware response.

## Tasks / Subtasks

- [ ] Task 1: Pass tools for native providers (AC: #1)
  - [ ] In handleChat(), check `provider.SupportsTools() && h.adapter != nil`
  - [ ] If true: replace nil with `h.adapter.ToolsForProvider()` at line 190
- [ ] Task 2: Handle ToolCall chunks in streaming loop (AC: #2, #3, #4)
  - [ ] Detect `chunk.ToolCall != nil` in stream loop (lines 204-256)
  - [ ] Accumulate tool call arguments across incremental chunks
  - [ ] When complete: check `RequiresConfirmation`
  - [ ] If no confirmation: call `executor.ExecuteTool()`
  - [ ] Emit `tool_call` SSE event to frontend
  - [ ] Build tool result message, append to history
  - [ ] Make SECOND `SendMessageStream()` call with updated history
  - [ ] Stream second response to client
- [ ] Task 3: Save tool_use and tool_result messages to DB (AC: #3)
  - [ ] Save tool_use message (model's call) and tool_result message (execution result)
  - [ ] Use existing `RoleToolUse` and `RoleToolResult` roles

## Dev Notes

- Primary: `backend/internal/api/handler.go` — handleChat() lines 92-287
- ToolCall accumulation: Claude sends `input_json_delta` chunks, OpenAI sends incremental `arguments` strings
- The second SSE stream is a continuation of the SAME HTTP response
- Existing roles: `RoleToolUse`, `RoleToolResult` at models.go lines 16-17

### References

- [Source: backend/internal/api/handler.go#handleChat lines 92-287]
- [Source: backend/internal/providers/provider.go#ChatProvider interface lines 12-30]
- [Source: backend/internal/models/models.go#StreamChunk.ToolCall line 73]

## Dev Agent Record

### Agent Model Used
### Debug Log References
### Completion Notes List
### File List
