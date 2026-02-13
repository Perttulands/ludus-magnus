# Agent Academy CLI Architecture

## Overview

Agent Academy is a Go CLI tool for training AI agents through iterative evaluation loops. It operates entirely locally with no external dependencies beyond LLM provider APIs.

**Core Design Principles:**
1. **Local-first**: All state in local JSON files
2. **Machine-readable**: Every operation supports `--json` output
3. **Deep observability**: Capture everything (tokens, timing, tool calls, costs)
4. **Single binary**: No runtime dependencies, cross-platform
5. **Agent-friendly**: Designed for AI coding agents to use autonomously

## System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        CLI Layer                             │
│  (cobra commands: session, quickstart, training, run, etc.) │
└────────────────┬────────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────────┐
│                     Engine Layer                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐  │
│  │  Generate    │  │  Execute     │  │  Evolve          │  │
│  │  - Agent gen │  │  - Run agent │  │  - Evolution     │  │
│  │  - Prompt    │  │  - Observe   │  │  - Directives    │  │
│  │    templates │  │  - Metadata  │  │  - Feedback      │  │
│  └──────────────┘  └──────────────┘  └──────────────────┘  │
└────────────────┬────────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────────┐
│                   Provider Layer                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Provider Interface                                  │  │
│  │  - GenerateAgent(need, directives) -> AgentDef      │  │
│  │  - ExecuteAgent(agent, input) -> (output, metadata) │  │
│  └──────────────────────────────────────────────────────┘  │
│                           │                                  │
│           ┌───────────────┴───────────────┐                 │
│           ▼                               ▼                 │
│  ┌─────────────────┐           ┌─────────────────┐         │
│  │ AnthropicProvider│           │ Future providers│         │
│  │ - Messages API  │           │ - OpenAI, etc.  │         │
│  │ - Cost calc     │           │                 │         │
│  └─────────────────┘           └─────────────────┘         │
└────────────────┬────────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────────┐
│                     State Layer                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  State Management                                    │  │
│  │  - Load/Save JSON                                    │  │
│  │  - Schema validation                                 │  │
│  │  - Migration framework                               │  │
│  └──────────────────────────────────────────────────────┘  │
│                           │                                  │
│                           ▼                                  │
│              .agent-academy/state.json                      │
│              (local file, JSON format)                      │
└─────────────────────────────────────────────────────────────┘
```

## Package Structure

```
agent-academy/
├── main.go                     # Entry point, wire up cobra
├── cmd/                        # CLI command definitions (cobra)
│   ├── root.go                 # Root command, global flags
│   ├── session.go              # session {new,list,inspect}
│   ├── quickstart.go           # quickstart {init}
│   ├── training.go             # training {init,iterate}
│   ├── run.go                  # run <session-id>
│   ├── evaluate.go             # evaluate <artifact-id>
│   ├── artifact.go             # artifact {list,inspect}
│   ├── lineage.go              # lineage {lock,unlock}
│   ├── directive.go            # directive {set,clear}
│   ├── promote.go              # promote <session-id>
│   ├── iterate.go              # iterate <session-id>
│   ├── export.go               # export {agent,evidence}
│   └── doctor.go               # doctor (diagnostics)
│
├── internal/
│   ├── state/                  # State management
│   │   ├── schema.go           # Data model structs (Session, Lineage, etc.)
│   │   ├── persistence.go      # Load/Save JSON
│   │   ├── migration.go        # Version migration framework
│   │   └── artifact.go         # Artifact management helpers
│   │
│   ├── engine/                 # Core training logic
│   │   ├── generate.go         # Agent definition generation
│   │   ├── execute.go          # Agent execution with observability
│   │   ├── evolve.go           # Evolution prompt construction
│   │   └── observability.go    # Metadata capture (tokens, timing, cost)
│   │
│   ├── provider/               # LLM provider adapters
│   │   ├── interface.go        # Provider interface definition
│   │   ├── anthropic.go        # Anthropic implementation
│   │   └── metadata.go         # Shared metadata types
│   │
│   └── export/                 # Export functionality
│       ├── agent.go            # Agent export (JSON/Python/TS)
│       └── evidence.go         # Evidence pack export
│
├── test/
│   └── integration/            # Integration tests
│       ├── quickstart_test.go  # End-to-end quickstart flow
│       └── training_test.go    # End-to-end training flow
│
├── docs/
│   ├── PRD.md                  # Product requirements (this doc's sibling)
│   ├── ARCHITECTURE.md         # This document
│   └── CLI_USAGE.md            # CLI reference and examples
│
├── go.mod                      # Go module definition
├── go.sum                      # Dependency checksums
└── .gitignore                  # Ignore binaries, state files
```

## Data Model

### Core Entities

```go
// State is the root object persisted to .agent-academy/state.json
type State struct {
    Version  string                `json:"version"`   // Schema version (e.g., "1.0")
    Sessions map[string]*Session   `json:"sessions"`  // Keyed by session ID
}

