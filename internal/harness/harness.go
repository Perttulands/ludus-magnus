package harness

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// TestCase defines one assertion against agent output.
type TestCase struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"` // "contains", "regex", "not_contains", "equals"
	Expected    string `json:"expected"`
	Weight      float64 `json:"weight"` // 0.0-1.0, default 1.0
	Description string `json:"description,omitempty"`
}

// TestSuite groups related test cases for evaluation.
type TestSuite struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	TestCases []TestCase `json:"test_cases"`
}

// TestResult captures the outcome of running one test case.
type TestResult struct {
	TestCaseID string  `json:"test_case_id"`
	TestName   string  `json:"test_name"`
	Passed     bool    `json:"passed"`
	Score      float64 `json:"score"` // weighted score: weight * (1.0 if passed, 0.0 otherwise)
	Detail     string  `json:"detail,omitempty"`
}

// SuiteResult captures the aggregate outcome of a test suite run.
type SuiteResult struct {
	SuiteID     string       `json:"suite_id"`
	SuiteName   string       `json:"suite_name"`
	Results     []TestResult `json:"results"`
	TotalScore  float64      `json:"total_score"`  // sum of weighted scores
	MaxScore    float64      `json:"max_score"`     // sum of all weights
	PassRate    float64      `json:"pass_rate"`     // 0.0-1.0
	Passed      int          `json:"passed"`
	Failed      int          `json:"failed"`
	DurationMS  int          `json:"duration_ms"`
	RunAt       string       `json:"run_at"`
}

// RunSuite executes all test cases in a suite against the given output.
func RunSuite(suite TestSuite, output string) SuiteResult {
	start := time.Now()
	results := make([]TestResult, 0, len(suite.TestCases))

	var totalScore, maxScore float64
	var passed, failed int

	for _, tc := range suite.TestCases {
		result := runTestCase(tc, output)
		results = append(results, result)
		totalScore += result.Score
		weight := tc.Weight
		if weight <= 0 {
			weight = 1.0
		}
		maxScore += weight
		if result.Passed {
			passed++
		} else {
			failed++
		}
	}

	passRate := 0.0
	if len(suite.TestCases) > 0 {
		passRate = float64(passed) / float64(len(suite.TestCases))
	}

	return SuiteResult{
		SuiteID:    suite.ID,
		SuiteName:  suite.Name,
		Results:    results,
		TotalScore: totalScore,
		MaxScore:   maxScore,
		PassRate:   passRate,
		Passed:     passed,
		Failed:     failed,
		DurationMS: int(time.Since(start).Milliseconds()),
		RunAt:      time.Now().UTC().Format(time.RFC3339),
	}
}

func runTestCase(tc TestCase, output string) TestResult {
	weight := tc.Weight
	if weight <= 0 {
		weight = 1.0
	}

	pass, detail := evaluate(tc.Type, tc.Expected, output)

	score := 0.0
	if pass {
		score = weight
	}

	return TestResult{
		TestCaseID: tc.ID,
		TestName:   tc.Name,
		Passed:     pass,
		Score:      score,
		Detail:     detail,
	}
}

func evaluate(checkType, expected, output string) (bool, string) {
	switch strings.ToLower(strings.TrimSpace(checkType)) {
	case "contains":
		if strings.Contains(output, expected) {
			return true, fmt.Sprintf("output contains %q", expected)
		}
		return false, fmt.Sprintf("output does not contain %q", expected)

	case "not_contains":
		if !strings.Contains(output, expected) {
			return true, fmt.Sprintf("output does not contain %q (as expected)", expected)
		}
		return false, fmt.Sprintf("output contains %q (unexpected)", expected)

	case "regex":
		re, err := regexp.Compile(expected)
		if err != nil {
			return false, fmt.Sprintf("invalid regex %q: %v", expected, err)
		}
		if re.MatchString(output) {
			return true, fmt.Sprintf("output matches regex %q", expected)
		}
		return false, fmt.Sprintf("output does not match regex %q", expected)

	case "equals":
		if strings.TrimSpace(output) == strings.TrimSpace(expected) {
			return true, "output equals expected"
		}
		return false, "output does not equal expected"

	default:
		return false, fmt.Sprintf("unknown test type %q", checkType)
	}
}

// NormalizedScore returns the suite score as 1-10 scale for integration with evaluation.
func (sr SuiteResult) NormalizedScore() int {
	if sr.MaxScore <= 0 {
		return 1
	}
	ratio := sr.TotalScore / sr.MaxScore
	score := int(ratio*9) + 1 // maps 0.0->1, 1.0->10
	if score < 1 {
		score = 1
	}
	if score > 10 {
		score = 10
	}
	return score
}
