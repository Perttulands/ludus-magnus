package export_test

import (
	"strings"
	"testing"

	exporter "github.com/Perttulands/ludus-magnus/internal/export"
	"github.com/Perttulands/ludus-magnus/internal/state"
)

func TestEvidencePackJSON(t *testing.T) {
	t.Parallel()

	st := sampleStateWithEvidence("ses_abc12345", "art_abc12345")
	payload, err := exporter.EvidencePack(st, "ses_abc12345", "json")
	if err != nil {
		t.Fatalf("EvidencePack() error = %v", err)
	}

	if !strings.Contains(payload, `"session_id": "ses_abc12345"`) {
		t.Fatalf("expected session id in export, got %q", payload)
	}
	if !strings.Contains(payload, `"agent_versions"`) {
		t.Fatalf("expected agent version history in export, got %q", payload)
	}
	if !strings.Contains(payload, `"evaluation": {`) {
		t.Fatalf("expected evaluated artifacts in export, got %q", payload)
	}
	if !strings.HasSuffix(payload, "\n") {
		t.Fatalf("expected trailing newline, got %q", payload)
	}
}

func TestEvidencePackReturnsErrorWhenSessionMissing(t *testing.T) {
	t.Parallel()

	st := sampleStateWithEvidence("ses_abc12345", "art_abc12345")
	_, err := exporter.EvidencePack(st, "ses_missing", "json")
	if err == nil {
		t.Fatalf("expected missing session error")
	}
	if !strings.Contains(err.Error(), `session "ses_missing" not found`) {
		t.Fatalf("expected session not found error, got %v", err)
	}
}

func TestEvidencePackReturnsErrorOnUnsupportedFormat(t *testing.T) {
	t.Parallel()

	st := sampleStateWithEvidence("ses_abc12345", "art_abc12345")
	_, err := exporter.EvidencePack(st, "ses_abc12345", "yaml")
	if err == nil {
		t.Fatalf("expected unsupported format error")
	}
	if !strings.Contains(err.Error(), `unsupported export format "yaml"`) {
		t.Fatalf("expected unsupported format error, got %v", err)
	}
}

func sampleStateWithEvidence(sessionID string, artifactID string) state.State {
	st := state.NewState()
	st.Sessions[sessionID] = state.Session{
		ID:        sessionID,
		Mode:      "quickstart",
		Need:      "evidence export",
		CreatedAt: "2026-02-13T00:00:00Z",
		Status:    "active",
		Lineages: map[string]state.Lineage{
			"lin_1": {
				ID:        "lin_1",
				SessionID: sessionID,
				Name:      "main",
				Locked:    false,
				Agents: []state.Agent{
					{
						ID:        "agt_1",
						LineageID: "lin_1",
						Version:   1,
						Definition: state.AgentDefinition{
							SystemPrompt: "You are concise.",
							Model:        "gpt-4.1",
							Temperature:  0.7,
							MaxTokens:    2048,
							Tools:        []any{},
						},
						CreatedAt: "2026-02-13T00:01:00Z",
					},
				},
				Artifacts: []state.Artifact{
					{
						ID:      artifactID,
						AgentID: "agt_1",
						Input:   "test input",
						Output:  "test output",
						Evaluation: &state.Evaluation{
							Score:       8,
							Comment:     "good",
							EvaluatedAt: "2026-02-13T00:02:00Z",
						},
					},
				},
				Directives: state.Directives{Oneshot: []state.Directive{}, Sticky: []state.Directive{}},
			},
		},
	}
	return st
}
