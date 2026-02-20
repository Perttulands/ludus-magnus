package mutation

import (
	"context"
	"math/rand"
	"testing"

	"github.com/Perttulands/ludus-magnus/internal/provider"
	"github.com/Perttulands/ludus-magnus/internal/state"
)

type mockProvider struct {
	systemPrompt string
	err          error
}

func (m *mockProvider) GenerateAgent(_ context.Context, _ string, _ []string) (provider.AgentDefinition, provider.Metadata, error) {
	if m.err != nil {
		return provider.AgentDefinition{}, provider.Metadata{}, m.err
	}
	return provider.AgentDefinition{SystemPrompt: m.systemPrompt}, provider.Metadata{}, nil
}

func (m *mockProvider) ExecuteAgent(_ context.Context, _ provider.AgentDefinition, _ string) (string, provider.Metadata, error) {
	return "", provider.Metadata{}, nil
}

func (m *mockProvider) GetMetadata() provider.ProviderInfo {
	return provider.ProviderInfo{Provider: "mock"}
}

func baseAgent() state.AgentDefinition {
	return state.AgentDefinition{
		SystemPrompt: "You are a helpful assistant.",
		Model:        "claude-sonnet-4-5",
		Temperature:  1.0,
		MaxTokens:    4096,
	}
}

func TestRephraseOp(t *testing.T) {
	mock := &mockProvider{systemPrompt: "You are a friendly helper."}
	op := RephraseOp{}

	result, err := op.Mutate(context.Background(), baseAgent(), mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.SystemPrompt != "You are a friendly helper." {
		t.Errorf("unexpected prompt: %q", result.SystemPrompt)
	}
	if result.Model != "claude-sonnet-4-5" {
		t.Errorf("model should be preserved: %q", result.Model)
	}
	if op.Name() != OpRephrase {
		t.Errorf("name = %q, want %q", op.Name(), OpRephrase)
	}
}

func TestExpandOp(t *testing.T) {
	mock := &mockProvider{systemPrompt: "You are a helpful assistant. Always provide examples."}
	op := ExpandOp{}

	result, err := op.Mutate(context.Background(), baseAgent(), mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.SystemPrompt == "" {
		t.Error("expected non-empty prompt")
	}
	if op.Name() != OpExpand {
		t.Errorf("name = %q, want %q", op.Name(), OpExpand)
	}
}

func TestSimplifyOp(t *testing.T) {
	mock := &mockProvider{systemPrompt: "Be helpful."}
	op := SimplifyOp{}

	result, err := op.Mutate(context.Background(), baseAgent(), mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.SystemPrompt != "Be helpful." {
		t.Errorf("unexpected prompt: %q", result.SystemPrompt)
	}
	if op.Name() != OpSimplify {
		t.Errorf("name = %q, want %q", op.Name(), OpSimplify)
	}
}

func TestCrossoverOp(t *testing.T) {
	mock := &mockProvider{systemPrompt: "Combined prompt result."}
	partner := state.AgentDefinition{SystemPrompt: "Another approach."}
	op := CrossoverOp{Partner: partner}

	result, err := op.Mutate(context.Background(), baseAgent(), mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.SystemPrompt != "Combined prompt result." {
		t.Errorf("unexpected prompt: %q", result.SystemPrompt)
	}
	if op.Name() != OpCrossover {
		t.Errorf("name = %q, want %q", op.Name(), OpCrossover)
	}
}

func TestTargetedOp(t *testing.T) {
	mock := &mockProvider{systemPrompt: "Improved for speed."}
	op := TargetedOp{Directive: "Focus on response speed"}

	result, err := op.Mutate(context.Background(), baseAgent(), mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.SystemPrompt != "Improved for speed." {
		t.Errorf("unexpected prompt: %q", result.SystemPrompt)
	}
	if op.Name() != OpTargeted {
		t.Errorf("name = %q, want %q", op.Name(), OpTargeted)
	}
}

func TestMutateNilProvider(t *testing.T) {
	op := RephraseOp{}
	_, err := op.Mutate(context.Background(), baseAgent(), nil)
	if err == nil {
		t.Error("expected error for nil provider")
	}
}

func TestMutateEmptyResult(t *testing.T) {
	mock := &mockProvider{systemPrompt: ""}
	op := RephraseOp{}
	_, err := op.Mutate(context.Background(), baseAgent(), mock)
	if err == nil {
		t.Error("expected error for empty mutation result")
	}
}

func TestNewOperatorValid(t *testing.T) {
	for _, name := range AllOperators {
		op, err := NewOperator(name)
		if err != nil {
			t.Errorf("NewOperator(%q) error: %v", name, err)
		}
		if op == nil {
			t.Errorf("NewOperator(%q) returned nil", name)
		}
	}
}

func TestNewOperatorInvalid(t *testing.T) {
	_, err := NewOperator("invalid")
	if err == nil {
		t.Error("expected error for invalid operator")
	}
}

func TestRandomOperator(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	op := RandomOperator(rng)
	if op == nil {
		t.Error("RandomOperator returned nil")
	}
	name := op.Name()
	if name != OpRephrase && name != OpExpand && name != OpSimplify {
		t.Errorf("unexpected random operator: %q", name)
	}
}

func TestRandomOperatorNilRng(t *testing.T) {
	op := RandomOperator(nil)
	if op == nil {
		t.Error("RandomOperator(nil) returned nil")
	}
}

func TestMutatePreservesModelSettings(t *testing.T) {
	mock := &mockProvider{systemPrompt: "mutated"}
	agent := state.AgentDefinition{
		SystemPrompt: "original",
		Model:        "custom-model",
		Temperature:  0.5,
		MaxTokens:    2048,
		Tools:        []any{"tool1"},
	}

	op := RephraseOp{}
	result, err := op.Mutate(context.Background(), agent, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Model != "custom-model" {
		t.Errorf("Model = %q, want %q", result.Model, "custom-model")
	}
	if result.Temperature != 0.5 {
		t.Errorf("Temperature = %f, want 0.5", result.Temperature)
	}
	if result.MaxTokens != 2048 {
		t.Errorf("MaxTokens = %d, want 2048", result.MaxTokens)
	}
}
