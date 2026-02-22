import { EnvironmentProviders, InjectionToken, makeEnvironmentProviders } from '@angular/core';
import { Observable } from 'rxjs';

export interface MarkdownRenderOptions {
    enableSyntaxHighlighting?: boolean;
}

export interface AiChatConfig {
    apiBaseUrl: string;                        // Backend base URL
    authTokenFn: () => Observable<string>;     // JWT token provider
    appName?: string;                          // Display name (default: "AI Assistant")
    appDescription?: string;                   // For system prompt context
    enableTools?: boolean;                     // Default: true
    enableActionCards?: boolean;               // Default: true
    sidebarWidth?: number;                     // Default: 400 (px)
    sidebarMinWidth?: number;                  // Default: 300
    sidebarMaxWidth?: number;                  // Default: 600
    markdownOptions?: MarkdownRenderOptions;   // Code highlighting, etc.
}

export const AI_CHAT_CONFIG = new InjectionToken<AiChatConfig>('AI_CHAT_CONFIG');

export function provideAiChat(config: AiChatConfig): EnvironmentProviders {
    return makeEnvironmentProviders([
        { provide: AI_CHAT_CONFIG, useValue: config }
    ]);
}
