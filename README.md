# AI Chat Platform

A complete AI Chat platform with a reusable **Angular frontend library** (`@mdp/ai-chat`) and a configurable **multi-provider backend** (Go). Designed to be integrated into any Angular application — EHR, School ERP, Data Backup, and more.

## 🚀 Features

- **Multi-Provider Support** — Claude, OpenAI, Gemini, Ollama, and NeuralGateway
- **Frontend Library** — Reusable `@mdp/ai-chat` Angular library with sidebar, settings, and usage dashboard
- **Streaming** — Server-Sent Events (SSE) for real-time AI responses
- **Tool Use** — AI can call your app's internal APIs (search patients, schedule appointments, etc.)
- **Secure** — AES-256-GCM encryption for API keys, server-side credential management
- **BYOK** — Users can bring their own API keys (Claude, OpenAI, Gemini)
- **Analytics** — Token usage tracking and visualization per user
- **Action Cards** — AI can propose actions that the user can approve/dismiss

---

## 📁 Project Structure

```
ai-chat-platform/
├── backend/                    # Go backend server
│   ├── cmd/server/             # Server entrypoint
│   ├── cmd/migrate/            # Database migration tool
│   ├── db/migrations/          # SQL migration files
│   └── internal/
│       ├── api/                # HTTP handlers
│       ├── config/             # App configuration
│       ├── domain/             # Adapter, orchestrator, tool executor
│       ├── models/             # Data models
│       ├── providers/          # AI providers (claude, openai, gemini, ollama, neuralgate)
│       ├── registry/           # Provider registry
│       └── store/              # PostgreSQL store
├── frontend/
│   └── projects/
│       ├── mdp-ai-chat/        # 📦 Reusable Angular library
│       │   └── src/lib/
│       │       ├── components/ # AiSidebar, AiSettings, AiUsageDashboard, AiActionCard, AiEmptyState
│       │       ├── services/   # AiChatService, AiProviderConfigService, AiUsageService, AiContextService
│       │       ├── models/     # TypeScript interfaces
│       │       ├── pipes/      # MarkdownRenderPipe
│       │       └── providers/  # provideAiChat() configuration
│       └── demo-app/           # Demo application for development
├── docker-compose.yml          # Full-stack Docker Compose
├── Makefile                    # Build & dev shortcuts
├── start.sh                    # One-command dev startup
└── setup_db.sh                 # Database setup helper
```

---

## 🛠️ Prerequisites

| Tool        | Version  | Purpose                    |
|-------------|----------|----------------------------|
| Go          | 1.22+    | Backend compilation        |
| Node.js     | 18+      | Frontend build & dev       |
| PostgreSQL  | 15+      | Chat history & config      |
| Docker      | 20+      | *(Optional)* Containerized deployment |

---

## 🏁 Getting Started (Development)

### Quick Start

```bash
# 1. Clone the repo
git clone https://github.com/badurubalaji/ai-chat-platform.git
cd ai-chat-platform

# 2. Set up everything (downloads deps, builds binaries)
make setup

# 3. Start PostgreSQL (via Docker)
docker-compose up -d postgres

# 4. Run database migrations
make migrate-up

# 5. Start the full dev environment
./start.sh
# → Backend:  http://localhost:8080
# → Frontend: http://localhost:4200
```

### Step-by-Step Setup

#### 1. Database

**Option A: Docker (recommended)**
```bash
docker-compose up -d postgres
```

**Option B: Local PostgreSQL**
```bash
# Create user and database
sudo -u postgres psql -c "CREATE USER ai_user WITH PASSWORD 'ai_password';"
sudo -u postgres psql -c "CREATE DATABASE ai_chat_db OWNER ai_user;"

# Or use the helper script
./setup_db.sh
```

Set a custom connection string if needed:
```bash
export DATABASE_URL="postgres://user:pass@localhost:5432/dbname?sslmode=disable"
```

#### 2. Build Backend
```bash
# Build server and migration tool
make build-backend
cd backend && go build -o ../bin/migrate cmd/migrate/main.go && cd ..

# Apply database schema
./bin/migrate up
```

#### 3. Start Backend
```bash
./bin/server
# Runs on http://localhost:8080
```

#### 4. Start Frontend
```bash
cd frontend
npm install
npm start
# Runs on http://localhost:4200
```

---

## 🔧 Environment Variables

| Variable            | Description                          | Default                                                                              | Required     |
|---------------------|--------------------------------------|--------------------------------------------------------------------------------------|--------------|
| `DATABASE_URL`      | PostgreSQL connection string         | `postgres://ai_user:ai_password@localhost:5432/ai_chat_db?sslmode=disable`          | Yes          |
| `AI_ENCRYPTION_KEY` | 32-byte hex key for AES-256-GCM     | Dev default (insecure)                                                               | Yes (prod)   |
| `PORT`              | Backend listen port                  | `8080`                                                                               | No           |
| `ADAPTER_CONFIG_PATH` | Path to domain adapter config      | *(empty = generic mode)*                                                             | Per-app      |

