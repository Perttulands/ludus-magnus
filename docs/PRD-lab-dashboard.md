# PRD: Chiron Lab Dashboard

**Author:** Claude (auto-generated)
**Date:** 2026-03-09
**Status:** Draft

---

## 1. Problem

Chiron produces structured experiment data across 5+ experiments and 26+ runs. Today, understanding results requires: manually reading LAB-BOOK.md files, grepping JSON, eyeballing diffs, and mentally stitching together cross-experiment trends. There is no unified view from "what did we learn?" down to "what exactly did the agent do in run ext-r2?"

## 2. Goal

A local lab dashboard that tells the full story — from high-level findings down to raw run data — where every number is a drill-down. Think: interactive lab notebook where experiments, conditions, and individual runs are all explorable.

---

## 3. View Levels

### Level 1 — Experiment Overview

A table of all experiments with:

| Column | Source |
|--------|--------|
| ID | Directory name (e.g., `015-35b-extensions`) |
| Name | `experiment.yaml` → `name` |
| Date | Earliest `meta.json` → `timestamp` across runs |
| Hypothesis | Parsed from LAB-BOOK.md (first line after `**Hypothesis:**`) |
| Result | Parsed from LAB-BOOK.md decision section: PASS / REJECTED / NEUTRAL |
| Mean Score | Computed from manual B1-B4 in LAB-BOOK.md results table |
| Runs | Count of `meta.json` files in `runs/` |
| Model(s) | Unique models from `experiment.yaml` → `models[].id` |

**Interactions:**
- Sortable by any column
- Filterable by result (pass/fail/neutral) and model
- Click row → Level 2

**Mockup:**
```
┌─────────────────────────────────────────────────────────────────────────┐
│  CHIRON LAB                                              [Filter ▾]    │
├──────┬──────────────────────┬────────┬──────────┬──────┬───────┬──────┤
│ EXP  │ Hypothesis           │ Result │ Mean     │ Runs │ Model │ Date │
├──────┼──────────────────────┼────────┼──────────┼──────┼───────┼──────┤
│ 015  │ Extensions neutral?  │ NEUTRAL│ 5.00/8   │  3   │ 35B   │ 03-09│
│ 014  │ BR hint helps 35B    │ REJECT │ 2.83/8   │  3   │ 35B   │ 03-08│
│ 013  │ 35B > 9B             │ PASS   │ 5.33/8   │  6   │ both  │ 03-07│
│ 012  │ B3 prompt boost      │ REJECT │ 2.00/8   │  3   │ 9B    │ 03-06│
│ 011  │ Extensions vs bare   │ PASS   │ 5.33/8   │  6   │ 9B    │ 03-05│
└──────┴──────────────────────┴────────┴──────────┴──────┴───────┴──────┘
```

### Level 2 — Experiment Detail

Click into an experiment. Three panels:

**Panel A — Lab Book** (left, 60% width)
- Full LAB-BOOK.md rendered as HTML (markdown → HTML)
- All markdown features: tables, headers, emphasis, code blocks

**Panel B — Run Matrix** (right, 40% width)
- Table: one row per run (condition × replica)

| Column | Source |
|--------|--------|
| Run ID | Directory name (e.g., `ext-r1-1`) |
| B1–B4 | From LAB-BOOK.md results table (manual scores) |
| Total | Sum of B1–B4 |
| Turns | `meta.json` → `turns` |
| Wall Time | `meta.json` → `duration_ms` (formatted as Xm Ys) |
| Tokens | `meta.json` → `tokens_in + tokens_out` |
| Tools | `meta.json` → `tool_calls` |
| Edits | `meta.json` → `edit_count` |

- Click any run → Level 3
- Color-code cells: green (max), yellow (mid), red (min) per column

**Panel C — Config** (collapsible drawer)
- Raw `experiment.yaml` rendered with syntax highlighting
- System prompt text (from referenced file)

**Mockup:**
```
┌─────────────────────────────────────────────┬───────────────────────────┐
│  ← Back to Overview                         │  Run Matrix               │
│                                             ├───────┬──┬──┬──┬──┬──────┤
│  # Lab Book — EXP-015                       │ Run   │B1│B2│B3│B4│Total │
│                                             ├───────┼──┼──┼──┼──┼──────┤
│  **Hypothesis:** Extensions neutral at 35B  │ ext-r1│ 2│ 2│ 0│ 1│ 5/8  │
│                                             │ ext-r2│ 2│ 2│ 0│ 1│ 5/8  │
│  **Background:** EXP-013 showed 35B wins... │ ext-r3│ 2│ 2│ 0│ 1│ 5/8  │
│                                             ├───────┼──┼──┼──┼──┼──────┤
│  ## Results                                 │ Mean  │2.0│2.0│0.0│1.0│5.0│
│  | Run | B1 | B2 | B3 | B4 | Total |      │       │  │  │  │  │      │
│  ...                                        │ [Config ▾]               │
└─────────────────────────────────────────────┴───────────────────────────┘
```

