import { Component, ChangeDetectionStrategy, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatCardModule } from '@angular/material/card';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatChipsModule } from '@angular/material/chips';
import { AiProposedAction } from '../../models/ai-chat.model';

@Component({
    selector: 'mdp-ai-action-card',
    standalone: true,
    changeDetection: ChangeDetectionStrategy.OnPush,
    imports: [CommonModule, MatCardModule, MatButtonModule, MatIconModule, MatChipsModule],
    template: `
    <mat-card class="action-card" appearance="outlined">
      <mat-card-header>
        <mat-icon mat-card-avatar class="action-icon">auto_fix_high</mat-icon>
        <mat-card-title>{{ action().title }}</mat-card-title>
        <mat-card-subtitle>
          <mat-chip-set>
            <mat-chip [highlighted]="true">{{ action().action_type }}</mat-chip>
          </mat-chip-set>
        </mat-card-subtitle>
      </mat-card-header>

      <mat-card-content>
        <p class="summary">{{ action().summary }}</p>

        @if (paramKeys().length > 0) {
          <div class="params-preview">
            <span class="params-label">Parameters:</span>
            <div class="params-grid">
              @for (key of paramKeys(); track key) {
                <div class="param-item">
                  <span class="param-key">{{ key }}:</span>
                  <span class="param-value">{{ action().params[key] }}</span>
                </div>
              }
            </div>
          </div>
        }
      </mat-card-content>

      <mat-card-actions align="end">
        <button mat-button (click)="dismiss.emit()">
          <mat-icon>close</mat-icon> Dismiss
        </button>
        <button mat-flat-button color="primary" (click)="apply.emit(action())">
          <mat-icon>check</mat-icon> Review & Apply
        </button>
      </mat-card-actions>
    </mat-card>
  `,
    styles: [`
    .action-card {
      margin: 8px 0;
      border-left: 4px solid var(--mat-sys-primary, #6750a4);
      animation: slideIn 0.3s ease-out;
    }

    @keyframes slideIn {
      from { opacity: 0; transform: translateY(8px); }
      to { opacity: 1; transform: translateY(0); }
    }

    .action-icon {
      color: var(--mat-sys-primary, #6750a4);
    }

    .summary {
      margin: 8px 0;
      color: var(--mat-sys-on-surface-variant, #49454f);
      font-size: 0.875rem;
    }

    .params-preview {
      background: var(--mat-sys-surface-container-low, #f7f2fa);
      border-radius: 8px;
      padding: 12px;
      margin-top: 8px;
    }

    .params-label {
      font-size: 0.75rem;
      font-weight: 500;
      text-transform: uppercase;
      color: var(--mat-sys-on-surface-variant, #49454f);
    }

    .params-grid {
      display: grid;
      grid-template-columns: auto 1fr;
      gap: 4px 12px;
      margin-top: 4px;
    }

    .param-key {
      font-weight: 500;
      font-size: 0.8125rem;
    }

    .param-value {
      font-size: 0.8125rem;
      color: var(--mat-sys-on-surface-variant, #49454f);
    }
  `]
})
export class AiActionCardComponent {
    readonly action = input.required<AiProposedAction>();
    readonly apply = output<AiProposedAction>();
    readonly dismiss = output<void>();

    paramKeys(): string[] {
        const params = this.action()?.params;
        return params ? Object.keys(params) : [];
    }
}
