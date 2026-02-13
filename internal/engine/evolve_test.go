package engine

import (
	"strings"
	"testing"

	"github.com/Perttulands/ludus-magnus/internal/state"
)

func TestGenerateEvolutionPromptIncludesFeedbackAndAverage(t *testing.T) {
	agents := []state.Agent{
		{
			Version: 1,
			Definition: state.AgentDefinition{
				SystemPrompt: "You are a helpful coding assistant.",
			},
		},
	}

	artifacts := []state.Artifact{
		{Evaluation: &state.Evaluation{Score: 4, Comment: "Too verbose"}},
		{Evaluation: &state.Evaluation{Score: 7, Comment: "Good structure"}},
		{Evaluation: &state.Evaluation{Score: 9, Comment: "Excellent edge-case handling"}},
	}

	directives := []state.Directive{{Text: "Preserve concise error handling."}}

	prompt := GenerateEvolutionPrompt(agents, artifacts, directives)

	for _, feedback := range []string{"Too verbose", "Good structure", "Excellent edge-case handling"} {
		if !strings.Contains(prompt, feedback) {
			t.Fatalf("expected prompt to include feedback %q, got:\n%s", feedback, prompt)
		}
	}

	if !strings.Contains(prompt, "Average score: 6.67/10") {
		t.Fatalf("expected average score 6.67 in prompt, got:\n%s", prompt)
	}

	if !strings.Contains(prompt, "System Prompt: You are a helpful coding assistant.") {
		t.Fatalf("expected prompt to include previous system prompt verbatim, got:\n%s", prompt)
	}

	if !strings.Contains(prompt, "Preserve concise error handling.") {
		t.Fatalf("expected prompt to include directive text, got:\n%s", prompt)
	}
}

func TestGenerateEvolutionPromptWithoutEvaluationsStillGeneratesPrompt(t *testing.T) {
	agents := []state.Agent{
		{
			Version: 2,
			Definition: state.AgentDefinition{
				SystemPrompt: "You are a precise refactoring agent.",
			},
		},
	}

	artifacts := []state.Artifact{{ID: "art_1"}, {ID: "art_2"}}
	directives := []state.Directive{{Text: "Always propose minimal diffs."}}

	prompt := GenerateEvolutionPrompt(agents, artifacts, directives)

	if !strings.Contains(prompt, "No evaluation yet") {
		t.Fatalf("expected no-evaluation summary in prompt, got:\n%s", prompt)
	}

	if !strings.Contains(prompt, "Always propose minimal diffs.") {
		t.Fatalf("expected directive in prompt, got:\n%s", prompt)
	}

	if !strings.Contains(prompt, "System Prompt: You are a precise refactoring agent.") {
		t.Fatalf("expected current system prompt in prompt, got:\n%s", prompt)
	}
}
