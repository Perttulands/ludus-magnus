package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Perttulands/chiron/internal/experiment"
	"github.com/Perttulands/chiron/internal/sandbox"
	"github.com/spf13/cobra"
)

func newExperimentRunCmd() *cobra.Command {
	var (
		dryRun    bool
		model     string
		condition string
		replicas  int
	)

	cmd := &cobra.Command{
		Use:   "run <config.yaml>",
		Short: "Run an experiment matrix",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath := args[0]

			cfg, err := experiment.LoadConfig(configPath)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			baseDir := filepath.Dir(configPath)
			if abs, err := filepath.Abs(baseDir); err == nil {
				baseDir = abs
			}

			if err := cfg.Validate(baseDir); err != nil {
				return fmt.Errorf("validating config: %w", err)
			}

			opts := experiment.RunOptions{
				ModelFilter:     model,
				ConditionFilter: condition,
				ReplicaOverride: replicas,
				DryRun:          dryRun,
			}

			cells := experiment.MatrixCells(cfg, opts)
			fmt.Fprintf(os.Stderr, "Experiment: %s (%d cells)\n", cfg.Name, len(cells))

			executor := &sandbox.Executor{}
			runner := experiment.NewRunner(executor, baseDir)

			results, err := runner.Run(cmd.Context(), cfg, opts)
			if err != nil {
				return err
			}

			fmt.Fprintf(os.Stderr, "\nCompleted: %d/%d cells\n", len(results), len(cells))
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print plan without executing")
	cmd.Flags().StringVar(&model, "model", "", "Run only this model")
	cmd.Flags().StringVar(&condition, "condition", "", "Run only this condition")
	cmd.Flags().IntVar(&replicas, "replicas", 0, "Override replica count")

	return cmd
}
