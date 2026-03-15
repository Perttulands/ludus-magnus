package trace

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
)

// ExperimentSummary is the dashboard-ready aggregate output for an experiment.
type ExperimentSummary struct {
	ExperimentName     string              `json:"experiment_name"`
	TotalRuns          int                 `json:"total_runs"`
	SuccessRate        float64             `json:"success_rate"`
	AvgTokensIn        float64             `json:"avg_tokens_in"`
	AvgTokensOut       float64             `json:"avg_tokens_out"`
	AvgTurns           float64             `json:"avg_turns"`
	AvgEdits           float64             `json:"avg_edits"`
	AvgToolCalls       float64             `json:"avg_tool_calls"`
	TestAfterEditRate  float64             `json:"test_after_edit_rate"`
	ReadOnlyFailRate   float64             `json:"read_only_fail_rate"`
	MostCommonFailure  string              `json:"most_common_failure,omitempty"`
	TopTools           map[string]int      `json:"top_tools"`
	FailureTagCounts   map[string]int      `json:"failure_tag_counts"`
	ModelAggregates    []ModelAggregate    `json:"model_aggregates"`
	ConditionAggs      []ConditionAgg      `json:"condition_aggregates"`
	CrossAggregates    []CrossAggregate    `json:"cross_aggregates"`
}

// ModelAggregate holds per-model stats.
type ModelAggregate struct {
	Model           string         `json:"model"`
	Runs            int            `json:"runs"`
	SuccessRate     float64        `json:"success_rate"`
	AvgTurns        float64        `json:"avg_turns"`
	AvgEdits        float64        `json:"avg_edits"`
	AvgTokensIn     float64        `json:"avg_tokens_in"`
	AvgTokensOut    float64        `json:"avg_tokens_out"`
	AvgToolCalls    float64        `json:"avg_tool_calls"`
	TopTools        map[string]int `json:"top_tools"`
	FailureTags     map[string]int `json:"failure_tags"`
}

// ConditionAgg holds per-condition stats.
type ConditionAgg struct {
	Condition       string         `json:"condition"`
	Runs            int            `json:"runs"`
	SuccessRate     float64        `json:"success_rate"`
	AvgTurns        float64        `json:"avg_turns"`
	AvgEdits        float64        `json:"avg_edits"`
	AvgTokensIn     float64        `json:"avg_tokens_in"`
	AvgTokensOut    float64        `json:"avg_tokens_out"`
	TopTools        map[string]int `json:"top_tools"`
}

// CrossAggregate holds model×condition stats.
type CrossAggregate struct {
	Model       string  `json:"model"`
	Condition   string  `json:"condition"`
	Runs        int     `json:"runs"`
	SuccessRate float64 `json:"success_rate"`
	AvgTurns    float64 `json:"avg_turns"`
	AvgEdits    float64 `json:"avg_edits"`
	AvgTokensIn float64 `json:"avg_tokens_in"`
}

// Aggregate computes experiment-level statistics from a slice of RunTraces.
func Aggregate(name string, traces []*RunTrace) *ExperimentSummary {
	s := &ExperimentSummary{
		ExperimentName:   name,
		TotalRuns:        len(traces),
		TopTools:         map[string]int{},
		FailureTagCounts: map[string]int{},
	}

	if len(traces) == 0 {
		return s
	}

	var (
		successes      int
		tokensIn       float64
		tokensOut      float64
		turns          float64
		edits          float64
		toolCalls      float64
		testAfterEdits int
		readOnlyFails  int
	)

	type key struct{ model, condition string }
	modelTraces := map[string][]*RunTrace{}
	condTraces := map[string][]*RunTrace{}
	crossTraces := map[key][]*RunTrace{}

	for _, t := range traces {
		if t.Outcome == "success" {
			successes++
		}
		tokensIn += float64(t.Metrics.TotalTokensIn)
		tokensOut += float64(t.Metrics.TotalTokensOut)
		turns += float64(t.Metrics.TotalTurns)
		edits += float64(t.Metrics.EditCount)
		toolCalls += float64(t.Metrics.TotalToolCalls)
		if t.Metrics.TestAfterEdit {
			testAfterEdits++
		}

		for name, count := range t.Metrics.ToolCallsByName {
			s.TopTools[name] += count
		}
		for _, tag := range t.FailureTags {
			s.FailureTagCounts[tag.Tag]++
			if tag.Tag == "read_only_no_edit" {
				readOnlyFails++
			}
		}

		// Extract condition from RunID (format: "model/condition").
		condition := ""
		if parts := splitRunID(t.RunID); len(parts) == 2 {
			condition = parts[1]
		}

		modelTraces[t.Model] = append(modelTraces[t.Model], t)
		if condition != "" {
			condTraces[condition] = append(condTraces[condition], t)
			crossTraces[key{t.Model, condition}] = append(crossTraces[key{t.Model, condition}], t)
		}
	}

	n := float64(len(traces))
	s.SuccessRate = float64(successes) / n
	s.AvgTokensIn = tokensIn / n
	s.AvgTokensOut = tokensOut / n
	s.AvgTurns = turns / n
	s.AvgEdits = edits / n
	s.AvgToolCalls = toolCalls / n
	if editsRuns := countRunsWithEdits(traces); editsRuns > 0 {
		s.TestAfterEditRate = float64(testAfterEdits) / float64(editsRuns)
	}
	s.ReadOnlyFailRate = float64(readOnlyFails) / n
	s.MostCommonFailure = mostCommonTag(s.FailureTagCounts)

	// Model aggregates.
	for model, ts := range modelTraces {
		s.ModelAggregates = append(s.ModelAggregates, aggregateModel(model, ts))
	}
	sort.Slice(s.ModelAggregates, func(i, j int) bool {
		return s.ModelAggregates[i].Model < s.ModelAggregates[j].Model
	})

	// Condition aggregates.
	for cond, ts := range condTraces {
		s.ConditionAggs = append(s.ConditionAggs, aggregateCondition(cond, ts))
	}
	sort.Slice(s.ConditionAggs, func(i, j int) bool {
		return s.ConditionAggs[i].Condition < s.ConditionAggs[j].Condition
	})

	// Cross aggregates.
	for k, ts := range crossTraces {
		s.CrossAggregates = append(s.CrossAggregates, aggregateCross(k.model, k.condition, ts))
	}
	sort.Slice(s.CrossAggregates, func(i, j int) bool {
		if s.CrossAggregates[i].Model != s.CrossAggregates[j].Model {
			return s.CrossAggregates[i].Model < s.CrossAggregates[j].Model
		}
		return s.CrossAggregates[i].Condition < s.CrossAggregates[j].Condition
	})

	return s
}

