package scoring

import (
	"testing"

	"github.com/Perttulands/ludus-magnus/internal/harness"
	"github.com/Perttulands/ludus-magnus/internal/truthsayer"
)

func intPtr(v int) *int { return &v }

func TestScoreAllComponents(t *testing.T) {
	input := Input{
		HarnessResult: &harness.SuiteResult{
			TotalScore: 10.0,
			MaxScore:   10.0,
		},
		TruthsayerResult: &truthsayer.ScanResult{
			Errors: 0, Warnings: 0,
		},
		ManualScore:   intPtr(8),
		DurationMS:    5000,
		MaxDurationMS: 10000,
	}

	result := Score(input, DefaultWeights())

	if result.Normalized < 1 || result.Normalized > 10 {
		t.Errorf("normalized score %d out of range", result.Normalized)
	}

	availableCount := 0
	for _, c := range result.Components {
		if c.Available {
			availableCount++
		}
	}
	if availableCount != 4 {
		t.Errorf("expected 4 available components, got %d", availableCount)
	}
}

func TestScoreManualOnly(t *testing.T) {
	input := Input{
		ManualScore: intPtr(7),
	}

	result := Score(input, DefaultWeights())

	if result.Normalized != 7 {
		t.Errorf("expected normalized 7 for manual-only score of 7, got %d", result.Normalized)
	}
	if result.TotalWeight != DefaultWeights().Manual {
		t.Errorf("expected total weight %f, got %f", DefaultWeights().Manual, result.TotalWeight)
	}
}

func TestScoreNoComponents(t *testing.T) {
	input := Input{}
	result := Score(input, DefaultWeights())

	if result.Normalized != 1 {
		t.Errorf("expected normalized 1 for no data, got %d", result.Normalized)
	}
	if result.TotalWeight != 0 {
		t.Errorf("expected total weight 0, got %f", result.TotalWeight)
	}
}

func TestScoreHarnessAndTruthsayer(t *testing.T) {
	input := Input{
		HarnessResult: &harness.SuiteResult{
			TotalScore: 5.0,
			MaxScore:   10.0,
		},
		TruthsayerResult: &truthsayer.ScanResult{
			Errors: 1, Warnings: 2,
		},
	}

	result := Score(input, DefaultWeights())

	// Harness: NormalizedScore of 5/10 = 5
	// Truthsayer: 10 - 2 - 2 = 6
	if result.Normalized < 1 || result.Normalized > 10 {
		t.Errorf("normalized score %d out of range", result.Normalized)
	}
}

func TestEfficiencyScoreUnderBudget(t *testing.T) {
	score := efficiencyScore(5000, 10000)
	if score != 10 {
		t.Errorf("expected 10 for under budget, got %d", score)
	}
}

func TestEfficiencyScoreAtBudget(t *testing.T) {
	score := efficiencyScore(10000, 10000)
	if score != 10 {
		t.Errorf("expected 10 at budget, got %d", score)
	}
}

func TestEfficiencyScoreOverBudget(t *testing.T) {
	score := efficiencyScore(15000, 10000)
	// ratio = 1.5, score = 10 - 0.5*9 = 5
	if score != 5 {
		t.Errorf("expected 5 for 1.5x budget, got %d", score)
	}
}

func TestEfficiencyScoreDoubleBudget(t *testing.T) {
	score := efficiencyScore(20000, 10000)
	if score != 1 {
		t.Errorf("expected 1 for 2x budget, got %d", score)
	}
}

func TestDefaultWeightsSum(t *testing.T) {
	w := DefaultWeights()
	sum := w.Harness + w.Truthsayer + w.Manual + w.Efficiency
	if sum < 0.99 || sum > 1.01 {
		t.Errorf("default weights sum to %f, expected ~1.0", sum)
	}
}

func TestScoreManualClamp(t *testing.T) {
	input := Input{ManualScore: intPtr(15)}
	result := Score(input, DefaultWeights())
	if result.Normalized != 10 {
		t.Errorf("expected clamped normalized 10, got %d", result.Normalized)
	}

	input2 := Input{ManualScore: intPtr(-5)}
	result2 := Score(input2, DefaultWeights())
	if result2.Normalized != 1 {
		t.Errorf("expected clamped normalized 1, got %d", result2.Normalized)
	}
}