### Level 3 — Run Detail

Full detail for a single run. Tabbed layout:

**Tab 1 — Summary**
- Score card: B1–B4 with dimension names and explanations
  - Green check / red X per dimension
  - Score bar visualization (0–2 for B1-B3, 0–1 for B4)
- Metadata card: model, condition, replica, wall time, turns, tokens (in/out), tool calls, edit count, sandbox engine, timestamp

**Tab 2 — Transcript** (the hard part)
- Parse `raw-output.jsonl` and render as a conversation
- Event rendering rules:

| JSONL `type` | Render As |
|-------------|-----------|
| `message_start` + `message_end` (role=user) | User message bubble (blue) |
| `message_start` + `message_end` (role=assistant) | Assistant message bubble (gray) |
| `message_update` (thinking_start/delta/end) | Collapsible "Thinking" block (italic, muted) |
| `tool_call` | Tool invocation block: tool name + input (collapsible JSON) |
| `tool_result` | Tool result block: output text (collapsible, truncated at 50 lines with expand) |
| `turn_start` | Turn separator with turn number |
| `session` | Session header with metadata |

- Thinking blocks collapsed by default (click to expand)
- Tool call + result paired visually (indented under the assistant message)
- Tool results truncated to 50 lines with "Show all (N lines)" expand
- Sticky turn counter in the left margin
- Search within transcript (ctrl+F style, highlights matches)

**Tab 3 — Diff**
- `workspace.diff` rendered with syntax highlighting (unified diff format)
- File headers as collapsible sections
- Addition lines green, deletion lines red
- Line numbers shown

**Tab 4 — BR Invocations**
- `br-invocations.log` rendered as a table: timestamp, command, arguments
- Empty state: "No br invocations in this run"

**Mockup (Transcript tab):**
```
┌──────────────────────────────────────────────────────────────────────┐
│  ← EXP-015 / ext-r1                                                 │
│  [Summary] [Transcript] [Diff] [BR Log]                             │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ── Turn 1 ──────────────────────────────────────────────────────── │
│                                                                      │
│  ┌─ USER ───────────────────────────────────────────────────────┐   │
│  │ You are a senior Go developer. A bug report has come in...   │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  ┌─ ASSISTANT ──────────────────────────────────────────────────┐   │
│  │ ▸ Thinking (collapsed)                                       │   │
│  │                                                               │   │
│  │ Let me start by understanding the codebase structure.         │   │
│  │                                                               │   │
│  │  ┌─ read ("/workspace/cmd/main.go") ──────────────────────┐  │   │
│  │  │ package main...                                         │  │   │
│  │  │ [Show all 142 lines]                                    │  │   │
│  │  └────────────────────────────────────────────────────────┘  │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  ── Turn 2 ──────────────────────────────────────────────────────── │
│  ...                                                                 │
└──────────────────────────────────────────────────────────────────────┘
```

### Level 4 — Cross-Experiment Analysis

A dedicated analysis view accessible from the top nav.

**Chart A — Score Trend**
- X-axis: experiment ID (chronological)
- Y-axis: mean total score (0–8)
- One line per model (9B blue, 35B orange)
- Error bars showing min–max range
- Hover: tooltip with experiment name, mean, range

**Chart B — Dimension Heatmap**
- Rows: experiments (chronological)
- Columns: B1, B2, B3, B4
- Cell color: mean score (red=0, yellow=1, green=2)
- Shows at a glance which dimensions are strong vs stuck

**Chart C — Efficiency Scatter**
- X-axis: wall time (seconds)
- Y-axis: total score
- One dot per run, colored by model
- Shows quality-vs-cost tradeoff

**Table — Variable Impact**
- Rows: variables tested (extensions, model size, prompt specificity, BR hints)
- Columns: experiment, delta score, p-value (if n≥3), verdict
- Quick answer to "what actually moved the needle?"

---

## 4. Data Flow

