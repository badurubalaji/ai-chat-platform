import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatCardModule } from '@angular/material/card';
import { MatGridListModule } from '@angular/material/grid-list';
import { MatIconModule } from '@angular/material/icon';
import { MatButtonModule } from '@angular/material/button';
import { AiUsageDashboardComponent } from 'mdp-ai-chat';

@Component({
    selector: 'app-dashboard',
    standalone: true,
    imports: [
        CommonModule,
        MatCardModule,
        MatGridListModule,
        MatIconModule,
        MatButtonModule,
        AiUsageDashboardComponent
    ],
    template: `
    <div class="dashboard-container">
      <h1>Dashboard</h1>
      
      <div class="grid">
        <mat-card class="kpi-card">
          <mat-card-header>
            <mat-icon mat-card-avatar>storage</mat-icon>
            <mat-card-title>Storage Usage</mat-card-title>
          </mat-card-header>
          <mat-card-content>
            <div class="value">4.2 TB</div>
            <div class="trend positive">+12%</div>
          </mat-card-content>
        </mat-card>

        <mat-card class="kpi-card">
          <mat-card-header>
            <mat-icon mat-card-avatar>cloud_queue</mat-icon>
            <mat-card-title>Active Jobs</mat-card-title>
          </mat-card-header>
          <mat-card-content>
            <div class="value">24</div>
            <div class="trend">Running</div>
          </mat-card-content>
        </mat-card>

        <mat-card class="kpi-card">
          <mat-card-header>
            <mat-icon mat-card-avatar>warning</mat-icon>
            <mat-card-title>Alerts</mat-card-title>
          </mat-card-header>
          <mat-card-content>
            <div class="value warn">3</div>
            <div class="trend warn">Critical</div>
          </mat-card-content>
        </mat-card>
      </div>

      <div class="ai-section">
        <mdp-ai-usage-dashboard></mdp-ai-usage-dashboard>
      </div>
    </div>
  `,
    styles: [`
    .dashboard-container {
      max-width: 1200px;
      margin: 0 auto;
    }

    .grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
      gap: 24px;
      margin-bottom: 32px;
    }

    .kpi-card {
      .value {
        font-size: 2.5rem;
        font-weight: 500;
        margin-top: 16px;
        &.warn { color: #d32f2f; }
      }
      .trend {
        font-size: 1rem;
        color: #666;
        &.positive { color: #388e3c; }
        &.warn { color: #d32f2f; }
      }
    }

    .ai-section {
      margin-top: 32px;
    }
  `]
})
export class DashboardComponent { }
