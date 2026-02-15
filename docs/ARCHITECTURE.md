# Ludus Magnus Architecture

## Overview

Ludus Magnus is a Go CLI tool for training AI agents through iterative evaluation loops. It operates entirely locally with no external dependencies beyond LLM provider APIs.

**Design principles:**
1. **Local-first**: All state in a single JSON file
2. **Machine-readable**: Every command supports `--json` output
3. **Observable**: Capture tokens, timing, costs, tool calls
4. **Single binary**: No runtime dependencies, cross-platform
5. **Agent-friendly**: Designed for AI coding agents to use autonomously

## Layers

```
CLI Layer (cobra commands)
    |
Engine Layer (generate, execute, evolve)
    |
Provider Layer (anthropic, openai-compatible)
    |
State Layer (JSON persistence)
```

### CLI Layer (`cmd/`)

Cobra commands that parse flags, load state, call engine/provider, save state, print output. Every command supports `--json` for machine-readable output.

### Engine Layer (`internal/engine/`)

- **generate.go** - Builds generation prompts from user intent + directives, calls provider, returns agent definition
- **execute.go** - Runs agents via API or CLI mode (claude/codex), captures output
- **evolve.go** - Synthesizes evaluation feedback into evolution prompts for next agent version
- **observability.go** - Token counting, cost calculation, metadata capture

### Provider Layer (`internal/provider/`)

- **interface.go** - `Provider` interface: `GenerateAgent`, `ExecuteAgent`, `GetMetadata`
- **factory.go** - Creates provider from config (env vars + CLI flags)
- **anthropic.go** - Anthropic Messages API adapter
- **openai_compatible.go** - OpenAI chat completions adapter (works with OpenAI, LiteLLM, OpenRouter)

### State Layer (`internal/state/`)

- **schema.go** - Data structures: State, Session, Lineage, Agent, Artifact, Evaluation, Directive
- **persistence.go** - Load/Save JSON at `.ludus-magnus/state.json`
- **migration.go** - Schema version migration framework (v0.9 -> v1.0)
- **artifact.go** - Collision-safe artifact ID generation
- **artifact_lookup.go** - Global artifact lookup across sessions
- **evaluation.go** - Immutable evaluation storage (score 1-10)

### Export Layer (`internal/export/`)

- **agent.go** - Export agent definitions as JSON, Python, or TypeScript
- **evidence.go** - Export session evidence packs for analysis

## Data Model

```
State (v1.0)
  sessions: map[session_id]
    Session
      mode: "quickstart" | "training"
      need: string
      status: "active" | "closed"
      lineages: map[lineage_id]
        Lineage
          name: "main" | "A" | "B" | "C" | "D"
          locked: bool
          agents: []Agent
            version: int (1, 2, 3...)
            definition: AgentDefinition
              system_prompt, model, temperature, max_tokens, tools
            generation_metadata: tokens, duration, cost
          artifacts: []Artifact
            input, output
            execution_metadata: mode, tokens, duration, cost, tool_calls
            evaluation: score (1-10), comment (immutable once set)
          directives:
            oneshot: [] (cleared after iterate)
            sticky: [] (preserved across iterations)
```

## Key Flows

**Generate**: Engine builds prompt from intent + directives -> Provider calls LLM -> Returns AgentDefinition

**Execute**: Engine sends agent's system_prompt + user input to provider (API mode) or spawns claude/codex binary (CLI mode) -> Returns output + metadata

**Evolve**: Engine collects all evaluated artifacts + directives -> Builds evolution prompt with score histogram, feedback patterns, current prompt -> Provider generates improved prompt -> New agent version created, oneshot directives cleared

## State Management

- JSON file at `.ludus-magnus/state.json` relative to working directory
- Auto-created on first save
- Pretty-printed with 2-space indent
- Migration framework for schema upgrades
- Artifact IDs are globally unique (UUID-based, collision-checked)
- Evaluations are immutable (cannot re-score)

## Testing

- **Unit tests**: `internal/*/` packages cover state, engine, provider, export
- **Integration tests**: `test/integration/` builds the binary, runs against mock HTTP server, verifies full quickstart and training flows end-to-end
