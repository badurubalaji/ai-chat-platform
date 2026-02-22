# AI Chat Platform — Integration Guide

A multi-provider AI chat backend with SSE streaming. Integrate it into **any app** (Angular, React, Vue, Flutter, iOS, Android, CLI, Python scripts, etc.) using standard HTTP requests.

---

## Quick Start

### 1. Start the Backend

```bash
# Set environment variables
export DATABASE_URL="postgres://ai_user:ai_password@localhost:5432/ai_chat_db?sslmode=disable"
export AI_ENCRYPTION_KEY="0123456789abcdef0123456789abcdef"  # 32-byte hex key

# Run
./start.sh
# Backend: http://localhost:8080
# Frontend Demo: http://localhost:4200
```

---

## API Reference

**Base URL:** `http://localhost:8080`

All endpoints accept/return JSON. CORS is enabled for all origins.

---

### 🔌 List Available Providers

```http
GET /api/v1/ai/providers
```

**Response:**
```json
[
  { "id": "openai",  "name": "OpenAI",          "requires_endpoint": false },
  { "id": "claude",  "name": "Anthropic Claude", "requires_endpoint": false },
  { "id": "gemini",  "name": "Google Gemini",    "requires_endpoint": false },
  { "id": "ollama",  "name": "Ollama (Local)",   "requires_endpoint": true, "default_endpoint": "http://localhost:11434" },
  { "id": "generic", "name": "Generic OpenAI",   "requires_endpoint": true }
]
```

---

### ⚙️ Configure Provider

**Save configuration:**
```http
POST /api/v1/ai/config
Content-Type: application/json

{
  "provider": "openai",
  "model": "gpt-4o",
  "apiKey": "sk-xxxxxxxxxxxxxxxx",
  "endpoint_url": "",
  "enabled": true,
  "settings": {
    "temperature": 0.7,
    "max_tokens": 4096,
    "system_prompt_prefix": ""
  }
}
```

**Get current config:**
```http
GET /api/v1/ai/config
```
Returns config without the API key — includes `"api_key_set": true/false`.

**Delete config:**
```http
DELETE /api/v1/ai/config
```

---

### 🧪 Test Connection

```http
POST /api/v1/ai/config/test
Content-Type: application/json

{
  "provider": "openai",
  "apiKey": "sk-xxxxxxxxxxxxxxxx",
  "model": "gpt-4o",
  "endpoint_url": ""
}
```

**Response (always 200):**
```json
{ "status": "success" }
// or
{ "status": "error", "message": "invalid API key" }
```

---

### 📋 List Models

```http
POST /api/v1/ai/config/models
Content-Type: application/json

{
  "provider": "openai",
  "apiKey": "sk-xxxxxxxxxxxxxxxx",
  "endpoint_url": ""
}
```

**Response:**
```json
["gpt-4o", "gpt-4o-mini", "gpt-4-turbo", "gpt-3.5-turbo"]
```

---

### 💬 Chat (SSE Streaming)

This is the core endpoint. It returns **Server-Sent Events (SSE)**.

```http
POST /api/v1/ai/chat
Content-Type: application/json

{
  "message": "What is nuclear fusion?",
  "conversation_id": null,
  "context": { "page": "/dashboard", "role": "admin" }
}
```

- `conversation_id`: Pass `null` for new conversations. The response will include the created ID.
- `context`: Optional metadata your app can send (current page, user role, etc.)

**SSE Response Stream:**

```
event: chunk
data: {"content":"Nuclear"}

event: chunk
data: {"content":" fusion is"}

event: chunk
data: {"content":" the process"}

event: done
data: {"content":"","done":true,"conversation_id":"48c2ca40-...","usage":{"input_tokens":5,"output_tokens":150}}
```

**Error event (if API call fails):**
```
event: error
data: OpenAI API error (status 429): rate limit exceeded
```

---

### 📂 Conversations

**List all conversations:**
```http
GET /api/v1/ai/conversations
```

**Get conversation with messages:**
```http
GET /api/v1/ai/conversations/{id}
```

**Create conversation:**
```http
POST /api/v1/ai/conversations
Content-Type: application/json

{ "title": "Project Discussion" }
```