// Session represents a training session (quickstart or training mode)
type Session struct {
    ID        string                 `json:"id"`         // UUID: ses_abc123
    Mode      string                 `json:"mode"`       // "quickstart" | "training"
    Need      string                 `json:"need"`       // User-provided intent
    CreatedAt time.Time              `json:"created_at"`
    Status    string                 `json:"status"`     // "active" | "closed"
    Lineages  map[string]*Lineage    `json:"lineages"`   // Keyed by lineage name
}

// Lineage is a branch of agent evolution
type Lineage struct {
    ID         string              `json:"id"`          // UUID: lin_xyz789
    SessionID  string              `json:"session_id"`
    Name       string              `json:"name"`        // "main" | "A" | "B" | "C" | "D"
    Locked     bool                `json:"locked"`      // Training mode lock
    Agents     []*AgentVersion     `json:"agents"`      // Version history
    Artifacts  []*Artifact         `json:"artifacts"`   // Execution results
    Directives *DirectiveSet       `json:"directives"`  // Guidance
}

// AgentVersion is a versioned agent definition
type AgentVersion struct {
    ID                  string               `json:"id"`           // UUID: agt_def456
    LineageID           string               `json:"lineage_id"`
    Version             int                  `json:"version"`      // 1, 2, 3, ...
    Definition          *AgentDefinition     `json:"definition"`
    CreatedAt           time.Time            `json:"created_at"`
    GenerationMetadata  *GenerationMetadata  `json:"generation_metadata"`
}

// AgentDefinition is the executable agent config
type AgentDefinition struct {
    SystemPrompt string   `json:"system_prompt"`
    Model        string   `json:"model"`         // "claude-sonnet-4-5"
    Temperature  float64  `json:"temperature"`   // 1.0
    MaxTokens    int      `json:"max_tokens"`    // 4096
    Tools        []Tool   `json:"tools"`         // Empty in v1
}

// Artifact is the result of one agent execution
type Artifact struct {
    ID                string               `json:"id"`           // UUID: art_ghi789
    AgentID           string               `json:"agent_id"`
    Input             string               `json:"input"`
    Output            string               `json:"output"`
    CreatedAt         time.Time            `json:"created_at"`
    ExecutionMetadata *ExecutionMetadata   `json:"execution_metadata"`
    Evaluation        *Evaluation          `json:"evaluation,omitempty"`
}

// Evaluation is user feedback on an artifact
type Evaluation struct {
    Score       int       `json:"score"`        // 1-10
    Comment     string    `json:"comment"`
    EvaluatedAt time.Time `json:"evaluated_at"`
}

// ExecutionMetadata captures observability data
type ExecutionMetadata struct {
    TokensInput  int         `json:"tokens_input"`
    TokensOutput int         `json:"tokens_output"`
    DurationMs   int64       `json:"duration_ms"`
    CostUSD      float64     `json:"cost_usd"`
    ToolCalls    []ToolCall  `json:"tool_calls"`
}

// ToolCall records a single tool invocation
type ToolCall struct {
    Name       string `json:"name"`
    Input      string `json:"input"`       // JSON string
    Output     string `json:"output"`      // JSON string
    DurationMs int64  `json:"duration_ms"`
}

// DirectiveSet contains guidance for evolution
type DirectiveSet struct {
    Oneshot []Directive `json:"oneshot"`  // Cleared after use
    Sticky  []Directive `json:"sticky"`   // Persist across iterations
}

// Directive is one piece of guidance
type Directive struct {
    ID        string    `json:"id"`         // UUID: dir_jkl012
    Text      string    `json:"text"`
    CreatedAt time.Time `json:"created_at"`
}

