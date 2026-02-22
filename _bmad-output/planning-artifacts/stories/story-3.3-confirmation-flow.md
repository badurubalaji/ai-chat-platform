# Story 3.3: Implement Confirmation Flow for Destructive Tools

Status: ready-for-dev

## Story

As an end user,
I want the AI to ask for my confirmation before executing destructive actions,
so that I can review and approve or reject operations.

## Acceptance Criteria

1. **Given** tool with `requires_confirmation=true`, **When** handler detects the call, **Then** emits `tool_confirm` SSE event with details.
2. **Given** `tool_confirm` emitted, **When** user sends approval via `/api/v1/ai/chat/confirm`, **Then** tool is executed normally.
3. **Given** `tool_confirm` emitted, **When** user dismisses, **Then** tool NOT executed and model receives "User cancelled".
4. **Given** confirm endpoint called with unknown ID, **When** looked up, **Then** 404 returned.

## Tasks / Subtasks

- [ ] Task 1: Implement pending confirmations store (AC: #1, #4)
  - [ ] Add `pendingConfirmations sync.Map` to Handler
  - [ ] `PendingConfirmation` struct: ID, ConversationID, ToolName, Arguments, CreatedAt, ResultChan (buffered chan, cap 1)
  - [ ] Cleanup goroutine: remove expired confirmations (TTL: 5 min)
- [ ] Task 2: Emit `tool_confirm` SSE event (AC: #1)
  - [ ] When confirmation-required tool detected, create PendingConfirmation
  - [ ] Emit: `event: tool_confirm\ndata: {"confirmation_id":"...","tool":"...","description":"...","params":{...}}\n\n`
  - [ ] Block with `select` on ResultChan or context cancellation
- [ ] Task 3: Add `/api/v1/ai/chat/confirm` POST endpoint (AC: #2, #3)
  - [ ] Add route in `ServeHTTP()` method
  - [ ] Request: `{"confirmation_id": "...", "approved": true/false}`
  - [ ] Look up pending, send result to channel, return 200
- [ ] Task 4: Handle confirmation result (AC: #2, #3)
  - [ ] Approved: execute tool, continue with response
  - [ ] Dismissed: tool_result with "User cancelled", send to model for graceful response

## Dev Notes

- Primary: `backend/internal/api/handler.go`
- SSE connection stays open while waiting — the `tool_confirm` event is on the same stream
- ResultChan MUST be buffered (cap 1) to prevent goroutine leak on timeout
- Add route at handler.go ~line 73 in ServeHTTP switch block
- Use `sync.Map` for thread-safe concurrent access

### References

- [Source: backend/internal/api/handler.go#ServeHTTP routing ~line 63-90]

## Dev Agent Record

### Agent Model Used
### Debug Log References
### Completion Notes List
### File List
