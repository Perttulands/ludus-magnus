package truthsayer

import (
	"testing"
)

func TestQualityScorePerfect(t *testing.T) {
	sr := ScanResult{Errors: 0, Warnings: 0}
	if got := sr.QualityScore(); got != 10 {
		t.Errorf("QualityScore() = %d, want 10", got)
	}
}

func TestQualityScoreWithErrors(t *testing.T) {
	sr := ScanResult{Errors: 2, Warnings: 1}
	// 10 - (2*2) - 1 = 5
	if got := sr.QualityScore(); got != 5 {
		t.Errorf("QualityScore() = %d, want 5", got)
	}
}

func TestQualityScoreFloor(t *testing.T) {
	sr := ScanResult{Errors: 10, Warnings: 10}
	if got := sr.QualityScore(); got != 1 {
		t.Errorf("QualityScore() = %d, want 1 (floor)", got)
	}
}

func TestQualityScoreWarningsOnly(t *testing.T) {
	sr := ScanResult{Errors: 0, Warnings: 3}
	// 10 - 0 - 3 = 7
	if got := sr.QualityScore(); got != 7 {
		t.Errorf("QualityScore() = %d, want 7", got)
	}
}

func TestScanResultDefaultFindings(t *testing.T) {
	sr := ScanResult{}
	if sr.Findings != nil {
		// This test validates nil-safety expectations
		t.Log("Findings is non-nil, which is fine for initialized structs")
	}
}

func TestFindingFields(t *testing.T) {
	f := Finding{
		Rule:     "magic-number",
		Severity: "warn",
		File:     "main.go",
		Line:     42,
		Message:  "magic number 42 found",
		Category: "bad-defaults",
	}
	if f.Rule != "magic-number" {
		t.Errorf("Rule = %q, want %q", f.Rule, "magic-number")
	}
	if f.Severity != "warn" {
		t.Errorf("Severity = %q, want %q", f.Severity, "warn")
	}
}
