package domain

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Executor handles HTTP-based tool execution against internal services.
type Executor struct {
	adapter  *Adapter
	registry *ToolRegistry
}

// NewExecutor creates a new tool executor for the given adapter.
func NewExecutor(adapter *Adapter) *Executor {
	return &Executor{adapter: adapter}
}

// NewExecutorWithRegistry creates a tool executor that resolves tools from the registry.
func NewExecutorWithRegistry(adapter *Adapter, registry *ToolRegistry) *Executor {
	return &Executor{adapter: adapter, registry: registry}
}

// ExecuteToolConfig executes a tool given its config directly.
func (e *Executor) ExecuteToolConfig(ctx context.Context, tool *ToolConfig, arguments json.RawMessage, tenantID, userID string) (json.RawMessage, error) {
	return e.executeHTTP(ctx, tool, arguments, tenantID, userID)
}

// ExecuteTool executes a tool by making an HTTP call to the configured internal service.
// Returns the response body as json.RawMessage, or an error.
func (e *Executor) ExecuteTool(ctx context.Context, toolName string, arguments json.RawMessage, tenantID, userID string) (json.RawMessage, error) {
	// Try registry first (includes adapter tools)
	if e.registry != nil {
		if tc, ok := e.registry.LookupTool(ctx, tenantID, toolName); ok {
			return e.executeHTTP(ctx, tc, arguments, tenantID, userID)
		}
	}

	// Fallback to adapter-only lookup
	tool, ok := e.adapter.ToolByName(toolName)
	if !ok {
		return nil, fmt.Errorf("unknown tool: %s", toolName)
	}

	return e.executeHTTP(ctx, tool, arguments, tenantID, userID)
}

func (e *Executor) executeHTTP(ctx context.Context, tool *ToolConfig, arguments json.RawMessage, tenantID, userID string) (json.RawMessage, error) {

	// Parse arguments
	var args map[string]interface{}
	if len(arguments) > 0 {
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid tool arguments: %w", err)
		}
	}
	if args == nil {
		args = make(map[string]interface{})
	}

	// Interpolate URL path parameters
	execURL := tool.Execution.URL
	for key, val := range args {
		placeholder := "{" + key + "}"
		if strings.Contains(execURL, placeholder) {
			execURL = strings.ReplaceAll(execURL, placeholder, fmt.Sprintf("%v", val))
		}
	}

	// Set timeout
	timeoutMs := tool.Execution.TimeoutMs
	if timeoutMs <= 0 {
		timeoutMs = 5000
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	// Build request
	method := strings.ToUpper(tool.Execution.Method)
	var req *http.Request
	var err error

	switch method {
	case "GET":
		// Append arguments as query parameters
		parsedURL, parseErr := url.Parse(execURL)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid tool URL: %w", parseErr)
		}
		q := parsedURL.Query()
		for key, val := range args {
			if !strings.Contains(tool.Execution.URL, "{"+key+"}") {
				q.Set(key, fmt.Sprintf("%v", val))
			}
		}
		parsedURL.RawQuery = q.Encode()
		req, err = http.NewRequestWithContext(ctx, method, parsedURL.String(), nil)

	default:
		// POST, PUT, DELETE — send arguments as JSON body
		body, _ := json.Marshal(args)
		req, err = http.NewRequestWithContext(ctx, method, execURL, bytes.NewReader(body))
		if req != nil {
			req.Header.Set("Content-Type", "application/json")
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set configured headers
	for key, val := range tool.Execution.Headers {
		req.Header.Set(key, val)
	}
	// Always pass tenant and user context
	req.Header.Set("X-Tenant-ID", tenantID)
	req.Header.Set("X-User-ID", userID)

	// Execute
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return json.RawMessage(fmt.Sprintf(`{"error": "service unavailable: %s"}`, err.Error())), fmt.Errorf("tool execution failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return json.RawMessage(respBody), fmt.Errorf("tool returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	// Ensure valid JSON response
	if !json.Valid(respBody) {
		return json.RawMessage(fmt.Sprintf(`{"result": %q}`, string(respBody))), nil
	}

	return json.RawMessage(respBody), nil
}