Generate a production encryption key:
```bash
openssl rand -hex 32
```

---

## 🚢 Deployment Guide

### Architecture Overview

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

Each consuming application gets:
- **Frontend**: `@mdp/ai-chat` npm package installed as a dependency
- **Backend**: Its own `ai-chat-backend` container with a domain-specific `adapter-config.json`

---

### Step 1: Build & Push Backend Docker Image

**Create `backend/Dockerfile`:**
```dockerfile
FROM golang:1.24 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /server ./cmd/server

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /server /server
ENTRYPOINT ["/server"]
```

```bash
cd backend
docker build -t your-registry/ai-chat-backend:1.0.0 .
docker push your-registry/ai-chat-backend:1.0.0
```

---

### Step 2: Publish Frontend Library to Private npm Registry

**Option A: Verdaccio (Self-Hosted — recommended for on-premise)**
```bash
# Run Verdaccio
docker run -d --name verdaccio -p 4873:4873 verdaccio/verdaccio

# Add user
npm adduser --registry http://your-verdaccio-host:4873
```

**Option B: GitHub Packages**
```ini
# frontend/.npmrc
@mdp:registry=https://npm.pkg.github.com
//npm.pkg.github.com/:_authToken=${GITHUB_TOKEN}
```

**Option C: GitLab Package Registry**
```ini
# frontend/.npmrc
@mdp:registry=https://gitlab.yourcompany.com/api/v4/projects/<PROJECT_ID>/packages/npm/
//gitlab.yourcompany.com/api/v4/projects/<PROJECT_ID>/packages/npm/:_authToken=${GITLAB_TOKEN}
```

**Build & Publish:**
```bash
cd frontend
ng build mdp-ai-chat
cd dist/mdp-ai-chat
npm publish --registry http://your-verdaccio-host:4873  # or just `npm publish` if .npmrc configured
```

---

### Step 3: Deploy Per Application

Each application (EHR, School ERP, etc.) needs:

1. **A PostgreSQL database**
2. **An `adapter-config.json`** with domain-specific settings
3. **A running `ai-chat-backend` container**

**Example `docker-compose.ehr.yml`:**
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
    external: true

volumes:
  ehr-pgdata:
```

**Example `adapter-config-ehr.json`:**
```json
{
  "domain": "ehr",
  "display_name": "Smart Health Assistant",
  "system_prompt": "You are a healthcare AI assistant...",
  "default_provider": {
    "provider": "neuralgate",
    "model": "auto",
    "endpoint_url": "https://neuralgateway.hospital.local",
    "client_id": "<your-client-id>",
    "client_secret": "<your-client-secret>"
  },
  "tools": [
    {
      "name": "search_patients",
      "description": "Search for patients by name or MRN",
      "parameters": { ... },
      "execution": {
        "type": "http",
        "method": "GET",
        "url": "http://ehr-api:3000/api/patients/search"
      }
    }
  ]
}
```

> **Important:** The backend container must be on the same Docker network as your app's API so tool execution URLs resolve correctly.

**Run migrations and start:**
```bash
# Run migrations
psql $DATABASE_URL -f backend/db/migrations/000001_init_schema.up.sql
psql $DATABASE_URL -f backend/db/migrations/000002_tool_executions.up.sql

# Start
docker-compose -f docker-compose.ehr.yml up -d
```

---

## 📦 Frontend Library Installation Guide

### Step 1: Configure Private Registry

Add to your Angular project's `.npmrc`:
```ini
# For Verdaccio
@mdp:registry=http://your-verdaccio-host:4873

# For GitHub Packages
# @mdp:registry=https://npm.pkg.github.com
# //npm.pkg.github.com/:_authToken=${GITHUB_TOKEN}
```

### Step 2: Install the Package

```bash
npm install @mdp/ai-chat
```

### Step 3: Configure in `app.config.ts`

```typescript
import { ApplicationConfig } from '@angular/core';
import { provideRouter } from '@angular/router';
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http';
import { provideAnimations } from '@angular/platform-browser/animations';
import { provideAiChat } from '@mdp/ai-chat';
import { inject } from '@angular/core';
import { AuthService } from './core/auth/auth.service';

export const appConfig: ApplicationConfig = {
  providers: [
    provideRouter(routes),
    provideHttpClient(withInterceptorsFromDi()),
    provideAnimations(),
    provideAiChat({
      apiBaseUrl: 'https://your-ai-backend-url:8080',  // Your backend URL
      authTokenFn: () => inject(AuthService).getAccessToken$(),
      appName: 'My AI Assistant',
      enableTools: true,
      enableActionCards: true,
    })
  ]
};
```

### Step 4: Add AI Sidebar to Your Layout

```typescript
import { Component } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { AiSidebarComponent, AiProposedAction } from '@mdp/ai-chat';

