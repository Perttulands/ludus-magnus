package challenge

import (
	"testing"

	"github.com/Perttulands/ludus-magnus/internal/harness"
)

func TestValidateValid(t *testing.T) {
	c := Challenge{
		ID:          "ch_001",
		Name:        "simple feature",
		Type:        TypeFeature,
		Description: "Implement a greeting function",
		Input:       "Create a function that returns hello world",
	}
	if err := c.Validate(); err != nil {
		t.Errorf("expected valid, got %v", err)
	}
}

func TestValidateMissingID(t *testing.T) {
	c := Challenge{Name: "x", Type: TypeFeature, Description: "x", Input: "x"}
	if err := c.Validate(); err == nil {
		t.Error("expected error for missing ID")
	}
}

func TestValidateMissingName(t *testing.T) {
	c := Challenge{ID: "x", Type: TypeFeature, Description: "x", Input: "x"}
	if err := c.Validate(); err == nil {
		t.Error("expected error for missing name")
	}
}

func TestValidateInvalidType(t *testing.T) {
	c := Challenge{ID: "x", Name: "x", Type: "invalid", Description: "x", Input: "x"}
	if err := c.Validate(); err == nil {
		t.Error("expected error for invalid type")
	}
}

func TestValidateMissingDescription(t *testing.T) {
	c := Challenge{ID: "x", Name: "x", Type: TypeBugfix, Input: "x"}
	if err := c.Validate(); err == nil {
		t.Error("expected error for missing description")
	}
}

func TestValidateMissingInput(t *testing.T) {
	c := Challenge{ID: "x", Name: "x", Type: TypeRefactor, Description: "x"}
	if err := c.Validate(); err == nil {
		t.Error("expected error for missing input")
	}
}

func TestValidateAllTypes(t *testing.T) {
	for _, tp := range ValidTypes {
		c := Challenge{ID: "x", Name: "x", Type: tp, Description: "x", Input: "x"}
		if err := c.Validate(); err != nil {
			t.Errorf("type %q should be valid, got %v", tp, err)
		}
	}
}

func TestTotalWeight(t *testing.T) {
	c := Challenge{
		TestSuite: harness.TestSuite{
			TestCases: []harness.TestCase{
				{Weight: 2.0},
				{Weight: 3.0},
				{Weight: 0}, // should default to 1.0
			},
		},
	}
	got := c.TotalWeight()
	if got != 6.0 {
		t.Errorf("TotalWeight() = %f, want 6.0", got)
	}
}

func TestTotalWeightEmpty(t *testing.T) {
	c := Challenge{}
	if got := c.TotalWeight(); got != 0 {
		t.Errorf("TotalWeight() = %f, want 0", got)
	}
}
