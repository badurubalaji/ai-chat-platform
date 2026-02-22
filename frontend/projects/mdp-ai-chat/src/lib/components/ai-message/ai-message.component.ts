import { Component, ChangeDetectionStrategy, EventEmitter, Input, Output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatIconModule } from '@angular/material/icon';
import { MatButtonModule } from '@angular/material/button';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { AiActionCardComponent } from '../ai-action-card/ai-action-card.component';
import { MarkdownRenderPipe } from '../../pipes/markdown-render.pipe';
import { AiMessage, AiProposedAction, AiToolConfirmation } from '../../models/ai-chat.model';

@Component({
  selector: 'mdp-ai-message',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [CommonModule, MatIconModule, MatButtonModule, MatProgressSpinnerModule, AiActionCardComponent, MarkdownRenderPipe],
  template: `
    @if (message.role === 'tool_call' || message.role === 'tool_result') {
      <div class="tool-message" [class.tool-cancelled]="message.metadata?.tool_status === 'cancelled'" [class.tool-error]="message.metadata?.tool_status === 'error'">
        <mat-icon class="tool-icon">build</mat-icon>
        <span class="tool-name">{{ message.metadata?.tool_name || message.content }}</span>
        @if (message.metadata?.tool_status === 'executing' && !message.metadata?.tool_confirmation) {
          <mat-spinner diameter="16"></mat-spinner>
        } @else if (message.metadata?.tool_status === 'cancelled') {
          <mat-icon class="tool-status-icon cancelled">block</mat-icon>
          <span class="tool-status-text">Cancelled</span>
        } @else if (message.metadata?.tool_status === 'error') {
          <mat-icon class="tool-status-icon error">error</mat-icon>
        } @else if (message.metadata?.tool_status === 'complete') {
          <mat-icon class="tool-status-icon success">check_circle</mat-icon>
        }
      </div>

      @if (message.metadata?.tool_confirmation; as confirmation) {
        <div class="tool-confirm-card">
          <div class="confirm-header">
            <mat-icon>security</mat-icon>
            <span>Action requires confirmation</span>
          </div>
          <div class="confirm-body">
            <div class="confirm-tool-name">{{ confirmation.tool }}</div>
            <p class="confirm-description">{{ confirmation.description }}</p>
            @if (confirmParamKeys(confirmation).length > 0) {
              <div class="confirm-params">
                @for (key of confirmParamKeys(confirmation); track key) {
                  <div class="confirm-param">
                    <span class="param-key">{{ key }}:</span>
                    <span class="param-value">{{ confirmation.params[key] }}</span>
                  </div>
                }
              </div>
            }
          </div>
          <div class="confirm-actions">
            <button mat-button (click)="dismissTool.emit(confirmation)">
              <mat-icon>close</mat-icon> Dismiss
            </button>
            <button mat-flat-button color="primary" (click)="confirmToolFn.emit(confirmation)">
              <mat-icon>check</mat-icon> Approve
            </button>
          </div>
        </div>
      }
    } @else if (message.role === 'system') {
      <div class="system-message">
        <mat-icon>info</mat-icon>
        <span>{{ message.content }}</span>
      </div>
    } @else {
      <div class="message-container" [class.user]="message.role === 'user'" [class.assistant]="message.role === 'assistant'">
        <div class="avatar">
          <mat-icon>{{ message.role === 'user' ? 'person' : 'smart_toy' }}</mat-icon>
        </div>
        
        <div class="content">
          @if (message.role === 'assistant') {
            <div class="markdown-body" [innerHTML]="message.content | markdownRender"></div>
          } @else {
            <div class="markdown-body">{{ message.content }}</div>
          }
          
          @if (message.metadata?.action_card; as action) {
            <mdp-ai-action-card 
              [action]="action"
              (apply)="onAction($event)"
              (dismiss)="null">
            </mdp-ai-action-card>
          }
        </div>
      </div>
    }
  `,
  styles: [`
    .message-container {
      display: flex;
      gap: 12px;
      max-width: 85%;
      
      &.user {
        align-self: flex-end;
        flex-direction: row-reverse;
        
        .content {
          background: var(--ai-bubble-user-bg, #1976d2);
          color: var(--ai-bubble-user-text, #fff);
          border-radius: 12px 12px 0 12px;
        }
      }

      &.assistant {
        align-self: flex-start;
        
        .content {
          background: var(--ai-bubble-assistant-bg, #f5f5f5);
          color: var(--ai-bubble-assistant-text, #212121);
          border-radius: 12px 12px 12px 0;
        }
      }
    }

    .content {
      padding: 12px 16px;
      box-shadow: 0 1px 2px rgba(0,0,0,0.1);
      overflow-wrap: break-word;
    }

    .avatar {
      width: 32px;
      height: 32px;
      display: flex;
      align-items: center;
      justify-content: center;
      border-radius: 50%;
      background: #eee;
      color: #666;
      flex-shrink: 0;
    }

    .markdown-body {
      :host ::ng-deep {
        h1, h2, h3 { margin: 8px 0 4px; }
        pre { 
          background: rgba(0,0,0,0.05); 
          padding: 8px; 
          border-radius: 4px; 
          overflow-x: auto;
        }
        code { 
          background: rgba(0,0,0,0.05); 
          padding: 2px 4px; 
          border-radius: 2px;
          font-size: 0.85em;
        }
        table {
          border-collapse: collapse;
          margin: 8px 0;
          th, td { 
            border: 1px solid rgba(0,0,0,0.1); 
            padding: 4px 8px; 
          }
        }
        ul, ol { padding-left: 20px; }
        a { color: inherit; text-decoration: underline; }
      }
    }

    .tool-message {
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 8px 16px;
      border-radius: 8px;
      background: var(--mat-sys-surface-container, #f3edf7);
      color: var(--mat-sys-on-surface-variant, #49454f);
      font-size: 0.8125rem;
      align-self: center;
    }

    .tool-icon {
      font-size: 16px;
      width: 16px;
      height: 16px;
    }

    .tool-name {
      font-weight: 500;
    }

    .tool-status-icon {
      font-size: 16px;
      width: 16px;
      height: 16px;
      color: #388e3c;
    }

    .tool-cancelled {
      opacity: 0.6;
    }

    .tool-error {
      border-left: 3px solid #d32f2f;
    }

    .tool-status-icon.success { color: #388e3c; }
    .tool-status-icon.cancelled { color: #9e9e9e; }
    .tool-status-icon.error { color: #d32f2f; }

    .tool-status-text {
      font-size: 0.75rem;
      color: #9e9e9e;
    }

    .tool-confirm-card {
      margin: 8px 0;
      padding: 0;
      border: 1px solid var(--mat-sys-outline-variant, #cac4d0);
      border-left: 4px solid var(--mat-sys-primary, #6750a4);
      border-radius: 8px;
      background: var(--mat-sys-surface-container-low, #f7f2fa);
      align-self: center;
      max-width: 90%;
      animation: slideIn 0.3s ease-out;
    }

    @keyframes slideIn {
      from { opacity: 0; transform: translateY(8px); }
      to { opacity: 1; transform: translateY(0); }
    }

    .confirm-header {
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 12px 16px;
      border-bottom: 1px solid var(--mat-sys-outline-variant, #cac4d0);
      font-weight: 500;
      font-size: 0.875rem;
      color: var(--mat-sys-primary, #6750a4);
    }

    .confirm-header mat-icon {
      font-size: 20px;
      width: 20px;
      height: 20px;
    }

    .confirm-body {
      padding: 12px 16px;
    }

    .confirm-tool-name {
      font-weight: 600;
      font-size: 0.875rem;
      margin-bottom: 4px;
    }

    .confirm-description {
      font-size: 0.8125rem;
      color: var(--mat-sys-on-surface-variant, #49454f);
      margin: 0 0 8px;
    }

    .confirm-params {
      background: rgba(0, 0, 0, 0.04);
      border-radius: 4px;
      padding: 8px 12px;
      display: grid;
      grid-template-columns: auto 1fr;
      gap: 2px 12px;
    }

    .confirm-param .param-key {
      font-weight: 500;
      font-size: 0.8125rem;
    }

    .confirm-param .param-value {
      font-size: 0.8125rem;
      color: var(--mat-sys-on-surface-variant, #49454f);
    }

    .confirm-param {
      display: contents;
    }

    .confirm-actions {
      display: flex;
      justify-content: flex-end;
      gap: 8px;
      padding: 8px 16px;
      border-top: 1px solid var(--mat-sys-outline-variant, #cac4d0);
    }

    .system-message {
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 8px 16px;
      border-radius: 8px;
      background: #fff3e0;
      color: #e65100;
      font-size: 0.8125rem;
      align-self: center;
    }
  `]
})
export class AiMessageComponent {
  @Input({ required: true }) message!: AiMessage;
  @Output() actionRequested = new EventEmitter<AiProposedAction>();
  @Output() confirmTool = new EventEmitter<AiToolConfirmation>();
  @Output() dismissTool = new EventEmitter<AiToolConfirmation>();

  // Alias for template binding (avoid name collision with @Output)
  readonly confirmToolFn = this.confirmTool;

  onAction(action: AiProposedAction): void {
    this.actionRequested.emit(action);
  }

  confirmParamKeys(confirmation: AiToolConfirmation): string[] {
    return confirmation.params ? Object.keys(confirmation.params) : [];
  }
}