// WriteAggregates writes summary.json and runs.jsonl to the analysis/trace/ directory.
func WriteAggregates(expDir string, summary *ExperimentSummary, traces []*RunTrace) error {
	outDir := filepath.Join(expDir, "analysis", "trace")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("creating analysis dir: %w", err)
	}

	// Write summary.json.
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling summary: %w", err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "summary.json"), data, 0o644); err != nil {
		return fmt.Errorf("writing summary: %w", err)
	}

	// Write runs.jsonl (one trace per line).
	f, err := os.Create(filepath.Join(outDir, "runs.jsonl"))
	if err != nil {
		return fmt.Errorf("creating runs.jsonl: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	for _, t := range traces {
		if err := enc.Encode(t); err != nil {
			return fmt.Errorf("encoding run trace: %w", err)
		}
	}

	return nil
}

func aggregateModel(model string, traces []*RunTrace) ModelAggregate {
	a := ModelAggregate{
		Model:       model,
		Runs:        len(traces),
		TopTools:    map[string]int{},
		FailureTags: map[string]int{},
	}
	var successes int
	var turns, edits, tokensIn, tokensOut, tools float64
	for _, t := range traces {
		if t.Outcome == "success" {
			successes++
		}
		turns += float64(t.Metrics.TotalTurns)
		edits += float64(t.Metrics.EditCount)
		tokensIn += float64(t.Metrics.TotalTokensIn)
		tokensOut += float64(t.Metrics.TotalTokensOut)
		tools += float64(t.Metrics.TotalToolCalls)
		for name, count := range t.Metrics.ToolCallsByName {
			a.TopTools[name] += count
		}
		for _, tag := range t.FailureTags {
			a.FailureTags[tag.Tag]++
		}
	}
	n := float64(len(traces))
	a.SuccessRate = float64(successes) / n
	a.AvgTurns = roundTo(turns/n, 1)
	a.AvgEdits = roundTo(edits/n, 1)
	a.AvgTokensIn = roundTo(tokensIn/n, 0)
	a.AvgTokensOut = roundTo(tokensOut/n, 0)
	a.AvgToolCalls = roundTo(tools/n, 1)
	return a
}

func aggregateCondition(cond string, traces []*RunTrace) ConditionAgg {
	a := ConditionAgg{
		Condition: cond,
		Runs:      len(traces),
		TopTools:  map[string]int{},
	}
	var successes int
	var turns, edits, tokensIn, tokensOut float64
	for _, t := range traces {
		if t.Outcome == "success" {
			successes++
		}
		turns += float64(t.Metrics.TotalTurns)
		edits += float64(t.Metrics.EditCount)
		tokensIn += float64(t.Metrics.TotalTokensIn)
		tokensOut += float64(t.Metrics.TotalTokensOut)
		for name, count := range t.Metrics.ToolCallsByName {
			a.TopTools[name] += count
		}
	}
	n := float64(len(traces))
	a.SuccessRate = float64(successes) / n
	a.AvgTurns = roundTo(turns/n, 1)
	a.AvgEdits = roundTo(edits/n, 1)
	a.AvgTokensIn = roundTo(tokensIn/n, 0)
	a.AvgTokensOut = roundTo(tokensOut/n, 0)
	return a
}

func aggregateCross(model, cond string, traces []*RunTrace) CrossAggregate {
	a := CrossAggregate{Model: model, Condition: cond, Runs: len(traces)}
	var successes int
	var turns, edits, tokensIn float64
	for _, t := range traces {
		if t.Outcome == "success" {
			successes++
		}
		turns += float64(t.Metrics.TotalTurns)
		edits += float64(t.Metrics.EditCount)
		tokensIn += float64(t.Metrics.TotalTokensIn)
	}
	n := float64(len(traces))
	a.SuccessRate = float64(successes) / n
	a.AvgTurns = roundTo(turns/n, 1)
	a.AvgEdits = roundTo(edits/n, 1)
	a.AvgTokensIn = roundTo(tokensIn/n, 0)
	return a
}

func splitRunID(runID string) []string {
	for i := len(runID) - 1; i >= 0; i-- {
		if runID[i] == '/' {
			return []string{runID[:i], runID[i+1:]}
		}
	}
	return []string{runID}
}

func countRunsWithEdits(traces []*RunTrace) int {
	count := 0
	for _, t := range traces {
		if t.Metrics.EditCount > 0 {
			count++
		}
	}
	return count
}

func mostCommonTag(counts map[string]int) string {
	best := ""
	bestCount := 0
	for tag, count := range counts {
		if count > bestCount {
			best = tag
			bestCount = count
		}
	}
	return best
}

func roundTo(v float64, decimals int) float64 {
	pow := math.Pow(10, float64(decimals))
	return math.Round(v*pow) / pow
}
