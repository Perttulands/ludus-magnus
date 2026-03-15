package experiment

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Name       string            `yaml:"name"`
	Version    int               `yaml:"version"`
	Models     []ModelConfig     `yaml:"models"`
	Conditions []ConditionConfig `yaml:"conditions"`
	Scenario   ScenarioConfig    `yaml:"scenario"`
	Execution  ExecutionConfig   `yaml:"execution"`
	Scoring    ScoringConfig     `yaml:"scoring"`
}

type ModelConfig struct {
	ID       string         `yaml:"id"`
	Provider string         `yaml:"provider"`
	Options  map[string]any `yaml:"options"`
}

type ConditionConfig struct {
	Name         string `yaml:"name"`
	SystemPrompt string `yaml:"system_prompt"`
}

type ScenarioConfig struct {
	Workspace  string   `yaml:"workspace"`
	UserPrompt string   `yaml:"user_prompt"`
	Tools      []string `yaml:"tools"`
}

type ExecutionConfig struct {
	Replicas       int    `yaml:"replicas"`
	Sandbox        string `yaml:"sandbox"`
	TraceCapture   bool   `yaml:"trace_capture"`
	BrStub         bool   `yaml:"br_stub"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
}

type ScoringConfig struct {
	AutoScorers []AutoScorerConfig `yaml:"auto_scorers"`
	Weights     map[string]float64 `yaml:"weights"`
}

type AutoScorerConfig struct {
	Type   string         `yaml:"type"`
	Config map[string]any `yaml:"config,omitempty"`
}

// LoadConfig reads and parses an experiment YAML file, applying defaults.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	// Deprecation: trace_capture is always true — transcripts are always persisted.
	// Warn if the field is explicitly set so users know it has no effect.
	if cfg.Execution.TraceCapture {
		log.Println("DEPRECATED: execution.trace_capture is deprecated and has no effect. " +
			"Transcripts are always captured as transcript.jsonl. " +
			"Remove trace_capture from your config to silence this warning.")
	}

	// Apply defaults
	if cfg.Execution.Replicas == 0 {
		cfg.Execution.Replicas = 3
	}
	if cfg.Execution.Sandbox == "" {
		cfg.Execution.Sandbox = "bwrap"
	}
	if cfg.Execution.TimeoutSeconds == 0 {
		cfg.Execution.TimeoutSeconds = 600
	}

	return &cfg, nil
}

// Validate checks that all referenced files exist and values are sensible.
func (c *Config) Validate(baseDir string) error {
	if c.Name == "" {
		return fmt.Errorf("name is required")
	}
	if len(c.Models) == 0 {
		return fmt.Errorf("at least one model is required")
	}
	if len(c.Conditions) == 0 {
		return fmt.Errorf("at least one condition is required")
	}

	for i, cond := range c.Conditions {
		p := filepath.Join(baseDir, cond.SystemPrompt)
		if _, err := os.Stat(p); err != nil {
			return fmt.Errorf("conditions[%d].system_prompt: %s not found", i, p)
		}
	}

	wp := filepath.Join(baseDir, c.Scenario.Workspace)
	info, err := os.Stat(wp)
	if err != nil {
		return fmt.Errorf("scenario.workspace: %s not found", wp)
	}
	if !info.IsDir() {
		return fmt.Errorf("scenario.workspace: %s is not a directory", wp)
	}

	up := filepath.Join(baseDir, c.Scenario.UserPrompt)
	if _, err := os.Stat(up); err != nil {
		return fmt.Errorf("scenario.user_prompt: %s not found", up)
	}

	if c.Execution.Replicas <= 0 {
		return fmt.Errorf("execution.replicas must be > 0")
	}
	if c.Execution.Sandbox != "bwrap" && c.Execution.Sandbox != "none" {
		return fmt.Errorf("execution.sandbox must be \"bwrap\" or \"none\"")
	}
	if c.Execution.TimeoutSeconds <= 0 {
		return fmt.Errorf("execution.timeout_seconds must be > 0")
	}

	return nil
}

// MatrixSize returns total number of runs (models × conditions × replicas).
func (c *Config) MatrixSize() int {
	return len(c.Models) * len(c.Conditions) * c.Execution.Replicas
}

// CellKey returns the directory name for a specific cell.
func CellKey(model, condition string, replica int) string {
	return fmt.Sprintf("%s_%s_%d", model, condition, replica)
}
