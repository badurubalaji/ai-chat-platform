import { Component, ChangeDetectionStrategy, EventEmitter, Output, signal, inject, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatSidenavModule } from '@angular/material/sidenav';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatListModule } from '@angular/material/list';
import { MatTooltipModule } from '@angular/material/tooltip';
import { AiChatComponent } from '../ai-chat/ai-chat.component';
import { AiProposedAction, AiConversation } from '../../models/ai-chat.model';
import { AiChatService } from '../../services/ai-chat.service';
import { AiContextService } from '../../services/ai-context.service';

@Component({
  selector: 'mdp-ai-sidebar',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    CommonModule,
    MatSidenavModule,
    MatButtonModule,
    MatIconModule,
    MatListModule,
    MatTooltipModule,
    AiChatComponent
  ],
  template: `
    <div class="ai-sidebar-container" [style.width.px]="sidebarWidth()">
      <div class="resize-handle" (mousedown)="startResize($event)"></div>
      
      <div class="sidebar-header">
        <div class="header-left">
          <h2>AI Assistant</h2>
          @if (contextService.hasContext()) {
            <span class="context-badge" [matTooltip]="'Context: ' + contextService.currentPage()">
              <mat-icon>info</mat-icon>
            </span>
          }
        </div>
        <div class="header-actions">
          <button mat-icon-button (click)="toggleHistory()" matTooltip="History">
            <mat-icon>history</mat-icon>
          </button>
          <button mat-icon-button (click)="createNewChat()" matTooltip="New Chat">
            <mat-icon>add</mat-icon>
          </button>
          <button mat-icon-button (click)="closeRequested.emit()" matTooltip="Close">
            <mat-icon>close</mat-icon>
          </button>
        </div>
      </div>

      <div class="sidebar-content">
        @if (showHistory()) {
          <mat-nav-list>
            @for (chat of conversations(); track chat.id) {
              <a mat-list-item (click)="selectChat(chat)">
                <span matListItemTitle>{{ chat.title || 'New Chat' }}</span>
                <span matListItemLine>{{ chat.updated_at | date:'short' }}</span>
              </a>
            }
            @empty {
              <div class="empty-history">
                <mat-icon>chat_bubble_outline</mat-icon>
                <p>No conversations yet</p>
              </div>
            }
          </mat-nav-list>
        } @else {
          <mdp-ai-chat 
            [conversationId]="currentConversationId()" 
            (actionRequested)="onAction($event)">
          </mdp-ai-chat>
        }
      </div>
    </div>
  `,
  styles: [`
    :host {
      display: block;
      height: 100%;
      background: var(--ai-sidebar-bg, #fff);
      border-left: 1px solid var(--ai-sidebar-border, #e0e0e0);
      position: relative;
    }

    .ai-sidebar-container {
      display: flex;
      flex-direction: column;
      height: 100%;
      min-width: 300px;
      max-width: 600px;
    }

    .resize-handle {
      position: absolute;
      left: 0;
      top: 0;
      bottom: 0;
      width: 4px;
      cursor: col-resize;
      z-index: 10;
      &:hover { background: var(--color-primary, #1976d2); }
    }

    .sidebar-header {
      padding: 8px 8px 8px 16px;
      display: flex;
      justify-content: space-between;
      align-items: center;
      border-bottom: 1px solid var(--ai-sidebar-border, #e0e0e0);
      gap: 8px;
      min-height: 48px;
    }

    .header-left {
      display: flex;
      align-items: center;
      gap: 6px;
      flex: 1;
      min-width: 0;

      h2 { margin: 0; font-size: 1.1rem; white-space: nowrap; }
    }

    .header-actions {
      display: flex;
      align-items: center;
      flex-shrink: 0;
    }

    .context-badge {
      display: inline-flex;
      align-items: center;
      font-size: 0.75rem;
      color: var(--mat-sys-primary, #6750a4);
      mat-icon {
        font-size: 16px;
        width: 16px;
        height: 16px;
      }
    }

    .sidebar-content {
      flex: 1;
      overflow: hidden;
      display: flex;
      flex-direction: column;
    }

    mat-nav-list {
      flex: 1;
      overflow-y: auto;
    }

    .empty-history {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      padding: 48px 24px;
      color: #999;
      mat-icon { font-size: 48px; width: 48px; height: 48px; }
      p { margin-top: 12px; }
    }
  `]
})
export class AiSidebarComponent implements OnInit {
  @Output() actionRequested = new EventEmitter<AiProposedAction>();
  @Output() closeRequested = new EventEmitter<void>();

  readonly contextService = inject(AiContextService);
  private readonly chatService = inject(AiChatService);

  showHistory = signal(false);
  conversations = signal<AiConversation[]>([]);
  currentConversationId = signal<string | null>(null);
  sidebarWidth = signal(400);

  private isResizing = false;

  ngOnInit(): void {
    this.loadConversations();
  }

  private loadConversations(): void {
    this.chatService.getConversations().subscribe({
      next: (convos) => this.conversations.set(convos),
      error: (err) => console.error('Failed to load conversations:', err)
    });
  }

  toggleHistory(): void {
    this.showHistory.update(v => !v);
    if (this.showHistory()) {
      this.loadConversations();
    }
  }

  createNewChat(): void {
    this.currentConversationId.set(null);
    this.showHistory.set(false);
  }

  selectChat(chat: AiConversation): void {
    this.currentConversationId.set(chat.id);
    this.showHistory.set(false);
  }

  onAction(action: AiProposedAction): void {
    this.actionRequested.emit(action);
  }

  startResize(event: MouseEvent): void {
    event.preventDefault();
    this.isResizing = true;
    const startX = event.clientX;
    const startWidth = this.sidebarWidth();

    const onMouseMove = (e: MouseEvent) => {
      if (!this.isResizing) return;
      const delta = startX - e.clientX; // Moving left = wider
      const newWidth = Math.min(600, Math.max(300, startWidth + delta));
      this.sidebarWidth.set(newWidth);
    };

    const onMouseUp = () => {
      this.isResizing = false;
      document.removeEventListener('mousemove', onMouseMove);
      document.removeEventListener('mouseup', onMouseUp);
    };

    document.addEventListener('mousemove', onMouseMove);
    document.addEventListener('mouseup', onMouseUp);
  }
}
