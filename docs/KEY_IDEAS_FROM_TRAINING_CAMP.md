# Key Ideas From Training Camp

## 1. Train Agents, Not Prompts In Isolation

Training signal should improve a full agent definition (system prompt, execution flow, parameters, and optional tools), not only one prompt string.

## 2. Minimal User Burden, Maximum Signal

The strongest interaction model is:
1. State intent and constraints.
2. Evaluate produced artifacts.

Everything else (generation, variation, evolution) is orchestration.

## 3. Two-Phase Learning Pattern

- Exploration: fast single-agent iteration to validate concept viability.
- Optimization: parallel lineages with comparative evaluation to find robust winners.

This mirrors how teams naturally work and avoids premature overhead.

## 4. Lineage-Based Evolution

Maintaining independent lineages prevents converging too early on one approach.

Useful controls:
- lock winners
- regenerate unlocked lineages
- add one-shot or sticky directives

## 5. Cycle Semantics

A cycle should be explicit and reproducible:
1. Generate or evolve agent definition.
2. Execute on input(s).
3. Store artifacts.
4. Collect evaluation.
5. Apply evolution.

## 6. Durable Training Signal

Persist everything needed for replay and offline analysis:
- session metadata
- agent versions
- artifacts
- scores/comments
- directives and lock state

This enables future dataset extraction (SFT, preference pairs, reward modeling).

## 7. Reality-First Product Behavior

No fake-success paths. Failed LLM/tool calls should be visible and attributable.

## 8. Exportability

Agent definitions and evidence should be exportable to portable formats so outcomes can be reused outside the trainer.

## 9. Architecture Separation

The conceptual separation remains useful in CLI form:
- Master Trainer: strategy and evolution logic
- Store: state and lineage history
- Executor: runs agent definitions

## 10. CLI Translation Principle

The web UI constructs map cleanly to command operations:
- card/grid view -> `session status` and structured output
- slider scoring -> `evaluate --score`
- regenerate button -> `iterate`
- view agent modal -> `agent show`
