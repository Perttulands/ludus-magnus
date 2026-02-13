package engine

import (
	"strings"

	"github.com/Perttulands/ludus-magnus/internal/provider"
	"github.com/Perttulands/ludus-magnus/internal/state"
)

var anthropicPricing2026 = map[string]struct {
	inputPerMillion  float64
	outputPerMillion float64
}{
	"claude-sonnet-4-5": {inputPerMillion: 3.0, outputPerMillion: 15.0},
	"claude-opus-4-6":   {inputPerMillion: 15.0, outputPerMillion: 75.0},
	"claude-haiku-4-5":  {inputPerMillion: 0.80, outputPerMillion: 4.0},
}

// ProviderResponse is the normalized response payload used for metadata capture.
type ProviderResponse struct {
	Mode     string
	Provider *string
	Model    string
	Metadata provider.Metadata
}

// CaptureExecutionMetadata extracts execution signals into state metadata fields.
func CaptureExecutionMetadata(response ProviderResponse) state.ExecutionMetadata {
	tokensInput := response.Metadata.TokensInput
	tokensOutput := response.Metadata.TokensOutput
	if tokensInput == 0 && tokensOutput == 0 && response.Metadata.TokensUsed > 0 {
		tokensOutput = response.Metadata.TokensUsed
	}

	return state.ExecutionMetadata{
		Mode:         response.Mode,
		Provider:     response.Provider,
		TokensInput:  tokensInput,
		TokensOutput: tokensOutput,
		DurationMS:   response.Metadata.DurationMs,
		CostUSD:      calculateExecutionCost(response.Provider, response.Model, tokensInput, tokensOutput, response.Metadata.CostUSD),
		ToolCalls:    toStateToolCalls(response.Metadata.ToolCalls),
	}
}

func calculateExecutionCost(providerName *string, model string, tokensInput int, tokensOutput int, fallback float64) float64 {
	if providerName == nil {
		return fallback
	}

	if strings.TrimSpace(*providerName) != "anthropic" {
		return fallback
	}

	rate, ok := anthropicPricing2026[strings.TrimSpace(model)]
	if !ok {
		return fallback
	}

	return (float64(tokensInput)*rate.inputPerMillion + float64(tokensOutput)*rate.outputPerMillion) / 1_000_000.0
}

func toStateToolCalls(toolCalls []provider.ToolCall) []state.ToolCall {
	if len(toolCalls) == 0 {
		return []state.ToolCall{}
	}

	out := make([]state.ToolCall, 0, len(toolCalls))
	for _, call := range toolCalls {
		out = append(out, state.ToolCall{
			Name:       call.Name,
			Input:      call.Input,
			Output:     call.Output,
			DurationMS: call.DurationMs,
		})
	}
	return out
}
