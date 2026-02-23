export type AiProvider = 'claude' | 'openai' | 'gemini' | 'ollama' | 'generic' | 'neuralgate';

export interface AiFileAttachment {
    filename: string;
    content_type: string;
    base64: string;
    size: number;
}

export interface AiProviderConfig {
    id: string;
    tenant_id?: string;
    provider: AiProvider;
    model: string;
    apiKey?: string;              // Optional, only for sending updates
    endpoint_url?: string;
    settings: AiProviderSettings;
    enabled: boolean;
}

export interface AiProviderSettings {
    temperature: number;          // 0-1, default 0.7
    max_tokens: number;           // default 4096
    system_prompt_prefix?: string;
}

export interface AiConversation {
    id: string;
    title?: string;
    created_at: string;
    updated_at: string;
    message_count?: number;
}

export type AiMessageRole = 'user' | 'assistant' | 'system' | 'tool_call' | 'tool_result';

export interface AiMessage {
    id: string;
    role: AiMessageRole;
    content: string;
    attachments?: AiFileAttachment[];
    metadata?: AiMessageMetadata;
    created_at: string;
}

export interface AiMessageMetadata {
    tool_name?: string;
    tool_status?: 'executing' | 'complete' | 'error' | 'cancelled';
    action_card?: AiProposedAction;
    tool_confirmation?: AiToolConfirmation;
    token_usage?: { input: number; output: number };
}

export interface AiProposedAction {
    action_type: string;         // 'create_job', 'restart_agent', etc.
    title: string;               // "Create backup job for server-web-01"
    summary: string;             // Brief description
    params: Record<string, unknown>;  // Pre-filled parameters
    requires_confirmation: boolean;    // Always true for write actions
}

export interface AiToolConfirmation {
    confirmation_id: string;
    tool: string;
    description: string;
    params: Record<string, unknown>;
}

export interface AiStreamChunk {
    content: string;
    done: boolean;
    tool_call?: { tool: string; status: string };
    tool_confirm?: AiToolConfirmation;
    usage?: { input_tokens: number; output_tokens: number };
}

export interface AiUsageStats {
    daily: AiUsagePeriod[];
    total_input_tokens: number;
    total_output_tokens: number;
    conversation_count: number;
}

export interface AiUsagePeriod {
    date: string;
    input_tokens: number;
    output_tokens: number;
    request_count: number;
}

export interface AiModelInfo {
    id: string;
    name: string;
    context_window: number;
    supports_tools: boolean;
    supports_streaming: boolean;
}

export interface ProviderInfo {
    id: AiProvider;
    name: string;                // "Anthropic Claude"
    icon: string;                // Material icon name
    requires_endpoint: boolean;  // true for ollama/generic
    default_endpoint?: string;
}
