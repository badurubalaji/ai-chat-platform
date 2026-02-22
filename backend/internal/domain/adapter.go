package domain

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/mdp/ai-chat-platform/backend/internal/models"
)

// DefaultProviderConfig holds NeuralGateway credentials set at deployment time.
type DefaultProviderConfig struct {
	Provider     string `json:"provider"`
	Model        string `json:"model"`
	EndpointURL  string `json:"endpoint_url"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// ToolExecution defines how a tool is executed (HTTP call to internal service).
type ToolExecution struct {
	Type      string            `json:"type"`       // "http"
	Method    string            `json:"method"`      // GET, POST, PUT, DELETE
	URL       string            `json:"url"`         // Internal service URL with {param} placeholders
	Headers   map[string]string `json:"headers"`     // Optional headers
	TimeoutMs int               `json:"timeout_ms"`  // Default 5000
}

// ToolConfig defines a single callable tool from the adapter config.
type ToolConfig struct {
	Name                 string          `json:"name"`
	Description          string          `json:"description"`
	Parameters           json.RawMessage `json:"parameters"` // JSON Schema
	RequiredRole         string          `json:"required_role"`
	RequiresConfirmation bool            `json:"requires_confirmation"`
	Execution            ToolExecution   `json:"execution"`
}

// AdapterConfig is the raw JSON structure loaded from adapter-config.json.
type AdapterConfig struct {
	Domain          string                `json:"domain"`
	DisplayName     string                `json:"display_name"`
	SystemPrompt    string                `json:"system_prompt"`
	DefaultProvider *DefaultProviderConfig `json:"default_provider,omitempty"`
	Tools           []ToolConfig          `json:"tools"`
}

// Adapter is the immutable, read-only domain adapter loaded once at startup.
type Adapter struct {
	config    AdapterConfig
	toolIndex map[string]*ToolConfig
}

// LoadAdapter reads and validates adapter-config.json from the given path.
// If path is empty, returns nil adapter (generic mode).
func LoadAdapter(path string) (*Adapter, error) {
	if path == "" {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read adapter config at %s: %w", path, err)
	}

	var cfg AdapterConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid adapter config JSON: %w", err)
	}

	if err := validateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("adapter config validation failed: %w", err)
	}

	// Build tool index for fast lookup
	index := make(map[string]*ToolConfig, len(cfg.Tools))
	for i := range cfg.Tools {
		t := &cfg.Tools[i]
		if t.Execution.Method == "" {
			t.Execution.Method = "POST"
		}
		if t.Execution.TimeoutMs == 0 {
			t.Execution.TimeoutMs = 5000
		}
		index[t.Name] = t
	}

	return &Adapter{
		config:    cfg,
		toolIndex: index,
	}, nil
}

func validateConfig(cfg *AdapterConfig) error {
	if cfg.Domain == "" {
		return fmt.Errorf("domain is required")
	}
	if cfg.SystemPrompt == "" {
		return fmt.Errorf("system_prompt is required")
	}

	if dp := cfg.DefaultProvider; dp != nil {
		if dp.ClientID == "" || dp.ClientSecret == "" {
			return fmt.Errorf("default_provider requires client_id and client_secret")
		}
		if dp.EndpointURL == "" {
			return fmt.Errorf("default_provider requires endpoint_url")
		}
		if dp.Provider == "" {
			dp.Provider = "neuralgate"
		}
		if dp.Model == "" {
			dp.Model = "auto"
		}
	}

	for i, t := range cfg.Tools {
		if t.Name == "" {
			return fmt.Errorf("tool[%d]: name is required", i)
		}
		if t.Description == "" {
			return fmt.Errorf("tool[%d] %q: description is required", i, t.Name)
		}
		if t.Execution.URL == "" {
			return fmt.Errorf("tool[%d] %q: execution.url is required", i, t.Name)
		}
	}

	return nil
}

// Domain returns the domain identifier (e.g. "ehr", "backup").
func (a *Adapter) Domain() string { return a.config.Domain }

// DisplayName returns the human-readable domain name.
func (a *Adapter) DisplayName() string { return a.config.DisplayName }

// SystemPrompt returns the domain-specific system prompt.
func (a *Adapter) SystemPrompt() string { return a.config.SystemPrompt }

// Tools returns all tool configurations.
func (a *Adapter) Tools() []ToolConfig { return a.config.Tools }

// ToolByName looks up a tool by name.
func (a *Adapter) ToolByName(name string) (*ToolConfig, bool) {
	t, ok := a.toolIndex[name]
	return t, ok
}

// HasTools returns true if any tools are configured.
func (a *Adapter) HasTools() bool { return len(a.config.Tools) > 0 }

// HasDefaultProvider returns true if a default NeuralGateway provider is configured.
func (a *Adapter) HasDefaultProvider() bool { return a.config.DefaultProvider != nil }

// DefaultProvider returns the default provider config (may be nil).
func (a *Adapter) DefaultProvider() *DefaultProviderConfig { return a.config.DefaultProvider }

// ToolsForProvider converts adapter tools to models.Tool slice for use with
// providers that support native tool calling (Claude, OpenAI, Gemini).
func (a *Adapter) ToolsForProvider() []models.Tool {
	if len(a.config.Tools) == 0 {
		return nil
	}

	tools := make([]models.Tool, len(a.config.Tools))
	for i, t := range a.config.Tools {
		var params interface{}
		if len(t.Parameters) > 0 {
			json.Unmarshal(t.Parameters, &params)
		}
		tools[i] = models.Tool{
			Name:         t.Name,
			Description:  t.Description,
			Parameters:   params,
			RequiredRole: t.RequiredRole,
		}
	}
	return tools
}
