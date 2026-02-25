CREATE TABLE ai_tool_registry (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(100) NOT NULL,
    app_name VARCHAR(100) NOT NULL,
    tool_name VARCHAR(100) NOT NULL,
    description TEXT NOT NULL,
    parameters JSONB DEFAULT '{}',
    execution_config JSONB NOT NULL,
    requires_confirmation BOOLEAN DEFAULT false,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, app_name, tool_name)
);

CREATE INDEX idx_ai_tool_registry_tenant ON ai_tool_registry(tenant_id);
CREATE INDEX idx_ai_tool_registry_app ON ai_tool_registry(tenant_id, app_name);
