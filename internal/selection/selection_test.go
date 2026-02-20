package selection

import (
	"math/rand"
	"testing"

	"github.com/Perttulands/ludus-magnus/internal/tournament"
)

func testStandings() []tournament.Standing {
	return []tournament.Standing{
		{ContestantID: "c_1", AvgScore: 8.0, Rank: 1, BoutsWon: 3},
		{ContestantID: "c_2", AvgScore: 6.0, Rank: 2, BoutsWon: 2},
		{ContestantID: "c_3", AvgScore: 4.0, Rank: 3, BoutsWon: 1},
		{ContestantID: "c_4", AvgScore: 2.0, Rank: 4, BoutsWon: 0},
	}
}

func TestTruncationSelectTop2(t *testing.T) {
	sel := TruncationSelector{}
	selected := sel.Select(testStandings(), 2)

	if len(selected) != 2 {
		t.Fatalf("expected 2 selected, got %d", len(selected))
	}
	if selected[0].ContestantID != "c_1" {
		t.Errorf("first = %q, want %q", selected[0].ContestantID, "c_1")
	}
	if selected[1].ContestantID != "c_2" {
		t.Errorf("second = %q, want %q", selected[1].ContestantID, "c_2")
	}
}

func TestTruncationSelectMoreThanAvailable(t *testing.T) {
	sel := TruncationSelector{}
	selected := sel.Select(testStandings(), 10)
	if len(selected) != 4 {
		t.Errorf("expected 4 (all), got %d", len(selected))
	}
}

func TestTruncationSelectZero(t *testing.T) {
	sel := TruncationSelector{}
	selected := sel.Select(testStandings(), 0)
	if selected != nil {
		t.Errorf("expected nil, got %v", selected)
	}
}

func TestTournamentSelect(t *testing.T) {
	sel := TournamentSelector{Rng: rand.New(rand.NewSource(42))}
	selected := sel.Select(testStandings(), 2)

	if len(selected) != 2 {
		t.Fatalf("expected 2 selected, got %d", len(selected))
	}

	// Tournament selection should prefer higher scores
	ids := map[string]bool{}
	for _, s := range selected {
		ids[s.ContestantID] = true
	}
	if len(ids) != 2 {
		t.Error("selected contestants should be unique")
	}
}

func TestElitistSelect(t *testing.T) {
	sel := ElitistSelector{}
	selected := sel.Select(testStandings(), 2)

	if len(selected) != 2 {
		t.Fatalf("expected 2, got %d", len(selected))
	}
	if selected[0].ContestantID != "c_1" {
		t.Errorf("elite should be first: got %q", selected[0].ContestantID)
	}
}

func TestNewSelectorValid(t *testing.T) {
	for _, name := range []string{StrategyTruncation, StrategyTournament, StrategyElitist} {
		sel, err := NewSelector(name)
		if err != nil {
			t.Errorf("NewSelector(%q) error: %v", name, err)
		}
		if sel == nil {
			t.Errorf("NewSelector(%q) returned nil", name)
		}
	}
}

func TestNewSelectorInvalid(t *testing.T) {
	_, err := NewSelector("invalid")
	if err == nil {
		t.Error("expected error for invalid strategy")
	}
}

func TestPartition(t *testing.T) {
	standings := testStandings()
	winners := []tournament.Standing{standings[0], standings[1]}

	selected, eliminated := Partition(standings, winners)

	if len(selected) != 2 {
		t.Errorf("expected 2 selected, got %d", len(selected))
	}
	if len(eliminated) != 2 {
		t.Errorf("expected 2 eliminated, got %d", len(eliminated))
	}
}

func TestPartitionEmpty(t *testing.T) {
	selected, eliminated := Partition(nil, nil)
	if selected != nil || eliminated != nil {
		t.Error("expected nil for empty partition")
	}
}