```
┌─────────────────────────────────────────────┐
│  experiments/                                │
│  ├── NNN-name/                               │
│  │   ├── experiment.yaml  ──────┐            │
│  │   ├── LAB-BOOK.md      ──────┤            │
│  │   ├── prompts/*.txt    ──────┤            │
│  │   └── runs/                   │            │
│  │       └── model/condition/    │            │
│  │           ├── meta.json  ─────┤            │
│  │           ├── result.json ────┤            │
│  │           ├── raw-output.jsonl┤            │
│  │           ├── workspace.diff ─┤            │
│  │           └── br-invocations  │            │
│  └── ...                         │            │
└──────────────────────────────────┘            │
                                                ▼
                                   ┌────────────────────┐
                                   │  chiron dashboard   │
                                   │  (Go HTTP server)   │
                                   │                     │
                                   │  1. Scan experiments │
                                   │  2. Parse YAML/JSON │
                                   │  3. Parse LAB-BOOK  │
                                   │  4. Serve REST API  │
                                   │  5. Serve static UI │
                                   └────────┬───────────┘
                                            │
                                   ┌────────▼───────────┐
                                   │  React SPA          │
                                   │  (Vite build,       │
                                   │   served by Go)     │
                                   │                     │
                                   │  L1: Overview table  │
                                   │  L2: Experiment view │
                                   │  L3: Run detail      │
                                   │  L4: Cross-analysis  │
                                   └─────────────────────┘
```

---

## 5. Architecture Decision: `chiron dashboard` Command

**Decision: Add a `chiron dashboard` subcommand that starts a local HTTP server.**

Rationale:
- Chiron is already a Go CLI (Cobra). Adding a subcommand is natural.
- All data lives on disk in known paths. No database needed.
- Go's `net/http` + `embed` can serve a React SPA from a single binary.
- No external dependencies to install — just `chiron dashboard` and open a browser.
- The Go backend handles file scanning, YAML/JSON parsing, LAB-BOOK markdown parsing, and JSONL streaming.
- The React frontend handles rendering, interactivity, and charting.

Alternative considered — **static site generator**: Rejected. Transcripts are 30+ MB each; pre-rendering is wasteful. The JSONL viewer needs on-demand streaming (load first 100 events, lazy-load more on scroll). A live server handles this naturally.

Alternative considered — **reuse polis-command viewer**: The polis-command web UI is a vanilla JS SPA with no charting, no tables, and an event model incompatible with Chiron's data. Its WebSocket-based architecture is overkill for reading static experiment files. Not reusable beyond inspiration.

Alternative considered — **reuse orbit-ui**: Orbit-UI is React + Vite + Zustand + Express. Its architecture pattern (server collects data → pushes snapshots → React renders) is a good match, but it has no charting, no table components, and no code for parsing experiment data. **Reuse the tooling pattern (Vite + React + Zustand) but build the components fresh.**

---

## 6. Tech Stack

| Layer | Choice | Rationale |
|-------|--------|-----------|
| **Backend** | Go (in Chiron binary) | Already a Go project. Single binary. No Node.js dependency for serving. |
| **Backend framework** | `net/http` + stdlib | Chiron has no web deps yet. stdlib is sufficient for REST + static serving. |
| **Frontend framework** | React 18 + TypeScript | Orbit-UI precedent. Component model fits the drill-down navigation. |
| **State management** | Zustand | Lightweight, orbit-ui precedent. No Redux boilerplate. |
| **Build tool** | Vite | Orbit-UI precedent. Fast HMR for development. |
| **Charting** | Recharts | React-native, lightweight (< 200KB), good for line/bar/scatter/heatmap. |
| **Table** | TanStack Table (headless) | Sorting, filtering, pagination. No opinionated styling. |
| **Markdown rendering** | marked (browser) | Lightweight, renders LAB-BOOK.md in the browser. |
| **Diff rendering** | diff2html (browser) | Renders unified diffs with syntax highlighting. |
| **Code highlighting** | Prism.js (via diff2html) | YAML, Go, TypeScript highlighting in config and diff views. |
| **CSS** | Custom (no framework) | Orbit-UI pattern. Warm palette (inherit `--bg`, `--ink`, `--brand` vars). |
| **Embedding** | `go:embed` | Embed built frontend assets into Go binary. Single deployable. |

### Frontend directory structure

