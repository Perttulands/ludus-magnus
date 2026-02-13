package engine

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Perttulands/ludus-magnus/internal/provider"
	"github.com/Perttulands/ludus-magnus/internal/state"
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
			TokensUsed: 33,
			DurationMs: 12,
			CostUSD:    0.0042,
		},
		info: provider.ProviderInfo{
			Provider: "openai-compatible",
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
	if result.Metadata.TokensOutput != 33 {
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
