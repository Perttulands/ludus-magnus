package engine

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/Perttulands/ludus-magnus/internal/provider"
	"github.com/Perttulands/ludus-magnus/internal/state"
)

const (
	ExecutionModeAPI = "api"
	ExecutionModeCLI = "cli"
)

type ExecuteRequest struct {
	Mode       string
	Input      string
	Definition state.AgentDefinition
	Provider   provider.Provider
	Executor   string
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

func ptr(value string) *string {
	return &value
}