// GenerationMetadata captures agent generation cost
type GenerationMetadata struct {
    TokensUsed int     `json:"tokens_used"`
    DurationMs int64   `json:"duration_ms"`
    CostUSD    float64 `json:"cost_usd"`
}
```

## State Management

### Persistence Strategy

**Why JSON over SQLite:**
- Simplicity: No dependencies, easy to inspect/debug
- Portability: Works everywhere Go works
- Human-readable: Can manually inspect/edit if needed
- Version control: Can commit state.json to git for reproducibility
- Agent-friendly: Easy for AI agents to parse and understand

**File Location:**
- `.agent-academy/state.json` in current working directory
- Created on first operation if doesn't exist
- Directory is `.gitignore`d by default (user can choose to commit)

**Load/Save Operations:**
```go
// Load reads state.json and unmarshals to State struct
func LoadState(path string) (*State, error) {
    // Read file
    // Unmarshal JSON
    // Run migrations if version mismatch
    // Return State
}

// Save marshals State to JSON and writes to state.json
func SaveState(path string, state *State) error {
    // Marshal with indent (pretty JSON)
    // Write atomically (write to temp, rename)
    // Return error if failed
}
```

**Migration Framework:**
```go
// Migrations map old version -> migration function
var migrations = map[string]func(*State) error{
    "0.9": migrateFrom0_9,
    // Future versions add entries here
}

func migrateFrom0_9(state *State) error {
    // Update schema from 0.9 to 1.0
    // Add new fields with defaults
    // Return error if migration fails
}
```

### State File Size Management

**Growth Characteristics:**
- Session: ~500 bytes
- Lineage: ~200 bytes
- Agent: ~2-10KB (depends on prompt size)
- Artifact: ~2-50KB (depends on input/output size)
- Tool call: ~200-2000 bytes

**Estimated Size:**
- 10 sessions × 4 lineages × 5 agents × 10 artifacts ≈ 10-50MB
- 100 sessions ≈ 100-500MB

**Compaction Strategy (v2):**
- Archive old sessions to separate files
- Remove artifacts older than retention period
- Compress archived state with gzip
- User-triggered: `agent-academy compact --older-than 30d`

## Provider Adapter Layer

### Interface Design

```go
// Provider abstracts LLM provider operations
type Provider interface {
    // GenerateAgent creates a new agent definition from intent
    GenerateAgent(ctx context.Context, req GenerateAgentRequest) (*AgentDefinition, *GenerationMetadata, error)

    // ExecuteAgent runs an agent definition against input
    ExecuteAgent(ctx context.Context, req ExecuteAgentRequest) (*ExecuteAgentResponse, error)

    // GetName returns provider name for logging/diagnostics
    GetName() string
}

// GenerateAgentRequest encapsulates agent generation inputs
type GenerateAgentRequest struct {
    Need       string   // User intent
    Directives []string // Guidance strings
    Model      string   // Requested model (optional)
}

// ExecuteAgentRequest encapsulates execution inputs
type ExecuteAgentRequest struct {
    Agent AgentDefinition
    Input string
}

// ExecuteAgentResponse contains output and metadata
type ExecuteAgentResponse struct {
    Output   string
    Metadata ExecutionMetadata
}
```

### Anthropic Implementation

```go
type AnthropicProvider struct {
    apiKey     string
    httpClient *http.Client
}

func NewAnthropicProvider() (*AnthropicProvider, error) {
    apiKey := os.Getenv("ANTHROPIC_API_KEY")
    if apiKey == "" {
        return nil, errors.New("ANTHROPIC_API_KEY not set")
    }
    return &AnthropicProvider{
        apiKey:     apiKey,
        httpClient: &http.Client{Timeout: 60 * time.Second},
    }, nil
}

func (p *AnthropicProvider) GenerateAgent(ctx context.Context, req GenerateAgentRequest) (*AgentDefinition, *GenerationMetadata, error) {
    // Construct prompt from template + need + directives
    prompt := buildGenerationPrompt(req)

    // Call Messages API
    startTime := time.Now()
    resp, err := p.callMessagesAPI(ctx, prompt)
    if err != nil {
        return nil, nil, err
    }
    duration := time.Since(startTime)

    // Parse response to extract system prompt
    systemPrompt := extractSystemPrompt(resp.Content)

    // Calculate cost
    cost := calculateCost(resp.Usage.InputTokens, resp.Usage.OutputTokens, "claude-sonnet-4-5")

    // Build agent definition
    agent := &AgentDefinition{
        SystemPrompt: systemPrompt,
        Model:        "claude-sonnet-4-5",
        Temperature:  1.0,
        MaxTokens:    4096,
        Tools:        []Tool{}, // Empty in v1
    }

    metadata := &GenerationMetadata{
        TokensUsed: resp.Usage.InputTokens + resp.Usage.OutputTokens,
        DurationMs: duration.Milliseconds(),
        CostUSD:    cost,
    }

    return agent, metadata, nil
}

