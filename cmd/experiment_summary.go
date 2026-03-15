package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/Perttulands/chiron/internal/trace"
	"github.com/spf13/cobra"
)

func newExperimentSummaryCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "summary <experiment-dir>",
		Short: "Aggregate trace analysis across all runs for dashboard consumption",
		Long: `Reads trace.json artifacts from each run directory and produces experiment-level
aggregates. Writes analysis/trace/summary.json and analysis/trace/runs.jsonl.

Run 'chiron experiment trace' first to generate per-run trace.json files.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSummary(args[0], format)
		},
	}

	cmd.Flags().StringVar(&format, "format", "table", "Output format: table, json")
	return cmd
}

func runSummary(expDir, format string) error {
	runsDir := filepath.Join(expDir, "runs")
	if _, err := os.Stat(runsDir); err != nil {
		return fmt.Errorf("runs directory not found: %w", err)
	}

	var traces []*trace.RunTrace

	modelEntries, err := os.ReadDir(runsDir)
	if err != nil {
		return fmt.Errorf("reading runs dir: %w", err)
	}

	for _, modelEntry := range modelEntries {
		if !modelEntry.IsDir() || strings.HasPrefix(modelEntry.Name(), ".") {
			continue
		}
		modelDir := filepath.Join(runsDir, modelEntry.Name())
		cellEntries, err := os.ReadDir(modelDir)
		if err != nil {
			continue
		}

		for _, cellEntry := range cellEntries {
			if !cellEntry.IsDir() {
				continue
			}
			runDir := filepath.Join(modelDir, cellEntry.Name())
			traceJSONPath := filepath.Join(runDir, "trace.json")

			data, err := os.ReadFile(traceJSONPath)
			if err != nil {
				// No trace.json — try parsing on the fly.
				transcriptPath := trace.TranscriptPath(runDir)
				if _, serr := os.Stat(transcriptPath); serr != nil {
					continue
				}
				t, perr := trace.ParseFileWithMeta(runDir)
				if perr != nil {
					fmt.Fprintf(os.Stderr, "warning: parsing %s: %v\n", runDir, perr)
					continue
				}
				trace.ApplyHeuristics(t)
				traces = append(traces, t)
				continue
			}

			var t trace.RunTrace
			if err := json.Unmarshal(data, &t); err != nil {
				fmt.Fprintf(os.Stderr, "warning: reading trace.json in %s: %v\n", runDir, err)
				continue
			}
			traces = append(traces, &t)
		}
	}

	if len(traces) == 0 {
		fmt.Println("No traced runs found. Run 'chiron experiment trace' first.")
		return nil
	}

	summary := trace.Aggregate(filepath.Base(expDir), traces)

	// Write aggregate files.
	if err := trace.WriteAggregates(expDir, summary, traces); err != nil {
		fmt.Fprintf(os.Stderr, "warning: writing aggregates: %v\n", err)
	}

	switch format {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(summary)
	default:
		printSummaryTable(summary)
	}

	return nil
}

func printSummaryTable(s *trace.ExperimentSummary) {
	fmt.Printf("Experiment: %s (%d runs)\n", s.ExperimentName, s.TotalRuns)
	fmt.Printf("Success rate: %.0f%%\n", s.SuccessRate*100)
	fmt.Printf("Avg tokens: %.0f in / %.0f out\n", s.AvgTokensIn, s.AvgTokensOut)
	fmt.Printf("Avg turns: %.1f  Avg edits: %.1f  Avg tools: %.1f\n", s.AvgTurns, s.AvgEdits, s.AvgToolCalls)
	fmt.Printf("Test-after-edit rate: %.0f%%  Read-only failure rate: %.0f%%\n",
		s.TestAfterEditRate*100, s.ReadOnlyFailRate*100)
	if s.MostCommonFailure != "" {
		fmt.Printf("Most common failure: %s\n", s.MostCommonFailure)
	}

	if len(s.ConditionAggs) > 1 {
		fmt.Printf("\nCondition comparison:\n")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "CONDITION\tRUNS\tSUCCESS\tAVG TURNS\tAVG EDITS\n")
		fmt.Fprintf(w, "---------\t----\t-------\t---------\t---------\n")
		for _, c := range s.ConditionAggs {
			fmt.Fprintf(w, "%s\t%d\t%.0f%%\t%.1f\t%.1f\n",
				c.Condition, c.Runs, c.SuccessRate*100, c.AvgTurns, c.AvgEdits)
		}
		w.Flush()
	}

	if len(s.FailureTagCounts) > 0 {
		fmt.Printf("\nFailure tags:\n")
		for tag, count := range s.FailureTagCounts {
			fmt.Printf("  %s: %d\n", tag, count)
		}
	}

	fmt.Printf("\nAggregates written to analysis/trace/\n")
}
