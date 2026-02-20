package learningloop

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Perttulands/ludus-magnus/internal/tournament"
	"github.com/Perttulands/ludus-magnus/internal/training"
)

// TrainedPrompt represents a prompt ready for learning loop consumption.
type TrainedPrompt struct {
	PromptID     string  `json:"prompt_id"`
	SystemPrompt string  `json:"system_prompt"`
	Model        string  `json:"model"`
	AvgScore     float64 `json:"avg_score"`
	BoutsPlayed  int     `json:"bouts_played"`
	BoutsWon     int     `json:"bouts_won"`
	Generation   int     `json:"generation"`
	LineageID    string  `json:"lineage_id"`
	TrainedAt    string  `json:"trained_at"`
}

// TrainingReport summarizes a training run for the learning loop.
type TrainingReport struct {
	LoopID         string          `json:"loop_id"`
	Generations    int             `json:"generations"`
	BestScore      float64         `json:"best_score"`
	TrainedPrompts []TrainedPrompt `json:"trained_prompts"`
	CreatedAt      string          `json:"created_at"`
}

// ExportReport generates a training report from a completed loop.
func ExportReport(loop *training.Loop) (*TrainingReport, error) {
	if loop == nil {
		return nil, fmt.Errorf("loop is nil")
	}
	if !loop.IsComplete() {
		return nil, fmt.Errorf("loop is not complete (status: %s)", loop.Status)
	}
	if len(loop.Generations) == 0 {
		return nil, fmt.Errorf("loop has no generations")
	}

	// Extract winner prompts from the final generation standings
	lastGen := loop.Generations[len(loop.Generations)-1]
	prompts := make([]TrainedPrompt, 0, len(lastGen.Winners))

	for _, winner := range lastGen.Winners {
		contestant := findContestant(loop.Contestants, winner.ContestantID)
		if contestant == nil {
			continue
		}

		prompts = append(prompts, TrainedPrompt{
			PromptID:     fmt.Sprintf("%s_%s", loop.ID, winner.ContestantID),
			SystemPrompt: contestant.Agent.Definition.SystemPrompt,
			Model:        contestant.Agent.Definition.Model,
			AvgScore:     winner.AvgScore,
			BoutsPlayed:  winner.BoutsPlayed,
			BoutsWon:     winner.BoutsWon,
			Generation:   lastGen.Number,
			LineageID:    winner.LineageID,
			TrainedAt:    time.Now().UTC().Format(time.RFC3339),
		})
	}

	return &TrainingReport{
		LoopID:         loop.ID,
		Generations:    len(loop.Generations),
		BestScore:      loop.BestScore,
		TrainedPrompts: prompts,
		CreatedAt:      time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// WriteReport saves a training report as JSON to disk.
func WriteReport(report *TrainingReport, dir string) (string, error) {
	if report == nil {
		return "", fmt.Errorf("report is nil")
	}

	if dir == "" {
		dir = filepath.Join("state", "trained-prompts")
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create report directory: %w", err)
	}

	filename := fmt.Sprintf("report_%s.json", report.LoopID)
	path := filepath.Join(dir, filename)

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode report: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("write report: %w", err)
	}

	return path, nil
}

// ReadReport loads a training report from disk.
func ReadReport(path string) (*TrainingReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read report %q: %w", path, err)
	}

	var report TrainingReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("decode report %q: %w", path, err)
	}

	return &report, nil
}

func findContestant(contestants []tournament.Contestant, id string) *tournament.Contestant {
	for i, c := range contestants {
		if c.ID == id {
			return &contestants[i]
		}
	}
	return nil
}
