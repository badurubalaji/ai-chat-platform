import { Component, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterOutlet, RouterLink } from '@angular/router';
import { MatSidenavModule } from '@angular/material/sidenav';
import { MatToolbarModule } from '@angular/material/toolbar';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { AiSidebarComponent, AiProposedAction } from 'mdp-ai-chat';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [
    CommonModule,
    RouterOutlet,
    RouterLink,
    MatSidenavModule,
    MatToolbarModule,
    MatButtonModule,
    MatIconModule,
    AiSidebarComponent
  ],
  template: `
    <mat-sidenav-container class="sidenav-container">
      <mat-sidenav #drawer class="sidenav" fixedInViewport
          [attr.role]="isHandset() ? 'dialog' : 'navigation'"
          [mode]="isHandset() ? 'over' : 'side'"
          [opened]="isHandset() === false">
        <mat-toolbar>Menu</mat-toolbar>
        <button mat-button class="nav-link" routerLink="/dashboard">Dashboard</button>
        <button mat-button class="nav-link" routerLink="/items">Items</button>
        <button mat-button class="nav-link" routerLink="/settings">Settings</button>
      </mat-sidenav>
      
      <mat-sidenav-content>
        <mat-toolbar color="primary">
          @if (isHandset()) {
            <button
              type="button"
              aria-label="Toggle sidenav"
              mat-icon-button
              (click)="drawer.toggle()">
              <mat-icon aria-label="Side nav toggle icon">menu</mat-icon>
            </button>
          }
          <span>AI Chat Platform Demo</span>
          <span class="spacer"></span>
          <button mat-icon-button (click)="toggleAiSidebar()">
            <mat-icon>smart_toy</mat-icon>
          </button>
        </mat-toolbar>
        
        <div class="main-content">
          <router-outlet></router-outlet>
        </div>

      </mat-sidenav-content>

      <mat-sidenav position="end" mode="side" [opened]="showAiSidebar()" class="ai-sidenav">
        <mdp-ai-sidebar 
          (actionRequested)="onAiAction($event)">
        </mdp-ai-sidebar>
      </mat-sidenav>

    </mat-sidenav-container>
  `,
  styles: [`
    .sidenav-container {
      height: 100%;
    }

    .sidenav {
      width: 200px;
    }

    .main-content {
      padding: 24px;
    }

    .spacer {
      flex: 1 1 auto;
    }

    .nav-link {
      width: 100%;
      text-align: left;
      padding: 16px;
    }

    .ai-sidenav {
      width: 400px;
      border-left: 1px solid #e0e0e0;
    }
  `]
})
export class AppComponent {
  isHandset = signal(false);
  showAiSidebar = signal(true);

  toggleAiSidebar() {
    this.showAiSidebar.update(v => !v);
  }

  onAiAction(action: AiProposedAction) {
    console.log('AI Action Requested:', action);
  }
}
