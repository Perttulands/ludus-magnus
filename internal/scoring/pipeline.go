package scoring

import (
	"github.com/Perttulands/ludus-magnus/internal/harness"
	"github.com/Perttulands/ludus-magnus/internal/truthsayer"
)

// Weights controls how different scoring components contribute to the final score.
type Weights struct {
	Harness    float64 `json:"harness"`    // weight for test harness results
	Truthsayer float64 `json:"truthsayer"` // weight for truthsayer quality
	Manual     float64 `json:"manual"`     // weight for manual evaluation
	Efficiency float64 `json:"efficiency"` // weight for execution efficiency
}

// DefaultWeights returns balanced scoring weights.
func DefaultWeights() Weights {
	return Weights{
		Harness:    0.35,
		Truthsayer: 0.25,
		Manual:     0.30,
		Efficiency: 0.10,
	}
}

// Input provides the raw scores from each evaluation component.
type Input struct {
	HarnessResult    *harness.SuiteResult    `json:"harness_result,omitempty"`
	TruthsayerResult *truthsayer.ScanResult  `json:"truthsayer_result,omitempty"`
	ManualScore      *int                    `json:"manual_score,omitempty"` // 1-10
	DurationMS       int                     `json:"duration_ms"`
	MaxDurationMS    int                     `json:"max_duration_ms"` // expected max for efficiency calc
}

// ComponentScore captures one component's contribution.
type ComponentScore struct {
	Name       string  `json:"name"`
	RawScore   int     `json:"raw_score"`   // 1-10
	Weight     float64 `json:"weight"`
	Weighted   float64 `json:"weighted"`    // raw * weight
	Available  bool    `json:"available"`   // whether this component had data
}

// Result is the composite scoring output.
type Result struct {
	Components  []ComponentScore `json:"components"`
	FinalScore  float64          `json:"final_score"`  // weighted average
	Normalized  int              `json:"normalized"`    // 1-10 integer
	TotalWeight float64          `json:"total_weight"`  // sum of available weights
}

// Score computes a composite score from all available evaluation components.
func Score(input Input, weights Weights) Result {
	components := make([]ComponentScore, 0, 4)
	var totalWeighted, totalWeight float64

	// Harness component
	if input.HarnessResult != nil {
		raw := input.HarnessResult.NormalizedScore()
		w := weights.Harness
		components = append(components, ComponentScore{
			Name: "harness", RawScore: raw, Weight: w,
			Weighted: float64(raw) * w, Available: true,
		})
		totalWeighted += float64(raw) * w
		totalWeight += w
	} else {
		components = append(components, ComponentScore{
			Name: "harness", Available: false, Weight: weights.Harness,
		})
	}

	// Truthsayer component
	if input.TruthsayerResult != nil {
		raw := input.TruthsayerResult.QualityScore()
		w := weights.Truthsayer
		components = append(components, ComponentScore{
			Name: "truthsayer", RawScore: raw, Weight: w,
			Weighted: float64(raw) * w, Available: true,
		})
		totalWeighted += float64(raw) * w
		totalWeight += w
	} else {
		components = append(components, ComponentScore{
			Name: "truthsayer", Available: false, Weight: weights.Truthsayer,
		})
	}

	// Manual component
	if input.ManualScore != nil {
		raw := *input.ManualScore
		if raw < 1 {
			raw = 1
		}
		if raw > 10 {
			raw = 10
		}
		w := weights.Manual
		components = append(components, ComponentScore{
			Name: "manual", RawScore: raw, Weight: w,
			Weighted: float64(raw) * w, Available: true,
		})
		totalWeighted += float64(raw) * w
		totalWeight += w
	} else {
		components = append(components, ComponentScore{
			Name: "manual", Available: false, Weight: weights.Manual,
		})
	}

	// Efficiency component
	if input.MaxDurationMS > 0 && input.DurationMS > 0 {
		raw := efficiencyScore(input.DurationMS, input.MaxDurationMS)
		w := weights.Efficiency
		components = append(components, ComponentScore{
			Name: "efficiency", RawScore: raw, Weight: w,
			Weighted: float64(raw) * w, Available: true,
		})
		totalWeighted += float64(raw) * w
		totalWeight += w
	} else {
		components = append(components, ComponentScore{
			Name: "efficiency", Available: false, Weight: weights.Efficiency,
		})
	}

	finalScore := 0.0
	if totalWeight > 0 {
		finalScore = totalWeighted / totalWeight
	}

	normalized := int(finalScore + 0.5) // round
	if normalized < 1 {
		normalized = 1
	}
	if normalized > 10 {
		normalized = 10
	}

	return Result{
		Components:  components,
		FinalScore:  finalScore,
		Normalized:  normalized,
		TotalWeight: totalWeight,
	}
}

// efficiencyScore maps duration/maxDuration ratio to 1-10.
// At or under budget = 10, double budget = 1.
func efficiencyScore(durationMS, maxDurationMS int) int {
	if maxDurationMS <= 0 {
		return 5
	}
	ratio := float64(durationMS) / float64(maxDurationMS)
	if ratio <= 1.0 {
		return 10
	}
	// Linear decay from 10 to 1 as ratio goes from 1.0 to 2.0
	score := int(10 - (ratio-1.0)*9)
	if score < 1 {
		score = 1
	}
	return score
}
