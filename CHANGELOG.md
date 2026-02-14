# Changelog

All notable changes to Ludus Magnus (Agent Academy CLI).

Format: [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)

## [1.0.0] - 2026-02-13

### Added
- 33 features built via ralph loop (codex-medium, 37 iterations)
- **Sprint 1**: Agent definition generation from intent, quickstart initialization flow
- **Sprint 2**: Agent execution engine with provider integration, observability capture (tokens/timing/costs), artifact storage, evaluation commands (score/comment), status/inspect commands, globally unique artifact IDs
- **Sprint 3**: Evolution prompt generation, iterate command, training mode initialization (four lineages A/B/C/D), lock/unlock lineage controls, training iteration, promotion flow (quickstart → training)
- **Sprint 4**: Directive set/clear commands (oneshot/sticky), directive application in evolution, agent export (JSON/Python/TypeScript with tools arrays), evidence pack export, doctor command, JSON output flag for all commands
- **Sprint 5**: Integration tests (quickstart flow, training flow with promotion), CLI usage documentation, state file migration/compaction strategy
- Cobra-based command system with SQLite persistence (pure Go, no CGO)
- Styled terminal output and structured logging
