package training

import (
	"context"
	"fmt"
	"testing"

	"github.com/Perttulands/ludus-magnus/internal/challenge"
	"github.com/Perttulands/ludus-magnus/internal/harness"
	"github.com/Perttulands/ludus-magnus/internal/scoring"
	"github.com/Perttulands/ludus-magnus/internal/state"
	"github.com/Perttulands/ludus-magnus/internal/tournament"
)

var loopIDCounter int

func loopIDFunc(prefix string) string {
	loopIDCounter++
	return fmt.Sprintf("%s_%04d", prefix, loopIDCounter)
}

func loopContestants() []tournament.Contestant {
	return []tournament.Contestant{
		{ID: "c_1", LineageID: "lin_1", Agent: state.Agent{ID: "agt_1", Definition: state.AgentDefinition{SystemPrompt: "p1"}}},
		{ID: "c_2", LineageID: "lin_2", Agent: state.Agent{ID: "agt_2", Definition: state.AgentDefinition{SystemPrompt: "p2"}}},
		{ID: "c_3", LineageID: "lin_3", Agent: state.Agent{ID: "agt_3", Definition: state.AgentDefinition{SystemPrompt: "p3"}}},
	}
}

func loopChallenges() []challenge.Challenge {
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

func mockLoopExec(output string) tournament.Executor {
	return func(_ context.Context, _ state.AgentDefinition, _ string) (string, int, error) {
		return output, 100, nil
	}
}

func TestNewLoop(t *testing.T) {
	loopIDCounter = 0
	cfg := DefaultConfig(loopIDFunc)
	cfg.SelectionCount = 1
	loop, err := NewLoop(cfg, loopContestants())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loop.Status != StatusIdle {
		t.Errorf("status = %q, want %q", loop.Status, StatusIdle)
	}
}

func TestNewLoopTooFewContestants(t *testing.T) {
	cfg := DefaultConfig(loopIDFunc)
	_, err := NewLoop(cfg, []tournament.Contestant{{ID: "c_1"}})
	if err == nil {
		t.Error("expected error for < 2 contestants")
	}
}

func TestNewLoopBadMaxGenerations(t *testing.T) {
	cfg := DefaultConfig(loopIDFunc)
	cfg.MaxGenerations = 0
	_, err := NewLoop(cfg, loopContestants())
	if err == nil {
		t.Error("expected error for 0 max_generations")
	}
}

func TestNewLoopBadSelectionCount(t *testing.T) {
	cfg := DefaultConfig(loopIDFunc)
	cfg.SelectionCount = 5 // more than contestants
	_, err := NewLoop(cfg, loopContestants())
	if err == nil {
		t.Error("expected error for selection_count >= contestants")
	}
}

func TestRunGeneration(t *testing.T) {
	loopIDCounter = 100
	cfg := DefaultConfig(loopIDFunc)
	cfg.SelectionCount = 1
	cfg.MaxGenerations = 5
	cfg.TargetScore = 100 // unreachable, so we test pausing
	loop, _ := NewLoop(cfg, loopContestants())

	gen, err := loop.RunGeneration(context.Background(), loopChallenges(), mockLoopExec("hello"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gen.Number != 1 {
		t.Errorf("generation number = %d, want 1", gen.Number)
	}
	if len(gen.Winners) != 1 {
		t.Errorf("expected 1 winner, got %d", len(gen.Winners))
	}
	if loop.CurrentGeneration() != 1 {
		t.Errorf("current generation = %d, want 1", loop.CurrentGeneration())
	}
	if loop.Status != StatusPaused {
		t.Errorf("status = %q, want %q", loop.Status, StatusPaused)
	}
}

func TestRunGenerationCompletesByMaxGenerations(t *testing.T) {
	loopIDCounter = 200
	cfg := DefaultConfig(loopIDFunc)
	cfg.SelectionCount = 1
	cfg.MaxGenerations = 2
	cfg.TargetScore = 100 // unreachable
	loop, _ := NewLoop(cfg, loopContestants())

	loop.RunGeneration(context.Background(), loopChallenges(), mockLoopExec("hello"))
	loop.Status = StatusPaused // reset after gen 1
	loop.RunGeneration(context.Background(), loopChallenges(), mockLoopExec("hello"))

	if loop.Status != StatusComplete {
		t.Errorf("status = %q, want %q after max generations", loop.Status, StatusComplete)
	}
}

func TestRunGenerationOnCompletedLoop(t *testing.T) {
	loopIDCounter = 300
	cfg := DefaultConfig(loopIDFunc)
	cfg.SelectionCount = 1
	loop, _ := NewLoop(cfg, loopContestants())
	loop.Status = StatusComplete

	_, err := loop.RunGeneration(context.Background(), loopChallenges(), mockLoopExec("hello"))
	if err == nil {
		t.Error("expected error for completed loop")
	}
}

func TestSetContestants(t *testing.T) {
	loopIDCounter = 400
	cfg := DefaultConfig(loopIDFunc)
	cfg.SelectionCount = 1
	loop, _ := NewLoop(cfg, loopContestants())

	newContestants := loopContestants()[:2]
	loop.SetContestants(newContestants)
	if len(loop.Contestants) != 2 {
		t.Errorf("expected 2 contestants after set, got %d", len(loop.Contestants))
	}
}

func TestIsComplete(t *testing.T) {
	loopIDCounter = 500
	cfg := DefaultConfig(loopIDFunc)
	cfg.SelectionCount = 1
	loop, _ := NewLoop(cfg, loopContestants())

	if loop.IsComplete() {
		t.Error("new loop should not be complete")
	}
	loop.Status = StatusComplete
	if !loop.IsComplete() {
		t.Error("completed loop should be complete")
	}
	loop.Status = StatusFailed
	if !loop.IsComplete() {
		t.Error("failed loop should be complete")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig(loopIDFunc)
	if cfg.MaxGenerations != 10 {
		t.Errorf("MaxGenerations = %d, want 10", cfg.MaxGenerations)
	}
	if cfg.SelectionCount != 2 {
		t.Errorf("SelectionCount = %d, want 2", cfg.SelectionCount)
	}
	w := scoring.DefaultWeights()
	if cfg.Weights != w {
		t.Error("weights should be default")
	}
}
