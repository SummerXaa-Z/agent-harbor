package store

import (
	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"testing"
	"time"
)

func TestMemoryTraceReceiptSurvivesViewRetention(t *testing.T) {
	repo := NewMemory()
	event := domain.TraceEvent{ID: "retained-receipt", Source: "wrap", Outcome: "ok", CreatedAt: time.Now(), RiskFlags: []string{"write"}}
	if err := repo.IngestTraces(t.Context(), []domain.TraceEvent{event}); err != nil {
		t.Fatal(err)
	}
	event.RiskFlags[0] = "changed"
	if repo.traces[len(repo.traces)-1].RiskFlags[0] != "write" {
		t.Fatal("mutable slice retained")
	}
	event.RiskFlags[0] = "write"
	repo.mu.Lock()
	repo.traces = nil
	repo.mu.Unlock()
	if err := repo.IngestTraces(t.Context(), []domain.TraceEvent{event}); err != nil {
		t.Fatal(err)
	}
	if len(repo.traces) != 0 {
		t.Fatal("receipt eviction duplicated event")
	}
	event.Outcome = "error"
	if err := repo.IngestTraces(t.Context(), []domain.TraceEvent{event}); err == nil {
		t.Fatal("receipt accepted altered evidence")
	}
}
