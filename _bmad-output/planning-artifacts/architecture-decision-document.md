---
stepsCompleted: [1, 2, 3, 4, 5, 6]
inputDocuments: [NeuralGate API Integration Guide, handler.go, provider.go, models.go, registry.go]
workflowType: 'architecture'
project_name: 'ai-chat-platform'
user_name: 'Ashulabs'
date: '2026-02-21'
---

# Architecture Decision Document: Domain-Adapter-Driven Tool Orchestration

## 1. Executive Summary

This document captures the architectural decisions for transforming the AI Chat Platform from a generic chat relay into an embeddable, domain-aware AI assistant with tool execution capabilities. The platform will be deployable into in-house applications (EHR, School ERP, Data Backup & Recovery) using a single config-driven adapter pattern.

## 2. Current State

The AI Chat Platform has a Go backend (`handler.go`) that:
- Supports 6 providers: Claude, OpenAI, Gemini, Ollama, Generic, NeuralGate
- Implements SSE streaming via `SendMessageStream()`
- Passes `tools=nil` to all providers (line 190 of handler.go)
- Uses hardcoded system prompt: `"You are a helpful assistant."`
- Has a `ToolRegistry` with 6 mock tools that is **never imported or used** by the handler
- Stores conversations and messages in PostgreSQL with JSONB metadata

The Angular frontend (`mdp-ai-chat` library):
- Parses SSE events including `tool_call` and `tool_result` types (already defined but never emitted)
- Has `AiActionCardComponent` for proposed action display (exists but untriggered)
- Uses Angular Signals for state management

## 3. Architectural Decisions

### ADR-1: Single Config-Driven Domain Adapter

**Decision**: One universal `Adapter` struct loads `adapter-config.json` at startup. No separate Go packages per domain.

**Context**: The platform must support EHR, School ERP, and Backup & Recovery deployments. All target apps are in-house.

**Rationale**:
- Same binary deploys everywhere with different config files
- Adding a new domain = writing a new JSON file, not new Go code
- Zero runtime complexity (no registry, no plugin discovery)
- Config is immutable at runtime (no admin UI, no modification endpoints)

**Consequences**:
- `internal/domain/adapter.go` is the single adapter implementation
- `ADAPTER_CONFIG_PATH` env var points to the deployment config
- If no config path is set, platform operates in generic mode (backward compatible)

### ADR-2: Dual-Path Tool Calling Strategy

**Decision**: Two strategies based on provider capability — native function calling for Claude/OpenAI/Gemini, two-pass prompt engineering for NeuralGateway/Ollama.

**Context**: NeuralGateway's Chat API has no `tools` parameter. It returns plain text. Claude/OpenAI/Gemini have native function calling.

**Native Path** (providers where `SupportsTools()=true`):
```
User msg + tools → Provider → ToolCall chunk → Execute → Result → Provider (2nd call) → Response
```

**Two-Pass Path** (providers where `SupportsTools()=false`):
```
User msg + tool schemas in system prompt → Provider → Parse JSON from response → Execute → Result in 2nd prompt → Provider → Response
```

**Rationale**:
- Native function calling is more reliable and should be used when available
- Two-pass is a proven fallback for models without native tool support
- The `SupportsTools()` method already exists on all providers
- Both paths share the same `Adapter.Execute()` for tool execution

**Consequences**:
- `orchestrator.go` handles `BuildSystemPrompt()` (injects tool schemas) and `ParseToolCall()` (extracts JSON)
- `handler.go` branches on `provider.SupportsTools()` to choose the path
- Tool definitions in adapter config must be compatible with both paths

### ADR-3: Confirmation Flow via SSE + Separate Endpoint

**Decision**: Destructive tools emit a `tool_confirm` SSE event on the existing stream. User confirmation is sent via a separate `POST /api/v1/ai/chat/confirm` endpoint. The SSE connection remains open while waiting.

**Context**: Tools like `add_patient`, `restart_agent`, `trigger_restore` must not execute without explicit user approval.

**Flow**:
```
SSE stream: ... → event:tool_confirm (with confirmation_id, tool, params) → [WAITING]
User clicks Approve → POST /confirm {confirmation_id, approved: true}
SSE stream: → event:tool_call (executing) → event:tool_result → event:chunk (response) → event:done
```

**Rationale**:
- Keeps the SSE connection alive (no reconnection needed)
- Separate endpoint is clean for the frontend (action card buttons POST directly)
- In-memory pending confirmations with 5-minute TTL prevent stale state
- The handler goroutine blocks on a buffered channel, released by the confirm endpoint

**Rejected Alternative**: WebSocket-based bidirectional confirmation — adds complexity, platform is SSE-based.

**Consequences**:
- `PendingConfirmation` map in Handler with channel-based signaling
- New route in `ServeHTTP()` for `/confirm`
- Frontend needs `sendConfirmation()` method in `AiChatService`

### ADR-4: Provider Fallback — Adapter Default Overridden by User BYOK

**Decision**: If no user BYOK provider config exists in the database, fall back to the adapter's default NeuralGateway credentials. User BYOK always takes priority.

**Context**: The adapter config contains NeuralGateway `client_id` and `client_secret` set at deployment. End users can optionally configure their own Claude/GPT/Gemini keys.

**Resolution order**:
1. Check `ai_provider_configs` for tenant — if row exists, use user's provider (existing behavior)
2. If no row and adapter has `default_provider` — construct NeuralGateway credentials from adapter config
3. If neither — return "Provider not configured" error

