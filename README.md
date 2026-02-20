# 🏟️ Ludus Magnus

![Ludus Magnus](images/ludus-magnus.jpg)


*The Training Ground. Four sparring zones. Floating score tablets. An evolution spiral carved into the arena floor.*

---

The Ludus Magnus was the largest gladiatorial training school in Rome, connected to the Colosseum by an underground passage. Gladiators didn't walk into the arena on their first day. They trained. They sparred. They fought practice rounds until the trainers decided they were ready. Some lineages were better than others. The best survived.

This Ludus Magnus is an open-air arena — sunlit marble with holographic training dummies. Four sparring zones marked A, B, C, D, each running a different lineage of the same agent. Floating score tablets show 1-10 evaluations. You watch them compete, lock the winner, evolve the losers, and repeat until what walks out of here is actually good at its job. The evolution spiral carved into the floor isn't decorative. It's the process.

Any agent type trains here — coding, research, customer service, analysis. The arena doesn't care what you're building. It cares whether you're improving.

Train AI agents through iterative evaluation loops. Define what you need, generate an agent, run it, score it, evolve it.

## How it works

Ludus Magnus generates system prompts for AI agents, runs them against your inputs, collects your evaluations, and uses the feedback to evolve better prompts. All state is stored locally in a single JSON file.

**Quickstart flow** (single lineage):
```
init -> run -> evaluate -> iterate -> run again
```

**Training flow** (four parallel lineages A/B/C/D):
```
init -> run all -> evaluate all -> lock winners -> iterate losers -> repeat
```

## Install

```bash
make build
# Binary at ./bin/ludus-magnus
```

Or directly:
```bash
go install .
```

## Quick start

```bash
# Set your API key
export ANTHROPIC_API_KEY=sk-...

# Create a session and generate the first agent
ludus-magnus quickstart init --need "Customer support assistant that handles refund requests"

# Run the agent (use the session_id from output)
ludus-magnus run ses_XXXXXXXX --input "I want a refund for order #1234"

# Score the result (use the artifact_id from output)
ludus-magnus evaluate art_XXXXXXXX --score 5 --comment "Too generic, needs order lookup"

# Evolve the agent based on feedback
ludus-magnus iterate ses_XXXXXXXX

# Run the evolved agent
ludus-magnus run ses_XXXXXXXX --input "I want a refund for order #1234"
```

## Commands

| Command | Description |
|---------|-------------|
| `quickstart init --need "..."` | Create session with one lineage and first agent |
| `training init --need "..."` | Create session with four lineages (A/B/C/D) |
| `run <session-id> --input "..."` | Execute latest agent, store artifact |
| `evaluate <artifact-id> --score N` | Score artifact 1-10 with optional `--comment` |
| `iterate <session-id>` | Generate next agent version from evaluations |
| `training iterate <session-id>` | Iterate all unlocked training lineages |
| `promote <session-id>` | Convert quickstart to training (4 lineages) |
| `lineage lock <session-id> <name>` | Freeze a lineage (skip during iterate) |
| `lineage unlock <session-id> <name>` | Unfreeze a lineage |
| `directive set <session-id> <lineage> --text "..." --sticky` | Add persistent directive |
| `directive set <session-id> <lineage> --text "..." --oneshot` | Add one-time directive |
| `directive clear <session-id> <lineage> <directive-id>` | Remove a directive |
| `session list` | List all sessions |
| `session inspect <session-id>` | Show session details |
| `artifact list <session-id>` | List artifacts with scores |
| `artifact inspect <artifact-id>` | Show artifact details |
| `export agent <agent-id> --format json\|python\|typescript` | Export agent definition |
| `export evidence <session-id>` | Export session data for analysis |
| `doctor` | Check environment (API keys, executors) |

All commands support `--json` for machine-readable output.

## Providers

**Anthropic** (default):
```bash
export ANTHROPIC_API_KEY=sk-...
ludus-magnus quickstart init --need "..."
```

**OpenAI-compatible** (OpenAI, LiteLLM, OpenRouter):
```bash
export OPENAI_API_KEY=sk-...
ludus-magnus quickstart init --need "..." --provider openai-compatible --model gpt-4o
```

Override per-command with `--provider`, `--model`, `--base-url`, `--api-key`.

## Training workflow

```bash
# 1. Initialize with four variant strategies
ludus-magnus training init --need "Generate migration plans"

# 2. Run all lineages
for L in A B C D; do
  ludus-magnus run ses_XXX --lineage $L --input "Migrate users table"
done

# 3. Evaluate each artifact
ludus-magnus evaluate art_AAA --score 9 --comment "Best balance"
ludus-magnus evaluate art_BBB --score 6 --comment "Too conservative"
ludus-magnus evaluate art_CCC --score 7 --comment "Creative but uneven"
ludus-magnus evaluate art_DDD --score 5 --comment "Too risky"

# 4. Lock the winner, evolve the rest
ludus-magnus lineage lock ses_XXX A
ludus-magnus training iterate ses_XXX
```

## State

State lives at `.ludus-magnus/state.json` relative to your working directory. One directory per project keeps state isolated. Add `.ludus-magnus/` to your `.gitignore`.

## CLI execution mode

Run agents through `claude` or `codex` CLI tools instead of the API:

```bash
ludus-magnus run ses_XXX --mode cli --executor claude --input "..."
```

## Development

```bash
make test              # Unit tests
make test-integration  # Integration tests (builds binary, uses mock server)
make clean             # Remove build artifacts
```

## Part of the Agora

Ludus Magnus was forged in **[Athena's Agora](https://github.com/Perttulands/athena-workspace)** — an autonomous coding system where AI agents build software and a training arena makes sure they earn their place before deployment.

[Argus](https://github.com/Perttulands/argus) watches the server. [Truthsayer](https://github.com/Perttulands/truthsayer) watches the code. [Oathkeeper](https://github.com/Perttulands/oathkeeper) watches the promises. [Relay](https://github.com/Perttulands/relay) carries the messages. Ludus Magnus trains what the others will judge.

The [mythology](https://github.com/Perttulands/athena-workspace/blob/main/mythology.md) has the full story.

## License

MIT
