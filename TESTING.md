# Testing — Chiron

## Rubric Scores

| Dimension | Before | After | Notes |
|-----------|--------|-------|-------|
| E2E Realism | 3 | 3 | Integration test covers quickstart→run→evaluate→iterate full loop with mock API server. Strong for the core workflow but no coverage of training init, promote, export, doctor, or directive flows. |
| Unit Test Behaviour Focus | 2 | 4 | Before: most unit tests targeted observable behaviour but truthsayer only tested QualityScore (a pure function) and struct fields — none of the actual scanning logic. Provider tests existed only for happy paths. After: tests cover factory routing, all error paths, context cancellation, system prompt handling, default fallbacks, and truthsayer scanning via fake binaries. |
| Edge Case & Error Path Coverage | 1 | 3 | Before: only a handful of error tests (nil loop, nonexistent file, missing credentials). After: API error responses (429, 500, 401), empty API content/choices, unknown models with zero cost, corrupted JSON, exit codes 0/1/2/99, context cancellation, and flag validation. Still missing: timeout handling, concurrent access, filesystem permission errors. |
| Test Isolation & Reliability | 4 | 4 | Tests use t.TempDir(), t.Setenv(), httptest.NewServer. No shared state, no sleep(), no external dependencies. Fake binaries in truthsayer use temp scripts. Tests are parallel-safe where marked. |
| Regression Value | 2 | 3 | Before: tests would catch basic compilation failures and the QualityScore formula. After: tests catch provider routing bugs, API error handling regressions, truthsayer binary discovery and JSON parsing failures, cmd subcommand registration breakage, flag validation, and helper function logic errors. A missing subcommand or broken factory switch would be caught. Still not caught: state file corruption during cmd execution, provider selection bugs in multi-provider scenarios. |

**Before: 12/25 (Grade D)**
**After: 17/25 (Grade C)**

## Assessment Per Dimension

### E2E Realism (3/5)
The integration test in `test/integration/quickstart_test.go` is genuinely good — it builds the binary, runs a full quickstart→run→evaluate→iterate cycle against a mock OpenAI server, and verifies state on disk. This covers the most important user workflow. Missing: training init, promote, export, directive set/clear, doctor, and session management flows.

### Unit Test Behaviour Focus (4/5)
Tests now target behaviour at the right abstraction level. Provider tests verify HTTP requests are well-formed, error responses produce correct error messages, and defaults are applied correctly. Truthsayer tests verify scanning behaviour with real shell scripts as fake binaries — testing the actual code paths not just types. Cmd tests verify subcommand routing and helper logic without mocking internal packages.

### Edge Case & Error Path Coverage (3/5)
Major error paths are now covered: missing binaries, bad JSON output, non-zero exit codes, API errors, empty responses, unknown models, context cancellation, nil loops, corrupted files, and flag validation. The gap is at boundaries: concurrent checkpoint operations, filesystem permission failures, and HTTP timeout edge cases.

### Test Isolation & Reliability (4/5)
All tests are hermetic. Temp directories via `t.TempDir()` for filesystem tests, `httptest.NewServer` for provider tests, `t.Setenv` for environment variables. No global state mutation between tests. The truthsayer fake-binary approach is clean — scripts write to stdout and exit with controlled codes.

### Regression Value (3/5)
The suite would now catch: provider factory routing broken, API error handling removed, truthsayer binary discovery failing, exit code semantics changed, subcommand deregistered, helper function logic altered, default values changed. It would NOT catch: subtle state corruption bugs, race conditions in concurrent session access, or provider-specific API contract changes.

## What the Suite Is MISSING

### Critical gaps:
1. **Training init flow** — no test for `chiron training init` which generates 4 lineages. A bug here breaks the entire training workflow.
2. **Promote flow** — `chiron promote` converts quickstart→training with alternative strategies. Untested.
3. **Doctor command** — `chiron doctor` validates environment. No unit test runs checkProviderCredentials with various env combinations.
4. **Export formats** — `internal/export` at 74.3%. Python and TypeScript rendering have edge cases (special characters in system prompts, empty tools arrays) that aren't tested for regression.
5. **State migration** — `MigrateLegacyDir` and `MigrateState` handle the .ludus-magnus→.chiron rename. No test verifies migration doesn't corrupt state.
6. **Concurrent checkpoint access** — checkpoint Save/Load has no locking. Tests should verify behaviour under concurrent writes.
7. **CLI execution mode** — `engine.executeCLI` is untested (requires `claude` or `codex` binary). Could be tested with a fake binary approach like truthsayer.
8. **Session list/inspect with real data** — cmd tests verify routing but don't test the actual commands against a populated state file.

### Lower priority:
- Cost calculation accuracy across all pricing tiers
- Training loop termination conditions (target score reached, max generations)
- Challenge generation edge cases (empty corpus, single challenge)

## Coverage Summary

| Package | Before | After | Delta |
|---------|--------|-------|-------|
| cmd | 0.0% | 29.6% | +29.6% |
| internal/truthsayer | 8.0% | 92.0% | +84.0% |
| internal/provider | 70.1% | 92.0% | +21.9% |
| internal/checkpoint | 70.5% | 79.5% | +9.0% |
| internal/engine | 79.9% | 79.9% | — |
| internal/export | 74.3% | 74.3% | — |
| internal/state | 82.3% | 82.3% | — |
| **Overall estimated** | **~48.8%** | **~63%** | **+~14%** |

## Changelog

### 2026-02-28 — Agent: zeus (claude-opus-4-6)
- Added: 24 truthsayer tests covering ScanWithBinary (exit codes 0/1/2/99, bad JSON, empty output, clean scan, findings detected), ScanStringWithBinary (file creation, delegation), QualityScore edge cases (boundary values, floor), JSON round-trip for Finding/ScanResult/ScanOutput structs
- Added: 25 provider tests covering factory routing (normalize, unsupported, API key from flag), Anthropic error paths (429/500, empty content, unknown model cost, context cancellation, default maxTokens), OpenAI-compatible error paths (401, empty choices, no system prompt, total_tokens fallback), completionsURL routing, GetMetadata/defaults
- Added: 8 checkpoint tests covering corrupted JSON, empty file, nested directory creation, JSON validity, various reasons, subdirectory filtering in ListIn, LoadFrom error paths
- Added: 40 cmd tests covering subcommand registration (all 12 commands), help output (7 commands), argument validation (run/evaluate/iterate/promote), flag requirements (--input, --score), helper functions (findLineageByName, latestAgent, modelOrDefault, newPrefixedID, removeDirectiveByID, variantsForStrategy, normalizeDoctorProvider, agentVersionForArtifact, isJSONOutput, writeJSON), subcommand tree verification (session/artifact/directive/export/training/lineage/quickstart)
- Coverage delta: 48.8% → ~63% (97 new tests covering real scanning, API error handling, factory routing, CLI subcommand wiring, and state helper logic)
