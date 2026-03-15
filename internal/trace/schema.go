// Package trace provides a normalized trace schema and parser for Pi JSONL transcripts.
//
// The schema captures the structure of a single experiment run: session metadata,
// per-turn messages and tool calls, usage metrics, and outcome classification.
// The parser is streaming and preserves unknown event types for forward compatibility.
package trace

import "time"

// SchemaVersion is the current trace schema version.
const SchemaVersion = 1

// RunTrace is the canonical normalized trace for a single experiment run.
type RunTrace struct {
	SchemaVersion int            `json:"schema_version"`
	RunID         string         `json:"run_id"`
	SessionID     string         `json:"session_id,omitempty"`
	Model         string         `json:"model,omitempty"`
	Provider      string         `json:"provider,omitempty"`
	StartedAt     time.Time      `json:"started_at"`
	FinishedAt    time.Time      `json:"finished_at"`
	DurationMs    int64          `json:"duration_ms"`
	Turns         []Turn         `json:"turns"`
	Outcome       string         `json:"outcome"` // success, failure, abandoned, error
	Metrics       RunMetrics     `json:"metrics"`
	FailureTags   []FailureTag   `json:"failure_tags,omitempty"`
	Extensions    []ExtensionRef `json:"extensions,omitempty"`
	Warnings      []string       `json:"warnings,omitempty"`
}

// Turn represents a single conversational turn (user → assistant round-trip).
type Turn struct {
	Index      int        `json:"index"`
	StartedAt  time.Time  `json:"started_at,omitempty"`
	FinishedAt time.Time  `json:"finished_at,omitempty"`
	DurationMs int64      `json:"duration_ms,omitempty"`
	Messages   []Message  `json:"messages"`
	ToolCalls  []ToolCall `json:"tool_calls"`
	Usage      TurnUsage  `json:"usage"`
}

// Message is a single message within a turn.
type Message struct {
	Role       string `json:"role"`
	HasText    bool   `json:"has_text"`
	TextLen    int    `json:"text_len,omitempty"`
	HasThink   bool   `json:"has_thinking,omitempty"`
	StopReason string `json:"stop_reason,omitempty"`
}

// ToolCall represents a tool invocation extracted from the transcript.
type ToolCall struct {
	Name       string `json:"name"`
	DurationMs int64  `json:"duration_ms,omitempty"`
	ExitCode   int    `json:"exit_code,omitempty"`
	IsError    bool   `json:"is_error,omitempty"`
	InputLen   int    `json:"input_len,omitempty"`
	OutputLen  int    `json:"output_len,omitempty"`
}

// TurnUsage captures token usage for a turn.
type TurnUsage struct {
	InputTokens      int     `json:"input_tokens"`
	OutputTokens     int     `json:"output_tokens"`
	CacheReadTokens  int     `json:"cache_read_tokens,omitempty"`
	CacheWriteTokens int     `json:"cache_write_tokens,omitempty"`
	CostUSD          float64 `json:"cost_usd,omitempty"`
}

// RunMetrics aggregates metrics across the entire run.
type RunMetrics struct {
	TotalTurns       int            `json:"total_turns"`
	TotalTokensIn    int            `json:"total_tokens_in"`
	TotalTokensOut   int            `json:"total_tokens_out"`
	TotalToolCalls   int            `json:"total_tool_calls"`
	ToolCallsByName  map[string]int `json:"tool_calls_by_name"`
	EditCount        int            `json:"edit_count"`
	TestRunCount     int            `json:"test_run_count"`
	ReadCount        int            `json:"read_count"`
	TotalCostUSD     float64        `json:"total_cost_usd,omitempty"`
	FirstEditTurnIdx int            `json:"first_edit_turn_idx"` // -1 if no edits
	TestAfterEdit    bool           `json:"test_after_edit"`
}

// FailureTag labels a detected failure pattern with evidence.
type FailureTag struct {
	Tag      string `json:"tag"`
	Evidence string `json:"evidence"`
	TurnIdx  int    `json:"turn_idx,omitempty"`
}

// ExtensionRef records evidence of an extension or intervention in the transcript.
type ExtensionRef struct {
	Name     string `json:"name"`
	Evidence string `json:"evidence"` // explicit or inferred
	TurnIdx  int    `json:"turn_idx,omitempty"`
	Detail   string `json:"detail,omitempty"`
}
