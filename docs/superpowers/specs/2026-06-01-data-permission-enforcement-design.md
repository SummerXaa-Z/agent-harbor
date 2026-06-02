# Data Permission Enforcement Design

Date: 2026-06-01

## Goal

Build the next governance loop after tenant-scoped MCP capability control: make `dataScopes` enforceable instead of only descriptive.

The MVP should prevent privilege expansion in the control plane, compute the final effective data scopes at runtime, forward those scopes to downstream MCP targets as trusted context, and record the same scopes in trace evidence.

## Product Journey

The target user journey is:

1. A platform admin reviews a discovered MCP tool capability.
2. The admin assigns broad but bounded `dataScopes` to the tenant entitlement.
3. A tenant or workspace admin narrows those scopes for a workspace.
4. An operator narrows or inherits scopes for a caller instance and optional subject selector.
5. Runtime evaluates capability access and computes final effective scopes.
6. AgentHarbor forwards trusted scope context to the downstream MCP target.
7. Trace evidence shows which scopes were applied to the call.

This turns the current model from "tool permission with scope metadata" into "tool permission plus enforceable data boundary".

## Recommended MVP

Use **control-plane narrowing plus runtime context injection**.

This is deliberately narrower than argument rewriting. AgentHarbor should not guess every MCP tool's parameter schema in this sprint. Instead, it should guarantee that downstream targets receive a trusted, structured scope claim, and it should prevent administrators from granting a broader scope at lower levels.

## Non-Goals

- Do not rewrite arbitrary `tools/call.params.arguments`.
- Do not build database-specific SQL predicates yet.
- Do not enforce masking or row filters inside downstream systems.
- Do not expand this sprint to OpenAPI, data lakes, warehouses, or direct database connectors.
- Do not replace capability governance. This is an extension of PR #32.

## Data Scope Semantics

`DataScope` remains the shared model:

```text
dataDomain
dataset
schema
table
field
classification
region
tenantFilter
maskingPolicy
rowFilter
```

For the MVP, each scope is a set of optional dimensions. Multiple scopes are alternatives: access may match any one declared scope. A child scope is valid only if it is equal to or narrower than at least one parent scope.

Narrowing rule:

- If the parent omits a dimension, the child may set that dimension.
- If the parent sets a dimension, the child must set the same value or omit that dimension to inherit the parent value.
- A child may not set a different value for a dimension already fixed by the parent.
- Empty child scope list means "inherit parent scopes".
- Empty tenant entitlement scope list means "no additional data boundary declared".
- Effective scopes are materialized scopes. When a child omits a parent-fixed dimension, the effective scope fills that dimension from the parent.

Effective scope calculation:

1. Start with capability `dataScopes`.
2. Apply tenant entitlement scopes. If tenant scopes are empty, inherit capability scopes.
3. Apply workspace assignment scopes. If workspace scopes are empty, inherit tenant scopes.
4. Apply instance assignment scopes. If instance scopes are empty, inherit workspace scopes.

If capability scopes are empty and tenant scopes are provided, tenant scopes become the first declared boundary. This supports manually classifying capabilities after discovery.

## Control Plane Enforcement

The API must reject invalid scope expansion:

- Creating a tenant entitlement must ensure requested scopes are narrower than capability scopes when capability scopes are present.
- Creating a workspace assignment must ensure requested scopes are narrower than the entitlement's effective scopes.
- Creating an instance assignment must ensure requested scopes are narrower than the workspace assignment's effective scopes.

Validation failure should use:

```text
400 VALIDATION_FAILED
dataScopes must be equal to or narrower than parent dataScopes
```

The response should not expose raw sensitive payloads or downstream data.

## Runtime Enforcement

`EvaluateCapabilityAccess` should return the effective data scopes after inheritance and narrowing.

Runtime behavior for MCP:

- `tools/list` filtering remains capability-based.
- `tools/call` requires capability approval and grant chain as in PR #32.
- When an allowed call is proxied upstream, AgentHarbor injects a trusted context header to the upstream MCP request.
- When an allowed call has no upstream endpoint, AgentHarbor returns the current accepted response and records the same effective scopes in trace evidence.

