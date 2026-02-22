# AI Chat Platform — Setup, Private Registry & Integration Guide

## Table of Contents

1. [Overview](#1-overview)
2. [Private npm Registry Setup](#2-private-npm-registry-setup)
   - [Option A: Verdaccio (Self-Hosted)](#option-a-verdaccio-self-hosted)
   - [Option B: GitLab Package Registry](#option-b-gitlab-package-registry)
   - [Option C: GitHub Packages](#option-c-github-packages)
   - [Option D: JFrog Artifactory / Sonatype Nexus](#option-d-jfrog-artifactory--sonatype-nexus)
3. [Publishing mdp-ai-chat Library](#3-publishing-mdp-ai-chat-library)
4. [Backend Deployment](#4-backend-deployment)
5. [NeuralGateway Setup Per Application](#5-neuralgateway-setup-per-application)
6. [Integrating into EHR](#6-integrating-into-ehr)
7. [Integrating into School ERP](#7-integrating-into-school-erp)
8. [Integrating into Data Backup & Recovery](#8-integrating-into-data-backup--recovery)
9. [Adapter Config Reference](#9-adapter-config-reference)
10. [Deployment Checklist](#10-deployment-checklist)

---

## 1. Overview

The AI Chat Platform consists of two distributable artifacts:

| Artifact | Type | Distribution |
|----------|------|-------------|
| **`mdp-ai-chat`** | Angular library (npm package) | Private npm registry |
| **ai-chat-backend** | Go binary + migrations | Docker image or binary |

**Integration model:** Your existing applications (EHR, School ERP, etc.) install `mdp-ai-chat` from your private registry as a dependency. They do NOT clone or fork the ai-chat-platform repo. Each application gets its own backend instance configured with a domain-specific `adapter-config.json` containing NeuralGateway credentials obtained at deployment time.

```
┌──────────────────────────────────────────────────────────────┐
│  ai-chat-platform (this repo)                                │
│                                                              │
│  frontend/projects/mdp-ai-chat/  ──── npm publish ────┐     │
│  backend/                        ──── docker build ──┐ │     │
└──────────────────────────────────────────────────────┼─┼─────┘
                                                       │ │
          ┌────────────────────────────────────────────┘ │
          │                                              │
          ▼                                              ▼
   ┌─────────────────┐                    ┌─────────────────────┐
   │  Private npm     │                    │  Container Registry  │
   │  Registry        │                    │  (Docker Hub/GitLab) │
   │  (Verdaccio /    │                    │                     │
   │   GitLab / GH)   │                    │  ai-chat-backend    │
   └──────┬──────────┘                    └──────┬──────────────┘
          │ npm install                          │ docker pull
          │                                      │
    ┌─────┴──────────────────────────────────────┴──────────┐
    │                                                        │
    ▼                  ▼                    ▼                 │
┌──────────┐   ┌──────────────┐   ┌─────────────────┐       │
│  EHR App │   │ School ERP   │   │ Data Backup App │       │
│  Angular │   │ Angular      │   │ Angular         │       │
│  + backend│   │ + backend    │   │ + backend       │       │
└──────────┘   └──────────────┘   └─────────────────┘       │
     │               │                    │                   │
     └───────────────┴────────────────────┘                   │
                     │                                        │
                     ▼                                        │
              NeuralGateway                                   │
              (separate project per app                       │
               with its own client_id/secret)                 │
```

---

## 2. Private npm Registry Setup

### Option A: Verdaccio (Self-Hosted) — Recommended for On-Premise

Verdaccio is a lightweight, zero-config private npm registry you can run on your own infrastructure.

#### Install & Run

```bash
# Install globally
npm install -g verdaccio

# Run (default port 4873)
verdaccio

# Or run with Docker
docker run -d \
  --name verdaccio \
  -p 4873:4873 \
  -v verdaccio-storage:/verdaccio/storage \
  -v verdaccio-conf:/verdaccio/conf \
  verdaccio/verdaccio
```

#### Configure Verdaccio

**`/verdaccio/conf/config.yaml`:**
```yaml
storage: ./storage
auth:
  htpasswd:
    file: ./htpasswd
    max_users: 100

uplinks:
  npmjs:
    url: https://registry.npmjs.org/

packages:
  # Scope your private packages
  '@mdp/*':
    access: $authenticated
    publish: $authenticated
    unpublish: $authenticated

  # Everything else proxied to public npm
  '**':
    access: $all
    proxy: npmjs

server:
  keepAliveTimeout: 60

listen:
  - 0.0.0.0:4873

# Optional: Enable HTTPS
# https:
#   key: /path/to/server.key
#   cert: /path/to/server.crt
```

#### Create User

```bash
npm adduser --registry http://your-verdaccio-host:4873
# Enter username, password, email
```

#### Docker Compose (Production)

```yaml
services:
  verdaccio:
    image: verdaccio/verdaccio:6
    container_name: verdaccio
    ports:
      - "4873:4873"
    volumes:
      - ./verdaccio/config.yaml:/verdaccio/conf/config.yaml:ro
      - ./verdaccio/htpasswd:/verdaccio/conf/htpasswd
      - verdaccio-storage:/verdaccio/storage
    restart: unless-stopped

volumes:
  verdaccio-storage:
```

---

### Option B: GitLab Package Registry

If your organization uses GitLab, the package registry is built-in.

#### Configure .npmrc in ai-chat-platform

**`frontend/.npmrc`:**
```ini
# Replace 123 with your GitLab project ID
@mdp:registry=https://gitlab.yourcompany.com/api/v4/projects/123/packages/npm/
//gitlab.yourcompany.com/api/v4/projects/123/packages/npm/:_authToken=${GITLAB_TOKEN}
```

#### Publish

```bash
cd frontend
npm run build -- mdp-ai-chat
cd dist/mdp-ai-chat
npm publish
```

#### Install in Consuming Apps

**EHR app `.npmrc`:**
```ini
@mdp:registry=https://gitlab.yourcompany.com/api/v4/projects/123/packages/npm/
//gitlab.yourcompany.com/api/v4/projects/123/packages/npm/:_authToken=${GITLAB_TOKEN}
```

```bash
npm install @mdp/ai-chat
```

---

### Option C: GitHub Packages

#### Configure .npmrc in ai-chat-platform

**`frontend/.npmrc`:**
```ini
@mdp:registry=https://npm.pkg.github.com
//npm.pkg.github.com/:_authToken=${GITHUB_TOKEN}
```

#### Publish

```bash
cd frontend
npm run build -- mdp-ai-chat
cd dist/mdp-ai-chat
npm publish
```

#### Install in Consuming Apps

**EHR app `.npmrc`:**
```ini
@mdp:registry=https://npm.pkg.github.com
//npm.pkg.github.com/:_authToken=${GITHUB_TOKEN}
```

```bash
npm install @mdp/ai-chat
```

> **Note:** GitHub Packages requires a Personal Access Token (PAT) with `read:packages` scope for installs.

---

### Option D: JFrog Artifactory / Sonatype Nexus

If your organization already runs Artifactory or Nexus:

```ini
# .npmrc
@mdp:registry=https://artifactory.yourcompany.com/api/npm/npm-private/
//artifactory.yourcompany.com/api/npm/npm-private/:_authToken=${ARTIFACTORY_TOKEN}
```

The publish and install flows are identical to the above options.

---

## 3. Publishing mdp-ai-chat Library

### Step 1: Update Package Scope and Version

The library package.json needs a scoped name to work with private registries.

**`frontend/projects/mdp-ai-chat/package.json`:**
```json
{
  "name": "@mdp/ai-chat",
  "version": "1.0.0",
  "peerDependencies": {
    "@angular/common": "^21.1.0",
    "@angular/core": "^21.1.0",
    "@angular/material": "^21.1.0"
  },
  "dependencies": {
    "tslib": "^2.3.0"
  },
  "sideEffects": false
}
```

### Step 2: Build the Library

```bash
cd /path/to/ai-chat-platform/frontend
ng build mdp-ai-chat
```

Output is at `frontend/dist/mdp-ai-chat/`.

### Step 3: Publish

```bash
cd frontend/dist/mdp-ai-chat

# For Verdaccio
npm publish --registry http://your-verdaccio-host:4873

# For GitLab / GitHub / Artifactory (uses .npmrc config)
npm publish
```

### Step 4: Verify

```bash
npm info @mdp/ai-chat --registry http://your-verdaccio-host:4873
```

### CI/CD Publish (GitLab CI Example)

```yaml
# .gitlab-ci.yml
publish-ai-chat:
  stage: publish
  image: node:20
  script:
    - cd frontend
    - npm ci
    - npx ng build mdp-ai-chat
    - cd dist/mdp-ai-chat
    - echo "@mdp:registry=https://gitlab.yourcompany.com/api/v4/projects/${CI_PROJECT_ID}/packages/npm/" > .npmrc
    - echo "//gitlab.yourcompany.com/api/v4/projects/${CI_PROJECT_ID}/packages/npm/:_authToken=${CI_JOB_TOKEN}" >> .npmrc
    - npm publish
  only:
    - tags
```

---

## 4. Backend Deployment

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `DATABASE_URL` | PostgreSQL connection string | dev default | Yes |
| `PORT` | Server listen port | `8080` | No |
| `AI_ENCRYPTION_KEY` | 32-byte hex key for AES-256-GCM | dev default | Yes (production) |
| `ADAPTER_CONFIG_PATH` | Path to `adapter-config.json` | _(empty = generic mode)_ | Yes |

### Generate Encryption Key

```bash
openssl rand -hex 32
```

### Database Setup

```sql
CREATE USER ai_user WITH PASSWORD 'your_secure_password';
CREATE DATABASE ai_chat_db OWNER ai_user;
```

```bash
# Run migrations
psql $DATABASE_URL -f backend/db/migrations/000001_init_schema.up.sql
psql $DATABASE_URL -f backend/db/migrations/000002_tool_executions.up.sql
```

### Build Backend Docker Image

**`backend/Dockerfile`:**
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

Each application deployment pulls this same image but mounts a different `adapter-config.json`.

---

## 5. NeuralGateway Setup Per Application

Each application (EHR, School ERP, etc.) needs its **own NeuralGateway project** with separate credentials. This ensures model usage isolation and billing separation.

### For Each Application:

#### Step 1: Create NeuralGateway Project

1. Log in to NeuralGateway admin dashboard
2. **Projects** > **Create New Project**
3. Project name: `ehr` / `school-erp` / `data-backup`
4. Assign model (e.g., Llama 3, Mistral, Qwen)

#### Step 2: Generate Client Credentials

1. Project settings > **API Access** > **Client Credentials**
2. **Generate New Client**
3. Save the **Client ID** and **Client Secret**

#### Step 3: Add to adapter-config.json

```json
{
  "default_provider": {
    "provider": "neuralgate",
    "model": "auto",
    "endpoint_url": "https://neuralgateway.yourcompany.com",
    "client_id": "<client-id-from-step-2>",
    "client_secret": "<client-secret-from-step-2>"
  }
}
```

### Credentials Per Application

| Application | NeuralGateway Project | client_id | client_secret |
|-------------|----------------------|-----------|---------------|
| EHR | `ehr` | `ehr-client-xxx` | `ehr-secret-xxx` |
| School ERP | `school-erp` | `school-client-xxx` | `school-secret-xxx` |
| Data Backup | `data-backup` | `backup-client-xxx` | `backup-secret-xxx` |

> **Security:** These credentials are in `adapter-config.json` which is mounted read-only into the backend container. They are never sent to the frontend. The backend uses them server-side for the OAuth2 `client_credentials` flow to obtain access tokens.

---

## 6. Integrating into EHR

### 6.1 Frontend: Install Library in EHR Angular App

```bash
# In your EHR Angular project root
# Configure private registry (one-time)
echo "@mdp:registry=http://your-verdaccio-host:4873" >> .npmrc

# Install
npm install @mdp/ai-chat
```

### 6.2 Frontend: Wire Into EHR App

**`src/app/app.config.ts`:**
```typescript
import { ApplicationConfig } from '@angular/core';
import { provideRouter } from '@angular/router';
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http';
import { provideAnimations } from '@angular/platform-browser/animations';
import { provideAiChat } from '@mdp/ai-chat';
import { AuthService } from './core/auth/auth.service';
import { inject } from '@angular/core';
import { routes } from './app.routes';

export const appConfig: ApplicationConfig = {
  providers: [
    provideRouter(routes),
    provideHttpClient(withInterceptorsFromDi()),
    provideAnimations(),
    provideAiChat({
      apiBaseUrl: environment.aiChatBackendUrl,   // e.g. 'https://ehr-ai.hospital.local'
      authTokenFn: () => inject(AuthService).getAccessToken$(),
      appName: 'Smart Health Assistant',
      enableTools: true,
      enableActionCards: true,
    })
  ]
};
```

**`src/app/layouts/main-layout.component.ts`:**
```typescript
import { Component } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { AiSidebarComponent } from '@mdp/ai-chat';
import { AiProposedAction } from '@mdp/ai-chat';

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
  currentPageContext = 'Patient Dashboard';

  handleAiAction(action: AiProposedAction): void {
    // Handle actions the AI proposes (e.g., navigate to patient, open form)
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

**`src/app/pages/settings/settings.component.ts`:**
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

### 6.3 Backend: Create adapter-config-ehr.json

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
          "query": { "type": "string", "description": "Patient name, MRN, or date of birth" },
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
    }
  ]
}
```

### 6.4 Deploy EHR Backend

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
      - ehr-network    # Same network as ehr-api service

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

> **Key:** The `ehr-ai-backend` container must be on the same Docker network as `ehr-api` so tool execution URLs like `http://ehr-api:3000/api/patients/search` resolve correctly.

---

## 7. Integrating into School ERP

### 7.1 Frontend: Install & Configure

```bash
# In School ERP Angular project
echo "@mdp:registry=http://your-verdaccio-host:4873" >> .npmrc
npm install @mdp/ai-chat
```

**`app.config.ts`:**
```typescript
provideAiChat({
  apiBaseUrl: environment.aiChatBackendUrl,
  authTokenFn: () => inject(AuthService).getAccessToken$(),
  appName: 'School AI Assistant',
  enableTools: true,
  enableActionCards: true,
})
```

### 7.2 Backend: adapter-config-school.json

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

### 7.3 Deploy

```yaml
# docker-compose.school.yml
services:
  school-ai-backend:
    image: your-registry/ai-chat-backend:1.0.0
    environment:
      DATABASE_URL: postgres://ai_user:${DB_PASSWORD}@postgres:5432/school_ai_db?sslmode=disable
      AI_ENCRYPTION_KEY: ${AI_ENCRYPTION_KEY}
      ADAPTER_CONFIG_PATH: /config/adapter-config.json
    volumes:
      - ./adapter-config-school.json:/config/adapter-config.json:ro
    networks:
      - school-network
```

---

## 8. Integrating into Data Backup & Recovery

### 8.1 Frontend: Install & Configure

```bash
echo "@mdp:registry=http://your-verdaccio-host:4873" >> .npmrc
npm install @mdp/ai-chat
```

**`app.config.ts`:**
```typescript
provideAiChat({
  apiBaseUrl: environment.aiChatBackendUrl,
  authTokenFn: () => inject(AuthService).getAccessToken$(),
  appName: 'Backup & Recovery Assistant',
  enableTools: true,
  enableActionCards: true,
})
```

### 8.2 Backend: adapter-config-backup.json

```json
{
  "domain": "backup",
  "display_name": "Backup & Recovery Assistant",
  "system_prompt": "You are an AI assistant for the Data Backup & Recovery platform. You help system administrators manage backup policies, monitor backup jobs, restore data, and troubleshoot backup failures. Be precise with technical details. Always recommend confirming destructive operations like restores and policy deletions.",
  "default_provider": {
    "provider": "neuralgate",
    "model": "auto",
    "endpoint_url": "https://neuralgateway.ops-infra.local",
    "client_id": "backup-ai-client-ghi789",
    "client_secret": "backup-ai-secret-rst654"
  },
  "tools": [
    {
      "name": "list_backup_policies",
      "description": "List all backup policies or filter by server name",
      "parameters": {
        "type": "object",
        "properties": {
          "server_name": { "type": "string" },
          "status": { "type": "string", "enum": ["active", "paused", "failed"] }
        }
      },
      "execution": {
        "method": "GET",
        "url": "http://backup-api:5000/api/policies"
      }
    },
    {
      "name": "create_backup_policy",
      "description": "Create a new backup policy for a server or database",
      "parameters": {
        "type": "object",
        "properties": {
          "name": { "type": "string" },
          "target": { "type": "string" },
          "schedule": { "type": "string", "description": "Cron expression" },
          "retention_days": { "type": "integer", "default": 30 },
          "backup_type": { "type": "string", "enum": ["full", "incremental", "differential"] }
        },
        "required": ["name", "target", "schedule"]
      },
      "requires_confirmation": true,
      "execution": {
        "method": "POST",
        "url": "http://backup-api:5000/api/policies"
      }
    },
    {
      "name": "get_backup_status",
      "description": "Get the status and history of backup jobs for a server",
      "parameters": {
        "type": "object",
        "properties": {
          "server_name": { "type": "string" },
          "last_n": { "type": "integer", "default": 5 }
        },
        "required": ["server_name"]
      },
      "execution": {
        "method": "GET",
        "url": "http://backup-api:5000/api/jobs"
      }
    },
    {
      "name": "trigger_backup",
      "description": "Trigger an immediate backup for a server or policy",
      "parameters": {
        "type": "object",
        "properties": {
          "policy_id": { "type": "string" },
          "backup_type": { "type": "string", "enum": ["full", "incremental"] }
        },
        "required": ["policy_id"]
      },
      "requires_confirmation": true,
      "execution": {
        "method": "POST",
        "url": "http://backup-api:5000/api/jobs/trigger",
        "timeout_ms": 15000
      }
    },
    {
      "name": "restore_backup",
      "description": "Restore a server or database from a specific backup snapshot",
      "parameters": {
        "type": "object",
        "properties": {
          "backup_id": { "type": "string" },
          "target": { "type": "string" },
          "overwrite": { "type": "boolean", "default": false }
        },
        "required": ["backup_id"]
      },
      "requires_confirmation": true,
      "execution": {
        "method": "POST",
        "url": "http://backup-api:5000/api/restore",
        "timeout_ms": 30000
      }
    }
  ]
}
```

### 8.3 Deploy

```yaml
# docker-compose.backup.yml
services:
  backup-ai-backend:
    image: your-registry/ai-chat-backend:1.0.0
    environment:
      DATABASE_URL: postgres://ai_user:${DB_PASSWORD}@postgres:5432/backup_ai_db?sslmode=disable
      AI_ENCRYPTION_KEY: ${AI_ENCRYPTION_KEY}
      ADAPTER_CONFIG_PATH: /config/adapter-config.json
    volumes:
      - ./adapter-config-backup.json:/config/adapter-config.json:ro
    networks:
      - backup-network
```

---

## 9. Adapter Config Reference

### How Provider Resolution Works

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

### Tool URL Interpolation

Tool execution URLs support `{param}` placeholders:

```json
{ "url": "http://ehr-api:3000/api/patients/{patient_id}/vitals" }
```

If AI calls with `{"patient_id": "P-12345"}`, request goes to:
`GET http://ehr-api:3000/api/patients/P-12345/vitals`

- **GET:** Non-URL arguments become query parameters
- **POST/PUT/DELETE:** Arguments sent as JSON body
- `X-Tenant-ID` and `X-User-ID` headers are automatically injected

### Confirmation Flow

Tools with `"requires_confirmation": true` trigger an SSE `tool_confirm` event. The frontend shows an Approve/Dismiss card. The backend blocks until the user responds via `POST /api/v1/ai/chat/confirm` (5-minute timeout).

### Audit Logging

Every tool execution is logged to `ai_tool_executions` with tenant/user ID, tool name, arguments, result, status, confirmation flag, and duration. This table supports HIPAA/FERPA compliance auditing.

---

## 10. Deployment Checklist

### Per Application (EHR, School ERP, etc.)

**NeuralGateway:**
- [ ] Create project in NeuralGateway
- [ ] Assign model to project
- [ ] Generate client credentials (client_id + client_secret)

**Backend:**
- [ ] Create PostgreSQL database
- [ ] Run migrations (000001 + 000002)
- [ ] Generate `AI_ENCRYPTION_KEY` (`openssl rand -hex 32`)
- [ ] Create `adapter-config.json` with NeuralGateway credentials and tools
- [ ] Tool URLs point to the application's internal API (same Docker network)
- [ ] Deploy `ai-chat-backend` container with `ADAPTER_CONFIG_PATH` mounted read-only
- [ ] Verify startup log: `Domain adapter loaded: ehr (Smart Health Assistant) with 5 tools`

**Frontend:**
- [ ] Add `.npmrc` pointing to private registry
- [ ] `npm install @mdp/ai-chat`
- [ ] Configure `provideAiChat()` with backend URL and auth token function
- [ ] Add `<mdp-ai-sidebar>` to main layout
- [ ] Add `<mdp-ai-settings>` to settings page (for BYOK)
- [ ] Wire `(actionRequested)` output if needed

**Networking:**
- [ ] Backend can reach NeuralGateway endpoint URL
- [ ] Backend can reach tool execution URLs (internal app APIs)
- [ ] Frontend can reach backend (CORS configured)

**Security:**
- [ ] Encryption key is NOT the dev default
- [ ] `adapter-config.json` mounted read-only (`:ro`)
- [ ] NeuralGateway credentials never exposed to frontend
- [ ] HTTPS in production
