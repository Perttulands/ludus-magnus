package selection

import (
	"fmt"
	"math/rand"
	"sort"

	"github.com/Perttulands/ludus-magnus/internal/tournament"
)

// Strategy names for selection methods.
const (
	StrategyTruncation = "truncation"
	StrategyTournament = "tournament"
	StrategyElitist    = "elitist"
)

// Selector picks winners from tournament standings.
type Selector interface {
	Select(standings []tournament.Standing, n int) []tournament.Standing
}

// TruncationSelector keeps the top N by rank.
type TruncationSelector struct{}

func (TruncationSelector) Select(standings []tournament.Standing, n int) []tournament.Standing {
	if n <= 0 || len(standings) == 0 {
		return nil
	}
	sorted := make([]tournament.Standing, len(standings))
	copy(sorted, standings)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Rank < sorted[j].Rank })
	if n > len(sorted) {
		n = len(sorted)
	}
	return sorted[:n]
}

// TournamentSelector runs pairwise comparisons to select winners.
type TournamentSelector struct {
	Rng *rand.Rand
}

func (ts TournamentSelector) Select(standings []tournament.Standing, n int) []tournament.Standing {
	if n <= 0 || len(standings) == 0 {
		return nil
	}

	rng := ts.Rng
	if rng == nil {
		rng = rand.New(rand.NewSource(42))
	}

	selected := make([]tournament.Standing, 0, n)
	used := map[string]bool{}

	for len(selected) < n && len(used) < len(standings) {
		// Pick two random contestants, keep the better one
		a := standings[rng.Intn(len(standings))]
		b := standings[rng.Intn(len(standings))]

		winner := a
		if b.AvgScore > a.AvgScore {
			winner = b
		}

		if !used[winner.ContestantID] {
			used[winner.ContestantID] = true
			selected = append(selected, winner)
		}
	}

	return selected
}

// ElitistSelector always keeps the single best, fills remaining by truncation.
type ElitistSelector struct{}

func (ElitistSelector) Select(standings []tournament.Standing, n int) []tournament.Standing {
	if n <= 0 || len(standings) == 0 {
		return nil
	}
	return TruncationSelector{}.Select(standings, n)
}

// NewSelector creates a selector by strategy name.
func NewSelector(strategy string) (Selector, error) {
	switch strategy {
	case StrategyTruncation:
		return TruncationSelector{}, nil
	case StrategyTournament:
		return TournamentSelector{}, nil
	case StrategyElitist:
		return ElitistSelector{}, nil
	default:
		return nil, fmt.Errorf("unknown selection strategy %q; choose from: %s, %s, %s",
			strategy, StrategyTruncation, StrategyTournament, StrategyElitist)
	}
}

// Partition splits standings into winners and losers based on selection.
func Partition(standings []tournament.Standing, winners []tournament.Standing) (selected, eliminated []tournament.Standing) {
	winnerIDs := map[string]bool{}
	for _, w := range winners {
		winnerIDs[w.ContestantID] = true
	}

	for _, s := range standings {
		if winnerIDs[s.ContestantID] {
			selected = append(selected, s)
		} else {
			eliminated = append(eliminated, s)
		}
	}
	return selected, eliminated
}
