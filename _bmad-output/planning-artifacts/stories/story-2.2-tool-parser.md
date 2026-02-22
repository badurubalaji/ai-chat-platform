# Story 2.2: Build Tool-Call Parser for Two-Pass Responses

Status: ready-for-dev

## Story

As the platform,
I want to parse tool-call intent from model text responses,
so that non-tool-supporting models can trigger tool execution.

## Acceptance Criteria

1. **Given** response with JSON tool_call block, **When** `ParseToolCall()` is called, **Then** returns `*models.ToolCall` with name and arguments.
2. **Given** plain conversational text, **When** `ParseToolCall()` is called, **Then** returns nil.
3. **Given** malformed JSON, **When** `ParseToolCall()` is called, **Then** returns nil (graceful degradation).
4. **Given** JSON in markdown code fences, **When** `ParseToolCall()` is called, **Then** still extracts correctly.

## Tasks / Subtasks

- [ ] Task 1: Implement `ParseToolCall()` in `orchestrator.go` (AC: #1-#4)
  - [ ] Signature: `ParseToolCall(response string) (*models.ToolCall, string)` — returns tool call + remaining text
  - [ ] Strategy: regex for ```json...``` blocks first, then raw `{"tool_call":` pattern
  - [ ] Validate: must have `name` string and `arguments` object
  - [ ] Generate unique ID: `"tc_" + uuid`
  - [ ] If multiple tool calls, return first only (V1)
- [ ] Task 2: Write table-driven unit tests
  - [ ] Clean JSON in code fence
  - [ ] JSON without code fence
  - [ ] Text mixed with tool call
  - [ ] Pure conversational (no tool)
  - [ ] Malformed JSON
  - [ ] Multiple tool calls (returns first)

## Dev Notes

- Add to: `backend/internal/domain/orchestrator.go`
- `models.ToolCall` at `backend/internal/models/models.go:63` has `ID`, `Name`, `Arguments` (string)
- Use `regexp.MustCompile` at package level for performance
- The "remaining text" return allows handler to stream conversational content if model mixed text with tool call

### References

- [Source: backend/internal/models/models.go#ToolCall struct lines 63-67]

## Dev Agent Record

### Agent Model Used
### Debug Log References
### Completion Notes List
### File List
