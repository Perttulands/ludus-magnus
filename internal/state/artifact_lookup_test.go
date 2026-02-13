package state_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Perttulands/ludus-magnus/internal/state"
)

func TestLoadArtifactByIDReturnsUniqueArtifact(t *testing.T) {
	tempDir := t.TempDir()

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
				Artifacts: []state.Artifact{{ID: "art_unique1", AgentID: "agt_1", Input: "in", Output: "out"}},
			},
		},
	}
	if err := state.Save(filepath.Join(tempDir, ".ludus-magnus", "state.json"), st); err != nil {
		t.Fatalf("save state: %v", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	artifact, err := state.LoadArtifactByID("art_unique1")
	if err != nil {
		t.Fatalf("load artifact: %v", err)
	}
	if artifact.Output != "out" {
		t.Fatalf("expected output 'out', got %q", artifact.Output)
	}
}

func TestLoadArtifactByIDRejectsDuplicateID(t *testing.T) {
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
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	_, err = state.LoadArtifactByID("art_dupe123")
	if err == nil {
		t.Fatalf("expected non-unique artifact id error")
	}
	if !strings.Contains(err.Error(), "not unique") {
		t.Fatalf("expected non-unique error, got %v", err)
	}
}
