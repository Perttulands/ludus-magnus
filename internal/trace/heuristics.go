package trace

import (
	"strings"
)

// ApplyHeuristics analyzes a RunTrace and populates FailureTags and Extensions.
func ApplyHeuristics(t *RunTrace) {
	detectReadOnlyNoEdit(t)
	detectPathResolutionLoop(t)
	detectEditWithoutTest(t)
	detectTestFixLoop(t)
	detectToolErrorLoop(t)
	detectReportOmittedAfterEdit(t)
	detectExtensionEvidence(t)
	updateTestAfterEdit(t)
}

// detectReadOnlyNoEdit flags runs where the agent only reads but never edits.
func detectReadOnlyNoEdit(t *RunTrace) {
	if t.Metrics.EditCount == 0 && t.Metrics.ReadCount > 0 && t.Metrics.TotalTurns > 2 {
		t.FailureTags = append(t.FailureTags, FailureTag{
			Tag:      "read_only_no_edit",
			Evidence: "Agent read files but never produced an edit or write",
		})
	}
}

// detectPathResolutionLoop detects repeated read attempts suggesting path confusion.
func detectPathResolutionLoop(t *RunTrace) {
	// Look for 3+ consecutive turns with only read/glob/ls tools and no edits.
	streak := 0
	for _, turn := range t.Turns {
		allReads := len(turn.ToolCalls) > 0
		for _, tc := range turn.ToolCalls {
			name := strings.ToLower(tc.Name)
			if name != "read" && name != "glob" && name != "grep" && name != "ls" && name != "bash" {
				allReads = false
				break
			}
		}
		if allReads && len(turn.ToolCalls) > 0 {
			streak++
		} else {
			streak = 0
		}
		if streak >= 4 {
			t.FailureTags = append(t.FailureTags, FailureTag{
				Tag:      "path_resolution_loop",
				Evidence: "4+ consecutive turns of only read/search tools — possible path confusion",
				TurnIdx:  turn.Index,
			})
			return
		}
	}
}

// detectEditWithoutTest flags runs where edits happen but no test is ever run.
func detectEditWithoutTest(t *RunTrace) {
	if t.Metrics.EditCount > 0 && !hasTestExecution(t) {
		t.FailureTags = append(t.FailureTags, FailureTag{
			Tag:      "edit_without_test",
			Evidence: "Code was edited but no test execution was detected",
		})
	}
}

// detectTestFixLoop detects repeated test→edit→test cycles suggesting a fix loop.
func detectTestFixLoop(t *RunTrace) {
	// Look for alternating test-then-edit patterns.
	cycles := 0
	lastWasTest := false
	for _, turn := range t.Turns {
		hasEdit := false
		hasTest := false
		for _, tc := range turn.ToolCalls {
			name := strings.ToLower(tc.Name)
			if name == "edit" || name == "write" {
				hasEdit = true
			}
			if name == "bash" {
				hasTest = true // Broad heuristic — bash may be test
			}
		}
		if hasTest && lastWasTest && hasEdit {
			cycles++
		}
		lastWasTest = hasTest
	}
	if cycles >= 3 {
		t.FailureTags = append(t.FailureTags, FailureTag{
			Tag:      "test_fix_loop",
			Evidence: "3+ test→edit→test cycles detected — possible fix loop",
		})
	}
}

// detectToolErrorLoop detects repeated tool errors.
func detectToolErrorLoop(t *RunTrace) {
	errorStreak := 0
	for _, turn := range t.Turns {
		hasError := false
		for _, tc := range turn.ToolCalls {
			if tc.IsError {
				hasError = true
				break
			}
		}
		if hasError {
			errorStreak++
		} else {
			errorStreak = 0
		}
		if errorStreak >= 3 {
			t.FailureTags = append(t.FailureTags, FailureTag{
				Tag:      "tool_error_loop",
				Evidence: "3+ consecutive turns with tool errors",
				TurnIdx:  turn.Index,
			})
			return
		}
	}
}

// detectReportOmittedAfterEdit flags runs that edit code but never produce a summary/report.
func detectReportOmittedAfterEdit(t *RunTrace) {
	if t.Metrics.EditCount == 0 {
		return
	}
	// Check if the last turn has a text message (report/summary).
	if len(t.Turns) == 0 {
		return
	}
	lastTurn := t.Turns[len(t.Turns)-1]
	hasReport := false
	for _, msg := range lastTurn.Messages {
		if msg.Role == "assistant" && msg.HasText && msg.TextLen > 100 {
			hasReport = true
		}
	}
	if !hasReport {
		t.FailureTags = append(t.FailureTags, FailureTag{
			Tag:      "report_omitted_after_edit",
			Evidence: "Agent edited code but did not produce a summary in the final turn",
		})
	}
}

// detectExtensionEvidence looks for signs of extensions/interventions in the transcript.
func detectExtensionEvidence(t *RunTrace) {
	for _, turn := range t.Turns {
		for _, tc := range turn.ToolCalls {
			name := strings.ToLower(tc.Name)
			// br stub invocations indicate beads integration.
			if name == "bash" {
				// Inferred — we can't see the bash command content in the trace,
				// but br-invocations.log presence would confirm.
			}
		}
		for _, msg := range turn.Messages {
			if msg.Role == "user" && msg.TextLen > 0 {
				// A mid-conversation user message (after turn 0) suggests intervention.
				if turn.Index > 0 {
					t.Extensions = append(t.Extensions, ExtensionRef{
						Name:     "user_intervention",
						Evidence: "inferred",
						TurnIdx:  turn.Index,
						Detail:   "User message appeared after initial prompt",
					})
				}
			}
		}
	}

	// Check for br stub usage via tool call names.
	if count, ok := t.Metrics.ToolCallsByName["br"]; ok && count > 0 {
		t.Extensions = append(t.Extensions, ExtensionRef{
			Name:     "beads_integration",
			Evidence: "explicit",
			Detail:   "Agent invoked br (beads) tool",
		})
	}
}

// hasTestExecution checks if any bash tool call looks like a test execution.
func hasTestExecution(t *RunTrace) bool {
	// Since we can't see bash command content in the trace, we use a heuristic:
	// If bash was called after edits, it's likely test execution.
	if t.Metrics.FirstEditTurnIdx == -1 {
		return false
	}
	bashAfterEdit := false
	for _, turn := range t.Turns {
		if turn.Index <= t.Metrics.FirstEditTurnIdx {
			continue
		}
		for _, tc := range turn.ToolCalls {
			if strings.ToLower(tc.Name) == "bash" {
				bashAfterEdit = true
				break
			}
		}
	}
	return bashAfterEdit
}

// updateTestAfterEdit sets the TestAfterEdit metric.
func updateTestAfterEdit(t *RunTrace) {
	t.Metrics.TestAfterEdit = hasTestExecution(t)
}
