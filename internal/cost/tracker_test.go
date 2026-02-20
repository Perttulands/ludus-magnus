package cost

import (
	"testing"
)

func TestNewTracker(t *testing.T) {
	tracker := New(10.0)
	if tracker.TotalCost() != 0 {
		t.Errorf("new tracker should have 0 cost, got %f", tracker.TotalCost())
	}
	if tracker.Remaining() != 10.0 {
		t.Errorf("remaining = %f, want 10.0", tracker.Remaining())
	}
}

func TestRecord(t *testing.T) {
	tracker := New(10.0)
	tracker.Record(Event{
		Operation: "generate",
		Model:     "claude-sonnet-4-5",
		TokensIn:  1000,
		TokensOut: 500,
		CostUSD:   0.50,
	})

	if tracker.TotalCost() != 0.50 {
		t.Errorf("total cost = %f, want 0.50", tracker.TotalCost())
	}
	if tracker.Remaining() != 9.50 {
		t.Errorf("remaining = %f, want 9.50", tracker.Remaining())
	}
}

func TestMultipleRecords(t *testing.T) {
	tracker := New(5.0)
	tracker.Record(Event{Operation: "generate", CostUSD: 1.0})
	tracker.Record(Event{Operation: "execute", CostUSD: 2.0})
	tracker.Record(Event{Operation: "mutate", CostUSD: 0.5})

	if tracker.TotalCost() != 3.5 {
		t.Errorf("total cost = %f, want 3.5", tracker.TotalCost())
	}
}

func TestOverBudget(t *testing.T) {
	tracker := New(1.0)
	if tracker.OverBudget() {
		t.Error("should not be over budget initially")
	}

	tracker.Record(Event{CostUSD: 1.5})
	if !tracker.OverBudget() {
		t.Error("should be over budget after exceeding")
	}
}

func TestOverBudgetNoBudget(t *testing.T) {
	tracker := New(0) // no budget
	tracker.Record(Event{CostUSD: 1000.0})
	if tracker.OverBudget() {
		t.Error("should never be over budget when budget is 0 (unlimited)")
	}
}

func TestCheckBudget(t *testing.T) {
	tracker := New(5.0)
	tracker.Record(Event{CostUSD: 3.0})

	if err := tracker.CheckBudget(1.0); err != nil {
		t.Errorf("should be within budget: %v", err)
	}

	if err := tracker.CheckBudget(3.0); err == nil {
		t.Error("should exceed budget")
	}
}

func TestCheckBudgetNoBudget(t *testing.T) {
	tracker := New(0)
	if err := tracker.CheckBudget(1000.0); err != nil {
		t.Errorf("no budget should always pass: %v", err)
	}
}

func TestSummarize(t *testing.T) {
	tracker := New(10.0)
	tracker.Record(Event{Operation: "generate", Model: "claude-sonnet-4-5", CostUSD: 1.0, TokensIn: 100, TokensOut: 200, DurationMS: 500})
	tracker.Record(Event{Operation: "execute", Model: "claude-sonnet-4-5", CostUSD: 2.0, TokensIn: 300, TokensOut: 400, DurationMS: 1000})
	tracker.Record(Event{Operation: "generate", Model: "claude-opus-4-6", CostUSD: 3.0, TokensIn: 50, TokensOut: 100, DurationMS: 2000})

	s := tracker.Summarize()

	if s.TotalCostUSD != 6.0 {
		t.Errorf("TotalCostUSD = %f, want 6.0", s.TotalCostUSD)
	}
	if s.TotalTokensIn != 450 {
		t.Errorf("TotalTokensIn = %d, want 450", s.TotalTokensIn)
	}
	if s.TotalTokensOut != 700 {
		t.Errorf("TotalTokensOut = %d, want 700", s.TotalTokensOut)
	}
	if s.EventCount != 3 {
		t.Errorf("EventCount = %d, want 3", s.EventCount)
	}
	if s.ByOperation["generate"] != 4.0 {
		t.Errorf("generate cost = %f, want 4.0", s.ByOperation["generate"])
	}
	if s.ByOperation["execute"] != 2.0 {
		t.Errorf("execute cost = %f, want 2.0", s.ByOperation["execute"])
	}
	if s.ByModel["claude-sonnet-4-5"] != 3.0 {
		t.Errorf("sonnet cost = %f, want 3.0", s.ByModel["claude-sonnet-4-5"])
	}
	if s.Remaining != 4.0 {
		t.Errorf("Remaining = %f, want 4.0", s.Remaining)
	}
	if s.OverBudget {
		t.Error("should not be over budget")
	}
}

func TestEvents(t *testing.T) {
	tracker := New(10.0)
	tracker.Record(Event{Operation: "generate", CostUSD: 1.0})
	tracker.Record(Event{Operation: "execute", CostUSD: 2.0})

	events := tracker.Events()
	if len(events) != 2 {
		t.Errorf("expected 2 events, got %d", len(events))
	}
}

func TestReset(t *testing.T) {
	tracker := New(10.0)
	tracker.Record(Event{CostUSD: 5.0})
	tracker.Reset()

	if tracker.TotalCost() != 0 {
		t.Errorf("cost after reset = %f, want 0", tracker.TotalCost())
	}
	if len(tracker.Events()) != 0 {
		t.Errorf("events after reset = %d, want 0", len(tracker.Events()))
	}
}

func TestRecordAutoTimestamp(t *testing.T) {
	tracker := New(10.0)
	tracker.Record(Event{Operation: "test"})

	events := tracker.Events()
	if events[0].Timestamp == "" {
		t.Error("expected auto-generated timestamp")
	}
}
