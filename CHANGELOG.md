# Changelog

Format: [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)

## [Unreleased]

### Added
- Test harness integration: `internal/harness` package with TestCase, TestSuite, SuiteResult types
- Test types: contains, not_contains, regex, equals with weighted scoring
- NormalizedScore() maps suite results to 1-10 evaluation scale

### Changed
- README: mythology-forward rewrite — each README now reads like discovering a character in a world

## [1.0.1] - 2026-02-19

### Added
- "For Agents" section in README: install, what-this-is, and runtime usage for agent consumers

## [1.0.0] - 2026-02-15

### Added
- Quickstart flow: init session, generate agent, run, evaluate, iterate
- Training flow: four parallel lineages (A/B/C/D) with lock/unlock
- Promotion: convert quickstart to training mode
- Directives: oneshot (cleared after iterate) and sticky (persistent)
- Agent export: JSON, Python, TypeScript formats
- Evidence pack export for session analysis
- Anthropic and OpenAI-compatible provider adapters
- CLI execution mode (claude/codex binaries)
- Observability: token counts, timing, cost calculation per operation
- `--json` output flag on all commands
- Doctor command for environment diagnostics
- State migration framework (v0.9 -> v1.0)
- Integration tests with mock server (quickstart and training flows)

### Changed
- Fixed provider GenerateAgent to pass through engine prompts directly
- Removed SQLite dependency (JSON-only state)
- Removed unused dependencies (viper, lipgloss, charmbracelet/log, tablewriter)

### Removed
- Old academy CLI scaffolding (cmd/academy, internal/cli, internal/store, internal/session, pkg/types)
- Empty placeholder packages (executor, evaluator, evolution, agent)
- Python acceptance test suite (replaced by Go integration tests)
- Development planning docs and log files
