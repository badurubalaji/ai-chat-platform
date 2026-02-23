# NeuralGate API Integration Guide

> **Source:** [neuralgate/docs/08-api-integration-guide.md](https://github.com/badurubalaji/neuralgate/blob/master/docs/08-api-integration-guide.md)

Complete developer guide for integrating with the NeuralGate AI Gateway platform.

## Table of Contents

1. [Prerequisites](#1-prerequisites)
2. [Authentication](#2-authentication)
3. [Chat API](#3-chat-api)
4. [Generate API](#4-generate-api)
5. [Embeddings API](#5-embeddings-api)
6. [Conversation Management](#6-conversation-management)
7. [File Uploads](#7-file-uploads)
8. [Feedback](#8-feedback)
9. [Error Handling](#9-error-handling)
10. [Rate Limiting](#10-rate-limiting)
11. [Complete Code Examples](#11-complete-code-examples)
12. [Best Practices](#12-best-practices)

---

## 1. Prerequisites

Before making API calls, ensure the following hierarchy is set up in the NeuralGate admin dashboard:

```
Client (organization)
  └── Project (workspace)
        ├── Model (deployed & active)
        └── API Key (scoped to project)
```

**Setup steps:**

1. Create a **Client** (organization/tenant)
2. Create a **Project** under the client
3. Register and deploy a **Model** for the project
4. Generate an **API Key** — this gives you a `client_id` and `client_secret`

The API key pair is used for OAuth2 authentication to obtain access tokens.

---

## 2. Authentication

NeuralGate uses **OAuth 2.0 `client_credentials` grant** for machine-to-machine authentication.

### Request a Token

**Endpoint:** `POST /api/v1/oauth/token`

#### cURL

```bash
curl -X POST http://YOUR_SERVER:8090/api/v1/oauth/token \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "ng_your_client_id",
    "client_secret": "ngs_your_client_secret",
    "grant_type": "client_credentials"
  }'
```

#### Python

```python
import requests

TOKEN_URL = "http://YOUR_SERVER:8090/api/v1/oauth/token"

def get_token(client_id: str, client_secret: str) -> str:
    resp = requests.post(TOKEN_URL, json={
        "client_id": client_id,
        "client_secret": client_secret,
        "grant_type": "client_credentials"
    })
    resp.raise_for_status()
    return resp.json()["access_token"]

token = get_token("ng_xxx", "ngs_xxx")
```

#### JavaScript

```javascript
async function getToken(clientId, clientSecret) {
  const resp = await fetch('http://YOUR_SERVER:8090/api/v1/oauth/token', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      client_id: clientId,
      client_secret: clientSecret,
      grant_type: 'client_credentials'
    })
  });
  const data = await resp.json();
  return data.access_token;
}
```

#### Response

```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "token_type": "Bearer",
  "expires_in": 900
}
```

Tokens expire after **900 seconds (15 minutes)**. Cache the token and refresh before expiry.

### Revoke a Token

```bash
curl -X POST http://YOUR_SERVER:8090/api/v1/oauth/revoke \
  -H "Content-Type: application/json" \
  -d '{ "token": "eyJhbGciOiJIUzI1NiIs..." }'
```

---

## 3. Chat API

Send multi-turn chat completions with optional streaming.

**Endpoint:** `POST /api/v1/ai/chat`

### Request Format

```json
{
  "model": "your-model-name",
  "messages": [
    { "role": "user", "content": "Hello, how can you help me?" }
  ],
  "stream": false,
  "conversation_id": "optional-uuid",
  "options": {
    "temperature": 0.7,
    "top_p": 0.9,
    "top_k": 40,
    "num_predict": 512,
    "num_ctx": 4096
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `model` | string | No* | Model name. Auto-resolved from project if omitted. |
| `messages` | array | Yes | Array of `{role, content}` objects. Roles: `user`, `assistant`, `system`. |
| `stream` | boolean | No | `true` for streaming SSE responses. Default: `false`. |
| `conversation_id` | string | No | UUID of existing conversation to continue. |
| `options` | object | No | Model parameters (temperature, top_p, etc.). |

*Model is auto-resolved if the project has exactly one active model.

### Non-Streaming Response

```json
{
  "data": {
    "conversation_id": "abc-123-def-456",
    "message_id": "msg-789",
    "model": "your-model-name",
    "message": {
      "role": "assistant",
      "content": "Hello! I'm here to help you with..."
    },
    "done": true,
    "prompt_eval_count": 42,
    "eval_count": 128
  }
}
```

### Streaming Response (SSE)

When `stream: true`, the response is a stream of Server-Sent Events:

```
event: message
data: {"model":"your-model","message":{"role":"assistant","content":"Hello"},"done":false}

event: message
data: {"model":"your-model","message":{"role":"assistant","content":"!"},"done":false}

event: message
data: {"model":"your-model","message":{"role":"assistant","content":""},"done":true,"prompt_eval_count":42,"eval_count":128}

event: conversation
data: {"conversation_id":"abc-123","message_id":"msg-789"}
```

The final `conversation` event contains the persisted conversation and message IDs.

#### cURL (Streaming)

```bash
curl -N -X POST http://YOUR_SERVER:8090/api/v1/ai/chat \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "your-model",
    "messages": [{"role": "user", "content": "Tell me a story"}],
    "stream": true
  }'
```

#### Python (Streaming)

```python
import requests
import json

def stream_chat(token, model, messages):
    resp = requests.post(
        "http://YOUR_SERVER:8090/api/v1/ai/chat",
        headers={"Authorization": f"Bearer {token}"},
        json={"model": model, "messages": messages, "stream": True},
        stream=True
    )
    for line in resp.iter_lines():
        if line:
            line = line.decode("utf-8")
            if line.startswith("data: "):
                chunk = json.loads(line[6:])
                if "message" in chunk:
                    print(chunk["message"].get("content", ""), end="", flush=True)
                if chunk.get("done"):
                    break

stream_chat(token, "your-model", [{"role": "user", "content": "Hello"}])
```

#### JavaScript (Streaming)

```javascript
async function streamChat(token, model, messages) {
  const resp = await fetch('http://YOUR_SERVER:8090/api/v1/ai/chat', {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({ model, messages, stream: true })
  });

  const reader = resp.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true });
    const lines = buffer.split('\n');
    buffer = lines.pop();
    for (const line of lines) {
      if (line.startsWith('data: ')) {
        const chunk = JSON.parse(line.slice(6));
        process.stdout.write(chunk.message?.content || '');
        if (chunk.done) return;
      }
    }
  }
}
```

### Available Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `temperature` | float | 0.8 | Randomness (0.0 = deterministic, 2.0 = very random) |
| `top_p` | float | 0.9 | Nucleus sampling threshold |
| `top_k` | int | 40 | Limits token selection to top K candidates |
| `num_predict` | int | 128 | Maximum tokens to generate |
| `num_ctx` | int | 2048 | Context window size in tokens |
| `repeat_penalty` | float | 1.1 | Penalty for repeating tokens |
| `seed` | int | — | Random seed for reproducibility |
| `stop` | array | — | Stop sequences (e.g., `["\n", "END"]`) |

---

## 4. Generate API

Simple text completion (non-chat format).

**Endpoint:** `POST /api/v1/ai/generate`

### Request

```json
{
  "model": "your-model",
  "prompt": "Write a haiku about coding:",
  "system": "You are a creative poet.",
  "options": {
    "temperature": 0.9
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `model` | string | Yes | Model name |
| `prompt` | string | Yes | The input prompt |
| `system` | string | No | System prompt override |
| `options` | object | No | Model parameters |

#### cURL

```bash
curl -X POST http://YOUR_SERVER:8090/api/v1/ai/generate \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "your-model",
    "prompt": "Summarize the benefits of AI in healthcare"
  }'
```

#### Python

```python
resp = requests.post(
    "http://YOUR_SERVER:8090/api/v1/ai/generate",
    headers={"Authorization": f"Bearer {token}"},
    json={"model": "your-model", "prompt": "Summarize the benefits of AI"}
)
print(resp.json()["data"]["response"])
```

### Response

```json
{
  "data": {
    "model": "your-model",
    "response": "AI in healthcare can...",
    "done": true,
    "prompt_eval_count": 15,
    "eval_count": 89
  }
}
```

---

## 5. Embeddings API

Generate vector embeddings for semantic search and RAG pipelines.

**Endpoint:** `POST /api/v1/ai/embeddings`

### Request

```json
{
  "model": "your-embedding-model",
  "prompt": "The patient presents with chronic fatigue"
}
```

#### cURL

```bash
curl -X POST http://YOUR_SERVER:8090/api/v1/ai/embeddings \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "nomic-embed-text",
    "prompt": "The patient presents with chronic fatigue"
  }'
```

#### Python

```python
resp = requests.post(
    "http://YOUR_SERVER:8090/api/v1/ai/embeddings",
    headers={"Authorization": f"Bearer {token}"},
    json={"model": "nomic-embed-text", "prompt": "chronic fatigue symptoms"}
)
embedding = resp.json()["data"]["embedding"]
print(f"Embedding dimension: {len(embedding)}")
```

### Response

```json
{
  "data": {
    "embedding": [0.0123, -0.0456, 0.0789, ...]
  }
}
```

---

## 6. Conversation Management

NeuralGate persists conversations server-side. You don't need to manage message arrays manually.

### How It Works

1. Send a chat request **without** `conversation_id` → a new conversation is created
2. The response includes `conversation_id` and `message_id`
3. Pass `conversation_id` in subsequent requests → server auto-loads history
4. You only send the **new message** — the server prepends previous context

### Start a New Conversation

```bash
curl -X POST http://YOUR_SERVER:8090/api/v1/ai/chat \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [{"role": "user", "content": "What is diabetes?"}]
  }'

# Response: { "data": { "conversation_id": "abc-123", ... } }
```

### Continue a Conversation

```bash
curl -X POST http://YOUR_SERVER:8090/api/v1/ai/chat \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "conversation_id": "abc-123",
    "messages": [{"role": "user", "content": "What are the symptoms?"}]
  }'
```

### List Conversations

```bash
curl http://YOUR_SERVER:8090/api/v1/ai/conversations \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### Get Conversation with Messages

```bash
curl http://YOUR_SERVER:8090/api/v1/ai/conversations/CONVERSATION_ID \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### Update Conversation Title

```bash
curl -X PUT http://YOUR_SERVER:8090/api/v1/ai/conversations/CONVERSATION_ID \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title": "Diabetes FAQ"}'
```

### Delete a Conversation

```bash
curl -X DELETE http://YOUR_SERVER:8090/api/v1/ai/conversations/CONVERSATION_ID \
  -H "Authorization: Bearer YOUR_TOKEN"
```

---

## 7. File Uploads

Send files alongside chat messages for multimodal understanding.

### Multipart Form-Data

```bash
curl -X POST http://YOUR_SERVER:8090/api/v1/ai/chat \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -F "model=your-model" \
  -F 'messages=[{"role": "user", "content": "Describe this document"}]' \
  -F "files=@report.pdf"
```

### Base64 JSON

```bash
curl -X POST http://YOUR_SERVER:8090/api/v1/ai/chat \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "your-model",
    "messages": [{
      "role": "user",
      "content": "Analyze this image",
      "files": [{
        "base64": "iVBORw0KGgo...",
        "content_type": "image/png"
      }]
    }]
  }'
```

### Python (File Upload)

```python
import json

with open("report.pdf", "rb") as f:
    resp = requests.post(
        "http://YOUR_SERVER:8090/api/v1/ai/chat",
        headers={"Authorization": f"Bearer {token}"},
        files={"files": ("report.pdf", f, "application/pdf")},
        data={
            "model": "your-model",
            "messages": json.dumps([
                {"role": "user", "content": "Summarize this document"}
            ])
        }
    )
print(resp.json()["data"]["message"]["content"])
```

### Supported File Types

| Type | Extensions | Processing |
|------|-----------|------------|
| **Images** | PNG, JPEG, WebP | Passed directly to vision models |
| **PDFs** | PDF | Text extracted and injected into context |
| **Audio** | MP3, WAV | Transcribed via Whisper, text injected |
| **Video** | MP4 | Audio track transcribed via Whisper |

---

## 8. Feedback

Submit feedback on AI responses for quality tracking and DPO training.

**Endpoint:** `POST /api/v1/ai/conversations/:conversationId/messages/:messageId/feedback`

### Submit Feedback

```bash
curl -X POST http://YOUR_SERVER:8090/api/v1/ai/conversations/CONV_ID/messages/MSG_ID/feedback \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "rating": 1,
    "comment": "Accurate and helpful response"
  }'
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `rating` | int | Yes | `1` (thumbs up) or `-1` (thumbs down) |
| `comment` | string | No | Optional text feedback |

### Get Feedback for a Conversation

```bash
curl http://YOUR_SERVER:8090/api/v1/ai/conversations/CONV_ID/feedback \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### Export DPO Training Data

Export feedback as DPO (Direct Preference Optimization) training pairs:

```bash
curl http://YOUR_SERVER:8090/api/v1/projects/PROJECT_ID/feedback/export/dpo \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -o dpo_data.jsonl
```

---

## 9. Error Handling

All responses use a standard envelope format:

### Success

```json
{
  "data": { ... },
  "meta": { ... }
}
```

### Error

```json
{
  "error": {
    "code": 400,
    "message": "model is required"
  }
}
```

### Common Error Codes

| Code | Meaning | Action |
|------|---------|--------|
| `400` | Bad Request | Fix request format or missing fields |
| `401` | Unauthorized | Token expired — request a new one |
| `403` | Forbidden | Model not available for project, or access denied |
| `404` | Not Found | Resource doesn't exist |
| `409` | Conflict | Duplicate resource (e.g., email already exists) |
| `429` | Too Many Requests | Rate limit exceeded — back off and retry |
| `500` | Internal Error | Server issue — retry with backoff |
| `503` | Service Unavailable | Model loading or inference server busy |

### Retry Strategy

```python
import time
import random

def retry_request(fn, max_retries=3):
    for attempt in range(max_retries):
        try:
            resp = fn()
            if resp.status_code == 429:
                wait = int(resp.headers.get("Retry-After", 2 ** attempt))
                time.sleep(wait + random.uniform(0, 1))
                continue
            resp.raise_for_status()
            return resp.json()
        except requests.exceptions.RequestException:
            if attempt == max_retries - 1:
                raise
            time.sleep(2 ** attempt + random.uniform(0, 1))
```

---

## 10. Rate Limiting

API keys have per-minute (RPM) and per-day (RPD) rate limits configured at the project level.

### Rate Limit Response

When exceeded, the API returns:

```
HTTP/1.1 429 Too Many Requests
Retry-After: 30
Content-Type: application/json

{
  "error": {
    "code": 429,
    "message": "rate limit exceeded"
  }
}
```

### Handling Rate Limits

1. Check for `429` status codes
2. Read the `Retry-After` header for wait time in seconds
3. Implement exponential backoff with jitter
4. Cache tokens to avoid unnecessary auth requests

---

## 11. Complete Code Examples

### Python Client

```python
import requests
import json
import time

class NeuralGateClient:
    def __init__(self, base_url, client_id, client_secret):
        self.base_url = base_url.rstrip("/")
        self.client_id = client_id
        self.client_secret = client_secret
        self._token = None
        self._token_expires = 0

    @property
    def token(self):
        if self._token and time.time() < self._token_expires - 60:
            return self._token
        resp = requests.post(f"{self.base_url}/api/v1/oauth/token", json={
            "client_id": self.client_id,
            "client_secret": self.client_secret,
            "grant_type": "client_credentials"
        })
        resp.raise_for_status()
        data = resp.json()
        self._token = data["access_token"]
        self._token_expires = time.time() + data["expires_in"]
        return self._token

    def _headers(self):
        return {"Authorization": f"Bearer {self.token}", "Content-Type": "application/json"}

    def chat(self, messages, model=None, conversation_id=None, stream=False, options=None):
        payload = {"messages": messages, "stream": stream}
        if model: payload["model"] = model
        if conversation_id: payload["conversation_id"] = conversation_id
        if options: payload["options"] = options

        if stream:
            return self._stream_chat(payload)

        resp = requests.post(f"{self.base_url}/api/v1/ai/chat",
                             headers=self._headers(), json=payload)
        resp.raise_for_status()
        return resp.json()["data"]

    def _stream_chat(self, payload):
        resp = requests.post(f"{self.base_url}/api/v1/ai/chat",
                             headers=self._headers(), json=payload, stream=True)
        resp.raise_for_status()
        full_content = ""
        conversation_id = None
        for line in resp.iter_lines():
            if not line: continue
            line = line.decode("utf-8")
            if line.startswith("data: "):
                chunk = json.loads(line[6:])
                if "message" in chunk:
                    content = chunk["message"].get("content", "")
                    full_content += content
                    yield {"type": "token", "content": content}
                if chunk.get("done"):
                    yield {"type": "done", "prompt_tokens": chunk.get("prompt_eval_count"),
                           "completion_tokens": chunk.get("eval_count")}
            elif line.startswith("event: conversation"):
                pass  # next line has conversation data
            elif "conversation_id" in line and line.startswith("data: "):
                conv_data = json.loads(line[6:])
                conversation_id = conv_data.get("conversation_id")
                yield {"type": "conversation", "conversation_id": conversation_id}

    def generate(self, prompt, model, system=None, options=None):
        payload = {"model": model, "prompt": prompt}
        if system: payload["system"] = system
        if options: payload["options"] = options
        resp = requests.post(f"{self.base_url}/api/v1/ai/generate",
                             headers=self._headers(), json=payload)
        resp.raise_for_status()
        return resp.json()["data"]

    def embeddings(self, prompt, model):
        resp = requests.post(f"{self.base_url}/api/v1/ai/embeddings",
                             headers=self._headers(), json={"model": model, "prompt": prompt})
        resp.raise_for_status()
        return resp.json()["data"]["embedding"]

    def list_conversations(self):
        resp = requests.get(f"{self.base_url}/api/v1/ai/conversations",
                            headers=self._headers())
        resp.raise_for_status()
        return resp.json()["data"]

    def submit_feedback(self, conversation_id, message_id, rating, comment=None):
        payload = {"rating": rating}
        if comment: payload["comment"] = comment
        resp = requests.post(
            f"{self.base_url}/api/v1/ai/conversations/{conversation_id}/messages/{message_id}/feedback",
            headers=self._headers(), json=payload)
        resp.raise_for_status()
        return resp.json()["data"]


# Usage
client = NeuralGateClient("http://localhost:8090", "ng_xxx", "ngs_xxx")

# Simple chat
result = client.chat([{"role": "user", "content": "What is diabetes?"}])
print(result["message"]["content"])

# Streaming chat
for event in client.chat([{"role": "user", "content": "Tell me a story"}], stream=True):
    if event["type"] == "token":
        print(event["content"], end="", flush=True)

# Continue conversation
result2 = client.chat(
    [{"role": "user", "content": "What are the symptoms?"}],
    conversation_id=result["conversation_id"]
)
```

### JavaScript/TypeScript Client

```javascript
class NeuralGateClient {
  constructor(baseUrl, clientId, clientSecret) {
    this.baseUrl = baseUrl.replace(/\/$/, '');
    this.clientId = clientId;
    this.clientSecret = clientSecret;
    this._token = null;
    this._tokenExpires = 0;
  }

  async getToken() {
    if (this._token && Date.now() / 1000 < this._tokenExpires - 60) {
      return this._token;
    }
    const resp = await fetch(`${this.baseUrl}/api/v1/oauth/token`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        client_id: this.clientId,
        client_secret: this.clientSecret,
        grant_type: 'client_credentials'
      })
    });
    const data = await resp.json();
    this._token = data.access_token;
    this._tokenExpires = Date.now() / 1000 + data.expires_in;
    return this._token;
  }

  async chat(messages, { model, conversationId, stream = false, options } = {}) {
    const token = await this.getToken();
    const payload = { messages, stream };
    if (model) payload.model = model;
    if (conversationId) payload.conversation_id = conversationId;
    if (options) payload.options = options;

    const resp = await fetch(`${this.baseUrl}/api/v1/ai/chat`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(payload)
    });

    if (!stream) {
      const json = await resp.json();
      return json.data;
    }

    return this._readStream(resp);
  }

  async *_readStream(resp) {
    const reader = resp.body.getReader();
    const decoder = new TextDecoder();
    let buffer = '';

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split('\n');
      buffer = lines.pop();
      for (const line of lines) {
        if (line.startsWith('data: ')) {
          yield JSON.parse(line.slice(6));
        }
      }
    }
  }

  async generate(prompt, model, { system, options } = {}) {
    const token = await this.getToken();
    const payload = { model, prompt };
    if (system) payload.system = system;
    if (options) payload.options = options;
    const resp = await fetch(`${this.baseUrl}/api/v1/ai/generate`, {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });
    return (await resp.json()).data;
  }

  async embeddings(prompt, model) {
    const token = await this.getToken();
    const resp = await fetch(`${this.baseUrl}/api/v1/ai/embeddings`, {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({ model, prompt })
    });
    return (await resp.json()).data.embedding;
  }
}

// Usage
const client = new NeuralGateClient('http://localhost:8090', 'ng_xxx', 'ngs_xxx');

// Simple chat
const result = await client.chat([{ role: 'user', content: 'What is diabetes?' }]);
console.log(result.message.content);

// Streaming
for await (const chunk of await client.chat(
  [{ role: 'user', content: 'Tell me a story' }],
  { stream: true }
)) {
  process.stdout.write(chunk.message?.content || '');
}
```

---

## 12. Best Practices

### Token Management

- Cache access tokens and refresh **60 seconds before expiry**
- Never hardcode tokens — always exchange credentials at runtime
- Store `client_id` and `client_secret` in environment variables

### Streaming

- Prefer streaming for user-facing applications (faster perceived response)
- Handle the final `conversation` SSE event to get `conversation_id`
- Implement timeout handling for long-running streams

### Conversations

- Reuse `conversation_id` for multi-turn context — don't resend full history
- The server caps history to the most recent N messages (configurable)
- Delete old conversations to keep storage manageable

### Error Recovery

- Implement exponential backoff with jitter for retries
- On `401`, refresh your token and retry once
- On `429`, respect the `Retry-After` header
- On `503`, the model may be loading — wait and retry

### Security

- Rotate API keys regularly
- Use scoped keys with minimum necessary permissions
- Never expose credentials in client-side code or logs
- Use HTTPS in production

### Performance

- Use `num_predict` to cap response length for predictable latency
- Lower `temperature` for deterministic outputs (caching-friendly)
- Set appropriate `num_ctx` — larger context = slower inference
- Batch requests where possible using the Batch Jobs API
