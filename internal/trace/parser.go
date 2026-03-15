package trace

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// rawEvent is a loosely-typed representation of a JSONL event line.
type rawEvent struct {
	Type    string          `json:"type"`
	RawJSON json.RawMessage `json:"-"` // original bytes for forward compat
}

// sessionEvent captures the session-level metadata.
type sessionEvent struct {
	Type      string `json:"type"`
	Version   int    `json:"version"`
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	CWD       string `json:"cwd"`
}

// messageEvent captures message_start / message_end events.
type messageEvent struct {
	Type    string         `json:"type"`
	Message messagePayload `json:"message"`
}

type messagePayload struct {
	Role       string           `json:"role"`
	Content    []contentBlock   `json:"content"`
	API        string           `json:"api"`
	Provider   string           `json:"provider"`
	Model      string           `json:"model"`
	Usage      usagePayload     `json:"usage"`
	StopReason string           `json:"stopReason"`
	Timestamp  json.Number      `json:"timestamp"`
}

type contentBlock struct {
	Type     string `json:"type"` // text, thinking, tool_use, tool_result
	Text     string `json:"text,omitempty"`
	Thinking string `json:"thinking,omitempty"`
	Name     string `json:"name,omitempty"`    // tool name for tool_use
	Input    any    `json:"input,omitempty"`   // tool input
	Content  any    `json:"content,omitempty"` // tool result content
	IsError  bool   `json:"is_error,omitempty"`
}

type usagePayload struct {
	Input      int         `json:"input"`
	Output     int         `json:"output"`
	CacheRead  int         `json:"cacheRead"`
	CacheWrite int         `json:"cacheWrite"`
	Cost       costPayload `json:"cost"`
}

type costPayload struct {
	Total float64 `json:"total"`
}

// toolExecutionEvent captures tool_execution_start/end events.
// Pi uses toolName/toolCallId; the parser also accepts name for compatibility.
type toolExecutionEvent struct {
	Type       string `json:"type"`
	ToolName   string `json:"toolName"`
	Name       string `json:"name"`       // fallback field name
	ToolCallID string `json:"toolCallId"`
	IsError    bool   `json:"isError"`
	// These fields may appear in some formats.
	DurationMs int64               `json:"duration_ms"`
	ExitCode   int                 `json:"exit_code"`
	InputLen   int                 `json:"input_len"`
	OutputLen  int                 `json:"output_len"`
	Args       map[string]any      `json:"args"`
	Result     *toolResultPayload  `json:"result"`
}

type toolResultPayload struct {
	Content []contentBlock `json:"content"`
}

// toolExecName returns the best available tool name.
func (te *toolExecutionEvent) toolExecName() string {
	if te.ToolName != "" {
		return te.ToolName
	}
	return te.Name
}

// ParseFile parses a raw-output.jsonl file and returns a normalized RunTrace.
func ParseFile(path string) (*RunTrace, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening transcript: %w", err)
	}
	defer f.Close()
	return Parse(f, filepath.Base(filepath.Dir(path)))
}

