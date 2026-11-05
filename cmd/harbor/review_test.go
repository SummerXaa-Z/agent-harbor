package main

import (
	"strings"
	"testing"
)

func reviewRow(target, capability, decision string, flags ...string) traceRow {
	return traceRow{
		TargetID:     target,
		CapabilityID: capability,
		Decision:     decision,
		RiskFlags:    flags,
		CallerID:     "agt_caller",
	}
}

func TestAggregateReviewSuggestsOnlyUnblockedHighRisk(t *testing.T) {
	names := map[string]string{"cap_write": "tgt.update_ticket", "cap_read": "tgt.search", "cap_export": "tgt.export"}
	rows := []traceRow{
		reviewRow("tgt", "cap_write", "allowed", "write"),            // unblocked high-risk -> suggestion
		reviewRow("tgt", "cap_write", "allowed", "write"),            // counted twice
		reviewRow("tgt", "cap_export", "denied", "export"),           // high-risk but blocked -> no suggestion
		reviewRow("tgt", "cap_read", "allowed"),                      // no flags -> never suggested
	}
	aggregate := aggregateReview(rows, names)
	if aggregate.TotalCalls != 4 || aggregate.DeniedCalls != 1 {
		t.Fatalf("totals wrong: %+v", aggregate)
	}
	suggestions := strings.Join(aggregate.suggestions(), "\n")
	if !strings.Contains(suggestions, `"tgt.update_ticket"`) || !strings.Contains(suggestions, "2 call(s)") {
		t.Fatalf("write candidate missing: %s", suggestions)
	}
	if strings.Contains(suggestions, "export") || strings.Contains(suggestions, "search") {
		t.Fatalf("blocked or unflagged capabilities must not be suggested: %s", suggestions)
	}
}

func TestAggregateReviewHandlesUnknownCapabilityIds(t *testing.T) {
	aggregate := aggregateReview([]traceRow{reviewRow("tgt", "cap_unmapped", "allowed", "admin")}, map[string]string{})
	if aggregate.CapabilityUse["tgt.cap_unmapped"].Calls != 1 {
		t.Fatalf("unmapped ids fall back to target.id: %+v", aggregate.CapabilityUse)
	}
}
