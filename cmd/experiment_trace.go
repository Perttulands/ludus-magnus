package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/tabwriter"

	"github.com/Perttulands/chiron/internal/trace"
	"github.com/spf13/cobra"
)

func newExperimentTraceCmd() *cobra.Command {
	var (
		format  string
		runFlag string
		force   bool
	)

	cmd := &cobra.Command{
		Use:   "trace <experiment-dir>",
		Short: "Generate per-run trace analysis from transcripts",
		Long: `Parse experiment run transcripts and produce normalized trace artifacts.

Each analyzed run gets a trace.json in its run directory.
Use --run to analyze a single run, or omit to analyze all runs.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			expDir := args[0]
			return runTraceAnalysis(expDir, format, runFlag, force)
		},
	}

	cmd.Flags().StringVar(&format, "format", "table", "Output format: table, json")
	cmd.Flags().StringVar(&runFlag, "run", "", "Analyze only this run directory (model/condition-replica)")
	cmd.Flags().BoolVar(&force, "force", false, "Regenerate trace.json even if it already exists")

	return cmd
}

// traceRunRe matches run dirs like "model-name/condition-1".
var traceRunRe = regexp.MustCompile(`^(.+)-(\d+)$`)

func runTraceAnalysis(expDir, format, runFilter string, force bool) error {
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
			if traceRunRe.FindString(cellEntry.Name()) == "" {
				continue
			}

			runPath := filepath.Join(model, cellEntry.Name())
			if runFilter != "" && runPath != runFilter {
				continue
			}

			runDir := filepath.Join(modelDir, cellEntry.Name())

			// Check if trace.json already exists.
			traceJSONPath := filepath.Join(runDir, "trace.json")
			if !force {
				if _, err := os.Stat(traceJSONPath); err == nil {
					// Already traced — load and display.
					data, err := os.ReadFile(traceJSONPath)
					if err == nil {
						var t trace.RunTrace
						if json.Unmarshal(data, &t) == nil {
							traces = append(traces, &t)
							continue
						}
					}
				}
			}

			// Check if transcript exists.
			transcriptPath := trace.TranscriptPath(runDir)
			if _, err := os.Stat(transcriptPath); err != nil {
				fmt.Fprintf(os.Stderr, "warning: no transcript in %s\n", runDir)
				continue
			}

			t, err := trace.ParseFileWithMeta(runDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: parsing %s: %v\n", runDir, err)
				continue
			}

			// Apply heuristics to detect failure modes and extension evidence.
			trace.ApplyHeuristics(t)

			// Write trace.json artifact.
			data, err := json.MarshalIndent(t, "", "  ")
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: marshalling trace for %s: %v\n", runDir, err)
				continue
			}
			if err := os.WriteFile(traceJSONPath, data, 0o644); err != nil {
				fmt.Fprintf(os.Stderr, "warning: writing trace.json for %s: %v\n", runDir, err)
			}

			traces = append(traces, t)
		}
	}

	if len(traces) == 0 {
		fmt.Println("No runs found to trace.")
		return nil
	}

	switch format {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if len(traces) == 1 {
			return enc.Encode(traces[0])
		}
		return enc.Encode(traces)
	default:
		printTraceTable(traces)
	}

	return nil
}

func printTraceTable(traces []*trace.RunTrace) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "RUN\tMODEL\tTURNS\tTOOLS\tEDITS\tTOKENS IN\tTOKENS OUT\tOUTCOME\n")
	fmt.Fprintf(w, "---\t-----\t-----\t-----\t-----\t---------\t----------\t-------\n")

	for _, t := range traces {
		fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%d\t%d\t%d\t%s\n",
			t.RunID,
			t.Model,
			t.Metrics.TotalTurns,
			t.Metrics.TotalToolCalls,
			t.Metrics.EditCount,
			t.Metrics.TotalTokensIn,
			t.Metrics.TotalTokensOut,
			t.Outcome,
		)
	}
	w.Flush()

	fmt.Printf("\nTraced %d runs. trace.json artifacts written to each run directory.\n", len(traces))
}
