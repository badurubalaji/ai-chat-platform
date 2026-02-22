# Story 4.2: Wire Audit Logging into Tool Executor and Store

Status: ready-for-dev

## Story

As a platform operator,
I want every tool execution logged with timing, status, and confirmation info,
so that audit queries show who did what and when.

## Acceptance Criteria

1. **Given** successful tool execution, **When** complete, **Then** row inserted with status=success and duration.
2. **Given** failed tool execution, **When** error occurs, **Then** row inserted with status=error.
3. **Given** user dismisses confirmation, **When** processed, **Then** row inserted with status=cancelled.

## Tasks / Subtasks

- [ ] Task 1: Add `LogToolExecution` to Store interface (AC: #1-#3)
  - [ ] Add method to Store interface in `store.go`
  - [ ] Implement SQL INSERT in PostgresStore
- [ ] Task 2: Create audit helper (AC: #1-#3)
  - [ ] Create `backend/internal/domain/audit.go`
  - [ ] Helper wraps tool execution with timing measurement
  - [ ] Returns `*models.ToolExecution` ready for logging
- [ ] Task 3: Wire into handleChat() tool execution points
  - [ ] After every `executor.ExecuteTool()` call, log execution
  - [ ] After every cancellation, log with status=cancelled

## Dev Notes

- Modify: `backend/internal/store/store.go` — add method following `LogUsage` pattern (line 139)
- Create: `backend/internal/domain/audit.go`
- Use `time.Since(start).Milliseconds()` for duration

### References

- [Source: backend/internal/store/store.go#LogUsage pattern lines 139-143]
- [Source: backend/internal/models/models.go#ToolExecution — from Story 4.1]

## Dev Agent Record

### Agent Model Used
### Debug Log References
### Completion Notes List
### File List
