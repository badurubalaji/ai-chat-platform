import { Component, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatCardModule } from '@angular/material/card';
import { MatGridListModule } from '@angular/material/grid-list';
import { MatIconModule } from '@angular/material/icon';
import { MatButtonModule } from '@angular/material/button';
import { MatTableModule } from '@angular/material/table';

interface MockItem {
    id: string;
    name: string;
    type: string;
    status: 'active' | 'inactive' | 'error';
    lastRun: string;
}

@Component({
    selector: 'app-items',
    standalone: true,
    changeDetection: ChangeDetectionStrategy.OnPush,
    imports: [
        CommonModule,
        MatCardModule,
        MatGridListModule,
        MatIconModule,
        MatButtonModule,
        MatTableModule
    ],
    template: `
    <div class="items-container">
      <h1>Items</h1>
      <p>Browse and manage your system items. Navigating here sets the AI context to "items".</p>
      
      <div class="items-grid">
        @for (item of items; track item.id) {
          <mat-card class="item-card">
            <mat-card-header>
              <mat-icon mat-card-avatar>{{ getIcon(item.type) }}</mat-icon>
              <mat-card-title>{{ item.name }}</mat-card-title>
              <mat-card-subtitle>{{ item.type | titlecase }}</mat-card-subtitle>
            </mat-card-header>
            <mat-card-content>
              <div class="item-meta">
                <span class="status" [class]="item.status">
                  <mat-icon>{{ item.status === 'active' ? 'check_circle' : item.status === 'error' ? 'error' : 'pause_circle' }}</mat-icon>
                  {{ item.status | titlecase }}
                </span>
                <span class="last-run">Last run: {{ item.lastRun }}</span>
              </div>
            </mat-card-content>
            <mat-card-actions align="end">
              <button mat-button color="primary">Details</button>
            </mat-card-actions>
          </mat-card>
        }
      </div>
    </div>
  `,
    styles: [`
    .items-container {
      max-width: 1200px;
      margin: 0 auto;
    }

    .items-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
      gap: 16px;
      margin-top: 24px;
    }

    .item-card { }

    .item-meta {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-top: 12px;
    }

    .status {
      display: flex;
      align-items: center;
      gap: 4px;
      font-size: 0.875rem;
      &.active { color: #388e3c; }
      &.inactive { color: #757575; }
      &.error { color: #d32f2f; }
      mat-icon {
        font-size: 16px;
        width: 16px;
        height: 16px;
      }
    }

    .last-run {
      font-size: 0.75rem;
      color: #999;
    }
  `]
})
export class ItemsComponent {
    items: MockItem[] = [
        { id: 'agent-1', name: 'Production Agent', type: 'agent', status: 'active', lastRun: '5 min ago' },
        { id: 'job-1', name: 'Daily Backup Job', type: 'job', status: 'active', lastRun: '2 hours ago' },
        { id: 'job-2', name: 'Weekly Report', type: 'job', status: 'error', lastRun: '1 day ago' },
        { id: 'policy-1', name: 'Retention Policy', type: 'policy', status: 'active', lastRun: 'N/A' },
        { id: 'storage-1', name: 'S3 Bucket - Primary', type: 'storage', status: 'active', lastRun: '30 min ago' },
        { id: 'agent-2', name: 'Dev Agent', type: 'agent', status: 'inactive', lastRun: '3 days ago' },
    ];

    getIcon(type: string): string {
        switch (type) {
            case 'agent': return 'dns';
            case 'job': return 'work';
            case 'policy': return 'policy';
            case 'storage': return 'storage';
            default: return 'category';
        }
    }
}
