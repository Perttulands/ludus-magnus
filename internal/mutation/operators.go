package mutation

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"

	"github.com/Perttulands/ludus-magnus/internal/provider"
	"github.com/Perttulands/ludus-magnus/internal/state"
)

// Operator names for mutation strategies.
const (
	OpRephrase   = "rephrase"
	OpExpand     = "expand"
	OpSimplify   = "simplify"
	OpCrossover  = "crossover"
	OpTargeted   = "targeted"
)

// AllOperators lists all available mutation operators.
var AllOperators = []string{OpRephrase, OpExpand, OpSimplify, OpCrossover, OpTargeted}

// Operator mutates an agent definition to produce a variant.
type Operator interface {
	Name() string
	Mutate(ctx context.Context, agent state.AgentDefinition, p provider.Provider) (state.AgentDefinition, error)
}

// RephraseOp rewrites the prompt with different wording while preserving intent.
type RephraseOp struct{}

func (RephraseOp) Name() string { return OpRephrase }
func (RephraseOp) Mutate(ctx context.Context, agent state.AgentDefinition, p provider.Provider) (state.AgentDefinition, error) {
	return mutateWithPrompt(ctx, agent, p, fmt.Sprintf(
		`Rephrase this system prompt using different wording while preserving the exact same intent and instructions.
Keep the same level of detail. Change sentence structure, vocabulary, and phrasing.

Original prompt:
%s

Output JSON: {"system_prompt": "the rephrased prompt"}`, agent.SystemPrompt))
}

// ExpandOp adds more detail, examples, and edge cases to the prompt.
type ExpandOp struct{}

func (ExpandOp) Name() string { return OpExpand }
func (ExpandOp) Mutate(ctx context.Context, agent state.AgentDefinition, p provider.Provider) (state.AgentDefinition, error) {
	return mutateWithPrompt(ctx, agent, p, fmt.Sprintf(
		`Expand this system prompt by adding more detail, examples, and edge case handling.
Make it more thorough without changing the core instructions.

Original prompt:
%s

Output JSON: {"system_prompt": "the expanded prompt"}`, agent.SystemPrompt))
}

// SimplifyOp makes the prompt shorter and more direct.
type SimplifyOp struct{}

func (SimplifyOp) Name() string { return OpSimplify }
func (SimplifyOp) Mutate(ctx context.Context, agent state.AgentDefinition, p provider.Provider) (state.AgentDefinition, error) {
	return mutateWithPrompt(ctx, agent, p, fmt.Sprintf(
		`Simplify this system prompt. Remove redundancy, tighten wording, keep only essential instructions.
The result should be shorter but equally effective.

Original prompt:
%s

Output JSON: {"system_prompt": "the simplified prompt"}`, agent.SystemPrompt))
}

// CrossoverOp combines elements from two prompts.
type CrossoverOp struct {
	Partner state.AgentDefinition
}

func (c CrossoverOp) Name() string { return OpCrossover }
func (c CrossoverOp) Mutate(ctx context.Context, agent state.AgentDefinition, p provider.Provider) (state.AgentDefinition, error) {
	return mutateWithPrompt(ctx, agent, p, fmt.Sprintf(
		`Combine the best elements of these two system prompts into a single improved prompt.
Take the strongest instructions from each.

Prompt A:
%s

Prompt B:
%s

Output JSON: {"system_prompt": "the combined prompt"}`, agent.SystemPrompt, c.Partner.SystemPrompt))
}

// TargetedOp applies a specific improvement directive.
type TargetedOp struct {
	Directive string
}

func (t TargetedOp) Name() string { return OpTargeted }
func (t TargetedOp) Mutate(ctx context.Context, agent state.AgentDefinition, p provider.Provider) (state.AgentDefinition, error) {
	return mutateWithPrompt(ctx, agent, p, fmt.Sprintf(
		`Improve this system prompt based on the following specific directive:
%s

Original prompt:
%s

Output JSON: {"system_prompt": "the improved prompt"}`, t.Directive, agent.SystemPrompt))
}

// mutateWithPrompt sends a mutation prompt to the provider and extracts the result.
func mutateWithPrompt(ctx context.Context, agent state.AgentDefinition, p provider.Provider, prompt string) (state.AgentDefinition, error) {
	if p == nil {
		return state.AgentDefinition{}, fmt.Errorf("provider is required")
	}

	generated, _, err := p.GenerateAgent(ctx, prompt, nil)
	if err != nil {
		return state.AgentDefinition{}, fmt.Errorf("mutation failed: %w", err)
	}

	newPrompt := strings.TrimSpace(generated.SystemPrompt)
	if newPrompt == "" {
		// Try parsing as JSON in case the provider returned raw JSON
		var parsed struct {
			SystemPrompt string `json:"system_prompt"`
		}
		if jsonErr := json.Unmarshal([]byte(generated.SystemPrompt), &parsed); jsonErr == nil && parsed.SystemPrompt != "" {
			newPrompt = parsed.SystemPrompt
		}
	}

	if newPrompt == "" {
		return state.AgentDefinition{}, fmt.Errorf("mutation produced empty prompt")
	}

	return state.AgentDefinition{
		SystemPrompt: newPrompt,
		Model:        agent.Model,
		Temperature:  agent.Temperature,
		MaxTokens:    agent.MaxTokens,
		Tools:        agent.Tools,
	}, nil
}

// RandomOperator returns a random mutation operator (excluding crossover and targeted).
func RandomOperator(rng *rand.Rand) Operator {
	ops := []Operator{RephraseOp{}, ExpandOp{}, SimplifyOp{}}
	if rng == nil {
		rng = rand.New(rand.NewSource(42))
	}
	return ops[rng.Intn(len(ops))]
}

// NewOperator creates an operator by name.
func NewOperator(name string) (Operator, error) {
	switch name {
	case OpRephrase:
		return RephraseOp{}, nil
	case OpExpand:
		return ExpandOp{}, nil
	case OpSimplify:
		return SimplifyOp{}, nil
	case OpCrossover:
		return CrossoverOp{}, nil
	case OpTargeted:
		return TargetedOp{}, nil
	default:
		return nil, fmt.Errorf("unknown operator %q; choose from: %s", name, strings.Join(AllOperators, ", "))
	}
}
