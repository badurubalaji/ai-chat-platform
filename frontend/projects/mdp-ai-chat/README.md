# @mdp/ai-chat

> Embeddable AI Chat sidebar for Angular applications with streaming, tool calling, and multi-provider support.

[![npm version](https://img.shields.io/npm/v/@mdp/ai-chat)](https://npm.ashulabs.com/-/web/detail/@mdp/ai-chat)
[![Angular](https://img.shields.io/badge/Angular-21+-dd0031)](https://angular.dev)
[![License](https://img.shields.io/badge/license-private-blue)]()

---

## Table of Contents

- [Features](#features)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Components](#components)
  - [AiSidebarComponent](#aisidebarcomponent)
  - [AiSettingsComponent](#aisettingscomponent)
  - [AiUsageDashboardComponent](#aiusagedashboardcomponent)
  - [AiActionCardComponent](#aiactioncardcomponent)
  - [AiEmptyStateComponent](#aiemptystatecomponent)
- [Services](#services)
  - [AiChatService](#aichatservice)
  - [AiContextService](#aicontextservice)
  - [AiProviderConfigService](#aiproviderconfigservice)
  - [AiUsageService](#aiusageservice)
- [Pipes](#pipes)
- [Models & Types](#models--types)
- [Usage Examples](#usage-examples)
  - [Basic Sidebar Integration](#basic-sidebar-integration)
  - [Page Context Awareness](#page-context-awareness)
  - [Handling AI-Proposed Actions](#handling-ai-proposed-actions)
  - [Settings Page with BYOK](#settings-page-with-byok)
  - [Usage Dashboard](#usage-dashboard)
  - [Custom Chat Interface](#custom-chat-interface)
- [Theming & CSS Variables](#theming--css-variables)
- [Backend Requirements](#backend-requirements)
- [Changelog](#changelog)

---

## Features

- **SSE Streaming** — Real-time chat responses with Server-Sent Events
- **Multi-Provider** — Claude, OpenAI, Gemini, Ollama, NeuralGateway out of the box
- **Tool Calling** — Config-driven tool execution with confirmation flows for destructive actions
- **BYOK** — Users can bring their own API keys for cloud providers
- **Markdown Rendering** — Rich formatting for assistant responses (code blocks, tables, lists)
- **Conversation History** — Browse, resume, and delete past conversations
- **Resizable Sidebar** — Drag to resize (300px–600px) with persistent width
- **Page Context** — Pass current page context to AI for relevant responses
- **Action Cards** — AI can propose actions with parameter previews and approve/dismiss flow
- **Usage Dashboard** — Token consumption tracking with daily breakdown
- **Material Design** — Fully themed with Angular Material and CSS variable overrides
- **Standalone Components** — Tree-shakeable, import only what you need
- **OnPush Change Detection** — Optimized performance with Angular Signals

---

## Prerequisites

| Package | Version |
|---------|---------|
| `@angular/core` | ^21.1.0 |
| `@angular/common` | ^21.1.0 |
| `@angular/material` | ^21.1.0 |
| `@angular/cdk` | ^21.1.0 |

---

## Installation

### From Private Registry

Add the registry to your project's `.npmrc`:

```ini
# .npmrc
@mdp:registry=https://npm.ashulabs.com/
```

Then install:

```bash
npm install @mdp/ai-chat
```

### From Tarball (Offline)

```bash
npm install ./libs/mdp-ai-chat-1.0.1.tgz
```

---

## Quick Start

**1. Configure the provider in `app.config.ts`:**

```typescript
import { ApplicationConfig, inject } from '@angular/core';
import { provideRouter } from '@angular/router';
import { provideHttpClient } from '@angular/common/http';
import { provideAnimations } from '@angular/platform-browser/animations';
import { provideAiChat } from '@mdp/ai-chat';
import { AuthService } from './core/auth.service';
import { routes } from './app.routes';

export const appConfig: ApplicationConfig = {
  providers: [
    provideRouter(routes),
    provideHttpClient(),
    provideAnimations(),
    provideAiChat({
      apiBaseUrl: 'https://ai-backend.yourapp.com',
      authTokenFn: () => inject(AuthService).getAccessToken$(),
    })
  ]
};
```

**2. Add the sidebar to your layout:**

```typescript
import { Component } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { AiSidebarComponent } from '@mdp/ai-chat';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [RouterOutlet, AiSidebarComponent],
  template: `
    <div class="app-layout">
      <main><router-outlet /></main>
      <mdp-ai-sidebar (actionRequested)="onAction($event)" />
    </div>
  `,
  styles: [`
    .app-layout {
      display: flex;
      height: 100vh;
    }
    main { flex: 1; overflow: auto; }
  `]
})
export class AppComponent {
  onAction(action: any) {
    console.log('AI proposed action:', action);
  }
}
```

That's it. You now have a working AI chat sidebar.

---

## Configuration

### `provideAiChat(config: AiChatConfig)`

Call in `app.config.ts` providers array to configure the library.

```typescript
provideAiChat({
  apiBaseUrl: 'https://ai-backend.yourapp.com',
  authTokenFn: () => inject(AuthService).getAccessToken$(),
  appName: 'Smart Health Assistant',
  appDescription: 'AI assistant for the EHR system',
  enableTools: true,
  enableActionCards: true,
  sidebarWidth: 400,
  sidebarMinWidth: 300,
  sidebarMaxWidth: 600,
  markdownOptions: {
    enableSyntaxHighlighting: true,
  }
})
```

### AiChatConfig

| Property | Type | Default | Description |
|----------|------|---------|-------------|
| `apiBaseUrl` | `string` | **required** | Backend base URL (no trailing slash) |
| `authTokenFn` | `() => Observable<string>` | **required** | Returns JWT/Bearer token for API calls |
| `appName` | `string` | `'AI Assistant'` | Display name in the sidebar header |
| `appDescription` | `string` | — | Additional context sent to the AI model |
| `enableTools` | `boolean` | `true` | Enable tool calling and execution |
| `enableActionCards` | `boolean` | `true` | Show action proposal cards in chat |
| `sidebarWidth` | `number` | `400` | Initial sidebar width in pixels |
| `sidebarMinWidth` | `number` | `300` | Minimum sidebar width when resizing |
| `sidebarMaxWidth` | `number` | `600` | Maximum sidebar width when resizing |
| `markdownOptions` | `MarkdownRenderOptions` | — | Markdown rendering configuration |

---

## Components

### AiSidebarComponent

Full-featured AI chat sidebar with conversation history and resize handle.

```typescript
import { AiSidebarComponent } from '@mdp/ai-chat';
```

**Selector:** `<mdp-ai-sidebar>`

| Output | Type | Description |
|--------|------|-------------|
| `actionRequested` | `EventEmitter<AiProposedAction>` | Emitted when AI proposes an action the host app should handle |

```html
<mdp-ai-sidebar (actionRequested)="handleAction($event)" />
```

**Includes:** Chat input, message list with streaming, conversation history panel, new chat button, page context badge, drag-to-resize handle.

---

### AiSettingsComponent

Provider configuration panel for BYOK (Bring Your Own Key).

```typescript
import { AiSettingsComponent } from '@mdp/ai-chat';
```

**Selector:** `<mdp-ai-settings>`

```html
<mdp-ai-settings />
```

Allows users to:
- Select provider (Claude, OpenAI, Gemini, Ollama, Generic, NeuralGate)
- Enter API key (encrypted server-side with AES-256-GCM)
- Choose model from dropdown or enter custom model name
- Configure advanced settings (temperature, max tokens, system prompt prefix)
- Test connection before saving
- Delete configuration to revert to default provider

---

### AiUsageDashboardComponent

Token usage statistics and daily trends.

```typescript
import { AiUsageDashboardComponent } from '@mdp/ai-chat';
```

**Selector:** `<mdp-ai-usage-dashboard>`

```html
<mdp-ai-usage-dashboard />
```

Displays:
- Period toggle (7 / 30 / 90 days)
- Total input and output tokens
- Estimated cost
- Daily usage bar chart

---

### AiActionCardComponent

Displays a proposed action from the AI with parameter preview and approve/dismiss buttons.

```typescript
import { AiActionCardComponent } from '@mdp/ai-chat';
```

**Selector:** `<mdp-ai-action-card>`

| Input | Type | Description |
|-------|------|-------------|
| `action` | `InputSignal<AiProposedAction>` | The proposed action to display |

| Output | Type | Description |
|--------|------|-------------|
| `apply` | `OutputEmitterRef<AiProposedAction>` | User clicked "Review & Apply" |
| `dismiss` | `OutputEmitterRef<void>` | User clicked "Dismiss" |

```html
<mdp-ai-action-card
  [action]="proposedAction"
  (apply)="executeAction($event)"
  (dismiss)="ignoreAction()" />
```

---

### AiEmptyStateComponent

Placeholder shown when no AI provider is configured.

```typescript
import { AiEmptyStateComponent } from '@mdp/ai-chat';
```

**Selector:** `<mdp-ai-empty-state>`

---

## Services

### AiChatService

Core chat service for sending messages and managing conversations.

```typescript
import { AiChatService } from '@mdp/ai-chat';
```

| Method | Returns | Description |
|--------|---------|-------------|
| `sendMessage(conversationId, message, context?)` | `Observable<AiStreamChunk>` | Send a chat message and receive streaming response |
| `getConversations()` | `Observable<AiConversation[]>` | List all conversations for the current user |
| `getConversation(id)` | `Observable<{ conversation, messages }>` | Get a conversation with its message history |
| `createConversation()` | `Observable<AiConversation>` | Create a new empty conversation |
| `deleteConversation(id)` | `Observable<void>` | Delete a conversation |
| `sendConfirmation(confirmationId, approved)` | `Observable<{ status }>` | Approve or dismiss a tool confirmation |

#### Streaming Example

```typescript
private chatService = inject(AiChatService);

sendMessage(text: string) {
  this.chatService.sendMessage(this.conversationId, text).subscribe({
    next: (chunk) => {
      if (chunk.tool_confirm) {
        // Show confirmation card
        this.showConfirmation(chunk.tool_confirm);
      } else if (chunk.tool_call) {
        // Tool is executing
        console.log(`Tool ${chunk.tool_call.tool}: ${chunk.tool_call.status}`);
      } else if (chunk.done) {
        // Stream complete
        console.log('Usage:', chunk.usage);
      } else {
        // Append content
        this.response += chunk.content;
      }
    },
    error: (err) => console.error('Chat error:', err),
    complete: () => console.log('Stream ended')
  });
}
```

---

### AiContextService

Pass page context to the AI so it gives relevant responses.

```typescript
import { AiContextService } from '@mdp/ai-chat';
```

| Method / Property | Type | Description |
|-------------------|------|-------------|
| `setContext(ctx)` | `void` | Set current page context |
| `clearContext()` | `void` | Clear page context |
| `getContext()` | `PageContext \| null` | Get current context value |
| `context` | `Signal<PageContext \| null>` | Readonly signal of current context |
| `hasContext` | `Signal<boolean>` | Whether context is set |
| `currentPage` | `Signal<string>` | Current page name |

#### Example: Set Context on Route Change

```typescript
import { AiContextService } from '@mdp/ai-chat';

@Component({ ... })
export class PatientDetailComponent implements OnInit {
  private contextService = inject(AiContextService);
  private route = inject(ActivatedRoute);

  ngOnInit() {
    const patientId = this.route.snapshot.params['id'];
    this.contextService.setContext({
      page: 'Patient Detail',
      patient_id: patientId,
      section: 'vitals'
    });
  }

  ngOnDestroy() {
    this.contextService.clearContext();
  }
}
```

Now when the user asks "Show me this patient's lab results", the AI knows which patient they're looking at.

---

### AiProviderConfigService

Manage BYOK provider configurations.

```typescript
import { AiProviderConfigService } from '@mdp/ai-chat';
```

| Method | Returns | Description |
|--------|---------|-------------|
| `getConfig()` | `Observable<AiProviderConfig>` | Get current provider configuration |
| `saveConfig(config)` | `Observable<AiProviderConfig>` | Save/update provider configuration |
| `deleteConfig()` | `Observable<void>` | Delete configuration (revert to default) |
| `testConnection(config)` | `Observable<boolean>` | Test if the provider is reachable |
| `getModels(provider, apiKey, endpointUrl?)` | `Observable<string[]>` | List available models for a provider |

---

### AiUsageService

Retrieve token usage statistics.

```typescript
import { AiUsageService } from '@mdp/ai-chat';
```

| Method | Returns | Description |
|--------|---------|-------------|
| `getUsageStats(days?)` | `Observable<AiUsageStats>` | Get usage stats for the last N days (default: 30) |

---

## Pipes

### MarkdownRenderPipe

Renders markdown content to HTML. Used internally by `AiMessageComponent`.

```typescript
import { MarkdownRenderPipe } from '@mdp/ai-chat';
```

```html
<div [innerHTML]="markdownContent | markdownRender"></div>
```

---

## Models & Types

### AiProvider

```typescript
type AiProvider = 'claude' | 'openai' | 'gemini' | 'ollama' | 'generic' | 'neuralgate';
```

### AiConversation

```typescript
interface AiConversation {
  id: string;
  title?: string;
  created_at: string;
  updated_at: string;
  message_count?: number;
}
```

### AiMessage

```typescript
interface AiMessage {
  id: string;
  role: 'user' | 'assistant' | 'system' | 'tool_call' | 'tool_result';
  content: string;
  metadata?: AiMessageMetadata;
  created_at: string;
}
```

### AiMessageMetadata

```typescript
interface AiMessageMetadata {
  tool_name?: string;
  tool_status?: 'executing' | 'complete' | 'error' | 'cancelled';
  action_card?: AiProposedAction;
  tool_confirmation?: AiToolConfirmation;
  token_usage?: { input: number; output: number };
}
```

### AiProposedAction

```typescript
interface AiProposedAction {
  action_type: string;
  title: string;
  summary: string;
  params: Record<string, unknown>;
  requires_confirmation: boolean;
}
```

### AiToolConfirmation

```typescript
interface AiToolConfirmation {
  confirmation_id: string;
  tool: string;
  description: string;
  params: Record<string, unknown>;
}
```

### AiStreamChunk

```typescript
interface AiStreamChunk {
  content: string;
  done: boolean;
  tool_call?: { tool: string; status: string };
  tool_confirm?: AiToolConfirmation;
  usage?: { input_tokens: number; output_tokens: number };
}
```

### AiUsageStats

```typescript
interface AiUsageStats {
  daily: AiUsagePeriod[];
  total_input_tokens: number;
  total_output_tokens: number;
  conversation_count: number;
}

interface AiUsagePeriod {
  date: string;
  input_tokens: number;
  output_tokens: number;
  request_count: number;
}
```

### AiProviderConfig

```typescript
interface AiProviderConfig {
  id: string;
  tenant_id?: string;
  provider: AiProvider;
  model: string;
  apiKey?: string;
  endpoint_url?: string;
  settings: AiProviderSettings;
  enabled: boolean;
}

interface AiProviderSettings {
  temperature: number;       // 0–1, default 0.7
  max_tokens: number;        // default 4096
  system_prompt_prefix?: string;
}
```

---

## Usage Examples

### Basic Sidebar Integration

The simplest integration — add the sidebar to your app layout:

```typescript
import { Component } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { AiSidebarComponent } from '@mdp/ai-chat';

@Component({
  selector: 'app-layout',
  standalone: true,
  imports: [RouterOutlet, AiSidebarComponent],
  template: `
    <div class="layout">
      <nav><!-- your nav --></nav>
      <main><router-outlet /></main>
      <mdp-ai-sidebar />
    </div>
  `,
  styles: [`
    .layout {
      display: flex;
      height: 100vh;
    }
    main {
      flex: 1;
      overflow: auto;
    }
  `]
})
export class LayoutComponent {}
```

---

### Page Context Awareness

Make the AI aware of what the user is currently viewing:

```typescript
import { Component, inject, OnInit, OnDestroy } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { AiContextService } from '@mdp/ai-chat';

@Component({
  selector: 'app-student-detail',
  template: `<!-- student detail page -->`
})
export class StudentDetailComponent implements OnInit, OnDestroy {
  private contextService = inject(AiContextService);
  private route = inject(ActivatedRoute);

  ngOnInit() {
    this.contextService.setContext({
      page: 'Student Detail',
      student_id: this.route.snapshot.params['id'],
      section: 'grades'
    });
  }

  ngOnDestroy() {
    this.contextService.clearContext();
  }
}
```

User asks: *"What are this student's attendance records?"*
AI knows which student and can call the right tool with the correct student_id.

---

### Handling AI-Proposed Actions

When the AI proposes an action (e.g., "navigate to patient"), handle it in your app:

```typescript
import { Component } from '@angular/core';
import { Router } from '@angular/router';
import { AiSidebarComponent, AiProposedAction } from '@mdp/ai-chat';

@Component({
  imports: [AiSidebarComponent],
  template: `
    <mdp-ai-sidebar (actionRequested)="handleAction($event)" />
  `
})
export class LayoutComponent {
  private router = inject(Router);

  handleAction(action: AiProposedAction) {
    switch (action.action_type) {
      case 'view_patient':
        this.router.navigate(['/patients', action.params['patient_id']]);
        break;
      case 'create_backup':
        this.router.navigate(['/backups/new'], {
          queryParams: { server: action.params['target'] }
        });
        break;
      case 'enroll_student':
        this.router.navigate(['/students/enroll'], {
          queryParams: action.params
        });
        break;
      default:
        console.log('Unhandled action:', action);
    }
  }
}
```

---

### Settings Page with BYOK

Let users configure their own AI provider:

```typescript
import { Component } from '@angular/core';
import { AiSettingsComponent } from '@mdp/ai-chat';

@Component({
  selector: 'app-ai-settings-page',
  standalone: true,
  imports: [AiSettingsComponent],
  template: `
    <div class="settings-container">
      <h1>AI Provider Settings</h1>
      <p>
        Optionally configure your own AI provider.
        If not configured, the default in-house model will be used.
      </p>
      <mdp-ai-settings />
    </div>
  `,
  styles: [`
    .settings-container {
      max-width: 800px;
      margin: 24px auto;
      padding: 0 24px;
    }
  `]
})
export class AiSettingsPageComponent {}
```

---

### Usage Dashboard

Show token consumption on an admin page:

```typescript
import { Component } from '@angular/core';
import { AiUsageDashboardComponent } from '@mdp/ai-chat';

@Component({
  selector: 'app-admin-dashboard',
  standalone: true,
  imports: [AiUsageDashboardComponent],
  template: `
    <h1>AI Usage Overview</h1>
    <mdp-ai-usage-dashboard />
  `
})
export class AdminDashboardComponent {}
```

---

### Custom Chat Interface

Build your own chat UI using the service directly:

```typescript
import { Component, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { AiChatService, AiMessage, MarkdownRenderPipe } from '@mdp/ai-chat';

@Component({
  selector: 'app-custom-chat',
  standalone: true,
  imports: [FormsModule, MarkdownRenderPipe],
  template: `
    <div class="messages">
      @for (msg of messages(); track msg.id) {
        <div [class]="msg.role">
          @if (msg.role === 'assistant') {
            <div [innerHTML]="msg.content | markdownRender"></div>
          } @else {
            <p>{{ msg.content }}</p>
          }
        </div>
      }
    </div>
    <input [(ngModel)]="input" (keyup.enter)="send()" placeholder="Ask something..." />
  `
})
export class CustomChatComponent {
  private chatService = inject(AiChatService);
  messages = signal<AiMessage[]>([]);
  input = '';
  private conversationId: string | null = null;

  send() {
    if (!this.input.trim()) return;
    const text = this.input;
    this.input = '';

    // Add user message
    this.messages.update(msgs => [...msgs, {
      id: crypto.randomUUID(),
      role: 'user',
      content: text,
      created_at: new Date().toISOString()
    }]);

    // Stream response
    let assistantContent = '';
    this.chatService.sendMessage(this.conversationId, text).subscribe({
      next: (chunk) => {
        if (chunk.done) return;
        assistantContent += chunk.content;
        this.messages.update(msgs => {
          const last = msgs[msgs.length - 1];
          if (last?.role === 'assistant') {
            return [...msgs.slice(0, -1), { ...last, content: assistantContent }];
          }
          return [...msgs, {
            id: crypto.randomUUID(),
            role: 'assistant' as const,
            content: assistantContent,
            created_at: new Date().toISOString()
          }];
        });
      }
    });
  }
}
```

---

## Theming & CSS Variables

Override these CSS variables to match your application's theme:

```css
/* In your global styles or component host */
:root {
  /* Sidebar */
  --ai-sidebar-bg: #ffffff;
  --ai-sidebar-border: #e0e0e0;

  /* Chat bubbles */
  --ai-bubble-user-bg: #1976d2;
  --ai-bubble-user-text: #ffffff;
  --ai-bubble-assistant-bg: #f5f5f5;
  --ai-bubble-assistant-text: #212121;

  /* Typing indicator */
  --ai-typing-dot: #757575;
}

/* Dark theme example */
.dark-theme {
  --ai-sidebar-bg: #1e1e1e;
  --ai-sidebar-border: #333333;
  --ai-bubble-user-bg: #1565c0;
  --ai-bubble-user-text: #ffffff;
  --ai-bubble-assistant-bg: #2d2d2d;
  --ai-bubble-assistant-text: #e0e0e0;
  --ai-typing-dot: #999999;
}
```

The library also respects Angular Material's theme tokens:
- `--mat-sys-primary`
- `--mat-sys-on-surface-variant`
- `--mat-sys-surface-container`
- `--mat-sys-surface-container-low`
- `--mat-sys-outline-variant`

---

## Backend Requirements

This library requires the **ai-chat-platform backend** running with these API endpoints:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/ai/chat` | POST | Send message (SSE streaming response) |
| `/api/v1/ai/chat/confirm` | POST | Approve/dismiss tool confirmation |
| `/api/v1/ai/conversations` | GET | List conversations |
| `/api/v1/ai/conversations` | POST | Create conversation |
| `/api/v1/ai/conversations/:id` | GET | Get conversation with messages |
| `/api/v1/ai/conversations/:id` | DELETE | Delete conversation |
| `/api/v1/ai/config` | GET | Get provider config |
| `/api/v1/ai/config` | POST | Save provider config |
| `/api/v1/ai/config` | DELETE | Delete provider config |
| `/api/v1/ai/config/test` | POST | Test provider connection |
| `/api/v1/ai/config/models` | POST | List available models |
| `/api/v1/ai/usage` | GET | Get usage statistics |

All endpoints require `Authorization: Bearer <token>` header, provided automatically via `authTokenFn`.

See the [Integration Guide](https://npm.ashulabs.com/-/web/detail/@mdp/ai-chat) for full backend setup instructions.

---

## Changelog

### 1.0.1 (2026-02-21)
- Updated package description and README

### 1.0.0 (2026-02-21)
- Initial release
- Sidebar component with streaming chat
- Tool calling with confirmation flows
- BYOK provider settings
- Usage dashboard
- Page context service
- Markdown rendering
- 6 providers: Claude, OpenAI, Gemini, Ollama, Generic, NeuralGateway
