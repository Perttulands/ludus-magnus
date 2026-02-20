package training

import (
	"context"
	"fmt"
	"time"

	"github.com/Perttulands/ludus-magnus/internal/challenge"
	"github.com/Perttulands/ludus-magnus/internal/scoring"
	"github.com/Perttulands/ludus-magnus/internal/selection"
	"github.com/Perttulands/ludus-magnus/internal/tournament"
)

// Status tracks the training loop lifecycle.
const (
	StatusIdle     = "idle"
	StatusRunning  = "running"
	StatusPaused   = "paused"
	StatusComplete = "complete"
	StatusFailed   = "failed"
)

// Config controls the training loop behavior.
type Config struct {
	MaxGenerations   int                `json:"max_generations"`
	SelectionCount   int                `json:"selection_count"` // how many winners to keep
	SelectionStrategy string            `json:"selection_strategy"`
	Weights          scoring.Weights    `json:"weights"`
	TargetScore      float64            `json:"target_score"` // stop if avg score >= this
	IDFunc           func(string) string `json:"-"`
}

// DefaultConfig returns sensible training defaults.
func DefaultConfig(idFunc func(string) string) Config {
	return Config{
		MaxGenerations:    10,
		SelectionCount:    2,
		SelectionStrategy: selection.StrategyTruncation,
		Weights:           scoring.DefaultWeights(),
		TargetScore:       9.0,
		IDFunc:            idFunc,
	}
}

// Generation records one generation of the training loop.
type Generation struct {
	Number      int                     `json:"number"`
	Tournament  tournament.Tournament   `json:"tournament"`
	Winners     []tournament.Standing   `json:"winners"`
	Eliminated  []tournament.Standing   `json:"eliminated"`
	BestScore   float64                 `json:"best_score"`
	AvgScore    float64                 `json:"avg_score"`
	DurationMS  int                     `json:"duration_ms"`
	CompletedAt string                  `json:"completed_at"`
}

// Loop represents a complete training run.
type Loop struct {
	ID           string                   `json:"id"`
	Status       string                   `json:"status"`
	Config       Config                   `json:"config"`
	Generations  []Generation             `json:"generations"`
	Contestants  []tournament.Contestant  `json:"contestants"`
	BestScore    float64                  `json:"best_score"`
	CreatedAt    string                   `json:"created_at"`
	CompletedAt  string                   `json:"completed_at,omitempty"`
}

// Mutator generates new contestant variants from winners.
type Mutator func(ctx context.Context, winners []tournament.Standing, contestants []tournament.Contestant) ([]tournament.Contestant, error)

// NewLoop creates a training loop ready to run.
func NewLoop(cfg Config, contestants []tournament.Contestant) (*Loop, error) {
	if len(contestants) < 2 {
		return nil, fmt.Errorf("training requires at least 2 contestants, got %d", len(contestants))
	}
	if cfg.MaxGenerations <= 0 {
		return nil, fmt.Errorf("max_generations must be > 0")
	}
	if cfg.SelectionCount <= 0 || cfg.SelectionCount >= len(contestants) {
		return nil, fmt.Errorf("selection_count must be between 1 and %d", len(contestants)-1)
	}

	return &Loop{
		ID:          cfg.IDFunc("loop"),
		Status:      StatusIdle,
		Config:      cfg,
		Contestants: contestants,
		Generations: []Generation{},
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// RunGeneration executes one generation of the training loop.
func (l *Loop) RunGeneration(ctx context.Context, challenges []challenge.Challenge, exec tournament.Executor) (*Generation, error) {
	if l.Status == StatusComplete || l.Status == StatusFailed {
		return nil, fmt.Errorf("loop is %s", l.Status)
	}

	l.Status = StatusRunning
	genNum := len(l.Generations) + 1
	start := time.Now()

	// Create and run tournament
	trn, err := tournament.New(tournament.Config{
		Name:    fmt.Sprintf("Generation %d", genNum),
		Weights: l.Config.Weights,
		IDFunc:  l.Config.IDFunc,
	}, l.Contestants, challenges)
	if err != nil {
		l.Status = StatusFailed
		return nil, fmt.Errorf("create tournament: %w", err)
	}

	if err := trn.Run(ctx, exec); err != nil {
		l.Status = StatusFailed
		return nil, fmt.Errorf("run tournament: %w", err)
	}

	// Select winners
	sel, err := selection.NewSelector(l.Config.SelectionStrategy)
	if err != nil {
		l.Status = StatusFailed
		return nil, fmt.Errorf("create selector: %w", err)
	}

	winners := sel.Select(trn.Standings, l.Config.SelectionCount)
	_, eliminated := selection.Partition(trn.Standings, winners)

	// Compute stats
	var bestScore, totalScore float64
	for _, s := range trn.Standings {
		totalScore += s.AvgScore
		if s.AvgScore > bestScore {
			bestScore = s.AvgScore
		}
	}
	avgScore := 0.0
	if len(trn.Standings) > 0 {
		avgScore = totalScore / float64(len(trn.Standings))
	}

	if bestScore > l.BestScore {
		l.BestScore = bestScore
	}

	gen := Generation{
		Number:      genNum,
		Tournament:  *trn,
		Winners:     winners,
		Eliminated:  eliminated,
		BestScore:   bestScore,
		AvgScore:    avgScore,
		DurationMS:  int(time.Since(start).Milliseconds()),
		CompletedAt: time.Now().UTC().Format(time.RFC3339),
	}

	l.Generations = append(l.Generations, gen)

	// Check termination
	if genNum >= l.Config.MaxGenerations {
		l.Status = StatusComplete
		l.CompletedAt = time.Now().UTC().Format(time.RFC3339)
	} else if bestScore >= l.Config.TargetScore {
		l.Status = StatusComplete
		l.CompletedAt = time.Now().UTC().Format(time.RFC3339)
	} else {
		l.Status = StatusPaused
	}

	return &gen, nil
}

// SetContestants replaces the contestant pool (used after mutation).
func (l *Loop) SetContestants(contestants []tournament.Contestant) {
	l.Contestants = contestants
}

// IsComplete returns whether the loop has finished.
func (l *Loop) IsComplete() bool {
	return l.Status == StatusComplete || l.Status == StatusFailed
}

// CurrentGeneration returns the number of the current generation.
func (l *Loop) CurrentGeneration() int {
	return len(l.Generations)
}