```
dashboard/
├── vite.config.ts
├── package.json
├── tsconfig.json
├── index.html
└── src/
    ├── main.tsx
    ├── store.ts                  # Zustand: experiments, selected experiment/run
    ├── api.ts                    # fetch wrappers for /api/v1/*
    ├── types.ts                  # TypeScript interfaces matching Go API responses
    ├── App.tsx                   # Router: Overview | Experiment | Run | Analysis
    ├── components/
    │   ├── ExperimentTable.tsx   # L1: sortable/filterable experiment list
    │   ├── ExperimentDetail.tsx  # L2: lab book + run matrix
    │   ├── LabBook.tsx           # Rendered markdown
    │   ├── RunMatrix.tsx         # Condition × replica table with heatmap
    │   ├── RunDetail.tsx         # L3: tabbed run view
    │   ├── ScoreCard.tsx         # B1-B4 visual breakdown
    │   ├── TranscriptViewer.tsx  # JSONL → conversation renderer
    │   ├── DiffViewer.tsx        # Syntax-highlighted diff
    │   ├── BrLog.tsx             # BR invocations table
    │   ├── AnalysisView.tsx      # L4: cross-experiment charts
    │   ├── ScoreTrend.tsx        # Recharts line chart
    │   ├── DimensionHeatmap.tsx  # Recharts heatmap
    │   └── EfficiencyScatter.tsx # Recharts scatter
    └── styles.css
```

---

## 7. API Design

All endpoints under `/api/v1/`. Go backend reads from disk on each request (data is small, no caching needed initially).

### `GET /api/v1/experiments`

List all experiments with summary data.

```json
[
  {
    "id": "015-35b-extensions",
    "name": "015-35b-extensions",
    "date": "2026-03-09T01:44:16Z",
    "hypothesis": "Extensions neutral at 35B",
    "result": "NEUTRAL",
    "mean_score": 5.0,
    "max_score": 8,
    "run_count": 3,
    "models": ["qwen3.5:35b-t03"]
  }
]
```

**Implementation:** Scan `experiments/*/`, parse each `experiment.yaml` + `LAB-BOOK.md` header + count `meta.json` files.

### `GET /api/v1/experiments/:id`

Full experiment detail.

```json
{
  "id": "015-35b-extensions",
  "config": { /* raw experiment.yaml as JSON */ },
  "lab_book_md": "# Lab Book — EXP-015...",
  "system_prompt": "You are a senior Go developer...",
  "runs": [
    {
      "id": "ext-r1-1",
      "model": "qwen3.5:35b-t03",
      "condition": "ext-r1",
      "replica": 1,
      "meta": { /* meta.json contents */ },
      "result": { /* result.json contents */ },
      "scores": { "B1": 2, "B2": 2, "B3": 0, "B4": 1, "total": 5 },
      "has_transcript": true,
      "has_diff": true,
      "has_br_log": true
    }
  ]
}
```

**Scores:** B1–B4 are parsed from the LAB-BOOK.md results table. The Go backend uses a simple markdown table parser to extract per-run scores. If auto-scores exist in `scores.json`, those are included too.

### `GET /api/v1/experiments/:id/runs/:runId/transcript`

Stream the JSONL transcript. Returns parsed events (not raw JSONL).

```json
{
  "events": [
    {
      "type": "turn_start",
      "turn_number": 1,
      "timestamp": "2026-03-09T01:25:58.326Z"
    },
    {
      "type": "user_message",
      "content": "You are a senior Go developer...",
      "timestamp": 1773019558331
    },
    {
      "type": "assistant_message",
      "content": "Let me start by understanding the codebase.",
      "thinking": "The user wants me to fix a bug...",
      "tool_calls": [
        {
          "id": "call_123",
          "tool": "read",
          "input": {"file_path": "/workspace/cmd/main.go"},
          "output": "package main\n...",
          "is_error": false
        }
      ],
      "usage": {"input": 1024, "output": 256},
      "timestamp": 1773019558332
    }
  ],
  "total_events": 847
}
```

**Pagination:** `?offset=0&limit=100` — load 100 events at a time. Frontend uses infinite scroll.

**Implementation:** The Go backend reads `raw-output.jsonl` line by line, assembles `turn_start` → `message_start/end` → `tool_call/result` sequences into the structured events above. Thinking deltas are concatenated into a single `thinking` string. Tool calls and their results are paired by `callID`.

### `GET /api/v1/experiments/:id/runs/:runId/diff`

```json
{
  "raw": "diff -ruN ...",
  "files_changed": 2,
  "additions": 15,
  "deletions": 8
}
```

### `GET /api/v1/experiments/:id/runs/:runId/br-log`

```json
{
  "invocations": [
    {
      "timestamp": "2026-03-09T03:42:40+02:00",
      "command": "create",
      "args": ["--title", "Fix race in runner.go", "--desc", "..."]
    }
  ]
}
```

