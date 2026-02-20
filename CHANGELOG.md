# Changelog

Format: [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)

## [Unreleased]

### Added
- Test harness integration: `internal/harness` package with TestCase, TestSuite, SuiteResult types
- Test types: contains, not_contains, regex, equals with weighted scoring
- NormalizedScore() maps suite results to 1-10 evaluation scale
- Truthsayer integration: `internal/truthsayer` package wrapping truthsayer binary
- Scan files/directories/strings with JSON output parsing
- QualityScore() converts findings to 1-10 scale (errors=-2, warnings=-1)
- Scoring pipeline: `internal/scoring` package combining harness, truthsayer, manual, and efficiency scores
- Configurable weights (default: harness 35%, truthsayer 25%, manual 30%, efficiency 10%)
- Composite scoring with weighted average and normalized 1-10 output
- Challenge schema: `internal/challenge` package with Challenge and ChallengeSet types
- Four challenge types: feature, bugfix, refactor, review with difficulty levels
- Challenge validation and integrated test suite support
- Challenge generator: LLM-powered synthetic challenge creation with Generate and GenerateBatch
- Tournament runner: `internal/tournament` package with Bout, Round, RunBout, RunRound, RunAll
- Contestant abstraction wrapping agents for competition
- Tournament orchestrator: lifecycle management with New, Run, Winner, TopN, standings computation
- Selection and evolution: `internal/selection` with truncation, tournament, and elitist strategies
- Partition function to split standings into winners and eliminated
- Training loop runner: `internal/training` package with Loop, Generation, Config types
- RunGeneration with auto-termination by max generations or target score
- Mutation operator library: `internal/mutation` with rephrase, expand, simplify, crossover, targeted operators
- Operator interface, RandomOperator, NewOperator factory
- Checkpoint and resume: `internal/checkpoint` with Save, Load, Exists, Remove, List
- JSON-based checkpoint persistence for training loop state
- Learning Loop integration: `internal/learningloop` with TrainedPrompt and TrainingReport
- ExportReport extracts winner prompts from completed training loops
- WriteReport/ReadReport for JSON persistence to state/trained-prompts/
- Cost tracking: `internal/cost` with Tracker, Event, Summary types
- Budget enforcement with CheckBudget, OverBudget, Remaining
- Cost aggregation by operation and model
- Production dispatch integration: `internal/dispatch` with PromptManifest and DeployedPrompt
- GenerateManifest converts training reports to dispatch-ready format
- BestPrompt extracts highest-scoring prompt from manifest

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
