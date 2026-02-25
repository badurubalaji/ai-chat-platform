import { Component, ChangeDetectionStrategy, EventEmitter, Input, Output, ElementRef, ViewChild, signal, inject, OnChanges, SimpleChanges } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { CdkTextareaAutosize, TextFieldModule } from '@angular/cdk/text-field';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatChipsModule } from '@angular/material/chips';
import { AiMessageComponent } from '../ai-message/ai-message.component';
import { AiEmptyStateComponent } from '../ai-empty-state/ai-empty-state.component';
import { AiChatService } from '../../services/ai-chat.service';
import { AiContextService } from '../../services/ai-context.service';
import { AiMessage, AiProposedAction, AiToolConfirmation } from '../../models/ai-chat.model';
import { Subscription } from 'rxjs';

const ACCEPTED_TYPES = 'image/png,image/jpeg,image/webp,application/pdf,audio/mpeg,audio/wav,video/mp4';
const MAX_FILE_SIZE = 20 * 1024 * 1024; // 20MB
const MAX_FILES = 5;

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
    MatChipsModule,
    TextFieldModule,
    AiMessageComponent,
    AiEmptyStateComponent
  ],
  template: `
    <div class="chat-messages" #scrollContainer
         (dragover)="onDragOver($event)"
         (dragleave)="onDragLeave($event)"
         (drop)="onDrop($event)">
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

      @if (isDragging()) {
        <div class="drop-overlay">
          <mat-icon>cloud_upload</mat-icon>
          <span>Drop files here</span>
        </div>
      }
    </div>

    @if (selectedFiles().length > 0) {
      <div class="file-preview">
        @for (file of selectedFiles(); track file.name) {
          <div class="file-chip">
            <mat-icon class="file-icon">{{ getFileIcon(file) }}</mat-icon>
            <span class="file-name">{{ file.name }}</span>
            <span class="file-size">{{ formatFileSize(file.size) }}</span>
            <button mat-icon-button class="file-remove" (click)="removeFile(file)">
              <mat-icon>close</mat-icon>
            </button>
          </div>
        }
      </div>
    }

    <div class="chat-input">
      <input type="file" #fileInput
             [accept]="acceptedTypes"
             multiple
             (change)="onFilesSelected($event)"
             style="display: none">
      <mat-form-field appearance="outline" class="full-width">
        <textarea matInput
               cdkTextareaAutosize
               cdkAutosizeMinRows="1"
               cdkAutosizeMaxRows="5"
               [(ngModel)]="userInput"
               (keydown.enter)="onEnterKey($event)"
               placeholder="Ask AI..."
               [disabled]="isLoading()"></textarea>
        <div matSuffix class="input-actions">
          <button mat-icon-button (click)="fileInput.click()" [disabled]="isLoading()" matTooltip="Attach files">
            <mat-icon>attach_file</mat-icon>
          </button>
          @if (isLoading()) {
            <button mat-icon-button (click)="cancelRequest()" class="cancel-btn" matTooltip="Stop generating">
              <mat-icon>stop_circle</mat-icon>
            </button>
          } @else {
            <button mat-icon-button (click)="sendMessage()" [disabled]="!userInput.trim() && selectedFiles().length === 0">
              <mat-icon>send</mat-icon>
            </button>
          }
        </div>
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
      position: relative;
      background: var(--ai-chat-bg, var(--ai-sidebar-bg, #fff));
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

      textarea {
        resize: none;
        line-height: 1.4;
      }
    }

    .input-actions {
      display: flex;
      align-items: center;
      gap: 0;
    }

    .cancel-btn {
      color: var(--mat-sys-error, #b3261e);
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

    .file-preview {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;
      padding: 8px 16px 0;
      border-top: 1px solid var(--ai-sidebar-border, #e0e0e0);
      background: var(--ai-sidebar-bg, #fff);
    }

    .file-chip {
      display: flex;
      align-items: center;
      gap: 6px;
      padding: 4px 8px;
      border-radius: 16px;
      background: var(--mat-sys-surface-container, #f0edf6);
      font-size: 0.75rem;
      max-width: 200px;
    }

    .file-icon {
      font-size: 16px;
      width: 16px;
      height: 16px;
      color: var(--mat-sys-primary, #6750a4);
    }

    .file-name {
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
      flex: 1;
    }

    .file-size {
      color: var(--mat-sys-on-surface-variant, #49454f);
      white-space: nowrap;
    }

    .file-remove {
      width: 20px!important;
      height: 20px!important;
      line-height: 20px!important;
      
      mat-icon {
        font-size: 14px;
        width: 14px;
        height: 14px;
      }
    }

    .drop-overlay {
      position: absolute;
      inset: 0;
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      gap: 8px;
      background: rgba(103, 80, 164, 0.1);
      border: 2px dashed var(--mat-sys-primary, #6750a4);
      border-radius: 12px;
      color: var(--mat-sys-primary, #6750a4);
      font-size: 1rem;
      z-index: 10;
      pointer-events: none;

      mat-icon {
        font-size: 48px;
        width: 48px;
        height: 48px;
      }
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
  isDragging = signal(false);
  selectedFiles = signal<File[]>([]);
  userInput = '';
  readonly acceptedTypes = ACCEPTED_TYPES;

  private chatService = inject(AiChatService);
  private activeSubscription: Subscription | null = null;
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

  // -- File handling --

  onFilesSelected(event: Event): void {
    const input = event.target as HTMLInputElement;
    if (input.files) {
      this.addFiles(Array.from(input.files));
      input.value = ''; // Reset so same file can be re-selected
    }
  }

  onDragOver(event: DragEvent): void {
    event.preventDefault();
    event.stopPropagation();
    this.isDragging.set(true);
  }

  onDragLeave(event: DragEvent): void {
    event.preventDefault();
    event.stopPropagation();
    this.isDragging.set(false);
  }

  onDrop(event: DragEvent): void {
    event.preventDefault();
    event.stopPropagation();
    this.isDragging.set(false);
    if (event.dataTransfer?.files) {
      this.addFiles(Array.from(event.dataTransfer.files));
    }
  }

  private addFiles(files: File[]): void {
    const current = this.selectedFiles();
    const remaining = MAX_FILES - current.length;
    const valid = files
      .filter(f => f.size <= MAX_FILE_SIZE)
      .filter(f => ACCEPTED_TYPES.split(',').some(t => f.type === t || f.name.endsWith(t.replace('*', ''))))
      .slice(0, remaining);
    this.selectedFiles.set([...current, ...valid]);
  }

  removeFile(file: File): void {
    this.selectedFiles.update(files => files.filter(f => f !== file));
  }

  getFileIcon(file: File): string {
    if (file.type.startsWith('image/')) return 'image';
    if (file.type === 'application/pdf') return 'picture_as_pdf';
    if (file.type.startsWith('audio/')) return 'audiotrack';
    if (file.type.startsWith('video/')) return 'videocam';
    return 'insert_drive_file';
  }

  formatFileSize(bytes: number): string {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
    return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
  }

  // -- Messaging --

  cancelRequest(): void {
    if (this.activeSubscription) {
      this.activeSubscription.unsubscribe();
      this.activeSubscription = null;
    }
    this.chatService.cancelRequest();
    this.isLoading.set(false);
    this.isTyping.set(false);
  }

  onEnterKey(event: Event): void {
    const ke = event as KeyboardEvent;
    if (!ke.shiftKey) {
      ke.preventDefault();
      this.sendMessage();
    }
  }

  sendMessage(): void {
    const files = this.selectedFiles();
    if ((!this.userInput.trim() && files.length === 0) || this.isLoading()) return;

    const content = this.userInput;
    this.userInput = '';
    this.selectedFiles.set([]);
    this.isLoading.set(true);
    this.isTyping.set(true);

    // Optimistic UI update with attachment indicators
    const attachments = files.map(f => ({
      filename: f.name,
      content_type: f.type,
      base64: '', // Don't store base64 in UI memory
      size: f.size
    }));

    const userMsg: AiMessage = {
      id: crypto.randomUUID(),
      role: 'user',
      content,
      attachments: attachments.length > 0 ? attachments : undefined,
      created_at: new Date().toISOString()
    };
    this.messages.update(msgs => [...msgs, userMsg]);
    this.scrollToBottom();

    // Include page context if available
    const context = this.contextService.getContext();

    this.activeSubscription = this.chatService.sendMessage(this.conversationId, content, context, files.length > 0 ? files : undefined).subscribe({
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
        this.activeSubscription = null;
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
