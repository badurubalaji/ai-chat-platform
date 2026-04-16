package domain

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// singleToolAdapter constructs a bare adapter with one tool for executor tests.
func singleToolAdapter(tool ToolConfig) *Adapter {
	return &Adapter{
		config:    AdapterConfig{Domain: "test", SystemPrompt: "p", Tools: []ToolConfig{tool}},
		toolIndex: map[string]*ToolConfig{tool.Name: &tool},
	}
}

func TestExecuteTool_UnknownToolReturnsError(t *testing.T) {
	e := NewExecutor(singleToolAdapter(ToolConfig{
		Name:      "exists",
		Execution: ToolExecution{Method: "POST", URL: "http://x", TimeoutMs: 1000},
	}))
	_, err := e.ExecuteTool(context.Background(), "nope", json.RawMessage(`{}`), "t", "u")
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}
	if !strings.Contains(err.Error(), "unknown tool") {
		t.Errorf("error = %v, want 'unknown tool'", err)
	}
}

func TestExecuteTool_POSTBodyAndHeaders(t *testing.T) {
	var capturedMethod, capturedCT, capturedTenant, capturedUser, capturedCustom string
	var capturedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedCT = r.Header.Get("Content-Type")
		capturedTenant = r.Header.Get("X-Tenant-ID")
		capturedUser = r.Header.Get("X-User-ID")
		capturedCustom = r.Header.Get("X-Custom")
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	tool := ToolConfig{
		Name: "create",
		Execution: ToolExecution{
			Method:    "POST",
			URL:       srv.URL + "/resource",
			Headers:   map[string]string{"X-Custom": "hello"},
			TimeoutMs: 2000,
		},
	}
	e := NewExecutor(singleToolAdapter(tool))

	res, err := e.ExecuteTool(context.Background(), "create",
		json.RawMessage(`{"name":"Jane"}`), "tenant-a", "user-b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(res), `"ok":true`) {
		t.Errorf("result = %s", string(res))
	}
	if capturedMethod != "POST" {
		t.Errorf("method = %q, want POST", capturedMethod)
	}
	if capturedCT != "application/json" {
		t.Errorf("Content-Type = %q", capturedCT)
	}
	if capturedTenant != "tenant-a" {
		t.Errorf("X-Tenant-ID = %q", capturedTenant)
	}
	if capturedUser != "user-b" {
		t.Errorf("X-User-ID = %q", capturedUser)
	}
	if capturedCustom != "hello" {
		t.Errorf("X-Custom = %q, configured headers should pass through", capturedCustom)
	}
	// Body should be the arguments JSON (order-independent check)
	var got map[string]interface{}
	if err := json.Unmarshal(capturedBody, &got); err != nil {
		t.Fatalf("body unmarshal: %v (body=%s)", err, capturedBody)
	}
	if got["name"] != "Jane" {
		t.Errorf("body name = %v, want Jane", got["name"])
	}
}

func TestExecuteTool_GETPathInterpolation(t *testing.T) {
	var capturedPath, capturedQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	tool := ToolConfig{
		Name: "get_patient",
		Execution: ToolExecution{
			Method:    "GET",
			URL:       srv.URL + "/patients/{id}",
			TimeoutMs: 2000,
		},
	}
	e := NewExecutor(singleToolAdapter(tool))

	// id is interpolated into the path; extra args go to the query string.
	_, err := e.ExecuteTool(context.Background(), "get_patient",
		json.RawMessage(`{"id":"P-42","detail":"full"}`), "t", "u")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedPath != "/patients/P-42" {
		t.Errorf("path = %q, want /patients/P-42", capturedPath)
	}
	if !strings.Contains(capturedQuery, "detail=full") {
		t.Errorf("query = %q, missing detail=full", capturedQuery)
	}
	if strings.Contains(capturedQuery, "id=") {
		t.Errorf("id was interpolated into path but also appears in query: %q", capturedQuery)
	}
}

func TestExecuteTool_HTTPErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
	}))
	defer srv.Close()

	tool := ToolConfig{
		Name:      "x",
		Execution: ToolExecution{Method: "POST", URL: srv.URL, TimeoutMs: 2000},
	}
	e := NewExecutor(singleToolAdapter(tool))

	_, err := e.ExecuteTool(context.Background(), "x", json.RawMessage(`{}`), "t", "u")
	if err == nil {
		t.Fatal("expected error for 403")
	}
	if !strings.Contains(err.Error(), "HTTP 403") {
		t.Errorf("error = %v, want HTTP 403 wrapper", err)
	}
}

func TestExecuteTool_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	tool := ToolConfig{
		Name:      "slow",
		Execution: ToolExecution{Method: "POST", URL: srv.URL, TimeoutMs: 50},
	}
	e := NewExecutor(singleToolAdapter(tool))

	start := time.Now()
	_, err := e.ExecuteTool(context.Background(), "slow", json.RawMessage(`{}`), "t", "u")
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if time.Since(start) > 500*time.Millisecond {
		t.Errorf("took %v, timeout not respected", time.Since(start))
	}
}

func TestExecuteTool_NonJSONResponseWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`plain text response`))
	}))
	defer srv.Close()

	tool := ToolConfig{
		Name:      "plain",
		Execution: ToolExecution{Method: "POST", URL: srv.URL, TimeoutMs: 2000},
	}
	e := NewExecutor(singleToolAdapter(tool))

	got, err := e.ExecuteTool(context.Background(), "plain", json.RawMessage(`{}`), "t", "u")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Must return valid JSON for downstream consumers.
	var parsed map[string]interface{}
	if err := json.Unmarshal(got, &parsed); err != nil {
		t.Fatalf("expected JSON-wrapped response, got raw %q", string(got))
	}
	if parsed["result"] != "plain text response" {
		t.Errorf("wrapped result = %v", parsed["result"])
	}
}

func TestExecuteTool_InvalidArgumentsJSON(t *testing.T) {
	tool := ToolConfig{
		Name:      "x",
		Execution: ToolExecution{Method: "POST", URL: "http://unused", TimeoutMs: 1000},
	}
	e := NewExecutor(singleToolAdapter(tool))
	_, err := e.ExecuteTool(context.Background(), "x", json.RawMessage(`{not json}`), "t", "u")
	if err == nil {
		t.Fatal("expected error for invalid arguments")
	}
	if !strings.Contains(err.Error(), "invalid tool arguments") {
		t.Errorf("error = %v", err)
	}
}

func TestExecuteTool_DefaultTimeoutApplied(t *testing.T) {
	// A tool with TimeoutMs==0 should pick up the 5000ms default in the executor.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	tool := ToolConfig{
		Name:      "defaults",
		Execution: ToolExecution{Method: "POST", URL: srv.URL}, // no TimeoutMs
	}
	e := NewExecutor(singleToolAdapter(tool))

	if _, err := e.ExecuteTool(context.Background(), "defaults", json.RawMessage(`{}`), "t", "u"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
