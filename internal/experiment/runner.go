package experiment

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Perttulands/chiron/internal/sandbox"
)

// Runner executes the model × condition × replica matrix defined by a Config.
type Runner struct {
	executor *sandbox.Executor
	baseDir  string
}

// NewRunner creates a Runner backed by the given executor.
// baseDir is the root directory of the experiment (where the YAML lives).
func NewRunner(executor *sandbox.Executor, baseDir string) *Runner {
	return &Runner{executor: executor, baseDir: baseDir}
}

// CellResult holds the outcome of a single matrix cell.
type CellResult struct {
	Model     string
	Condition string
	Replica   int
	Result    *sandbox.RunResult
	Scores    map[string]float64
}

// RunOptions control which cells are executed and how.
type RunOptions struct {
	// ModelFilter, if non-empty, restricts execution to models whose ID
	// matches this string exactly.
	ModelFilter string

	// ConditionFilter, if non-empty, restricts execution to conditions whose
	// Name matches this string exactly.
	ConditionFilter string

	// ReplicaOverride, if > 0, overrides cfg.Execution.Replicas.
	ReplicaOverride int

	// DryRun prints what would run without executing anything.
	DryRun bool
}

// Cell is a single point in the model × condition × replica matrix.
type Cell struct {
	Model     ModelConfig
	Condition ConditionConfig
	Replica   int
}

// MatrixCells returns the full list of cells for the given config and options,
// applying any filters specified in opts. It does not execute anything.
func MatrixCells(cfg *Config, opts RunOptions) []Cell {
	replicas := cfg.Execution.Replicas
	if opts.ReplicaOverride > 0 {
		replicas = opts.ReplicaOverride
	}

	var cells []Cell
	for _, model := range cfg.Models {
		if opts.ModelFilter != "" && model.ID != opts.ModelFilter {
			continue
		}
		for _, cond := range cfg.Conditions {
			if opts.ConditionFilter != "" && cond.Name != opts.ConditionFilter {
				continue
			}
			for r := 1; r <= replicas; r++ {
				cells = append(cells, Cell{
					Model:     model,
					Condition: cond,
					Replica:   r,
				})
			}
		}
	}
	return cells
}

// Run executes the matrix defined by cfg, filtered/overridden by opts.
// Already-completed cells (identified by the presence of meta.json) are
// skipped automatically.
func (r *Runner) Run(ctx context.Context, cfg *Config, opts RunOptions) ([]CellResult, error) {
	cells := MatrixCells(cfg, opts)
	total := len(cells)

	if opts.DryRun {
		for _, c := range cells {
			fmt.Printf("Would run: model=%s condition=%s replica=%d\n",
				c.Model.ID, c.Condition.Name, c.Replica)
		}
		return nil, nil
	}

	// Read user prompt once — it's shared across all cells.
	userPromptPath := filepath.Join(r.baseDir, cfg.Scenario.UserPrompt)
	userPromptBytes, err := os.ReadFile(userPromptPath)
	if err != nil {
		return nil, fmt.Errorf("reading user prompt %s: %w", userPromptPath, err)
	}
	userPrompt := string(userPromptBytes)

	scenarioDir := filepath.Join(r.baseDir, cfg.Scenario.Workspace)

	var results []CellResult

	for i, cell := range cells {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		outDir := cellOutputDir(r.baseDir, cell.Model.ID, cell.Condition.Name, cell.Replica)

		// Skip cells that have already been completed.
		if _, statErr := os.Stat(filepath.Join(outDir, "meta.json")); statErr == nil {
			log.Printf("[%d/%d][%d%%] SKIP model=%s condition=%s replica=%d (already done)",
				i+1, total, pct(i+1, total),
				cell.Model.ID, cell.Condition.Name, cell.Replica)
			continue
		}

		// Read per-condition system prompt.
		sysPromptPath := filepath.Join(r.baseDir, cell.Condition.SystemPrompt)
		sysPromptBytes, err := os.ReadFile(sysPromptPath)
		if err != nil {
			return results, fmt.Errorf("reading system prompt %s: %w", sysPromptPath, err)
		}

		sbCfg := sandbox.Config{
			Engine:  cfg.Execution.Sandbox,
			Tools:   cfg.Scenario.Tools,
			BrStub:  cfg.Execution.BrStub,
			Timeout: time.Duration(cfg.Execution.TimeoutSeconds) * time.Second,
		}

		wallStart := time.Now()
		res, runErr := r.executor.Run(
			ctx,
			sbCfg,
			cell.Model.ID,
			cell.Model.Provider,
			string(sysPromptBytes),
			userPrompt,
			scenarioDir,
		)
		wallDuration := time.Since(wallStart)

		if runErr != nil {
			return results, fmt.Errorf("cell model=%s condition=%s replica=%d: %w",
				cell.Model.ID, cell.Condition.Name, cell.Replica, runErr)
		}

		if err := saveResults(outDir, cell, res, wallDuration, cfg.Execution.Sandbox); err != nil {
			return results, fmt.Errorf("saving results for model=%s condition=%s replica=%d: %w",
				cell.Model.ID, cell.Condition.Name, cell.Replica, err)
		}

		log.Printf("[%d/%d][%d%%] model=%s condition=%s replica=%d turns=%d wall=%ds",
			i+1, total, pct(i+1, total),
			cell.Model.ID, cell.Condition.Name, cell.Replica,
			res.Turns, int(wallDuration.Seconds()))

		results = append(results, CellResult{
			Model:     cell.Model.ID,
			Condition: cell.Condition.Name,
			Replica:   cell.Replica,
			Result:    res,
			Scores:    nil, // scoring is a separate pass
		})
	}

	return results, nil
}

