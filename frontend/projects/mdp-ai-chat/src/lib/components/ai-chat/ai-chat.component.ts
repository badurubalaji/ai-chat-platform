import { Component, ChangeDetectionStrategy, EventEmitter, Input, Output, ElementRef, ViewChild, signal, inject, OnChanges, SimpleChanges } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { AiMessageComponent } from '../ai-message/ai-message.component';
import { AiEmptyStateComponent } from '../ai-empty-state/ai-empty-state.component';
import { AiChatService } from '../../services/ai-chat.service';
import { AiContextService } from '../../services/ai-context.service';
import { AiMessage, AiProposedAction, AiToolConfirmation } from '../../models/ai-chat.model';

@Component({
  selector: 'mdp-ai-chat',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    CommonModule,
    FormsModule,
    MatButtonModule,
    MatIconModule,
    MatInputModule,
    MatProgressSpinnerModule,
    AiMessageComponent,
    AiEmptyStateComponent
  ],
  template: `
    <div class="chat-messages" #scrollContainer>
      @if (messages().length === 0 && !isLoading()) {
        <div class="welcome-text">
          <mat-icon class="welcome-icon">smart_toy</mat-icon>
          <h3>How can I help you?</h3>
          <p>Ask me anything about your system.</p>
        </div>
      }

      @for (msg of messages(); track msg.id) {
        <mdp-ai-message
          [message]="msg"
          (actionRequested)="onAction($event)"
          (confirmTool)="onConfirmTool($event)"
          (dismissTool)="onDismissTool($event)">
        </mdp-ai-message>
      }
      
      @if (isTyping()) {
        <div class="typing-indicator">
          <span>.</span><span>.</span><span>.</span>
        </div>
      }
    </div>

    <div class="chat-input">
      <mat-form-field appearance="outline" class="full-width">
        <input matInput 
               [(ngModel)]="userInput" 
               (keyup.enter)="sendMessage()" 
               placeholder="Ask AI..." 
               [disabled]="isLoading()">
        <button mat-icon-button matSuffix (click)="sendMessage()" [disabled]="!userInput.trim() || isLoading()">
          <mat-icon>send</mat-icon>
        </button>
      </mat-form-field>
    </div>
  `,
  styles: [`
    :host {
      display: flex;
      flex-direction: column;
      height: 100%;
    }

    .chat-messages {
      flex: 1;
      overflow-y: auto;
      padding: 16px;
      display: flex;
      flex-direction: column;
      gap: 12px;
    }

    .welcome-text {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      flex: 1;
      text-align: center;
      color: var(--mat-sys-on-surface-variant, #49454f);
      h3 { margin: 12px 0 4px; }
      p { margin: 0; font-size: 0.875rem; }
    }

    .welcome-icon {
      font-size: 48px;
      width: 48px;
      height: 48px;
      color: var(--mat-sys-primary, #6750a4);
    }

    .chat-input {
      padding: 16px;
      border-top: 1px solid var(--ai-sidebar-border, #e0e0e0);
      background: var(--ai-sidebar-bg, #fff);
    }

    .full-width {
      width: 100%;
    }

    .typing-indicator {
      padding: 8px 16px;
      color: var(--ai-typing-dot, #757575);
      font-size: 24px;
      line-height: 10px;
      
      span {
        animation: blink 1.4s infinite both;
        &:nth-child(2) { animation-delay: 0.2s; }
        &:nth-child(3) { animation-delay: 0.4s; }
      }
    }

    @keyframes blink {
      0% { opacity: 0.2; }
      20% { opacity: 1; }
      100% { opacity: 0.2; }
    }
  `]
})
export class AiChatComponent implements OnChanges {
  @Input() conversationId: string | null = null;
  @Output() actionRequested = new EventEmitter<AiProposedAction>();

  @ViewChild('scrollContainer') private scrollContainer!: ElementRef;

  messages = signal<AiMessage[]>([]);
  isTyping = signal(false);
  isLoading = signal(false);
  userInput = '';

  private chatService = inject(AiChatService);
  private contextService = inject(AiContextService);

  ngOnChanges(changes: SimpleChanges): void {
    if (changes['conversationId'] && this.conversationId) {
      this.loadConversation(this.conversationId);
    }
  }

  private loadConversation(id: string): void {
    this.chatService.getConversation(id).subscribe({
      next: (data) => {
        if (data.messages) {
          this.messages.set(data.messages);
          this.scrollToBottom();
        }
      },
      error: (err) => console.error('Failed to load conversation:', err)
    });
  }

