package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Perttulands/chiron/internal/provider"
	"github.com/Perttulands/chiron/internal/state"
)

const (
	ExecutionModeAPI    = "api"
	ExecutionModeCLI    = "cli"
	ExecutionModeSealed = "sealed"
)

type ExecuteRequest struct {
	Mode       string
	Input      string
	Definition state.AgentDefinition
	Provider   provider.Provider
	Executor   string

	// Sealed mode fields
	HarnessScript string // Path to sealed harness script (e.g. run-sealed-pi.sh)
	HarnessModel  string // Model to pass to harness (e.g. qwen3.5:9b)
	Condition     string // Condition name for harness (e.g. "minimal", "action-oriented")
	RunNumber     int    // Run number for harness
	OutputDir     string // Where harness writes results (optional, derived from harness)
}

type ExecuteResult struct {
	Output   string
	Metadata state.ExecutionMetadata
}

func Execute(ctx context.Context, req ExecuteRequest) (ExecuteResult, error) {
	mode := strings.TrimSpace(req.Mode)
	if mode == "" {
		mode = ExecutionModeAPI
	}

	switch mode {
	case ExecutionModeAPI:
		return executeAPI(ctx, req)
	case ExecutionModeCLI:
		return executeCLI(ctx, req)
	case ExecutionModeSealed:
		return executeSealed(ctx, req)
	default:
		return ExecuteResult{}, fmt.Errorf("unsupported mode %q", mode)
	}
}

func executeAPI(ctx context.Context, req ExecuteRequest) (ExecuteResult, error) {
	if req.Provider == nil {
		return ExecuteResult{}, fmt.Errorf("provider is required for api mode")
	}

	out, meta, err := req.Provider.ExecuteAgent(ctx, provider.AgentDefinition{
		SystemPrompt: req.Definition.SystemPrompt,
		Model:        req.Definition.Model,
		Temperature:  req.Definition.Temperature,
		MaxTokens:    req.Definition.MaxTokens,
	}, req.Input)
	if err != nil {
		return ExecuteResult{}, fmt.Errorf("execute provider call: %w", err)
	}

	info := req.Provider.GetMetadata()
	providerName := strings.TrimSpace(info.Provider)
	if providerName == "" {
		providerName = "unknown"
	}

	return ExecuteResult{
		Output: out,
		Metadata: CaptureExecutionMetadata(ProviderResponse{
			Mode:     ExecutionModeAPI,
			Provider: ptr(providerName),
			Model:    info.Model,
			Metadata: meta,
		}),
	}, nil
}

func executeCLI(ctx context.Context, req ExecuteRequest) (ExecuteResult, error) {
	executorName := strings.TrimSpace(req.Executor)
	switch executorName {
	case "codex", "claude":
	default:
		return ExecuteResult{}, fmt.Errorf("executor must be one of: codex, claude")
	}

	commandPath, err := exec.LookPath(executorName)
	if err != nil {
		return ExecuteResult{}, fmt.Errorf("resolve executor %q: %w", executorName, err)
	}

	start := time.Now()
	cliInput := fmt.Sprintf("system_prompt:\n%s\n\nuser_input:\n%s\n", req.Definition.SystemPrompt, req.Input)
	cmd := exec.CommandContext(ctx, commandPath)
	cmd.Stdin = strings.NewReader(cliInput)
	output, err := cmd.CombinedOutput()
	duration := int(time.Since(start).Milliseconds())
	if err != nil {
		return ExecuteResult{}, fmt.Errorf("run executor %q: %w", executorName, err)
	}

	return ExecuteResult{
		Output: strings.TrimSpace(string(output)),
		Metadata: state.ExecutionMetadata{
			Mode:            ExecutionModeCLI,
			Executor:        ptr(executorName),
			ExecutorCommand: ptr(commandPath),
			TokensInput:     0,
			TokensOutput:    0,
			DurationMS:      duration,
			CostUSD:         0,
			ToolCalls:       []state.ToolCall{},
		},
	}, nil
}

// sealedResult mirrors the JSON schema written by sealed harness scripts.
type sealedResult struct {
	Type              string         `json:"type"`
	Subtype           string         `json:"subtype"`
	Result            string         `json:"result"`
	NumTurns          int            `json:"num_turns"`
	DurationMS        int            `json:"duration_ms"`
	TotalCostUSD      float64        `json:"total_cost_usd"`
	Usage             sealedUsage    `json:"usage"`
	ToolCallsObserved []string       `json:"tool_calls_observed"`
	ToolSummary       map[string]int `json:"tool_summary"`
	Model             string         `json:"model"`
	Executor          string         `json:"executor"`
}

type sealedUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

func executeSealed(ctx context.Context, req ExecuteRequest) (ExecuteResult, error) {
	if strings.TrimSpace(req.HarnessScript) == "" {
		return ExecuteResult{}, fmt.Errorf("harness script path is required for sealed mode")
	}

	if strings.TrimSpace(req.HarnessModel) == "" {
		return ExecuteResult{}, fmt.Errorf("harness model is required for sealed mode")
	}

	harnessPath := req.HarnessScript
	if _, err := os.Stat(harnessPath); err != nil {
		return ExecuteResult{}, fmt.Errorf("harness script not found: %w", err)
	}

	// Write system prompt to a temp file
	promptFile, err := os.CreateTemp("", "chiron-sealed-prompt-*.txt")
	if err != nil {
		return ExecuteResult{}, fmt.Errorf("create temp prompt file: %w", err)
	}
	defer os.Remove(promptFile.Name())

	if _, err := promptFile.WriteString(req.Definition.SystemPrompt); err != nil {
		promptFile.Close()
		return ExecuteResult{}, fmt.Errorf("write system prompt: %w", err)
	}
	promptFile.Close()

	// Derive condition and run number
	condition := strings.TrimSpace(req.Condition)
	if condition == "" {
		condition = "chiron"
	}

	runNumber := req.RunNumber
	if runNumber == 0 {
		runNumber = int(time.Now().Unix())
	}

	// Call: bash <HarnessScript> <Condition> <RunNumber> <tempPromptFile> <HarnessModel>
	cmd := exec.CommandContext(ctx, "bash", harnessPath,
		condition,
		fmt.Sprintf("%d", runNumber),
		promptFile.Name(),
		req.HarnessModel,
	)
	cmd.Env = append(os.Environ(), fmt.Sprintf("CHIRON_INPUT=%s", req.Input))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return ExecuteResult{}, fmt.Errorf("run sealed harness: %w\noutput: %s", err, string(output))
	}

	// Find result.json — check OutputDir first, then look in harness output for path
	resultPath := ""
	if strings.TrimSpace(req.OutputDir) != "" {
		candidate := filepath.Join(req.OutputDir, "result.json")
		if _, statErr := os.Stat(candidate); statErr == nil {
			resultPath = candidate
		}
	}

	// Fallback: scan harness stdout for result location
	if resultPath == "" {
		for _, line := range strings.Split(string(output), "\n") {
			trimmed := strings.TrimSpace(line)
			// Direct result.json path
			if strings.HasSuffix(trimmed, "result.json") {
				if _, statErr := os.Stat(trimmed); statErr == nil {
					resultPath = trimmed
					break
				}
			}
			// "run complete: <dir>" pattern from run-sealed-pi.sh
			if strings.HasPrefix(trimmed, "run complete: ") {
				dir := strings.TrimPrefix(trimmed, "run complete: ")
				candidate := filepath.Join(dir, "result.json")
				if _, statErr := os.Stat(candidate); statErr == nil {
					resultPath = candidate
					break
				}
			}
		}
	}

	if resultPath == "" {
		return ExecuteResult{}, fmt.Errorf("sealed harness did not produce a result.json (output: %s)", string(output))
	}

	return parseSealedResult(resultPath)
}

func parseSealedResult(resultPath string) (ExecuteResult, error) {
	data, err := os.ReadFile(resultPath)
	if err != nil {
		return ExecuteResult{}, fmt.Errorf("read sealed result: %w", err)
	}

	var sr sealedResult
	if err := json.Unmarshal(data, &sr); err != nil {
		return ExecuteResult{}, fmt.Errorf("parse sealed result: %w", err)
	}

	// Convert tool_calls_observed + tool_summary to state.ToolCall slice
	toolCalls := make([]state.ToolCall, 0, len(sr.ToolCallsObserved))
	for _, name := range sr.ToolCallsObserved {
		toolCalls = append(toolCalls, state.ToolCall{Name: name})
	}

	executorName := sr.Executor
	if executorName == "" {
		executorName = "sealed"
	}

	return ExecuteResult{
		Output: sr.Result,
		Metadata: state.ExecutionMetadata{
			Mode:         ExecutionModeSealed,
			Executor:     ptr(executorName),
			TokensInput:  sr.Usage.InputTokens,
			TokensOutput: sr.Usage.OutputTokens,
			DurationMS:   sr.DurationMS,
			CostUSD:      sr.TotalCostUSD,
			ToolCalls:    toolCalls,
		},
	}, nil
}

func ptr(value string) *string {
	return &value
}
