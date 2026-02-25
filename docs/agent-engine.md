# Agent Engine & Dynamic Tool Registry

## Overview

The Agent Engine extends the ai-chat-platform with a **generic, multi-app tool-use framework**. Any application (EHR, CRM, ITSM, etc.) can register tools that the AI agent can invoke on behalf of users.

## Architecture

```
External Apps (EHR, CRM, etc.)
    |
    | REST API: Register/manage tools
    v
┌──────────────────┐
│  Tool Registry    │  DB-backed, multi-tenant, multi-app
│  (CRUD API)       │
└────────┬─────────┘
         |
    ┌────┴────────────┐
    │  Agent Engine     │  Multi-step execution loop
    │                   │
    │  1. Think         │  Send messages + tools to LLM
    │  2. Act           │  Execute tool call (with optional confirmation)
    │  3. Observe       │  Append tool result to context
    │  4. Repeat/Done   │  Loop or return final answer
    └────────┬─────────┘
             |
    ┌────────┴─────────┐
    │  Provider Layer    │  Claude / OpenAI / Gemini / Ollama / etc.
    └──────────────────┘
```

## Tool Registry

### Registration Model

```json
{
  "app_name": "ehr",
  "tool_name": "create_patient",
  "description": "Register a new patient in the EHR system",
  "parameters": {
    "type": "object",
    "properties": {
      "name":   { "type": "string", "description": "Patient full name" },
      "dob":    { "type": "string", "description": "Date of birth (YYYY-MM-DD)" },
      "gender": { "type": "string", "enum": ["male", "female", "other"] }
    },
    "required": ["name"]
  },
  "execution": {
    "type": "http",
    "method": "POST",
    "url": "https://ehr-api.internal/patients",
    "headers": { "X-API-Key": "secret" },
    "timeout_ms": 10000
  },
  "requires_confirmation": true
}
```

### REST API

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST   | `/api/v1/ai/registry/tools` | Register a tool |
| GET    | `/api/v1/ai/registry/tools` | List tools (filter: `?app_name=ehr`) |
| PUT    | `/api/v1/ai/registry/tools/:id` | Update a tool |
| DELETE | `/api/v1/ai/registry/tools/:id` | Delete a tool |

### Tool Resolution Priority

1. Static adapter tools (from `adapter-config.json`) — highest priority
2. Dynamic registry tools (from database) — merged at runtime
3. If duplicate name: adapter tool wins

## Agent Engine

### Execution Loop

```
User message → Agent.Run()
  │
  ├─ Iteration 1:
  │   ├─ LLM returns tool_call: "search_patients"
  │   ├─ Execute tool → get results
  │   └─ Append tool_result to messages
  │
  ├─ Iteration 2:
  │   ├─ LLM returns tool_call: "get_patient_history"
  │   ├─ Execute tool → get results
  │   └─ Append tool_result to messages
  │
  └─ Iteration 3:
      └─ LLM returns text response (no tool call) → DONE
```

### Safety

- **Max iterations**: Configurable (default: 5) to prevent infinite loops
- **Confirmation**: Tools can require user approval before execution
- **Audit**: Every tool execution is logged with arguments, result, duration
- **Timeout**: Per-tool configurable HTTP timeout

### SSE Event Flow (multi-step)

```
event: chunk        → streaming text (iteration 1 thinking)
event: tool_call    → { tool: "search_patients", status: "executing" }
event: tool_result  → { tool: "search_patients", status: "complete" }
event: chunk        → streaming text (iteration 2 thinking)
event: tool_call    → { tool: "get_patient_history", status: "executing" }
event: tool_result  → { tool: "get_patient_history", status: "complete" }
event: chunk        → streaming final answer
event: done         → { done: true, usage: {...} }
```

## Database Schema

### ai_tool_registry

| Column | Type | Description |
|--------|------|-------------|
| id | UUID | Primary key |
| tenant_id | VARCHAR(100) | Tenant isolation |
| app_name | VARCHAR(100) | Application identifier (e.g., "ehr", "crm") |
| tool_name | VARCHAR(100) | Unique tool name within app |
| description | TEXT | Tool description for LLM |
| parameters | JSONB | JSON Schema for tool parameters |
| execution_config | JSONB | HTTP execution config (url, method, headers, timeout) |
| requires_confirmation | BOOLEAN | Whether user must approve before execution |
| enabled | BOOLEAN | Toggle tool on/off without deleting |
| created_at | TIMESTAMPTZ | Creation timestamp |
| updated_at | TIMESTAMPTZ | Last update timestamp |