  sendMessage(): void {
    if (!this.userInput.trim() || this.isLoading()) return;

    const content = this.userInput;
    this.userInput = '';
    this.isLoading.set(true);
    this.isTyping.set(true);

    // Optimistic UI update
    const userMsg: AiMessage = {
      id: crypto.randomUUID(),
      role: 'user',
      content,
      created_at: new Date().toISOString()
    };
    this.messages.update(msgs => [...msgs, userMsg]);
    this.scrollToBottom();

    // Include page context if available
    const context = this.contextService.getContext();

    this.chatService.sendMessage(this.conversationId, content, context).subscribe({
      next: (chunk) => {
        this.isTyping.set(false);
        if (chunk.tool_confirm) {
          this.addConfirmationMessage(chunk.tool_confirm);
        } else if (chunk.tool_call) {
          this.handleToolCallEvent(chunk.tool_call.tool, chunk.tool_call.status);
        } else {
          this.updateAssistantMessage(chunk.content, chunk.done);
        }
        // Update conversation ID if returned in done event
        if (chunk.done && (chunk as unknown as Record<string, unknown>)['conversation_id']) {
          this.conversationId = (chunk as unknown as Record<string, unknown>)['conversation_id'] as string;
        }
      },
      error: (err) => {
        this.isTyping.set(false);
        this.isLoading.set(false);
        // Extract meaningful error message
        let errorMsg = 'Failed to get response. Please try again.';
        if (err?.message) {
          let msg = err.message;
          // Try to extract message from embedded JSON in error string
          const jsonMatch = msg.match(/\{[\s\S]*\}/);
          if (jsonMatch) {
            try {
              const parsed = JSON.parse(jsonMatch[0]);
              msg = parsed?.error?.message || parsed?.message || msg;
            } catch { /* keep original msg */ }
          }
          // Map common errors to user-friendly messages
          if (msg.includes('credit balance') || msg.includes('insufficient_quota') || msg.includes('exceeded your current quota')) {
            errorMsg = 'API credit balance is too low. Please check your plan and billing details.';
          } else if (msg.includes('invalid') && msg.includes('key')) {
            errorMsg = 'Invalid API key. Please update your API key in Settings.';
          } else {
            errorMsg = msg;
          }
        }
        this.messages.update(msgs => [...msgs, {
          id: crypto.randomUUID(),
          role: 'system',
          content: `⚠️ ${errorMsg}`,
          created_at: new Date().toISOString()
        }]);
        this.scrollToBottom();
        console.error('Chat error:', err);
      },
      complete: () => {
        this.isLoading.set(false);
      }
    });
  }

  private addConfirmationMessage(confirmation: AiToolConfirmation): void {
    this.messages.update(msgs => [...msgs, {
      id: crypto.randomUUID(),
      role: 'tool_call' as const,
      content: confirmation.tool,
      metadata: {
        tool_name: confirmation.tool,
        tool_status: 'executing' as const,
        tool_confirmation: confirmation
      },
      created_at: new Date().toISOString()
    }]);
    this.scrollToBottom();
  }

  private handleToolCallEvent(toolName: string, status: string): void {
    this.messages.update(msgs => {
      // Update existing tool_call message if present (e.g., after confirmation)
      const idx = msgs.findIndex(m => m.role === 'tool_call' && m.metadata?.tool_name === toolName && m.metadata?.tool_status !== 'complete' && m.metadata?.tool_status !== 'error');
      if (idx >= 0) {
        const updated = [...msgs];
        updated[idx] = {
          ...updated[idx],
          metadata: {
            ...updated[idx].metadata,
            tool_status: status as 'executing' | 'complete' | 'error',
            tool_confirmation: undefined // clear confirmation card
          }
        };
        return updated;
      }
      // No existing message — add new
      return [...msgs, {
        id: crypto.randomUUID(),
        role: 'tool_call' as const,
        content: toolName,
        metadata: { tool_name: toolName, tool_status: status as 'executing' | 'complete' | 'error' },
        created_at: new Date().toISOString()
      }];
    });
    this.scrollToBottom();
  }

  onConfirmTool(confirmation: AiToolConfirmation): void {
    this.chatService.sendConfirmation(confirmation.confirmation_id, true).subscribe();
    // Update the message to show executing state
    this.messages.update(msgs => msgs.map(m =>
      m.metadata?.tool_confirmation?.confirmation_id === confirmation.confirmation_id
        ? { ...m, metadata: { ...m.metadata, tool_status: 'executing' as const, tool_confirmation: undefined } }
        : m
    ));
  }

  onDismissTool(confirmation: AiToolConfirmation): void {
    this.chatService.sendConfirmation(confirmation.confirmation_id, false).subscribe();
    // Update the message to show cancelled state
    this.messages.update(msgs => msgs.map(m =>
      m.metadata?.tool_confirmation?.confirmation_id === confirmation.confirmation_id
        ? { ...m, metadata: { ...m.metadata, tool_status: 'cancelled' as const, tool_confirmation: undefined } }
        : m
    ));
  }

  private updateAssistantMessage(content: string, done: boolean): void {
    this.messages.update(msgs => {
      const lastMsg = msgs[msgs.length - 1];
      if (lastMsg && lastMsg.role === 'assistant' && !done) {
        // Append to existing
        return [
          ...msgs.slice(0, -1),
          { ...lastMsg, content: lastMsg.content + content }
        ];
      } else if (content) {
        // New assistant message start
        return [...msgs, {
          id: crypto.randomUUID(),
          role: 'assistant' as const,
          content,
          created_at: new Date().toISOString()
        }];
      }
      return msgs;
    });
    this.scrollToBottom();
  }

  private scrollToBottom(): void {
    setTimeout(() => {
      if (this.scrollContainer) {
        this.scrollContainer.nativeElement.scrollTop = this.scrollContainer.nativeElement.scrollHeight;
      }
    });
  }

  onAction(action: AiProposedAction): void {
    this.actionRequested.emit(action);
  }
}
