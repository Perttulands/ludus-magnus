# Chiron CLI Usage

This guide documents the `chiron` command surface, common flags, and end-to-end workflows.

## Commands

### Global format flag

Use `--json` on most commands to return machine-readable output.

```bash
chiron --json session list
chiron --json run ses_12345678 --input "hello"
```

### Session commands

Create a session directly:

```bash
chiron session new --mode quickstart --need "Build a safe code review assistant"
```

List sessions:

```bash
chiron session list
chiron --json session list
```

Inspect one session:

```bash
chiron session inspect ses_12345678
```

### Quickstart commands

Initialize quickstart with one `main` lineage and first generated agent:

```bash
chiron quickstart init \
  --need "Refactor Python code safely" \
  --provider openai-compatible \
  --model gpt-4.1 \
  --base-url http://127.0.0.1:8000 \
  --api-key test-key
```

### Training commands

Initialize training with lineages `A/B/C/D`:

```bash
chiron training init \
  --need "Design robust API tests" \
  --provider anthropic
```

Iterate all unlocked training lineages:

```bash
chiron training iterate ses_12345678
```

### Run command

Run latest agent in a lineage and store artifact:

```bash
chiron run ses_12345678 --input "Solve task X"
```

Run one training lineage explicitly:

```bash
chiron run ses_12345678 --lineage A --input "Solve task X"
```

Run using CLI executor mode (`claude` or `codex`):

```bash
chiron run ses_12345678 \
  --lineage A \
  --mode cli \
  --executor codex \
  --input "Implement tests for parser"
```

### Evaluation commands

Score an artifact with optional comment:

```bash
chiron evaluate art_12345678 --score 8 --comment "Good correctness, improve naming"
```

### Iterate command

Iterate one lineage (`main` default for quickstart):

```bash
chiron iterate ses_12345678
chiron iterate ses_12345678 --lineage B
```

### Lineage lock commands

Lock/unlock a training lineage:

```bash
chiron lineage lock ses_12345678 A
chiron lineage unlock ses_12345678 A
```

### Promotion command

Promote quickstart session into training session:

```bash
chiron promote ses_12345678
```

### Directive commands

Set one-shot directive:

```bash
chiron directive set ses_12345678 A \
  --text "Increase robustness checks" \
  --oneshot
```

Set sticky directive:

```bash
chiron directive set ses_12345678 A \
  --text "Always explain tradeoffs" \
  --sticky
```

Clear a directive:

```bash
chiron directive clear ses_12345678 A dir_12345678
```

### Artifact commands

List artifacts in a session:

```bash
chiron artifact list ses_12345678
chiron --json artifact list ses_12345678
```

Inspect one artifact:

```bash
chiron artifact inspect art_12345678
```

### Export commands

Export one agent definition:

```bash
chiron export agent agt_12345678 --format json
chiron export agent agt_12345678 --format python
chiron export agent agt_12345678 --format typescript
```

Export one session evidence pack:

```bash
chiron export evidence ses_12345678 --format json
```

### Doctor command

Validate credentials, provider initialization, optional executors, and state readability:

```bash
chiron doctor
chiron doctor --provider openai-compatible --api-key test-key --json
```

## Workflows

### Quickstart Workflow

Copy-paste sequence:

```bash
# 1) Initialize quickstart
chiron quickstart init --need "Build a deterministic parser" --provider anthropic

# 2) Capture generated session id from output, then run
chiron run <session-id> --input "Parse: a,b,c"

# 3) Capture artifact id and evaluate
chiron evaluate <artifact-id> --score 7 --comment "Works but needs clearer error handling"

# 4) Evolve next version
chiron iterate <session-id>

# 5) Run evolved agent
chiron run <session-id> --input "Parse: a,b,c,d"
```

### Training Workflow

Copy-paste sequence:

```bash
# 1) Initialize training with A/B/C/D
chiron training init --need "Generate reliable migration plans" --provider anthropic

# 2) Run all variants (replace <session-id>)
chiron run <session-id> --lineage A --input "Plan DB migration"
chiron run <session-id> --lineage B --input "Plan DB migration"
chiron run <session-id> --lineage C --input "Plan DB migration"
chiron run <session-id> --lineage D --input "Plan DB migration"

# 3) Evaluate each produced artifact id
chiron evaluate <artifact-A> --score 9 --comment "Best balance"
chiron evaluate <artifact-B> --score 6 --comment "Too conservative"
chiron evaluate <artifact-C> --score 7 --comment "Creative but uneven"
chiron evaluate <artifact-D> --score 5 --comment "Too risky"

# 4) Lock winners
chiron lineage lock <session-id> A

# 5) Regenerate only unlocked variants
chiron training iterate <session-id>
```

### Promotion Workflow

Copy-paste sequence:

```bash
# 1) Start quickstart
chiron quickstart init --need "Create test-first bugfix prompts" --provider anthropic

# 2) Promote quickstart session into training mode
chiron promote <session-id>

# 3) Iterate training lineages after promotion
chiron training iterate <session-id>
```

## JSON Output Examples

Session listing:

```bash
chiron --json session list
```

Training iterate summary:

```bash
chiron --json training iterate <session-id>
```

Directive set response:

```bash
chiron --json directive set <session-id> A --text "Keep steps minimal" --sticky
```

## State File

State is stored relative to current working directory at:

```text
.chiron/state.json
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
