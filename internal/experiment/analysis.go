package experiment

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// AnalysisResult holds the full analysis of an experiment.
type AnalysisResult struct {
	ExperimentName string             `json:"experiment_name"`
	TotalRuns      int                `json:"total_runs"`
	Models         []ModelSummary     `json:"models"`
	Conditions     []ConditionSummary `json:"conditions"`
	Matrix         []CellSummary      `json:"matrix"`
}

// ModelSummary aggregates stats for a single model.
type ModelSummary struct {
	Model       string         `json:"model"`
	Runs        int            `json:"runs"`
	AvgTurns    float64        `json:"avg_turns"`
	AvgEdits    float64        `json:"avg_edits"`
	EditRate    float64        `json:"edit_rate"`
	AvgScore    float64        `json:"avg_score"`
	StdDevScore float64        `json:"std_dev_score"`
	ToolUsage   map[string]int `json:"tool_usage"`
}

// ConditionSummary aggregates stats for a single condition.
type ConditionSummary struct {
	Condition   string  `json:"condition"`
	Runs        int     `json:"runs"`
	AvgTurns    float64 `json:"avg_turns"`
	AvgEdits    float64 `json:"avg_edits"`
	EditRate    float64 `json:"edit_rate"`
	AvgScore    float64 `json:"avg_score"`
	StdDevScore float64 `json:"std_dev_score"`
}

// CellSummary aggregates stats for a model×condition cell.
type CellSummary struct {
	Model     string         `json:"model"`
	Condition string         `json:"condition"`
	Runs      int            `json:"runs"`
	AvgTurns  float64        `json:"avg_turns"`
	AvgEdits  float64        `json:"avg_edits"`
	EditRate  float64        `json:"edit_rate"`
	AvgScore  float64        `json:"avg_score"`
	StdDev    float64        `json:"std_dev"`
	ToolUsage map[string]int `json:"tool_usage"`
}

// runMeta mirrors the meta.json fields we care about.
type runMeta struct {
	NumTurns       int     `json:"num_turns"`
	EditWriteCalls int     `json:"edit_write_calls"`
	WallElapsedS   float64 `json:"wall_elapsed_s"`
	TotalToolCalls int     `json:"total_tool_calls"`
}

// runScores mirrors a scores.json file.
type runScores struct {
	Composite float64                    `json:"composite"`
	Scorers   map[string]runScorerResult `json:"scorers"`
}

type runScorerResult struct {
	Score   float64        `json:"score"`
	Details map[string]any `json:"details"`
}

type runData struct {
	model     string
	condition string
	meta      runMeta
	scores    *runScores
}

// conditionReplicaRe parses "<condition>-<replica>" directory names.
var conditionReplicaRe = regexp.MustCompile(`^(.+)-(\d+)$`)

// Analyze reads completed run results from an experiment directory and produces analysis.
func Analyze(experimentDir string) (*AnalysisResult, error) {
	runsDir := filepath.Join(experimentDir, "runs")
	if _, err := os.Stat(runsDir); err != nil {
		return nil, fmt.Errorf("runs directory not found: %w", err)
	}

	var runs []runData

	// Walk runs/<model>/<condition>-<replica>/
	modelEntries, err := os.ReadDir(runsDir)
	if err != nil {
		return nil, fmt.Errorf("reading runs dir: %w", err)
	}

	for _, modelEntry := range modelEntries {
		if !modelEntry.IsDir() || strings.HasPrefix(modelEntry.Name(), ".") {
			continue
		}
		model := modelEntry.Name()
		modelDir := filepath.Join(runsDir, model)

		cellEntries, err := os.ReadDir(modelDir)
		if err != nil {
			continue
		}

		for _, cellEntry := range cellEntries {
			if !cellEntry.IsDir() {
				continue
			}
			m := conditionReplicaRe.FindStringSubmatch(cellEntry.Name())
			if m == nil {
				continue
			}
			condition := m[1]

			metaPath := filepath.Join(modelDir, cellEntry.Name(), "meta.json")
			metaData, err := os.ReadFile(metaPath)
			if err != nil {
				continue
			}

			var meta runMeta
			if err := json.Unmarshal(metaData, &meta); err != nil {
				continue
			}

			rd := runData{model: model, condition: condition, meta: meta}

			scoresPath := filepath.Join(modelDir, cellEntry.Name(), "scores.json")
			if scoresData, err := os.ReadFile(scoresPath); err == nil {
				var sc runScores
				if json.Unmarshal(scoresData, &sc) == nil {
					rd.scores = &sc
				}
			}

			runs = append(runs, rd)
		}
	}

	return aggregate(filepath.Base(experimentDir), runs), nil
}

