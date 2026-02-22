import { Injectable, Inject, NgZone } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, Observer } from 'rxjs';
import { AiChatConfig, AI_CHAT_CONFIG } from '../providers/ai-chat-config';
import { AiStreamChunk, AiConversation, AiMessage, AiToolConfirmation } from '../models/ai-chat.model';

@Injectable({
    providedIn: 'root'
})
export class AiChatService {
    private apiUrl: string;

    constructor(
        @Inject(AI_CHAT_CONFIG) private config: AiChatConfig,
        private http: HttpClient,
        private ngZone: NgZone
    ) {
        this.apiUrl = `${config.apiBaseUrl}/api/v1/ai`;
    }

    sendMessage(conversationId: string | null, message: string, context?: Record<string, unknown> | null): Observable<AiStreamChunk> {
        return new Observable((observer: Observer<AiStreamChunk>) => {
            const url = `${this.apiUrl}/chat`;

            this.config.authTokenFn().subscribe({
                next: (token) => {
                    this.fetchStream(url, token, conversationId, message, context ?? undefined, observer);
                },
                error: (err) => observer.error(err)
            });
        });
    }

    private async fetchStream(
        url: string,
        token: string,
        conversationId: string | null,
        message: string,
        context: Record<string, unknown> | undefined,
        observer: Observer<AiStreamChunk>
    ) {
        try {
            const response = await fetch(url, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${token}`
                },
                body: JSON.stringify({
                    conversation_id: conversationId,
                    message,
                    context
                })
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const reader = response.body?.getReader();
            if (!reader) {
                throw new Error('Response body is not readable');
            }

            const decoder = new TextDecoder();
            let currentEventType = 'chunk';

            while (true) {
                const { done, value } = await reader.read();
                if (done) break;

                const chunk = decoder.decode(value, { stream: true });
                const lines = chunk.split('\n');

                this.ngZone.run(() => {
                    for (const line of lines) {
                        if (line.startsWith('event: ')) {
                            currentEventType = line.slice(7).trim();
                        } else if (line.startsWith('data: ')) {
                            const dataStr = line.slice(6);
                            try {
                                const data = JSON.parse(dataStr);

                                switch (currentEventType) {
                                    case 'chunk':
                                        observer.next({
                                            content: data.content || '',
                                            done: false
                                        });
                                        break;

                                    case 'done':
                                        observer.next({
                                            content: '',
                                            done: true,
                                            usage: data.usage ? {
                                                input_tokens: data.usage.input_tokens,
                                                output_tokens: data.usage.output_tokens
                                            } : undefined
                                        });
                                        break;

                                    case 'tool_call':
                                        observer.next({
                                            content: '',
                                            done: false,
                                            tool_call: {
                                                tool: data.tool,
                                                status: data.status || 'executing'
                                            }
                                        });
                                        break;

                                    case 'tool_confirm':
                                        observer.next({
                                            content: '',
                                            done: false,
                                            tool_confirm: {
                                                confirmation_id: data.confirmation_id,
                                                tool: data.tool,
                                                description: data.description,
                                                params: data.params
                                            }
                                        });
                                        break;

                                    case 'tool_result':
                                        observer.next({
                                            content: '',
                                            done: false,
                                            tool_call: {
                                                tool: data.tool,
                                                status: data.status || 'complete'
                                            }
                                        });
                                        break;

                                    case 'error':
                                        observer.error(new Error(data.message || data.error?.message || JSON.stringify(data)));
                                        break;
                                }

                                // Reset event type after processing data
                                currentEventType = 'chunk';
                            } catch (e) {
                                // JSON parse failed — if this is an error event, the data is plain text
                                if (currentEventType === 'error') {
                                    observer.error(new Error(dataStr));
                                }
                                // ignore parse errors for partial lines of other event types
                            }
                        }
                    }
                });
            }
            observer.complete();
        } catch (err) {
            this.ngZone.run(() => observer.error(err));
        }
    }

    getConversations(): Observable<AiConversation[]> {
        return this.http.get<AiConversation[]>(`${this.apiUrl}/conversations`);
    }

    getConversation(id: string): Observable<{ conversation: AiConversation; messages: AiMessage[] }> {
        return this.http.get<{ conversation: AiConversation; messages: AiMessage[] }>(`${this.apiUrl}/conversations/${id}`);
    }

    createConversation(): Observable<AiConversation> {
        return this.http.post<AiConversation>(`${this.apiUrl}/conversations`, {});
    }

    deleteConversation(id: string): Observable<void> {
        return this.http.delete<void>(`${this.apiUrl}/conversations/${id}`);
    }

    sendConfirmation(confirmationId: string, approved: boolean): Observable<{ status: string }> {
        return this.http.post<{ status: string }>(`${this.apiUrl}/chat/confirm`, {
            confirmation_id: confirmationId,
            approved
        });
    }
}