**Delete conversation:**
```http
DELETE /api/v1/ai/conversations/{id}
```

---

### 📊 Usage Statistics

```http
GET /api/v1/ai/usage?days=30
```

**Response:**
```json
{
  "total_input_tokens": 15234,
  "total_output_tokens": 42567,
  "conversation_count": 45,
  "daily": [
    { "date": "2026-02-13", "input_tokens": 500, "output_tokens": 1200, "request_count": 8 }
  ]
}
```

---

## Integration Examples

### JavaScript / TypeScript (Any Framework)

```javascript
async function chat(message, conversationId = null) {
  const response = await fetch('http://localhost:8080/api/v1/ai/chat', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ message, conversation_id: conversationId })
  });

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let fullText = '';
  let currentEvent = 'chunk';

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;

    const text = decoder.decode(value, { stream: true });
    for (const line of text.split('\n')) {
      if (line.startsWith('event: ')) {
        currentEvent = line.slice(7).trim();
      } else if (line.startsWith('data: ')) {
        const data = line.slice(6);
        if (currentEvent === 'chunk') {
          const parsed = JSON.parse(data);
          fullText += parsed.content;
          console.log('Streaming:', parsed.content);  // Update UI here
        } else if (currentEvent === 'done') {
          const parsed = JSON.parse(data);
          console.log('Done! Conversation:', parsed.conversation_id);
          return { text: fullText, conversationId: parsed.conversation_id };
        } else if (currentEvent === 'error') {
          throw new Error(data);
        }
      }
    }
  }
}

// Usage
const result = await chat('What is nuclear fusion?');
// Continue the conversation
const followUp = await chat('Tell me more', result.conversationId);
```

---

### Python

```python
import requests
import json

BASE_URL = "http://localhost:8080"

# Configure provider
requests.post(f"{BASE_URL}/api/v1/ai/config", json={
    "provider": "openai",
    "model": "gpt-4o",
    "apiKey": "sk-...",
    "enabled": True
})

# Chat with streaming
def chat(message, conversation_id=None):
    response = requests.post(
        f"{BASE_URL}/api/v1/ai/chat",
        json={"message": message, "conversation_id": conversation_id},
        stream=True
    )

    full_text = ""
    current_event = "chunk"

    for line in response.iter_lines(decode_unicode=True):
        if line.startswith("event: "):
            current_event = line[7:].strip()
        elif line.startswith("data: "):
            data = line[6:]
            if current_event == "chunk":
                parsed = json.loads(data)
                full_text += parsed["content"]
                print(parsed["content"], end="", flush=True)
            elif current_event == "done":
                parsed = json.loads(data)
                print()  # newline
                return full_text, parsed.get("conversation_id")
            elif current_event == "error":
                raise Exception(data)

# Usage
text, convo_id = chat("What is quantum computing?")
text2, _ = chat("Explain it simply", convo_id)
```

---

### cURL

```bash
# Configure
curl -X POST http://localhost:8080/api/v1/ai/config \
  -H "Content-Type: application/json" \
  -d '{"provider":"openai","model":"gpt-4o","apiKey":"sk-...","enabled":true}'

# Chat (streaming)
curl -N http://localhost:8080/api/v1/ai/chat \
  -H "Content-Type: application/json" \
  -d '{"message":"Hello!","conversation_id":null}'

# List conversations
curl http://localhost:8080/api/v1/ai/conversations

# Get usage
curl "http://localhost:8080/api/v1/ai/usage?days=7"
```

---

### Angular (Using the mdp-ai-chat Library)

The frontend library `mdp-ai-chat` provides ready-made components:

```typescript
// app.config.ts
import { provideAiChat } from 'mdp-ai-chat';

export const appConfig = {
  providers: [
    provideAiChat({
      apiBaseUrl: 'http://localhost:8080',
      authTokenFn: () => of('your-auth-token')
    })
  ]
};
```

```html
<!-- Drop-in chat sidebar -->
<mdp-ai-sidebar></mdp-ai-sidebar>

<!-- Settings page -->
<mdp-ai-settings></mdp-ai-settings>

<!-- Usage dashboard -->
<mdp-ai-usage-dashboard></mdp-ai-usage-dashboard>
```