@Component({
  selector: 'app-main-layout',
  standalone: true,
  imports: [RouterOutlet, AiSidebarComponent],
  template: `
    <div class="app-layout">
      <app-header />
      <div class="content-area">
        <app-nav-sidebar />
        <main>
          <router-outlet />
        </main>
        <mdp-ai-sidebar
          [contextBadge]="currentPageContext"
          (actionRequested)="handleAiAction($event)">
        </mdp-ai-sidebar>
      </div>
    </div>
  `
})
export class MainLayoutComponent {
  currentPageContext = 'Dashboard';

  handleAiAction(action: AiProposedAction): void {
    switch (action.action_type) {
      case 'navigate':
        this.router.navigate([action.params['route']]);
        break;
    }
  }
}
```

### Step 5: Add AI Settings Page (Optional — for BYOK)

```typescript
import { AiSettingsComponent, AiUsageDashboardComponent } from '@mdp/ai-chat';

@Component({
  imports: [AiSettingsComponent, AiUsageDashboardComponent],
  template: `
    <h2>AI Provider Settings</h2>
    <p>Configure your own AI provider (optional).</p>
    <mdp-ai-settings />

    <h2>AI Usage Dashboard</h2>
    <mdp-ai-usage-dashboard />
  `
})
export class SettingsComponent {}
```

### Configuration Options

| Option                | Type                          | Default          | Description                        |
|-----------------------|-------------------------------|------------------|------------------------------------|
| `apiBaseUrl`          | `string`                      | —                | Backend URL (**required**)         |
| `authTokenFn`         | `() => Observable<string>`    | —                | JWT token provider (**required**)  |
| `appName`             | `string`                      | `AI Assistant`   | Display name in sidebar header     |
| `appDescription`      | `string`                      | —                | Context sent to system prompt      |
| `enableTools`         | `boolean`                     | `true`           | Enable tool use / function calling |
| `enableActionCards`   | `boolean`                     | `true`           | Enable AI-proposed action cards    |
| `sidebarWidth`        | `number`                      | `400`            | Initial sidebar width (px)         |
| `sidebarMinWidth`     | `number`                      | `300`            | Minimum sidebar width (px)         |
| `sidebarMaxWidth`     | `number`                      | `600`            | Maximum sidebar width (px)         |
| `markdownOptions`     | `MarkdownRenderOptions`       | —                | Syntax highlighting config         |

### Exported Components & Services

| Export                       | Type        | Purpose                                    |
|------------------------------|-------------|--------------------------------------------|
| `AiSidebarComponent`        | Component   | Main chat sidebar panel                    |
| `AiSettingsComponent`       | Component   | BYOK provider settings form               |
| `AiUsageDashboardComponent` | Component   | Token usage charts & stats                 |
| `AiActionCardComponent`     | Component   | Actionable card proposed by AI             |
| `AiEmptyStateComponent`     | Component   | Empty state placeholder                    |
| `AiChatService`             | Service     | Chat messaging & streaming                 |
| `AiProviderConfigService`   | Service     | Provider CRUD operations                   |
| `AiUsageService`            | Service     | Token usage data                           |
| `AiContextService`          | Service     | Page context injection                     |
| `MarkdownRenderPipe`        | Pipe        | Markdown → HTML rendering                  |
| `provideAiChat()`           | Function    | Angular provider factory                   |

---

## 🧪 Testing

```bash
# Backend tests
cd backend && go test ./...

# Frontend tests
cd frontend && npx ng test mdp-ai-chat --watch=false

# All tests
make test
```

---

## 📚 Additional Documentation

- [Integration Guide](_bmad-output/integration-guide.md) — Detailed per-app integration walkthrough (EHR, School ERP, Data Backup)

---

## 📋 Deployment Checklist

### Per Application

**Infrastructure:**
- [ ] PostgreSQL database created
- [ ] Migrations applied (`000001` + `000002`)
- [ ] `AI_ENCRYPTION_KEY` generated (`openssl rand -hex 32`)

**Backend:**
- [ ] `adapter-config.json` created with provider credentials and tools
- [ ] `ai-chat-backend` container deployed with `ADAPTER_CONFIG_PATH` mounted `:ro`
- [ ] Backend container on same Docker network as your app's API

**Frontend:**
- [ ] `.npmrc` configured for private registry
- [ ] `npm install @mdp/ai-chat`
- [ ] `provideAiChat()` configured with backend URL and auth token
- [ ] `<mdp-ai-sidebar>` added to main layout
- [ ] `<mdp-ai-settings>` added to settings page (optional)

**Security:**
- [ ] Production encryption key (not dev default)
- [ ] HTTPS enabled
- [ ] Credentials never exposed to frontend
- [ ] `adapter-config.json` mounted read-only

---

## 📄 License

Private — Internal use only.
