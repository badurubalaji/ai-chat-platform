---
stepsCompleted: [1, 2, 3, 4]
inputDocuments: [architecture-decision-document.md, NeuralGate API Guide]
---

# ai-chat-platform - Epic Breakdown

## Overview

This document provides the complete epic and story breakdown for the Domain-Adapter-Driven Tool Orchestration feature, decomposing the architecture decisions into implementable stories.

## Requirements Inventory

### Functional Requirements

- FR-1: Load domain configuration from JSON file at startup
- FR-2: Use adapter's system prompt instead of hardcoded string
- FR-3: Pass adapter tools to providers that support native tool calling
- FR-4: Implement two-pass tool orchestration for non-tool-supporting providers
- FR-5: Execute tools via HTTP calls to internal services
- FR-6: Confirm destructive tool actions before execution
- FR-7: Fall back to adapter's NeuralGateway config when no user BYOK exists
- FR-8: Log all tool executions to the database
- FR-9: Display tool confirmation cards in the frontend
- FR-10: Send user confirmation/dismissal back to backend

### Non-Functional Requirements

- NFR-1: Adapter config loading must fail fast at startup on invalid JSON
- NFR-2: Tool execution HTTP calls must respect configured timeouts
- NFR-3: Two-pass orchestration must not break streaming UX
- NFR-4: Confirmation flow must be stateless-capable (conversation-scoped)
- NFR-5: Backward compatible — no adapter config = existing behavior

### FR Coverage Map

| FR | Epic | Stories |
|----|------|---------|
| FR-1 | Epic 1 | 1.1, 1.2 |
| FR-2 | Epic 1 | 1.3 |
| FR-3 | Epic 3 | 3.1 |
| FR-4 | Epic 2 | 2.1, 2.2 |
| FR-5 | Epic 2 | 2.3 |
| FR-6 | Epic 3 | 3.3 |
| FR-7 | Epic 5 | 5.1 |
| FR-8 | Epic 4 | 4.1, 4.2 |
| FR-9 | Epic 6 | 6.1, 6.2 |
| FR-10 | Epic 6 | 6.2, 6.3 |

## Epic List

1. **Epic 1: Domain Adapter & Config Loading** — Load adapter-config.json, wire into handler
2. **Epic 2: Tool Orchestrator & Executor** — Two-pass orchestration, tool parsing, HTTP execution
3. **Epic 3: Wire Tool Orchestration into handleChat()** — Modify chat handler for both paths + confirmation
4. **Epic 4: Database Migration & Audit Logging** — New tables, logging helpers
5. **Epic 5: NeuralGateway Default Provider Fallback** — Model routing with BYOK override
6. **Epic 6: Frontend Tool Confirmation Flow** — SSE events, action cards, confirm/dismiss

## Epic 1: Domain Adapter & Config Loading

Create the `domain` package with the `AdapterConfig` struct and loading logic so that the adapter is available to the handler at startup.

### Story 1.1: Define AdapterConfig struct and JSON loader

As a platform developer,
I want a Go struct that represents the adapter-config.json schema,
so that domain configuration can be loaded and validated at startup.

**Acceptance Criteria:**

**Given** a valid `adapter-config.json` at `ADAPTER_CONFIG_PATH`
**When** the application starts
**Then** the config is loaded into an immutable struct

**Given** `ADAPTER_CONFIG_PATH` is not set
**When** the application starts
**Then** the adapter is nil and platform operates in generic mode

**Given** `ADAPTER_CONFIG_PATH` points to invalid JSON
**When** the application starts
**Then** startup fails with a clear error message

### Story 1.2: Wire adapter loading into main.go and pass to handler

As a platform developer,
I want the adapter loaded at startup and injected into the handler,
so that handleChat() has access to domain tools and system prompt.

**Acceptance Criteria:**

**Given** a valid `ADAPTER_CONFIG_PATH`
**When** `main.go` runs
**Then** the adapter is loaded and passed to `NewHandler()`

**Given** no adapter config path
**When** `main.go` runs
**Then** `NewHandler()` receives nil adapter and server starts normally

### Story 1.3: Use adapter system prompt in handleChat()

As an end user of an embedded domain app,
I want the AI to respond with domain-specific knowledge,
so that the assistant understands my application context.

