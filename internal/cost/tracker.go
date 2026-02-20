package cost

import (
	"fmt"
	"sync"
	"time"
)

// Event records a single cost-generating operation.
type Event struct {
	Operation  string  `json:"operation"` // "generate", "execute", "mutate", "challenge"
	Model      string  `json:"model"`
	TokensIn   int     `json:"tokens_in"`
	TokensOut  int     `json:"tokens_out"`
	CostUSD    float64 `json:"cost_usd"`
	DurationMS int     `json:"duration_ms"`
	Timestamp  string  `json:"timestamp"`
}

// Summary aggregates cost data for reporting.
type Summary struct {
	TotalCostUSD    float64            `json:"total_cost_usd"`
	TotalTokensIn   int                `json:"total_tokens_in"`
	TotalTokensOut  int                `json:"total_tokens_out"`
	TotalDurationMS int                `json:"total_duration_ms"`
	EventCount      int                `json:"event_count"`
	ByOperation     map[string]float64 `json:"by_operation"`
	ByModel         map[string]float64 `json:"by_model"`
	BudgetUSD       float64            `json:"budget_usd"`
	Remaining       float64            `json:"remaining"`
	OverBudget      bool               `json:"over_budget"`
}

// Tracker monitors costs and enforces budgets.
type Tracker struct {
	mu        sync.Mutex
	events    []Event
	budgetUSD float64
}

// New creates a cost tracker with a budget.
func New(budgetUSD float64) *Tracker {
	return &Tracker{
		events:    []Event{},
		budgetUSD: budgetUSD,
	}
}

// Record adds a cost event to the tracker.
func (t *Tracker) Record(event Event) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if event.Timestamp == "" {
		event.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	t.events = append(t.events, event)
}

// TotalCost returns the current total cost.
func (t *Tracker) TotalCost() float64 {
	t.mu.Lock()
	defer t.mu.Unlock()

	var total float64
	for _, e := range t.events {
		total += e.CostUSD
	}
	return total
}

// Remaining returns how much budget is left.
func (t *Tracker) Remaining() float64 {
	return t.budgetUSD - t.TotalCost()
}

// OverBudget returns whether total cost exceeds budget.
func (t *Tracker) OverBudget() bool {
	return t.budgetUSD > 0 && t.TotalCost() > t.budgetUSD
}

// CheckBudget returns an error if the budget would be exceeded.
func (t *Tracker) CheckBudget(estimatedCostUSD float64) error {
	if t.budgetUSD <= 0 {
		return nil // no budget set
	}
	projected := t.TotalCost() + estimatedCostUSD
	if projected > t.budgetUSD {
		return fmt.Errorf("budget exceeded: projected $%.4f > budget $%.4f (remaining: $%.4f)",
			projected, t.budgetUSD, t.Remaining())
	}
	return nil
}

// Summarize returns an aggregate cost report.
func (t *Tracker) Summarize() Summary {
	t.mu.Lock()
	defer t.mu.Unlock()

	s := Summary{
		ByOperation: map[string]float64{},
		ByModel:     map[string]float64{},
		BudgetUSD:   t.budgetUSD,
	}

	for _, e := range t.events {
		s.TotalCostUSD += e.CostUSD
		s.TotalTokensIn += e.TokensIn
		s.TotalTokensOut += e.TokensOut
		s.TotalDurationMS += e.DurationMS
		s.EventCount++
		s.ByOperation[e.Operation] += e.CostUSD
		if e.Model != "" {
			s.ByModel[e.Model] += e.CostUSD
		}
	}

	s.Remaining = t.budgetUSD - s.TotalCostUSD
	s.OverBudget = t.budgetUSD > 0 && s.TotalCostUSD > t.budgetUSD

	return s
}

// Events returns a copy of all recorded events.
func (t *Tracker) Events() []Event {
	t.mu.Lock()
	defer t.mu.Unlock()

	out := make([]Event, len(t.events))
	copy(out, t.events)
	return out
}

// Reset clears all events (keeps budget).
func (t *Tracker) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.events = []Event{}
}
