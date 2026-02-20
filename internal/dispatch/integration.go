package dispatch

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Perttulands/ludus-magnus/internal/learningloop"
)

// PromptManifest is what production dispatch consumes.
type PromptManifest struct {
	Version     string           `json:"version"`
	Prompts     []DeployedPrompt `json:"prompts"`
	GeneratedAt string           `json:"generated_at"`
	SourceLoop  string           `json:"source_loop"`
}

// DeployedPrompt is a single prompt ready for production use.
type DeployedPrompt struct {
	ID           string  `json:"id"`
	SystemPrompt string  `json:"system_prompt"`
	Model        string  `json:"model"`
	Score        float64 `json:"score"`
	LineageID    string  `json:"lineage_id"`
	DeployedAt   string  `json:"deployed_at"`
}

// GenerateManifest creates a production-ready manifest from a training report.
func GenerateManifest(report *learningloop.TrainingReport) (*PromptManifest, error) {
	if report == nil {
		return nil, fmt.Errorf("report is nil")
	}
	if len(report.TrainedPrompts) == 0 {
		return nil, fmt.Errorf("report has no trained prompts")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	prompts := make([]DeployedPrompt, 0, len(report.TrainedPrompts))

	for _, tp := range report.TrainedPrompts {
		prompts = append(prompts, DeployedPrompt{
			ID:           tp.PromptID,
			SystemPrompt: tp.SystemPrompt,
			Model:        tp.Model,
			Score:        tp.AvgScore,
			LineageID:    tp.LineageID,
			DeployedAt:   now,
		})
	}

	return &PromptManifest{
		Version:     "1.0",
		Prompts:     prompts,
		GeneratedAt: now,
		SourceLoop:  report.LoopID,
	}, nil
}

// WriteManifest saves a manifest to the production dispatch directory.
func WriteManifest(manifest *PromptManifest, dir string) (string, error) {
	if manifest == nil {
		return "", fmt.Errorf("manifest is nil")
	}

	if dir == "" {
		dir = filepath.Join("state", "dispatch")
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create dispatch directory: %w", err)
	}

	filename := fmt.Sprintf("manifest_%s.json", manifest.SourceLoop)
	path := filepath.Join(dir, filename)

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode manifest: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("write manifest: %w", err)
	}

	return path, nil
}

// ReadManifest loads a manifest from disk.
func ReadManifest(path string) (*PromptManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest %q: %w", path, err)
	}

	var manifest PromptManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("decode manifest %q: %w", path, err)
	}

	return &manifest, nil
}

// BestPrompt returns the highest-scoring prompt from a manifest.
func (m *PromptManifest) BestPrompt() (DeployedPrompt, error) {
	if len(m.Prompts) == 0 {
		return DeployedPrompt{}, fmt.Errorf("manifest has no prompts")
	}

	best := m.Prompts[0]
	for _, p := range m.Prompts[1:] {
		if p.Score > best.Score {
			best = p
		}
	}
	return best, nil
}
