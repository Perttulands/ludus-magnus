package challenge

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Perttulands/ludus-magnus/internal/harness"
	"github.com/Perttulands/ludus-magnus/internal/provider"
)

// GenerateRequest specifies what kind of challenge to generate.
type GenerateRequest struct {
	Type       string // feature, bugfix, refactor, review
	Difficulty string // easy, medium, hard
	Domain     string // e.g., "web API", "CLI tool", "data processing"
	Tags       []string
}

// generatedChallenge is the JSON structure the LLM returns.
type generatedChallenge struct {
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Input       string               `json:"input"`
	Context     string               `json:"context"`
	TestCases   []generatedTestCase  `json:"test_cases"`
}

type generatedTestCase struct {
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	Expected string  `json:"expected"`
	Weight   float64 `json:"weight"`
}

// Generate creates a synthetic challenge using an LLM provider.
func Generate(ctx context.Context, req GenerateRequest, p provider.Provider, idFunc func(string) string) (Challenge, error) {
	if p == nil {
		return Challenge{}, fmt.Errorf("provider is required")
	}

	challengeType := strings.TrimSpace(req.Type)
	if challengeType == "" {
		challengeType = TypeFeature
	}
	if !isValidType(challengeType) {
		return Challenge{}, fmt.Errorf("invalid challenge type %q", challengeType)
	}

	difficulty := strings.TrimSpace(req.Difficulty)
	if difficulty == "" {
		difficulty = DifficultyMedium
	}

	domain := strings.TrimSpace(req.Domain)
	if domain == "" {
		domain = "general software engineering"
	}

	prompt := buildGenerationPrompt(challengeType, difficulty, domain)
	generated, _, err := p.GenerateAgent(ctx, prompt, nil)
	if err != nil {
		return Challenge{}, fmt.Errorf("generate challenge: %w", err)
	}

	var parsed generatedChallenge
	if err := json.Unmarshal([]byte(generated.SystemPrompt), &parsed); err != nil {
		return Challenge{}, fmt.Errorf("parse challenge response: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	challengeID := idFunc("ch")
	suiteID := idFunc("ts")

	testCases := make([]harness.TestCase, 0, len(parsed.TestCases))
	for i, tc := range parsed.TestCases {
		tcID := idFunc("tc")
		weight := tc.Weight
		if weight <= 0 {
			weight = 1.0
		}
		tcType := tc.Type
		if tcType == "" {
			tcType = "contains"
		}
		testCases = append(testCases, harness.TestCase{
			ID:       tcID,
			Name:     tc.Name,
			Type:     tcType,
			Expected: tc.Expected,
			Weight:   weight,
		})
		_ = i
	}

	return Challenge{
		ID:          challengeID,
		Name:        parsed.Name,
		Type:        challengeType,
		Difficulty:  difficulty,
		Description: parsed.Description,
		Input:       parsed.Input,
		Context:     parsed.Context,
		TestSuite: harness.TestSuite{
			ID:        suiteID,
			Name:      fmt.Sprintf("Tests for %s", parsed.Name),
			TestCases: testCases,
		},
		Tags:      req.Tags,
		CreatedAt: now,
	}, nil
}

func buildGenerationPrompt(challengeType, difficulty, domain string) string {
	return fmt.Sprintf(`Generate a synthetic %s challenge for AI agent evaluation.

Domain: %s
Difficulty: %s

Create a challenge that tests an AI agent's ability to handle a %s task.
The challenge should be realistic and have clear evaluation criteria.

Output a JSON object:
{
  "name": "short challenge name",
  "description": "detailed description of what the agent must do",
  "input": "the exact prompt/input the agent will receive",
  "context": "any code or context the agent needs (can be empty string)",
  "test_cases": [
    {
      "name": "test case name",
      "type": "contains|not_contains|regex|equals",
      "expected": "the expected pattern or value",
      "weight": 1.0
    }
  ]
}

Include 3-5 test cases that verify the agent's output quality.
For %s difficulty, calibrate complexity accordingly.`, challengeType, domain, difficulty, challengeType, difficulty)
}

// GenerateBatch creates multiple challenges at once.
func GenerateBatch(ctx context.Context, count int, req GenerateRequest, p provider.Provider, idFunc func(string) string) ([]Challenge, error) {
	if count <= 0 {
		return nil, fmt.Errorf("count must be positive")
	}

	challenges := make([]Challenge, 0, count)
	for i := 0; i < count; i++ {
		ch, err := Generate(ctx, req, p, idFunc)
		if err != nil {
			return challenges, fmt.Errorf("generate challenge %d/%d: %w", i+1, count, err)
		}
		challenges = append(challenges, ch)
	}
	return challenges, nil
}
