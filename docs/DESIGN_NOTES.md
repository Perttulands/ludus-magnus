# Design Notes — From Perttu

## Provider Flexibility
- Original AI Training Camp used LiteLLM for provider abstraction
- Agent Academy must support multiple LLM providers flexibly
- Go equivalent: provider adapter layer that supports OpenAI-compatible APIs, Anthropic, and any OpenAI-proxy (LiteLLM, OpenRouter, etc.)
- Config-driven: user specifies provider + model in config, system routes accordingly

## CLI Coding Agents as Executors
- Agent definitions should be executable via CLI coding agents (Claude Code, Codex) not just raw API calls
- This means: generate a system prompt + tools definition → hand it to `claude -p` or `codex` for execution
- The agent being trained might itself be a coding agent that edits files, runs commands, etc.
- Execution mode should be pluggable: direct API call OR CLI agent delegation

## Implication
The executor layer needs two modes:
1. **API mode**: direct LLM call with system prompt + messages (fast, cheap, for simple agents)
2. **CLI mode**: spawn a coding agent with the agent definition as its prompt (powerful, for complex agents that need tools/files)
