package domain

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "adapter-config.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return path
}

func TestLoadAdapter_EmptyPathReturnsNil(t *testing.T) {
	a, err := LoadAdapter("")
	if err != nil {
		t.Fatalf("expected no error for empty path, got %v", err)
	}
	if a != nil {
		t.Fatalf("expected nil adapter for empty path, got %+v", a)
	}
}

func TestLoadAdapter_FileNotFound(t *testing.T) {
	_, err := LoadAdapter(filepath.Join(t.TempDir(), "does-not-exist.json"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "failed to read adapter config") {
		t.Errorf("expected read error, got: %v", err)
	}
}

func TestLoadAdapter_InvalidJSON(t *testing.T) {
	path := writeTempConfig(t, `{ this is not json `)
	_, err := LoadAdapter(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "invalid adapter config JSON") {
		t.Errorf("expected JSON error, got: %v", err)
	}
}

func TestLoadAdapter_ValidConfig(t *testing.T) {
	path := writeTempConfig(t, `{
		"domain": "ehr",
		"display_name": "EHR Assistant",
		"system_prompt": "You are an EHR helper.",
		"tools": [
			{
				"name": "list_patients",
				"description": "List all patients",
				"parameters": {"type":"object","properties":{}},
				"execution": {"type":"http","method":"GET","url":"http://svc/patients"}
			}
		]
	}`)

	a, err := LoadAdapter(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a == nil {
		t.Fatal("expected non-nil adapter")
	}
	if a.Domain() != "ehr" {
		t.Errorf("Domain() = %q, want ehr", a.Domain())
	}
	if a.DisplayName() != "EHR Assistant" {
		t.Errorf("DisplayName() = %q", a.DisplayName())
	}
	if a.SystemPrompt() != "You are an EHR helper." {
		t.Errorf("SystemPrompt() = %q", a.SystemPrompt())
	}
	if !a.HasTools() {
		t.Error("HasTools() = false, want true")
	}
	if len(a.Tools()) != 1 {
		t.Fatalf("Tools() len = %d, want 1", len(a.Tools()))
	}
	if a.HasDefaultProvider() {
		t.Error("HasDefaultProvider() = true, want false (no default_provider in config)")
	}

	tool, ok := a.ToolByName("list_patients")
	if !ok {
		t.Fatal("ToolByName(list_patients) not found")
	}
	if tool.Execution.TimeoutMs != 5000 {
		t.Errorf("default TimeoutMs = %d, want 5000", tool.Execution.TimeoutMs)
	}
	if _, ok := a.ToolByName("missing"); ok {
		t.Error("ToolByName(missing) returned ok=true")
	}
}

func TestLoadAdapter_DefaultsMethodToPOST(t *testing.T) {
	// Tool with no explicit method should default to POST
	path := writeTempConfig(t, `{
		"domain": "x",
		"system_prompt": "p",
		"tools": [{"name":"t","description":"d","execution":{"type":"http","url":"http://x"}}]
	}`)
	a, err := LoadAdapter(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tool, _ := a.ToolByName("t")
	if tool.Execution.Method != "POST" {
		t.Errorf("default Method = %q, want POST", tool.Execution.Method)
	}
}

func TestLoadAdapter_ValidationFailures(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		wantMsg string
	}{
		{
			name:    "missing domain",
			body:    `{"system_prompt":"p","tools":[]}`,
			wantMsg: "domain is required",
		},
		{
			name:    "missing system_prompt",
			body:    `{"domain":"x","tools":[]}`,
			wantMsg: "system_prompt is required",
		},
		{
			name:    "tool missing name",
			body:    `{"domain":"x","system_prompt":"p","tools":[{"description":"d","execution":{"type":"http","url":"http://x"}}]}`,
			wantMsg: "name is required",
		},
		{
			name:    "tool missing description",
			body:    `{"domain":"x","system_prompt":"p","tools":[{"name":"t","execution":{"type":"http","url":"http://x"}}]}`,
			wantMsg: "description is required",
		},
		{
			name:    "tool missing execution url",
			body:    `{"domain":"x","system_prompt":"p","tools":[{"name":"t","description":"d","execution":{"type":"http"}}]}`,
			wantMsg: "execution.url is required",
		},
		{
			name:    "default_provider missing client creds",
			body:    `{"domain":"x","system_prompt":"p","default_provider":{"endpoint_url":"http://ng"},"tools":[]}`,
			wantMsg: "client_id and client_secret",
		},
		{
			name:    "default_provider missing endpoint",
			body:    `{"domain":"x","system_prompt":"p","default_provider":{"client_id":"a","client_secret":"b"},"tools":[]}`,
			wantMsg: "endpoint_url",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := writeTempConfig(t, tc.body)
			_, err := LoadAdapter(path)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), tc.wantMsg) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.wantMsg)
			}
		})
	}
}

func TestLoadAdapter_DefaultProviderDefaults(t *testing.T) {
	path := writeTempConfig(t, `{
		"domain": "x",
		"system_prompt": "p",
		"default_provider": {
			"endpoint_url": "http://ng",
			"client_id": "id",
			"client_secret": "secret"
		},
		"tools": []
	}`)
	a, err := LoadAdapter(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !a.HasDefaultProvider() {
		t.Fatal("HasDefaultProvider() = false")
	}
	dp := a.DefaultProvider()
	if dp.Provider != "neuralgate" {
		t.Errorf("default Provider = %q, want neuralgate", dp.Provider)
	}
	if dp.Model != "auto" {
		t.Errorf("default Model = %q, want auto", dp.Model)
	}
}

func TestAdapter_ToolsForProvider(t *testing.T) {
	path := writeTempConfig(t, `{
		"domain": "x",
		"system_prompt": "p",
		"tools": [
			{
				"name": "a",
				"description": "desc-a",
				"parameters": {"type":"object","properties":{"q":{"type":"string"}}},
				"required_role": "admin",
				"execution": {"type":"http","method":"GET","url":"http://x/a"}
			},
			{
				"name": "b",
				"description": "desc-b",
				"execution": {"type":"http","url":"http://x/b"}
			}
		]
	}`)
	a, err := LoadAdapter(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	tools := a.ToolsForProvider()
	if len(tools) != 2 {
		t.Fatalf("len = %d, want 2", len(tools))
	}
	if tools[0].Name != "a" || tools[0].RequiredRole != "admin" {
		t.Errorf("tool[0] = %+v", tools[0])
	}
	if tools[0].Parameters == nil {
		t.Error("tool[0].Parameters should be decoded, got nil")
	}
	if tools[1].Name != "b" {
		t.Errorf("tool[1].Name = %q, want b", tools[1].Name)
	}
}

func TestAdapter_ToolsForProvider_EmptyWhenNoTools(t *testing.T) {
	path := writeTempConfig(t, `{"domain":"x","system_prompt":"p","tools":[]}`)
	a, _ := LoadAdapter(path)
	if tools := a.ToolsForProvider(); tools != nil {
		t.Errorf("want nil, got %v", tools)
	}
	if a.HasTools() {
		t.Error("HasTools() = true, want false")
	}
}
