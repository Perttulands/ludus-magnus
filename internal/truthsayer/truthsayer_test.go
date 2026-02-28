package truthsayer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// --- QualityScore tests ---

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

func TestQualityScoreEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		errors   int
		warnings int
		want     int
	}{
		{"one error drops 2", 1, 0, 8},
		{"five errors floors to 1", 5, 0, 1},
		{"one warning drops 1", 0, 1, 9},
		{"ten warnings floors to 1", 0, 10, 1},
		{"exact zero boundary", 3, 4, 1}, // 10-6-4=0 → floors to 1
		{"negative boundary", 4, 5, 1},   // 10-8-5=-3 → floors to 1
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sr := ScanResult{Errors: tt.errors, Warnings: tt.warnings}
			if got := sr.QualityScore(); got != tt.want {
				t.Errorf("QualityScore(e=%d,w=%d) = %d, want %d", tt.errors, tt.warnings, got, tt.want)
			}
		})
	}
}

// --- ScanWithBinary error path tests ---

func TestScanWithBinaryNotFound(t *testing.T) {
	_, err := ScanWithBinary("nonexistent-binary-chiron-test-xyz", "/tmp")
	if err == nil {
		t.Fatal("expected error for missing binary")
	}
	if got := err.Error(); !contains(got, "truthsayer binary not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestScanWithBinaryBadPath(t *testing.T) {
	// Create a fake "truthsayer" script that exits 2 (tool error)
	dir := t.TempDir()
	fakeBin := filepath.Join(dir, "truthsayer")
	if err := os.WriteFile(fakeBin, []byte("#!/bin/sh\necho 'tool error' >&2\nexit 2\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := ScanWithBinary(fakeBin, "/nonexistent/path")
	if err == nil {
		t.Fatal("expected error for exit code 2")
	}
	if got := err.Error(); !contains(got, "truthsayer tool error (exit 2)") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestScanWithBinaryUnknownExitCode(t *testing.T) {
	dir := t.TempDir()
	fakeBin := filepath.Join(dir, "truthsayer")
	if err := os.WriteFile(fakeBin, []byte("#!/bin/sh\necho 'something bad'\nexit 99\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := ScanWithBinary(fakeBin, "/tmp")
	if err == nil {
		t.Fatal("expected error for exit code 99")
	}
	if got := err.Error(); !contains(got, "truthsayer failed with exit 99") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestScanWithBinaryUnknownExitCodeNoOutput(t *testing.T) {
	dir := t.TempDir()
	fakeBin := filepath.Join(dir, "truthsayer")
	if err := os.WriteFile(fakeBin, []byte("#!/bin/sh\nexit 99\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := ScanWithBinary(fakeBin, "/tmp")
	if err == nil {
		t.Fatal("expected error for exit code 99")
	}
	if got := err.Error(); !contains(got, "truthsayer failed with exit 99") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestScanWithBinaryCleanExit(t *testing.T) {
	dir := t.TempDir()
	fakeBin := filepath.Join(dir, "truthsayer")
	output := ScanOutput{
		Summary:  ScanSummary{Errors: 0, Warnings: 0, Info: 1},
		Findings: []Finding{{Rule: "test-rule", Severity: "info", File: "test.go", Line: 1, Message: "test finding", Category: "test"}},
	}
	jsonOut, _ := json.Marshal(output)
	script := fmt.Sprintf("#!/bin/sh\necho '%s'\nexit 0\n", string(jsonOut))
	if err := os.WriteFile(fakeBin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := ScanWithBinary(fakeBin, "/tmp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
	if len(result.Findings) != 1 {
		t.Errorf("expected 1 finding, got %d", len(result.Findings))
	}
	if result.Findings[0].Rule != "test-rule" {
		t.Errorf("finding rule = %q, want %q", result.Findings[0].Rule, "test-rule")
	}
	if result.Info != 1 {
		t.Errorf("Info = %d, want 1", result.Info)
	}
	if result.ScannedAt == "" {
		t.Error("ScannedAt should not be empty")
	}
	if result.DurationMS < 0 {
		t.Errorf("DurationMS should be non-negative, got %d", result.DurationMS)
	}
}

func TestScanWithBinaryFindingsDetected(t *testing.T) {
	dir := t.TempDir()
	fakeBin := filepath.Join(dir, "truthsayer")
	output := ScanOutput{
		Summary: ScanSummary{Errors: 1, Warnings: 2, Info: 0},
		Findings: []Finding{
			{Rule: "r1", Severity: "error", File: "a.go", Line: 10, Message: "error finding", Category: "cat1"},
			{Rule: "r2", Severity: "warn", File: "b.go", Line: 20, Message: "warn1", Category: "cat2"},
			{Rule: "r3", Severity: "warn", File: "c.go", Line: 30, Message: "warn2", Category: "cat2"},
		},
	}
	jsonOut, _ := json.Marshal(output)
	// exit 1 = findings detected (expected, not an error)
	script := fmt.Sprintf("#!/bin/sh\necho '%s'\nexit 1\n", string(jsonOut))
	if err := os.WriteFile(fakeBin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := ScanWithBinary(fakeBin, "/tmp")
	if err != nil {
		t.Fatalf("exit code 1 should not be an error, got: %v", err)
	}
	if result.ExitCode != 1 {
		t.Errorf("ExitCode = %d, want 1", result.ExitCode)
	}
	if result.Errors != 1 {
		t.Errorf("Errors = %d, want 1", result.Errors)
	}
	if result.Warnings != 2 {
		t.Errorf("Warnings = %d, want 2", result.Warnings)
	}
	if len(result.Findings) != 3 {
		t.Errorf("expected 3 findings, got %d", len(result.Findings))
	}
}

func TestScanWithBinaryBadJSON(t *testing.T) {
	dir := t.TempDir()
	fakeBin := filepath.Join(dir, "truthsayer")
	script := "#!/bin/sh\necho 'not json at all'\nexit 0\n"
	if err := os.WriteFile(fakeBin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := ScanWithBinary(fakeBin, "/tmp")
	if err == nil {
		t.Fatal("expected error for invalid JSON output")
	}
	if got := err.Error(); !contains(got, "decode truthsayer output") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestScanWithBinaryEmptyOutput(t *testing.T) {
	dir := t.TempDir()
	fakeBin := filepath.Join(dir, "truthsayer")
	script := "#!/bin/sh\nexit 0\n"
	if err := os.WriteFile(fakeBin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := ScanWithBinary(fakeBin, "/tmp")
	if err != nil {
		t.Fatalf("empty output with exit 0 should not error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("expected 0 findings for empty output, got %d", len(result.Findings))
	}
}

// --- ScanStringWithBinary tests ---

func TestScanStringWithBinaryNotFound(t *testing.T) {
	_, err := ScanStringWithBinary("nonexistent-binary-chiron-test-xyz", "content", "test.go")
	if err == nil {
		t.Fatal("expected error for missing binary")
	}
}

func TestScanStringWithBinaryCreatesFileAndScans(t *testing.T) {
	dir := t.TempDir()
	fakeBin := filepath.Join(dir, "truthsayer")
	// Script that verifies the temp file exists and outputs clean JSON
	output := ScanOutput{
		Summary:  ScanSummary{Errors: 0, Warnings: 0, Info: 0},
		Findings: []Finding{},
	}
	jsonOut, _ := json.Marshal(output)
	// The script checks $1 is a file path that exists
	script := fmt.Sprintf("#!/bin/sh\nif [ ! -f \"$2\" ]; then echo 'file not found' >&2; exit 2; fi\necho '%s'\nexit 0\n", string(jsonOut))
	if err := os.WriteFile(fakeBin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := ScanStringWithBinary(fakeBin, "package main\n", "test.go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
}

// --- Scan delegates to ScanWithBinary("truthsayer", ...) ---

func TestScanUsesDefaultBinary(t *testing.T) {
	// Scan delegates to ScanWithBinary("truthsayer", ...).
	// If truthsayer is installed, it will either succeed or return a scan error.
	// If not installed, it returns "binary not found".
	// Either way, this tests the delegation path — no panic, no wrong function called.
	dir := t.TempDir()
	cleanFile := filepath.Join(dir, "clean.go")
	os.WriteFile(cleanFile, []byte("package main\n"), 0o644)

	result, err := Scan(dir)
	if err != nil {
		// Acceptable: either binary not found or scan error
		t.Logf("Scan returned error (expected if truthsayer not installed): %v", err)
		return
	}
	// If it succeeds, the result should be well-formed
	if result.ScannedAt == "" {
		t.Error("ScannedAt should not be empty on successful scan")
	}
}

func TestScanStringUsesDefaultBinary(t *testing.T) {
	result, err := ScanString("package main\n", "test.go")
	if err != nil {
		t.Logf("ScanString returned error (expected if truthsayer not installed): %v", err)
		return
	}
	if result.ScannedAt == "" {
		t.Error("ScannedAt should not be empty on successful scan")
	}
}

// --- Finding struct tests ---

func TestFindingJSONRoundTrip(t *testing.T) {
	f := Finding{
		Rule:     "magic-number",
		Severity: "warn",
		File:     "main.go",
		Line:     42,
		Message:  "magic number 42 found",
		Category: "bad-defaults",
	}
	data, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded Finding
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded != f {
		t.Errorf("round-trip mismatch: got %+v, want %+v", decoded, f)
	}
}

func TestScanResultJSONRoundTrip(t *testing.T) {
	sr := ScanResult{
		Findings: []Finding{
			{Rule: "r1", Severity: "error", File: "a.go", Line: 1, Message: "msg1", Category: "c1"},
		},
		Errors:     1,
		Warnings:   0,
		Info:       0,
		ExitCode:   1,
		DurationMS: 42,
		ScannedAt:  "2026-02-28T00:00:00Z",
	}
	data, err := json.Marshal(sr)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded ScanResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Errors != sr.Errors {
		t.Errorf("Errors mismatch: %d vs %d", decoded.Errors, sr.Errors)
	}
	if decoded.ExitCode != sr.ExitCode {
		t.Errorf("ExitCode mismatch: %d vs %d", decoded.ExitCode, sr.ExitCode)
	}
	if len(decoded.Findings) != len(sr.Findings) {
		t.Errorf("Findings count mismatch: %d vs %d", len(decoded.Findings), len(sr.Findings))
	}
}

func TestScanResultFindingsNilSafety(t *testing.T) {
	// A ScanResult with nil Findings should still allow QualityScore
	sr := ScanResult{Errors: 1, Warnings: 1, Findings: nil}
	if got := sr.QualityScore(); got != 7 {
		t.Errorf("QualityScore() = %d, want 7", got)
	}
}

// --- ScanOutput JSON decoding ---

func TestScanOutputDecoding(t *testing.T) {
	raw := `{"summary":{"errors":2,"warnings":1,"info":3},"findings":[{"rule":"r1","severity":"error","file":"f.go","line":10,"message":"m1","category":"c1"}]}`
	var so ScanOutput
	if err := json.Unmarshal([]byte(raw), &so); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if so.Summary.Errors != 2 {
		t.Errorf("Errors = %d, want 2", so.Summary.Errors)
	}
	if so.Summary.Warnings != 1 {
		t.Errorf("Warnings = %d, want 1", so.Summary.Warnings)
	}
	if so.Summary.Info != 3 {
		t.Errorf("Info = %d, want 3", so.Summary.Info)
	}
	if len(so.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(so.Findings))
	}
	if so.Findings[0].Rule != "r1" {
		t.Errorf("finding rule = %q, want %q", so.Findings[0].Rule, "r1")
	}
}

// helper
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsImpl(s, substr))
}
func containsImpl(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