// Parse reads JSONL from r and returns a normalized RunTrace.
// runID is used as the trace's RunID (typically the run directory name).
func Parse(r io.Reader, runID string) (*RunTrace, error) {
	trace := &RunTrace{
		SchemaVersion: SchemaVersion,
		RunID:         runID,
		Metrics: RunMetrics{
			ToolCallsByName:  map[string]int{},
			FirstEditTurnIdx: -1,
		},
	}

	scanner := bufio.NewScanner(r)
	// Allow large lines (some message_update events embed full content).
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	var (
		currentTurn *Turn
		turnIdx     int
		lineNum     int
	)

	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var evt rawEvent
		if err := json.Unmarshal(line, &evt); err != nil {
			trace.Warnings = append(trace.Warnings, fmt.Sprintf("line %d: malformed JSON: %v", lineNum, err))
			continue
		}
		evt.RawJSON = append(json.RawMessage(nil), line...)

		switch evt.Type {
		case "session":
			var se sessionEvent
			if err := json.Unmarshal(line, &se); err == nil {
				trace.SessionID = se.ID
				if t, err := time.Parse(time.RFC3339Nano, se.Timestamp); err == nil {
					trace.StartedAt = t
				}
			}

		case "turn_start":
			currentTurn = &Turn{Index: turnIdx}
			if trace.StartedAt.IsZero() {
				trace.StartedAt = time.Now()
			}

		case "turn_end":
			if currentTurn != nil {
				trace.Turns = append(trace.Turns, *currentTurn)
				turnIdx++
			}
			currentTurn = nil

		case "message_start", "message_end":
			var me messageEvent
			if err := json.Unmarshal(line, &me); err != nil {
				continue
			}
			if currentTurn == nil {
				// Message outside a turn — create an implicit turn.
				currentTurn = &Turn{Index: turnIdx}
			}
			msg := Message{
				Role:       me.Message.Role,
				StopReason: me.Message.StopReason,
			}
			for _, cb := range me.Message.Content {
				switch cb.Type {
				case "text":
					msg.HasText = true
					msg.TextLen += len(cb.Text)
				case "thinking":
					msg.HasThink = true
				case "tool_use":
					// Tool calls from message content are recorded as lightweight
					// entries. If tool_execution_end events also exist, those
					// provide richer data and are counted there instead.
					// We track names here for cases where tool_execution events
					// are absent (e.g., older transcript formats).
					msg.StopReason = "tool_use"
				case "tool_result":
					// Handled via tool_execution events when available.
				}
			}

			// Only record message_end (which has final content).
			if evt.Type == "message_end" {
				currentTurn.Messages = append(currentTurn.Messages, msg)
			}

			// Extract model/provider from first assistant message.
			if me.Message.Role == "assistant" && trace.Model == "" {
				trace.Model = me.Message.Model
				trace.Provider = me.Message.Provider
			}

			// Accumulate usage from message_end.
			if evt.Type == "message_end" && me.Message.Role == "assistant" {
				currentTurn.Usage.InputTokens += me.Message.Usage.Input
				currentTurn.Usage.OutputTokens += me.Message.Usage.Output
				currentTurn.Usage.CacheReadTokens += me.Message.Usage.CacheRead
				currentTurn.Usage.CacheWriteTokens += me.Message.Usage.CacheWrite
				currentTurn.Usage.CostUSD += me.Message.Usage.Cost.Total
			}

		case "tool_execution_start", "tool_execution_end":
			var te toolExecutionEvent
			if err := json.Unmarshal(line, &te); err != nil {
				continue
			}
			name := te.toolExecName()
			if currentTurn != nil && evt.Type == "tool_execution_end" {
				tc := ToolCall{
					Name:       name,
					DurationMs: te.DurationMs,
					ExitCode:   te.ExitCode,
					IsError:    te.IsError,
					InputLen:   te.InputLen,
					OutputLen:  te.OutputLen,
				}
				if te.Result != nil {
					for _, c := range te.Result.Content {
						tc.OutputLen += len(c.Text)
					}
				}
				if te.Args != nil {
					if b, err := json.Marshal(te.Args); err == nil {
						tc.InputLen = len(b)
					}
				}
				currentTurn.ToolCalls = append(currentTurn.ToolCalls, tc)
				trace.Metrics.TotalToolCalls++
				trace.Metrics.ToolCallsByName[name]++
				classifyTool(trace, name, turnIdx)
			}

		case "agent_start":
			// No action needed — session start is sufficient.

		case "agent_end":
			// Captures the final timestamp.
			trace.FinishedAt = time.Now()

		case "message_update":
			// Streaming deltas — skip for trace purposes (content captured in message_end).

		default:
			// Unknown event type — record as warning for forward compatibility.
			// Don't warn on every streaming delta.
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading transcript: %w", err)
	}

	// Close any unclosed turn.
	if currentTurn != nil {
		trace.Turns = append(trace.Turns, *currentTurn)
	}

	// Finalize metrics.
	trace.Metrics.TotalTurns = len(trace.Turns)
	for _, t := range trace.Turns {
		trace.Metrics.TotalTokensIn += t.Usage.InputTokens
		trace.Metrics.TotalTokensOut += t.Usage.OutputTokens
		trace.Metrics.TotalCostUSD += t.Usage.CostUSD
	}

	// Determine outcome.
	trace.Outcome = classifyOutcome(trace)

	// Duration from timestamps.
	if !trace.StartedAt.IsZero() && !trace.FinishedAt.IsZero() {
		trace.DurationMs = trace.FinishedAt.Sub(trace.StartedAt).Milliseconds()
	}

	return trace, nil
}

