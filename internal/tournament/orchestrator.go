package tournament

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/Perttulands/ludus-magnus/internal/challenge"
	"github.com/Perttulands/ludus-magnus/internal/scoring"
)

// Status tracks the lifecycle of a tournament.
const (
	StatusPending   = "pending"
	StatusRunning   = "running"
	StatusScoring   = "scoring"
	StatusComplete  = "complete"
	StatusFailed    = "failed"
)

// Tournament represents a full competition between prompt variants.
type Tournament struct {
	ID           string              `json:"id"`
	Name         string              `json:"name"`
	Status       string              `json:"status"`
	Contestants  []Contestant        `json:"contestants"`
	Challenges   []challenge.Challenge `json:"challenges"`
	Rounds       []Round             `json:"rounds"`
	Standings    []Standing          `json:"standings"`
	Weights      scoring.Weights     `json:"weights"`
	CreatedAt    string              `json:"created_at"`
	CompletedAt  string              `json:"completed_at,omitempty"`
	DurationMS   int                 `json:"duration_ms"`
}

// Standing captures a contestant's aggregate tournament performance.
type Standing struct {
	ContestantID string  `json:"contestant_id"`
	LineageID    string  `json:"lineage_id"`
	TotalScore   float64 `json:"total_score"`
	AvgScore     float64 `json:"avg_score"`
	BoutsPlayed  int     `json:"bouts_played"`
	BoutsWon     int     `json:"bouts_won"`
	Rank         int     `json:"rank"`
}

// Config controls tournament creation.
type Config struct {
	Name    string
	Weights scoring.Weights
	IDFunc  func(string) string
}

// New creates a tournament in pending state.
func New(cfg Config, contestants []Contestant, challenges []challenge.Challenge) (*Tournament, error) {
	if len(contestants) < 2 {
		return nil, fmt.Errorf("tournament requires at least 2 contestants, got %d", len(contestants))
	}
	if len(challenges) == 0 {
		return nil, fmt.Errorf("tournament requires at least 1 challenge")
	}

	id := cfg.IDFunc("trn")
	name := cfg.Name
	if name == "" {
		name = fmt.Sprintf("Tournament %s", id)
	}

	return &Tournament{
		ID:          id,
		Name:        name,
		Status:      StatusPending,
		Contestants: contestants,
		Challenges:  challenges,
		Rounds:      []Round{},
		Standings:   []Standing{},
		Weights:     cfg.Weights,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// Run executes the full tournament: all rounds, then computes standings.
func (t *Tournament) Run(ctx context.Context, exec Executor) error {
	if t.Status != StatusPending {
		return fmt.Errorf("tournament %q is %s, not pending", t.ID, t.Status)
	}

	t.Status = StatusRunning
	start := time.Now()

	rounds, err := RunAll(ctx, t.Contestants, t.Challenges, exec, t.Weights)
	if err != nil {
		t.Status = StatusFailed
		return fmt.Errorf("run tournament: %w", err)
	}

	t.Rounds = rounds
	t.Status = StatusScoring
	t.Standings = computeStandings(t.Contestants, t.Rounds)
	t.Status = StatusComplete
	t.DurationMS = int(time.Since(start).Milliseconds())
	t.CompletedAt = time.Now().UTC().Format(time.RFC3339)

	return nil
}

// Winner returns the top-ranked contestant, or error if tournament is not complete.
func (t *Tournament) Winner() (Standing, error) {
	if t.Status != StatusComplete {
		return Standing{}, fmt.Errorf("tournament not complete (status: %s)", t.Status)
	}
	if len(t.Standings) == 0 {
		return Standing{}, fmt.Errorf("no standings")
	}
	return t.Standings[0], nil
}

// TopN returns the top N contestants by rank.
func (t *Tournament) TopN(n int) []Standing {
	if n <= 0 || len(t.Standings) == 0 {
		return nil
	}
	if n > len(t.Standings) {
		n = len(t.Standings)
	}
	return t.Standings[:n]
}

func computeStandings(contestants []Contestant, rounds []Round) []Standing {
	scores := map[string]*Standing{}

	for _, c := range contestants {
		scores[c.ID] = &Standing{
			ContestantID: c.ID,
			LineageID:    c.LineageID,
		}
	}

	// For each round, find the round winner
	for _, round := range rounds {
		var bestScore float64
		var bestID string
		for _, bout := range round.Bouts {
			s := scores[bout.ContestantID]
			if s == nil {
				continue
			}
			s.TotalScore += bout.CompositeScore.FinalScore
			s.BoutsPlayed++
			if bout.CompositeScore.FinalScore > bestScore {
				bestScore = bout.CompositeScore.FinalScore
				bestID = bout.ContestantID
			}
		}
		if bestID != "" {
			scores[bestID].BoutsWon++
		}
	}

	standings := make([]Standing, 0, len(scores))
	for _, s := range scores {
		if s.BoutsPlayed > 0 {
			s.AvgScore = s.TotalScore / float64(s.BoutsPlayed)
		}
		standings = append(standings, *s)
	}

	sort.Slice(standings, func(i, j int) bool {
		if standings[i].AvgScore != standings[j].AvgScore {
			return standings[i].AvgScore > standings[j].AvgScore
		}
		return standings[i].BoutsWon > standings[j].BoutsWon
	})

	for i := range standings {
		standings[i].Rank = i + 1
	}

	return standings
}