**Acceptance Criteria:**

**Given** an adapter with a custom system prompt
**When** a chat message is sent
**Then** the adapter's system prompt replaces "You are a helpful assistant."

**Given** no adapter (nil)
**When** a chat message is sent
**Then** the hardcoded fallback prompt is used

## Epic 2: Tool Orchestrator & Executor

Build the orchestration logic for two-pass tool calling and the HTTP-based tool executor.

### Story 2.1: Build system prompt injection for two-pass tool orchestration

As the platform,
I want to inject tool schemas into the system prompt for non-tool-supporting models,
so that NeuralGateway and Ollama can express tool-calling intent through structured text.

**Acceptance Criteria:**

**Given** adapter tools
**When** `BuildSystemPrompt()` is called
**Then** the system prompt includes tool schemas and JSON format instructions

**Given** no tools in adapter
**When** `BuildSystemPrompt()` is called
**Then** the raw adapter system prompt is returned unchanged

### Story 2.2: Build tool-call parser for two-pass responses

As the platform,
I want to parse tool-call intent from model text responses,
so that non-tool-supporting models can trigger tool execution.

**Acceptance Criteria:**

**Given** a response with JSON tool_call block
**When** `ParseToolCall()` is called
**Then** it returns a `*models.ToolCall` with name and arguments

**Given** plain conversational text
**When** `ParseToolCall()` is called
**Then** it returns nil

**Given** malformed JSON in tool_call block
**When** `ParseToolCall()` is called
**Then** it returns nil (graceful degradation)

**Given** JSON in markdown code fences
**When** `ParseToolCall()` is called
**Then** it still extracts the tool call correctly

### Story 2.3: Build HTTP-based tool executor

As the platform,
I want to execute tools by making HTTP calls to internal service endpoints,
so that AI tool calls translate into real actions in the domain application.

**Acceptance Criteria:**

**Given** a tool call matching an adapter tool
**When** `ExecuteTool()` is called
**Then** it makes an HTTP request with correct method, headers, and body

**Given** a tool URL with `{param}` placeholders
**When** arguments include that param
**Then** the URL is interpolated

**Given** a tool HTTP call times out
**When** execution completes
**Then** a structured error is returned

**Given** an unknown tool name
**When** `ExecuteTool()` is called
**Then** it returns a "tool not found" error

## Epic 3: Wire Tool Orchestration into handleChat()

Modify the chat handler to use the orchestrator and executor, supporting both native and two-pass paths.

### Story 3.1: Wire native tool calling path (Claude/OpenAI/Gemini)

As an end user with a BYOK Claude/OpenAI/Gemini key,
I want the AI to use domain tools through native function calling,
so that tool invocations work seamlessly within the streaming chat experience.

**Acceptance Criteria:**

**Given** a provider with `SupportsTools()=true` and adapter tools
**When** a chat message is sent
**Then** tools are passed to `SendMessageStream()` instead of nil

**Given** the provider returns a ToolCall chunk
**When** the tool does NOT require confirmation
**Then** the handler executes immediately and sends result back to model

**Given** the tool execution succeeds
**When** the result is sent to the model
**Then** the model generates a natural response and it streams to the client

### Story 3.2: Wire two-pass tool calling path (NeuralGateway/Ollama)

As an end user using the default NeuralGateway provider,
I want tools to work through prompt-engineered two-pass orchestration,
so that I get the same tool functionality as BYOK users.

**Acceptance Criteria:**

**Given** a provider with `SupportsTools()=false` and adapter tools
**When** a chat message is sent
**Then** `BuildSystemPrompt()` is used and tools are NOT passed to provider

**Given** the model's response contains a tool_call JSON block
**When** the full response is accumulated
**Then** the handler parses it, executes the tool, and makes a second model call

**Given** no tool_call in the response
**When** `ParseToolCall()` returns nil
**Then** the response streams to the client as-is

### Story 3.3: Implement confirmation flow for destructive tools

As an end user,
I want the AI to ask for my confirmation before executing destructive actions,
so that I can review and approve or reject operations.

**Acceptance Criteria:**

**Given** a tool with `requires_confirmation=true`
**When** the handler detects the tool call
**Then** it emits a `tool_confirm` SSE event with tool details