### `GET /api/v1/analysis`

Cross-experiment aggregation.

```json
{
  "experiments": [
    {
      "id": "011-extensions-vs-bare",
      "date": "2026-03-05",
      "models": ["qwen3.5:9b-t03"],
      "variables": ["extensions"],
      "dimensions": {"B1": 1.67, "B2": 1.67, "B3": 0.0, "B4": 0.0},
      "mean_score": 3.33,
      "score_range": [1, 6],
      "run_count": 6
    }
  ],
  "variables": [
    {
      "name": "model_size",
      "experiments": ["013-model-scale-35b"],
      "delta": 2.0,
      "verdict": "35B wins (+60%)"
    },
    {
      "name": "extensions",
      "experiments": ["011-extensions-vs-bare", "015-35b-extensions"],
      "delta": -2.33,
      "verdict": "Hurt 9B, neutral 35B"
    }
  ]
}
```

---

## 8. LAB-BOOK Parsing

The LAB-BOOK.md files follow a consistent format. The Go backend extracts structured data using simple regex/string parsing:

| Field | Extraction Method |
|-------|-------------------|
| Hypothesis | Line matching `**Hypothesis:**` — take text after colon |
| Result/Decision | Section under `## Decision` — scan for PASS/REJECT/NEUTRAL keywords |
| Per-run B1–B4 scores | Markdown table under `## Results` — parse pipe-delimited rows |
| Mean scores | Row starting with `**Mean**` or `Mean` in results table |
| Date | `**Date:**` field |
| Model | `**Model:**` field |

**Edge cases:**
- Some experiments have multiple results tables (comparison tables). Use the first table under `## Results` or `### Results Table`.
- B1–B4 column headers may vary slightly. Match by position (columns 2–5 after Run column).

---

## 9. Transcript Viewer — Detailed Spec

This is the most complex component. The JSONL files are 30–40 MB each.

### Parsing Strategy (Go backend)

1. Read `raw-output.jsonl` line by line
2. Maintain state machine:
   - `session` event → extract session metadata
   - `turn_start` → increment turn counter
   - `message_start` (role=user) → begin user message
   - `message_end` (role=user) → finalize user message content
   - `message_start` (role=assistant) → begin assistant message, extract usage
   - `message_update` (thinking_start) → begin thinking accumulator
   - `message_update` (thinking_delta) → append to thinking
   - `message_update` (thinking_end) → finalize thinking
   - `message_update` (text delta) → append to assistant content
   - `tool_call` → record tool name, input, callID
   - `tool_result` → match by callID, attach output
   - `message_end` (role=assistant) → finalize assistant message with tool calls
3. Emit structured events (as defined in API section)
4. Support offset/limit for pagination

### Rendering Strategy (React frontend)

- Virtual scrolling (only render visible events) — transcripts can have 800+ events
- Thinking blocks: collapsed by default, gray italic, click to expand
- Tool calls: rendered inline under assistant message
  - Header: tool name + input summary (file path for read, command for bash)
  - Body: collapsible, shows full input JSON + output text
  - Output truncated to 50 lines, "Show all" button
- User messages: distinct background color, full width
- Assistant messages: distinct background, may contain multiple tool calls
- Turn separators: horizontal rule with "Turn N" badge
- Token counter per assistant message (from usage field)

### Performance

- Initial load: first 50 events (covers ~2-3 turns)
- Scroll down → fetch next 50 events
- Backend caches parsed transcript in memory for duration of request session
- Frontend accumulates events in Zustand store

---

## 10. What to Reuse

### From Orbit-UI

| Asset | Reuse? | Notes |
|-------|--------|-------|
| Vite config pattern | Yes | Port config, proxy setup, build output |
| Zustand store pattern | Yes | Same state management approach |
| CSS variables / palette | Yes | Warm palette (`--bg`, `--ink`, `--brand`) |
| Responsive CSS grid | Yes | Mobile-first breakpoints |
| WebSocket architecture | No | Dashboard reads static files, not live streams |
| Express server | No | Using Go backend instead |
| Component code | No | All components are Orbit-specific |

### From Polis-Command

| Asset | Reuse? | Notes |
|-------|--------|-------|
| JSONL parsing patterns | Conceptually | Different event schema, but same approach |
| SQLite indexer | No | Overkill for <100 runs. File-based is fine. |
| REST API patterns | Conceptually | Similar endpoint structure |
| Vanilla JS SPA | No | Using React |
| Ask/LLM feature | No | Not needed for v1 |