func (p *AnthropicProvider) ExecuteAgent(ctx context.Context, req ExecuteAgentRequest) (*ExecuteAgentResponse, error) {
    // Call Messages API with agent's system prompt and user input
    startTime := time.Now()
    resp, err := p.callMessagesAPI(ctx, req.Agent.SystemPrompt, req.Input)
    if err != nil {
        return nil, err
    }
    duration := time.Since(startTime)

    // Extract tool calls if any
    toolCalls := extractToolCalls(resp.Content)

    // Calculate cost
    cost := calculateCost(resp.Usage.InputTokens, resp.Usage.OutputTokens, req.Agent.Model)

    // Build response
    return &ExecuteAgentResponse{
        Output: resp.Content[0].Text,
        Metadata: ExecutionMetadata{
            TokensInput:  resp.Usage.InputTokens,
            TokensOutput: resp.Usage.OutputTokens,
            DurationMs:   duration.Milliseconds(),
            CostUSD:      cost,
            ToolCalls:    toolCalls,
        },
    }, nil
}

// Cost calculation based on 2026 Anthropic pricing
func calculateCost(inputTokens, outputTokens int, model string) float64 {
    pricing := map[string]struct{ input, output float64 }{
        "claude-sonnet-4-5": {3.0, 15.0},   // $ per MTok
        "claude-opus-4-6":   {15.0, 75.0},
        "claude-haiku-4-5":  {0.80, 4.0},
    }

    p, ok := pricing[model]
    if !ok {
        p = pricing["claude-sonnet-4-5"] // Default
    }

    return (float64(inputTokens) * p.input / 1_000_000) +
           (float64(outputTokens) * p.output / 1_000_000)
}
```

### Future Providers

To add OpenAI, Cohere, etc:
1. Implement `Provider` interface
2. Add provider-specific pricing table
3. Add factory function: `NewOpenAIProvider()`
4. Update doctor command to check provider-specific env vars

## Generation/Evolution Logic

### Agent Generation Prompt Template

```go
const generationTemplate = `You are a master AI agent trainer. Generate a high-quality system prompt for an AI agent.

User Need: {{.Need}}

{{if .Directives}}
Directives (constraints/guidance):
{{range .Directives}}
- {{.}}
{{end}}
{{end}}

Output a JSON object with the following structure:
{
  "system_prompt": "the complete system prompt for the agent",
  "reasoning": "brief explanation of your design choices"
}

Focus on clarity, specificity, and task alignment. The agent will use Claude Sonnet 4.5.`
```

### Evolution Prompt Template

```go
const evolutionTemplate = `You are a master AI agent trainer. Improve the following agent based on evaluation feedback.

CURRENT AGENT (version {{.Version}}):
System Prompt: {{.CurrentSystemPrompt}}

EVALUATION SUMMARY:
- Total artifacts: {{.ArtifactCount}}
- Average score: {{.AvgScore}}/10
- Score distribution: {{.ScoreHistogram}}

FEEDBACK:
{{range .Feedback}}
- [{{.Score}}/10] {{.Comment}}
{{end}}

{{if .Directives}}
DIRECTIVES:
{{range .Directives}}
- {{.Text}}
{{end}}
{{end}}

Output a JSON object with the following structure:
{
  "system_prompt": "the improved system prompt",
  "reasoning": "brief explanation of changes made"
}

Focus on addressing low-scoring feedback while preserving high-scoring behaviors.`
```

### Evolution Logic

```go
func GenerateEvolutionPrompt(lineage *Lineage) (string, error) {
    // Get latest agent
    latestAgent := lineage.Agents[len(lineage.Agents)-1]

    // Collect evaluated artifacts
    evaluatedArtifacts := filterEvaluated(lineage.Artifacts)
    if len(evaluatedArtifacts) == 0 {
        return "", errors.New("no evaluated artifacts to evolve from")
    }

    // Calculate summary stats
    avgScore := calculateAvgScore(evaluatedArtifacts)
    scoreHist := buildScoreHistogram(evaluatedArtifacts)

    // Extract feedback
    feedback := make([]struct{ Score int; Comment string }, 0)
    for _, art := range evaluatedArtifacts {
        if art.Evaluation != nil && art.Evaluation.Comment != "" {
            feedback = append(feedback, struct{ Score int; Comment string }{
                Score:   art.Evaluation.Score,
                Comment: art.Evaluation.Comment,
            })
        }
    }

    // Collect directives (oneshot + sticky)
    directives := append(lineage.Directives.Oneshot, lineage.Directives.Sticky...)

    // Render template
    data := struct {
        Version             int
        CurrentSystemPrompt string
        ArtifactCount       int
        AvgScore            float64
        ScoreHistogram      string
        Feedback            []struct{ Score int; Comment string }
        Directives          []Directive
    }{
        Version:             latestAgent.Version,
        CurrentSystemPrompt: latestAgent.Definition.SystemPrompt,
        ArtifactCount:       len(evaluatedArtifacts),
        AvgScore:            avgScore,
        ScoreHistogram:      scoreHist,
        Feedback:            feedback,
        Directives:          directives,
    }

    return renderTemplate(evolutionTemplate, data)
}