// cellOutputDir returns the canonical output directory for a matrix cell.
// safeModelID replaces ':' and '/' with '-' so it is usable as a path component.
func cellOutputDir(baseDir, modelID, conditionName string, replica int) string {
	safeModelID := strings.NewReplacer(":", "-", "/", "-").Replace(modelID)
	return filepath.Join(baseDir, "runs", safeModelID,
		fmt.Sprintf("%s-%d", conditionName, replica))
}

// saveResults writes all output artefacts for a completed cell.
func saveResults(outDir string, cell Cell, res *sandbox.RunResult, wall time.Duration, engine string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", outDir, err)
	}

	// raw-output.jsonl
	if err := os.WriteFile(filepath.Join(outDir, "raw-output.jsonl"),
		res.RawOutput, 0o644); err != nil {
		return err
	}

	// workspace.diff
	if err := os.WriteFile(filepath.Join(outDir, "workspace.diff"),
		[]byte(res.WorkspaceDiff), 0o644); err != nil {
		return err
	}

	// br-invocations.log
	if err := os.WriteFile(filepath.Join(outDir, "br-invocations.log"),
		[]byte(res.BrLog), 0o644); err != nil {
		return err
	}

	// result.json — structured summary intended for human review / quick diffs.
	resultDoc := map[string]any{
		"model":      cell.Model.ID,
		"condition":  cell.Condition.Name,
		"replica":    cell.Replica,
		"turns":      res.Turns,
		"tool_calls": len(res.ToolCalls),
		"tokens":     res.TokensIn + res.TokensOut,
		"duration":   float64(res.DurationMs) / 1000.0,
		"exit_code":  res.ExitCode,
		"edit_count": res.EditCount,
	}
	if err := writeJSON(filepath.Join(outDir, "result.json"), resultDoc); err != nil {
		return err
	}

	// meta.json — written last; its presence signals the cell is complete.
	metaDoc := map[string]any{
		"model":          cell.Model.ID,
		"condition":      cell.Condition.Name,
		"replica":        cell.Replica,
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
		"duration_ms":    wall.Milliseconds(),
		"turns":          res.Turns,
		"tokens_in":      res.TokensIn,
		"tokens_out":     res.TokensOut,
		"edit_count":     res.EditCount,
		"tool_calls":     len(res.ToolCalls),
		"sandbox_engine": engine,
	}
	if err := writeJSON(filepath.Join(outDir, "meta.json"), metaDoc); err != nil {
		return err
	}

	return nil
}

// writeJSON marshals v to indented JSON and writes it to path.
func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling %s: %w", path, err)
	}
	return os.WriteFile(path, data, 0o644)
}

// pct computes an integer percentage (0–100).
func pct(n, total int) int {
	if total == 0 {
		return 100
	}
	return n * 100 / total
}
