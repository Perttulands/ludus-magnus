package dispatch

import (
	"testing"

	"github.com/Perttulands/ludus-magnus/internal/learningloop"
)

func testReport() *learningloop.TrainingReport {
	return &learningloop.TrainingReport{
		LoopID:      "loop_001",
		Generations: 3,
		BestScore:   8.5,
		TrainedPrompts: []learningloop.TrainedPrompt{
			{
				PromptID:     "loop_001_c_1",
				SystemPrompt: "You are a helpful assistant.",
				Model:        "claude-sonnet-4-5",
				AvgScore:     8.5,
				BoutsPlayed:  9,
				BoutsWon:     6,
				LineageID:    "lin_1",
			},
			{
				PromptID:     "loop_001_c_2",
				SystemPrompt: "You are a coding expert.",
				Model:        "claude-sonnet-4-5",
				AvgScore:     7.0,
				BoutsPlayed:  9,
				BoutsWon:     3,
				LineageID:    "lin_2",
			},
		},
	}
}

func TestGenerateManifest(t *testing.T) {
	manifest, err := GenerateManifest(testReport())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if manifest.Version != "1.0" {
		t.Errorf("Version = %q, want %q", manifest.Version, "1.0")
	}
	if manifest.SourceLoop != "loop_001" {
		t.Errorf("SourceLoop = %q, want %q", manifest.SourceLoop, "loop_001")
	}
	if len(manifest.Prompts) != 2 {
		t.Fatalf("expected 2 prompts, got %d", len(manifest.Prompts))
	}
	if manifest.Prompts[0].ID != "loop_001_c_1" {
		t.Errorf("first prompt ID = %q, want %q", manifest.Prompts[0].ID, "loop_001_c_1")
	}
}

func TestGenerateManifestNilReport(t *testing.T) {
	_, err := GenerateManifest(nil)
	if err == nil {
		t.Error("expected error for nil report")
	}
}

func TestGenerateManifestEmptyPrompts(t *testing.T) {
	report := &learningloop.TrainingReport{LoopID: "x"}
	_, err := GenerateManifest(report)
	if err == nil {
		t.Error("expected error for empty prompts")
	}
}

func TestWriteAndReadManifest(t *testing.T) {
	dir := t.TempDir()
	manifest, _ := GenerateManifest(testReport())

	path, err := WriteManifest(manifest, dir)
	if err != nil {
		t.Fatalf("write error: %v", err)
	}

	loaded, err := ReadManifest(path)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	if loaded.SourceLoop != manifest.SourceLoop {
		t.Errorf("SourceLoop = %q, want %q", loaded.SourceLoop, manifest.SourceLoop)
	}
	if len(loaded.Prompts) != 2 {
		t.Errorf("expected 2 prompts, got %d", len(loaded.Prompts))
	}
}

func TestWriteManifestNil(t *testing.T) {
	_, err := WriteManifest(nil, t.TempDir())
	if err == nil {
		t.Error("expected error for nil manifest")
	}
}

func TestReadManifestNonexistent(t *testing.T) {
	_, err := ReadManifest("/tmp/nonexistent_manifest_test.json")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestBestPrompt(t *testing.T) {
	manifest, _ := GenerateManifest(testReport())
	best, err := manifest.BestPrompt()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if best.Score != 8.5 {
		t.Errorf("best score = %f, want 8.5", best.Score)
	}
	if best.ID != "loop_001_c_1" {
		t.Errorf("best ID = %q, want %q", best.ID, "loop_001_c_1")
	}
}

func TestBestPromptEmpty(t *testing.T) {
	manifest := &PromptManifest{}
	_, err := manifest.BestPrompt()
	if err == nil {
		t.Error("expected error for empty manifest")
	}
}
