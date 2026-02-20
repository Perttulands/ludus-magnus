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

var orchIDCounter int

func orchIDFunc(prefix string) string {
	orchIDCounter++
	return fmt.Sprintf("%s_%04d", prefix, orchIDCounter)
}

func orchContestants() []Contestant {
	return []Contestant{
		{ID: "c_1", LineageID: "lin_1", Agent: state.Agent{ID: "agt_1", Definition: state.AgentDefinition{SystemPrompt: "p1"}}},
		{ID: "c_2", LineageID: "lin_2", Agent: state.Agent{ID: "agt_2", Definition: state.AgentDefinition{SystemPrompt: "p2"}}},
	}
}

func orchChallenges() []challenge.Challenge {
	return []challenge.Challenge{
		{
			ID: "ch_1", Name: "test", Type: challenge.TypeFeature,
			Input: "write hello", Description: "test",
			TestSuite: harness.TestSuite{
				ID: "ts_1", Name: "ts",
				TestCases: []harness.TestCase{
					{ID: "tc_1", Name: "has hello", Type: "contains", Expected: "hello", Weight: 1.0},
				},
			},
		},
	}
}

func TestNewTournament(t *testing.T) {
	orchIDCounter = 0
	trn, err := New(Config{
		Name:    "Test Tournament",
		Weights: scoring.DefaultWeights(),
		IDFunc:  orchIDFunc,
	}, orchContestants(), orchChallenges())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if trn.Status != StatusPending {
		t.Errorf("status = %q, want %q", trn.Status, StatusPending)
	}
	if trn.Name != "Test Tournament" {
		t.Errorf("name = %q, want %q", trn.Name, "Test Tournament")
	}
}

func TestNewTournamentTooFewContestants(t *testing.T) {
	_, err := New(Config{IDFunc: orchIDFunc, Weights: scoring.DefaultWeights()},
		[]Contestant{{ID: "c_1"}}, orchChallenges())
	if err == nil {
		t.Error("expected error for < 2 contestants")
	}
}

func TestNewTournamentNoChallenges(t *testing.T) {
	_, err := New(Config{IDFunc: orchIDFunc, Weights: scoring.DefaultWeights()},
		orchContestants(), nil)
	if err == nil {
		t.Error("expected error for no challenges")
	}
}

func TestTournamentRun(t *testing.T) {
	orchIDCounter = 100
	trn, _ := New(Config{
		Weights: scoring.DefaultWeights(),
		IDFunc:  orchIDFunc,
	}, orchContestants(), orchChallenges())

	err := trn.Run(context.Background(), mockExec("hello world"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if trn.Status != StatusComplete {
		t.Errorf("status = %q, want %q", trn.Status, StatusComplete)
	}
	if len(trn.Rounds) != 1 {
		t.Errorf("expected 1 round, got %d", len(trn.Rounds))
	}
	if len(trn.Standings) != 2 {
		t.Errorf("expected 2 standings, got %d", len(trn.Standings))
	}
	if trn.Standings[0].Rank != 1 {
		t.Errorf("first standing rank = %d, want 1", trn.Standings[0].Rank)
	}
}

func TestTournamentRunNotPending(t *testing.T) {
	orchIDCounter = 200
	trn, _ := New(Config{Weights: scoring.DefaultWeights(), IDFunc: orchIDFunc},
		orchContestants(), orchChallenges())
	trn.Status = StatusComplete

	err := trn.Run(context.Background(), mockExec("x"))
	if err == nil {
		t.Error("expected error for non-pending tournament")
	}
}

func TestTournamentWinner(t *testing.T) {
	orchIDCounter = 300
	trn, _ := New(Config{Weights: scoring.DefaultWeights(), IDFunc: orchIDFunc},
		orchContestants(), orchChallenges())
	_ = trn.Run(context.Background(), mockExec("hello"))

	winner, err := trn.Winner()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if winner.Rank != 1 {
		t.Errorf("winner rank = %d, want 1", winner.Rank)
	}
}

func TestTournamentWinnerNotComplete(t *testing.T) {
	orchIDCounter = 400
	trn, _ := New(Config{Weights: scoring.DefaultWeights(), IDFunc: orchIDFunc},
		orchContestants(), orchChallenges())

	_, err := trn.Winner()
	if err == nil {
		t.Error("expected error for pending tournament")
	}
}

func TestTournamentTopN(t *testing.T) {
	orchIDCounter = 500
	trn, _ := New(Config{Weights: scoring.DefaultWeights(), IDFunc: orchIDFunc},
		orchContestants(), orchChallenges())
	_ = trn.Run(context.Background(), mockExec("hello"))

	top := trn.TopN(1)
	if len(top) != 1 {
		t.Errorf("TopN(1) returned %d, want 1", len(top))
	}

	topAll := trn.TopN(10)
	if len(topAll) != 2 {
		t.Errorf("TopN(10) returned %d, want 2 (capped)", len(topAll))
	}

	topZero := trn.TopN(0)
	if topZero != nil {
		t.Errorf("TopN(0) should return nil, got %v", topZero)
	}
}
