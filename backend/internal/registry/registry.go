package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/mdp/ai-chat-platform/backend/internal/models"
)

// ToolExecutor is a function that executes a tool
type ToolExecutor func(ctx context.Context, params json.RawMessage) (json.RawMessage, error)

// ToolRegistry manages the available tools
type ToolRegistry struct {
	mu    sync.RWMutex
	tools map[string]models.Tool
	execs map[string]ToolExecutor
}

func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]models.Tool),
		execs: make(map[string]ToolExecutor),
	}
}

// Register adds a tool to the registry
func (r *ToolRegistry) Register(tool models.Tool, executor ToolExecutor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[tool.Name] = tool
	r.execs[tool.Name] = executor
}

// GetTool returns a tool definition by name
func (r *ToolRegistry) GetTool(name string) (models.Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tool, ok := r.tools[name]
	return tool, ok
}

// ListTools returns all registered tools
func (r *ToolRegistry) ListTools() []models.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tools []models.Tool
	for _, t := range r.tools {
		tools = append(tools, t)
	}
	return tools
}

// Execute runs a tool by name with the given parameters
func (r *ToolRegistry) Execute(ctx context.Context, name string, params json.RawMessage) (json.RawMessage, error) {
	r.mu.RLock()
	executor, ok := r.execs[name]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("tool not found: %s", name)
	}

	return executor(ctx, params)
}

// DefaultRegistry is a global instance
var DefaultRegistry = NewToolRegistry()

