package harness

import (
	"testing"
)

func TestRunSuiteAllPass(t *testing.T) {
	suite := TestSuite{
		ID:   "suite_1",
		Name: "basic checks",
		TestCases: []TestCase{
			{ID: "tc_1", Name: "has greeting", Type: "contains", Expected: "hello", Weight: 1.0},
			{ID: "tc_2", Name: "has name", Type: "contains", Expected: "world", Weight: 1.0},
		},
	}

	result := RunSuite(suite, "hello world")

	if result.Passed != 2 {
		t.Errorf("expected 2 passed, got %d", result.Passed)
	}
	if result.Failed != 0 {
		t.Errorf("expected 0 failed, got %d", result.Failed)
	}
	if result.PassRate != 1.0 {
		t.Errorf("expected pass rate 1.0, got %f", result.PassRate)
	}
	if result.TotalScore != 2.0 {
		t.Errorf("expected total score 2.0, got %f", result.TotalScore)
	}
}

func TestRunSuitePartialPass(t *testing.T) {
	suite := TestSuite{
		ID:   "suite_2",
		Name: "partial",
		TestCases: []TestCase{
			{ID: "tc_1", Name: "has hello", Type: "contains", Expected: "hello", Weight: 1.0},
			{ID: "tc_2", Name: "has missing", Type: "contains", Expected: "missing", Weight: 1.0},
		},
	}

	result := RunSuite(suite, "hello world")

	if result.Passed != 1 {
		t.Errorf("expected 1 passed, got %d", result.Passed)
	}
	if result.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", result.Failed)
	}
	if result.PassRate != 0.5 {
		t.Errorf("expected pass rate 0.5, got %f", result.PassRate)
	}
}

func TestRunSuiteNotContains(t *testing.T) {
	suite := TestSuite{
		ID:   "suite_3",
		Name: "not contains",
		TestCases: []TestCase{
			{ID: "tc_1", Name: "no error", Type: "not_contains", Expected: "error", Weight: 1.0},
		},
	}

	result := RunSuite(suite, "everything is fine")
	if result.Passed != 1 {
		t.Errorf("expected 1 passed, got %d", result.Passed)
	}

	result2 := RunSuite(suite, "there was an error")
	if result2.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", result2.Failed)
	}
}

func TestRunSuiteRegex(t *testing.T) {
	suite := TestSuite{
		ID:   "suite_4",
		Name: "regex",
		TestCases: []TestCase{
			{ID: "tc_1", Name: "has number", Type: "regex", Expected: `\d+`, Weight: 1.0},
		},
	}

	result := RunSuite(suite, "count is 42")
	if result.Passed != 1 {
		t.Errorf("expected 1 passed, got %d", result.Passed)
	}

	result2 := RunSuite(suite, "no numbers here")
	if result2.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", result2.Failed)
	}
}

func TestRunSuiteEquals(t *testing.T) {
	suite := TestSuite{
		ID:   "suite_5",
		Name: "equals",
		TestCases: []TestCase{
			{ID: "tc_1", Name: "exact match", Type: "equals", Expected: "exact output", Weight: 1.0},
		},
	}

	result := RunSuite(suite, "exact output")
	if result.Passed != 1 {
		t.Errorf("expected 1 passed, got %d", result.Passed)
	}

	result2 := RunSuite(suite, "different output")
	if result2.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", result2.Failed)
	}
}

func TestRunSuiteWeightedScoring(t *testing.T) {
	suite := TestSuite{
		ID:   "suite_6",
		Name: "weighted",
		TestCases: []TestCase{
			{ID: "tc_1", Name: "critical", Type: "contains", Expected: "hello", Weight: 3.0},
			{ID: "tc_2", Name: "minor", Type: "contains", Expected: "missing", Weight: 1.0},
		},
	}

	result := RunSuite(suite, "hello world")
	if result.TotalScore != 3.0 {
		t.Errorf("expected total score 3.0, got %f", result.TotalScore)
	}
	if result.MaxScore != 4.0 {
		t.Errorf("expected max score 4.0, got %f", result.MaxScore)
	}
}

func TestRunSuiteDefaultWeight(t *testing.T) {
	suite := TestSuite{
		ID:   "suite_7",
		Name: "default weight",
		TestCases: []TestCase{
			{ID: "tc_1", Name: "no weight set", Type: "contains", Expected: "hello"},
		},
	}

	result := RunSuite(suite, "hello")
	if result.TotalScore != 1.0 {
		t.Errorf("expected total score 1.0 (default weight), got %f", result.TotalScore)
	}
}

func TestRunSuiteEmpty(t *testing.T) {
	suite := TestSuite{ID: "suite_empty", Name: "empty"}
	result := RunSuite(suite, "anything")
	if result.PassRate != 0.0 {
		t.Errorf("expected pass rate 0.0 for empty suite, got %f", result.PassRate)
	}
}

func TestNormalizedScore(t *testing.T) {
	tests := []struct {
		total, max float64
		want       int
	}{
		{0, 10, 1},
		{10, 10, 10},
		{5, 10, 5},
		{0, 0, 1},
	}
	for _, tt := range tests {
		sr := SuiteResult{TotalScore: tt.total, MaxScore: tt.max}
		got := sr.NormalizedScore()
		if got != tt.want {
			t.Errorf("NormalizedScore(%f/%f) = %d, want %d", tt.total, tt.max, got, tt.want)
		}
	}
}

func TestUnknownTestType(t *testing.T) {
	suite := TestSuite{
		ID:   "suite_unknown",
		Name: "unknown type",
		TestCases: []TestCase{
			{ID: "tc_1", Name: "bad type", Type: "invalid_type", Expected: "x", Weight: 1.0},
		},
	}

	result := RunSuite(suite, "anything")
	if result.Failed != 1 {
		t.Errorf("expected 1 failed for unknown type, got %d", result.Failed)
	}
}
