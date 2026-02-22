import { Component, ChangeDetectionStrategy, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatCardModule } from '@angular/material/card';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';

@Component({
    selector: 'mdp-ai-empty-state',
    standalone: true,
    changeDetection: ChangeDetectionStrategy.OnPush,
    imports: [CommonModule, MatCardModule, MatButtonModule, MatIconModule],
    template: `
    <div class="empty-state">
      <div class="empty-icon-container">
        <mat-icon class="empty-icon">psychology</mat-icon>
      </div>
      <h3>AI Assistant</h3>
      <p class="empty-message">
        Configure an AI provider to start using the AI assistant.
        Supports OpenAI, Claude, Gemini, Ollama, and more.
      </p>
      <button mat-flat-button color="primary" (click)="configureClicked.emit()">
        <mat-icon>settings</mat-icon>
        Configure Provider
      </button>
    </div>
  `,
    styles: [`
    .empty-state {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      padding: 48px 24px;
      text-align: center;
      height: 100%;
      min-height: 300px;
    }

    .empty-icon-container {
      width: 80px;
      height: 80px;
      border-radius: 50%;
      background: var(--mat-sys-primary-container, #eaddff);
      display: flex;
      align-items: center;
      justify-content: center;
      margin-bottom: 16px;
    }

    .empty-icon {
      font-size: 40px;
      width: 40px;
      height: 40px;
      color: var(--mat-sys-on-primary-container, #21005d);
    }

    h3 {
      margin: 0 0 8px;
      font-size: 1.25rem;
      color: var(--mat-sys-on-surface, #1d1b20);
    }

    .empty-message {
      margin: 0 0 24px;
      color: var(--mat-sys-on-surface-variant, #49454f);
      font-size: 0.875rem;
      max-width: 280px;
      line-height: 1.5;
    }
  `]
})
export class AiEmptyStateComponent {
    readonly configureClicked = output<void>();
}
