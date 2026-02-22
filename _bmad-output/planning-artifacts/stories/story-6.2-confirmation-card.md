# Story 6.2: Display Confirmation Card in Chat Component

Status: ready-for-dev

## Story

As an end user,
I want to see a clear confirmation card when the AI wants to perform a destructive action,
so that I can review and approve or reject.

## Acceptance Criteria

1. **Given** `tool_confirm` chunk received, **When** processed, **Then** confirmation action card displayed in chat.
2. **Given** card displayed, **When** user clicks "Approve", **Then** confirmation POST sent to backend.
3. **Given** card displayed, **When** user clicks "Dismiss", **Then** dismissal POST sent and card marked dismissed.

## Tasks / Subtasks

- [ ] Task 1: Handle tool_confirm in sendMessage() (AC: #1)
  - [ ] Add check for `chunk.tool_confirm` in stream subscription
  - [ ] Create message with role `'tool_call'` and metadata.action_card populated from confirmation data
  - [ ] Store confirmation_id for later approval/dismissal
- [ ] Task 2: Add `sendConfirmation()` to AiChatService (AC: #2, #3)
  - [ ] `sendConfirmation(confirmationId: string, approved: boolean): Observable<void>`
  - [ ] POST to `${apiUrl}/chat/confirm` with `{ confirmation_id, approved }`
- [ ] Task 3: Wire AiActionCardComponent events (AC: #2, #3)
  - [ ] Handle `apply` output: call `sendConfirmation(id, true)`
  - [ ] Handle `dismiss` output: call `sendConfirmation(id, false)`
  - [ ] Update message metadata to reflect decision

## Dev Notes

- Component: `frontend/.../ai-chat/ai-chat.component.ts`
- `AiActionCardComponent` already has `apply` and `dismiss` outputs
- `AiProposedAction` interface already has the right shape for confirmation cards
- The existing `actionRequested` EventEmitter on AiChatComponent re-emits upward — need to handle internally for confirmation

### References

- [Source: frontend/.../ai-chat.component.ts#sendMessage stream handling]
- [Source: frontend/.../ai-action-card/ai-action-card.component.ts#apply/dismiss outputs]
- [Source: frontend/.../ai-chat.model.ts#AiProposedAction interface]

## Dev Agent Record

### Agent Model Used
### Debug Log References
### Completion Notes List
### File List
