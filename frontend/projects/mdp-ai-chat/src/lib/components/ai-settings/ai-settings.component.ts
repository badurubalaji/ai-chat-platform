import { Component, ChangeDetectionStrategy, EventEmitter, Input, Output, signal, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ReactiveFormsModule, FormBuilder, Validators } from '@angular/forms';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatButtonModule } from '@angular/material/button';
import { MatSlideToggleModule } from '@angular/material/slide-toggle';
import { MatCardModule } from '@angular/material/card';
import { MatIconModule } from '@angular/material/icon';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { AiProviderConfig, ProviderInfo, AiProvider } from '../../models/ai-chat.model';
import { AiProviderConfigService } from '../../services/ai-provider-config.service';

@Component({
  selector: 'mdp-ai-settings',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
    MatButtonModule,
    MatSlideToggleModule,
    MatCardModule,
    MatIconModule,
    MatSnackBarModule
  ],
  template: `
    <mat-card class="settings-card">
      <mat-card-header>
        <mat-card-title>AI Provider Configuration</mat-card-title>
      </mat-card-header>
      
      <mat-card-content>
        <form [formGroup]="configForm" (ngSubmit)="saveConfig()">
          
          <mat-form-field appearance="outline" class="full-width">
            <mat-label>Provider</mat-label>
            <mat-select formControlName="provider" (selectionChange)="onProviderChange()">
              @for (p of availableProviders; track p.id) {
                <mat-option [value]="p.id">{{ p.name }}</mat-option>
              }
            </mat-select>
          </mat-form-field>

          <mat-form-field appearance="outline" class="full-width">
            <mat-label>Model</mat-label>
            @if (isCustomModel()) {
              <input matInput formControlName="model" placeholder="e.g. llama3">
            } @else {
              <mat-select formControlName="model">
                @for (m of currentProviderModels(); track m) {
                  <mat-option [value]="m">{{ m }}</mat-option>
                }
              </mat-select>
            }
            @if (isCustomModel()) {
              <button mat-icon-button matSuffix (click)="isCustomModel.set(false)" [disabled]="currentProviderModels().length === 0" title="Select from list">
                <mat-icon>list</mat-icon>
              </button>
            } @else {
              <button mat-icon-button matSuffix (click)="isCustomModel.set(true)" title="Enter custom model">
                <mat-icon>edit</mat-icon>
              </button>
            }
          </mat-form-field>

          @if (isNeuralGate()) {
            <mat-form-field appearance="outline" class="full-width">
              <mat-label>Client ID</mat-label>
              <input matInput formControlName="clientId" placeholder="ng_your_client_id">
              <mat-icon matSuffix>fingerprint</mat-icon>
            </mat-form-field>

            <mat-form-field appearance="outline" class="full-width">
              <mat-label>Client Secret</mat-label>
              <input matInput type="password" formControlName="clientSecret"
                [placeholder]="apiKeySet() ? '••••••••••• (saved, leave blank to keep)' : 'ngs_your_client_secret'">
              <mat-icon matSuffix>key</mat-icon>
              @if (apiKeySet()) {
                <mat-hint>Credentials are saved. Leave blank to keep current.</mat-hint>
              }
            </mat-form-field>
          } @else {
            <mat-form-field appearance="outline" class="full-width">
              <mat-label>API Key</mat-label>
              <input matInput type="password" formControlName="apiKey"
                [placeholder]="apiKeySet() ? '••••••••••• (saved, leave blank to keep)' : 'sk-...'">
              <mat-icon matSuffix>key</mat-icon>
              @if (apiKeySet()) {
                <mat-hint>Key is saved. Leave blank to keep current key.</mat-hint>
              }
            </mat-form-field>
          }

          @if (showEndpointField()) {
            <mat-form-field appearance="outline" class="full-width">
              <mat-label>Endpoint URL</mat-label>
              <input matInput formControlName="endpoint_url" placeholder="https://api.openai.com/v1">
            </mat-form-field>
          }

          <div class="advanced-settings" formGroupName="settings">
            <h3>Advanced Settings</h3>
            
            <mat-form-field appearance="outline" class="half-width">
              <mat-label>Temperature</mat-label>
              <input matInput type="number" formControlName="temperature" min="0" max="2" step="0.1">
            </mat-form-field>

            <mat-form-field appearance="outline" class="half-width">
              <mat-label>Max Tokens</mat-label>
              <input matInput type="number" formControlName="max_tokens">
            </mat-form-field>

            <mat-form-field appearance="outline" class="full-width">
              <mat-label>System Prompt Prefix</mat-label>
              <textarea matInput formControlName="system_prompt_prefix" rows="3"></textarea>
            </mat-form-field>
          </div>

          <div class="actions">
            <button mat-stroked-button type="button" (click)="testConnection()" [disabled]="testing()">
              <mat-icon>check_circle</mat-icon>
              {{ testing() ? 'Testing...' : 'Test Connection' }}
            </button>
            <button mat-flat-button color="primary" type="submit" [disabled]="configForm.invalid || saving()">
              {{ saving() ? 'Saving...' : 'Save Configuration' }}
            </button>
          </div>

        </form>
      </mat-card-content>
    </mat-card>
  `,
  styles: [`
    .settings-card {
      max-width: 600px;
      margin: 20px auto;
    }
    
    form {
      display: flex;
      flex-direction: column;
      gap: 16px;
      margin-top: 20px;
    }

    .full-width { width: 100%; }
    
    .half-width { 
      width: 48%; 
      margin-right: 4%;
      &:last-child { margin-right: 0; }
    }

    .advanced-settings {
      border-top: 1px solid #eee;
      padding-top: 16px;
      margin-top: 8px;
    }

    .actions {
      display: flex;
      justify-content: flex-end;
      gap: 12px;
      margin-top: 16px;
    }
  `]
})
export class AiSettingsComponent {
  @Input() currentConfig: AiProviderConfig | null = null;
  @Output() configSaved = new EventEmitter<AiProviderConfig>();

