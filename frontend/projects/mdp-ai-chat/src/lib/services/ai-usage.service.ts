import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { AI_CHAT_CONFIG } from '../providers/ai-chat-config';
import { AiUsageStats } from '../models/ai-chat.model';

@Injectable({
    providedIn: 'root'
})
export class AiUsageService {
    private config = inject(AI_CHAT_CONFIG);
    private http = inject(HttpClient);

    getUsageStats(days: number = 30): Observable<AiUsageStats> {
        return this.http.get<AiUsageStats>(`${this.config.apiBaseUrl}/api/v1/ai/usage?days=${days}`);
    }
}