func init() {
	// get_system_summary — Overview of system health
	DefaultRegistry.Register(models.Tool{
		Name:         "get_system_summary",
		Description:  "Get an overview of system health and status",
		RequiredRole: "viewer",
		Parameters: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	}, func(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
		return json.Marshal(map[string]interface{}{
			"status":        "healthy",
			"active_agents": 12,
			"running_jobs":  24,
			"failed_jobs":   3,
			"storage_used":  "4.2 TB",
			"active_alerts": 3,
			"uptime_hours":  720,
		})
	})

	// list_items — Generic item list with filters
	DefaultRegistry.Register(models.Tool{
		Name:         "list_items",
		Description:  "List items with optional filters. Supports agents, jobs, policies, and storage items.",
		RequiredRole: "viewer",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"type":   map[string]interface{}{"type": "string", "enum": []string{"agents", "jobs", "policies", "storage"}, "description": "Type of items to list"},
				"status": map[string]interface{}{"type": "string", "description": "Optional status filter"},
				"limit":  map[string]interface{}{"type": "number", "description": "Max number of results (default 10)"},
			},
			"required": []string{"type"},
		},
	}, func(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
		type args struct {
			Type   string `json:"type"`
			Status string `json:"status"`
			Limit  int    `json:"limit"`
		}
		var input args
		if err := json.Unmarshal(params, &input); err != nil {
			return nil, err
		}
		if input.Limit == 0 {
			input.Limit = 10
		}

		// Return mock data based on type
		items := []map[string]interface{}{
			{"id": "item-1", "name": fmt.Sprintf("Sample %s #1", input.Type), "status": "active"},
			{"id": "item-2", "name": fmt.Sprintf("Sample %s #2", input.Type), "status": "active"},
			{"id": "item-3", "name": fmt.Sprintf("Sample %s #3", input.Type), "status": "inactive"},
		}
		return json.Marshal(map[string]interface{}{"items": items, "total": len(items)})
	})

	// get_item_detail — Detail by ID
	DefaultRegistry.Register(models.Tool{
		Name:         "get_item_detail",
		Description:  "Get detailed information about a specific item by type and ID",
		RequiredRole: "viewer",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"type": map[string]interface{}{"type": "string", "description": "Item type (agents/jobs/policies/storage)"},
				"id":   map[string]interface{}{"type": "string", "description": "Item ID"},
			},
			"required": []string{"type", "id"},
		},
	}, func(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
		type args struct {
			Type string `json:"type"`
			ID   string `json:"id"`
		}
		var input args
		if err := json.Unmarshal(params, &input); err != nil {
			return nil, err
		}
		return json.Marshal(map[string]interface{}{
			"id":          input.ID,
			"type":        input.Type,
			"name":        fmt.Sprintf("Detail for %s %s", input.Type, input.ID),
			"status":      "active",
			"created_at":  "2024-01-15T10:00:00Z",
			"description": "Detailed information about this item",
		})
	})

	// search_logs — Search across logs
	DefaultRegistry.Register(models.Tool{
		Name:         "search_logs",
		Description:  "Search across system logs with optional severity and time range filters",
		RequiredRole: "operator",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query":      map[string]interface{}{"type": "string", "description": "Search query string"},
				"severity":   map[string]interface{}{"type": "string", "description": "Log severity filter (debug/info/warn/error)"},
				"time_range": map[string]interface{}{"type": "string", "description": "Time range (1h/6h/24h/7d/30d)"},
			},
			"required": []string{"query"},
		},
	}, func(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
		type args struct {
			Query     string `json:"query"`
			Severity  string `json:"severity"`
			TimeRange string `json:"time_range"`
		}
		var input args
		if err := json.Unmarshal(params, &input); err != nil {
			return nil, err
		}
		return json.Marshal(map[string]interface{}{
			"query": input.Query,
			"results": []map[string]string{
				{"timestamp": "2024-05-23T12:00:00Z", "severity": "error", "message": fmt.Sprintf("Log entry matching: %s", input.Query)},
				{"timestamp": "2024-05-23T11:55:00Z", "severity": "warn", "message": fmt.Sprintf("Related log for: %s", input.Query)},
			},
			"total": 2,
		})
	})

	// analyze_failure — Analyze why something failed
	DefaultRegistry.Register(models.Tool{
		Name:         "analyze_failure",
		Description:  "Analyze why a specific job or agent failed with root cause analysis",
		RequiredRole: "operator",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"type": map[string]interface{}{"type": "string", "enum": []string{"job", "agent"}, "description": "Type of item that failed"},
				"id":   map[string]interface{}{"type": "string", "description": "ID of the failed item"},
			},
			"required": []string{"type", "id"},
		},
	}, func(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
		type args struct {
			Type string `json:"type"`
			ID   string `json:"id"`
		}
		var input args
		if err := json.Unmarshal(params, &input); err != nil {
			return nil, err
		}
		return json.Marshal(map[string]interface{}{
			"item_type":      input.Type,
			"item_id":        input.ID,
			"failure_reason": "Connection timeout to target storage endpoint",
			"root_cause":     "Network connectivity issue between agent and storage target",
			"timestamp":      "2024-05-23T11:45:00Z",
			"suggestion":     "Check network connectivity and firewall rules, then retry the operation",
		})
	})

	// propose_action — Propose a write action (NEVER auto-execute)
	DefaultRegistry.Register(models.Tool{
		Name:         "propose_action",
		Description:  "Propose a create, update, or delete action for user review. NEVER auto-executes — always requires user confirmation.",
		RequiredRole: "admin",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"action_type": map[string]interface{}{"type": "string", "description": "Type of action (create_job, restart_agent, update_policy, delete_item)"},
				"params":      map[string]interface{}{"type": "object", "description": "Pre-filled parameters for the action"},
			},
			"required": []string{"action_type", "params"},
		},
	}, func(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
		type args struct {
			ActionType string                 `json:"action_type"`
			Params     map[string]interface{} `json:"params"`
		}
		var input args
		if err := json.Unmarshal(params, &input); err != nil {
			return nil, err
		}
		return json.Marshal(map[string]interface{}{
			"action_type":           input.ActionType,
			"params":                input.Params,
			"requires_confirmation": true,
			"status":                "proposed",
			"message":               fmt.Sprintf("Action '%s' proposed — requires user confirmation before execution", input.ActionType),
		})
	})
}