func aggregate(name string, runs []runData) *AnalysisResult {
	result := &AnalysisResult{
		ExperimentName: name,
		TotalRuns:      len(runs),
	}

	// Group by model, condition, and cell
	type key struct{ model, condition string }

	modelRuns := map[string][]runData{}
	condRuns := map[string][]runData{}
	cellRuns := map[key][]runData{}

	for _, r := range runs {
		modelRuns[r.model] = append(modelRuns[r.model], r)
		condRuns[r.condition] = append(condRuns[r.condition], r)
		cellRuns[key{r.model, r.condition}] = append(cellRuns[key{r.model, r.condition}], r)
	}

	for model, rs := range modelRuns {
		s := summarizeModel(model, rs)
		result.Models = append(result.Models, s)
	}

	for cond, rs := range condRuns {
		s := summarizeCondition(cond, rs)
		result.Conditions = append(result.Conditions, s)
	}

	for k, rs := range cellRuns {
		s := summarizeCell(k.model, k.condition, rs)
		result.Matrix = append(result.Matrix, s)
	}

	return result
}

func summarizeModel(model string, runs []runData) ModelSummary {
	s := ModelSummary{Model: model, Runs: len(runs), ToolUsage: map[string]int{}}
	var turns, edits float64
	var withEdits int
	var scores []float64

	for _, r := range runs {
		turns += float64(r.meta.NumTurns)
		edits += float64(r.meta.EditWriteCalls)
		if r.meta.EditWriteCalls > 0 {
			withEdits++
		}
		s.ToolUsage["total"] += r.meta.TotalToolCalls
		if r.scores != nil {
			scores = append(scores, r.scores.Composite)
		}
	}

	n := float64(len(runs))
	s.AvgTurns = turns / n
	s.AvgEdits = edits / n
	s.EditRate = float64(withEdits) / n
	if len(scores) > 0 {
		s.AvgScore = mean(scores)
		s.StdDevScore = stddev(scores)
	}
	return s
}

func summarizeCondition(cond string, runs []runData) ConditionSummary {
	s := ConditionSummary{Condition: cond, Runs: len(runs)}
	var turns, edits float64
	var withEdits int
	var scores []float64

	for _, r := range runs {
		turns += float64(r.meta.NumTurns)
		edits += float64(r.meta.EditWriteCalls)
		if r.meta.EditWriteCalls > 0 {
			withEdits++
		}
		if r.scores != nil {
			scores = append(scores, r.scores.Composite)
		}
	}

	n := float64(len(runs))
	s.AvgTurns = turns / n
	s.AvgEdits = edits / n
	s.EditRate = float64(withEdits) / n
	if len(scores) > 0 {
		s.AvgScore = mean(scores)
		s.StdDevScore = stddev(scores)
	}
	return s
}

func summarizeCell(model, cond string, runs []runData) CellSummary {
	s := CellSummary{Model: model, Condition: cond, Runs: len(runs), ToolUsage: map[string]int{}}
	var turns, edits float64
	var withEdits int
	var scores []float64

	for _, r := range runs {
		turns += float64(r.meta.NumTurns)
		edits += float64(r.meta.EditWriteCalls)
		if r.meta.EditWriteCalls > 0 {
			withEdits++
		}
		s.ToolUsage["total"] += r.meta.TotalToolCalls
		if r.scores != nil {
			scores = append(scores, r.scores.Composite)
		}
	}

	n := float64(len(runs))
	s.AvgTurns = turns / n
	s.AvgEdits = edits / n
	s.EditRate = float64(withEdits) / n
	if len(scores) > 0 {
		s.AvgScore = mean(scores)
		s.StdDev = stddev(scores)
	}
	return s
}

func mean(vals []float64) float64 {
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

func stddev(vals []float64) float64 {
	if len(vals) < 2 {
		return 0
	}
	m := mean(vals)
	sum := 0.0
	for _, v := range vals {
		d := v - m
		sum += d * d
	}
	return math.Sqrt(sum / float64(len(vals)))
}
