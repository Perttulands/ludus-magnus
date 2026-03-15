# PR #1 Release Plan

## Goal

Ship the smallest correct version of PR #1 now, and defer broader provider design questions into follow-up beads.

## What To Ship Now

Ship these parts of the PR:

- `sealed` execution mode in `cmd/run.go` and `internal/engine/execute.go`
- external harness invocation with `--harness`, `--harness-model`, `--condition`, and `--run-number`
- parsing `result.json` from the harness output
- `pi-cli` as a local provider path for the default local Pi + Ollama setup
- doctor support that checks the `pi` binary exists

## What Not To Ship In This PR

Do not expand the contract beyond what is already implemented and tested.

Defer these items:

- generic remote Ollama endpoint support through `pi-cli`
- any claim that `pi-cli` really supports `--base-url`
- treating bare `ollama` as an alias for `pi-cli`
- broader provider taxonomy changes unless they are already settled

## Required Changes Before Merge

### 1. Narrow the `pi-cli` contract

Make `pi-cli` explicitly mean:

- use the local `pi` binary
- use the local/default Ollama-backed Pi setup

That contract should be reflected in code, help text, and docs.

### 2. Remove misleading endpoint behavior

The current PR stores `baseURL` for `pi-cli` but does not use it during execution.

Before merge, do one of these:

- remove `--base-url` support for `pi-cli`, or
- fail fast with a clear error when `--provider pi-cli` is used with a non-empty base URL

Preferred for this PR: fail fast or remove the alias surface, not partial support.

### 3. Keep provider aliases conservative

Allowed aliases for this PR:

- `pi`
- `pi-cli`
- optionally `pi-ollama`

Do not map bare `ollama` to `pi-cli` in this PR. That name should stay available for a distinct native Ollama provider if needed.

### 4. Keep doctor honest

`doctor --provider pi-cli` should validate only the runtime assumptions this PR actually depends on:

- `pi` binary exists
- messaging should describe `pi-cli` as a local provider path

It should not imply that arbitrary Ollama endpoints are supported unless that is truly implemented.

### 5. Decide on sealed metadata scope

If cheap to include now, add parsing for `provider` from harness `result.json` and store it in execution metadata.

If not, defer it and track it in a bead.

## Tests Required Before Merge

Keep the existing green test suite, and add focused tests for the narrowed contract:

- provider normalization does not map bare `ollama` to `pi-cli`
- `pi-cli` rejects unsupported `baseURL` usage, if that is the chosen behavior
- sealed mode tests continue to pass
- if provider metadata is included now, sealed result parsing preserves it

## Docs Required Before Merge

Update CLI help, README, and PR description so they match the shipped behavior:

- sealed mode delegates execution to an external harness
- `pi-cli` is local-only in this release
- remote/custom Ollama endpoint support is not part of this PR

## Follow-up Beads

Create these beads before merge so deferred work is explicit:

1. Clarify provider taxonomy: `pi-cli` vs `ollama-native`
2. Add real custom endpoint support for `pi-cli`, or remove `baseURL` from that path permanently
3. Preserve provider metadata in sealed result parsing
4. Add integration coverage for local-provider and sealed-harness contracts

## Merge Criteria

Merge PR #1 only when all of the following are true:

- no exposed flag or alias makes a false promise
- tests are green
- docs and help text match actual runtime behavior
- deferred design work is captured in beads
