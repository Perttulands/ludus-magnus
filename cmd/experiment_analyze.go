package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/Perttulands/chiron/internal/experiment"
	"github.com/spf13/cobra"
)

func newExperimentAnalyzeCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "analyze <experiment-dir>",
		Short: "Analyze experiment results",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := experiment.Analyze(args[0])
			if err != nil {
				return err
			}

			switch format {
			case "json":
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(result)
			case "csv":
				return printCSV(result)
			default:
				printTable(result)
				return nil
			}
		},
	}

	cmd.Flags().StringVar(&format, "format", "table", "Output format: table, json, csv")
	return cmd
}

func printTable(r *experiment.AnalysisResult) {
	fmt.Printf("Experiment: %s (%d runs)\n\n", r.ExperimentName, r.TotalRuns)
	fmt.Printf("%-20s| %-22s| %4s | %9s | %9s | %9s | %9s\n",
		"Model", "Condition", "Runs", "Avg Turns", "Avg Edits", "Edit Rate", "Avg Score")
	fmt.Printf("%-20s|%-22s-|------|-----------|-----------|-----------|----------\n",
		"--------------------", "-----------------------")

	for _, c := range r.Matrix {
		scoreStr := "N/A"
		if c.AvgScore > 0 {
			scoreStr = fmt.Sprintf("%.3f", c.AvgScore)
		}
		fmt.Printf("%-20s| %-22s| %4d | %9.1f | %9.1f | %8.1f%% | %9s\n",
			c.Model, c.Condition, c.Runs, c.AvgTurns, c.AvgEdits, c.EditRate*100, scoreStr)
	}
}

func printCSV(r *experiment.AnalysisResult) error {
	w := csv.NewWriter(os.Stdout)
	defer w.Flush()

	if err := w.Write([]string{"model", "condition", "runs", "avg_turns", "avg_edits", "edit_rate", "avg_score", "std_dev"}); err != nil {
		return err
	}
	for _, c := range r.Matrix {
		if err := w.Write([]string{
			c.Model, c.Condition,
			strconv.Itoa(c.Runs),
			fmt.Sprintf("%.1f", c.AvgTurns),
			fmt.Sprintf("%.1f", c.AvgEdits),
			fmt.Sprintf("%.3f", c.EditRate),
			fmt.Sprintf("%.3f", c.AvgScore),
			fmt.Sprintf("%.3f", c.StdDev),
		}); err != nil {
			return err
		}
	}
	return nil
}
