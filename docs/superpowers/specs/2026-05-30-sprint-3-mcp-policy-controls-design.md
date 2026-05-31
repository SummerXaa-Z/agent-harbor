# AgentHarbor Sprint 3 MCP Policy Controls Design

## Goal

Advance AgentHarbor's data-plane from coarse route authorization to protocol-aware MCP governance, while keeping the clean-room Go runtime small and testable.

## Scope

Sprint 3 adds three backend controls and one console usability improvement:

- MCP method policy: parse JSON-RPC request `method` and use that value as the route key for authorization and traces.
- Upstream headers: allow non-secret string headers in `channelConfig.headers` and forward them to MCP/OpenAPI upstreams.
- Proxy timeout: allow `channelConfig.timeoutMs` to bound upstream calls, returning `504 UPSTREAM_TIMEOUT` on deadline.
- Route key presets: expose common MCP route keys in the console create-grant form.

## API Behavior

- `POST /api/v1/mcp/agents/{targetId}` and `/rpc` inspect the JSON body before authorization.
- Valid MCP requests require a string `method`; empty, missing, or non-string methods return `400 VALIDATION_FAILED` and do not write a trace.
- The derived MCP route key is exactly the JSON-RPC method, for example `initialize`, `tools/list`, or `tools/call`.
- `AccessGrant.routeType=mcp, routeKey=tools/call` authorizes only `tools/call`; wildcard route keys still work by storing an empty route key.
- Stub responses continue to return `route: "mcp"` and now include the actual route key in the trace.

## Channel Config

`channelConfig.headers` is optional and must be a JSON object of string-to-string values. Header names are rejected when they look secret-bearing, such as `authorization`, `x-api-key`, `cookie`, `token`, or `secret`. Values are forwarded as-is to upstreams.

`channelConfig.timeoutMs` is optional. Default is 10000 ms. Minimum is 1 ms. Maximum is 30000 ms. Invalid values fail agent creation with `400 VALIDATION_FAILED`. Upstream requests run with the configured timeout layered on top of the incoming request context.

## Error Handling

- Malformed MCP JSON returns `400 VALIDATION_FAILED`.
- Missing/invalid MCP method returns `400 VALIDATION_FAILED`.
- Upstream timeout returns `504 UPSTREAM_TIMEOUT`.
- Other network failures keep Sprint 2 behavior: `502 UPSTREAM_ERROR`.
- Timeout/network failures still record an allowed trace first because authorization succeeded before upstream execution.

## Testing

- HTTP tests cover method-specific allow/deny, malformed MCP body, header forwarding, timeout validation, and timeout response.
- Existing demo scripts continue to pass.
- Frontend build verifies route key presets compile.