// After evolution, clear oneshot directives
func applyEvolution(lineage *Lineage, newAgent *AgentVersion) {
    lineage.Agents = append(lineage.Agents, newAgent)
    lineage.Directives.Oneshot = []Directive{} // Clear oneshot
    // Sticky directives remain
}
```

## CLI Command Flow Examples

### Quickstart Flow

```
$ agent-academy quickstart init --need "customer care agent"
Created session: ses_abc123
Created lineage: main (lin_xyz789)
Generated agent: agt_def456 (version 1)

$ agent-academy run ses_abc123 --input "My order is late"
Executing agent agt_def456...
Created artifact: art_ghi789
Output: I apologize for the delay. Let me check your order status...

$ agent-academy evaluate art_ghi789 --score 6 --comment "good but tone too formal"
Evaluated artifact art_ghi789: 6/10

$ agent-academy iterate ses_abc123
Generating evolution for lineage main...
Generated agent: agt_jkl012 (version 2)

$ agent-academy run ses_abc123 --input "My order is late"
Executing agent agt_jkl012...
Created artifact: art_mno345
Output: Hey! Sorry your order's running late. Let me look that up for you...
```

### Training Flow

```
$ agent-academy training init --need "customer care agent"
Created session: ses_abc123
Created lineages: A, B, C, D
Generated 4 agent variants

$ agent-academy run ses_abc123 --lineage A --input "My order is late"
$ agent-academy run ses_abc123 --lineage B --input "My order is late"
$ agent-academy run ses_abc123 --lineage C --input "My order is late"
$ agent-academy run ses_abc123 --lineage D --input "My order is late"

$ agent-academy artifact list ses_abc123
ID          Lineage  Version  Score  Created
art_123     A        1        9      2026-02-13T10:30:00Z
art_456     B        1        7      2026-02-13T10:31:00Z
art_789     C        1        5      2026-02-13T10:32:00Z
art_012     D        1        8      2026-02-13T10:33:00Z

$ agent-academy lineage lock ses_abc123 A
$ agent-academy lineage lock ses_abc123 D

$ agent-academy training iterate ses_abc123
Regenerated 2 lineages: B, C
Locked: A, D