**Given** a `tool_confirm` was emitted
**When** the user sends approval via `/api/v1/ai/chat/confirm`
**Then** the tool is executed normally

**Given** a `tool_confirm` was emitted
**When** the user dismisses
**Then** the tool is NOT executed and model receives "User cancelled"

## Epic 4: Database Migration & Audit Logging

Add the `ai_tool_executions` table and wire audit logging.

### Story 4.1: Create database migration for ai_tool_executions

As a platform operator,
I want tool executions recorded in the database,
so that I have an audit trail of all AI-triggered actions.

**Acceptance Criteria:**

**Given** the migration is applied
**When** querying the database
**Then** `ai_tool_executions` table exists with all required columns and indexes

### Story 4.2: Wire audit logging into tool executor and store

As a platform operator,
I want every tool execution logged with timing, status, and confirmation info,
so that audit queries show who did what and when.

**Acceptance Criteria:**

**Given** a tool executed successfully
**When** execution completes
**Then** a row is inserted with status=success and duration

**Given** a tool execution fails
**When** the error occurs
**Then** a row is inserted with status=error

**Given** a user dismisses confirmation
**When** dismissal is processed
**Then** a row is inserted with status=cancelled

## Epic 5: NeuralGateway Default Provider Fallback

Implement model routing with BYOK override.

### Story 5.1: Implement provider resolution with BYOK override

As an end user who has not configured a BYOK key,
I want the AI chat to work out of the box using the deployment's default NeuralGateway,
so that I can use the chat immediately without provider setup.

**Acceptance Criteria:**

**Given** no user provider config and adapter has `default_provider`
**When** a chat message is sent
**Then** the adapter's NeuralGateway credentials are used

**Given** a user HAS configured a BYOK provider
**When** a chat message is sent
**Then** the user's BYOK provider is used

**Given** neither user BYOK nor adapter default exists
**When** a chat message is sent
**Then** "Provider not configured" error is returned

## Epic 6: Frontend Tool Confirmation Flow

Handle new SSE event types and display confirmation UI.

### Story 6.1: Handle tool_confirm SSE event in AiChatService

As the frontend service layer,
I want to parse `tool_confirm` SSE events from the backend,
so that the chat component can display confirmation UI.

**Acceptance Criteria:**

**Given** backend sends `event: tool_confirm`
**When** the service parses it
**Then** it emits an `AiStreamChunk` with `tool_confirm` field

### Story 6.2: Display confirmation card in chat component

As an end user,
I want to see a clear confirmation card when the AI wants to perform a destructive action,
so that I can review and approve or reject.

**Acceptance Criteria:**

**Given** a `tool_confirm` chunk is received
**When** the chat component processes it
**Then** a confirmation action card is displayed

**Given** the user clicks "Approve"
**When** the button is pressed
**Then** a confirmation POST is sent to the backend

**Given** the user clicks "Dismiss"
**When** the button is pressed
**Then** a dismissal POST is sent and the card is marked as dismissed

### Story 6.3: Update frontend to show tool execution lifecycle

As an end user,
I want to see the full lifecycle of a tool execution in the chat,
so that I understand what the AI is doing on my behalf.

**Acceptance Criteria:**

**Given** a tool is being executed
**When** `tool_call` SSE event arrives with status `executing`
**Then** a tool message with a spinner is shown

**Given** a tool has finished
**When** `tool_result` SSE event arrives
**Then** the spinner is replaced with a checkmark

**Given** a tool execution fails
**When** an error tool_result arrives
**Then** an error indicator is shown

## Dependency Graph

```
1.1 → 1.2 → [1.3, 2.1, 4.1]
              2.1 → 2.2 → 2.3 → [3.1, 3.2]
              4.1 → 4.2
              3.1 + 3.2 → 3.3 → 5.1 → [6.1, 6.2, 6.3]
```

## Sprint Allocation

| Sprint | Stories | Theme |
|--------|---------|-------|
| Sprint 1 | 1.1, 1.2, 1.3, 4.1 | Foundation |
| Sprint 2 | 2.1, 2.2, 2.3, 4.2 | Orchestration |
| Sprint 3 | 3.1, 3.2, 5.1 | Integration |
| Sprint 4 | 3.3, 6.1, 6.2, 6.3 | Confirmation + Frontend |
