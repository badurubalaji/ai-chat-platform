import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { AiSettingsComponent, AiProviderConfig } from 'mdp-ai-chat';

@Component({
    selector: 'app-settings',
    standalone: true,
    imports: [CommonModule, AiSettingsComponent],
    template: `
    <div class="settings-container">
      <h1>Settings</h1>
      <p>Configure your application settings and AI providers here.</p>
      
      <div class="ai-config-section">
        <mdp-ai-settings 
          [currentConfig]="demoConfig"
          (configSaved)="onConfigSaved($event)">
        </mdp-ai-settings>
      </div>
    </div>
  `,
    styles: [`
    .settings-container {
      max-width: 800px;
      margin: 0 auto;
    }
    .ai-config-section {
      margin-top: 24px;
    }
  `]
})
export class SettingsComponent {
    demoConfig: AiProviderConfig = {
        id: 'demo-config',
        // tenant_id: 'default', // removed as it's not in the interface
        provider: 'openai',
        model: 'gpt-4o',
        settings: {
            temperature: 0.7,
            max_tokens: 4096
        },
        enabled: true
    };

    onConfigSaved(config: AiProviderConfig) {
        console.log('Config saved:', config);
        // In a real app, save to backend
    }
}
