import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { map } from 'rxjs/operators';
import { AI_CHAT_CONFIG } from '../providers/ai-chat-config';
import { AiProviderConfig } from '../models/ai-chat.model';

@Injectable({
    providedIn: 'root'
})
export class AiProviderConfigService {
    private http = inject(HttpClient);
    private config = inject(AI_CHAT_CONFIG);

    getConfig(): Observable<AiProviderConfig> {
        return this.http.get<AiProviderConfig>(`${this.config.apiBaseUrl}/api/v1/ai/config`);
    }

    saveConfig(config: AiProviderConfig): Observable<AiProviderConfig> {
        return this.http.post<AiProviderConfig>(`${this.config.apiBaseUrl}/api/v1/ai/config`, config);
    }

    deleteConfig(): Observable<void> {
        return this.http.delete<void>(`${this.config.apiBaseUrl}/api/v1/ai/config`);
    }

    testConnection(config: AiProviderConfig): Observable<boolean> {
        return this.http.post<{ status: string; message?: string }>(`${this.config.apiBaseUrl}/api/v1/ai/config/test`, {
            provider: config.provider,
            apiKey: (config as unknown as Record<string, unknown>)['apiKey'] || '',
            model: config.model,
            endpoint_url: config.endpoint_url
        }).pipe(
            map(res => {
                if (res.status === 'error') {
                    throw new Error(res.message || 'Connection test failed');
                }
                return true;
            })
        );
    }

    getModels(provider: string, apiKey: string, endpointUrl?: string): Observable<string[]> {
        return this.http.post<string[]>(`${this.config.apiBaseUrl}/api/v1/ai/config/models`, {
            provider,
            apiKey,
            endpoint_url: endpointUrl
        });
    }
}
