# Story 6.3: Update Frontend to Show Tool Execution Lifecycle

Status: done

## Audit (2026-04-16)

- AC#1 ✅ `ai-message.component.ts:20-21` renders `<mat-spinner diameter="16">` when `tool_status === 'executing'` (and no active confirmation card).
- AC#2 ✅ `ai-chat.component.ts:492-518 handleToolCallEvent` updates the existing tool_call message in place (no duplicate) by matching on tool_name + status. Complete status renders `check_circle` at `ai-message.component.ts:27-29`.
- AC#3 ✅ `error` status renders `error` icon at `ai-message.component.ts:25-26`; backend sends `{"tool":"...","status":"error","error":"..."}` SSE at `agent.go:187-190`.
- AC#4 ✅ On approve (`ai-chat.component.ts:520-528`), message transitions to `tool_status: 'executing'` with `tool_confirmation` cleared — confirmation card disappears and spinner appears. Buttons aren't "disabled" verbatim but they're removed entirely, satisfying the UX intent. Cancelled path adds explicit `'cancelled'` status + `block` icon (`ai-message.component.ts:22-24`).

## Story

As an end user,
I want to see the full lifecycle of a tool execution in the chat,
so that I understand what the AI is doing on my behalf.

## Acceptance Criteria

1. **Given** tool executing, **When** `tool_call` SSE with status `executing`, **Then** spinner shown.
2. **Given** tool finished, **When** `tool_result` SSE arrives, **Then** spinner replaced with checkmark.
3. **Given** tool failed, **When** error tool_result arrives, **Then** error indicator shown.
4. **Given** confirmation approved, **When** tool executing, **Then** action card buttons disabled with "Processing..." indicator.

## Tasks / Subtasks

- [ ] Task 1: Enhance tool message status updates (AC: #1, #2, #3)
  - [ ] Modify `addToolMessage()` to support updating existing tool message status
  - [ ] When `tool_result` arrives for existing `executing` message, update metadata instead of adding duplicate
- [ ] Task 2: Handle confirmation state in action card (AC: #4)
  - [ ] Add `status` field: `'pending' | 'approved' | 'dismissed' | 'executing'`
  - [ ] When approved: update to `'approved'` then `'executing'`
  - [ ] Render disabled state in `AiActionCardComponent` when not `'pending'`

## Dev Notes

- Component: `frontend/.../ai-chat/ai-chat.component.ts` — `addToolMessage()` and `updateAssistantMessage()`
- `AiMessageComponent` already handles tool_call with spinner/checkmark
- Key change: make status update reactive via signal `messages.update()`
- Action card: `frontend/.../ai-action-card/ai-action-card.component.ts`

### References

- [Source: frontend/.../ai-message/ai-message.component.ts#tool_call rendering]
- [Source: frontend/.../ai-chat.component.ts#addToolMessage]
- [Source: frontend/.../ai-action-card/ai-action-card.component.ts#template]

## Dev Agent Record

### Agent Model Used
### Debug Log References
### Completion Notes List
### File List
