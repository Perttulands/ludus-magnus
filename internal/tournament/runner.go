package tournament

import (
	"context"
	"fmt"
	"time"

	"github.com/Perttulands/ludus-magnus/internal/challenge"
	"github.com/Perttulands/ludus-magnus/internal/harness"
	"github.com/Perttulands/ludus-magnus/internal/scoring"
	"github.com/Perttulands/ludus-magnus/internal/state"
)

// Contestant represents one prompt variant competing in a tournament.
type Contestant struct {
	ID         string                `json:"id"`
	LineageID  string                `json:"lineage_id"`
	Agent      state.Agent           `json:"agent"`
}

// Bout is the result of one contestant against one challenge.
type Bout struct {
	ContestantID string              `json:"contestant_id"`
	ChallengeID  string              `json:"challenge_id"`
	Output       string              `json:"output"`
	HarnessResult harness.SuiteResult `json:"harness_result"`
	CompositeScore scoring.Result     `json:"composite_score"`
	DurationMS   int                 `json:"duration_ms"`
	Error        string              `json:"error,omitempty"`
}

// Round groups all bouts for one challenge.
type Round struct {
	ChallengeID string `json:"challenge_id"`
	Bouts       []Bout `json:"bouts"`
}

// Executor is the function signature for running an agent on an input.
type Executor func(ctx context.Context, agent state.AgentDefinition, input string) (output string, durationMS int, err error)

// RunBout executes one contestant against one challenge and scores the result.
func RunBout(ctx context.Context, contestant Contestant, ch challenge.Challenge, exec Executor, weights scoring.Weights) Bout {
	start := time.Now()
	output, durationMS, err := exec(ctx, contestant.Agent.Definition, ch.Input)
	if durationMS == 0 {
		durationMS = int(time.Since(start).Milliseconds())
	}

	bout := Bout{
		ContestantID: contestant.ID,
		ChallengeID:  ch.ID,
		DurationMS:   durationMS,
	}

	if err != nil {
		bout.Error = err.Error()
		bout.CompositeScore = scoring.Score(scoring.Input{}, weights)
		bout.HarnessResult = harness.SuiteResult{SuiteID: ch.TestSuite.ID}
		return bout
	}

	bout.Output = output
	harnessResult := harness.RunSuite(ch.TestSuite, output)
	bout.HarnessResult = harnessResult

	bout.CompositeScore = scoring.Score(scoring.Input{
		HarnessResult: &harnessResult,
		DurationMS:    durationMS,
		MaxDurationMS: ch.MaxDurationMS,
	}, weights)

	return bout
}

// RunRound executes all contestants against one challenge.
func RunRound(ctx context.Context, contestants []Contestant, ch challenge.Challenge, exec Executor, weights scoring.Weights) Round {
	bouts := make([]Bout, 0, len(contestants))
	for _, c := range contestants {
		bout := RunBout(ctx, c, ch, exec, weights)
		bouts = append(bouts, bout)
	}
	return Round{
		ChallengeID: ch.ID,
		Bouts:       bouts,
	}
}

// RunAll executes all contestants against all challenges.
func RunAll(ctx context.Context, contestants []Contestant, challenges []challenge.Challenge, exec Executor, weights scoring.Weights) ([]Round, error) {
	if len(contestants) == 0 {
		return nil, fmt.Errorf("no contestants")
	}
	if len(challenges) == 0 {
		return nil, fmt.Errorf("no challenges")
	}

	rounds := make([]Round, 0, len(challenges))
	for _, ch := range challenges {
		round := RunRound(ctx, contestants, ch, exec, weights)
		rounds = append(rounds, round)
	}
	return rounds, nil
}