**Exported Components:**
| Component | Description |
|-----------|-------------|
| `AiSidebarComponent` | Full chat sidebar with conversation history |
| `AiSettingsComponent` | Provider config form with test connection |
| `AiUsageDashboardComponent` | Token usage charts and stats |
| `AiActionCardComponent` | Proposed action cards from AI responses |

**Exported Services:**
| Service | Description |
|---------|-------------|
| `AiChatService` | `sendMessage()` returns `Observable<AiStreamChunk>` |
| `AiProviderConfigService` | `saveConfig()`, `testConnection()`, `getModels()` |
| `AiUsageService` | `getUsageStats()` |
| `AiContextService` | Set/get page context sent with chat messages |

---

## Provider-Specific Setup

### OpenAI
- **API Key:** Get from [platform.openai.com/api-keys](https://platform.openai.com/api-keys)
- **Models:** `gpt-4o`, `gpt-4o-mini`, `o1`, `o3-mini`
- **Endpoint:** Not required (uses `https://api.openai.com/v1`)

### Anthropic Claude
- **API Key:** Get from [console.anthropic.com](https://console.anthropic.com)
- **Models:** `claude-sonnet-4-5-20250929`, `claude-opus-4-6`, `claude-haiku-4-5-20251001`
- **Endpoint:** Not required (uses `https://api.anthropic.com`)

### Google Gemini
- **API Key:** Get from [aistudio.google.com](https://aistudio.google.com/apikey)
- **Models:** `gemini-2.0-flash`, `gemini-2.0-pro`
- **Endpoint:** Not required (uses `https://generativelanguage.googleapis.com`)

### Ollama (Local)
- **API Key:** Not required
- **Endpoint:** `http://localhost:11434` (default)
- **Models:** Auto-detected from your installed models
- **Setup:** Install from [ollama.com](https://ollama.com), then `ollama pull llama3`

### Generic OpenAI-Compatible
- **Works with:** LM Studio, vLLM, Together AI, Groq, Fireworks, etc.
- **Endpoint:** Required — must point to an OpenAI-compatible API base URL
- **Models:** Auto-detected from the `/models` endpoint

---

## Architecture

```
┌─────────────────┐    HTTP/SSE     ┌──────────────────┐
│   Your App      │ ──────────────► │  Go Backend      │
│  (any language)  │                │  :8080            │
└─────────────────┘                │                  │
                                   │  ┌─────────────┐ │     ┌──────────────┐
                                   │  │ Router      │ │     │ PostgreSQL   │
                                   │  │ (handler.go)│─┼────►│ (messages,   │
                                   │  └──────┬──────┘ │     │  configs,    │
                                   │         │        │     │  usage)      │
                                   │  ┌──────▼──────┐ │     └──────────────┘
                                   │  │ Provider    │ │
                                   │  │ Registry    │ │
                                   │  └──────┬──────┘ │
                                   └─────────┼────────┘
                                             │
                          ┌──────────────────┼──────────────────┐
                          │                  │                  │
                   ┌──────▼──────┐   ┌──────▼──────┐   ┌──────▼──────┐
                   │   OpenAI    │   │   Claude    │   │   Gemini    │
                   │   Ollama    │   │   Generic   │   │             │
                   └─────────────┘   └─────────────┘   └─────────────┘
```

---

## SSE Event Types

| Event | Data Format | Description |
|-------|-------------|-------------|
| `chunk` | `{"content": "text"}` | Partial text from the AI model |
| `done` | `{"done": true, "conversation_id": "uuid", "usage": {...}}` | Stream complete |
| `error` | Plain text error message | API error occurred |
| `tool_call` | `{"tool": "name", "status": "executing"}` | Tool being called (future) |
| `tool_result` | `{"tool": "name", "status": "complete"}` | Tool finished (future) |

---

## Security Notes

- API keys are encrypted at rest using **AES-256-GCM**
- Keys are never returned in API responses (`api_key_set: true/false` only)
- CORS is currently open (`*`) — restrict in production
- No authentication middleware yet — add JWT/API key auth before production deployment
