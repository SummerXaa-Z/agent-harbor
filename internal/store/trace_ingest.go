package store

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/jackc/pgx/v5"
)

// Canonicalize representation, not evidence: unknown historical outcomes stay blank.
func normalizeTrace(event domain.TraceEvent) domain.TraceEvent {
	if event.Source == "" {
		event.Source = "gateway"
	}
	event.CreatedAt = event.CreatedAt.UTC()
	if len(event.RiskFlags) == 0 {
		event.RiskFlags = nil
	}
	if len(event.DataScopes) == 0 {
		event.DataScopes = nil
	}
	return event
}
func traceCollision() error {
	return domain.Conflict("TRACE_ID_CONFLICT", "trace id is already associated with different evidence")
}

// IngestTraces commits all or none. Receipts outlive the bounded visible history,
// but are volatile like all memory-store state.
func (m *Memory) IngestTraces(_ context.Context, events []domain.TraceEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	pending := make(map[string][]byte)
	additions := make([]domain.TraceEvent, 0, len(events))
	for _, event := range events {
		event = normalizeTrace(event)
		payload, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("marshal ingested trace: %w", err)
		}
		old, exists := pending[event.ID]
		if !exists {
			old, exists = m.traceReceipts[event.ID]
		}
		if exists {
			if !bytes.Equal(old, payload) {
				return traceCollision()
			}
			continue
		}
		// Never adopt an existing gateway ID.
		for _, existing := range m.traces {
			if existing.ID == event.ID {
				return traceCollision()
			}
		}
		pending[event.ID] = payload
		var copy domain.TraceEvent
		if err := json.Unmarshal(payload, &copy); err != nil {
			return err
		}
		additions = append(additions, copy)
	}
	if m.traceReceipts == nil {
		m.traceReceipts = make(map[string][]byte)
	}
	for _, event := range additions {
		m.traceReceipts[event.ID] = pending[event.ID]
		m.appendTraceLocked(event)
	}
	return nil
}
func (p *Postgres) IngestTraces(ctx context.Context, events []domain.TraceEvent) error {
	return p.withTx(ctx, func(tx pgx.Tx) error {
		for _, event := range events {
			event = normalizeTrace(event)
			payload, err := json.Marshal(event)
			if err != nil {
				return fmt.Errorf("marshal ingested trace: %w", err)
			}
			inserted, err := insertTrace(ctx, tx, event, payload, true)
			if err != nil {
				return err
			}
			if inserted {
				continue
			}
			// ON CONFLICT waits for concurrent writers; READ COMMITTED then sees receipt.
			var identical bool
			if err := tx.QueryRow(ctx, "select coalesce(ingest_payload = $2::jsonb, false) from trace_events where id = $1", event.ID, payload).Scan(&identical); err != nil {
				return fmt.Errorf("read trace receipt: %w", err)
			}
			if !identical {
				return traceCollision()
			}
		}
		return nil
	})
}