  private fb = inject(FormBuilder);
  private snackBar = inject(MatSnackBar);
  private configService = inject(AiProviderConfigService);

  availableProviders: ProviderInfo[] = [
    { id: 'claude', name: 'Anthropic Claude', icon: 'smart_toy', requires_endpoint: false },
    { id: 'openai', name: 'OpenAI', icon: 'smart_toy', requires_endpoint: false },
    { id: 'gemini', name: 'Google Gemini', icon: 'smart_toy', requires_endpoint: false },
    { id: 'ollama', name: 'Ollama (Local)', icon: 'dns', requires_endpoint: true, default_endpoint: 'http://localhost:11434' },
    { id: 'generic', name: 'Generic OpenAI', icon: 'settings_ethernet', requires_endpoint: true },
    { id: 'neuralgate', name: 'NeuralGate AI Gateway', icon: 'hub', requires_endpoint: true }
  ];

  providerModels: Record<string, string[]> = {
    'claude': [
      'claude-sonnet-4-5-20250929',
      'claude-opus-4-6',
      'claude-haiku-4-5-20251001'
    ],
    'openai': [
      'gpt-4o',
      'gpt-4o-mini',
      'o1',
      'o3-mini'
    ],
    'gemini': [
      'gemini-2.0-flash',
      'gemini-2.0-pro'
    ],
    'ollama': [
      'llama3',
      'mistral',
      'gemma',
      'codellama'
    ],
    'neuralgate': [
      'auto'
    ]
  };

  configForm = this.fb.group({
    provider: ['openai' as AiProvider, Validators.required],
    model: ['gpt-4o', Validators.required],
    apiKey: [''],
    clientId: [''],
    clientSecret: [''],
    endpoint_url: [''],
    settings: this.fb.group({
      temperature: [0.7],
      max_tokens: [4096],
      system_prompt_prefix: ['']
    }),
    enabled: [true]
  });

  testing = signal(false);
  saving = signal(false);
  showEndpointField = signal(false);
  currentProviderModels = signal<string[]>([]);
  isCustomModel = signal(false);
  apiKeySet = signal(false);
  isNeuralGate = signal(false);