// TranscriptPath returns the path to the transcript file in a run directory.
// It prefers transcript.jsonl (canonical) and falls back to raw-output.jsonl (legacy).
func TranscriptPath(runDir string) string {
	canonical := filepath.Join(runDir, "transcript.jsonl")
	if _, err := os.Stat(canonical); err == nil {
		return canonical
	}
	return filepath.Join(runDir, "raw-output.jsonl")
}

// ParseFileWithMeta parses the transcript and enriches the trace with meta.json + result.json.
func ParseFileWithMeta(runDir string) (*RunTrace, error) {
	jsonlPath := TranscriptPath(runDir)
	trace, err := ParseFile(jsonlPath)
	if err != nil {
		return nil, err
	}

	// Enrich from meta.json.
	metaPath := filepath.Join(runDir, "meta.json")
	if data, err := os.ReadFile(metaPath); err == nil {
		var meta map[string]any
		if json.Unmarshal(data, &meta) == nil {
			if m, ok := meta["model"].(string); ok && trace.Model == "" {
				trace.Model = m
			}
			if ts, ok := meta["timestamp"].(string); ok {
				if t, err := time.Parse(time.RFC3339, ts); err == nil {
					trace.FinishedAt = t
				}
			}
			if ms, ok := meta["duration_ms"].(float64); ok {
				trace.DurationMs = int64(ms)
				if !trace.FinishedAt.IsZero() {
					trace.StartedAt = trace.FinishedAt.Add(-time.Duration(int64(ms)) * time.Millisecond)
				}
			}
			if ti, ok := meta["tokens_in"].(float64); ok {
				trace.Metrics.TotalTokensIn = int(ti)
			}
			if to, ok := meta["tokens_out"].(float64); ok {
				trace.Metrics.TotalTokensOut = int(to)
			}
		}
	}

	// Enrich from result.json.
	resultPath := filepath.Join(runDir, "result.json")
	if data, err := os.ReadFile(resultPath); err == nil {
		var result map[string]any
		if json.Unmarshal(data, &result) == nil {
			if ec, ok := result["exit_code"].(float64); ok && ec != 0 {
				trace.Outcome = "failure"
			}
			if cond, ok := result["condition"].(string); ok {
				trace.RunID = fmt.Sprintf("%s/%s", trace.Model, cond)
			}
		}
	}

	// Enrich from scores.json.
	scoresPath := filepath.Join(runDir, "scores.json")
	if data, err := os.ReadFile(scoresPath); err == nil {
		var scores map[string]any
		if json.Unmarshal(data, &scores) == nil {
			// Scores available — attach as metadata but don't modify outcome.
		}
	}

	return trace, nil
}

// classifyTool updates metrics based on tool name.
func classifyTool(trace *RunTrace, name string, turnIdx int) {
	lower := strings.ToLower(name)
	switch {
	case lower == "edit" || lower == "write" || lower == "notedit" || lower == "notebookedit":
		trace.Metrics.EditCount++
		if trace.Metrics.FirstEditTurnIdx == -1 {
			trace.Metrics.FirstEditTurnIdx = turnIdx
		}
	case lower == "read" || lower == "glob" || lower == "grep" || lower == "ls":
		trace.Metrics.ReadCount++
	}
}

// classifyOutcome determines the run outcome from the trace.
func classifyOutcome(trace *RunTrace) string {
	if len(trace.Turns) == 0 {
		return "abandoned"
	}
	// If we have no edits, it might be a read-only failure.
	if trace.Metrics.EditCount == 0 {
		return "failure"
	}
	// Default to success — overridden by result.json exit_code in ParseFileWithMeta.
	return "success"
}
