package state_test

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/Perttulands/ludus-magnus/internal/state"
)

func TestAddArtifactAddsToMatchingLineageByID(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, ".ludus-magnus", "state.json")

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
				Artifacts: []state.Artifact{},
			},
		},
	}

	if err := state.Save(statePath, st); err != nil {
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

	err = state.AddArtifact("ses_1", "lin_main", state.Artifact{AgentID: "agt_1", Input: "in", Output: "out"})
	if err != nil {
		t.Fatalf("add artifact: %v", err)
	}

	got, err := state.Load("")
	if err != nil {
		t.Fatalf("load updated state: %v", err)
	}

	artifacts := got.Sessions["ses_1"].Lineages["main"].Artifacts
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	if matched := regexp.MustCompile(`^art_[a-f0-9]{8}$`).MatchString(artifacts[0].ID); !matched {
		t.Fatalf("artifact id %q does not match expected prefix pattern", artifacts[0].ID)
	}
	if artifacts[0].CreatedAt == "" {
		t.Fatalf("expected created_at to be set")
	}
}

func TestAddArtifactErrorsWhenSessionMissing(t *testing.T) {
	tempDir := t.TempDir()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}

	if err := state.AddArtifact("ses_missing", "lin_main", state.Artifact{}); err == nil {
		t.Fatalf("expected session not found error")
	}
}

func TestAddArtifactErrorsWhenLineageMissing(t *testing.T) {
	tempDir := t.TempDir()

	st := state.NewState()
	st.Sessions["ses_1"] = state.Session{
		ID:        "ses_1",
		Mode:      "quickstart",
		Need:      "need",
		CreatedAt: "2026-02-13T10:30:00Z",
		Status:    "active",
		Lineages:  map[string]state.Lineage{},
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

	if err := state.AddArtifact("ses_1", "lin_missing", state.Artifact{}); err == nil {
		t.Fatalf("expected lineage not found error")
	}
}
