package domain

import (
	"context"
	"encoding/json"

	"github.com/mdp/ai-chat-platform/backend/internal/models"
)

// ToolSource provides tools from any source (adapter config, database, etc.)
type ToolSource interface {
	ListTools(ctx context.Context, tenantID string, appName string) ([]*models.RegisteredTool, error)
}

// ToolRegistry merges tools from static adapter config and dynamic database registry.
// Adapter tools take priority over registry tools with the same name.
type ToolRegistry struct {
	adapter *Adapter
	store   ToolSource
}

// NewToolRegistry creates a registry that merges adapter + database tools.
func NewToolRegistry(adapter *Adapter, store ToolSource) *ToolRegistry {
	return &ToolRegistry{adapter: adapter, store: store}
}

// GetAllToolConfigs returns merged tool configs from adapter + registry for a tenant.
// Adapter tools take priority over registry tools with the same name.
func (r *ToolRegistry) GetAllToolConfigs(ctx context.Context, tenantID string) ([]ToolConfig, error) {
	// Start with adapter tools (static, highest priority)
	var tools []ToolConfig
	seen := make(map[string]bool)

	if r.adapter != nil {
		for _, t := range r.adapter.Tools() {
			tools = append(tools, t)
			seen[t.Name] = true
		}
	}

	// Add dynamic registry tools (skip duplicates)
	if r.store != nil {
		registered, err := r.store.ListTools(ctx, tenantID, "")
		if err != nil {
			return tools, err // return adapter tools even if registry fails
		}

		for _, rt := range registered {
			if seen[rt.ToolName] {
				continue // adapter tool takes priority
			}

			tc, err := registeredToolToConfig(rt)
			if err != nil {
				continue // skip malformed tools
			}
			tools = append(tools, tc)
			seen[rt.ToolName] = true
		}
	}

	return tools, nil
}

// GetAllModelTools returns tools in models.Tool format for providers.
func (r *ToolRegistry) GetAllModelTools(ctx context.Context, tenantID string) ([]models.Tool, error) {
	configs, err := r.GetAllToolConfigs(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	tools := make([]models.Tool, 0, len(configs))
	for _, tc := range configs {
		var params interface{}
		if len(tc.Parameters) > 0 {
			json.Unmarshal(tc.Parameters, &params)
		}
		tools = append(tools, models.Tool{
			Name:         tc.Name,
			Description:  tc.Description,
			Parameters:   params,
			RequiredRole: tc.RequiredRole,
		})
	}
	return tools, nil
}

// LookupTool finds a tool config by name across all sources.
func (r *ToolRegistry) LookupTool(ctx context.Context, tenantID string, toolName string) (*ToolConfig, bool) {
	// Check adapter first
	if r.adapter != nil {
		if tc, ok := r.adapter.ToolByName(toolName); ok {
			return tc, true
		}
	}

	// Check registry
	if r.store != nil {
		tools, err := r.store.ListTools(ctx, tenantID, "")
		if err != nil {
			return nil, false
		}
		for _, rt := range tools {
			if rt.ToolName == toolName {
				tc, err := registeredToolToConfig(rt)
				if err != nil {
					return nil, false
				}
				return &tc, true
			}
		}
	}

	return nil, false
}

// HasTools returns true if any tools are available from any source.
func (r *ToolRegistry) HasTools(ctx context.Context, tenantID string) bool {
	if r.adapter != nil && r.adapter.HasTools() {
		return true
	}
	if r.store != nil {
		tools, err := r.store.ListTools(ctx, tenantID, "")
		if err != nil {
			return false
		}
		return len(tools) > 0
	}
	return false
}

// registeredToolToConfig converts a RegisteredTool from the DB to a ToolConfig.
func registeredToolToConfig(rt *models.RegisteredTool) (ToolConfig, error) {
	var exec ToolExecution
	if err := json.Unmarshal(rt.ExecutionConfig, &exec); err != nil {
		return ToolConfig{}, err
	}

	// Apply defaults
	if exec.Method == "" {
		exec.Method = "POST"
	}
	if exec.TimeoutMs == 0 {
		exec.TimeoutMs = 5000
	}

	return ToolConfig{
		Name:                 rt.ToolName,
		Description:          rt.Description,
		Parameters:           rt.Parameters,
		RequiresConfirmation: rt.RequiresConfirmation,
		Execution:            exec,
	}, nil
}
