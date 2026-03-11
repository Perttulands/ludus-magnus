package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Perttulands/chiron/internal/experiment"
	"github.com/spf13/cobra"
)

func newExperimentScoreCmd() *cobra.Command {
	var scorers string

	cmd := &cobra.Command{
		Use:   "score <experiment-dir>",
		Short: "Run auto-scorers on experiment runs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			expDir := args[0]

			scorerNames := strings.Split(scorers, ",")
			autoScorers := buildScorers(scorerNames)
			if len(autoScorers) == 0 {
				return fmt.Errorf("no valid scorers specified")
			}

			return scoreExperiment(context.Background(), expDir, autoScorers)
		},
	}

	cmd.Flags().StringVar(&scorers, "scorers", "workspace_diff,br_stub", "Comma-separated list of scorers")
	return cmd
}

func buildScorers(names []string) []experiment.AutoScorer {
	var result []experiment.AutoScorer
	for _, name := range names {
		switch strings.TrimSpace(name) {
		case "workspace_diff":
			result = append(result, &experiment.WorkspaceDiffScorer{})
		case "br_stub":
			result = append(result, &experiment.BrStubScorer{})
		case "test_pass":
			result = append(result, &experiment.TestPassScorer{})
		}
	}
	return result
}

var condReplicaRe = regexp.MustCompile(`^(.+)-(\d+)$`)

func scoreExperiment(ctx context.Context, expDir string, scorers []experiment.AutoScorer) error {
	runsDir := filepath.Join(expDir, "runs")
	modelEntries, err := os.ReadDir(runsDir)
	if err != nil {
		return fmt.Errorf("reading runs: %w", err)
	}

	scored := 0
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
			if condReplicaRe.FindString(cellEntry.Name()) == "" {
				continue
			}

			runDir := filepath.Join(modelDir, cellEntry.Name())
			if err := scoreRun(ctx, runDir, scorers); err != nil {
				fmt.Fprintf(os.Stderr, "warning: scoring %s: %v\n", runDir, err)
				continue
			}
			scored++
		}
	}

	fmt.Printf("Scored %d runs\n", scored)
	return nil
}

func scoreRun(ctx context.Context, runDir string, scorers []experiment.AutoScorer) error {
	input := &experiment.ScorerInput{WorkDir: runDir}

	// Read workspace.diff if it exists
	if data, err := os.ReadFile(filepath.Join(runDir, "workspace.diff")); err == nil {
		input.WorkspaceDiff = string(data)
	}

	// Read br log if it exists
	if data, err := os.ReadFile(filepath.Join(runDir, "br.log")); err == nil {
		input.BrLog = string(data)
	}

	// Read response.txt
	if data, err := os.ReadFile(filepath.Join(runDir, "response.txt")); err == nil {
		input.ResponseText = string(data)
	}

	type scorerResult struct {
		Score   float64        `json:"score"`
		Details map[string]any `json:"details"`
	}

	results := map[string]scorerResult{}
	composite := 0.0

	for _, s := range scorers {
		score, details, err := s.Score(ctx, input)
		if err != nil {
			return fmt.Errorf("scorer %s: %w", s.Name(), err)
		}
		results[s.Name()] = scorerResult{Score: score, Details: details}
		composite += score
	}

	if len(scorers) > 0 {
		composite /= float64(len(scorers))
	}

	output := map[string]any{
		"composite": composite,
		"scorers":   results,
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(runDir, "scores.json"), data, 0644)
}
