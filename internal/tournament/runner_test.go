package tournament

import (
	"context"
	"fmt"
	"testing"

	"github.com/Perttulands/ludus-magnus/internal/challenge"
	"github.com/Perttulands/ludus-magnus/internal/harness"
	"github.com/Perttulands/ludus-magnus/internal/scoring"
	"github.com/Perttulands/ludus-magnus/internal/state"
)

func mockExec(output string) Executor {
	return func(_ context.Context, _ state.AgentDefinition, _ string) (string, int, error) {
		return output, 100, nil
	}
}

func failExec() Executor {
	return func(_ context.Context, _ state.AgentDefinition, _ string) (string, int, error) {
		return "", 0, fmt.Errorf("execution failed")
	}
}

func testChallenge() challenge.Challenge {
	return challenge.Challenge{
		ID:   "ch_1",
		Name: "test challenge",
		Type: challenge.TypeFeature,
		Input: "write hello world",
		Description: "test",
		TestSuite: harness.TestSuite{
			ID:   "ts_1",
			Name: "test suite",
			TestCases: []harness.TestCase{
				{ID: "tc_1", Name: "has hello", Type: "contains", Expected: "hello", Weight: 1.0},
				{ID: "tc_2", Name: "has world", Type: "contains", Expected: "world", Weight: 1.0},
			},
		},
		MaxDurationMS: 5000,
	}
}

func testContestants() []Contestant {
	return []Contestant{
		{
			ID: "c_1", LineageID: "lin_1",
			Agent: state.Agent{ID: "agt_1", Definition: state.AgentDefinition{SystemPrompt: "p1"}},
		},
		{
			ID: "c_2", LineageID: "lin_2",
			Agent: state.Agent{ID: "agt_2", Definition: state.AgentDefinition{SystemPrompt: "p2"}},
		},
	}
}

func TestRunBoutSuccess(t *testing.T) {
	contestant := testContestants()[0]
	ch := testChallenge()

	bout := RunBout(context.Background(), contestant, ch, mockExec("hello world"), scoring.DefaultWeights())

	if bout.Error != "" {
		t.Errorf("unexpected error: %s", bout.Error)
	}
	if bout.Output != "hello world" {
		t.Errorf("output = %q, want %q", bout.Output, "hello world")
	}
	if bout.HarnessResult.Passed != 2 {
		t.Errorf("expected 2 passed tests, got %d", bout.HarnessResult.Passed)
	}
	if bout.CompositeScore.Normalized < 1 {
		t.Errorf("normalized score should be >= 1, got %d", bout.CompositeScore.Normalized)
	}
}

func TestRunBoutFailure(t *testing.T) {
	contestant := testContestants()[0]
	ch := testChallenge()

	bout := RunBout(context.Background(), contestant, ch, failExec(), scoring.DefaultWeights())

	if bout.Error == "" {
		t.Error("expected error in bout")
	}
	if bout.Output != "" {
		t.Errorf("expected empty output, got %q", bout.Output)
	}
}

func TestRunBoutPartialMatch(t *testing.T) {
	contestant := testContestants()[0]
	ch := testChallenge()

	bout := RunBout(context.Background(), contestant, ch, mockExec("hello there"), scoring.DefaultWeights())
	if bout.HarnessResult.Passed != 1 {
		t.Errorf("expected 1 passed test, got %d", bout.HarnessResult.Passed)
	}
}

func TestRunRound(t *testing.T) {
	contestants := testContestants()
	ch := testChallenge()

	round := RunRound(context.Background(), contestants, ch, mockExec("hello world"), scoring.DefaultWeights())

	if round.ChallengeID != "ch_1" {
		t.Errorf("ChallengeID = %q, want %q", round.ChallengeID, "ch_1")
	}
	if len(round.Bouts) != 2 {
		t.Errorf("expected 2 bouts, got %d", len(round.Bouts))
	}
}

func TestRunAllSuccess(t *testing.T) {
	contestants := testContestants()
	challenges := []challenge.Challenge{testChallenge()}

	rounds, err := RunAll(context.Background(), contestants, challenges, mockExec("hello world"), scoring.DefaultWeights())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rounds) != 1 {
		t.Errorf("expected 1 round, got %d", len(rounds))
	}
}

func TestRunAllNoContestants(t *testing.T) {
	_, err := RunAll(context.Background(), nil, []challenge.Challenge{testChallenge()}, mockExec("x"), scoring.DefaultWeights())
	if err == nil {
		t.Error("expected error for no contestants")
	}
}

func TestRunAllNoChallenges(t *testing.T) {
	_, err := RunAll(context.Background(), testContestants(), nil, mockExec("x"), scoring.DefaultWeights())
	if err == nil {
		t.Error("expected error for no challenges")
	}
}
