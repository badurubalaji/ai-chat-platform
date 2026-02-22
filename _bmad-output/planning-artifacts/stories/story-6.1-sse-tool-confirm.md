# Story 6.1: Handle tool_confirm SSE Event in AiChatService

Status: ready-for-dev

## Story

As the frontend service layer,
I want to parse `tool_confirm` SSE events from the backend,
so that the chat component can display confirmation UI.

## Acceptance Criteria

1. **Given** backend sends `event: tool_confirm`, **When** service parses it, **Then** emits `AiStreamChunk` with `tool_confirm` field.
2. **Given** `tool_confirm` chunk emitted, **When** chat component receives it, **Then** it can distinguish from regular tool_call events.

## Tasks / Subtasks

- [ ] Task 1: Extend `AiStreamChunk` model (AC: #1)
  - [ ] Add `tool_confirm?: AiToolConfirmation` to `AiStreamChunk`
  - [ ] Define `AiToolConfirmation`: `{ confirmation_id: string; tool: string; description: string; params: Record<string, unknown> }`
- [ ] Task 2: Handle `tool_confirm` event in `fetchStream()` (AC: #1, #2)
  - [ ] Add case `'tool_confirm'` in SSE switch (~line 103)
  - [ ] Emit chunk with `tool_confirm` field populated

## Dev Notes

- Model: `frontend/projects/mdp-ai-chat/src/lib/models/ai-chat.model.ts`
- Service: `frontend/projects/mdp-ai-chat/src/lib/services/ai-chat.service.ts`
- Backward compatible: existing tool_call/tool_result handling unchanged

### References

- [Source: frontend/.../ai-chat.model.ts#AiStreamChunk interface]
- [Source: frontend/.../ai-chat.service.ts#fetchStream SSE parsing ~line 84-110]

## Dev Agent Record

### Agent Model Used
### Debug Log References
### Completion Notes List
### File List
