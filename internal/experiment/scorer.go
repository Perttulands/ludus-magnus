package experiment

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// ScorerInput contains the data scorers need from a completed run.
type ScorerInput struct {
	WorkspaceDiff string   // unified diff output
	BrLog         string   // br stub invocation log
	ToolCalls     []string // tool names used
	EditCount     int      // number of edit/write calls
	Turns         int
	ExitCode      int
	ResponseText  string // final agent response
	WorkDir       string // path to post-run workspace (for running tests)
}

// AutoScorer scores a completed experiment run.
type AutoScorer interface {
	Name() string
	Score(ctx context.Context, input *ScorerInput) (float64, map[string]any, error)
}

// WorkspaceDiffScorer scores based on whether workspace was modified.
type WorkspaceDiffScorer struct{}

func (s *WorkspaceDiffScorer) Name() string { return "workspace_diff" }

func (s *WorkspaceDiffScorer) Score(_ context.Context, input *ScorerInput) (float64, map[string]any, error) {
	diff := strings.TrimSpace(input.WorkspaceDiff)
	if diff == "" {
		return 0.0, map[string]any{"diff_lines": 0, "files_changed": 0}, nil
	}

	lines := strings.Count(diff, "\n") + 1
	files := 0
	for _, line := range strings.Split(diff, "\n") {
		if strings.HasPrefix(line, "+++ ") || strings.HasPrefix(line, "--- ") {
			files++
		}
	}
	files = files / 2 // pairs of ---/+++
	if files < 1 {
		files = 1
	}

	score := 0.5
	if lines > 20 {
		score = 1.0
	}

	return score, map[string]any{"diff_lines": lines, "files_changed": files}, nil
}

// TestPassScorer runs a test command and scores based on pass rate.
type TestPassScorer struct {
	TestCommand string
}

func (s *TestPassScorer) Name() string { return "test_pass" }

func (s *TestPassScorer) Score(ctx context.Context, input *ScorerInput) (float64, map[string]any, error) {
	cmd := s.TestCommand
	if cmd == "" {
		cmd = "go test ./..."
	}

	c := exec.CommandContext(ctx, "sh", "-c", cmd)
	c.Dir = input.WorkDir
	out, err := c.CombinedOutput()
	output := string(out)

	passed, failed, total := parseTestOutput(output)

	details := map[string]any{
		"tests_total":  total,
		"tests_passed": passed,
		"tests_failed": failed,
		"output":       output,
	}

	if total == 0 {
		if err != nil {
			return 0.0, details, nil
		}
		return 1.0, details, nil
	}

	return float64(passed) / float64(total), details, nil
}

func parseTestOutput(output string) (passed, failed, total int) {
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "ok ") {
			passed++
			total++
		} else if strings.HasPrefix(line, "FAIL") && !strings.HasPrefix(line, "FAIL\t") {
			// "FAIL\t" with a package path = failed package
		}
		if strings.HasPrefix(line, "FAIL\t") {
			failed++
			total++
		}
	}
	return
}

// BrStubScorer scores based on br stub usage.
type BrStubScorer struct{}

func (s *BrStubScorer) Name() string { return "br_stub" }

func (s *BrStubScorer) Score(_ context.Context, input *ScorerInput) (float64, map[string]any, error) {
	log := strings.TrimSpace(input.BrLog)
	if log == "" {
		return 0.0, map[string]any{"invocations": 0, "commands": []string{}}, nil
	}

	lines := strings.Split(log, "\n")
	var commands []string
	hasCreate := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		commands = append(commands, line)
		if strings.Contains(line, "create") {
			hasCreate = true
		}
	}

	score := 0.5
	if hasCreate {
		score = 1.0
	}

	return score, map[string]any{"invocations": len(commands), "commands": commands}, nil
}

// SignalDetectionScorer detects regex patterns in response text and workspace diff.
type SignalDetectionScorer struct {
	Signals map[string]string // name → regex pattern
}

func (s *SignalDetectionScorer) Name() string { return "signal_detection" }

func (s *SignalDetectionScorer) Score(_ context.Context, input *ScorerInput) (float64, map[string]any, error) {
	if len(s.Signals) == 0 {
		return 0.0, map[string]any{}, nil
	}

	corpus := input.ResponseText + "\n" + input.WorkspaceDiff
	detected := 0
	details := map[string]any{}

	for name, pattern := range s.Signals {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return 0, nil, fmt.Errorf("invalid regex for signal %q: %w", name, err)
		}
		if re.MatchString(corpus) {
			details[name] = true
			detected++
		} else {
			details[name] = false
		}
	}

	return float64(detected) / float64(len(s.Signals)), details, nil
}
