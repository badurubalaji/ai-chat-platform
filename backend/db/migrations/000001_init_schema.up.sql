-- Tenant AI provider configuration
CREATE TABLE ai_provider_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(100) NOT NULL,
    provider VARCHAR(50) NOT NULL,          -- claude, openai, gemini, ollama, generic
    model VARCHAR(100) NOT NULL,
    api_key_encrypted BYTEA NOT NULL,       -- encrypted at rest (AES-256-GCM)
    endpoint_url VARCHAR(500),              -- custom endpoint for ollama/self-hosted
    settings JSONB DEFAULT '{}',            -- temperature, max_tokens, system_prompt_prefix
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id)
);

-- Chat conversations
CREATE TABLE ai_conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(100) NOT NULL,
    user_id UUID NOT NULL,
    title VARCHAR(255),                     -- auto-generated from first message
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Chat messages
CREATE TABLE ai_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES ai_conversations(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL,              -- user, assistant, system, tool_use, tool_result
    content TEXT NOT NULL,
    metadata JSONB DEFAULT '{}',            -- tool calls, action cards, token counts
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Token usage tracking
CREATE TABLE ai_usage_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(100) NOT NULL,
    user_id UUID NOT NULL,
    conversation_id UUID NOT NULL,
    provider VARCHAR(50) NOT NULL,
    model VARCHAR(100) NOT NULL,
    input_tokens INT NOT NULL DEFAULT 0,
    output_tokens INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_ai_conversations_tenant ON ai_conversations(tenant_id, user_id);
CREATE INDEX idx_ai_messages_conversation ON ai_messages(conversation_id, created_at);
CREATE INDEX idx_ai_usage_tenant ON ai_usage_logs(tenant_id, created_at);