  ngOnInit() {
    // Load config from backend
    this.configService.getConfig().subscribe({
      next: (config: any) => {
        if (config && config.provider) {
          this.currentConfig = config;
          this.apiKeySet.set(!!config.api_key_set);
          this.configForm.patchValue({
            provider: config.provider,
            model: config.model,
            endpoint_url: config.endpoint_url,
            settings: config.settings || { temperature: 0.7, max_tokens: 4096, system_prompt_prefix: '' },
            enabled: config.enabled
          });
          // Don't require apiKey if one is already saved
          if (config.api_key_set) {
            this.configForm.get('apiKey')?.clearValidators();
            this.configForm.get('apiKey')?.updateValueAndValidity();
          }
        }
        this.onProviderChange();
      },
      error: () => {
        this.onProviderChange();
      }
    });
  }

  onProviderChange() {
    const providerId = this.configForm.get('provider')?.value as AiProvider;
    const providerInfo = this.availableProviders.find(p => p.id === providerId);

    const wasNeuralGate = this.isNeuralGate();
    this.isNeuralGate.set(providerId === 'neuralgate');
    this.showEndpointField.set(!!providerInfo?.requires_endpoint);

    // Clear credentials when switching between NeuralGate and other providers
    if (wasNeuralGate && providerId !== 'neuralgate') {
      this.configForm.get('clientId')?.setValue('');
      this.configForm.get('clientSecret')?.setValue('');
    } else if (!wasNeuralGate && providerId === 'neuralgate') {
      this.configForm.get('apiKey')?.setValue('');
    }

    if (providerInfo?.default_endpoint && !this.configForm.get('endpoint_url')?.value) {
      this.configForm.get('endpoint_url')?.setValue(providerInfo.default_endpoint);
    }

    // Update models
    const models = this.providerModels[providerId] || [];
    this.currentProviderModels.set(models);

    // Determine if we should show custom input
    // If usage of generic, or if the list is empty (shouldn't happen with new defaults), allow custom
    // But user wants dropdown. For 'generic', we might need input, but let's provide common ones or allow editable?
    // Start with dropdown if models exist.
    this.isCustomModel.set(models.length === 0);

    // Set default model if current value is invalid for new provider
    const currentModel = this.configForm.get('model')?.value;
    if (models.length > 0 && (!currentModel || !models.includes(currentModel))) {
      this.configForm.get('model')?.setValue(models[0]);
    } else if (models.length === 0 && !currentModel) {
      this.configForm.get('model')?.setValue('');
    }
  }

  testConnection() {
    if (this.configForm.valid) {
      this.testing.set(true);
      const config = this.getFormConfig();

      this.configService.testConnection(config).subscribe({
        next: () => {
          this.testing.set(false);
          this.snackBar.open('Connection successful!', 'Close', { duration: 3000 });
        },
        error: (err) => {
          this.testing.set(false);
          this.snackBar.open(`Connection failed: ${err.message}`, 'Close', { duration: 5000 });
        }
      });
    }
  }

  saveConfig() {
    if (this.configForm.valid) {
      this.saving.set(true);
      const config = this.getFormConfig();

      this.configService.saveConfig(config).subscribe({
        next: (savedConfig) => {
          this.saving.set(false);
          this.currentConfig = savedConfig;
          this.configSaved.emit(savedConfig);
          this.snackBar.open('Configuration saved', 'Close', { duration: 3000 });
        },
        error: (err) => {
          this.saving.set(false);
          this.snackBar.open(`Failed to save: ${err.message}`, 'Close', { duration: 5000 });
        }
      });
    }
  }

  private getFormConfig(): AiProviderConfig {
    const formValue = this.configForm.value;

    // For NeuralGate, concatenate client_id:client_secret as the apiKey
    let apiKey = formValue.apiKey || '';
    if (formValue.provider === 'neuralgate') {
      const clientId = formValue.clientId || '';
      const clientSecret = formValue.clientSecret || '';
      if (clientId && clientSecret) {
        apiKey = `${clientId}:${clientSecret}`;
      } else {
        apiKey = '';
      }
    }

    return {
      id: this.currentConfig?.id || '',
      tenant_id: this.currentConfig?.tenant_id || '',
      provider: formValue.provider as AiProvider,
      model: formValue.model || '',
      apiKey: apiKey,
      endpoint_url: formValue.endpoint_url || undefined,
      settings: formValue.settings,
      enabled: formValue.enabled === true
    } as AiProviderConfig;
  }
}
