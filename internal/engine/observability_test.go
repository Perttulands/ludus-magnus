package engine

import (
	"testing"

	"github.com/Perttulands/ludus-magnus/internal/provider"
)

func TestCaptureExecutionMetadataAnthropicPricing(t *testing.T) {
	providerName := "anthropic"

	meta := CaptureExecutionMetadata(ProviderResponse{
		Mode:     ExecutionModeAPI,
		Provider: &providerName,
		Model:    "claude-sonnet-4-5",
		Metadata: provider.Metadata{
			TokensInput:  1_000_000,
			TokensOutput: 1_000_000,
			DurationMs:   55,
		},
	})

	if meta.CostUSD != 18.0 {
		t.Fatalf("unexpected cost: got %f want 18.0", meta.CostUSD)
	}
	if meta.TokensInput != 1_000_000 {
		t.Fatalf("unexpected tokens input: %d", meta.TokensInput)
	}
	if meta.TokensOutput != 1_000_000 {
		t.Fatalf("unexpected tokens output: %d", meta.TokensOutput)
	}
}

func TestCaptureExecutionMetadataFallsBackToProviderCost(t *testing.T) {
	providerName := "openai-compatible"

	meta := CaptureExecutionMetadata(ProviderResponse{
		Mode:     ExecutionModeAPI,
		Provider: &providerName,
		Model:    "unknown-model",
		Metadata: provider.Metadata{
			TokensInput:  100,
			TokensOutput: 50,
			DurationMs:   22,
			CostUSD:      0.1234,
		},
	})

	if meta.CostUSD != 0.1234 {
		t.Fatalf("unexpected fallback cost: %f", meta.CostUSD)
	}
}
