package challenge

import (
	"fmt"
	"strings"

	"github.com/Perttulands/ludus-magnus/internal/harness"
)

// Challenge types matching the PRD.
const (
	TypeFeature  = "feature"
	TypeBugfix   = "bugfix"
	TypeRefactor = "refactor"
	TypeReview   = "review"
)

// ValidTypes lists all recognized challenge types.
var ValidTypes = []string{TypeFeature, TypeBugfix, TypeRefactor, TypeReview}

// Difficulty levels for challenges.
const (
	DifficultyEasy   = "easy"
	DifficultyMedium = "medium"
	DifficultyHard   = "hard"
)

// Challenge defines a synthetic evaluation task for agent training.
type Challenge struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Type        string            `json:"type"`       // feature, bugfix, refactor, review
	Difficulty  string            `json:"difficulty"`  // easy, medium, hard
	Description string            `json:"description"` // what the agent must do
	Input       string            `json:"input"`       // the input/prompt given to the agent
	Context     string            `json:"context,omitempty"` // optional code or context
	TestSuite   harness.TestSuite `json:"test_suite"`  // how to verify the output
	Tags        []string          `json:"tags,omitempty"`
	CreatedAt   string            `json:"created_at"`
	MaxDurationMS int             `json:"max_duration_ms,omitempty"` // expected time budget
}

// ChallengeSet groups challenges for a tournament.
type ChallengeSet struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Challenges []Challenge `json:"challenges"`
	CreatedAt  string      `json:"created_at"`
}

// Validate checks that a challenge has required fields and valid types.
func (c Challenge) Validate() error {
	if strings.TrimSpace(c.ID) == "" {
		return fmt.Errorf("challenge id is required")
	}
	if strings.TrimSpace(c.Name) == "" {
		return fmt.Errorf("challenge name is required")
	}
	if !isValidType(c.Type) {
		return fmt.Errorf("invalid challenge type %q; must be one of: %s", c.Type, strings.Join(ValidTypes, ", "))
	}
	if strings.TrimSpace(c.Description) == "" {
		return fmt.Errorf("challenge description is required")
	}
	if strings.TrimSpace(c.Input) == "" {
		return fmt.Errorf("challenge input is required")
	}
	return nil
}

func isValidType(t string) bool {
	for _, valid := range ValidTypes {
		if t == valid {
			return true
		}
	}
	return false
}

// TotalWeight returns the sum of all test case weights in the challenge.
func (c Challenge) TotalWeight() float64 {
	var total float64
	for _, tc := range c.TestSuite.TestCases {
		w := tc.Weight
		if w <= 0 {
			w = 1.0
		}
		total += w
	}
	return total
}
