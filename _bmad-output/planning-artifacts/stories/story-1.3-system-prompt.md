# Story 1.3: Use Adapter System Prompt in handleChat()

Status: done

## Audit (2026-04-16)

- AC#1 ✅ `backend/internal/api/handler.go:510-515` — `resolveSystemPrompt()` returns `h.adapter.SystemPrompt()` when adapter non-nil.
- AC#2 ✅ Same method returns `"You are a helpful assistant."` fallback when nil.
- Invocation site: `handler.go:306`, passed to agent via `RunParams.SystemPrompt`.

## Story

As an end user of an embedded domain app,
I want the AI to respond with domain-specific knowledge,
so that the assistant understands my application context.

## Acceptance Criteria

1. **Given** an adapter with a custom system prompt, **When** a chat message is sent, **Then** the adapter's prompt is used instead of "You are a helpful assistant."
2. **Given** no adapter (nil), **When** a chat message is sent, **Then** the hardcoded fallback is used.

## Tasks / Subtasks

- [ ] Task 1: Replace hardcoded system prompt (AC: #1, #2)
  - [ ] Add `resolveSystemPrompt() string` method to Handler
  - [ ] Returns `h.adapter.SystemPrompt()` if adapter non-nil
  - [ ] Returns `"You are a helpful assistant."` otherwise
  - [ ] Replace hardcoded string at handler.go line 190

## Dev Notes

- Modify: `backend/internal/api/handler.go` line 190 — the `SendMessageStream` call
- Minimal change, safe to deploy independently
- No tool calling wired yet — that is Epic 3

### References

- [Source: backend/internal/api/handler.go#SendMessageStream call line 190]

## Dev Agent Record

### Agent Model Used
### Debug Log References
### Completion Notes List
### File List
