package state_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Perttulands/ludus-magnus/internal/state"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	t.Parallel()

	testPath := filepath.Join(t.TempDir(), "test-state.json")
	want := sampleState()

	if err := state.Save(testPath, want); err != nil {
		t.Fatalf("save state: %v", err)
	}

	got, err := state.Load(testPath)
	if err != nil {
		t.Fatalf("load state: %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round-trip mismatch\nwant: %#v\n got: %#v", want, got)
	}
}

func TestSaveCreatesStateDirectory(t *testing.T) {
	t.Parallel()

	testPath := filepath.Join(t.TempDir(), ".ludus-magnus", "state.json")

	if err := state.Save(testPath, state.NewState()); err != nil {
		t.Fatalf("save state: %v", err)
	}

	if info, err := os.Stat(filepath.Dir(testPath)); err != nil {
		t.Fatalf("stat state directory: %v", err)
	} else if !info.IsDir() {
		t.Fatalf("expected directory at %s", filepath.Dir(testPath))
	}
}

func TestSaveUsesDefaultStatePath(t *testing.T) {
	tempDir := t.TempDir()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("change to temp directory: %v", err)
	}

	if err := state.Save("", state.NewState()); err != nil {
		t.Fatalf("save state with default path: %v", err)
	}

	defaultPath := filepath.Join(tempDir, ".ludus-magnus", "state.json")
	if info, err := os.Stat(defaultPath); err != nil {
		t.Fatalf("stat default state path: %v", err)
	} else if info.Size() == 0 {
		t.Fatalf("expected non-empty state file at %s", defaultPath)
	}
}

func sampleState() state.State {
	provider := "anthropic"
	executor := "claude"
	executorCmd := "claude -p ..."

	return state.State{
		Version: "1.0",
		Sessions: map[string]state.Session{
			"ses_abc123": {
				ID:        "ses_abc123",
				Mode:      "quickstart",
				Need:      "customer care agent",
				CreatedAt: "2026-02-13T10:30:00Z",
				Status:    "active",
				Lineages: map[string]state.Lineage{
					"main": {
						ID:        "lin_xyz789",
						SessionID: "ses_abc123",
						Name:      "main",
						Locked:    false,
						Agents: []state.Agent{
							{
								ID:        "agt_def456",
								LineageID: "lin_xyz789",
								Version:   1,
								Definition: state.AgentDefinition{
									SystemPrompt: "You are...",
									Model:        "claude-sonnet-4-5",
									Temperature:  1.0,
									MaxTokens:    4096,
									Tools:        []any{},
								},
								CreatedAt: "2026-02-13T10:31:00Z",
								GenerationMetadata: state.GenerationMetadata{
									Provider:   "anthropic",
									Model:      "claude-sonnet-4-5",
									TokensUsed: 1234,
									DurationMS: 567,
									CostUSD:    0.0123,
								},
							},
						},
						Artifacts: []state.Artifact{
							{
								ID:        "art_ghi789",
								AgentID:   "agt_def456",
								Input:     "test input string",
								Output:    "agent response",
								CreatedAt: "2026-02-13T10:32:00Z",
								ExecutionMetadata: state.ExecutionMetadata{
									Mode:            "api",
									Provider:        &provider,
									Executor:        &executor,
									ExecutorCommand: &executorCmd,
									TokensInput:     100,
									TokensOutput:    500,
									DurationMS:      2345,
									CostUSD:         0.0456,
									ToolCalls: []state.ToolCall{
										{
											Name:       "get_customer_info",
											Input:      `{"customer_id":"123"}`,
											Output:     `{"name":"Alice"}`,
											DurationMS: 123,
										},
									},
								},
								Evaluation: &state.Evaluation{
									Score:       8,
									Comment:     "good response but tone could be friendlier",
									EvaluatedAt: "2026-02-13T10:35:00Z",
								},
							},
						},
						Directives: state.Directives{
							Oneshot: []state.Directive{},
							Sticky: []state.Directive{
								{
									ID:        "dir_jkl012",
									Text:      "always use friendly tone",
									CreatedAt: "2026-02-13T10:36:00Z",
								},
							},
						},
					},
				},
			},
		},
	}
}
