# Sprint 6 Design: Runtime Metrics

## Scope

Sprint 6 adds an API-backed Runtime Signals loop. Trace rows remain the source of truth; metrics are computed in the HTTP layer by reading scoped traces and summarizing dimensions that already matter to operators.

## Trace Enrichment

`TraceEvent` gains four OTel-ready fields:

- `durationMs`: elapsed gateway data-plane duration in milliseconds.
- `upstreamAttempts`: final upstream attempt count for proxied calls.
- `upstreamStatus`: final upstream HTTP status when an upstream response exists.
- `upstreamError`: stable gateway error code for gateway-generated upstream failures.

Denied traces keep zero-value upstream fields. Allowed local stub traces also keep zero upstream fields because no upstream call was made.

## Proxy Recording

Allowed trace recording moves from "before proxy" to "after proxy result is known" for proxied targets. The proxy helper returns a small result object with handled/attempt/status/error/duration fields and still writes the HTTP response. `handleDataPlane` records the allowed trace after proxy completion so the trace remains append-only.

## Metrics API

`GET /api/v1/metrics/runtime` is an admin endpoint and accepts the same `tenantId` / `workspaceId` scope parameters as audit traces.

Response shape matches `frontend/src/types.ts` `SystemMetric[]`:

- `gateway_calls_total`: all scoped traces.
- `allowed_rate`: allowed traces divided by total traces.
- `upstream_error_rate`: allowed proxied traces with `upstreamError` divided by allowed proxied traces.
- `avg_latency_ms`: average `durationMs` across proxied traces with positive duration.

Metric `status` is thresholded conservatively:

- Allowed rate: healthy at `>=95`, warning at `>=80`, critical below.
- Upstream error rate: healthy at `<=1`, warning at `<=5`, critical above.
- Average latency: healthy at `<=300ms`, warning at `<=1000ms`, critical above.
- Gateway call volume: healthy when non-zero, warning when zero.

## Frontend

`loadConsoleData()` adds a fifth fetch for `/api/v1/metrics/runtime` and keeps `systemMetrics` as the network-failure fallback. `SignalBoard` can render the same metric shape unchanged.

## Follow-up

- Add OpenTelemetry spans/counters exporting these same dimensions.
- Add time-window query params and SQL aggregation after trace volume grows.
- Add per-route/caller/target drilldowns in the Signal Board.