**Unique constraint**: `(tenant_id, app_name, tool_name)`

## Example Usage

### 1. Register EHR Tools

```bash
curl -X POST /api/v1/ai/registry/tools \
  -H "Content-Type: application/json" \
  -d '{
    "app_name": "ehr",
    "tool_name": "create_patient",
    "description": "Register a new patient",
    "parameters": { "type": "object", "properties": { "name": { "type": "string" } }, "required": ["name"] },
    "execution": { "type": "http", "method": "POST", "url": "https://ehr-api/patients" },
    "requires_confirmation": true
  }'
```

### 2. User Asks AI

```
User: "Register a new patient named John Doe, born January 15, 1990"
```

### 3. Agent Executes

```
AI → tool_call: create_patient({ name: "John Doe", dob: "1990-01-15" })
   → confirmation required → user approves
   → execute HTTP POST to EHR API
   → tool_result: { patient_id: "P-12345", status: "registered" }
AI → "Patient John Doe has been successfully registered with ID P-12345."
```

## Quick Start (Demo)

### Prerequisites
- PostgreSQL running with `ai_chat_db` database
- Go 1.21+

### Steps

```bash
# 1. Apply migrations (creates ai_tool_registry table)
cd backend
go run ./cmd/migrate/ up

# 2. Start mock EHR API (port 8085, with seeded patient data)
go run ./cmd/mock-ehr/ &

# 3. Start AI backend
PORT=8086 go run ./cmd/server/ &

# 4. Register 5 EHR tools with the agent
bash cmd/mock-ehr/register-tools.sh http://localhost:8086

# 5. Verify tools are registered
curl -s http://localhost:8086/api/v1/ai/registry/tools \
  -H "X-Tenant-ID: default-tenant" | python -m json.tool
```

### Available Mock EHR Tools

| Tool | Method | Confirmation | Description |
|------|--------|-------------|-------------|
| `create_patient` | POST | Yes | Register a new patient |
| `get_patient` | GET | No | Get patient by ID |
| `search_patients` | GET | No | Search patients by name |
| `get_patient_history` | GET | No | Get medical history |
| `list_patients` | GET | No | List all patients |

### Seeded Test Data

| Patient ID | Name | History Records |
|-----------|------|----------------|
| P-1001 | John Smith | 4 (visits, labs, prescription) |
| P-1002 | Sarah Johnson | 3 (visit, diagnosis, prescription) |
| P-1003 | Michael Brown | 0 |

### Test Queries

Once a provider is configured (BYOK API key via Settings UI):

- **Simple tool call**: "Show me all patients"
- **Search + lookup**: "Find patient John Smith"
- **Multi-step**: "Summarize John Smith's health status" (searches, gets history, summarizes)
- **With confirmation**: "Register a new patient named Jane Doe, born 1995-06-20"
- **Error handling**: "Get history for patient P-9999" (patient not found)

## Files Reference

| File | Purpose |
|------|---------|
| `backend/internal/domain/agent.go` | Multi-step agent execution loop |
| `backend/internal/domain/registry.go` | Tool registry (merges adapter + DB tools) |
| `backend/internal/domain/executor.go` | HTTP-based tool execution |
| `backend/internal/domain/adapter.go` | Static adapter config loading |
| `backend/internal/domain/orchestrator.go` | Two-pass tool calling for non-native providers |
| `backend/internal/domain/audit.go` | Tool execution audit logging |
| `backend/internal/api/handler.go` | HTTP handlers (chat, registry CRUD) |
| `backend/internal/store/store.go` | PostgreSQL data access layer |
| `backend/internal/models/models.go` | Data models (RegisteredTool, etc.) |
| `backend/db/migrations/000003_tool_registry.up.sql` | Tool registry DB schema |
| `backend/cmd/mock-ehr/main.go` | Mock EHR REST API for testing |
| `backend/cmd/mock-ehr/register-tools.sh` | Script to register demo EHR tools |
