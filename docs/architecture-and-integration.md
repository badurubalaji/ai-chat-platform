# AI Chat Platform — Architecture & Integration Documentation

## Table of Contents

1. [Platform Overview](#1-platform-overview)
2. [Architecture](#2-architecture)
3. [How the Backend Integrates](#3-how-the-backend-integrates)
4. [Backend Integration Approaches](#4-backend-integration-approaches)
5. [Configuring for EHR](#5-configuring-for-ehr)
6. [Configuring for School ERP](#6-configuring-for-school-erp)
7. [What's the Same vs What's Different](#7-whats-the-same-vs-whats-different)
8. [Provider Resolution Flow](#8-provider-resolution-flow)
9. [Tool Execution & Confirmation](#9-tool-execution--confirmation)
10. [Agent Engine & Dynamic Tool Registry](#10-agent-engine--dynamic-tool-registry)
11. [Deployment Checklist](#11-deployment-checklist)

---

## 1. Platform Overview

The AI Chat Platform consists of **two distributable artifacts**:

| Artifact | Type | Language | Distribution |
|----------|------|----------|-------------|
| `@mdp/ai-chat` | Angular library (npm package) | TypeScript | Private npm registry |
| `ai-chat-backend` | Server binary | Go | Docker image or binary |

**Key capabilities:**

- **Multi-Provider Support** — Claude, OpenAI, Gemini, Ollama, NeuralGateway
- **Streaming** — Server-Sent Events (SSE) for real-time AI responses
- **Tool Use** — AI can call your app's internal APIs (search patients, enroll students, etc.)
- **Agent Engine** — Multi-step tool chaining with dynamic tool registry
- **Dynamic Tool Registry** — REST API for apps to register tools at runtime
- **BYOK** — Users can bring their own API keys
- **Secure** — AES-256-GCM encryption for API keys, server-side credential management
- **Analytics** — Token usage tracking and visualization per user
- **Action Cards** — AI can propose actions the user can approve/dismiss

---

## 2. Architecture

### High-Level Architecture

```
┌──────────────────────────────────────────────────────────┐
│  ai-chat-platform (this repo)                            │
│                                                          │
│  frontend/projects/mdp-ai-chat/  ── npm publish ──┐     │
│  backend/                        ── docker build ──┐     │
└────────────────────────────────────────────────────┼─┼───┘
                                                     │ │
         ┌───────────────────────────────────────────┘ │
         ▼                                             ▼
  ┌─────────────────┐                   ┌─────────────────────┐
  │  Private npm     │                   │  Container Registry  │
  │  Registry        │                   │  (Docker Hub/GitLab) │
  └──────┬──────────┘                   └──────┬──────────────┘
         │ npm install                         │ docker pull
         ▼                                     ▼
  ┌────────────┐  ┌──────────────┐  ┌─────────────────┐
  │  EHR App   │  │ School ERP   │  │ Data Backup App │
  │  + backend │  │ + backend    │  │ + backend       │
  └────────────┘  └──────────────┘  └─────────────────┘
```

### Per-App Data Flow

```
┌─────────────────────────────────────────┐
│  App Frontend (Angular)                 │
│                                         │
│  provideAiChat({                        │
│    apiBaseUrl: 'https://app-ai:8080'    │  ◄── Points to THIS app's backend
│  })                                     │
│                                         │
│  <mdp-ai-sidebar>                       │
│    User types a question                │
│    │                                    │
│    │ POST /api/v1/ai/chat               │
│    ▼                                    │
└────┼────────────────────────────────────┘
     │  HTTP / SSE (over network)
     ▼
┌─────────────────────────────────────────┐
│  ai-chat-backend (Go)                   │
│  adapter-config.json loaded             │
│                                         │
│  1. Receives chat message               │
│  2. Adds domain system prompt           │
│  3. Sends to AI provider                │
│  4. AI decides to use a tool            │
│  5. Backend calls app's internal API    │
│  6. Streams response back via SSE       │
└─────────────────────────────────────────┘
```

---

## 3. How the Backend Integrates

The `@mdp/ai-chat` Angular library is **purely frontend UI** — it has no backend logic. It makes HTTP calls to whatever backend URL you configure via `provideAiChat({ apiBaseUrl })`.

**Why the backend is required:**

| Reason | Details |
|--------|---------|
| **Security** | API keys are encrypted (AES-256-GCM) and stored server-side. Never exposed to the browser. |
| **Streaming** | Acts as an SSE proxy between the frontend and AI provider for real-time token streaming. |
| **Tool Execution** | When AI decides to call a tool (e.g., "search patients"), the backend makes the HTTP call to your app's internal API. The browser never sees internal service URLs. |
| **Chat History** | Conversations and token usage are persisted in PostgreSQL. |
| **Provider Abstraction** | Supports Claude, OpenAI, Gemini, Ollama, NeuralGateway interchangeably. Users can switch without frontend changes. |

---

## 4. Backend Integration Approaches

### Option A: Separate Go Service (Current — Recommended)

```
Your App (Java/Spring)  ←→  ai-chat-backend (Go, port 8080)  ←→  AI Provider
```

| Pros | Cons |
|------|------|
| Already built and working | Extra container to manage |
| Same image for all apps | Needs its own DB |
| Clean isolation | Network hop for tool calls |
| Scales independently | — |

### Option B: Embedded Java Module (Port to Spring Boot)

```
Your App (Java/Spring)
  └── ai-chat-module (new Spring module)  ←→  AI Provider
```

| Pros | Cons |
|------|------|
| Single deployable | Requires rewriting ~2,000 lines Go → Java |
| Shares existing DB & auth | One language, but more coupling |
| Direct in-process service calls | AI bugs could affect your app |

### Option C: Hybrid — Spring Boot Starter Library

```xml
<dependency>
  <groupId>com.mdp</groupId>
  <artifactId>ai-chat-spring-boot-starter</artifactId>
  <version>1.0.0</version>
</dependency>
```

| Pros | Cons |
|------|------|
| No separate service | One-time porting effort |
| Reusable across Java apps | Only works for Java/Spring apps |
| Auto-configured via application.yml | — |

**Recommendation:** Use **Option A** (separate Go service) if AI chat is deployed across multiple apps. Consider **Option B/C** only if all apps are Java/Spring and you want a single deployable.

---

## 5. Configuring for EHR

### 5.1 Frontend — Install in EHR Angular Project

```bash
# One-time: configure private registry
echo "@mdp:registry=http://your-verdaccio-host:4873" >> .npmrc

# Install
npm install @mdp/ai-chat
```

**`src/app/app.config.ts`:**
```typescript
import { provideAiChat } from '@mdp/ai-chat';

export const appConfig: ApplicationConfig = {
  providers: [
    // ... existing providers
    provideAiChat({
      apiBaseUrl: 'http://ehr-ai:8080',
      authTokenFn: () => inject(AuthService).getAccessToken$(),
      appName: 'Smart Health Assistant',
      enableTools: true,
      enableActionCards: true,
    })
  ]
};
```

**Add sidebar to main layout:**
```typescript
import { AiSidebarComponent, AiProposedAction } from '@mdp/ai-chat';

@Component({
  imports: [RouterOutlet, AiSidebarComponent],
  template: `
    <div class="app-layout">
      <app-header />
      <div class="content-area">
        <app-nav-sidebar />
        <main><router-outlet /></main>
        <mdp-ai-sidebar
          [contextBadge]="currentPageContext"
          (actionRequested)="handleAiAction($event)">
        </mdp-ai-sidebar>
      </div>
    </div>
  `
})
export class MainLayoutComponent {
  currentPageContext = 'Patient Dashboard';

  handleAiAction(action: AiProposedAction): void {
    switch (action.action_type) {
      case 'view_patient':
        this.router.navigate(['/patients', action.params['patient_id']]);
        break;
      case 'open_scheduler':
        this.router.navigate(['/appointments/new'], { queryParams: action.params });
        break;
    }
  }
}
```

**Add settings page (optional — for BYOK):**
```typescript
import { AiSettingsComponent, AiUsageDashboardComponent } from '@mdp/ai-chat';

@Component({
  imports: [AiSettingsComponent, AiUsageDashboardComponent],
  template: `
    <h2>AI Provider Settings</h2>
    <p>Configure your own AI provider (optional). Default uses the hospital's in-house model.</p>
    <mdp-ai-settings />

    <h2>AI Usage Dashboard</h2>
    <mdp-ai-usage-dashboard />
  `
})
export class SettingsComponent {}
```

### 5.2 Backend — `adapter-config-ehr.json`

```json
{
  "domain": "ehr",
  "display_name": "Smart Health Assistant",
  "system_prompt": "You are a healthcare AI assistant integrated with the Electronic Health Records system. You help doctors and nurses with patient information, scheduling, and clinical documentation. Always prioritize patient safety. Never provide medical diagnoses — only assist with data retrieval and administrative tasks. Follow HIPAA guidelines strictly.",
  "default_provider": {
    "provider": "neuralgate",
    "model": "auto",
    "endpoint_url": "https://neuralgateway.hospital-infra.local",
    "client_id": "ehr-ai-client-abc123",
    "client_secret": "ehr-ai-secret-xyz789"
  },
  "tools": [
    {
      "name": "search_patients",
      "description": "Search for patients by name, MRN, or date of birth",
      "parameters": {
        "type": "object",
        "properties": {
          "query": { "type": "string", "description": "Patient name, MRN, or DOB" },
          "limit": { "type": "integer", "default": 10 }
        },
        "required": ["query"]
      },
      "execution": {
        "type": "http",
        "method": "GET",
        "url": "http://ehr-api:3000/api/patients/search",
        "timeout_ms": 5000
      }
    },
    {
      "name": "get_patient_vitals",
      "description": "Retrieve the latest vital signs for a patient",
      "parameters": {
        "type": "object",
        "properties": {
          "patient_id": { "type": "string" }
        },
        "required": ["patient_id"]
      },
      "execution": {
        "type": "http",
        "method": "GET",
        "url": "http://ehr-api:3000/api/patients/{patient_id}/vitals"
      }
    },
    {
      "name": "schedule_appointment",
      "description": "Schedule an appointment for a patient with a doctor",
      "parameters": {
        "type": "object",
        "properties": {
          "patient_id": { "type": "string" },
          "doctor_id": { "type": "string" },
          "date": { "type": "string", "format": "date" },
          "time": { "type": "string" },
          "reason": { "type": "string" }
        },
        "required": ["patient_id", "doctor_id", "date", "time"]
      },
      "requires_confirmation": true,
      "execution": {
        "type": "http",
        "method": "POST",
        "url": "http://ehr-api:3000/api/appointments"
      }
    },
    {
      "name": "get_lab_results",
      "description": "Retrieve lab test results for a patient",
      "parameters": {
        "type": "object",
        "properties": {
          "patient_id": { "type": "string" },
          "test_type": { "type": "string" }
        },
        "required": ["patient_id"]
      },
      "execution": {
        "type": "http",
        "method": "GET",
        "url": "http://ehr-api:3000/api/patients/{patient_id}/labs"
      }
    },
    {
      "name": "add_patient",
      "description": "Register a new patient in the EHR system",
      "parameters": {
        "type": "object",
        "properties": {
          "first_name": { "type": "string" },
          "last_name": { "type": "string" },
          "date_of_birth": { "type": "string", "format": "date" },
          "gender": { "type": "string", "enum": ["male", "female", "other"] },
          "phone": { "type": "string" },
          "email": { "type": "string" }
        },
        "required": ["first_name", "last_name", "date_of_birth"]
      },
      "requires_confirmation": true,
      "execution": {
        "type": "http",
        "method": "POST",
        "url": "http://ehr-api:3000/api/patients",
        "timeout_ms": 10000
      }
    }
  ]
}
```

### 5.3 Deploy EHR

**`docker-compose.ehr.yml`:**
```yaml
services:
  ehr-ai-backend:
    image: your-registry/ai-chat-backend:1.0.0
    environment:
      DATABASE_URL: postgres://ai_user:${DB_PASSWORD}@postgres:5432/ehr_ai_db?sslmode=disable
      AI_ENCRYPTION_KEY: ${AI_ENCRYPTION_KEY}
      ADAPTER_CONFIG_PATH: /config/adapter-config.json
      PORT: "8080"
    volumes:
      - ./adapter-config-ehr.json:/config/adapter-config.json:ro
    ports:
      - "8080:8080"
    depends_on:
      - postgres
    networks:
      - ehr-network

  postgres:
    image: postgres:16
    environment:
      POSTGRES_DB: ehr_ai_db
      POSTGRES_USER: ai_user
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - ehr-pgdata:/var/lib/postgresql/data
    networks:
      - ehr-network

networks:
  ehr-network:
    external: true   # Join existing EHR app network so tool URLs resolve

volumes:
  ehr-pgdata:
```

> **Important:** The `ehr-ai-backend` container must be on the same Docker network as `ehr-api` so tool execution URLs like `http://ehr-api:3000/api/patients/search` resolve correctly.

```bash
# Run migrations
psql $DATABASE_URL -f backend/db/migrations/000001_init_schema.up.sql
psql $DATABASE_URL -f backend/db/migrations/000002_tool_executions.up.sql

# Start
docker-compose -f docker-compose.ehr.yml up -d
```

---

## 6. Configuring for School ERP

### 6.1 Frontend — Install in School ERP Angular Project

```bash
echo "@mdp:registry=http://your-verdaccio-host:4873" >> .npmrc
npm install @mdp/ai-chat
```

**`src/app/app.config.ts`:**
```typescript
provideAiChat({
  apiBaseUrl: 'http://school-ai:8081',
  authTokenFn: () => inject(AuthService).getAccessToken$(),
  appName: 'School AI Assistant',
  enableTools: true,
  enableActionCards: true,
})
```

### 6.2 Backend — `adapter-config-school.json`

```json
{
  "domain": "school-erp",
  "display_name": "School AI Assistant",
  "system_prompt": "You are an AI assistant for the School ERP system. You help school administrators, teachers, and staff manage student records, attendance, grades, and scheduling. Follow FERPA guidelines — never disclose student information to unauthorized parties. Be helpful and concise.",
  "default_provider": {
    "provider": "neuralgate",
    "model": "auto",
    "endpoint_url": "https://neuralgateway.school-infra.local",
    "client_id": "school-erp-client-def456",
    "client_secret": "school-erp-secret-uvw321"
  },
  "tools": [
    {
      "name": "search_students",
      "description": "Search for students by name, student ID, or grade level",
      "parameters": {
        "type": "object",
        "properties": {
          "query": { "type": "string" },
          "grade": { "type": "string" }
        },
        "required": ["query"]
      },
      "execution": {
        "method": "GET",
        "url": "http://school-api:4000/api/students/search"
      }
    },
    {
      "name": "enroll_student",
      "description": "Enroll a new student into the school system",
      "parameters": {
        "type": "object",
        "properties": {
          "first_name": { "type": "string" },
          "last_name": { "type": "string" },
          "date_of_birth": { "type": "string", "format": "date" },
          "grade": { "type": "string" },
          "parent_name": { "type": "string" },
          "parent_phone": { "type": "string" },
          "parent_email": { "type": "string" }
        },
        "required": ["first_name", "last_name", "date_of_birth", "grade"]
      },
      "requires_confirmation": true,
      "execution": {
        "method": "POST",
        "url": "http://school-api:4000/api/students"
      }
    },
    {
      "name": "get_attendance",
      "description": "Get attendance records for a student or class",
      "parameters": {
        "type": "object",
        "properties": {
          "student_id": { "type": "string" },
          "class_id": { "type": "string" },
          "date_from": { "type": "string", "format": "date" },
          "date_to": { "type": "string", "format": "date" }
        }
      },
      "execution": {
        "method": "GET",
        "url": "http://school-api:4000/api/attendance"
      }
    },
    {
      "name": "record_grade",
      "description": "Record or update a student's grade for an assignment or exam",
      "parameters": {
        "type": "object",
        "properties": {
          "student_id": { "type": "string" },
          "subject": { "type": "string" },
          "assignment": { "type": "string" },
          "score": { "type": "number" },
          "max_score": { "type": "number" }
        },
        "required": ["student_id", "subject", "assignment", "score"]
      },
      "requires_confirmation": true,
      "execution": {
        "method": "POST",
        "url": "http://school-api:4000/api/grades"
      }
    },
    {
      "name": "get_class_schedule",
      "description": "Retrieve the class schedule for a teacher or student",
      "parameters": {
        "type": "object",
        "properties": {
          "teacher_id": { "type": "string" },
          "student_id": { "type": "string" },
          "day": { "type": "string" }
        }
      },
      "execution": {
        "method": "GET",
        "url": "http://school-api:4000/api/schedules"
      }
    }
  ]
}
```

### 6.3 Deploy School ERP

**`docker-compose.school.yml`:**
```yaml
services:
  school-ai-backend:
    image: your-registry/ai-chat-backend:1.0.0
    environment:
      DATABASE_URL: postgres://ai_user:${DB_PASSWORD}@postgres:5432/school_ai_db?sslmode=disable
      AI_ENCRYPTION_KEY: ${AI_ENCRYPTION_KEY}
      ADAPTER_CONFIG_PATH: /config/adapter-config.json
    volumes:
      - ./adapter-config-school.json:/config/adapter-config.json:ro
    ports:
      - "8081:8080"
    networks:
      - school-network

  postgres:
    image: postgres:16
    environment:
      POSTGRES_DB: school_ai_db
      POSTGRES_USER: ai_user
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - school-pgdata:/var/lib/postgresql/data
    networks:
      - school-network

networks:
  school-network:
    external: true

volumes:
  school-pgdata:
```

---

## 7. What's the Same vs What's Different

| Item | EHR | School ERP |
|------|-----|------------|
| **npm package** | `@mdp/ai-chat` ✅ same | `@mdp/ai-chat` ✅ same |
| **Docker image** | `ai-chat-backend:1.0.0` ✅ same | `ai-chat-backend:1.0.0` ✅ same |
| **`apiBaseUrl`** | `http://ehr-ai:8080` | `http://school-ai:8081` |
| **`appName`** | `Smart Health Assistant` | `School AI Assistant` |
| **System prompt** | Healthcare + HIPAA | Education + FERPA |
| **Tool URLs** | `http://ehr-api:3000/api/...` | `http://school-api:4000/api/...` |
| **NeuralGateway creds** | EHR project's credentials | School project's credentials |
| **PostgreSQL database** | `ehr_ai_db` | `school_ai_db` |

**In short — 3 things to configure per app:**

1. **Frontend** — `provideAiChat({ apiBaseUrl })` pointing to that app's backend
2. **Backend config** — `adapter-config.json` with domain-specific prompt, tools, and provider credentials
3. **Docker deploy** — Same image, different config file mounted

---

## 8. Provider Resolution Flow

```
User sends chat message
        │
        ▼
┌─ Does user have BYOK config? ─┐
│  (ai_provider_configs table)   │
│                                │
│  YES                    NO     │
│   │                      │     │
│   ▼                      ▼     │
│  Use user's           Use adapter's
│  Claude/OpenAI/        NeuralGateway
│  Gemini key            credentials
└───┴──────────────────────┴─────┘
        │
        ▼
   Send to AI provider
```

- If a user configures their own API key (via the `<mdp-ai-settings>` component), that key is used.
- Otherwise, the adapter config's `default_provider` (typically NeuralGateway) is used.
- Keys are encrypted with AES-256-GCM before storage.

---

## 9. Tool Execution & Confirmation

### Tool URL Interpolation

Tool execution URLs support `{param}` placeholders:

```json
{ "url": "http://ehr-api:3000/api/patients/{patient_id}/vitals" }
```

If AI calls with `{"patient_id": "P-12345"}`, the request goes to:
`GET http://ehr-api:3000/api/patients/P-12345/vitals`

- **GET:** Non-URL parameters become query parameters
- **POST/PUT/DELETE:** Parameters sent as JSON body
- `X-Tenant-ID` and `X-User-ID` headers are automatically injected

### Confirmation Flow

Tools with `"requires_confirmation": true` trigger an SSE `tool_confirm` event:

```
1. AI decides to call "schedule_appointment"
2. Backend sends SSE event: tool_confirm { tool, args }
3. Frontend shows Approve / Dismiss card to user
4. User clicks Approve
5. Frontend sends POST /api/v1/ai/chat/confirm
6. Backend executes the tool
7. Result streamed back via SSE
```

Timeout: 5 minutes. If user doesn't respond, the tool call is dismissed.

### Audit Logging

Every tool execution is logged to `ai_tool_executions` table with:
- Tenant/User ID
- Tool name and arguments
- Result and status (success/failure)
- Confirmation flag
- Duration

This supports HIPAA/FERPA compliance auditing.

---

## 10. Deployment Checklist

### Per Application (EHR, School ERP, etc.)

**NeuralGateway:**
- [ ] Create project in NeuralGateway
- [ ] Assign AI model to project
- [ ] Generate client credentials (`client_id` + `client_secret`)

**Infrastructure:**
- [ ] Create PostgreSQL database
- [ ] Run migrations (`000001_init_schema` + `000002_tool_executions`)
- [ ] Generate `AI_ENCRYPTION_KEY` (`openssl rand -hex 32`)

**Backend:**
- [ ] Create `adapter-config.json` with domain prompt, provider credentials, and tools
- [ ] Tool URLs point to application's internal API (same Docker network)
- [ ] Deploy `ai-chat-backend` container with `ADAPTER_CONFIG_PATH` mounted `:ro`
- [ ] Verify startup log: `Domain adapter loaded: <domain> (<display_name>) with N tools`

**Frontend:**
- [ ] Add `.npmrc` pointing to private registry
- [ ] `npm install @mdp/ai-chat`
- [ ] Configure `provideAiChat()` with backend URL and auth token function
- [ ] Add `<mdp-ai-sidebar>` to main layout
- [ ] Add `<mdp-ai-settings>` to settings page (optional, for BYOK)
- [ ] Wire `(actionRequested)` output event if needed

**Networking:**
- [ ] Backend can reach NeuralGateway endpoint URL
- [ ] Backend can reach tool execution URLs (internal app APIs)
- [ ] Frontend can reach backend (CORS configured if needed)

**Security:**
- [ ] Encryption key is NOT the dev default
- [ ] `adapter-config.json` mounted read-only (`:ro`)
- [ ] NeuralGateway credentials never exposed to frontend
- [ ] HTTPS enabled in production

---

## Configuration Reference

### `provideAiChat()` Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `apiBaseUrl` | `string` | — | Backend URL (**required**) |
| `authTokenFn` | `() => Observable<string>` | — | JWT token provider (**required**) |
| `appName` | `string` | `AI Assistant` | Display name in sidebar header |
| `appDescription` | `string` | — | Context sent to system prompt |
| `enableTools` | `boolean` | `true` | Enable tool use / function calling |
| `enableActionCards` | `boolean` | `true` | Enable AI-proposed action cards |
| `sidebarWidth` | `number` | `400` | Initial sidebar width (px) |
| `sidebarMinWidth` | `number` | `300` | Minimum sidebar width (px) |
| `sidebarMaxWidth` | `number` | `600` | Maximum sidebar width (px) |
| `markdownOptions` | `MarkdownRenderOptions` | — | Syntax highlighting config |

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `DATABASE_URL` | PostgreSQL connection string | Dev default | Yes |
| `AI_ENCRYPTION_KEY` | 32-byte hex key for AES-256-GCM | Dev default (insecure) | Yes (prod) |
| `PORT` | Backend listen port | `8080` | No |
| `ADAPTER_CONFIG_PATH` | Path to `adapter-config.json` | *(empty = generic mode)* | Per-app |

### Exported Components & Services

| Export | Type | Purpose |
|--------|------|---------|
| `AiSidebarComponent` | Component | Main chat sidebar panel |
| `AiSettingsComponent` | Component | BYOK provider settings form |
| `AiUsageDashboardComponent` | Component | Token usage charts & stats |
| `AiActionCardComponent` | Component | Actionable card proposed by AI |
| `AiEmptyStateComponent` | Component | Empty state placeholder |
| `AiChatService` | Service | Chat messaging & streaming |
| `AiProviderConfigService` | Service | Provider CRUD operations |
| `AiUsageService` | Service | Token usage data |
| `AiContextService` | Service | Page context injection |
| `MarkdownRenderPipe` | Pipe | Markdown → HTML rendering |
| `provideAiChat()` | Function | Angular provider factory |

---

## 10. Agent Engine & Dynamic Tool Registry

### Overview

The Agent Engine enables **multi-step tool-use conversations**. Instead of a single tool call per message, the AI can chain multiple tool calls to complete complex tasks.

There are two ways to provide tools to the AI:

| Method | Source | When to use |
|--------|--------|-------------|
| **Static (adapter-config.json)** | JSON file at startup | Fixed tools per deployment |
| **Dynamic (Tool Registry API)** | Database via REST API | Apps register tools at runtime |

Both sources are merged automatically. Static adapter tools take priority over registry tools with the same name.

### Agent Execution Flow

```
User: "Summarize John Smith's health status"
  |
  v
[Agent Engine - Iteration 1]
  AI decides: search_patients({ q: "John Smith" })
  -> Execute tool -> Returns patient P-1001
  -> Append result to context
  |
[Agent Engine - Iteration 2]
  AI decides: get_patient_history({ patient_id: "P-1001" })
  -> Execute tool -> Returns 4 history records
  -> Append result to context
  |
[Agent Engine - Iteration 3]
  AI generates final summary (no tool call)
  -> Stream response to user
  -> DONE
```

### Dynamic Tool Registry API

Register tools at runtime without restarting the backend:

```bash
# Register a tool
POST /api/v1/ai/registry/tools
Content-Type: application/json
X-Tenant-ID: my-tenant

{
  "app_name": "ehr",
  "tool_name": "create_patient",
  "description": "Register a new patient in the EHR system",
  "parameters": {
    "type": "object",
    "properties": {
      "name": { "type": "string", "description": "Patient full name" },
      "dob":  { "type": "string", "description": "Date of birth (YYYY-MM-DD)" }
    },
    "required": ["name"]
  },
  "execution": {
    "type": "http",
    "method": "POST",
    "url": "https://your-ehr-api/patients",
    "headers": { "X-API-Key": "secret" },
    "timeout_ms": 10000
  },
  "requires_confirmation": true
}

# List registered tools
GET /api/v1/ai/registry/tools?app_name=ehr

# Update a tool
PUT /api/v1/ai/registry/tools/{id}

# Delete a tool
DELETE /api/v1/ai/registry/tools/{id}
```

### Tool Registration Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `app_name` | string | Yes | Application identifier (e.g., "ehr", "crm") |
| `tool_name` | string | Yes | Unique tool name within the app |
| `description` | string | Yes | What the tool does (shown to AI) |
| `parameters` | JSON Schema | No | Tool input parameters |
| `execution` | object | Yes | HTTP execution config |
| `execution.type` | string | Yes | Always "http" |
| `execution.method` | string | Yes | GET, POST, PUT, DELETE |
| `execution.url` | string | Yes | URL with `{param}` placeholders |
| `execution.headers` | object | No | Custom HTTP headers |
| `execution.timeout_ms` | int | No | Timeout in ms (default: 5000) |
| `requires_confirmation` | bool | No | Whether user must approve before execution |

### SSE Events (Multi-Step)

During a multi-step agent execution, the frontend receives these SSE events:

```
event: chunk        data: {"content":"Let me search..."}
event: tool_call    data: {"tool":"search_patients","status":"executing"}
event: tool_result  data: {"tool":"search_patients","status":"complete"}
event: chunk        data: {"content":"Now let me get the history..."}
event: tool_call    data: {"tool":"get_patient_history","status":"executing"}
event: tool_result  data: {"tool":"get_patient_history","status":"complete"}
event: chunk        data: {"content":"Based on the records..."}
event: done         data: {"done":true,"conversation_id":"...","usage":{...}}
```

### Demo: Mock EHR Integration

A mock EHR service is included for testing:

```bash
# 1. Apply database migration
cd backend && go run ./cmd/migrate/ up

# 2. Start mock EHR API (port 8085)
go run ./cmd/mock-ehr/

# 3. Start backend (port 8080 or custom)
PORT=8086 go run ./cmd/server/

# 4. Register EHR tools
bash cmd/mock-ehr/register-tools.sh http://localhost:8086

# 5. Configure a provider (via Settings UI or API)
# Then try these queries in chat:
#   "Show me all patients"
#   "Find patient John Smith"
#   "Register a new patient named Jane Doe, born 1995-06-20"
#   "Summarize patient history for P-1001"
```

The mock EHR includes seeded data: 3 patients with medical history (visits, labs, prescriptions, diagnoses).

### Integrating Your Own App

To connect any application to the agent:

1. **Expose REST APIs** from your app for the actions the AI should perform
2. **Register tools** via the registry API, describing each action
3. **Configure confirmation** for sensitive operations (create, update, delete)
4. The AI will automatically discover and use registered tools based on user intent

No code changes needed in the ai-chat-platform itself.
