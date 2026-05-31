# Sprint 5 Design: Proxy Retry and Error Classification

## Scope

Sprint 5 extends the target Agent proxy controls with a bounded retry policy and more precise gateway error classification. It applies to MCP and OpenAPI upstream proxy calls after authorization has already succeeded.

## Channel Config

Target `channelConfig.retry` is optional:

```json
{
  "endpoint": "https://api.example.com/mcp",
  "retry": {
    "maxAttempts": 3,
    "backoffMs": 50,
    "statusCodes": [502, 503, 504]
  }
}
```

Defaults:

- `maxAttempts`: `1`
- `backoffMs`: `0`
- `statusCodes`: `[502, 503, 504]`

Validation:

- `retry` must be an object when present.
- `maxAttempts` must be an integer from 1 to 4.
- `backoffMs` must be an integer from 0 to 1000.
- `statusCodes` must be an array of integer HTTP status codes from 500 to 599.

## Proxy Behavior

The proxy reads the inbound request body into memory once and creates a fresh upstream request per attempt. Buffered bodies are capped at 4MiB; larger proxied calls return `413 PAYLOAD_TOO_LARGE` before hitting upstream. Retry happens only after authorization and trace recording.

Retryable cases:

- Network errors before an upstream response is received.
- Upstream responses whose status code is in `retry.statusCodes`.

Non-retry cases:

- Timeout after request context deadline or `timeoutMs`.
- Canceled request contexts.
- Non-retryable upstream status codes.
- Invalid upstream URL preparation.

Every proxied upstream response includes `X-AgentHarbor-Upstream-Attempts`. If retries are exhausted on retryable status responses, the gateway returns the last upstream response as-is.

## Error Classification

Gateway-generated proxy failures use stable error codes:

- `UPSTREAM_TIMEOUT` for request deadline exhaustion.
- `UPSTREAM_DNS_ERROR` for DNS lookup failures.
- `UPSTREAM_TLS_ERROR` for TLS handshake or certificate verification failures.
- `UPSTREAM_CONNECT_ERROR` for connection refused/reset/closed style connect failures.
- `UPSTREAM_ERROR` as a fallback for unclassified network failures.

## Follow-up

- Add OTel spans and counters with route, caller, target, attempt count, and classification dimensions.
- Add route-level retry overrides after route policy objects exist.
- Add circuit breaker semantics only after real production traffic requirements justify them.