**Rationale**:
- Users get a working AI assistant out of the box (no setup required)
- Power users can bring their own keys for Claude/GPT
- NeuralGateway credentials are never exposed to users
- The `neuralgate` provider expects `client_id:client_secret` as apiKey — adapter constructs this format

**Consequences**:
- `resolveProvider()` method extracted from `handleChat()` to encapsulate this logic
- `handleConfig()` GET returns `is_default: true` when using adapter fallback

### ADR-5: HTTP-Based Tool Execution

**Decision**: Tools execute by making HTTP calls to internal service endpoints defined in the adapter config.

**Context**: All target applications (EHR, School ERP, Backup) are in-house apps on the same infrastructure.

**Config example**:
```json
{
  "name": "add_patient",
  "execution": {
    "type": "http",
    "method": "POST",
    "url": "http://ehr-service.internal:3000/api/patients",
    "headers": {"X-Internal-Auth": "shared-secret"},
    "timeout_ms": 5000
  }
}
```

**Rationale**:
- HTTP is universal — works with any internal service regardless of language/framework
- URL path interpolation (`{patient_id}`) handles RESTful endpoints
- Configurable timeouts per tool prevent slow services from blocking chat
- Headers in config allow service-to-service auth without code changes

**Consequences**:
- `executor.go` implements HTTP client with per-tool timeout and URL interpolation
- No gRPC or direct function calls in V1 (can be extended later)

### ADR-6: Audit Logging in Dedicated Table

**Decision**: All tool executions are logged to `ai_tool_executions` with full details (arguments, result, status, timing).

**Context**: EHR (HIPAA), School ERP (FERPA), and Backup systems all require audit trails.

**Table**: `ai_tool_executions` with `tenant_id`, `user_id`, `conversation_id`, `tool_name`, `arguments` (JSONB), `result` (JSONB), `status`, `confirmed_by_user`, `execution_duration_ms`, `created_at`.

**Rationale**:
- Separate from `ai_messages` because tool executions are first-class audit entities
- JSONB for arguments/result allows flexible querying
- Status tracks the full lifecycle: pending → success/error/cancelled
- Duration tracking enables performance monitoring

## 4. adapter-config.json Schema

```json
{
  "domain": "string (required) — e.g. 'ehr', 'school-erp', 'backup'",
  "display_name": "string (required) — Human-readable name",
  "system_prompt": "string (required) — Domain-specific AI instructions",
  "default_provider": {
    "provider": "neuralgate",
    "model": "string — model name or 'auto'",
    "endpoint_url": "string — NeuralGateway URL",
    "client_id": "string — NG client ID",
    "client_secret": "string — NG client secret"
  },
  "tools": [
    {
      "name": "string (required)",
      "description": "string (required) — shown to model",
      "parameters": "object (required) — JSON Schema",
      "required_role": "string — 'admin', 'operator', 'viewer'",
      "requires_confirmation": "boolean — true for destructive ops",
      "execution": {
        "type": "http",
        "method": "string — GET, POST, PUT, DELETE",
        "url": "string — internal service URL with {param} placeholders",
        "headers": "object — key-value pairs",
        "timeout_ms": "integer — default 5000"
      }
    }
  ]
}
```

## 5. Component Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                    AI CHAT PLATFORM                              │
│                                                                  │
│  Startup:                                                        │
│    main.go → config.Load() → domain.LoadAdapter(path) → Handler │
│                                                                  │
│  Runtime (handleChat):                                           │
│                                                                  │
│  ┌──────────────┐    ┌────────────────┐    ┌──────────────────┐  │
│  │ Provider      │    │ Orchestrator   │    │ Adapter          │  │
│  │ Resolver      │    │                │    │ (read-only)      │  │
│  │               │    │ BuildSysPrompt │    │                  │  │
│  │ BYOK config?  │    │ ParseToolCall  │    │ SystemPrompt()   │  │
│  │   → use it    │    │                │    │ Tools()          │  │
│  │ No config?    │    └───────┬────────┘    │ ToolByName()     │  │
│  │   → adapter NG│            │             │ DefaultProvider()│  │
│  └──────┬───────┘    ┌───────▼────────┐    └──────────────────┘  │
│         │            │ Executor       │                           │
│         │            │                │    ┌──────────────────┐   │
│         │            │ HTTP calls to  │    │ Audit Logger     │   │
│         │            │ internal svc   │    │                  │   │
│         │            │ URL interp.    │    │ ai_tool_execs DB │   │
│         │            └────────────────┘    └──────────────────┘   │
│         │                                                         │
│  ┌──────▼──────────────────────────────────────────────────────┐  │
│  │ Providers: Claude | OpenAI | Gemini | NeuralGate | Ollama   │  │
│  └─────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────┘
```

## 6. Security Considerations

- **NeuralGateway credentials**: Stored in adapter-config.json, loaded at startup, never exposed via API
- **Tool endpoints**: Internal service URLs with optional `X-Internal-Auth` headers
- **API key encryption**: Existing AES-256-GCM encryption continues for BYOK keys in database
- **Audit trail**: Every tool execution logged with tenant, user, tool, arguments, and result
- **Confirmation gates**: Destructive operations require explicit user approval before execution
- **CORS**: Must be restricted per-deployment (currently `*` — needs tightening in production)

## 7. Backward Compatibility

- If `ADAPTER_CONFIG_PATH` is not set, the platform operates exactly as before (no tools, generic prompt)
- Existing `ai_provider_configs` table is unchanged — BYOK flow works as-is
- New `ai_tool_executions` table is additive (no schema changes to existing tables)
- Frontend SSE parsing already handles unknown event types gracefully
