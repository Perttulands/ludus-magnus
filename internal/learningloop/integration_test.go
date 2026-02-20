package learningloop

import (
	"path/filepath"
	"testing"

	"github.com/Perttulands/ludus-magnus/internal/state"
	"github.com/Perttulands/ludus-magnus/internal/tournament"
	"github.com/Perttulands/ludus-magnus/internal/training"
)

func completedLoop() *training.Loop {
	return &training.Loop{
		ID:     "loop_001",
		Status: training.StatusComplete,
		BestScore: 8.5,
		Contestants: []tournament.Contestant{
			{
				ID:        "c_1",
				LineageID: "lin_1",
				Agent: state.Agent{
					ID: "agt_1",
					Definition: state.AgentDefinition{
						SystemPrompt: "You are a helpful assistant.",
						Model:        "claude-sonnet-4-5",
					},
				},
			},
			{
				ID:        "c_2",
				LineageID: "lin_2",
				Agent: state.Agent{
					ID: "agt_2",
					Definition: state.AgentDefinition{
						SystemPrompt: "You are a coding expert.",
						Model:        "claude-sonnet-4-5",
					},
				},
			},
		},
		Generations: []training.Generation{
			{
				Number: 1,
				Winners: []tournament.Standing{
					{ContestantID: "c_1", LineageID: "lin_1", AvgScore: 8.5, BoutsPlayed: 3, BoutsWon: 2},
				},
				Eliminated: []tournament.Standing{
					{ContestantID: "c_2", LineageID: "lin_2", AvgScore: 6.0, BoutsPlayed: 3, BoutsWon: 1},
				},
			},
		},
	}
}

func TestExportReport(t *testing.T) {
	report, err := ExportReport(completedLoop())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.LoopID != "loop_001" {
		t.Errorf("LoopID = %q, want %q", report.LoopID, "loop_001")
	}
	if report.Generations != 1 {
		t.Errorf("Generations = %d, want 1", report.Generations)
	}
	if report.BestScore != 8.5 {
		t.Errorf("BestScore = %f, want 8.5", report.BestScore)
	}
	if len(report.TrainedPrompts) != 1 {
		t.Fatalf("expected 1 trained prompt, got %d", len(report.TrainedPrompts))
	}
	if report.TrainedPrompts[0].SystemPrompt != "You are a helpful assistant." {
		t.Errorf("unexpected prompt: %q", report.TrainedPrompts[0].SystemPrompt)
	}
}

func TestExportReportNilLoop(t *testing.T) {
	_, err := ExportReport(nil)
	if err == nil {
		t.Error("expected error for nil loop")
	}
}

func TestExportReportIncomplete(t *testing.T) {
	loop := completedLoop()
	loop.Status = training.StatusRunning
	_, err := ExportReport(loop)
	if err == nil {
		t.Error("expected error for incomplete loop")
	}
}

func TestExportReportNoGenerations(t *testing.T) {
	loop := completedLoop()
	loop.Generations = nil
	_, err := ExportReport(loop)
	if err == nil {
		t.Error("expected error for no generations")
	}
}

func TestWriteAndReadReport(t *testing.T) {
	dir := t.TempDir()
	report, _ := ExportReport(completedLoop())

	path, err := WriteReport(report, dir)
	if err != nil {
		t.Fatalf("write error: %v", err)
	}

	loaded, err := ReadReport(path)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	if loaded.LoopID != report.LoopID {
		t.Errorf("LoopID = %q, want %q", loaded.LoopID, report.LoopID)
	}
	if len(loaded.TrainedPrompts) != len(report.TrainedPrompts) {
		t.Errorf("prompt count = %d, want %d", len(loaded.TrainedPrompts), len(report.TrainedPrompts))
	}
}

func TestWriteReportNil(t *testing.T) {
	_, err := WriteReport(nil, t.TempDir())
	if err == nil {
		t.Error("expected error for nil report")
	}
}

func TestReadReportNonexistent(t *testing.T) {
	_, err := ReadReport(filepath.Join(t.TempDir(), "nonexistent.json"))
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}
