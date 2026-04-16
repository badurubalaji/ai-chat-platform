package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func toolsFixture() []ToolConfig {
	return []ToolConfig{
		{
			Name:        "list_patients",
			Description: "List all patients in the system.",
			Parameters:  json.RawMessage(`{"type":"object","properties":{}}`),
			Execution:   ToolExecution{Type: "http", Method: "GET", URL: "http://svc/patients", TimeoutMs: 5000},
		},
		{
			Name:                 "delete_patient",
			Description:          "Permanently delete a patient record.",
			Parameters:           json.RawMessage(`{"type":"object","properties":{"id":{"type":"string"}},"required":["id"]}`),
			RequiresConfirmation: true,
			Execution:            ToolExecution{Type: "http", Method: "DELETE", URL: "http://svc/patients/{id}", TimeoutMs: 5000},
		},
	}
}

// --- BuildSystemPrompt ---

func TestBuildSystemPrompt_NoToolsReturnsBase(t *testing.T) {
	o := NewOrchestratorWithTools(nil)
	// adapter nil + no tools -> empty string base
	if got := o.BuildSystemPrompt(); got != "" {
		t.Errorf("expected empty base prompt, got %q", got)
	}
}

func TestBuildSystemPrompt_EmptyAdapterToolsUnchanged(t *testing.T) {
	// Simulate adapter with prompt but no tools
	a := &Adapter{
		config:    AdapterConfig{Domain: "x", SystemPrompt: "BASE PROMPT"},
		toolIndex: map[string]*ToolConfig{},
	}
	o := NewOrchestrator(a)
	if got := o.BuildSystemPrompt(); got != "BASE PROMPT" {
		t.Errorf("want unchanged base prompt, got %q", got)
	}
}

func TestBuildSystemPrompt_WithToolsIncludesSchemasAndFormat(t *testing.T) {
	a := &Adapter{
		config: AdapterConfig{
			Domain:       "ehr",
			SystemPrompt: "You are a clinical assistant.",
			Tools:        toolsFixture(),
		},
		toolIndex: map[string]*ToolConfig{},
	}
	o := NewOrchestrator(a)
	got := o.BuildSystemPrompt()

	// Must keep base prompt
	if !strings.Contains(got, "You are a clinical assistant.") {
		t.Error("missing base prompt")
	}
	// Must include tool section header
	if !strings.Contains(got, "## Available Tools") {
		t.Error("missing Available Tools header")
	}
	// Must list both tools by name
	for _, name := range []string{"list_patients", "delete_patient"} {
		if !strings.Contains(got, name) {
			t.Errorf("missing tool %q", name)
		}
	}
	// Must include JSON format instruction matching ParseToolCall expectation
	if !strings.Contains(got, `{"tool_call": {"name": "tool_name", "arguments": {"param": "value"}}}`) {
		t.Error("missing canonical tool_call JSON format instruction")
	}
	// Must call out confirmation-required tool
	if !strings.Contains(got, "requires user confirmation") {
		t.Error("missing confirmation note for delete_patient")
	}
}

func TestBuildSystemPromptWithTools_OverrideRestoresState(t *testing.T) {
	o := NewOrchestratorWithTools([]ToolConfig{
		{Name: "initial", Description: "x"},
	})
	one := toolsFixture()[:1]
	_ = o.BuildSystemPromptWithTools(one)
	// After temporary override, original tools should be restored
	got := o.BuildSystemPrompt()
	if !strings.Contains(got, "initial") {
		t.Errorf("expected original tool restored after override, prompt: %q", got)
	}
	if strings.Contains(got, "delete_patient") {
		t.Error("override tool leaked into later prompt")
	}
}

// --- ParseToolCall ---

func TestParseToolCall_TableDriven(t *testing.T) {
	cases := []struct {
		name        string
		input       string
		wantToolCall bool
		wantName    string
		wantArgs    string
	}{
		{
			name:        "raw JSON",
			input:       `{"tool_call": {"name": "list_patients", "arguments": {}}}`,
			wantToolCall: true,
			wantName:    "list_patients",
			wantArgs:    "{}",
		},
		{
			name:        "json code fence",
			input:       "```json\n{\"tool_call\": {\"name\": \"search\", \"arguments\": {\"q\":\"foo\"}}}\n```",
			wantToolCall: true,
			wantName:    "search",
			wantArgs:    `{"q":"foo"}`,
		},
		{
			name:        "bare code fence (no language)",
			input:       "```\n{\"tool_call\": {\"name\": \"x\", \"arguments\": {}}}\n```",
			wantToolCall: true,
			wantName:    "x",
			wantArgs:    "{}",
		},
		{
			name:        "plain conversational text",
			input:       "I cannot help with that.",
			wantToolCall: false,
		},
		{
			name:        "malformed JSON returns nil",
			input:       `{"tool_call": {"name": "broken`,
			wantToolCall: false,
		},
		{
			name:        "tool_call wrapper with empty name returns nil",
			input:       `{"tool_call": {"name": "", "arguments": {}}}`,
			wantToolCall: false,
		},
		{
			name:        "unrelated JSON object returns nil",
			input:       `{"unrelated": 123}`,
			wantToolCall: false,
		},
	}

	o := NewOrchestratorWithTools(nil)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := o.ParseToolCall(tc.input)
			if tc.wantToolCall {
				if got == nil {
					t.Fatalf("expected a tool call, got nil")
				}
				if got.Name != tc.wantName {
					t.Errorf("Name = %q, want %q", got.Name, tc.wantName)
				}
				// Arguments is a JSON string; compare by structural equivalence
				var gotJSON, wantJSON interface{}
				_ = json.Unmarshal([]byte(got.Arguments), &gotJSON)
				_ = json.Unmarshal([]byte(tc.wantArgs), &wantJSON)
				if !jsonEqual(gotJSON, wantJSON) {
					t.Errorf("Arguments = %q, want equivalent to %q", got.Arguments, tc.wantArgs)
				}
				if !strings.HasPrefix(got.ID, "tc_") {
					t.Errorf("ID should have tc_ prefix, got %q", got.ID)
				}
			} else if got != nil {
				t.Errorf("expected nil tool call, got %+v", got)
			}
		})
	}
}

func TestParseToolCall_FirstOfMultiple(t *testing.T) {
	// When two JSON code fences are present, the first one should be returned.
	input := "```json\n{\"tool_call\": {\"name\": \"first\", \"arguments\": {}}}\n```\n\n" +
		"```json\n{\"tool_call\": {\"name\": \"second\", \"arguments\": {}}}\n```"
	o := NewOrchestratorWithTools(nil)
	got, _ := o.ParseToolCall(input)
	if got == nil {
		t.Fatal("expected tool call")
	}
	if got.Name != "first" {
		t.Errorf("Name = %q, want first", got.Name)
	}
}

// jsonEqual compares two already-unmarshalled JSON values for structural equality.
func jsonEqual(a, b interface{}) bool {
	ba, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	return string(ba) == string(bb)
}
