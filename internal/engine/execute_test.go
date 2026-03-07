package engine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Perttulands/chiron/internal/provider"
	"github.com/Perttulands/chiron/internal/state"
)

type execMockProvider struct {
	output string
	meta   provider.Metadata
	info   provider.ProviderInfo
}

func (m *execMockProvider) GenerateAgent(context.Context, string, []string) (provider.AgentDefinition, provider.Metadata, error) {
	return provider.AgentDefinition{}, provider.Metadata{}, nil
}

func (m *execMockProvider) ExecuteAgent(context.Context, provider.AgentDefinition, string) (string, provider.Metadata, error) {
	return m.output, m.meta, nil
}

func (m *execMockProvider) GetMetadata() provider.ProviderInfo {
	return m.info
}

func TestExecuteAPIMode(t *testing.T) {
	p := &execMockProvider{
		output: "provider output",
		meta: provider.Metadata{
			TokensInput:  21,
			TokensOutput: 12,
			TokensUsed:   33,
			DurationMs:   12,
			CostUSD:      0.0042,
		},
		info: provider.ProviderInfo{
			Provider: "openai-compatible",
			Model:    "gpt-4o-mini",
		},
	}

	result, err := Execute(context.Background(), ExecuteRequest{
		Mode:       ExecutionModeAPI,
		Input:      "hello",
		Definition: state.AgentDefinition{SystemPrompt: "sys", Model: "m", Temperature: 1.0, MaxTokens: 100},
		Provider:   p,
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result.Output != "provider output" {
		t.Fatalf("unexpected output: %q", result.Output)
	}
	if result.Metadata.Mode != ExecutionModeAPI {
		t.Fatalf("unexpected mode: %q", result.Metadata.Mode)
	}
	if result.Metadata.Provider == nil || *result.Metadata.Provider != "openai-compatible" {
		t.Fatalf("unexpected provider metadata: %+v", result.Metadata.Provider)
	}
	if result.Metadata.TokensInput != 21 {
		t.Fatalf("unexpected tokens input: %d", result.Metadata.TokensInput)
	}
	if result.Metadata.TokensOutput != 12 {
		t.Fatalf("unexpected tokens output: %d", result.Metadata.TokensOutput)
	}
	if result.Metadata.CostUSD != 0.0042 {
		t.Fatalf("unexpected cost: %f", result.Metadata.CostUSD)
	}
}

func TestExecuteCLIMode(t *testing.T) {
	tempDir := t.TempDir()
	executorPath := filepath.Join(tempDir, "codex")
	script := "#!/usr/bin/env bash\ncat >/dev/null\necho cli-output\n"
	if err := os.WriteFile(executorPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake executor: %v", err)
	}

	originalPath := os.Getenv("PATH")
	t.Setenv("PATH", tempDir+string(os.PathListSeparator)+originalPath)

	result, err := Execute(context.Background(), ExecuteRequest{
		Mode:       ExecutionModeCLI,
		Input:      "hello",
		Definition: state.AgentDefinition{SystemPrompt: "sys"},
		Executor:   "codex",
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result.Output != "cli-output" {
		t.Fatalf("unexpected output: %q", result.Output)
	}
	if result.Metadata.Mode != ExecutionModeCLI {
		t.Fatalf("unexpected mode: %q", result.Metadata.Mode)
	}
	if result.Metadata.Executor == nil || *result.Metadata.Executor != "codex" {
		t.Fatalf("unexpected executor metadata: %+v", result.Metadata.Executor)
	}
	if result.Metadata.ExecutorCommand == nil || !strings.Contains(*result.Metadata.ExecutorCommand, "codex") {
		t.Fatalf("unexpected executor command: %+v", result.Metadata.ExecutorCommand)
	}
}

func TestExecuteSealedMissingHarness(t *testing.T) {
	_, err := Execute(context.Background(), ExecuteRequest{
		Mode:          ExecutionModeSealed,
		Input:         "hello",
		Definition:    state.AgentDefinition{SystemPrompt: "sys"},
		HarnessScript: "/nonexistent/path/harness.sh",
		HarnessModel:  "qwen3.5:9b",
	})
	if err == nil {
		t.Fatal("expected error for missing harness script")
	}
	if !strings.Contains(err.Error(), "harness script not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecuteSealedMissingModel(t *testing.T) {
	_, err := Execute(context.Background(), ExecuteRequest{
		Mode:          ExecutionModeSealed,
		Input:         "hello",
		Definition:    state.AgentDefinition{SystemPrompt: "sys"},
		HarnessScript: "testdata/mock-harness.sh",
	})
	if err == nil {
		t.Fatal("expected error for missing harness model")
	}
	if !strings.Contains(err.Error(), "harness model is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecuteSealedParseResult(t *testing.T) {
	// Write a mock result.json and test parseSealedResult directly
	tmpDir := t.TempDir()
	resultPath := filepath.Join(tmpDir, "result.json")

	resultData := sealedResult{
		Type:              "result",
		Subtype:           "success",
		Result:            "parsed response text",
		NumTurns:          5,
		DurationMS:        60000,
		TotalCostUSD:      0.0,
		Usage:             sealedUsage{InputTokens: 1000, OutputTokens: 500},
		ToolCallsObserved: []string{"read", "bash", "edit"},
		ToolSummary:       map[string]int{"read": 2, "bash": 3, "edit": 1},
		Model:             "qwen3.5:9b",
		Executor:          "pi-cli",
	}

	data, err := json.Marshal(resultData)
	if err != nil {
		t.Fatalf("marshal test data: %v", err)
	}
	if err := os.WriteFile(resultPath, data, 0o644); err != nil {
		t.Fatalf("write test result.json: %v", err)
	}

	result, err := parseSealedResult(resultPath)
	if err != nil {
		t.Fatalf("parseSealedResult returned error: %v", err)
	}

	if result.Output != "parsed response text" {
		t.Fatalf("unexpected output: %q", result.Output)
	}
	if result.Metadata.Mode != ExecutionModeSealed {
		t.Fatalf("unexpected mode: %q", result.Metadata.Mode)
	}
	if result.Metadata.TokensInput != 1000 {
		t.Fatalf("unexpected tokens input: %d", result.Metadata.TokensInput)
	}
	if result.Metadata.TokensOutput != 500 {
		t.Fatalf("unexpected tokens output: %d", result.Metadata.TokensOutput)
	}
	if result.Metadata.DurationMS != 60000 {
		t.Fatalf("unexpected duration: %d", result.Metadata.DurationMS)
	}
	if result.Metadata.Executor == nil || *result.Metadata.Executor != "pi-cli" {
		t.Fatalf("unexpected executor: %+v", result.Metadata.Executor)
	}
	if len(result.Metadata.ToolCalls) != 3 {
		t.Fatalf("unexpected tool calls count: %d", len(result.Metadata.ToolCalls))
	}
	if result.Metadata.ToolCalls[0].Name != "read" {
		t.Fatalf("unexpected first tool call: %q", result.Metadata.ToolCalls[0].Name)
	}
}

func TestExecuteSealedEndToEnd(t *testing.T) {
	// Use the mock harness script for an end-to-end test
	harnessPath, err := filepath.Abs("testdata/mock-harness.sh")
	if err != nil {
		t.Fatalf("resolve harness path: %v", err)
	}

	if _, err := os.Stat(harnessPath); err != nil {
		t.Skipf("mock harness not found at %s", harnessPath)
	}

	result, err := Execute(context.Background(), ExecuteRequest{
		Mode:          ExecutionModeSealed,
		Input:         "test input",
		Definition:    state.AgentDefinition{SystemPrompt: "you are a test agent"},
		HarnessScript: harnessPath,
		HarnessModel:  "qwen3.5:9b",
		Condition:     "minimal",
		RunNumber:     42,
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if !strings.Contains(result.Output, "mock sealed response") {
		t.Fatalf("unexpected output: %q", result.Output)
	}
	if result.Metadata.Mode != ExecutionModeSealed {
		t.Fatalf("unexpected mode: %q", result.Metadata.Mode)
	}
	if result.Metadata.TokensInput != 800 {
		t.Fatalf("unexpected tokens input: %d", result.Metadata.TokensInput)
	}
	if result.Metadata.TokensOutput != 350 {
		t.Fatalf("unexpected tokens output: %d", result.Metadata.TokensOutput)
	}
	if result.Metadata.Executor == nil || *result.Metadata.Executor != "mock-harness" {
		t.Fatalf("unexpected executor: %+v", result.Metadata.Executor)
	}
}
