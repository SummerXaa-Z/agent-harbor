# Sprint 6 Brief: Runtime Metrics

Date: 2026-05-30
Status: Implementing on `codex/sprint-6-runtime-metrics`

## Goal

Turn Runtime Signals from local sample cards into backend-derived operational metrics, using data-plane traces as the first source of truth.

## User Stories

- As an operator, I can see gateway call volume and allowed/denied rate from real trace evidence.
- As an operator, I can see upstream error rate, average latency, and retry pressure without reading raw traces.
- As a developer, I can inspect each trace and know final upstream attempts, status, error code, and duration.
- As a frontend user, the Signal Board loads real runtime metrics when the Go API is reachable and keeps mock fallback only for backend-down local development.

## Acceptance

- `TraceEvent` includes optional `durationMs`, `upstreamAttempts`, `upstreamStatus`, and `upstreamError` fields.
- Proxied MCP/OpenAPI calls record attempts, final upstream status, stable upstream error code, and elapsed duration on the allowed trace.
- Denied calls still record immediately and participate in gateway call / allowed-rate metrics.
- `GET /api/v1/metrics/runtime?tenantId=&workspaceId=` returns `SystemMetric[]` in the frontend's existing shape.
- Runtime metrics are derived from scoped traces and include gateway calls, allowed rate, upstream error rate, and average upstream latency.
- The frontend loads runtime metrics from the API with mock fallback only on network failure.

## Non-goals

- No external OpenTelemetry exporter yet.
- No SQL-level aggregation or retention window yet.
- No per-route metrics drilldown UI yet.