Header:

```http
X-AgentHarbor-Context: <base64url-json>
```

`X-AgentHarbor-Context` is reserved. AgentHarbor must drop any inbound caller-provided value and must overwrite any target static or credential header with the runtime-generated value. This prevents callers or target configuration from spoofing governance context.

Payload shape:

```json
{
  "schemaVersion": "agentharbor.context.v1",
  "platformId": "default",
  "tenantId": "tenant-a",
  "workspaceId": "ws-sales",
  "callerInstanceId": "agt_console_ops",
  "subjectId": "user:ops",
  "targetId": "agt_policy_router",
  "capabilityId": "cap_search_customer",
  "capabilityVersion": 1,
  "dataScopes": []
}
```

The MVP uses unsigned base64url JSON because AgentHarbor already controls the outbound channel and credential injection. Signing can be added later if downstream targets need to verify context outside a trusted network boundary.

## Trace Evidence

Trace events should keep the same fields added by PR #32:

- tenantId
- workspaceId
- callerInstanceId
- subjectId
- capabilityId
- capabilityVersion
- entitlementId
- workspaceAssignmentId
- instanceAssignmentId
- dataScopes

The `dataScopes` field must store the effective scopes actually forwarded at runtime.

## API and UI Changes

Backend:

- Add shared data-scope narrowing helpers in the store or domain layer.
- Use helpers in create tenant entitlement, workspace assignment, and instance assignment handlers.
- Return inherited effective scopes from capability access evaluation.
- Inject `X-AgentHarbor-Context` for allowed MCP `tools/call` upstream calls.
- Treat `X-AgentHarbor-Context` as a reserved outbound header owned only by AgentHarbor runtime.

Frontend:

- Keep the current capability governance page.
- Show effective inherited scope text for selected capability/grant chain.
- No full data-scope editor is required for this MVP.

Scripts:

- Extend Sprint 12 or add Sprint 13 demo to prove:
  - workspace scope cannot expand tenant scope.
  - instance scope can inherit workspace scope.
  - allowed `tools/call` forwards `X-AgentHarbor-Context`.
  - trace stores the effective scopes.

## Error Handling

- Scope expansion returns `400 VALIDATION_FAILED`.
- Invalid context encoding should fail closed before proxying.
- Upstream proxy errors continue to use existing upstream error classification.
- If no data scopes are declared anywhere, access may still be allowed by capability assignment, but the forwarded context contains an empty `dataScopes` list.

## Testing

Unit tests:

- `DataScopesNarrowOrEqual` allows equal scopes.
- It allows child dimensions when parent omits them.
- It rejects different values for parent-fixed dimensions.
- Empty child inherits parent.

Store tests:

- Memory and Postgres evaluate inherited effective scopes.
- Deny precedence still wins over allow.
- Subject selector wildcard behavior remains intact.

HTTP tests:

- Tenant entitlement rejects expansion beyond capability scopes.
- Workspace assignment rejects expansion beyond entitlement scopes.
- Instance assignment inherits workspace scopes when omitted.
- Allowed MCP `tools/call` forwards `X-AgentHarbor-Context`.
- Caller-supplied `X-AgentHarbor-Context` is not forwarded.
- Trace records the effective scopes.

Frontend verification:

- `pnpm --dir frontend test`
- `pnpm --dir frontend build`

Full verification:

- `make check`

## Risks and Follow-Ups

Risk: Downstream MCP targets may ignore `X-AgentHarbor-Context`.

Mitigation: this sprint makes AgentHarbor's boundary explicit and auditable. A later sprint can add signed context, downstream attestation, or connector-specific argument rewriting.

Risk: The narrowing model is intentionally simple and string-exact.

Mitigation: keep it conservative. Region hierarchies, classification ordering, and expression-aware row filters can be added after the first working loop.

Follow-ups:

- Signed context header.
- Connector-specific enforcement adapters.
- Data-source-specific scope schemas.
- UI editor for structured data scopes.
- Update/delete APIs for grant-chain objects with transactional audit.
