# Sprint 10 Route Policy Retry Overrides Design

## Problem

Retry behavior currently lives on the target Agent `channelConfig.retry`. That works for simple gateways, but RoutePolicy is now the primary governance object. Operators need to tune retries per caller-target-route without cloning target Agents or changing all consumers at once.

## Design

Add optional `retry` to `RoutePolicy`:

- `maxAttempts`: integer 1-4.
- `backoffMs`: integer 0-1000.
- `statusCodes`: 5xx integer list.

Create and patch normalize this object using the same defaults as Agent retry config: omitted `maxAttempts` defaults to 1, omitted `backoffMs` defaults to 0, omitted `statusCodes` defaults to `[502,503,504]`. A policy with no `retry` field does not override target Agent retry.

The repository `EvaluateRouteAccess` returns the retry override on the access decision when the winning policy is an allow policy. The HTTP data plane passes that override into the proxy. Proxy retry resolution becomes:

1. If the matched allow policy has retry, use it.
2. Otherwise use target Agent `channelConfig.retry`.
3. Otherwise use the existing no-retry default.

Legacy access grants continue to return no override.

## API

`POST /api/v1/route-policies` accepts:

```json
{
  "name": "Allow tool calls with bounded retry",
  "callerAgentId": "agt_caller",
  "targetAgentId": "agt_target",
  "routeType": "mcp",
  "routeKey": "tools/call",
  "effect": "allow",
  "priority": 100,
  "retry": {
    "maxAttempts": 3,
    "backoffMs": 50,
    "statusCodes": [502, 503, 504]
  }
}
```

`PATCH /api/v1/route-policies/{id}` can replace retry with an object or clear it with `"retry": null`.

## Testing

- HTTP test: policy retry overrides target retry and produces more upstream attempts.
- HTTP test: invalid policy retry is rejected on create and patch.
- Store test: PostgreSQL persists route policy retry and returns it in route access decisions.
- Demo: create target with no retry, create policy retry, prove two attempts.