$ agent-academy export agent agt_A_v1 --format typescript > winner.ts
```

## Observability Strategy

### What We Capture

**Per Agent Generation:**
- Tokens used (for cost tracking)
- Duration (for latency analysis)
- Cost in USD (for budget monitoring)

**Per Agent Execution:**
- Input tokens (for cost/latency analysis)
- Output tokens (for cost/latency analysis)
- Total duration (for performance monitoring)
- Cost in USD (for budget tracking)
- Tool calls (name, input, output, duration) - for understanding agent behavior

**Per Evaluation:**
- Score (1-10, for preference learning)
- Comment (freeform feedback, for qualitative analysis)
- Timestamp (for temporal analysis)

### Why This Data Matters

**Training Signal:**
- Scores + comments → preference pairs for RLHF
- Input/output → demonstrations for supervised fine-tuning
- Tool calls → behavior analysis and debugging
- Cost/latency → performance optimization targets

**Offline Analysis:**
- Export evidence pack → analyze in Jupyter/pandas
- Track convergence over iterations
- Identify high-cost or slow operations
- Build datasets for model training

**Operational Visibility:**
- Monitor spending per session
- Detect performance regressions
- Audit tool usage patterns

### No External Telemetry

All observability data stays local:
- No phone-home to external services
- No background telemetry collection
- User controls all data (can delete state.json)
- Export only when user explicitly requests

## Error Handling Strategy

### Principles

1. **Fail fast, fail loud**: Don't hide errors, report them clearly
2. **Actionable messages**: Tell user what went wrong and how to fix
3. **No silent fallbacks**: Don't guess what user meant
4. **Preserve state**: Always save state before operations that can fail

### Error Categories

**Configuration Errors (Exit 1):**
- Missing ANTHROPIC_API_KEY → "Error: ANTHROPIC_API_KEY not set. Run: export ANTHROPIC_API_KEY=sk-ant-..."
- Invalid state.json → "Error: state file corrupted. Backup at .agent-academy/state.json.bak"

**Usage Errors (Exit 2):**
- Missing required flag → "Error: --need flag required. Usage: agent-academy quickstart init --need 'intent'"
- Invalid ID → "Error: session not found: ses_abc123. Run 'agent-academy session list' to see sessions."

**Provider Errors (Exit 3):**
- API failure → "Error: Anthropic API request failed: {error}. Check your API key and network."
- Rate limit → "Error: Rate limited. Wait 60s and retry."

**State Errors (Exit 4):**
- File permission → "Error: Cannot write state file. Check permissions on .agent-academy/"
- Disk full → "Error: Disk full. Free up space and retry."

### Example Error Messages

```
# Good: Specific, actionable
Error: ANTHROPIC_API_KEY not set
Fix: export ANTHROPIC_API_KEY=sk-ant-your-key-here

# Bad: Vague, unhelpful
Error: Configuration error
```

## Testing Strategy

### Unit Tests

**Coverage Target:** 80%+ for core packages

**Focus Areas:**
- State load/save round-trip
- Cost calculation accuracy
- Evolution prompt generation
- Directive application logic
- Migration framework

**Example:**
```go
func TestCostCalculation(t *testing.T) {
    cost := calculateCost(1000, 5000, "claude-sonnet-4-5")
    expected := (1000 * 3.0 / 1_000_000) + (5000 * 15.0 / 1_000_000)
    if cost != expected {
        t.Errorf("got %f, want %f", cost, expected)
    }
}
```

### Integration Tests

**Scope:** End-to-end workflows with mock provider

**Test Cases:**
1. Quickstart flow: init → run → evaluate → iterate
2. Training flow: init → run all lineages → evaluate → lock → iterate
3. Promotion flow: quickstart → promote → training iterate
4. Export flow: run → evaluate → export agent + evidence

**Mock Provider:**
```go
type MockProvider struct {
    GenerateFunc func(req GenerateAgentRequest) (*AgentDefinition, *GenerationMetadata, error)
    ExecuteFunc  func(req ExecuteAgentRequest) (*ExecuteAgentResponse, error)
}
```

### Manual Testing

**Pre-release Checklist:**
- [ ] Build on Linux/macOS/Windows
- [ ] Run doctor command (all checks pass)
- [ ] Complete quickstart workflow (real Anthropic API)
- [ ] Complete training workflow (real Anthropic API)
- [ ] Export agent definitions (JSON/Python/TypeScript)
- [ ] Export evidence pack, verify structure
- [ ] Verify --json output for all commands
- [ ] Test with invalid/missing API key
- [ ] Test with corrupted state.json

## Build and Deployment

### Build Process

```bash
# Build for current platform
go build -o agent-academy

# Build for all platforms
GOOS=linux GOARCH=amd64 go build -o agent-academy-linux-amd64
GOOS=darwin GOARCH=amd64 go build -o agent-academy-darwin-amd64
GOOS=darwin GOARCH=arm64 go build -o agent-academy-darwin-arm64
GOOS=windows GOARCH=amd64 go build -o agent-academy-windows-amd64.exe
```

### Distribution

**Method:** GitHub Releases
- Attach binaries for each platform
- Include SHA256 checksums
- Tag releases: `v1.0.0`

**Installation (recommended):**
```bash
# Download from GitHub Releases
curl -L https://github.com/user/agent-academy/releases/download/v1.0.0/agent-academy-linux-amd64 -o agent-academy

# Make executable
chmod +x agent-academy

