package challenge

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/Perttulands/ludus-magnus/internal/provider"
)

type mockProvider struct {
	response string
	err      error
}

func (m *mockProvider) GenerateAgent(_ context.Context, need string, _ []string) (provider.AgentDefinition, provider.Metadata, error) {
	if m.err != nil {
		return provider.AgentDefinition{}, provider.Metadata{}, m.err
	}
	return provider.AgentDefinition{SystemPrompt: m.response}, provider.Metadata{}, nil
}

func (m *mockProvider) ExecuteAgent(_ context.Context, _ provider.AgentDefinition, _ string) (string, provider.Metadata, error) {
	return "", provider.Metadata{}, nil
}

func (m *mockProvider) GetMetadata() provider.ProviderInfo {
	return provider.ProviderInfo{Provider: "mock"}
}

var idCounter int

func testIDFunc(prefix string) string {
	idCounter++
	return fmt.Sprintf("%s_%04d", prefix, idCounter)
}

func TestGenerateSuccess(t *testing.T) {
	idCounter = 0
	challengeJSON := generatedChallenge{
		Name:        "Implement hello",
		Description: "Create a hello world function",
		Input:       "Write a function that returns hello world",
		Context:     "",
		TestCases: []generatedTestCase{
			{Name: "has hello", Type: "contains", Expected: "hello", Weight: 1.0},
			{Name: "has world", Type: "contains", Expected: "world", Weight: 1.0},
		},
	}
	data, _ := json.Marshal(challengeJSON)

	mock := &mockProvider{response: string(data)}
	ch, err := Generate(context.Background(), GenerateRequest{
		Type:       TypeFeature,
		Difficulty: DifficultyEasy,
		Domain:     "testing",
	}, mock, testIDFunc)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ch.Name != "Implement hello" {
		t.Errorf("Name = %q, want %q", ch.Name, "Implement hello")
	}
	if ch.Type != TypeFeature {
		t.Errorf("Type = %q, want %q", ch.Type, TypeFeature)
	}
	if len(ch.TestSuite.TestCases) != 2 {
		t.Errorf("got %d test cases, want 2", len(ch.TestSuite.TestCases))
	}
}

func TestGenerateNilProvider(t *testing.T) {
	_, err := Generate(context.Background(), GenerateRequest{}, nil, testIDFunc)
	if err == nil {
		t.Error("expected error for nil provider")
	}
}

func TestGenerateInvalidType(t *testing.T) {
	mock := &mockProvider{}
	_, err := Generate(context.Background(), GenerateRequest{Type: "invalid"}, mock, testIDFunc)
	if err == nil {
		t.Error("expected error for invalid type")
	}
}

func TestGenerateProviderError(t *testing.T) {
	mock := &mockProvider{err: fmt.Errorf("api failed")}
	_, err := Generate(context.Background(), GenerateRequest{Type: TypeFeature}, mock, testIDFunc)
	if err == nil {
		t.Error("expected error when provider fails")
	}
}

func TestGenerateDefaultType(t *testing.T) {
	idCounter = 100
	challengeJSON := generatedChallenge{
		Name:        "Default",
		Description: "desc",
		Input:       "input",
		TestCases:   []generatedTestCase{{Name: "t1", Type: "contains", Expected: "x", Weight: 1}},
	}
	data, _ := json.Marshal(challengeJSON)
	mock := &mockProvider{response: string(data)}

	ch, err := Generate(context.Background(), GenerateRequest{}, mock, testIDFunc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ch.Type != TypeFeature {
		t.Errorf("default type = %q, want %q", ch.Type, TypeFeature)
	}
}

func TestBuildGenerationPrompt(t *testing.T) {
	prompt := buildGenerationPrompt(TypeBugfix, DifficultyHard, "web API")
	if !strings.Contains(prompt, "bugfix") {
		t.Error("prompt should contain challenge type")
	}
	if !strings.Contains(prompt, "hard") {
		t.Error("prompt should contain difficulty")
	}
	if !strings.Contains(prompt, "web API") {
		t.Error("prompt should contain domain")
	}
}

func TestGenerateBatchZero(t *testing.T) {
	mock := &mockProvider{}
	_, err := GenerateBatch(context.Background(), 0, GenerateRequest{}, mock, testIDFunc)
	if err == nil {
		t.Error("expected error for zero count")
	}
}
