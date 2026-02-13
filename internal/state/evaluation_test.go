package state_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Perttulands/ludus-magnus/internal/state"
)

func TestEvaluateArtifactSetsEvaluation(t *testing.T) {
	tempDir := t.TempDir()
	writeStateWithArtifact(t, tempDir, "art_abc12345", nil)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}

	if err := state.EvaluateArtifact("art_abc12345", 7, "good but needs improvement"); err != nil {
		t.Fatalf("evaluate artifact: %v", err)
	}

	got, err := state.Load("")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}

	evaluation := got.Sessions["ses_1"].Lineages["main"].Artifacts[0].Evaluation
	if evaluation == nil {
		t.Fatalf("expected evaluation to be set")
	}
	if evaluation.Score != 7 {
		t.Fatalf("expected score 7, got %d", evaluation.Score)
	}
	if evaluation.Comment != "good but needs improvement" {
		t.Fatalf("unexpected comment: %q", evaluation.Comment)
	}
	if evaluation.EvaluatedAt == "" {
		t.Fatalf("expected evaluated_at to be set")
	}
}

func TestEvaluateArtifactRejectsOutOfRangeScore(t *testing.T) {
	tempDir := t.TempDir()
	writeStateWithArtifact(t, tempDir, "art_abc12345", nil)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}

	if err := state.EvaluateArtifact("art_abc12345", 11, ""); err == nil {
		t.Fatalf("expected out-of-range score error")
	}
}

func TestEvaluateArtifactRejectsSecondEvaluation(t *testing.T) {
	tempDir := t.TempDir()
	writeStateWithArtifact(t, tempDir, "art_abc12345", &state.Evaluation{
		Score:       6,
		Comment:     "already scored",
		EvaluatedAt: "2026-02-13T12:00:00Z",
	})

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}

	if err := state.EvaluateArtifact("art_abc12345", 7, ""); err == nil {
		t.Fatalf("expected already evaluated error")
	}
}

func TestEvaluateArtifactRejectsNonUniqueArtifactID(t *testing.T) {
	tempDir := t.TempDir()

	st := state.NewState()
	st.Sessions["ses_1"] = state.Session{
		ID:        "ses_1",
		Mode:      "quickstart",
		Need:      "need-1",
		CreatedAt: "2026-02-13T10:30:00Z",
		Status:    "active",
		Lineages: map[string]state.Lineage{
			"main": {
				ID:        "lin_main_1",
				SessionID: "ses_1",
				Name:      "main",
				Artifacts: []state.Artifact{{ID: "art_dupe123", AgentID: "agt_1", Input: "in", Output: "out"}},
			},
		},
	}
	st.Sessions["ses_2"] = state.Session{
		ID:        "ses_2",
		Mode:      "training",
		Need:      "need-2",
		CreatedAt: "2026-02-13T10:31:00Z",
		Status:    "active",
		Lineages: map[string]state.Lineage{
			"A": {
				ID:        "lin_A_2",
				SessionID: "ses_2",
				Name:      "A",
				Artifacts: []state.Artifact{{ID: "art_dupe123", AgentID: "agt_2", Input: "in2", Output: "out2"}},
			},
		},
	}
	if err := state.Save(filepath.Join(tempDir, ".ludus-magnus", "state.json"), st); err != nil {
		t.Fatalf("save state: %v", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}

	err = state.EvaluateArtifact("art_dupe123", 7, "ambiguous")
	if err == nil {
		t.Fatalf("expected non-unique artifact id error")
	}
	if !strings.Contains(err.Error(), "not unique") {
		t.Fatalf("expected non-unique error, got %v", err)
	}
}

func writeStateWithArtifact(t *testing.T, tempDir string, artifactID string, evaluation *state.Evaluation) {
	t.Helper()

	st := state.NewState()
	st.Sessions["ses_1"] = state.Session{
		ID:        "ses_1",
		Mode:      "quickstart",
		Need:      "need",
		CreatedAt: "2026-02-13T10:30:00Z",
		Status:    "active",
		Lineages: map[string]state.Lineage{
			"main": {
				ID:        "lin_main",
				SessionID: "ses_1",
				Name:      "main",
				Artifacts: []state.Artifact{
					{
						ID:         artifactID,
						AgentID:    "agt_1",
						Input:      "input",
						Output:     "output",
						CreatedAt:  "2026-02-13T10:35:00Z",
						Evaluation: evaluation,
					},
				},
			},
		},
	}

	if err := state.Save(filepath.Join(tempDir, ".ludus-magnus", "state.json"), st); err != nil {
		t.Fatalf("save state: %v", err)
	}
}
