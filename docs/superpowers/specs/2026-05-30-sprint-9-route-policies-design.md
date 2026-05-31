# Sprint 9 Route Policies Design

## Problem

Route Governance currently shows access grants as if they were policies. That makes the control plane easy to demo, but it blocks richer To B behavior: explicit deny, priorities, disabled policies, later retry overrides, and clean audit semantics.

## Design

Add a first-class `RoutePolicy` domain object:

- `tenantId`, `workspaceId`: stored from the caller agent scope for management filtering.
- `callerAgentId`, `targetAgentId`: concrete subject and target.
- `routeType`, `routeKey`: exact route match; empty `routeKey` is wildcard.
- `effect`: `allow` or `deny`.
- `status`: `enabled` or `disabled`.
- `priority`: integer; higher value wins. Ties prefer `deny`, then older creation time, then ID.

Sprint 9 only supports policies where caller and target Agents share the same tenant and workspace. Cross-scope routes need explicit future product modeling so they do not create target-side governance blind spots.

The data plane asks the repository for a route access decision:

1. Find enabled matching policies for caller, target, route type, and route key.
2. If any policy matches, use the top-ranked policy decision.
3. If no policy matches, fall back to existing access grants.
4. Record the decision reason in trace audit data.

This preserves existing demos while making route policies the forward path.

## API

- `POST /api/v1/route-policies`
- `GET /api/v1/route-policies?tenantId=&workspaceId=`
- `PATCH /api/v1/route-policies/{id}`
- `DELETE /api/v1/route-policies/{id}`

`DELETE` disables the policy instead of hard-deleting it so audit and governance tables remain stable.

## Risks

- Policy precedence must be deterministic across memory and PostgreSQL stores.
- Legacy grants must remain compatible for existing scripts.
- The UI should not show duplicate rows when both policy and grant endpoints exist.
