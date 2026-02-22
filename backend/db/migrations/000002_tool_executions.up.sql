CREATE TABLE ai_tool_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(100) NOT NULL,
    user_id UUID NOT NULL,
    conversation_id UUID NOT NULL REFERENCES ai_conversations(id) ON DELETE CASCADE,
    tool_name VARCHAR(100) NOT NULL,
    arguments JSONB DEFAULT '{}',
    result JSONB,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    confirmed_by_user BOOLEAN DEFAULT false,
    execution_duration_ms INT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_ai_tool_executions_tenant ON ai_tool_executions(tenant_id, created_at);
CREATE INDEX idx_ai_tool_executions_conversation ON ai_tool_executions(conversation_id);