### Built Fresh

- Go HTTP server + API handlers
- LAB-BOOK.md parser
- JSONL transcript parser + paginator
- All React components
- Cross-experiment analysis engine

---

## 11. Build & Development Workflow

### Development

```bash
# Terminal 1: Go backend with hot reload
cd /home/polis/tools/chiron
go run ./cmd dashboard --dev --port 4200

# Terminal 2: Vite dev server (proxies API to Go)
cd /home/polis/tools/chiron/dashboard
npm run dev    # localhost:4201, proxies /api → localhost:4200
```

### Production Build

```bash
cd /home/polis/tools/chiron/dashboard
npm run build              # → dashboard/dist/

cd /home/polis/tools/chiron
go build -o bin/chiron .   # embeds dashboard/dist/ via go:embed
```

### Usage

```bash
chiron dashboard                    # Start on default port 4200
chiron dashboard --port 8080        # Custom port
chiron dashboard --experiments ./experiments  # Custom path (default: ./experiments)
```

Opens browser automatically. Runs until ctrl+C.

---

## 12. Implementation Phases

### Phase 1 — Skeleton (backend + L1 overview)

- [ ] Add `dashboard` Cobra subcommand
- [ ] Go HTTP server with `net/http`
- [ ] `GET /api/v1/experiments` — scan dirs, parse YAML, return list
- [ ] Vite + React scaffold in `dashboard/`
- [ ] `ExperimentTable` component with TanStack Table
- [ ] `go:embed` for production build
- [ ] Basic CSS (reuse Orbit palette)

### Phase 2 — Experiment Detail (L2)

- [ ] `GET /api/v1/experiments/:id` — full experiment data
- [ ] LAB-BOOK.md parser (hypothesis, result, scores)
- [ ] `ExperimentDetail` layout (lab book + run matrix)
- [ ] `LabBook` component (marked rendering)
- [ ] `RunMatrix` component with heatmap coloring
- [ ] Config drawer (YAML + prompt display)

### Phase 3 — Run Detail (L3)

- [ ] `GET /api/v1/experiments/:id/runs/:runId/transcript` with pagination
- [ ] JSONL state machine parser in Go
- [ ] `TranscriptViewer` with virtual scrolling, collapsible thinking/tools
- [ ] `GET /api/v1/experiments/:id/runs/:runId/diff`
- [ ] `DiffViewer` with diff2html
- [ ] `GET /api/v1/experiments/:id/runs/:runId/br-log`
- [ ] `BrLog` table component
- [ ] `ScoreCard` component

### Phase 4 — Cross-Analysis (L4)

- [ ] `GET /api/v1/analysis` — cross-experiment aggregation
- [ ] `ScoreTrend` line chart (Recharts)
- [ ] `DimensionHeatmap` (Recharts)
- [ ] `EfficiencyScatter` (Recharts)
- [ ] Variable impact table

### Phase 5 — Polish

- [ ] Search within transcript
- [ ] URL-based routing (shareable links to specific runs)
- [ ] Keyboard navigation (j/k for next/prev run)
- [ ] Light/dark mode toggle
- [ ] Export run comparison as markdown (for pasting into LAB-BOOK)

---

## 13. Non-Goals (v1)

- **No live experiment monitoring** — Chiron runs are batch; watch the terminal.
- **No database** — file-based is sufficient for <1000 runs.
- **No authentication** — local only.
- **No cloud deployment** — WSL2 / localhost.
- **No editing** — read-only dashboard. Scores are edited in LAB-BOOK.md files.
- **No LLM integration** — no "ask about this run" feature (could add later via polis-command's Ask pattern).

---

## 14. Open Questions

1. **B1–B4 source of truth:** Currently manual scores live only in LAB-BOOK.md tables. Should we also write a `scores.json` per run during manual scoring, so the dashboard doesn't depend on markdown parsing? This would be more reliable but adds a step.

2. **Experiment numbering:** Experiments 003-evolution and 003-mythology-impact-v2 share the 003 prefix. The dashboard should use the full directory name as ID, not just the number.

3. **Historical experiments:** EXP-003 has a different structure (no YAML config, different run layout). Should the dashboard support legacy formats or only experiments created by the current Chiron version?

4. **Auto-scorer vs manual scores:** Some experiments have `scores.json` from auto-scorers, others only have manual B1–B4 in LAB-BOOK.md. The dashboard should handle both, preferring manual scores when available.
