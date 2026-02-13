package export_test

import (
	"strings"
	"testing"

	exporter "github.com/Perttulands/ludus-magnus/internal/export"
	"github.com/Perttulands/ludus-magnus/internal/state"
)

func TestAgentDefinitionJSON(t *testing.T) {
	t.Parallel()

	st := sampleStateWithAgent("agt_abc12345")
	payload, err := exporter.AgentDefinition(st, "agt_abc12345", "json")
	if err != nil {
		t.Fatalf("AgentDefinition() error = %v", err)
	}

	if !strings.Contains(payload, `"system_prompt": "You are concise."`) {
		t.Fatalf("expected json export to include system_prompt, got %q", payload)
	}
	if !strings.HasSuffix(payload, "\n") {
		t.Fatalf("expected trailing newline, got %q", payload)
	}
}

func TestAgentDefinitionPython(t *testing.T) {
	t.Parallel()

	st := sampleStateWithAgent("agt_abc12345")
	payload, err := exporter.AgentDefinition(st, "agt_abc12345", "python")
	if err != nil {
		t.Fatalf("AgentDefinition() error = %v", err)
	}

	if !strings.Contains(payload, "agent_definition = {") {
		t.Fatalf("expected python dict export, got %q", payload)
	}
	if !strings.Contains(payload, `"model": "gpt-4.1"`) {
		t.Fatalf("expected model field in python export, got %q", payload)
	}
	if !strings.Contains(payload, `"tools": [{"name": "search", "type": "function"}]`) {
		t.Fatalf("expected python export to preserve tools array, got %q", payload)
	}
}

func TestAgentDefinitionTypeScript(t *testing.T) {
	t.Parallel()

	st := sampleStateWithAgent("agt_abc12345")
	payload, err := exporter.AgentDefinition(st, "agt_abc12345", "typescript")
	if err != nil {
		t.Fatalf("AgentDefinition() error = %v", err)
	}

	if !strings.Contains(payload, "const agentDefinition: AgentDefinition") {
		t.Fatalf("expected TypeScript object export, got %q", payload)
	}
	if !strings.Contains(payload, "systemPrompt") {
		t.Fatalf("expected camelCase TypeScript fields, got %q", payload)
	}
	if !strings.Contains(payload, `tools: [{"name":"search","type":"function"}]`) {
		t.Fatalf("expected typescript export to preserve tools array, got %q", payload)
	}
}

func TestAgentDefinitionReturnsErrorWhenAgentMissing(t *testing.T) {
	t.Parallel()

	st := sampleStateWithAgent("agt_abc12345")
	_, err := exporter.AgentDefinition(st, "agt_missing", "json")
	if err == nil {
		t.Fatalf("expected missing agent error")
	}
	if !strings.Contains(err.Error(), `agent "agt_missing" not found`) {
		t.Fatalf("expected agent not found error, got %v", err)
	}
}

func sampleStateWithAgent(agentID string) state.State {
	st := state.NewState()
	st.Sessions["ses_1"] = state.Session{
		ID:        "ses_1",
		Mode:      "quickstart",
		Need:      "test",
		CreatedAt: "2026-02-13T00:00:00Z",
		Status:    "active",
		Lineages: map[string]state.Lineage{
			"lin_1": {
				ID:        "lin_1",
				SessionID: "ses_1",
				Name:      "main",
				Agents: []state.Agent{
					{
						ID:        agentID,
						LineageID: "lin_1",
						Version:   1,
						Definition: state.AgentDefinition{
							SystemPrompt: "You are concise.",
							Model:        "gpt-4.1",
							Temperature:  0.7,
							MaxTokens:    2048,
							Tools: []any{
								map[string]any{
									"name": "search",
									"type": "function",
								},
							},
						},
					},
				},
				Artifacts:  []state.Artifact{},
				Directives: state.Directives{Oneshot: []state.Directive{}, Sticky: []state.Directive{}},
			},
		},
	}
	return st
}