# Move to PATH
sudo mv agent-academy /usr/local/bin/
```

### Versioning

**Semantic Versioning:** MAJOR.MINOR.PATCH
- MAJOR: Breaking state schema changes
- MINOR: New features, backward-compatible
- PATCH: Bug fixes

**State Schema Versioning:**
- Maintain migration path for all schema versions
- Test migrations on real state files before release
- Document breaking changes in CHANGELOG

## Future Architecture Considerations (v2+)

### Multi-Input Test Suites

**Current (v1):** Single input per execution
**Future (v2):** Test suite with multiple inputs

```go
type TestSuite struct {
    ID     string
    Name   string
    Inputs []TestInput
}

type TestInput struct {
    ID          string
    Description string
    Input       string
    Expected    string  // Optional expected output
}
```

**Workflow:**
```bash
agent-academy suite create --name "customer-care-tests"
agent-academy suite add <suite-id> --input "My order is late" --expected "empathetic, actionable"
agent-academy run <session-id> --suite <suite-id>
# Runs agent on all inputs in suite, stores artifacts
```

### Tool Definition Support

**Current (v1):** Empty tools array
**Future (v2):** Define custom tools

```go
type Tool struct {
    Name        string
    Description string
    InputSchema map[string]interface{}
}
```

**Workflow:**
```bash
agent-academy tool define --name get_order_status --schema order-schema.json
agent-academy quickstart init --need "customer care" --tools get_order_status,get_customer_info
```

### Parallel Execution

**Current (v1):** Sequential execution
**Future (v2):** Parallel execution across lineages

```bash
agent-academy training run <session-id> --all-lineages --parallel
# Runs all 4 lineages in parallel, stores artifacts
```

### State Backends

**Current (v1):** JSON file only
**Future (v2):** Pluggable backends

```go
type StateBackend interface {
    Load() (*State, error)
    Save(*State) error
}

// Implementations:
type JSONBackend struct { ... }
type SQLiteBackend struct { ... }
type PostgreSQLBackend struct { ... }  // For shared state
```

### Automatic Iteration

**Current (v1):** Manual iteration
**Future (v2):** Auto-iterate until convergence

```bash
agent-academy auto-train <session-id> \
    --max-iterations 10 \
    --convergence-threshold 8.5 \
    --suite <suite-id>

# Runs: execute → evaluate (auto-score or human) → iterate → repeat
# Stops when avg score >= 8.5 or max iterations reached
```

## Appendix: Key Design Decisions

### Why Go?

- **Single binary**: No runtime dependencies (unlike Python)
- **Fast**: Compilation and execution both fast
- **Cross-platform**: Easy to build for Linux/macOS/Windows
- **Agent-friendly**: AI agents can just run the binary, no setup
- **Strong typing**: Catches errors at compile time
- **Great CLI libs**: cobra, viper, etc.

### Why JSON over SQLite?

- **Simplicity**: No database driver dependencies
- **Portability**: Works everywhere, no native extensions
- **Human-readable**: Easy to inspect and debug
- **Version control**: Can commit state to git
- **Agent-friendly**: Easy for AI agents to parse
- **Trade-off**: Performance degrades at very large scale (100s of sessions), but acceptable for v1

### Why Local-First?

- **Privacy**: User data never leaves their machine
- **Simplicity**: No server infrastructure needed
- **Reliability**: Works offline
- **Cost**: No hosting fees
- **Control**: User owns their data completely

### Why Anthropic-Only (v1)?

- **Focus**: Nail one provider well before supporting many
- **Complexity**: Each provider has different APIs, pricing, features
- **Use case**: Primary users likely use Claude already
- **Extensibility**: Provider interface makes adding others straightforward in v2

---

## Summary

Agent Academy CLI is a focused, local-first tool for training AI agents through iterative evaluation. The architecture emphasizes:

1. **Simplicity**: JSON state, single binary, minimal dependencies
2. **Observability**: Capture everything for training signal
3. **Extensibility**: Provider interface, migration framework, pluggable backends (future)
4. **Agent-friendliness**: Machine-readable output, clear semantics, autonomous operation

The core value loop (define → generate → execute → evaluate → evolve) is implemented cleanly across three main layers: CLI commands (user interface), Engine logic (training semantics), Provider adapters (LLM integration), and State persistence (durability).

This architecture supports the v1 use case (training customer care agents and similar agentic workflows) while leaving room for future enhancements (test suites, tools, parallel execution, alternative backends).
