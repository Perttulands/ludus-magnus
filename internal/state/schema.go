package state

// State is the root JSON document stored at .ludus-magnus/state.json.
type State struct {
	Version  string             `json:"version"`
	Sessions map[string]Session `json:"sessions"`
}

// Session captures one quickstart or training run.
type Session struct {
	ID        string             `json:"id"`
	Mode      string             `json:"mode"`
	Need      string             `json:"need"`
	CreatedAt string             `json:"created_at"`
	Status    string             `json:"status"`
	Lineages  map[string]Lineage `json:"lineages"`
}

// Lineage stores generated agents and their artifacts.
type Lineage struct {
	ID         string     `json:"id"`
	SessionID  string     `json:"session_id"`
	Name       string     `json:"name"`
	Locked     bool       `json:"locked"`
	Agents     []Agent    `json:"agents"`
	Artifacts  []Artifact `json:"artifacts"`
	Directives Directives `json:"directives"`
}

// Agent stores one generated agent definition version.
type Agent struct {
	ID                 string             `json:"id"`
	LineageID          string             `json:"lineage_id"`
	Version            int                `json:"version"`
	Definition         AgentDefinition    `json:"definition"`
	CreatedAt          string             `json:"created_at"`
	GenerationMetadata GenerationMetadata `json:"generation_metadata"`
}

// AgentDefinition is the prompt/model/tools payload used for execution.
type AgentDefinition struct {
	SystemPrompt string  `json:"system_prompt"`
	Model        string  `json:"model"`
	Temperature  float64 `json:"temperature"`
	MaxTokens    int     `json:"max_tokens"`
	Tools        []any   `json:"tools"`
}

// GenerationMetadata tracks generation-level observability.
type GenerationMetadata struct {
	Provider   string  `json:"provider"`
	Model      string  `json:"model"`
	TokensUsed int     `json:"tokens_used"`
	DurationMS int     `json:"duration_ms"`
	CostUSD    float64 `json:"cost_usd"`
}

// Artifact stores one execution result for an agent.
type Artifact struct {
	ID                string            `json:"id"`
	AgentID           string            `json:"agent_id"`
	Input             string            `json:"input"`
	Output            string            `json:"output"`
	CreatedAt         string            `json:"created_at"`
	ExecutionMetadata ExecutionMetadata `json:"execution_metadata"`
	Evaluation        *Evaluation       `json:"evaluation,omitempty"`
}

// ExecutionMetadata tracks runtime signals and tool calls.
type ExecutionMetadata struct {
	Mode            string     `json:"mode"`
	Provider        *string    `json:"provider"`
	Executor        *string    `json:"executor"`
	ExecutorCommand *string    `json:"executor_command"`
	TokensInput     int        `json:"tokens_input"`
	TokensOutput    int        `json:"tokens_output"`
	DurationMS      int        `json:"duration_ms"`
	CostUSD         float64    `json:"cost_usd"`
	ToolCalls       []ToolCall `json:"tool_calls"`
}

// ToolCall captures a single tool invocation made by an agent.
type ToolCall struct {
	Name       string `json:"name"`
	Input      string `json:"input"`
	Output     string `json:"output"`
	DurationMS int    `json:"duration_ms"`
}

// Evaluation is reviewer feedback for one artifact.
type Evaluation struct {
	Score       int    `json:"score"`
	Comment     string `json:"comment"`
	EvaluatedAt string `json:"evaluated_at"`
}

// Directives stores per-lineage one-shot and sticky instructions.
type Directives struct {
	Oneshot []Directive `json:"oneshot"`
	Sticky  []Directive `json:"sticky"`
}

// Directive is one operator instruction.
type Directive struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	CreatedAt string `json:"created_at"`
}

// NewState returns an initialized v1 state document.
func NewState() State {
	return State{
		Version:  "1.0",
		Sessions: map[string]Session{},
	}
}
