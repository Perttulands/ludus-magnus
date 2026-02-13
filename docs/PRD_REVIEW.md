# PRD + Architecture Review

## Scope
Reviewed `docs/PRD.md`, `docs/ARCHITECTURE.md`, `docs/DESIGN_NOTES.md`, and current Go scaffold (`internal/*`, `cmd/*`, `go.mod`, `README.md`).

## Check Results
1. Are all 43 tasks implementable from PRD alone without ambiguity?
- No. The PRD actually defines 33 tasks, not 43. I fixed the task summary and dependency graph (`docs/PRD.md:21`, `docs/PRD.md:29`).

2. Does the data model cover everything needed?
- Partially. Core session/lineage/agent/artifact/evaluation is present, but reproducibility/history gaps remain (see Important issues).

3. Is the CLI command spec complete (all flags, all outputs)?
- No. It is still incomplete for full machine-contract use; only partial JSON schemas are documented (`docs/PRD.md:1240`, `docs/PRD.md:1254`).

4. Does the architecture support both API mode and CLI coding agent mode for execution?
- No. `ARCHITECTURE.md` still describes only provider API execution and no executor-mode abstraction (`docs/ARCHITECTURE.md:24`, `docs/ARCHITECTURE.md:35`, `docs/ARCHITECTURE.md:304`).

5. Is the provider adapter layer flexible enough for OpenAI-compatible APIs, Anthropic, LiteLLM?
- Partially in PRD (fixed), but not in architecture: architecture still positions multi-provider as future and v1 as Anthropic-only (`docs/ARCHITECTURE.md:437`, `docs/ARCHITECTURE.md:929`).

6. Are acceptance criteria testable?
- Partially. Many are testable, but several require real external API calls and unstable pricing assumptions (`docs/PRD.md:457`, `docs/PRD.md:571`, `docs/PRD.md:1496`).

7. Is the task ordering and dependency chain correct?
- Mostly after fixes, but some dependency intent is still implicit rather than explicit per story.

8. Any missing edge cases or error handling specs?
- Yes. Key edge/error cases are still underspecified (see Important issues).

## Issues Found

### Blocking
- Task inventory/dependency inconsistency in PRD.
  - Found: PRD said 43 tasks with an outdated graph; story list had 33 tasks.
  - Fixed in PRD: `docs/PRD.md:21`, `docs/PRD.md:29`.

- Design-notes requirements missing from PRD (provider flexibility + API/CLI executor modes).
  - Found: PRD was Anthropic-only and API-only, conflicting with `docs/DESIGN_NOTES.md:5` and `docs/DESIGN_NOTES.md:16`.
  - Fixed in PRD goals/stories: `docs/PRD.md:12`, `docs/PRD.md:344`, `docs/PRD.md:509`, `docs/PRD.md:1185`, `docs/PRD.md:1579`.

- Training execution path ambiguity.
  - Found: `run` command lacked lineage/mode flags while training examples required lineage-specific runs.
  - Fixed in PRD: `docs/PRD.md:516`.

- Iteration contradiction with “scoring not mandatory”.
  - Found: evolution logic depended only on evaluated artifacts while open questions allowed no-scoring iteration.
  - Fixed in PRD: `docs/PRD.md:732`, `docs/PRD.md:777`.

### Important
- CLI contract still incomplete for implementation from PRD alone.
  - Missing full per-command flag matrix (required/optional/defaults/allowed values), full JSON response schemas, and canonical error JSON schema.
  - Refs: `docs/PRD.md:1220`, `docs/PRD.md:1240`.

- Data model still lacks explicit immutable directive history, but evidence export requires “directives history”.
  - Refs: `docs/PRD.md:279` (schema), `docs/PRD.md:1131` (export requirement).

- Reproducibility metadata is still incomplete for CLI execution mode.
  - `execution_metadata` now includes mode/provider/executor command, but there is no explicit requirement for executor exit code/stderr capture.
  - Ref: `docs/PRD.md:264`.

- Acceptance criteria mix deterministic and non-deterministic checks.
  - Real API dependency and dynamic cost/usage outputs weaken CI reliability.
  - Refs: `docs/PRD.md:457`, `docs/PRD.md:571`.

- Architecture doc is now behind PRD and design notes.
  - It does not define API-vs-CLI executor layering and still frames Anthropic-only v1.
  - Refs: `docs/ARCHITECTURE.md:304`, `docs/ARCHITECTURE.md:929`.

### Minor
- Naming/packaging drift remains between planning docs and real scaffold naming conventions (`agent-academy` vs `academy`).
  - Refs: `docs/ARCHITECTURE.md:70`, `internal/cli/root.go:33`.

- Some acceptance criteria depend on exact human-readable text formatting that may change.
  - Refs: `docs/PRD.md:1208`, `docs/PRD.md:1292`.

## Existing Go Scaffold vs Architecture Doc
Mismatch is substantial.

- Persistence backend mismatch.
  - Architecture/PRD: JSON state file (`docs/ARCHITECTURE.md:56`, `docs/PRD.md:199`).
  - Scaffold: SQLite store (`go.mod:12`, `internal/store/store.go:13`, `README.md:8`).

- Command surface mismatch.
  - Architecture expects many commands (`docs/ARCHITECTURE.md:72`).
  - Scaffold currently exposes only `version`, `doctor`, `session new/list` (`internal/cli/root.go:44`, `internal/cli/session.go:22`).

- Binary/entrypoint/layout mismatch.
  - Architecture expects `main.go` + `cmd/` layout (`docs/ARCHITECTURE.md:71`).
  - Scaffold uses `cmd/academy/main.go` + `internal/cli` (`cmd/academy/main.go:1`, `internal/cli/root.go:1`).

- Runtime behavior mismatch.
  - Architecture expects provider/engine/state layers in active use (`docs/ARCHITECTURE.md:94`, `docs/ARCHITECTURE.md:100`).
  - Scaffold provider/evolution/executor/export packages are placeholders only (`internal/provider/doc.go`, `internal/evolution/doc.go`, `internal/executor/doc.go`, `internal/export/doc.go`).

## Suggestions for Improvement
- Add a canonical CLI contract section in PRD with complete per-command flag specs and JSON output schemas, including error payload schema and exit codes.
- Extend the data model with immutable directive event history and explicit executor result fields (`exit_code`, `stderr`, `started_at`, `finished_at`).
- Update `ARCHITECTURE.md` to match the revised PRD: add executor abstraction (`api` and `cli` modes) and provider factory details for Anthropic + OpenAI-compatible endpoints.
- Split acceptance criteria into deterministic CI checks vs optional manual/live-provider checks.
- Add a migration note from current SQLite scaffold to PRD JSON-state architecture, or explicitly choose one architecture and align all docs.

## Overall Assessment
PRD quality improved materially after the blocking fixes, but it is still not fully implementation-complete as a standalone machine-spec due to missing CLI/output contracts and a few data-model/history gaps. The biggest remaining risk is document divergence: current scaffold, PRD, and architecture are still not aligned on storage backend, command surface, and execution/provider model boundaries.
