# Story 2.1: Build System Prompt Injection for Two-Pass Tool Orchestration

Status: ready-for-dev

## Story

As the platform,
I want to inject tool schemas into the system prompt for non-tool-supporting models,
so that NeuralGateway and Ollama can express tool-calling intent through structured text.

## Acceptance Criteria

1. **Given** adapter tools, **When** `BuildSystemPrompt()` is called, **Then** it returns adapter prompt + tool schemas + JSON format instructions.
2. **Given** the built prompt is sent to NeuralGateway, **When** the model wants to call a tool, **Then** it responds with `{"tool_call": {"name": "...", "arguments": {...}}}`.
3. **Given** no tools in adapter, **When** `BuildSystemPrompt()` is called, **Then** the raw adapter prompt is returned unchanged.

## Tasks / Subtasks

- [ ] Task 1: Create `backend/internal/domain/orchestrator.go` (AC: #1, #2, #3)
  - [ ] Define `Orchestrator` struct holding `*Adapter` reference
  - [ ] Implement `NewOrchestrator(adapter *Adapter) *Orchestrator`
  - [ ] Implement `BuildSystemPrompt() string`:
    - Start with adapter's system prompt
    - If tools exist, append "## Available Tools" section
    - For each tool: name, description, parameters JSON schema
    - Append instruction: respond with ONLY `{"tool_call": {"name": "...", "arguments": {...}}}` when using a tool
    - Note which tools require confirmation
- [ ] Task 2: Write unit tests
  - [ ] With tools: prompt contains all tool names/schemas
  - [ ] Without tools: prompt unchanged
  - [ ] JSON format instruction present

## Dev Notes

- Create: `backend/internal/domain/orchestrator.go`
- The tool-call JSON format must match what `ParseToolCall()` (Story 2.2) expects
- Keep instructions concise — long system prompts eat context window
- Use `json.MarshalIndent` for readable tool schema in prompt

### References

- [Source: backend/internal/domain/adapter.go#Tools() method — from Story 1.1]

## Dev Agent Record

### Agent Model Used
### Debug Log References
### Completion Notes List
### File List
