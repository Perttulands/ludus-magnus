# Ludus Magnus CLI Usage

This guide documents the `ludus-magnus` command surface, common flags, and end-to-end workflows.

## Commands

### Global format flag

Use `--json` on most commands to return machine-readable output.

```bash
ludus-magnus --json session list
ludus-magnus --json run ses_12345678 --input "hello"
```

### Session commands

Create a session directly:

```bash
ludus-magnus session new --mode quickstart --need "Build a safe code review assistant"
```

List sessions:

```bash
ludus-magnus session list
ludus-magnus --json session list
```

Inspect one session:

```bash
ludus-magnus session inspect ses_12345678
```

### Quickstart commands

Initialize quickstart with one `main` lineage and first generated agent:

```bash
ludus-magnus quickstart init \
  --need "Refactor Python code safely" \
  --provider openai-compatible \
  --model gpt-4.1 \
  --base-url http://127.0.0.1:8000 \
  --api-key test-key
```

### Training commands

Initialize training with lineages `A/B/C/D`:

```bash
ludus-magnus training init \
  --need "Design robust API tests" \
  --provider anthropic
```

Iterate all unlocked training lineages:

```bash
ludus-magnus training iterate ses_12345678
```

### Run command

Run latest agent in a lineage and store artifact:

```bash
ludus-magnus run ses_12345678 --input "Solve task X"
```

Run one training lineage explicitly:

```bash
ludus-magnus run ses_12345678 --lineage A --input "Solve task X"
```

Run using CLI executor mode (`claude` or `codex`):

```bash
ludus-magnus run ses_12345678 \
  --lineage A \
  --mode cli \
  --executor codex \
  --input "Implement tests for parser"
```

### Evaluation commands

Score an artifact with optional comment:

```bash
ludus-magnus evaluate art_12345678 --score 8 --comment "Good correctness, improve naming"
```

### Iterate command

Iterate one lineage (`main` default for quickstart):

```bash
ludus-magnus iterate ses_12345678
ludus-magnus iterate ses_12345678 --lineage B
```

### Lineage lock commands

Lock/unlock a training lineage:

```bash
ludus-magnus lineage lock ses_12345678 A
ludus-magnus lineage unlock ses_12345678 A
```

### Promotion command

Promote quickstart session into training session:

```bash
ludus-magnus promote ses_12345678
```

### Directive commands

Set one-shot directive:

```bash
ludus-magnus directive set ses_12345678 A \
  --text "Increase robustness checks" \
  --oneshot
```

Set sticky directive:

```bash
ludus-magnus directive set ses_12345678 A \
  --text "Always explain tradeoffs" \
  --sticky
```

Clear a directive:

```bash
ludus-magnus directive clear ses_12345678 A dir_12345678
```

### Artifact commands

List artifacts in a session:

```bash
ludus-magnus artifact list ses_12345678
ludus-magnus --json artifact list ses_12345678
```

Inspect one artifact:

```bash
ludus-magnus artifact inspect art_12345678
```

### Export commands

Export one agent definition:

```bash
ludus-magnus export agent agt_12345678 --format json
ludus-magnus export agent agt_12345678 --format python
ludus-magnus export agent agt_12345678 --format typescript
```

Export one session evidence pack:

```bash
ludus-magnus export evidence ses_12345678 --format json
```

### Doctor command

Validate credentials, provider initialization, optional executors, and state readability:

```bash
ludus-magnus doctor
ludus-magnus doctor --provider openai-compatible --api-key test-key --json
```

## Workflows

### Quickstart Workflow

Copy-paste sequence:

```bash
# 1) Initialize quickstart
ludus-magnus quickstart init --need "Build a deterministic parser" --provider anthropic

# 2) Capture generated session id from output, then run
ludus-magnus run <session-id> --input "Parse: a,b,c"

# 3) Capture artifact id and evaluate
ludus-magnus evaluate <artifact-id> --score 7 --comment "Works but needs clearer error handling"

# 4) Evolve next version
ludus-magnus iterate <session-id>

# 5) Run evolved agent
ludus-magnus run <session-id> --input "Parse: a,b,c,d"
```

### Training Workflow

Copy-paste sequence:

```bash
# 1) Initialize training with A/B/C/D
ludus-magnus training init --need "Generate reliable migration plans" --provider anthropic

# 2) Run all variants (replace <session-id>)
ludus-magnus run <session-id> --lineage A --input "Plan DB migration"
ludus-magnus run <session-id> --lineage B --input "Plan DB migration"
ludus-magnus run <session-id> --lineage C --input "Plan DB migration"
ludus-magnus run <session-id> --lineage D --input "Plan DB migration"

# 3) Evaluate each produced artifact id
ludus-magnus evaluate <artifact-A> --score 9 --comment "Best balance"
ludus-magnus evaluate <artifact-B> --score 6 --comment "Too conservative"
ludus-magnus evaluate <artifact-C> --score 7 --comment "Creative but uneven"
ludus-magnus evaluate <artifact-D> --score 5 --comment "Too risky"

# 4) Lock winners
ludus-magnus lineage lock <session-id> A

# 5) Regenerate only unlocked variants
ludus-magnus training iterate <session-id>
```

### Promotion Workflow

Copy-paste sequence:

```bash
# 1) Start quickstart
ludus-magnus quickstart init --need "Create test-first bugfix prompts" --provider anthropic

# 2) Promote quickstart session into training mode
ludus-magnus promote <session-id>

# 3) Iterate training lineages after promotion
ludus-magnus training iterate <session-id>
```

## JSON Output Examples

Session listing:

```bash
ludus-magnus --json session list
```

Training iterate summary:

```bash
ludus-magnus --json training iterate <session-id>
```

Directive set response:

```bash
ludus-magnus --json directive set <session-id> A --text "Keep steps minimal" --sticky
```

## State File

State is stored relative to current working directory at:

```text
.ludus-magnus/state.json
```

High-level structure:

```text
version
sessions
  <session-id>
    mode
    need
    lineages
      <lineage-id>
        agents
        artifacts
        directives
```

Tip: keep one working directory per project so state stays isolated.
