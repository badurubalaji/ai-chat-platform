/*
 * Public API Surface of mdp-ai-chat
 */

export * from './lib/providers/ai-chat-config';
export * from './lib/models/ai-chat.model';

// Services
export { AiChatService } from './lib/services/ai-chat.service';
export { AiProviderConfigService } from './lib/services/ai-provider-config.service';
export { AiUsageService } from './lib/services/ai-usage.service';
export { AiContextService } from './lib/services/ai-context.service';

// Components
export { AiSidebarComponent } from './lib/components/ai-sidebar/ai-sidebar.component';
export { AiSettingsComponent } from './lib/components/ai-settings/ai-settings.component';
export { AiUsageDashboardComponent } from './lib/components/ai-usage-dashboard/ai-usage-dashboard.component';
export { AiActionCardComponent } from './lib/components/ai-action-card/ai-action-card.component';
export { AiEmptyStateComponent } from './lib/components/ai-empty-state/ai-empty-state.component';

// Pipes
export { MarkdownRenderPipe } from './lib/pipes/markdown-render.pipe';
