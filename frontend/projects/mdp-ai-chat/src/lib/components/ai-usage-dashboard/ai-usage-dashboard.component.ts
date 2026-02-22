import { Component, ChangeDetectionStrategy, OnInit, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatCardModule } from '@angular/material/card';
import { MatButtonModule } from '@angular/material/button';
import { MatButtonToggleModule } from '@angular/material/button-toggle';
import { MatIconModule } from '@angular/material/icon';
import { AiUsageStats, AiUsagePeriod } from '../../models/ai-chat.model';
import { AiUsageService } from '../../services/ai-usage.service';

@Component({
  selector: 'mdp-ai-usage-dashboard',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    CommonModule,
    MatCardModule,
    MatButtonModule,
    MatButtonToggleModule,
    MatIconModule
  ],
  template: `
    <div class="dashboard-container">
      <div class="header">
        <h2>Usage Dashboard</h2>
        <mat-button-toggle-group [value]="period()" (change)="onPeriodChange($event.value)">
          <mat-button-toggle value="7d">7 Days</mat-button-toggle>
          <mat-button-toggle value="30d">30 Days</mat-button-toggle>
          <mat-button-toggle value="90d">90 Days</mat-button-toggle>
        </mat-button-toggle-group>
      </div>

      @if (loading()) {
        <div class="loading-state">Loading usage statistics...</div>
      } @else if (error()) {
        <div class="error-state">{{ error() }}</div>
      } @else {

      <div class="stats-grid">
        <mat-card class="stat-card">
          <mat-card-header>
            <mat-icon mat-card-avatar class="stat-icon input">input</mat-icon>
            <mat-card-title>Input Tokens</mat-card-title>
            <mat-card-subtitle>Total prompt usage</mat-card-subtitle>
          </mat-card-header>
          <mat-card-content>
            <div class="stat-value">{{ stats()?.total_input_tokens | number }}</div>
          </mat-card-content>
        </mat-card>

        <mat-card class="stat-card">
          <mat-card-header>
            <mat-icon mat-card-avatar class="stat-icon output">output</mat-icon>
            <mat-card-title>Output Tokens</mat-card-title>
            <mat-card-subtitle>Total completion usage</mat-card-subtitle>
          </mat-card-header>
          <mat-card-content>
            <div class="stat-value">{{ stats()?.total_output_tokens | number }}</div>
          </mat-card-content>
        </mat-card>

        <mat-card class="stat-card">
          <mat-card-header>
            <mat-icon mat-card-avatar class="stat-icon cost">attach_money</mat-icon>
            <mat-card-title>Est. Cost</mat-card-title>
            <mat-card-subtitle>Approximate total cost</mat-card-subtitle>
          </mat-card-header>
          <mat-card-content>
            <div class="stat-value">$12.50</div> 
            <!-- Mock calculation -->
          </mat-card-content>
        </mat-card>
      </div>

      <mat-card class="chart-card">
        <mat-card-header>
          <mat-card-title>Daily Usage Trends</mat-card-title>
        </mat-card-header>
        <mat-card-content>
          <div class="chart-placeholder">
            <!-- In a real app, use Chart.js or D3 -->
            <div class="bar-container">
              @for (day of stats()?.daily; track day.date) {
                <div class="bar-group" [title]="day.date">
                  <div class="bar input" [style.height.%]="(day.input_tokens / 5000) * 100"></div>
                  <div class="bar output" [style.height.%]="(day.output_tokens / 5000) * 100"></div>
                  <span class="label">{{ day.date | date:'MM/dd' }}</span>
                </div>
              }
            </div>
          </div>
        </mat-card-content>
      </mat-card>
      }
    </div>
  `,
  styles: [`
    .dashboard-container {
      padding: 24px;
      max-width: 1200px;
      margin: 0 auto;
    }

    .header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 24px;
      h2 { margin: 0; }
    }

    .stats-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
      gap: 24px;
      margin-bottom: 24px;
    }

    .stat-card {
      .stat-value {
        font-size: 2rem;
        font-weight: 500;
        margin-top: 16px;
      }
    }

    .stat-icon {
      display: flex;
      align-items: center;
      justify-content: center;
      background: #f5f5f5;
      color: #666;
      &.input { color: #1976d2; background: #e3f2fd; }
      &.output { color: #388e3c; background: #e8f5e9; }
      &.cost { color: #fbc02d; background: #fffde7; }
    }

    .chart-card {
      min-height: 400px;
    }

    .chart-placeholder {
      height: 300px;
      display: flex;
      align-items: flex-end;
      padding: 24px 0;
    }

    .bar-container {
      display: flex;
      justify-content: space-around;
      width: 100%;
      height: 100%;
      align-items: flex-end;
    }

    .bar-group {
      display: flex;
      flex-direction: column;
      align-items: center;
      height: 100%;
      justify-content: flex-end;
      width: 40px;
      gap: 4px;

      .bar {
        width: 12px;
        border-radius: 2px;
        &.input { background: #90caf9; }
        &.output { background: #a5d6a7; }
      }
      
      .label {
        margin-top: 8px;
        font-size: 0.75rem;
        color: #757575;
      }
    }
  `]
})
export class AiUsageDashboardComponent implements OnInit {
  period = signal('7d');
  stats = signal<AiUsageStats | null>(null);
  loading = signal(false);
  error = signal<string | null>(null);

  private usageService = inject(AiUsageService);

  ngOnInit() {
    this.loadStats();
  }

  loadStats() {
    const days = parseInt(this.period().replace('d', ''), 10) || 7;
    this.loading.set(true);
    this.error.set(null); // Reset error

    this.usageService.getUsageStats(days).subscribe({
      next: (data) => {
        this.stats.set(data);
        this.loading.set(false);
      },
      error: (err) => {
        console.error('Failed to load usage stats:', err);
        this.error.set('Failed to load usage statistics.');
        this.loading.set(false);
      }
    });
  }

  onPeriodChange(val: string) {
    this.period.set(val);
    this.loadStats();
  }
}
