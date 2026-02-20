package truthsayer

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Finding represents one anti-pattern detected by truthsayer.
type Finding struct {
	Rule     string `json:"rule"`
	Severity string `json:"severity"` // "error", "warn", "info"
	File     string `json:"file"`
	Line     int    `json:"line"`
	Message  string `json:"message"`
	Category string `json:"category"`
}

// ScanResult captures the output of a truthsayer scan.
type ScanResult struct {
	Findings   []Finding `json:"findings"`
	Errors     int       `json:"errors"`
	Warnings   int       `json:"warnings"`
	Info       int       `json:"info"`
	ExitCode   int       `json:"exit_code"`
	DurationMS int       `json:"duration_ms"`
	ScannedAt  string    `json:"scanned_at"`
}

// ScanOutput is the JSON format truthsayer emits with --format json.
type ScanOutput struct {
	Summary  ScanSummary `json:"summary"`
	Findings []Finding   `json:"findings"`
}

// ScanSummary is the summary block from truthsayer JSON output.
type ScanSummary struct {
	Errors   int `json:"errors"`
	Warnings int `json:"warnings"`
	Info     int `json:"info"`
}

// Scan runs truthsayer against a file or directory and returns structured results.
func Scan(path string) (ScanResult, error) {
	return ScanWithBinary("truthsayer", path)
}

// ScanWithBinary runs a specific truthsayer binary against a path.
func ScanWithBinary(binary, path string) (ScanResult, error) {
	binPath, err := exec.LookPath(binary)
	if err != nil {
		return ScanResult{}, fmt.Errorf("truthsayer binary not found: %w", err)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return ScanResult{}, fmt.Errorf("resolve path %q: %w", path, err)
	}

	start := time.Now()
	cmd := exec.Command(binPath, "scan", absPath, "--format", "json")
	output, err := cmd.CombinedOutput()
	duration := int(time.Since(start).Milliseconds())

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return ScanResult{}, fmt.Errorf("run truthsayer: %w", err)
		}
	}

	// Exit code 2 means tool error (not findings)
	if exitCode == 2 {
		return ScanResult{}, fmt.Errorf("truthsayer tool error (exit 2): %s", strings.TrimSpace(string(output)))
	}

	result := ScanResult{
		ExitCode:   exitCode,
		DurationMS: duration,
		ScannedAt:  time.Now().UTC().Format(time.RFC3339),
	}

	if len(output) > 0 {
		var scanOut ScanOutput
		if jsonErr := json.Unmarshal(output, &scanOut); jsonErr == nil {
			result.Findings = scanOut.Findings
			result.Errors = scanOut.Summary.Errors
			result.Warnings = scanOut.Summary.Warnings
			result.Info = scanOut.Summary.Info
		}
	}

	if result.Findings == nil {
		result.Findings = []Finding{}
	}

	return result, nil
}

// ScanString writes content to a temp file and scans it.
func ScanString(content, filename string) (ScanResult, error) {
	return ScanStringWithBinary("truthsayer", content, filename)
}

// ScanStringWithBinary writes content to a temp file and scans with a specific binary.
func ScanStringWithBinary(binary, content, filename string) (ScanResult, error) {
	tmpDir, err := os.MkdirTemp("", "ludus-magnus-scan-*")
	if err != nil {
		return ScanResult{}, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	tmpFile := filepath.Join(tmpDir, filename)
	if err := os.WriteFile(tmpFile, []byte(content), 0o644); err != nil {
		return ScanResult{}, fmt.Errorf("write temp file: %w", err)
	}

	return ScanWithBinary(binary, tmpFile)
}

// QualityScore converts truthsayer findings to a 1-10 quality score.
// Each error deducts 2 points, each warning deducts 1 point from a base of 10.
func (sr ScanResult) QualityScore() int {
	score := 10 - (sr.Errors * 2) - sr.Warnings
	if score < 1 {
		score = 1
	}
	return score
}
