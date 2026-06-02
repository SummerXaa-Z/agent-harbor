# Tenant Permission Console Design

Date: 2026-06-02
Status: Approved by user direction to follow the recommended tenant access-profile UI approach

## Goal

Add a frontend console view that lets an operator answer one product-critical question:

```text
For this tenant scope, which MCP/agent capabilities are effectively reachable, at which tenant/workspace/instance level, under what data scope, and what recent traces prove or deny it?
```

This is the UI companion to `GET /api/v1/tenants/{id}/access-profile`. The primary journey starts with a tenant, not a caller, because tenant scope is the control point for registered MCP tools, agent permissions, and downstream data access.

## Non-Goals

- No permission write actions in this step.
- No entitlement wizard.
- No speculative simulation matrix.
- No new backend endpoint.
- No new frontend dependency.
- No separate landing page or marketing-style UI.

## User Journey

1. Operator opens the console and chooses `Access`.
2. Operator confirms or edits the tenant ID and optional filters: workspace, target, capability, caller instance, and trace limit.
3. Console loads `/api/v1/tenants/{tenantId}/access-profile`.
4. Operator sees summary counts for scope tenants, grants, targets, workspaces, callers, and recent allow/deny traces.
5. Operator inspects each grant chain from tenant entitlement to workspace assignment to instance assignment.
6. Operator checks effective data scopes and invalid scope reasons where narrowing rules were violated.
7. Operator reviews recent trace evidence tied to the same tenant profile.

## UX Shape

Use the existing operational console layout:

- Add one sidebar nav item: `Access`.
- Keep the current topbar, admin key input, scope strip, and metric strip patterns.
- Render an access filter bar inside the view, not a separate wizard.
- Use dense tables and rows instead of cards nested inside cards.
- Use existing `lucide-react` icons.
- Preserve fallback data behavior so the frontend remains useful without a running backend.

The main view should have three zones:

- `Tenant Access Profile`: summary, filters, and tenant scope list.
- `Effective Grant Chain`: entitlement rows with nested workspace and instance rows.
- `Trace Evidence`: recent allowed/denied traces returned by the profile endpoint.

## Data Contract

Frontend types should mirror the backend profile response:

- `Tenant`
- `AccessProfileSummary`
- `TenantAccessProfile`
- `TenantAccessProfileGrant`
- `TenantAccessProfileWorkspace`
- `TenantAccessProfileInstance`
- `AccessProfileFilters`

The API wrapper should call:

```http
GET /api/v1/tenants/{id}/access-profile?workspaceId=&targetId=&capabilityId=&callerInstanceId=&traceLimit=
```

`traceLimit` defaults to `20`; the UI may send `0` through `100`.

## Error and Fallback Behavior

- Network failure should use a local sample profile and show that the profile is fallback data.
- HTTP/API errors should remain visible and not silently fall back unless the failure is a fetch/network failure.
- Empty grants should render an empty row explaining no grant chain matched the current filters.
- `scopeStatus=invalid` should be visually prominent and include `scopeReason`.

## Testing

Frontend tests should target pure helper functions because the current frontend uses Node's built-in `node --test` rather than React component testing:

- filter query normalization
- trace limit validation/clamping rules used by the UI helper
- status-to-tone mapping
- data-scope summarization
- invalid grant counting

Build verification must run `pnpm --dir frontend test` and `pnpm --dir frontend build`.

## Grill-Me Pressure Test Consensus

- Should the first UI write permissions? No. The first trust-building journey is read-only explanation.
- Should the caller drive the flow? No. The product model says tenant is the permission control boundary; caller is a filter within tenant scope.
- Should this replace the existing Capabilities view? No. Capabilities manages discovery and grant creation; Access explains effective state after configuration.
- Should we build a visual graph? Not in MVP. Dense operational rows are easier to scan, test, and keep responsive.
- Should fallback data be removed? No. Existing console behavior depends on a useful offline/sample state.
